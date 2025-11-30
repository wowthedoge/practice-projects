
-- 2️⃣ Products (your catalog)
CREATE TABLE IF NOT EXISTS products (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT,
    price_cents INTEGER NOT NULL, -- store in cents to avoid float errors
    currency TEXT NOT NULL DEFAULT 'usd',
    stripe_product_id TEXT, -- e.g. prod_abc123
    stripe_price_id TEXT, -- e.g. price_def456
    created_at TIMESTAMP DEFAULT now()
);

-- 3️⃣ Orders
CREATE TABLE IF NOT EXISTS orders (
    id SERIAL PRIMARY KEY,
    status TEXT NOT NULL DEFAULT 'pending', -- e.g. pending, paid, failed
    payment_id TEXT, -- Stripe PaymentIntent or CheckoutSession ID
    total_amount INTEGER, -- in cents
    currency TEXT NOT NULL DEFAULT 'usd',
    created_at TIMESTAMP DEFAULT now()
);

-- 5️⃣ Processed Stripe Events (for idempotency)
CREATE TABLE IF NOT EXISTS processed_events (
    event_id TEXT PRIMARY KEY,
    created_at TIMESTAMP DEFAULT now()
);

INSERT INTO
    products (
        name,
        description,
        price_cents,
        currency,
        stripe_price_id
    )
VALUES (
        'Hahachipu',
        '',
        500,
        'myr',
        'price_1SSzljENWXLgYDXRd8hMvTAf'
    ),
    (
        'Lalabu',
        '',
        500,
        'myr',
        'price_1SSzlxENWXLgYDXRxl8MN48w'
    );
