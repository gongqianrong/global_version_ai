-- Rakutao Collection Gateway — PostgreSQL schema
-- Run once: docker exec -i <pg_container> psql -U rakutao -d rakutao < scripts/init-postgres.sql

CREATE TABLE IF NOT EXISTS users (
    id            BIGSERIAL    PRIMARY KEY,
    email         VARCHAR(255) NOT NULL UNIQUE,
    nickname      VARCHAR(100) NOT NULL DEFAULT '',
    password_hash VARCHAR(255) NOT NULL,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS cart_items (
    id           BIGSERIAL    PRIMARY KEY,
    user_id      BIGINT       NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    product_id   VARCHAR(255) NOT NULL,
    quantity     INT          NOT NULL DEFAULT 1 CHECK (quantity > 0),
    price_at_add BIGINT       NOT NULL,
    added_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, product_id)
);

CREATE TABLE IF NOT EXISTS favorites (
    id         BIGSERIAL    PRIMARY KEY,
    user_id    BIGINT       NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    product_id VARCHAR(255) NOT NULL,
    added_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, product_id)
);

CREATE TABLE IF NOT EXISTS oauth_accounts (
    id               BIGSERIAL    PRIMARY KEY,
    user_id          BIGINT       NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider         VARCHAR(20)  NOT NULL,
    provider_user_id VARCHAR(255) NOT NULL,
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    UNIQUE(provider, provider_user_id)
);
CREATE INDEX IF NOT EXISTS idx_oauth_accounts_user_id ON oauth_accounts(user_id);
