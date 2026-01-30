package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"l0/internal/domain/model"
	"l0/internal/domain/repository"

	"go.uber.org/zap"
)

type OrderRepository struct {
	db     *sql.DB
	logger *zap.Logger
}

func NewOrderRepository(db *sql.DB, logger *zap.Logger) repository.OrderRepository {
	return &OrderRepository{db: db, logger: logger}
}

func (r *OrderRepository) Save(ctx context.Context, order *model.Order) (err error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
		} else if err != nil {
			if rbErr := tx.Rollback(); rbErr != nil {
				r.logger.Error("Failed to rollback transaction",
					zap.Error(rbErr), zap.String("order_uid", order.OrderUID))
			}
		}
	}()

	_, err = tx.ExecContext(ctx, `
        INSERT INTO orders (
            order_uid, track_number, entry, locale, internal_signature, 
            customer_id, delivery_service, shardkey, sm_id, date_created, oof_shard
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		order.OrderUID, order.TrackNumber, order.Entry, order.Locale, order.InternalSignature,
		order.CustomerID, order.DeliveryService, order.Shardkey, order.SmID, order.DateCreated, order.OofShard)
	if err != nil {
		return fmt.Errorf("failed to insert order: %w", err)
	}

	_, err = tx.ExecContext(ctx, `
        INSERT INTO delivery (order_uid, name, phone, zip, city, address, region, email)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		order.OrderUID, order.Delivery.Name, order.Delivery.Phone, order.Delivery.Zip,
		order.Delivery.City, order.Delivery.Address, order.Delivery.Region, order.Delivery.Email)
	if err != nil {
		return fmt.Errorf("failed to insert delivery: %w", err)
	}

	_, err = tx.ExecContext(ctx, `
        INSERT INTO payment (
            order_uid, transaction, request_id, currency, provider, amount, 
            payment_dt, bank, delivery_cost, goods_total, custom_fee
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		order.OrderUID, order.Payment.Transaction, order.Payment.RequestID, order.Payment.Currency,
		order.Payment.Provider, order.Payment.Amount, order.Payment.PaymentDt, order.Payment.Bank,
		order.Payment.DeliveryCost, order.Payment.GoodsTotal, order.Payment.CustomFee)
	if err != nil {
		return fmt.Errorf("failed to insert payment: %w", err)
	}

	for _, item := range order.Items {
		_, err = tx.ExecContext(ctx, `
            INSERT INTO items (
                order_uid, chrt_id, track_number, price, rid, name, sale, 
                size, total_price, nm_id, brand, status
            ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
			order.OrderUID, item.ChrtID, item.TrackNumber, item.Price, item.Rid, item.Name,
			item.Sale, item.Size, item.TotalPrice, item.NmID, item.Brand, item.Status)
		if err != nil {
			return fmt.Errorf("failed to insert item: %w", err)
		}
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	return nil
}

func (r *OrderRepository) GetByUID(ctx context.Context, orderUID string) (*model.Order, error) {
	var order model.Order

	err := r.db.QueryRowContext(ctx, `
        SELECT order_uid, track_number, entry, locale, internal_signature, 
               customer_id, delivery_service, shardkey, sm_id, date_created, oof_shard
        FROM orders WHERE order_uid = $1`, orderUID).Scan(
		&order.OrderUID, &order.TrackNumber, &order.Entry, &order.Locale,
		&order.InternalSignature, &order.CustomerID, &order.DeliveryService,
		&order.Shardkey, &order.SmID, &order.DateCreated, &order.OofShard)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, model.ErrOrderNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get order: %w", err)
	}

	err = r.db.QueryRowContext(ctx, `
        SELECT name, phone, zip, city, address, region, email
        FROM delivery WHERE order_uid = $1`, orderUID).Scan(
		&order.Delivery.Name, &order.Delivery.Phone, &order.Delivery.Zip,
		&order.Delivery.City, &order.Delivery.Address, &order.Delivery.Region, &order.Delivery.Email)
	if err != nil {
		return nil, fmt.Errorf("failed to get delivery: %w", err)
	}

	err = r.db.QueryRowContext(ctx, `
        SELECT transaction, request_id, currency, provider, amount, payment_dt, 
               bank, delivery_cost, goods_total, custom_fee
        FROM payment WHERE order_uid = $1`, orderUID).Scan(
		&order.Payment.Transaction, &order.Payment.RequestID, &order.Payment.Currency,
		&order.Payment.Provider, &order.Payment.Amount, &order.Payment.PaymentDt,
		&order.Payment.Bank, &order.Payment.DeliveryCost, &order.Payment.GoodsTotal, &order.Payment.CustomFee)
	if err != nil {
		return nil, fmt.Errorf("failed to get payment: %w", err)
	}

	rows, err := r.db.QueryContext(ctx, `
        SELECT chrt_id, track_number, price, rid, name, sale, size, 
               total_price, nm_id, brand, status
        FROM items WHERE order_uid = $1`, orderUID)
	if err != nil {
		return nil, fmt.Errorf("failed to get items: %w", err)
	}

	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			r.logger.Warn("Failed to close rows in GetByUID",
				zap.Error(closeErr), zap.String("order_uid", orderUID))
		}
	}()

	for rows.Next() {
		var item model.Item
		if err := rows.Scan(
			&item.ChrtID, &item.TrackNumber, &item.Price, &item.Rid, &item.Name,
			&item.Sale, &item.Size, &item.TotalPrice, &item.NmID, &item.Brand, &item.Status); err != nil {
			return nil, fmt.Errorf("failed to scan item: %w", err)
		}
		order.Items = append(order.Items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error during rows iteration: %w", err)
	}

	return &order, nil
}

func (r *OrderRepository) GetAll(ctx context.Context) ([]*model.Order, error) {
	query := `SELECT 
            o.order_uid, o.track_number, o.entry, o.locale, o.internal_signature,
            o.customer_id, o.delivery_service, o.shardkey, o.sm_id, o.date_created, o.oof_shard,
            d.name, d.phone, d.zip, d.city, d.address, d.region, d.email,
            p.transaction, p.request_id, p.currency, p.provider, p.amount,
            p.payment_dt, p.bank, p.delivery_cost, p.goods_total, p.custom_fee
        FROM orders o
        JOIN delivery d ON o.order_uid = d.order_uid
        JOIN payment p ON o.order_uid = p.order_uid
        ORDER BY o.order_uid`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get orders: %w", err)
	}

	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			r.logger.Warn("Failed to close rows in GetAll", zap.Error(closeErr))
		}
	}()

	ordersMap := make(map[string]*model.Order)
	var orderUIDs []string

	for rows.Next() {
		var order model.Order

		if err := rows.Scan(
			&order.OrderUID, &order.TrackNumber, &order.Entry, &order.Locale,
			&order.InternalSignature, &order.CustomerID, &order.DeliveryService,
			&order.Shardkey, &order.SmID, &order.DateCreated, &order.OofShard,
			&order.Delivery.Name, &order.Delivery.Phone, &order.Delivery.Zip,
			&order.Delivery.City, &order.Delivery.Address, &order.Delivery.Region,
			&order.Delivery.Email,
			&order.Payment.Transaction, &order.Payment.RequestID, &order.Payment.Currency,
			&order.Payment.Provider, &order.Payment.Amount, &order.Payment.PaymentDt,
			&order.Payment.Bank, &order.Payment.DeliveryCost, &order.Payment.GoodsTotal,
			&order.Payment.CustomFee,
		); err != nil {
			return nil, fmt.Errorf("failed to scan order: %w", err)
		}
		order.Items = []model.Item{}
		ordersMap[order.OrderUID] = &order
		orderUIDs = append(orderUIDs, order.OrderUID)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed iterating orders: %w", err)
	}

	if len(orderUIDs) == 0 {
		return []*model.Order{}, nil
	}

	itemsQuery := `
    SELECT order_uid, chrt_id, track_number, price, rid, name, sale, size, total_price, nm_id, brand, status
    FROM items
    WHERE order_uid = ANY($1)
    ORDER BY order_uid, chrt_id`

	itemsRows, err := r.db.QueryContext(ctx, itemsQuery, orderUIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to get items: %w", err)
	}

	defer func() {
		if closeErr := itemsRows.Close(); closeErr != nil {
			r.logger.Warn("Failed to close item rows in GetAll", zap.Error(closeErr))
		}
	}()

	for itemsRows.Next() {
		var item model.Item
		var orderUID string

		if err := itemsRows.Scan(
			&orderUID, &item.ChrtID, &item.TrackNumber, &item.Price, &item.Rid, &item.Name,
			&item.Sale, &item.Size, &item.TotalPrice, &item.NmID, &item.Brand, &item.Status,
		); err != nil {
			return nil, fmt.Errorf("failed to scan item: %w", err)
		}
		if order, exists := ordersMap[orderUID]; exists {
			order.Items = append(order.Items, item)
		}
	}
	if err := itemsRows.Err(); err != nil {
		return nil, fmt.Errorf("failed iterating items: %w", err)
	}

	orders := make([]*model.Order, 0, len(ordersMap))
	for _, uid := range orderUIDs {
		orders = append(orders, ordersMap[uid])
	}

	return orders, nil
}

func (r *OrderRepository) Exists(ctx context.Context, orderUID string) (bool, error) {
	var exists bool
	err := r.db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM orders WHERE order_uid = $1)",
		orderUID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check if order exists: %w", err)
	}
	return exists, nil
}
