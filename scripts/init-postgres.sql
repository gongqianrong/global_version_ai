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

-- Recommendation system
CREATE TABLE IF NOT EXISTS user_preferences (
    id         BIGSERIAL    PRIMARY KEY,
    user_id    BIGINT       NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    category   VARCHAR(100) NOT NULL,
    weight     REAL         NOT NULL DEFAULT 1.0,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, category)
);

CREATE TABLE IF NOT EXISTS search_history (
    id         BIGSERIAL    PRIMARY KEY,
    user_id    BIGINT       NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    keyword    VARCHAR(500) NOT NULL,
    keyword_ja VARCHAR(500) NOT NULL DEFAULT '',
    platform   VARCHAR(50)  NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_search_history_user ON search_history(user_id, created_at DESC);

CREATE TABLE IF NOT EXISTS browsing_history (
    id         BIGSERIAL    PRIMARY KEY,
    user_id    BIGINT       NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    product_id VARCHAR(255) NOT NULL,
    category   VARCHAR(100) NOT NULL DEFAULT '',
    brand      VARCHAR(100) NOT NULL DEFAULT '',
    seller_id  VARCHAR(255) NOT NULL DEFAULT '',
    platform   VARCHAR(50)  NOT NULL DEFAULT '',
    viewed_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_browsing_history_user ON browsing_history(user_id, viewed_at DESC);

CREATE TABLE IF NOT EXISTS user_rec_weights (
    id          BIGSERIAL    PRIMARY KEY,
    user_id     BIGINT       NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    signal_type VARCHAR(20)  NOT NULL,
    dimension   VARCHAR(20)  NOT NULL,
    value       VARCHAR(255) NOT NULL,
    weight      REAL         NOT NULL DEFAULT 0.0,
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, signal_type, dimension, value)
);
CREATE INDEX IF NOT EXISTS idx_rec_weights_user ON user_rec_weights(user_id);

-- ------------------------------------------------------------
-- v1.3.0: 充值订单表
-- ------------------------------------------------------------
CREATE TABLE IF NOT EXISTS recharge_orders (
    id           BIGSERIAL    PRIMARY KEY,
    recharge_no  VARCHAR(64)  NOT NULL UNIQUE,
    user_id      BIGINT       NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    amount_jpy   BIGINT       NOT NULL CHECK (amount_jpy > 0),
    pay_method   VARCHAR(50)  NOT NULL,
    state        INT          NOT NULL DEFAULT 0,  -- 0=待支付 1=已支付 2=支付失败
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_recharge_orders_user ON recharge_orders(user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_recharge_orders_no   ON recharge_orders(recharge_no);

-- ------------------------------------------------------------
-- v1.3.0: 国际运单表
-- ------------------------------------------------------------
CREATE TABLE IF NOT EXISTS waybills (
    id               BIGSERIAL    PRIMARY KEY,
    waybill_no       VARCHAR(64)  NOT NULL UNIQUE,
    user_id          BIGINT       NOT NULL REFERENCES users(id),
    state            INT          NOT NULL DEFAULT 0,
    -- 0=待合单 1=待打包 2=待支付 3=待出库 4=已发货 5=已收货
    shipping_fee_jpy BIGINT       NOT NULL DEFAULT 0,
    carrier          VARCHAR(100) NOT NULL DEFAULT '',
    tracking_no      VARCHAR(100) NOT NULL DEFAULT '',
    tracking_url     TEXT         NOT NULL DEFAULT '',
    wms_waybill_no   VARCHAR(100) NOT NULL DEFAULT '',
    remark           TEXT         NOT NULL DEFAULT '',
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_waybills_user ON waybills(user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_waybills_no   ON waybills(waybill_no);

-- ------------------------------------------------------------
-- v1.3.0: 运单关联订单表（一个运单可合并多个订单）
-- ------------------------------------------------------------
CREATE TABLE IF NOT EXISTS waybill_orders (
    id           BIGSERIAL    PRIMARY KEY,
    waybill_id   BIGINT       NOT NULL REFERENCES waybills(id) ON DELETE CASCADE,
    waybill_no   VARCHAR(64)  NOT NULL,
    order_number VARCHAR(64)  NOT NULL REFERENCES orders(order_number),
    UNIQUE(waybill_no, order_number)
);
CREATE INDEX IF NOT EXISTS idx_waybill_orders_waybill ON waybill_orders(waybill_id);
CREATE INDEX IF NOT EXISTS idx_waybill_orders_order   ON waybill_orders(order_number);

-- ------------------------------------------------------------
-- v1.4.0: 指定购买（Order Link）
-- ------------------------------------------------------------
CREATE TABLE IF NOT EXISTS order_links (
    id            BIGSERIAL    PRIMARY KEY,
    link_no       VARCHAR(64)  NOT NULL UNIQUE,
    user_id       BIGINT       NOT NULL REFERENCES users(id),
    state         INT          NOT NULL DEFAULT 0,  -- 0=待报价 1=已报价 2=已支付 3=已取消
    total_amount  BIGINT       NOT NULL DEFAULT 0,  -- 报价总金额 JPY
    order_number  VARCHAR(64)  DEFAULT NULL,         -- 支付后关联的真实订单号
    remark        TEXT         NOT NULL DEFAULT '',
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_order_links_user ON order_links(user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_order_links_no   ON order_links(link_no);

CREATE TABLE IF NOT EXISTS order_link_items (
    id            BIGSERIAL    PRIMARY KEY,
    order_link_id BIGINT       NOT NULL REFERENCES order_links(id) ON DELETE CASCADE,
    goods_url     TEXT         NOT NULL,
    goods_name    TEXT         NOT NULL DEFAULT '',
    goods_img     TEXT         NOT NULL DEFAULT '',
    quantity      INT          NOT NULL DEFAULT 1 CHECK (quantity > 0),
    unit_price    BIGINT       NOT NULL DEFAULT 0,  -- 管理员报价时填入
    remark        TEXT         NOT NULL DEFAULT '',
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_order_link_items_link ON order_link_items(order_link_id);
