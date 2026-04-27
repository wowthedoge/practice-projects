package main

import (
	"bytes"
	"crypto"
	"crypto/ecdh"
	"crypto/hkdf"
	"crypto/hmac"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"net"
	"time"
)

type ClientHello struct {
	EphemeralPublicKey []byte
	Random             [32]byte
	CipherSuites       []uint16
}

func StartClient(caStore []*x509.Certificate) {
	conn, err := net.Dial("tcp", "localhost:8080")
	fatal(err, "Failed to initiate connection")
	defer conn.Close()

	fmt.Println("Starting TCP connection on localhost:8080")
	doHandshake(conn, caStore)
}

func doHandshake(conn net.Conn, caStore []*x509.Certificate) {

	var transcript []byte
	var buf bytes.Buffer
	dec := json.NewDecoder(conn)

	// 1. Generate keypair
	privateKey, err := ecdh.X25519().GenerateKey(rand.Reader)
	fatal(err, "Failed to create ECDH keypair")

	// 2. Send ClientHello
	var random [32]byte
	rand.Read(random[:])
	clientHello := ClientHello{
		EphemeralPublicKey: privateKey.PublicKey().Bytes(),
		Random:             random,
		CipherSuites:       []uint16{0x1301},
	}
	err = json.NewEncoder(&buf).Encode(clientHello)
	fatal(err, "Failed to construct ClientHello")
	conn.Write(buf.Bytes())
	fmt.Println("Client sent ClientHello")
	transcript = append(transcript, buf.Bytes()...)
	buf.Reset()

	// 3. Receive ServerHello
	var serverHello ServerHello
	err = dec.Decode(&serverHello)
	fatal(err, "Failed to decode ServerHello")
	fmt.Println("Client received ServerHello")
	json.NewEncoder(&buf).Encode(serverHello)
	transcript = append(transcript, buf.Bytes()...)
	buf.Reset()

	// 4. Compute shared secret
	serverPublicKey, err := ecdh.X25519().NewPublicKey(serverHello.EphemeralPublicKey)
	fatal(err, "Invalid ephemeral public key from ServerHello")
	sharedSecret, err := privateKey.ECDH(serverPublicKey)
	fatal(err, "Failed to compute shared secret on client")
	fmt.Println("Computed shared secret on client: ", string(sharedSecret))

	// 5. Derive keys
	clientWriteKey, serverWriteKey, clientNonce, serverNonce, err := DeriveKeys(sharedSecret)
	fatal(err, "Failed to derive keys")
	fmt.Println("Computed clientWriteKey on client: ", string(clientWriteKey))
	fmt.Println("Computed serverWriteKey on client: ", string(serverWriteKey))
	fmt.Println("Computed clientNonce on client: ", string(clientNonce))
	fmt.Println("Computed serverNonce on client: ", string(serverNonce))

	// 6. Receive and decrypt cert, verify with mock local CA store
	var encryptedCert EncryptedCertificate
	err = dec.Decode(&encryptedCert)
	fatal(err, "Failed to decode encryptedCert")
	fmt.Println("Client received encryptedCert")

	certBytes, err := decryptAESGCM(serverWriteKey, serverNonce, encryptedCert.Bytes)
	fatal(err, "Failed to decrypt encryptedCert")
	transcript = append(transcript, certBytes...)

	cert, err := x509.ParseCertificate(certBytes)
	fatal(err, "Failed to parse certificate in client")

	err = VerifyCert(caStore, cert)
	fatal(err, "Failed to verify certificate")

	// 7. Receive and verify CertificateVerify
	var certVerify CertificateVerify
	err = dec.Decode(&certVerify)
	fatal(err, "Failed to receive CertificateVerify")
	signature, err := decryptAESGCM(serverWriteKey, serverNonce, certVerify.EncryptedSignature)
	fatal(err, "Failed to decrypt CertificateVerify")
	hash := sha256.Sum256(transcript)
	err = rsa.VerifyPKCS1v15(cert.PublicKey.(*rsa.PublicKey), crypto.SHA256, hash[:], signature)
	fatal(err, "Failed to verify CertificateVerify - handshake has been tampered with")
	fmt.Println("Client verified CertificateVerify")
	_ = json.NewEncoder(&buf).Encode(certVerify)
	transcript = append(transcript, buf.Bytes()...)

	// 8. Receive and verify server Finished
	var finished Finished
	err = dec.Decode(&finished)
	fatal(err, "Failed to decode Finished")
	serverFinished, err := decryptAESGCM(serverWriteKey, serverNonce, finished.EncryptedVerifyData)
	fatal(err, "Failed to decrypt Finished")
	finishedKey, err := hkdf.Key(sha256.New, sharedSecret, nil, "finished", 32)
	fatal(err, "Failed to derive finishedKey")
	h := hmac.New(sha256.New, finishedKey)
	h.Write(transcript)
	clientFinished := h.Sum(nil)
	if !hmac.Equal(serverFinished, clientFinished) {
		fatal(nil, "Client failed to verify Finished - handshake tampered")
	}
	fmt.Println("Client verified Finished")

	// 9. Send own Finished
	encryptedClientFinished, err := encryptAESGCM(clientWriteKey, clientNonce, clientFinished)
	fatal(err, "Failed to encrypt Finished on client")
	err = json.NewEncoder(conn).Encode(Finished{EncryptedVerifyData: encryptedClientFinished})
	fatal(err, "Client failed to send Finished")
	fmt.Println("Client sent Finished")

	// Wait for server to finish
	time.Sleep(time.Second)
}

func DeriveKeys(sharedSecret []byte) (clientWriteKey, serverWriteKey, clientNonce, serverNonce []byte, err error) {
	clientWriteKey, err = hkdf.Key(sha256.New, sharedSecret, nil, "client key", 16)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("failed to derive keys: %w", err)
	}
	serverWriteKey, err = hkdf.Key(sha256.New, sharedSecret, nil, "server key", 16)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("failed to derive keys: %w", err)
	}
	clientNonce, err = hkdf.Key(sha256.New, sharedSecret, nil, "client iv", 12)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("failed to derive keys: %w", err)
	}
	serverNonce, err = hkdf.Key(sha256.New, sharedSecret, nil, "server iv", 12)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("failed to derive keys: %w", err)
	}
	return clientWriteKey, serverWriteKey, clientNonce, serverNonce, nil
}

func VerifyCert(caStore []*x509.Certificate, cert *x509.Certificate) error {
	for _, caCert := range caStore {
		certHash := sha256.Sum256(cert.RawTBSCertificate)
		caPublicKey := caCert.PublicKey.(*rsa.PublicKey)
		err := rsa.VerifyPKCS1v15(caPublicKey, crypto.SHA256, certHash[:], cert.Signature)
		if err == nil {
			now := time.Now()
			if now.Before(cert.NotBefore) || now.After(cert.NotAfter) {
				return fmt.Errorf("Cert expired or not yet valid")
			}
			err = cert.VerifyHostname("www.wowthedoge.com")
			if err != nil {
				return fmt.Errorf("Hostname mismatch: %w", err)
			}
			return nil
		}
	}
	return fmt.Errorf("Cert not signed by any trusted CA")
}
