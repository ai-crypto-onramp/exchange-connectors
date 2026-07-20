CREATE TABLE IF NOT EXISTS orders (
    venue_order_id  TEXT        PRIMARY KEY,
    client_order_id TEXT        NOT NULL,
    venue           TEXT        NOT NULL,
    pair            TEXT        NOT NULL,
    side            TEXT        NOT NULL,
    order_type      TEXT        NOT NULL,
    status          TEXT        NOT NULL,
    filled_qty      NUMERIC     NOT NULL DEFAULT 0,
    avg_price       NUMERIC     NOT NULL DEFAULT 0,
    quantity        NUMERIC     NOT NULL DEFAULT 0,
    price           NUMERIC     NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (venue, client_order_id)
);
CREATE INDEX IF NOT EXISTS orders_client_order_id_idx ON orders (client_order_id);
CREATE INDEX IF NOT EXISTS orders_venue_idx ON orders (venue);

CREATE TABLE IF NOT EXISTS fills (
    id              BIGSERIAL   PRIMARY KEY,
    venue_order_id  TEXT        NOT NULL,
    trade_id        TEXT        NOT NULL,
    venue           TEXT        NOT NULL,
    pair            TEXT        NOT NULL,
    price           NUMERIC     NOT NULL DEFAULT 0,
    quantity        NUMERIC     NOT NULL DEFAULT 0,
    fee             NUMERIC     NOT NULL DEFAULT 0,
    fee_asset       TEXT        NOT NULL DEFAULT '',
    ts              TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (venue, trade_id)
);
CREATE INDEX IF NOT EXISTS fills_venue_order_id_idx ON fills (venue_order_id);

CREATE TABLE IF NOT EXISTS idempotency_keys (
    key        TEXT        PRIMARY KEY,
    response   JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at TIMESTAMPTZ NOT NULL
);
CREATE INDEX IF NOT EXISTS idempotency_keys_expires_at_idx ON idempotency_keys (expires_at);