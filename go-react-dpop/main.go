package main

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type TokenRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type TokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

func main() {
	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowOrigins: []string{"http://localhost:5173"},
		AllowMethods: []string{"GET", "POST", "OPTIONS"},
		AllowHeaders: []string{"Accept", "Authorization", "Content-Type", "DPoP"},
	}))

	r.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "oAuth2 demo"})
	})

	// Normal Bearer token endpoints (no DPoP)
	r.POST("/token", tokenHandlerBearer)
	r.GET("/protected", BearerAuthMiddleware(), protectedHandlerBearer)

	// DPoP-protected endpoints
	r.POST("/token-dpop", ValidateDPoPMiddleware(), tokenHandlerDPoP)
	r.GET("/protected-dpop", ValidateDPoPMiddleware(), DPoPAuthMiddleware(), protectedHandlerDPoP)

	r.Run(":8080")
}

var jwtSecret = []byte("secret-key")

func tokenHandlerDPoP(c *gin.Context) {
	var req TokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	if req.Username != "demo" || req.Password != "password" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	jkt := c.GetString("dpop_jkt")
	claims := jwt.MapClaims{
		"sub": req.Username,
		"exp": jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
		"iat": jwt.NewNumericDate(time.Now()),
		"cnf": map[string]string{
			"jkt": jkt,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	accessToken, err := token.SignedString(jwtSecret)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Token creation failed"})
		return
	}

	c.JSON(http.StatusOK, TokenResponse{
		AccessToken: accessToken,
		TokenType:   "DPoP",
		ExpiresIn:   3600,
	})
}

func DPoPAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if !strings.HasPrefix(authHeader, "DPoP ") {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Expected DPoP auth scheme"})
			c.Abort()
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "DPoP ")

		// Validate token
		claims := jwt.MapClaims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(_ *jwt.Token) (interface{}, error) {
			return jwtSecret, nil
		})
		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		// Validate DPoP
		cnf, ok := claims["cnf"].(map[string]interface{})
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token missing cnf claim"})
			c.Abort()
			return
		}
		tokenJkt, ok := cnf["jkt"].(string)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token missing jkt in cnf"})
			c.Abort()
			return
		}

		dpopJkt := c.GetString("dpop_jkt")
		if tokenJkt != dpopJkt {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "DPoP key mismatch"})
			c.Abort()
			return
		}

		c.Set("userId", claims["sub"])
		c.Next()
	}
}

func protectedHandlerDPoP(c *gin.Context) {
	userId, _ := c.Get("userId")
	c.JSON(http.StatusOK, gin.H{
		"message": "Accessing protected data (DPoP protected)",
		"data":    "dpop-protected-data-123",
		"userId":  userId,
	})
}

// ============================================
// BEARER TOKEN ENDPOINTS (no DPoP binding)
// These show what happens WITHOUT DPoP
// ============================================

// Issues a Bearer token WITHOUT DPoP binding
func tokenHandlerBearer(c *gin.Context) {
	var req TokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	if req.Username != "demo" || req.Password != "password" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	// No cnf claim - token is NOT bound to any key!
	claims := jwt.MapClaims{
		"sub": req.Username,
		"exp": jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
		"iat": jwt.NewNumericDate(time.Now()),
		// No "cnf" claim - this token can be stolen and reused!
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	accessToken, err := token.SignedString(jwtSecret)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Token creation failed"})
		return
	}

	c.JSON(http.StatusOK, TokenResponse{
		AccessToken: accessToken,
		TokenType:   "Bearer", // Regular Bearer token
		ExpiresIn:   3600,
	})
}

// Simple Bearer token auth - NO DPoP validation
func BearerAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Expected Bearer auth scheme"})
			c.Abort()
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")

		claims := jwt.MapClaims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(_ *jwt.Token) (interface{}, error) {
			return jwtSecret, nil
		})
		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		// No DPoP validation - anyone with this token can use it!
		c.Set("userId", claims["sub"])
		c.Next()
	}
}

func protectedHandlerBearer(c *gin.Context) {
	userId, _ := c.Get("userId")
	c.JSON(http.StatusOK, gin.H{
		"message": "Accessing protected data (Bearer token - no DPoP)",
		"data":    "bearer-data-456",
		"userId":  userId,
		"warning": "This token has no DPoP binding - it can be stolen and reused!",
	})
}
