import { createServer } from "http";
import Anthropic from "@anthropic-ai/sdk";
import { addConnection } from "./store.js";
import { parseConnection } from "./parse.js";

const PORT = parseInt(process.env.PORT || "3000", 10);
const client = new Anthropic();

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
      res.writeHead(201, { "Content-Type": "application/json" });
      res.end(JSON.stringify(saved, null, 2));
    } catch (err) {
      console.error(`[${new Date().toISOString()}] Error:`, err);
      res.writeHead(500, { "Content-Type": "application/json" });
      res.end(
        JSON.stringify({
          error: "Failed to parse",
          details: String(err),
        })
      );
    }
    return;
  }

  res.writeHead(404, { "Content-Type": "application/json" });
  res.end(JSON.stringify({ error: "Not found" }));
});

server.listen(PORT, () => {
  console.log(`Server running on http://localhost:${PORT}`);
  console.log(`POST /add — send freeform text to add a connection`);
});
