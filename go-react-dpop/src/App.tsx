import { useState, type FormEvent } from "react";
import { generateAndSaveKeyPair, createDPoPProof } from "./dpopUtil";

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

type TokenType = "bearer" | "dpop";

function App() {
  const [username, setUsername] = useState<string>("demo");
  const [password, setPassword] = useState<string>("password");
  const [token, setToken] = useState<string>("");
  const [tokenType, setTokenType] = useState<TokenType | null>(null);
  const [protectedData, setProtectedData] =
    useState<ProtectedDataResponse | null>(null);
  const [error, setError] = useState<string>("");

  // Login with Bearer token (no DPoP)
  const handleLoginBearer = async (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    setError("");
    setProtectedData(null);

    try {
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
      setTokenType("bearer");
      setError("");
    } catch (err) {
      setError(err instanceof Error ? err.message : "An error occurred");
    }
  };

  // Login with DPoP token
  const handleLoginDPoP = async (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    setError("");
    setProtectedData(null);

    try {
      await generateAndSaveKeyPair();

      const url = `${API_URL}/token-dpop`;
      const method = "POST";

      const dpopProof = await createDPoPProof(method, url);

      const response = await fetch(url, {
        method: method,
        headers: {
          "Content-Type": "application/json",
          DPoP: dpopProof,
        },
        body: JSON.stringify({ username, password }),
      });

      if (!response.ok) {
        throw new Error("Login failed");
      }

      const data: TokenResponse = await response.json();
      setToken(data.access_token);
      setTokenType("dpop");
      setError("");
    } catch (err) {
      setError(err instanceof Error ? err.message : "An error occurred");
    }
  };

  // Access protected resource
  const fetchProtectedData = async () => {
    setError("");

    try {
      if (tokenType === "dpop") {
        const method = "GET";
        const url = `${API_URL}/protected-dpop`;
        const dpopProof = await createDPoPProof(method, url);

        const response = await fetch(url, {
          headers: {
            Authorization: `DPoP ${token}`,
            DPoP: dpopProof,
          },
        });

        if (!response.ok) {
          throw new Error("Failed to fetch protected data");
        }

        const data: ProtectedDataResponse = await response.json();
        setProtectedData(data);
      } else {
        // Bearer token
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
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : "An error occurred");
    }
  };

  // Logout
  const handleLogout = (): void => {
    setToken("");
    setTokenType(null);
    setProtectedData(null);
    setError("");
  };

  return (
    <div className="app">
      <h1>OAuth Demo</h1>

      {!token ? (
        <div className="login-section">
          <h2>Login</h2>
          <div className="credentials">
            <div>
              <label>Username</label>
              <input
                type="text"
                placeholder="demo"
                value={username}
                onChange={(e) => setUsername(e.target.value)}
              />
            </div>
            <div>
              <label>Password</label>
              <input
                type="password"
                placeholder="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
              />
            </div>
          </div>
          <div className="login-buttons">
            <form onSubmit={handleLoginBearer} style={{ display: "inline" }}>
              <button type="submit" className="bearer-btn">
                Login (Bearer Token)
              </button>
            </form>
            <form onSubmit={handleLoginDPoP} style={{ display: "inline" }}>
              <button type="submit" className="dpop-btn">
                Login (DPoP Token)
              </button>
            </form>
          </div>
          <p className="hint">
            Bearer tokens can be stolen and reused. DPoP tokens are bound to your key!
          </p>
        </div>
      ) : (
        <div className="authenticated-section">
          <h2>
            Authenticated
            <span className={`token-badge ${tokenType}`}>
              {tokenType === "dpop" ? "🔒 DPoP" : "⚠️ Bearer"}
            </span>
          </h2>
          <div className="token-display">
            <p>Access Token ({tokenType?.toUpperCase()}):</p>
            <code>{token}</code>
          </div>
          {tokenType === "bearer" && (
            <p className="warning">
              ⚠️ This Bearer token can be copied and used from any client!
            </p>
          )}
          {tokenType === "dpop" && (
            <p className="success">
              🔒 This DPoP token is bound to your key - it cannot be stolen!
            </p>
          )}

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
