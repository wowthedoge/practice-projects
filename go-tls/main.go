package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"log"
	"math/big"
	"net"
	"time"
)

func main() {

	mockCA := NewCertificateAuthority()

	// Generate server keypair
	serverKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		log.Fatal("Failed to generate RSA keypair")
	}
	fmt.Println("Generated server RSA keypair")

	// Create and send CSR to CA
	csr, err := createCSR(serverKey)
	if err != nil {
		log.Fatal("Failed to create CSR")
	}
	fmt.Println("Created CSR")

	csrResponse := make(chan *x509.Certificate)
	go func() {
		csrResponse <- mockCA.handleCSR(csr)
	}()
	serverCert := <-csrResponse
	fmt.Println("Received signed certificate from CA")
	fmt.Println("Subject:", serverCert.Subject)
	fmt.Println("Issuer:", serverCert.Issuer)
	fmt.Println("NotBefore:", serverCert.NotBefore)
	fmt.Println("NotAfter:", serverCert.NotAfter)
	fmt.Println("PublicKey:", serverCert.PublicKey)

	go StartServer(serverCert, serverKey)
	fmt.Println("Server started")
	time.Sleep(time.Second)
	StartClient([]*x509.Certificate{mockCA.cert})
}

func createCSR(privateKey *rsa.PrivateKey) (*x509.CertificateRequest, error) {
	csrTemplate := x509.CertificateRequest{
		Subject: pkix.Name{
			CommonName:   "localdoge",
			Organization: []string{"Doge, Inc"},
			Country:      []string{"GB"},
		},
		DNSNames:           []string{"wowthedoge.com", "www.wowthedoge.com"},
		IPAddresses:        []net.IP{net.ParseIP("127.0.0.1")},
		SignatureAlgorithm: x509.SHA256WithRSA,
	}

	csrBytes, err := x509.CreateCertificateRequest(rand.Reader, &csrTemplate, privateKey)
	if err != nil {
		return nil, err
	}
	csr, err := x509.ParseCertificateRequest(csrBytes)
	if err != nil {
		return nil, err
	}
	return csr, err
}

func (ca *CertificateAuthority) handleCSR(csr *x509.CertificateRequest) *x509.Certificate {

	certTemplate := x509.Certificate{
		SerialNumber: randomBigInt(),
		Subject:      csr.Subject,
		DNSNames:     csr.DNSNames,
		NotAfter:     time.Now().AddDate(1, 0, 0),
		NotBefore:    time.Now(),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}

	certBytes, _ := x509.CreateCertificate(rand.Reader, &certTemplate, ca.cert, csr.PublicKey, ca.privateKey)
	cert, _ := x509.ParseCertificate(certBytes)
	return cert
}

func NewCertificateAuthority() *CertificateAuthority {
	caKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		log.Fatal("Failed to generate CA RSA keypair")
	}

	caCertTemplate := x509.Certificate{
		SerialNumber: randomBigInt(),
		Subject: pkix.Name{
			CommonName:   "Let's Encrypt",
			Organization: []string{"Let's Encrypt, Inc"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(10, 0, 0),
		KeyUsage:              x509.KeyUsageCertSign,
		IsCA:                  true,
		BasicConstraintsValid: true,
	}

	caCertBytes, _ := x509.CreateCertificate(rand.Reader, &caCertTemplate, &caCertTemplate, &caKey.PublicKey, caKey)
	caCert, _ := x509.ParseCertificate(caCertBytes)
	return &CertificateAuthority{cert: caCert, privateKey: caKey}
}

type CertificateAuthority struct {
	cert       *x509.Certificate
	privateKey *rsa.PrivateKey
}

func randomBigInt() *big.Int {
	bigInt, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	return bigInt
}

func fatal(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %v", msg, err)
	}
}
