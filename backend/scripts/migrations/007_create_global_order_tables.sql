-- Global Order Records Table
-- Stores the mapping between international orders and local orders
CREATE TABLE IF NOT EXISTS global_order_records (
    id BIGSERIAL PRIMARY KEY,
    request_id VARCHAR(255) NOT NULL UNIQUE,
    global_order_number VARCHAR(255) NOT NULL UNIQUE,
    global_account_id VARCHAR(255) NOT NULL, -- 国际版用户ID，必填
    order_id BIGINT NOT NULL REFERENCES orders(id),
    order_number VARCHAR(255) NOT NULL,
    global_order_pay_type INTEGER NOT NULL,
    sync_time TIMESTAMP NOT NULL DEFAULT NOW(),
    payment_sync_state INTEGER NOT NULL DEFAULT 0, -- 0=not paid, 1=paid, 2=exception
    payment_number VARCHAR(255),
    payment_request_id VARCHAR(255),
    payment_sync_time TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Global Order Payments Table
-- Stores payment details for international orders
CREATE TABLE IF NOT EXISTS global_order_payments (
    id BIGSERIAL PRIMARY KEY,
    order_id BIGINT NOT NULL REFERENCES orders(id),
    payment_number VARCHAR(255) NOT NULL UNIQUE,
    pay_channel VARCHAR(50) NOT NULL,
    pay_currency VARCHAR(10) NOT NULL, -- 实际支付币种，例如JPY、USD
    pay_amount BIGINT NOT NULL, -- in cents/smallest unit
    pay_time TIMESTAMP NOT NULL,
    operator VARCHAR(255),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_global_order_records_request_id ON global_order_records(request_id);
CREATE INDEX IF NOT EXISTS idx_global_order_records_global_order_number ON global_order_records(global_order_number);
CREATE INDEX IF NOT EXISTS idx_global_order_records_global_account_id ON global_order_records(global_account_id);
CREATE INDEX IF NOT EXISTS idx_global_order_records_order_id ON global_order_records(order_id);
CREATE INDEX IF NOT EXISTS idx_global_order_records_payment_number ON global_order_records(payment_number);
CREATE INDEX IF NOT EXISTS idx_global_order_payments_order_id ON global_order_payments(order_id);
CREATE INDEX IF NOT EXISTS idx_global_order_payments_payment_number ON global_order_payments(payment_number);

-- Comments
COMMENT ON TABLE global_order_records IS 'Mapping between international orders and local orders';
COMMENT ON TABLE global_order_payments IS 'Payment details for international orders';
COMMENT ON COLUMN global_order_records.global_account_id IS '国际版用户ID，必填';
COMMENT ON COLUMN global_order_records.payment_sync_state IS '0=not paid, 1=paid, 2=exception';
COMMENT ON COLUMN global_order_payments.pay_currency IS '实际支付币种，例如JPY、USD';
COMMENT ON COLUMN global_order_payments.pay_amount IS 'Amount in cents/smallest currency unit';
