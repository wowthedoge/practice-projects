package main

import (
	"bytes"
	"crypto"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/hkdf"
	"crypto/hmac"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"log"
	"net"
)

type ServerHello struct {
	Random             [32]byte
	CipherSuite        uint16
	EphemeralPublicKey []byte
}

type EncryptedCertificate struct {
	Bytes []byte
}

func StartServer(serverCert *x509.Certificate, serverKey *rsa.PrivateKey) {

	// Start TCP server
	l, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}
	defer l.Close()
	fmt.Println("TCP server started on port 8080")

	// Handle connection
	for {
		conn, err := l.Accept()
		if err != nil {
			log.Println("Error accepting connection", err)
			continue
		}

		go handleHandshake(conn, serverCert, serverKey)
	}
}

func handleHandshake(conn net.Conn, cert *x509.Certificate, serverKey *rsa.PrivateKey) {

	var transcript []byte
	var buf bytes.Buffer
	dec := json.NewDecoder(conn)

	// 1. Receive ClientHello
	var clientHello ClientHello
	err := dec.Decode(&clientHello)
	fatal(err, "Failed to decode ClientHello")
	fmt.Println("Server received ClientHello")
	json.NewEncoder(&buf).Encode(clientHello)
	transcript = append(transcript, buf.Bytes()...)
	buf.Reset()

	// 2. Generate ephemeral keypair
	privateKey, err := ecdh.X25519().GenerateKey(rand.Reader)
	fatal(err, "Failed to create ECDH keypair")

	// 3. Send ServerHello
	var random [32]byte
	rand.Read(random[:])
	serverHello := ServerHello{
		Random:             random,
		CipherSuite:        0x1301,
		EphemeralPublicKey: privateKey.PublicKey().Bytes(),
	}
	err = json.NewEncoder(&buf).Encode(serverHello)
	fatal(err, "Failed to encode ServerHello")
	conn.Write(buf.Bytes())
	fmt.Println("Server sent ServerHello")
	transcript = append(transcript, buf.Bytes()...)
	buf.Reset()

	// 4. Compute shared secret
	clientPublicKey, err := ecdh.X25519().NewPublicKey(clientHello.EphemeralPublicKey)
	fatal(err, "Invalid client ephemeral public key")
	sharedSecret, err := privateKey.ECDH(clientPublicKey)
	fatal(err, "Failed to compute shared secret on server")
	fmt.Println("Computed shared secret on server: ", string(sharedSecret))

	// 5. Derive keys
	clientWriteKey, serverWriteKey, clientNonce, serverNonce, err := DeriveKeys(sharedSecret)
	fatal(err, "Failed to derive keys")
	fmt.Println("Computed clientWriteKey on server: ", string(clientWriteKey))
	fmt.Println("Computed serverWriteKey on server: ", string(serverWriteKey))
	fmt.Println("Computed clientNonce on server: ", string(clientNonce))
	fmt.Println("Computed serverNonce on server: ", string(serverNonce))

	// 6. Encrypt and send cert
	encryptedCert, err := encryptAESGCM(serverWriteKey, serverNonce, cert.Raw)
	fatal(err, "Failed to encrypt cert")
	err = json.NewEncoder(conn).Encode(EncryptedCertificate{Bytes: encryptedCert})
	fatal(err, "Failed to send EncryptedCertificate")
	fmt.Println("Server sent certificate")
	transcript = append(transcript, cert.Raw...)

	// 7. Sign and send CertificateVerify
	hashTranscript := sha256.Sum256(transcript)
	signature, err := rsa.SignPKCS1v15(rand.Reader, serverKey, crypto.SHA256, hashTranscript[:])
	fatal(err, "Failed to sign CertificateVerify")
	encryptedSignature, err := encryptAESGCM(serverWriteKey, serverNonce, signature)
	fatal(err, "Failed to encrypt CertificateVerify")
	err = json.NewEncoder(&buf).Encode(CertificateVerify{EncryptedSignature: encryptedSignature})
	fatal(err, "Failed to encode CertificateVerify")
	conn.Write(buf.Bytes())
	fatal(err, "Failed to send CertificateVerify")
	fmt.Println("Server sent CertificateVerify")
	transcript = append(transcript, buf.Bytes()...)
	buf.Reset()

	// 8. Send Finished
	finishedKey, err := hkdf.Key(sha256.New, sharedSecret, nil, "finished", 32)
	fatal(err, "Failed to derive Finished key")
	h := hmac.New(sha256.New, finishedKey)
	h.Write(transcript)
	verifyData := h.Sum(nil)
	encryptedFinished, err := encryptAESGCM(serverWriteKey, serverNonce, verifyData)
	fatal(err, "Failed to encrypt Finished")
	err = json.NewEncoder(conn).Encode(Finished{EncryptedVerifyData: encryptedFinished})
	fatal(err, "Failed to send Finished")
	fmt.Println("Server sent Finished")

	// 9. Receive and verify Finished
	var encryptedClientFinished Finished
	err = dec.Decode(&encryptedClientFinished)
	fatal(err, "Failed to decode client Finished from server")
	clientFinished, err := decryptAESGCM(clientWriteKey, clientNonce, encryptedClientFinished.EncryptedVerifyData)
	fatal(err, "Failed to decrypt client Finished")
	if !hmac.Equal(clientFinished, verifyData) {
		fatal(nil, "Server failed to verify Finished - handshake tampered")
	}
	fmt.Println("Server verified Finished")
}

func encryptAESGCM(key []byte, nonce []byte, plaintext []byte) ([]byte, error) {
	blockCipher, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("Failed to encrypt")
	}

	aesgcm, err := cipher.NewGCM(blockCipher)
	if err != nil {
		return nil, fmt.Errorf("Failed to encrypt")
	}

	return aesgcm.Seal(nil, nonce, plaintext, nil), nil
}

func decryptAESGCM(key []byte, nonce []byte, ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("Failed to decrypt")
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("Failed to decrypt")
	}

	plaintext, err := aesgcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("Failed to decrypt")
	}

	return plaintext, nil
}

type CertificateVerify struct {
	EncryptedSignature []byte
}

type Finished struct {
	EncryptedVerifyData []byte
}
