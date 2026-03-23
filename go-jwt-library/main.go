package main

import (
	"crypto"
	"crypto/hmac"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

type JWT struct {
	Header    Header
	Payload   Payload
	Signature []byte
}

type Header struct {
	Alg string `json:"alg"`
	Typ string `json:"typ"`
}

type Payload struct {
	Sub string `json:"sub"`
	Iat int64  `json:"iat"`
	Exp int64  `json:"exp"`
}

func CreateJWTHS256(payload Payload, secret string) (string, error) {
	headerBytes, _ := json.Marshal(Header{Alg: "HS256", Typ: "JWT"})
	encodedHeader := base64.RawURLEncoding.EncodeToString(headerBytes)

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", errors.New("Invalid payload")
	}
	encodedPayload := base64.RawURLEncoding.EncodeToString(payloadBytes)

	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(encodedHeader + "." + encodedPayload))
	encodedSignature := base64.RawURLEncoding.EncodeToString(h.Sum(nil))

	return encodedHeader + "." + encodedPayload + "." + encodedSignature, nil
}

func VerifyJWTHS256(token string, secret string) (Payload, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return Payload{}, errors.New("Invalid token")
	}
	encodedHeader := parts[0]
	encodedPayload := parts[1]
	encodedSignature := parts[2]

	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(encodedHeader + "." + encodedPayload))
	computedSignature := h.Sum(nil)
	actualSigature, err := base64.RawURLEncoding.DecodeString(encodedSignature)
	if err != nil {
		return Payload{}, errors.New("Invalid signature")
	}

	var payload Payload
	decodedPayload, err := base64.RawURLEncoding.DecodeString(encodedPayload)
	if err != nil {
		return Payload{}, errors.New("Invalid payload")
	}
	err = json.Unmarshal(decodedPayload, &payload)
	if err != nil {
		return Payload{}, errors.New("Invalid payload")
	}

	if time.Now().Unix() > payload.Exp {
		return Payload{}, errors.New("Token expired")
	}
	if !hmac.Equal(computedSignature, actualSigature) {
		return Payload{}, errors.New("Invalid signature")
	}

	return payload, nil
}

func CreateJWTRS256(payload Payload, privateKey *rsa.PrivateKey) (string, error) {
	headerBytes, err := json.Marshal(Header{Alg: "RS256", Typ: "JWT"})
	if err != nil {
		return "", errors.New("Invalid header")
	}
	encodedHeader := base64.RawURLEncoding.EncodeToString(headerBytes)

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", errors.New("Invalid payload")
	}
	encodedPayload := base64.RawURLEncoding.EncodeToString(payloadBytes)

	hash := sha256.Sum256([]byte(encodedHeader + "." + encodedPayload))
	signature, err := rsa.SignPKCS1v15(nil, privateKey, crypto.SHA256, hash[:])
	if err != nil {
		return "", errors.New("Failed to sign JWT")
	}
	encodedSignature := base64.RawURLEncoding.EncodeToString(signature)

	return encodedHeader + "." + encodedPayload + "." + encodedSignature, nil
}

func VerifyJWTRS256(token string, publicKey *rsa.PublicKey) (Payload, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return Payload{}, errors.New("Invalid token")
	}
	encodedHeader := parts[0]
	encodedPayload := parts[1]
	encodedSignature := parts[2]

	hashed := sha256.Sum256([]byte(encodedHeader + "." + encodedPayload))
	signature, err := base64.RawURLEncoding.DecodeString(encodedSignature)
	if err != nil {
		return Payload{}, errors.New("Invalid signature")
	}
	err = rsa.VerifyPKCS1v15(publicKey, crypto.SHA256, hashed[:], signature)
	if err != nil {
		return Payload{}, errors.New("Invalid signature")
	}

	var payload Payload
	decodedPayload, err := base64.RawURLEncoding.DecodeString(encodedPayload)
	if err != nil {
		return Payload{}, errors.New("Invalid payload")
	}
	err = json.Unmarshal(decodedPayload, &payload)
	if err != nil {
		return Payload{}, errors.New("Invalid payload")
	}

	if time.Now().Unix() > payload.Exp {
		return Payload{}, errors.New("Token expired")
	}

	return payload, nil
}

// func generateRSAKeypair() (privateKey *rsa.PrivateKey, publicKey *rsa.PublicKey, err error) {
// 	privateKey, err = rsa.GenerateKey(rand.Reader, 2048)
// 	if err != nil {
// 		return nil, nil, errors.New("RSA key generation failed")
// 	}
// 	publicKey = &privateKey.PublicKey
// 	return privateKey, publicKey, nil
// }

// func main() {
// 	secret := "special-secret"
// 	HS256token, _ := CreateJWTHS256(Payload{
// 		Sub: "subject1",
// 		Exp: 1874262173,
// 		Iat: 1774262173,
// 	}, secret)
// 	fmt.Println("Created HS256 token", HS256token)

// 	payload, _ := VerifyJWTHS256(HS256token, secret)
// 	fmt.Println(payload)

// 	tamperedHS256Token := HS256token + "tamper"

// 	payload, err := VerifyJWTHS256(tamperedHS256Token, secret)
// 	if err != nil {
// 		fmt.Println(err)
// 	}

// 	payload, err = VerifyJWTHS256(tamperedHS256Token, "other-secret")
// 	if err != nil {
// 		fmt.Println(err)
// 	}

// 	expiredHS256token, _ := CreateJWTHS256(Payload{
// 		Sub: "subject1",
// 		Exp: 1674262173,
// 		Iat: 1774262173,
// 	}, secret)
// 	payload, err = VerifyJWTHS256(expiredHS256token, secret)
// 	if err != nil {
// 		fmt.Println(err)
// 	}

// 	privateKey, publicKey, err := generateRSAKeypair()
// 	RS256token, _ := CreateJWTRS256(Payload{
// 		Sub: "subject1",
// 		Exp: 1874262173,
// 		Iat: 1774262173,
// 	}, privateKey)
// 	payload, err = VerifyJWTRS256(RS256token, publicKey)
// 	if err != nil {
// 		fmt.Println(err)
// 	}
// 	fmt.Println("Created RS256 token", RS256token)
// }
