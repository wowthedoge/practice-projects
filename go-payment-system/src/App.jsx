import React, { useState, useEffect } from "react";
import "./App.css";
import { loadStripe } from "@stripe/stripe-js";
import {
  Elements,
  useStripe,
  useElements,
  CardElement,
} from "@stripe/react-stripe-js";

const stripePromise = loadStripe(
  "pk_test_51SOHhjENWXLgYDXRCg47xHYbgjcvCtN6g0fD9NnxA2QzGBADdhw9x0DWBpkxPn3tIQjEy1DNEvIltodCZx7IwGeG00XHKxzvR1"
);

export default function App() {
  const [showCheckout, setShowCheckout] = useState(false);
  const [clientSecret, setClientSecret] = useState(null);
  const [orders, setOrders] = useState([]);

  // Centralized data fetching
  const fetchOrders = async () => {
    try {
      const res = await fetch("/api/orders");
      if (!res.ok) {
        console.error("Failed to fetch orders:", res.statusText);
        return;
      }
      const data = await res.json();
      setOrders(data.orders || []);
    } catch (error) {
      console.error("Error fetching orders:", error);
    }
  };

  // Handler for payment success
  const handlePaymentSuccess = () => {
    setShowCheckout(false);
    fetchOrders();
  };

  useEffect(() => {
    fetchOrders();
  }, []);

  return (
    <div className="main-container">
      <YourBasket 
        setShowCheckout={setShowCheckout} 
        setClientSecret={setClientSecret} 
      />

      {showCheckout && (
        <Elements stripe={stripePromise}>
          <CheckoutForm
            clientSecret={clientSecret}
            onSuccess={handlePaymentSuccess}
            onCancel={() => setShowCheckout(false)}
          />
        </Elements>
      )}

      <YourOrders orders={orders} />
    </div>
  );
}

const YourBasket = ({ setShowCheckout, setClientSecret }) => {
  const handleCheckout = async () => {
    try {
      console.log("Creating payment intent");
      const res = await fetch("/api/create-payment-intent", {
        method: "POST",
        headers: { "Content-Type": "application/json  " },
        body: JSON.stringify({
          items: [
            { id: 1, quantity: 1 },
            { id: 2, quantity: 1 },
          ],
        }),
      });

      if (!res.ok) {
        throw new Error(`HTTP error! status: ${res.status}`);
      }
      const { clientSecret } = await res.json();
      console.log("Payment intent created. Client secret:", clientSecret);
      setClientSecret(clientSecret);
      setShowCheckout(true);
    } catch (error) {
      console.error("Error creating payment intent:", error);
    }
  };

  return (
    <div className="your-basket">
      <h3>Your basket:</h3>
      <section>
        <div className="product">
          <img
            src="https://kgifts.shop/cdn/shop/files/rn-image_picker_lib_temp_72443b4f-025a-49fd-a403-a97f6f27df94.jpg?v=1742828855&width=1445"
            alt="Hahachipu"
          />
          <div className="description">
            <h3>Hahachipu</h3>
            <h5>RM5</h5>
          </div>
        </div>
        <div className="product">
          <img
            src="https://img4.dhresource.com/webp/m/f3/albu/ys/g/26/6bae8194-6778-4529-9957-8c614aa73afa.jpg"
            alt="Lalabu"
          />
          <div className="description">
            <h3>Lalabu</h3>
            <h5>RM5</h5>
          </div>
        </div>
        <button onClick={handleCheckout}>Checkout</button>
      </section>
    </div>
  );
};

const CheckoutForm = ({ clientSecret, onSuccess, onCancel }) => {
  const stripe = useStripe();
  const elements = useElements();

  const handlePay = async () => {
    if (!stripe || !elements) return;

    try {
      const result = await stripe.confirmCardPayment(clientSecret, {
        payment_method: {
          card: elements.getElement(CardElement),
        },
      });

      if (result.error) {
        console.error("Error confirming card payment:", result.error.message);
      } else {
        onSuccess(); // Trigger refresh in parent
      }
    } catch (err) {
      console.error("Error confirming card payment:", err.message);
    }
  };

  return (
    <div className="checkout-form">
      <h3>Enter Payment Details</h3>
      <CardElement />
      <div className="button-group">
        <button onClick={handlePay} disabled={!stripe}>
          Pay Now
        </button>
        <button className="cancel-button" onClick={onCancel}>
          Cancel
        </button>
      </div>
    </div>
  );
};

const YourOrders = ({ orders }) => {
    if (orders.length === 0) {
        return (
            <div className="your-orders">
                <h3>Your orders:</h3>
                <section>
                    <p>No orders yet</p>
                </section>
            </div>
        );
    }

    return (
      <div className="your-orders">
        <h3>Your orders:</h3>
        <section>
          {orders.map((order) => (
            <div className="order" key={order.id}>
              <h3>Order #{order.id}</h3>
              <h5>RM{order.amount}</h5>
              <h5>Status: {order.status}</h5>
              <h5>{order.created_at}</h5>
            </div>
          ))}
        </section>
      </div>
    );

};
