-- +goose Up
-- +goose StatementBegin
CREATE INDEX IF NOT EXISTS idx_orders_track_number ON orders(track_number);
CREATE INDEX IF NOT EXISTS idx_orders_customer_id ON orders(customer_id);
CREATE INDEX IF NOT EXISTS idx_orders_date_created ON orders(date_created DESC);

CREATE INDEX IF NOT EXISTS idx_delivery_order_uid ON delivery(order_uid);
CREATE INDEX IF NOT EXISTS idx_payment_order_uid ON payment(order_uid);
CREATE INDEX IF NOT EXISTS idx_items_order_uid ON items(order_uid);

CREATE INDEX IF NOT EXISTS idx_orders_customer_date ON orders(customer_id, date_created DESC);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_orders_track_number;
DROP INDEX IF EXISTS idx_orders_customer_id;
DROP INDEX IF EXISTS idx_orders_date_created;
DROP INDEX IF EXISTS idx_delivery_order_uid;
DROP INDEX IF EXISTS idx_payment_order_uid;
DROP INDEX IF EXISTS idx_items_order_uid;
DROP INDEX IF EXISTS idx_orders_customer_date;
-- +goose StatementEnd
