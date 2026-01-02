import { useState, type FormEvent } from "react";
import * as indexeddb from "./indexeddb";

const API_URL = "http://localhost:8080";

interface TokenResponse {
  access_token: string;
  token_type: string;
  expires_in: number;
}

interface ProtectedDataResponse {
  message: string;
  data: string;
  username: string;
}

function App() {
  const [username, setUsername] = useState<string>("demo");
  const [password, setPassword] = useState<string>("password");
  const [token, setToken] = useState<string>("");
  const [protectedData, setProtectedData] =
    useState<ProtectedDataResponse | null>(null);
  const [error, setError] = useState<string>("");

  const generateKeyPair = async (): Promise<CryptoKeyPair> => {
    console.log("Generating keypair...")
    const keyPair = await crypto.subtle.generateKey(
      {
        name: "ECDSA",
        namedCurve: "P-256",
      },
      false,
      ["sign", "verify"]
    );
    const publicKeyJwk = await crypto.subtle.exportKey(
      "jwk",
      keyPair.publicKey
    );
    console.log("Public key JWK:", publicKeyJwk);
    console.log("Try export private key:");
    try {
      const privateKeyJwk = await crypto.subtle.exportKey(
        "jwk",
        keyPair.privateKey
      );
    } catch (e) {
      console.log(e);
    }
    indexeddb.saveKeyPair(keyPair);
    return keyPair;
  };

  // Login and get token
  const handleLogin = async (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    setError("");
    setProtectedData(null);

    try {
      await generateKeyPair();
      
      const response = await fetch(`${API_URL}/token`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({ username, password }),
      });

      if (!response.ok) {
        throw new Error("Login failed");
      }

      const data: TokenResponse = await response.json();
      setToken(data.access_token);
      setError("");
    } catch (err) {
      setError(err instanceof Error ? err.message : "An error occurred");
    }
  };

  // Access protected resource
  const fetchProtectedData = async () => {
    setError("");

    try {
      const response = await fetch(`${API_URL}/protected`, {
        headers: {
          Authorization: `Bearer ${token}`,
        },
      });

      if (!response.ok) {
        throw new Error("Failed to fetch protected data");
      }

      const data: ProtectedDataResponse = await response.json();
      setProtectedData(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : "An error occurred");
    }
  };

  // Logout
  const handleLogout = (): void => {
    setToken("");
    setProtectedData(null);
    setError("");
  };

  return (
    <div className="app">
      <h1>OAuth Demo</h1>

      {!token ? (
        <div className="login-section">
          <h2>Login</h2>
          <form onSubmit={handleLogin}>
            Username
            <div>
              <input
                type="text"
                placeholder="demo"
                value={username}
                onChange={(e) => setUsername(e.target.value)}
              />
            </div>
            <div>
              Password
              <input
                type="password"
                placeholder="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
              />
            </div>
            <button type="submit">Login</button>
          </form>
        </div>
      ) : (
        <div className="authenticated-section">
          <h2>Authenticated</h2>
          <div className="token-display">
            <p>Access Token:</p>
            <code>{token}</code>
          </div>

          <div className="actions">
            <button onClick={fetchProtectedData}>Fetch Protected Data</button>
            <button onClick={handleLogout} className="logout">
              Logout
            </button>
          </div>

          {protectedData && (
            <div className="protected-data">
              <h3>Protected Data:</h3>
              <pre>{JSON.stringify(protectedData, null, 2)}</pre>
            </div>
          )}
        </div>
      )}

      {error && <div className="error">{error}</div>}
    </div>
  );
}

export default App;
