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

CREATE TABLE IF NOT EXISTS followed_sellers (
    id          BIGSERIAL    PRIMARY KEY,
    user_id     BIGINT       NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    seller_id   VARCHAR(255) NOT NULL,
    seller_name VARCHAR(255) NOT NULL DEFAULT '',
    followed_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, seller_id)
);
CREATE INDEX IF NOT EXISTS idx_followed_sellers_user_id ON followed_sellers(user_id);

-- Wallet system
ALTER TABLE users ADD COLUMN IF NOT EXISTS is_admin BOOLEAN NOT NULL DEFAULT FALSE;

CREATE TABLE IF NOT EXISTS wallets (
    id         BIGSERIAL    PRIMARY KEY,
    user_id    BIGINT       NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    balance    BIGINT       NOT NULL DEFAULT 0,  -- JPY
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS wallet_transactions (
    id             BIGSERIAL    PRIMARY KEY,
    user_id        BIGINT       NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type           VARCHAR(20)  NOT NULL,  -- recharge, purchase, refund, adjustment
    amount         BIGINT       NOT NULL,  -- positive=credit, negative=debit
    balance_after  BIGINT       NOT NULL,
    description    TEXT         NOT NULL DEFAULT '',
    related_order  VARCHAR(64)  DEFAULT NULL,
    created_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_wallet_tx_user ON wallet_transactions(user_id, created_at DESC);

-- Order system
CREATE TABLE IF NOT EXISTS orders (
    id                 BIGSERIAL    PRIMARY KEY,
    order_number       VARCHAR(64)  NOT NULL UNIQUE,
    user_id            BIGINT       NOT NULL REFERENCES users(id),
    order_state        INT          NOT NULL DEFAULT 0,
    order_total_jp     BIGINT       NOT NULL DEFAULT 0,
    commission_fee_jp  BIGINT       NOT NULL DEFAULT 0,
    shipping_fee_jp    BIGINT       NOT NULL DEFAULT 0,
    order_inprice_jp   BIGINT       NOT NULL DEFAULT 0,
    order_paytype      INT          NOT NULL DEFAULT 0,
    order_remark       TEXT         NOT NULL DEFAULT '',
    order_purchase_type INT         NOT NULL DEFAULT 1,
    created_at         TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_orders_user ON orders(user_id, created_at DESC);

CREATE TABLE IF NOT EXISTS order_details (
    id                 BIGSERIAL    PRIMARY KEY,
    order_id           BIGINT       NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    goods_mid          VARCHAR(255) NOT NULL,
    goods_name         TEXT         NOT NULL DEFAULT '',
    goods_num          INT          NOT NULL DEFAULT 1,
    goods_img          TEXT         NOT NULL DEFAULT '',
    goods_url          TEXT         NOT NULL DEFAULT '',
    goods_amount_jp    BIGINT       NOT NULL DEFAULT 0,
    commission_fee_jp  BIGINT       NOT NULL DEFAULT 0,
    shipping_fee_jp    BIGINT       NOT NULL DEFAULT 0,
    seller_id          VARCHAR(255) NOT NULL DEFAULT '',
    seller_name        VARCHAR(255) NOT NULL DEFAULT '',
    platform           VARCHAR(50)  NOT NULL DEFAULT '',
    condition          VARCHAR(50)  NOT NULL DEFAULT '',
    state              INT          NOT NULL DEFAULT 0,
    created_at         TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_order_details_order ON order_details(order_id);
