CREATE TABLE IF NOT EXISTS waitlist (
    id         TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    email      TEXT UNIQUE NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
