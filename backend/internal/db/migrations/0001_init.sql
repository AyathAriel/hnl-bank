-- Esquema inicial: usuarios/autenticación y metadatos de cuentas/transacciones.
-- La verdad financiera (balances) vive en TigerBeetle; este esquema es el
-- espejo "humano" para auth, listados rápidos e historial/auditoría.

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE IF NOT EXISTS users (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email         TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    full_name     TEXT NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS accounts (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    account_number        TEXT NOT NULL UNIQUE,
    user_id               UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    tigerbeetle_account_id NUMERIC(39, 0) NOT NULL UNIQUE,
    currency              TEXT NOT NULL DEFAULT 'USD',
    account_type          TEXT NOT NULL CHECK (account_type IN ('checking', 'savings', 'investment')),
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_accounts_user_id ON accounts(user_id);

CREATE TABLE IF NOT EXISTS transactions (
    id                       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tigerbeetle_transfer_id  NUMERIC(39, 0),
    from_account_number      TEXT NOT NULL,
    to_account_number        TEXT NOT NULL,
    amount                   NUMERIC(18, 2) NOT NULL CHECK (amount > 0),
    type                     TEXT NOT NULL CHECK (type IN ('deposit', 'withdrawal', 'transfer', 'internal_transfer')),
    description              TEXT NOT NULL DEFAULT '',
    status                   TEXT NOT NULL CHECK (status IN ('completed', 'failed')) DEFAULT 'completed',
    user_id                  UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at               TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_transactions_from_account ON transactions(from_account_number);
CREATE INDEX IF NOT EXISTS idx_transactions_to_account ON transactions(to_account_number);
CREATE INDEX IF NOT EXISTS idx_transactions_created_at ON transactions(created_at DESC);

CREATE TABLE IF NOT EXISTS revoked_tokens (
    jti        UUID PRIMARY KEY,
    expires_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_revoked_tokens_expires_at ON revoked_tokens(expires_at);
