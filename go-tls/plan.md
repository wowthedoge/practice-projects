# TLS 1.3 Handshake from Scratch — Project Plan

**Goal:** Implement a minimal TLS 1.3 handshake between two local Go processes without using `crypto/tls`. Understand every byte of the handshake by building it.

**Rules:**
- No `crypto/tls` package
- Use Go's crypto primitives: `crypto/ecdh`, `crypto/x509`, `crypto/aes`, `golang.org/x/crypto/hkdf`
- Two separate goroutines — client and server — communicating over a real TCP connection
- Write the logic yourself before asking for help

---

## Architecture

```
main.go        — starts server goroutine, then client
server.go      — TCP listener, handleHandshake()
client.go      — TCP dialer, doHandshake()
messages.go    — message structs and serialization
crypto.go      — key generation, HKDF, AES-GCM helpers
cert.go        — X.509 certificate generation and validation
```

---

## Function signatures

**server.go**
```go
func StartServer()
func handleHandshake(conn net.Conn)
```

**client.go**
```go
func StartClient()
func doHandshake(conn net.Conn)
```

**messages.go**
```go
func encodeClientHello(hello ClientHello) []byte
func decodeClientHello(data []byte) (ClientHello, error)
func encodeServerHello(hello ServerHello) []byte
func decodeServerHello(data []byte) (ServerHello, error)
```

**crypto.go**
```go
func generateX25519Keypair() (*ecdh.PrivateKey, *ecdh.PublicKey, error)
func computeSharedSecret(privateKey *ecdh.PrivateKey, peerPublicKey *ecdh.PublicKey) ([]byte, error)
func deriveKeys(sharedSecret []byte) (clientWriteKey, serverWriteKey []byte, err error)
func encryptAESGCM(key, plaintext, associatedData []byte) ([]byte, error)
func decryptAESGCM(key, ciphertext, associatedData []byte) ([]byte, error)
```

**cert.go**
```go
func generateSelfSignedCert() (*x509.Certificate, *rsa.PrivateKey, error)
func verifyCert(cert *x509.Certificate) error
func signHandshakeTranscript(transcript []byte, privateKey *rsa.PrivateKey) ([]byte, error)
func verifyHandshakeTranscript(transcript, signature []byte, publicKey *rsa.PublicKey) error
```

---

## Message structs

```go
type ClientHello struct {
    Random        [32]byte  // random bytes
    CipherSuites  []uint16  // supported cipher suites
    EphemeralKey  []byte    // X25519 public key
}

type ServerHello struct {
    Random       [32]byte   // random bytes
    CipherSuite  uint16     // chosen cipher suite
    EphemeralKey []byte     // X25519 public key
}

type Certificate struct {
    Raw []byte              // DER encoded certificate
}

type CertificateVerify struct {
    Signature []byte        // signature over handshake transcript
}

type Finished struct {
    VerifyData []byte       // HMAC over handshake transcript
}
```

---

## Part 1 — TCP connection ✓

- [x] Server listens on :8080
- [x] Client connects
- [x] Connection established

---

## Part 2 — ClientHello

1. Client generates X25519 ephemeral keypair
2. Client generates 32 random bytes
3. Client builds ClientHello struct
4. Client serializes it to bytes (JSON for simplicity)
5. Client sends it over the TCP connection
6. Server receives and deserializes it

**What to send:**
- Random bytes
- Supported cipher suites (just `TLS_AES_128_GCM_SHA256 = 0x1301`)
- Ephemeral X25519 public key bytes

---

## Part 3 — ServerHello + key derivation

7. Server generates X25519 ephemeral keypair
8. Server generates 32 random bytes
9. Server builds and sends ServerHello
10. Client receives ServerHello
11. Both sides compute shared secret: `X25519(myPrivateKey, theirPublicKey)`
12. Both sides run HKDF on shared secret to derive:
    - Client write key (AES-GCM key, 16 bytes)
    - Server write key (AES-GCM key, 16 bytes)
    - Client IV (12 bytes)
    - Server IV (12 bytes)

**HKDF derivation labels (TLS 1.3 spec):**
```
client write key → HKDF-Expand(secret, "client key", 16)
server write key → HKDF-Expand(secret, "server key", 16)
client IV        → HKDF-Expand(secret, "client iv",  12)
server IV        → HKDF-Expand(secret, "server iv",  12)
```

---

## Part 4 — Certificate + CertificateVerify

13. Server generates self-signed X.509 certificate
14. Server sends Certificate (DER encoded)
15. Server computes transcript hash — SHA256 over all messages so far
16. Server signs transcript hash with certificate private key
17. Server sends CertificateVerify
18. Client verifies signature using public key from certificate

---

## Part 5 — Finished

19. Server computes Finished — HMAC-SHA256 over full handshake transcript
20. Server sends Finished (encrypted with server write key)
21. Client verifies server Finished
22. Client sends its own Finished (encrypted with client write key)
23. Server verifies client Finished

**Finished construction:**
```
finished_key = HKDF-Expand(secret, "finished", 32)
verify_data  = HMAC-SHA256(finished_key, SHA256(handshake_transcript))
```

---

## Part 6 — Application data

24. Client encrypts "Hello from client" with AES-GCM using client write key
25. Client sends encrypted record
26. Server decrypts with client write key
27. Server prints plaintext
28. Server sends encrypted response
29. Client decrypts and prints

---

## Completion checklist

- [x] TCP connection established
- [ ] ClientHello sent and received
- [ ] ServerHello sent and received
- [ ] Shared secret computed — both sides match
- [ ] HKDF keys derived — both sides match
- [ ] Certificate generated and sent
- [ ] CertificateVerify signature verified
- [ ] Finished messages verified both directions
- [ ] Encrypted application data sent and decrypted

---

## Minimum viable completion (per learning plan)

The completion marker only requires:
- Parts 1, 2, 3 — key exchange
- Part 5 — Finished message

Parts 4 and 6 are stretch goals that make it complete.

---

## Key packages

| Package | Use |
|---|---|
| `crypto/ecdh` | X25519 key generation and DH |
| `golang.org/x/crypto/hkdf` | Key derivation |
| `crypto/aes` + `crypto/cipher` | AES-GCM encryption |
| `crypto/x509` | Certificate generation and parsing |
| `crypto/rsa` | Certificate signing key |
| `crypto/hmac` | Finished message |
| `encoding/json` | Message serialization (simplified) |
| `net` | TCP connection |
