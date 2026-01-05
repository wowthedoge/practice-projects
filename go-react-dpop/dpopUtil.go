package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/patrickmn/go-cache"
)

// Using go-cache for this demo, would use Redis/database for production
var jtiCache = cache.New(5*time.Minute, 5*time.Minute)

func ValidateDPoPMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {

		// Validate DPoP proof before issuing token
		dpopProof := c.GetHeader("DPoP")
		if dpopProof == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "DPoP proof required"})
			c.Abort()
			return
		}

		url := "http://" + c.Request.Host + c.Request.URL.Path
		method := c.Request.Method

		jkt, err := validateDPoPProof(dpopProof, method, url)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			c.Abort()
			return
		}

		c.Set("dpop_jkt", jkt)
		c.Next()
	}
}

type DPoPClaims struct {
	HttpMethod    string `json:"htm"`
	HttpTargetUri string `json:"htu"`
	JwtId         string `json:"jti"`
	jwt.RegisteredClaims
}

func validateDPoPProof(proof string, requestMethod string, requestUrl string) (string, error) {

	fmt.Println("Validating DPoP proof:", proof, "with method:", requestMethod, "and url:", requestUrl)

	claims := &DPoPClaims{}
	token, _, err := jwt.NewParser().ParseUnverified(proof, claims)
	if err != nil {
		return "", err
	}

	// Validate header
	if token.Header["typ"] != "dpop+jwt" {
		return "", errors.New("Invalid typ, expected 'dpop+jwt'")
	}
	if token.Header["alg"] != "ES256" {
		return "", errors.New("Invalid alg, expected 'ES256'")
	}
	jwkRaw, ok := token.Header["jwk"].(map[string]interface{})
	if !ok {
		return "", errors.New("Missing or invalid JWK in header")
	}
	publicKey, err := jwkToECDSAPublicKey(jwkRaw)
	if err != nil {
		return "", fmt.Errorf("invalid JWK: %w", err)
	}

	// Validate claims
	if claims.HttpMethod != requestMethod {
		return "", fmt.Errorf("Method mismatch: expected %s, got %s", requestMethod, claims.HttpMethod)
	}
	if claims.HttpTargetUri != requestUrl {
		return "", fmt.Errorf("Url mismatch: expected %s, got %s", requestUrl, claims.HttpTargetUri)
	}
	if time.Now().Unix()-claims.IssuedAt.Unix() > 60 {
		return "", errors.New("DPoP proof expired")
	}
	if err := jtiCache.Add(claims.JwtId, true, 1*time.Minute); err != nil {
		return "", errors.New("JTI already used")
	}

	// Verify signature
	_, err = jwt.Parse(proof, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodECDSA); !ok {
			return nil, fmt.Errorf("Unexpected signing method: %v", t.Header["alg"])
		}
		return publicKey, nil
	})
	if err != nil {
		return "", fmt.Errorf("signature verification failed: %w", err)
	}

	// Calculate and return JKT
	return createJKT(jwkRaw), nil
}

func jwkToECDSAPublicKey(jwk map[string]interface{}) (*ecdsa.PublicKey, error) {
	if jwk["kty"] != "EC" || jwk["crv"] != "P-256" {
		return nil, errors.New("Unsupported key type, expected EC P-256")
	}

	xBytes, err := base64.URLEncoding.DecodeString(jwk["x"].(string))
	if err != nil {
		return nil, errors.New("Failed to decode x coordinate of key")
	}
	yBytes, err := base64.URLEncoding.DecodeString(jwk["y"].(string))
	if err != nil {
		return nil, errors.New("Failed to decode y coordinate of key")
	}

	return &ecdsa.PublicKey{
		Curve: elliptic.P256(),
		X:     new(big.Int).SetBytes(xBytes),
		Y:     new(big.Int).SetBytes(yBytes),
	}, nil
}

func createJKT(jwk map[string]interface{}) string {
	hash := sha256.Sum256([]byte(fmt.Sprintf(`{"crv":"%s","kty":"%s","x":"%s","y":"%s"}`,
		jwk["crv"], jwk["kty"], jwk["x"], jwk["y"])))
	return base64.RawURLEncoding.EncodeToString(hash[:])
}
