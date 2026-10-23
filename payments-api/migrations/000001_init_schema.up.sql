-- 000001_init_schema.up.sql
-- Initial schema definition for Payment Processing Service

CREATE TABLE merchants (
    id         TEXT PRIMARY KEY,
    name       TEXT NOT NULL,
    status     TEXT NOT NULL CHECK (status IN ('ACTIVE', 'INACTIVE')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE payment_intents (
    id              UUID PRIMARY KEY,
    merchant_id     TEXT        NOT NULL REFERENCES merchants (id) ON DELETE CASCADE,
    order_reference TEXT        NOT NULL,
    amount          BIGINT      NOT NULL CHECK (amount > 0),
    currency        TEXT        NOT NULL CHECK (currency = 'CLP'),
    metadata        JSONB       NOT NULL DEFAULT '{}'::jsonb,
    correlation_id  TEXT        NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (merchant_id, order_reference)
);

CREATE TABLE transactions (
    id                 UUID PRIMARY KEY,
    payment_intent_id  UUID        NOT NULL REFERENCES payment_intents (id) ON DELETE CASCADE,
    merchant_id        TEXT        NOT NULL REFERENCES merchants (id) ON DELETE CASCADE,
    idempotency_key    TEXT        NOT NULL,
    payload_hash       TEXT        NOT NULL,
    status             TEXT        NOT NULL CHECK (status IN ('PENDING', 'PROCESSING', 'APPROVED', 'DECLINED', 'FAILED')),
    card_bin           TEXT        NOT NULL,
    card_last4         TEXT        NOT NULL,
    decline_source     TEXT        CHECK (decline_source IN ('INTERNAL', 'ACQUIRER')),
    result_code        TEXT,
    result_message     TEXT,
    authorization_code TEXT,
    correlation_id     TEXT        NOT NULL,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    completed_at       TIMESTAMPTZ
);

-- Concurrency and Idempotency Constraints
CREATE UNIQUE INDEX uq_tx_idempotency ON transactions (merchant_id, idempotency_key);
CREATE UNIQUE INDEX uq_tx_one_approved ON transactions (payment_intent_id) WHERE status = 'APPROVED';
CREATE UNIQUE INDEX uq_tx_one_active ON transactions (payment_intent_id) WHERE status IN ('PENDING', 'PROCESSING');

-- Operational and Query Indexes
CREATE INDEX ix_tx_list ON transactions (merchant_id, status, created_at DESC, id DESC);
CREATE INDEX ix_tx_created ON transactions (created_at DESC, id DESC);

CREATE TABLE transaction_events (
    id               BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    transaction_id   UUID        NOT NULL REFERENCES transactions (id) ON DELETE CASCADE,
    event_type       TEXT        NOT NULL CHECK (event_type IN ('TRANSACTION_CREATED', 'STATUS_CHANGED', 'ACQUIRER_ATTEMPT')),
    from_status      TEXT,
    to_status        TEXT,
    acquirer_attempt INT,
    latency_ms       INT,
    code             TEXT,
    message          TEXT,
    details          JSONB,
    correlation_id   TEXT        NOT NULL,
    occurred_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX ix_events_tx ON transaction_events (transaction_id, id);
