import * as indexeddb from "./indexeddb";
import * as jose from "jose";

export async function generateAndSaveKeyPair(): Promise<void> {
  console.log("=== Generating keypair...");
  const keyPair = await crypto.subtle.generateKey(
    {
      name: "ECDSA",
      namedCurve: "P-256",
    },
    false,
    ["sign", "verify"]
  );
  const publicKeyJwk = await crypto.subtle.exportKey("jwk", keyPair.publicKey);
  console.log("Generated public key:", publicKeyJwk);
  try {
    console.log("Generated private key:");
    const privateKeyJwk = await crypto.subtle.exportKey(
      "jwk",
      keyPair.privateKey
    );
  } catch (e) {
    console.log(e);
  }
  indexeddb.saveKeyPair(keyPair);
}

export async function createDPoPProof(
  method: string,
  url: string
): Promise<string> {
  console.log(`=== Creating DPoP proof for method ${method} and url ${url}...`);

  const keyPair = await indexeddb.getKeyPair();
  if (!keyPair) {
    throw new Error("No key pair found in IndexedDB. Please generate a key pair first.");
  }

  const publicKeyJwk = await crypto.subtle.exportKey("jwk", keyPair.publicKey);

  const dpopProof = await new jose.SignJWT(
    // Body of the DPoP proof JWT
    {
      htm: method,
      htu: url,
      jti: crypto.randomUUID(),
      iat: Math.floor(Date.now() / 1000),
    }
  )
    .setProtectedHeader(
      // Header of the DPoP proof JWT
      {
        typ: "dpop+jwt",
        alg: "ES256",
        jwk: publicKeyJwk,
      }
    )
    .sign(keyPair.privateKey);

  console.log("Created DPoP proof:", formatJwt(dpopProof));
  return dpopProof;
}

function formatJwt(jwt: string): string {
  const [headerB64, payloadB64, signature] = jwt.split(".");

  const decodeBase64Url = (str: string) => {
    const base64 = str.replace(/-/g, "+").replace(/_/g, "/");
    return JSON.parse(atob(base64));
  };

  const header = JSON.stringify(decodeBase64Url(headerB64), null, 2);
  const payload = JSON.stringify(decodeBase64Url(payloadB64), null, 2);

  return `
  ==================
  Raw: ${jwt}
  
  Header:
  ${header}
  
  Payload:
  ${payload}
  
  Signature: ${signature}
  ==================
  `;
}
