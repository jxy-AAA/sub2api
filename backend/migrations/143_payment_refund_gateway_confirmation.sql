ALTER TABLE payment_orders
    ADD COLUMN IF NOT EXISTS refund_gateway_confirmed_at TIMESTAMPTZ NULL,
    ADD COLUMN IF NOT EXISTS refund_gateway_refund_id VARCHAR(128) NULL,
    ADD COLUMN IF NOT EXISTS refund_idempotency_key VARCHAR(128) NULL;

CREATE INDEX IF NOT EXISTS idx_payment_orders_refund_gateway_confirmed_at
    ON payment_orders(refund_gateway_confirmed_at)
    WHERE refund_gateway_confirmed_at IS NOT NULL;
