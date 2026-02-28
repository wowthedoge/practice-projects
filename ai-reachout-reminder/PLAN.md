# AI Reachout Reminder - MVP Plan

## What it does
1. **Server** (`npm start`): HTTP server with `POST /add` — accepts freeform text, Claude parses it into a Connection, saves to MongoDB.
2. **Daily check** (`npm run check`): Reads connections from MongoDB, finds events needing reachout today, Claude crafts personalized messages, prints to terminal.

## Types
```ts
interface ConnectionEvent {
  date: string;       // ISO 8601 UTC
  name: string;      // event name
  summary: string;
}

interface Connection {
  _id?: ObjectId;     // MongoDB-generated
  name: string;
  phone: string | null;
  summary: string;    // freeform context about the person
  events: ConnectionEvent[];
}
```

## Stack
- TypeScript (`index.ts`, `server.ts`)
- Anthropic SDK (`@anthropic-ai/sdk`)
- MongoDB (via `mongodb` driver)
- tsx to run directly
- No auth, no sending — just print

## Logic
1. Load connections from MongoDB
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
- `store.ts` — MongoDB connection and CRUD
- `types.ts` — shared types
- `package.json` — anthropic sdk, mongodb, tsx

## Env vars
- `ANTHROPIC_API_KEY` — Claude API key
- `MONGO_URL` — MongoDB connection string (default: `mongodb://localhost:27017`)
