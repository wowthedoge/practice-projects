# JWT from Scratch — Project Plan

**Goal:** Implement JWT signing and verification in Go without using any JWT library. Touch HMAC, RSA signatures, Base64URL encoding, and JWKS — the core primitives from Phase 1.

**Rules:**
- No JWT libraries
- Use Go's standard crypto primitives only (`crypto/hmac`, `crypto/rsa`, `crypto/rand`, `crypto/sha256`)
- Write the logic yourself before asking for help
- Spend at least 15-20 minutes stuck before reaching for AI

---

## Part 1 — HS256 (HMAC-SHA256)

### Step 1 — JWT struct
Define a struct for the three JWT parts: header, payload, signature.

Header fields: `alg`, `typ`
Payload fields: `sub`, `iat`, `exp`, plus any custom claims you want

### Step 2 — Base64URL encoding
Implement Base64URL encode and decode.

Note: Base64URL is standard Base64 with `+` replaced by `-`, `/` replaced by `_`, and padding stripped. Go's `encoding/base64` has `base64.RawURLEncoding` which handles this.

### Step 3 — Sign with HS256
1. JSON encode the header → Base64URL encode it
2. JSON encode the payload → Base64URL encode it
3. Concatenate: `encodedHeader + "." + encodedPayload`
4. Compute `HMAC-SHA256(secret, concatenated string)`
5. Base64URL encode the HMAC output
6. Return `encodedHeader + "." + encodedPayload + "." + encodedSignature`

### Step 4 — Verify HS256
1. Split the token on `.` — you get three parts
2. Recompute the HMAC over `part[0] + "." + part[1]`
3. Compare your computed signature against `part[2]`
4. Use `hmac.Equal()` for comparison — not `==` (timing attack)
5. Decode the payload, check `exp` claim against current time

### Step 5 — Claims validation
- Reject if `exp` is in the past
- Reject if `iss` doesn't match expected issuer
- Reject if `aud` doesn't match expected audience

---

## Part 2 — RS256 (RSA signatures)

### Step 6 — Generate RSA keypair
Use `crypto/rsa` and `crypto/rand` to generate a 2048-bit RSA keypair.

### Step 7 — Sign with RS256
1. Same header/payload encoding as HS256
2. Hash `encodedHeader + "." + encodedPayload` with SHA-256
3. Sign the hash with the RSA private key using `rsa.SignPKCS1v15`
4. Base64URL encode the signature

### Step 8 — Verify RS256
1. Split and decode as before
2. Hash `part[0] + "." + part[1]` with SHA-256
3. Verify using `rsa.VerifyPKCS1v15` with the public key
4. Validate claims as before

### The key insight
The verifier no longer needs the secret. Anyone with the public key can verify. Anyone without the private key cannot forge.

---

## Part 3 — JWKS endpoint

### Step 9 — HTTP server
Stand up a simple `net/http` server.

### Step 10 — JWKS endpoint
Expose `GET /jwks` that returns the public key in JWKS format:

```json
{
  "keys": [
    {
      "kty": "RSA",
      "use": "sig",
      "kid": "key-1",
      "n": "<base64url encoded modulus>",
      "e": "<base64url encoded exponent>"
    }
  ]
}
```

RSA public key has two components: modulus `n` and exponent `e`. Both need to be Base64URL encoded.

### Step 11 — Remote verification
Build a verifier that:
1. Fetches the JWKS from the endpoint
2. Finds the key matching the `kid` in the JWT header
3. Reconstructs the RSA public key from `n` and `e`
4. Verifies the token

### The aha moment
Any service can now verify tokens by hitting the JWKS endpoint. No secret ever leaves the auth server. This is exactly how OAuth authorization servers work.

---

## Completion checklist

- [ ] Can sign a JWT with HS256 and verify it
- [ ] Can detect a tampered payload (flip a bit in the payload, verification fails)
- [ ] Can sign a JWT with RS256 and verify it with only the public key
- [ ] JWKS endpoint serves the public key correctly
- [ ] A separate verifier can verify tokens using only the JWKS endpoint
- [ ] Expiry validation works — expired tokens are rejected

---

## Concepts this touches

| Concept | Where |
|---|---|
| Base64 encoding vs encryption | Step 2 — payload is just encoded, not encrypted |
| HMAC | Step 3-4 |
| Shared secret problem | Step 4 — verifier needs the secret |
| RSA signatures | Step 7-8 |
| Public key verification | Step 8 — verifier only needs public key |
| Key distribution | Step 10-11 — JWKS solves it |
| Timing attacks | Step 4 — `hmac.Equal` vs `==` |

---

## Stretch goals

- Add ES256 (ECDSA with P-256 curve) as a third signing algorithm
- Add `kid` (key ID) rotation — serve multiple keys in JWKS, sign with one, verify with the matching one
- Add JWE (encrypted JWT) — understand why encryption is separate from signing
