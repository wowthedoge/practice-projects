import { createServer } from "http";
import { randomInt } from "crypto";
import { URLSearchParams } from "url";
import { readFileSync } from "fs";
import { dirname, join } from "path";
import { fileURLToPath } from "url";
import Anthropic from "@anthropic-ai/sdk";
import {
  addConnection,
  loadConnections,
  createSession,
  getSession,
  deleteSession,
  storeVerificationCode,
  verifyCode,
} from "./store.js";
import { parseConnection } from "./parse.js";
import { sendWhatsApp } from "./whatsapp.js";

const PORT = parseInt(process.env.PORT || "3000", 10);
const client = new Anthropic();

const __dirname = dirname(fileURLToPath(import.meta.url));
const HTML_PAGE = readFileSync(join(__dirname, "index.html"), "utf-8");

function readBody(req: import("http").IncomingMessage): Promise<string> {
  return new Promise((resolve) => {
    const chunks: Buffer[] = [];
    req.on("data", (chunk: Buffer) => chunks.push(chunk));
    req.on("end", () => resolve(Buffer.concat(chunks).toString()));
  });
}

function getCookie(req: import("http").IncomingMessage, name: string): string | null {
  const header = req.headers.cookie || "";
  const match = header.match(new RegExp(`(?:^|;\\s*)${name}=([^;]*)`));
  return match ? match[1] : null;
}

function jsonResponse(res: import("http").ServerResponse, status: number, data: unknown) {
  res.writeHead(status, { "Content-Type": "application/json" });
  res.end(JSON.stringify(data));
}

const server = createServer(async (req, res) => {
  // --- POST /add (curl fallback, no auth) ---
  if (req.method === "POST" && req.url === "/add") {
    const text = await readBody(req);

    console.log(`[${new Date().toISOString()}] POST /add — "${text.slice(0, 80)}${text.length > 80 ? "..." : ""}"`);

    if (!text.trim()) {
      jsonResponse(res, 400, { error: "Empty body" });
      return;
    }

    try {
      const parsed = await parseConnection(client, text);
      const saved = await addConnection({ ...parsed, owner: "anonymous" });
      console.log(`[${new Date().toISOString()}] Saved: ${saved.name} (${saved._id})`);
      const { _id, owner, ...rest } = saved;
      jsonResponse(res, 201, rest);
    } catch (err) {
      console.error(`[${new Date().toISOString()}] Error:`, err);
      jsonResponse(res, 500, { error: "Failed to process connection" });
    }
    return;
  }

  // --- POST /webhook (WhatsApp) ---
  if (req.method === "POST" && req.url === "/webhook") {
    const body_raw = await readBody(req);
    const params = new URLSearchParams(body_raw);

    const body = params.get("Body") || "";
    const from = params.get("From") || "";

    console.log(`[${new Date().toISOString()}] WhatsApp from ${from}: "${body.slice(0, 80)}${body.length > 80 ? "..." : ""}"`);

    if (!body.trim()) {
      res.writeHead(200);
      res.end();
      return;
    }

    try {
      const parsed = await parseConnection(client, body);
      const saved = await addConnection({ ...parsed, owner: from });
      console.log(`[${new Date().toISOString()}] Saved: ${saved.name} (${saved._id}) for ${from}`);
      await sendWhatsApp(from, `Saved ${saved.name}!`);
    } catch (err) {
      console.error(`[${new Date().toISOString()}] Webhook error:`, err);
      await sendWhatsApp(from, "Sorry, I couldn't parse that. Try again with a description of your connection.").catch(() => {});
    }

    res.writeHead(200);
    res.end();
    return;
  }

  // --- POST /auth/send-code ---
  if (req.method === "POST" && req.url === "/auth/send-code") {
    try {
      const { phone } = JSON.parse(await readBody(req));
      if (!phone) {
        jsonResponse(res, 400, { error: "Phone required" });
        return;
      }

      const code = String(randomInt(100000, 999999));
      const whatsappPhone = phone.startsWith("whatsapp:") ? phone : `whatsapp:${phone}`;
      storeVerificationCode(whatsappPhone, code);
      await sendWhatsApp(whatsappPhone, `Your verification code is: ${code}`);
      console.log(`[${new Date().toISOString()}] Sent verification code to ${whatsappPhone}`);
      jsonResponse(res, 200, { ok: true });
    } catch (err) {
      console.error(`[${new Date().toISOString()}] Send code error:`, err);
      jsonResponse(res, 500, { error: "Failed to send code" });
    }
    return;
  }

  // --- POST /auth/verify ---
  if (req.method === "POST" && req.url === "/auth/verify") {
    try {
      const { phone, code } = JSON.parse(await readBody(req));
      const whatsappPhone = phone.startsWith("whatsapp:") ? phone : `whatsapp:${phone}`;

      if (!verifyCode(whatsappPhone, code)) {
        jsonResponse(res, 401, { error: "Invalid or expired code" });
        return;
      }

      const token = await createSession(whatsappPhone);
      res.writeHead(200, {
        "Content-Type": "application/json",
        "Set-Cookie": `session=${token}; Path=/; HttpOnly; SameSite=Strict; Max-Age=${7 * 24 * 3600}`,
      });
      res.end(JSON.stringify({ ok: true }));
    } catch (err) {
      console.error(`[${new Date().toISOString()}] Verify error:`, err);
      jsonResponse(res, 500, { error: "Verification failed" });
    }
    return;
  }

  // --- POST /auth/logout ---
  if (req.method === "POST" && req.url === "/auth/logout") {
    const token = getCookie(req, "session");
    if (token) await deleteSession(token);
    res.writeHead(200, {
      "Content-Type": "application/json",
      "Set-Cookie": "session=; Path=/; HttpOnly; Max-Age=0",
    });
    res.end(JSON.stringify({ ok: true }));
    return;
  }

  // --- GET /auth/me ---
  if (req.method === "GET" && req.url === "/auth/me") {
    const token = getCookie(req, "session");
    if (!token) {
      jsonResponse(res, 401, { error: "Not authenticated" });
      return;
    }
    const session = await getSession(token);
    if (!session) {
      jsonResponse(res, 401, { error: "Not authenticated" });
      return;
    }
    jsonResponse(res, 200, { phone: session.phone });
    return;
  }

  // --- GET /connections (requires auth) ---
  if (req.method === "GET" && req.url === "/connections") {
    const token = getCookie(req, "session");
    if (!token) {
      jsonResponse(res, 401, { error: "Not authenticated" });
      return;
    }
    const session = await getSession(token);
    if (!session) {
      jsonResponse(res, 401, { error: "Not authenticated" });
      return;
    }

    try {
      const connections = await loadConnections(session.phone);
      const safe = connections.map(({ _id, owner, ...rest }) => rest);
      jsonResponse(res, 200, safe);
    } catch (err) {
      console.error(`[${new Date().toISOString()}] Error loading connections:`, err);
      jsonResponse(res, 500, { error: "Failed to load connections" });
    }
    return;
  }

  // --- Web UI ---
  if (req.method === "GET" && req.url === "/") {
    res.writeHead(200, { "Content-Type": "text/html" });
    res.end(HTML_PAGE);
    return;
  }

  jsonResponse(res, 404, { error: "Not found" });
});

server.listen(PORT, () => {
  console.log(`Server running on http://localhost:${PORT}`);
});
