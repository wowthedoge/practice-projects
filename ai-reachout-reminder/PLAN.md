# AI Reachout Reminder - MVP Plan

## What it does
1. **Server** (`npm start`): HTTP server with `POST /add` and `POST /webhook` (Twilio WhatsApp). Accepts freeform text, Claude parses it into a Connection, saves to MongoDB.
2. **Daily check** (`npm run check`): Reads connections from MongoDB, finds events needing reachout today, Claude crafts personalized messages, sends via WhatsApp.

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
- Twilio (WhatsApp messaging)

## Logic
1. Load connections from MongoDB
2. Find events needing reachout today:
   - **Birthdays**: reachout on the same day (`daysSince === 0`)
   - **Other events**: reachout 3 days after (`daysSince === 3`)
3. For each, call Claude Haiku to craft a personalized message (birthday → wish well, post-event → ask how it went)
4. Send personalized message via WhatsApp (also prints to terminal)

## Adding connections
Two ways to add connections:

1. **WhatsApp** (primary): Text the bot on WhatsApp. Twilio forwards to `POST /webhook`, Claude parses, saves to MongoDB, replies with confirmation.
2. **curl** (fallback): `POST /add` with freeform text body.

```bash
curl -X POST http://localhost:3000/add \
  -d "Met Sarah Chen at re:Invent, VP Eng at Acme. Birthday March 5. Phone 555-0101."
```

## Files
- `server.ts` — HTTP server with POST /add and POST /webhook
- `whatsapp.ts` — Twilio WhatsApp send helper
- `index.ts` — daily reachout checker
- `parse.ts` — Claude-powered freeform text → Connection parser
- `store.ts` — MongoDB connection and CRUD
- `types.ts` — shared types
- `package.json` — anthropic sdk, mongodb, twilio, tsx

## Env vars
- `ANTHROPIC_API_KEY` — Claude API key
- `MONGO_URL` — MongoDB connection string (default: `mongodb://localhost:27017`)
- `TWILIO_ACCOUNT_SID` — Twilio account SID
- `TWILIO_AUTH_TOKEN` — Twilio auth token
- `TWILIO_WHATSAPP_NUMBER` — Twilio sandbox/business number (e.g., `whatsapp:+14155238886`)

## Twilio WhatsApp setup
1. Sign up at twilio.com
2. Go to Messaging → Try it out → Send a WhatsApp message
3. Join the sandbox from your phone (send the join code to the Twilio number)
4. Set the webhook URL in the sandbox config: `https://<your-domain>/webhook`
5. Add `TWILIO_ACCOUNT_SID`, `TWILIO_AUTH_TOKEN`, `TWILIO_WHATSAPP_NUMBER` to Railway env vars
