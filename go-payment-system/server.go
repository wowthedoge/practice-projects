package main

import (
	"database/sql"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/stripe/stripe-go/v83"
	"github.com/stripe/stripe-go/v83/paymentintent"
	"github.com/stripe/stripe-go/v83/webhook"
)

var db *sql.DB

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}

	// Get Stripe secret key from environment variable
	stripeKey := os.Getenv("STRIPE_SECRET_KEY")
	if stripeKey == "" {
		log.Fatal("STRIPE_SECRET_KEY environment variable is not set")
	}
	stripe.Key = stripeKey

	var err error
	db, err = connectDB()
	if err != nil {
		log.Fatalf("Error connecting to database: %v", err)
	}

	defer db.Close()

	http.Handle("/", http.FileServer(http.Dir("public")))
	// http.HandleFunc("/api/create-checkout-session", createCheckoutSession)
	http.HandleFunc("/api/create-payment-intent", createPaymentIntent)
	http.HandleFunc("/api/orders", getOrders)
	http.HandleFunc("/api/webhook", handleWebhook)

	addr := "localhost:4242"
	log.Printf("Listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}

type CreatePaymentIntentRequest struct {
	Items []OrderItem `json:"items"`
}

type OrderItem struct {
	ID       int `json:"id"`
	Quantity int `json:"quantity"`
}

type CreatePaymentIntentResponse struct {
	ClientSecret string `json:"clientSecret"`
}

func createPaymentIntent(w http.ResponseWriter, r *http.Request) {
	log.Println("Creating payment intent")

	var req CreatePaymentIntentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("ERROR: Failed to decode request body: %v\n", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Invalid request body",
		})
		return
	}

	log.Printf("Received order items: %v\n", req.Items)

	totalAmount := getTotalPrice(req.Items)

	log.Printf("Total amount to charge: %d cents (MYR %.2f)", totalAmount, float64(totalAmount)/100)

	params := &stripe.PaymentIntentParams{
		Amount:   stripe.Int64(totalAmount),
		Currency: stripe.String("myr"),
	}

	paymentIntent, err := paymentintent.New(params)
	if err != nil {
		log.Printf("ERROR: Stripe payment intent creation failed: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Failed to create payment intent",
		})
		return
	}

	log.Printf("SUCCESS: Payment intent created with ID: %s", paymentIntent.ID)

	_, err = db.Exec("INSERT INTO orders (payment_id, status, total_amount, currency) VALUES ($1, $2, $3, $4)", paymentIntent.ID, "pending", totalAmount, "myr")
	if err != nil {
		log.Printf("ERROR: Failed to save order to database: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Failed to save order to database",
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(CreatePaymentIntentResponse{
		ClientSecret: paymentIntent.ClientSecret,
	})
}

func handleWebhook(w http.ResponseWriter, r *http.Request) {
	payload, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	endpointSecret := os.Getenv("STRIPE_WEBHOOK_SECRET")
	signatureHeader := r.Header.Get("Stripe-Signature")
	event, err := webhook.ConstructEvent(payload, signatureHeader, endpointSecret)
	if err != nil {
		log.Println("Webhook signature verification failed", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	switch event.Type {
	case "payment_intent.succeeded":
		var paymentIntent stripe.PaymentIntent
		err := json.Unmarshal(event.Data.Raw, &paymentIntent)
		if err != nil {
			log.Printf("ERROR: Failed to parse payment intent: %v\n", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		_, err = db.Exec("UPDATE orders SET status = $1 WHERE payment_id = $2", "completed", paymentIntent.ID)
		if err != nil {
			log.Printf("ERROR: Failed to update order status: %v\n", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		log.Printf("SUCCESS: Order with payment ID %s marked as completed", paymentIntent.ID)

	case "payment_intent.payment_failed":
		var paymentIntent stripe.PaymentIntent
		err := json.Unmarshal(event.Data.Raw, &paymentIntent)
		if err != nil {
			log.Printf("ERROR: Failed to parse payment intent: %v\n", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		_, err = db.Exec("UPDATE orders SET status = $1 WHERE payment_id = $2", "failed", paymentIntent.ID)
	}

	w.WriteHeader(http.StatusOK)
}

func getTotalPrice(items []OrderItem) int64 {
	var totalAmount int64
	for _, item := range items {
		var priceCents int64
		err := db.QueryRow("SELECT price_cents FROM products WHERE id = $1", item.ID).Scan(&priceCents)
		if err != nil {
			log.Printf("ERROR: Failed to get product %d from database: %v\n", item.ID, err)
			return 0
		}
		totalAmount += priceCents * int64(item.Quantity)
		log.Printf("Product ID %d: price=%d cents, quantity=%d, subtotal=%d cents",
			item.ID, priceCents, item.Quantity, priceCents*int64(item.Quantity))
	}
	return totalAmount
}

func getOrders(w http.ResponseWriter, r *http.Request) {
	log.Println("Getting orders")

	rows, err := db.Query(`
		SELECT id, payment_id, status, total_amount, currency, created_at
		FROM orders
		ORDER BY id DESC
	`)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
	defer rows.Close()

	var orders []map[string]interface{}

	for rows.Next() {
		var id int
		var paymentId, status, currency string
		var amount int64
		var createdAt string

		rows.Scan(&id, &paymentId, &status, &amount, &currency, &createdAt)

		orders = append(orders, map[string]any{
			"id":         id,
			"payment_id": paymentId,
			"status":     status,
			"amount":     amount,
			"currency":   currency,
			"created_at": createdAt,
		})

	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]any{
		"orders": orders,
	})
}

func connectDB() (*sql.DB, error) {
	db, err := sql.Open("postgres", os.Getenv("DATABASE_URL"))
	if err != nil {
		return nil, err
	}

	// Test the connection
	if err := db.Ping(); err != nil {
		return nil, err
	}

	log.Println("Successfully connected to database")
	return db, nil
}

// func createCheckoutSession(w http.ResponseWriter, r *http.Request) {
// 	params := &stripe.CheckoutSessionParams{
// 		LineItems: []*stripe.CheckoutSessionLineItemParams{
// 			{
// 				Price:    stripe.String("price_1SSzljENWXLgYDXRd8hMvTAf"),
// 				Quantity: stripe.Int64(1),
// 			},
// 			{
// 				Price:    stripe.String("price_1SSzlxENWXLgYDXRxl8MN48w"),
// 				Quantity: stripe.Int64(1),
// 			},
// 		},
// 		Mode:       stripe.String(string(stripe.CheckoutSessionModePayment)),
// 		SuccessURL: stripe.String("http://localhost:3000/"),
// 		CancelURL:  stripe.String("http://localhost:3000/"),
// 	}

// 	s, err := session.New(params)

// 	if err != nil {
// 		log.Printf("session.New: %v", err)
// 	}

// 	http.Redirect(w, r, s.URL, http.StatusSeeOther)
// }
