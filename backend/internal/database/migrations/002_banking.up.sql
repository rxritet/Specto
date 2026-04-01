-- Migration: Add banking tables (accounts and transfers)
-- 002_banking.up.sql

CREATE TABLE IF NOT EXISTS accounts (
    id         BIGSERIAL    PRIMARY KEY,
    user_id    BIGINT       NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    currency   VARCHAR(3)   NOT NULL DEFAULT 'KZT',
    balance    BIGINT       NOT NULL DEFAULT 0, -- Balance in cents
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS transfers (
    id               BIGSERIAL    PRIMARY KEY,
    sender_account_id   BIGINT       NOT NULL REFERENCES accounts(id),
    receiver_account_id BIGINT       NOT NULL REFERENCES accounts(id),
    amount           BIGINT       NOT NULL, -- Amount in cents
    currency         VARCHAR(3)   NOT NULL,
    description      TEXT         NOT NULL DEFAULT '',
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_accounts_user_id ON accounts(user_id);
CREATE INDEX IF NOT EXISTS idx_transfers_sender ON transfers(sender_account_id);
CREATE INDEX IF NOT EXISTS idx_transfers_receiver ON transfers(receiver_account_id);
