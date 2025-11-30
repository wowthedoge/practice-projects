# A simple payment system built with Go on Stripe.

To get started:
- have postgres instance running, create `payment_system` database
- add `DATABASE_URL`, `STRIPE_SECRET_KEY`, `STRIPE_WEBHOOK_SECRET` in .env
- run `migrate -path db/migrations -database "{{DATABASE_URL}}/payment_system?sslmode=disable" up`
- run `bun run dev`
