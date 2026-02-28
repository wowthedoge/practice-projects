# AI Reachout Reminder - MVP Plan

## What it does
1. **Server** (`npm start`): HTTP server with `POST /add` — accepts freeform text, Claude parses it into a Connection, saves to JSON.
2. **Daily check** (`npm run check`): Reads connections, finds events needing reachout today, Claude crafts personalized messages, prints to terminal.

## Data: `connections.json`
```json
[{
  "id": "...",
  "name": "...",
  "phone": "...",
  "summary": "...",
  "events": [
    { "date": "2026-03-15T00:00:00Z", "name": "Birthday", "summary": "" },
    { "date": "2026-03-10T00:00:00Z", "name": "AWS Summit", "summary": "they're attending" }
  ]
}]
```

## Types
```ts
interface ConnectionEvent {
  date: string;       // ISO 8601 UTC
  name: string;      // event name
  summary: string;
}

interface Connection {
  id: string;
  name: string;
  phone: string | null;
  summary: string;    // freeform context about the person
  events: ConnectionEvent[];
}
```

## Stack
- TypeScript CLI script (`index.ts`)
- Anthropic SDK (`@anthropic-ai/sdk`)
- tsx to run directly
- No DB, no auth, no sending — just print

## Logic
1. Load `connections.json`
2. Find events needing reachout today:
   - **Birthdays**: reachout on the same day (`daysSince === 0`)
   - **Other events**: reachout 3 days after (`daysSince === 3`)
3. For each, call Claude Haiku to craft a personalized message (birthday → wish well, post-event → ask how it went)
4. Print: name, phone, event, suggested message

## Adding connections
`POST /add` with freeform text body. Claude parses into Connection schema. Missing fields saved as null.

```bash
curl -X POST http://localhost:3000/add \
  -d "Met Sarah Chen at re:Invent, VP Eng at Acme. Birthday March 5. Phone 555-0101."
```

## Files
- `server.ts` — HTTP server with POST /add
- `index.ts` — daily reachout checker
- `parse.ts` — Claude-powered freeform text → Connection parser
- `store.ts` — read/write connections.json
- `types.ts` — shared types
- `connections.json` — seed data
- `package.json` — anthropic sdk, tsx
