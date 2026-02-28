import { createServer } from "http";
import { URLSearchParams } from "url";
import { readFileSync } from "fs";
import { dirname, join } from "path";
import { fileURLToPath } from "url";
import Anthropic from "@anthropic-ai/sdk";
import { addConnection, loadConnections } from "./store.js";
import { parseConnection } from "./parse.js";
import { sendWhatsApp } from "./whatsapp.js";

const PORT = parseInt(process.env.PORT || "3000", 10);
const client = new Anthropic();

const __dirname = dirname(fileURLToPath(import.meta.url));
const HTML_PAGE = readFileSync(join(__dirname, "index.html"), "utf-8");

const server = createServer(async (req, res) => {
  if (req.method === "POST" && req.url === "/add") {
    const chunks: Buffer[] = [];
    for await (const chunk of req) chunks.push(chunk as Buffer);
    const text = Buffer.concat(chunks).toString();

    console.log(`[${new Date().toISOString()}] POST /add — "${text.slice(0, 80)}${text.length > 80 ? "..." : ""}"`);

    if (!text.trim()) {
      res.writeHead(400, { "Content-Type": "application/json" });
      res.end(JSON.stringify({ error: "Empty body" }));
      return;
    }

    try {
      const parsed = await parseConnection(client, text);
      const saved = await addConnection(parsed);
      console.log(`[${new Date().toISOString()}] Saved: ${saved.name} (${saved._id})`);
      const { _id, ...rest } = saved;
      res.writeHead(201, { "Content-Type": "application/json" });
      res.end(JSON.stringify(rest, null, 2));
    } catch (err) {
      console.error(`[${new Date().toISOString()}] Error:`, err);
      res.writeHead(500, { "Content-Type": "application/json" });
      res.end(
        JSON.stringify({ error: "Failed to process connection" })
      );
    }
    return;
  }

  // WhatsApp webhook — Twilio sends form-encoded POST
  if (req.method === "POST" && req.url === "/webhook") {
    const chunks: Buffer[] = [];
    for await (const chunk of req) chunks.push(chunk as Buffer);
    const params = new URLSearchParams(Buffer.concat(chunks).toString());

    const body = params.get("Body") || "";
    const from = params.get("From") || ""; // e.g. "whatsapp:+1234567890"

    console.log(`[${new Date().toISOString()}] WhatsApp from ${from}: "${body.slice(0, 80)}${body.length > 80 ? "..." : ""}"`);

    if (!body.trim()) {
      res.writeHead(200);
      res.end();
      return;
    }

    try {
      const parsed = await parseConnection(client, body);
      const saved = await addConnection(parsed);
      console.log(`[${new Date().toISOString()}] Saved: ${saved.name} (${saved._id})`);
      await sendWhatsApp(from, `Saved ${saved.name}!`);
    } catch (err) {
      console.error(`[${new Date().toISOString()}] Webhook error:`, err);
      await sendWhatsApp(from, "Sorry, I couldn't parse that. Try again with a description of your connection.").catch(() => {});
    }

    res.writeHead(200);
    res.end();
    return;
  }

  // API: list all connections
  if (req.method === "GET" && req.url === "/connections") {
    try {
      const connections = await loadConnections();
      const safe = connections.map(({ _id, ...rest }) => rest);
      res.writeHead(200, { "Content-Type": "application/json" });
      res.end(JSON.stringify(safe, null, 2));
    } catch (err) {
      console.error(`[${new Date().toISOString()}] Error loading connections:`, err);
      res.writeHead(500, { "Content-Type": "application/json" });
      res.end(JSON.stringify({ error: "Failed to load connections" }));
    }
    return;
  }

  // Web UI
  if (req.method === "GET" && req.url === "/") {
    res.writeHead(200, { "Content-Type": "text/html" });
    res.end(HTML_PAGE);
    return;
  }

  res.writeHead(404, { "Content-Type": "application/json" });
  res.end(JSON.stringify({ error: "Not found" }));
});

server.listen(PORT, () => {
  console.log(`Server running on http://localhost:${PORT}`);
  console.log(`POST /add — send freeform text to add a connection`);
});
