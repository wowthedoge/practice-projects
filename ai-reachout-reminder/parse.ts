import Anthropic from "@anthropic-ai/sdk";
import type { Connection } from "./types.js";

const PARSE_PROMPT = `Extract connection details from this freeform text. Return valid JSON matching this schema:

{
  "name": string,
  "phone": string | null,
  "summary": string,
  "events": [{ "date": "ISO 8601 UTC", "name": string, "summary": string }]
}

Rules:
- "name" is required. If not found, set to "Unknown".
- "phone" can be null if not mentioned.
- "summary" should capture everything notable about the person (role, company, interests, personal details).
- "events" should capture any mentioned dates (birthdays, conferences, meetings). If a year isn't specified, assume the current or next occurrence.
- Today's date is ${new Date().toISOString().split("T")[0]}.
- Return ONLY the JSON object, no markdown fences or explanation.`;

export async function parseConnection(
  client: Anthropic,
  text: string
): Promise<Omit<Connection, "_id">> {
  const response = await client.messages.create({
    model: "claude-haiku-4-5",
    max_tokens: 1024,
    system: PARSE_PROMPT,
    messages: [{ role: "user", content: text }],
  });

  const raw =
    response.content[0].type === "text" ? response.content[0].text : "{}";
  return JSON.parse(raw);
}
