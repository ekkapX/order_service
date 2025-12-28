package db

import (
	"database/sql"
	"errors"
	"fmt"

	"l0/internal/domain/model"

	_ "github.com/jackc/pgx/v5/stdlib"
	"go.uber.org/zap"
)

func NewDB(dsn string, logger *zap.Logger) (*sql.DB, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		logger.Error("Failed to open DB connection", zap.Error(err))
		return nil, err
	}
	if err := db.Ping(); err != nil {
		logger.Error("Failed to ping DB", zap.Error(err))
		return nil, err
	}
	logger.Info("DB is ready")
	return db, nil
}

func SaveOrder(db *sql.DB, order model.Order, logger *zap.Logger) error {
	if order.OrderUID == "" {
		logger.Error("Order UID is empty")
		return fmt.Errorf("order_uid cannot be empty")
	}

	logger.Info("Starting to save order", zap.String("order_uid", order.OrderUID))

	var exists bool
	err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM orders WHERE order_uid = $1)", order.OrderUID).Scan(&exists)
	if err != nil {
		logger.Error("Failed to check if order exists", zap.Error(err), zap.String("order_uid", order.OrderUID))
		return err
	}
	if exists {
		logger.Info("Order already exists, skipping", zap.String("order_uid", order.OrderUID))
		return nil
	}

	tx, err := db.Begin()
	if err != nil {
		logger.Error("Failed to start transaction", zap.Error(err))
		return err
	}
	defer func() {
		if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
			logger.Error("Failed to rollback transaction", zap.Error(err))
		}
	}()

	_, err = tx.Exec(`
        INSERT INTO orders (
            order_uid, track_number, entry, locale, internal_signature, 
            customer_id, delivery_service, shardkey, sm_id, date_created, oof_shard
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		order.OrderUID, order.TrackNumber, order.Entry, order.Locale, order.InternalSignature,
		order.CustomerID, order.DeliveryService, order.Shardkey, order.SmID, order.DateCreated, order.OofShard)
	if err != nil {
		logger.Error("Failed to insert into orders", zap.Error(err), zap.String("order_uid", order.OrderUID))
		return err
	}

	_, err = tx.Exec(`
        INSERT INTO delivery (
            order_uid, name, phone, zip, city, address, region, email
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		order.OrderUID, order.Delivery.Name, order.Delivery.Phone, order.Delivery.Zip, order.Delivery.City,
		order.Delivery.Address, order.Delivery.Region, order.Delivery.Email)
	if err != nil {
		logger.Error("Failed to insert into delivery", zap.Error(err), zap.String("order_uid", order.OrderUID))
		return err
	}

	_, err = tx.Exec(`
        INSERT INTO payment (
            order_uid, transaction, request_id, currency, provider, amount, 
            payment_dt, bank, delivery_cost, goods_total, custom_fee
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		order.OrderUID, order.Payment.Transaction, order.Payment.RequestID, order.Payment.Currency, order.Payment.Provider,
		order.Payment.Amount, order.Payment.PaymentDt, order.Payment.Bank, order.Payment.DeliveryCost, order.Payment.GoodsTotal,
		order.Payment.CustomFee)
	if err != nil {
		logger.Error("Failed to insert into payment", zap.Error(err), zap.String("order_uid", order.OrderUID))
		return err
	}

	for _, item := range order.Items {
		_, err = tx.Exec(`
            INSERT INTO items (
                order_uid, chrt_id, track_number, price, rid, name, sale, 
                size, total_price, nm_id, brand, status
            ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
			order.OrderUID, item.ChrtID, item.TrackNumber, item.Price, item.Rid, item.Name, item.Sale,
			item.Size, item.TotalPrice, item.NmID, item.Brand, item.Status)
		if err != nil {
			logger.Error("Failed to insert into items", zap.Error(err), zap.String("order_uid", order.OrderUID))
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		logger.Error("Failed to commit transaction", zap.Error(err), zap.String("order_uid", order.OrderUID))
		return err
	}
	logger.Info("Order saved successfully", zap.String("order_uid", order.OrderUID))
	return nil
}

func GetOrder(dbConn *sql.DB, orderUID string, logger *zap.Logger) (model.Order, error) {
	var order model.Order

	err := dbConn.QueryRow(`
	SELECT order_uid, track_number, entry, locale, internal_signature, customer_id, delivery_service, shardkey, sm_id, date_created, oof_shard
	FROM orders
	WHERE order_uid = $1`,
		orderUID,
	).Scan(&order.OrderUID, &order.TrackNumber, &order.Entry, &order.Locale, &order.InternalSignature, &order.CustomerID, &order.DeliveryService, &order.Shardkey, &order.SmID, &order.DateCreated, &order.OofShard)
	if errors.Is(err, sql.ErrNoRows) {
		logger.Info("Order not found in DB", zap.String("order_uid", orderUID))
		return model.Order{}, fmt.Errorf("order not found")
	} else if err != nil {
		logger.Error("Failed to query order from DB", zap.Error(err), zap.String("order_uid", orderUID))
		return model.Order{}, err
	}

	err = dbConn.QueryRow(`
        SELECT name, phone, zip, city, address, region, email
        FROM delivery
        WHERE order_uid = $1`,
		orderUID,
	).Scan(&order.Delivery.Name, &order.Delivery.Phone, &order.Delivery.Zip, &order.Delivery.City, &order.Delivery.Address, &order.Delivery.Region, &order.Delivery.Email)
	if err != nil {
		logger.Error("Failed to query delivery from DB", zap.Error(err), zap.String("order_uid", orderUID))
		return model.Order{}, err
	}

	err = dbConn.QueryRow(`
        SELECT transaction, request_id, currency, provider, amount, payment_dt, bank, delivery_cost, goods_total, custom_fee
        FROM payment
        WHERE order_uid = $1`,
		orderUID,
	).Scan(&order.Payment.Transaction, &order.Payment.RequestID, &order.Payment.Currency, &order.Payment.Provider, &order.Payment.Amount, &order.Payment.PaymentDt, &order.Payment.Bank, &order.Payment.DeliveryCost, &order.Payment.GoodsTotal, &order.Payment.CustomFee)
	if err != nil {
		logger.Error("Failed to query payment from DB", zap.Error(err), zap.String("order_uid", orderUID))
		return model.Order{}, err
	}

	rows, err := dbConn.Query(`
        SELECT chrt_id, track_number, price, rid, name, sale, size, total_price, nm_id, brand, status
        FROM items
        WHERE order_uid = $1`,
		orderUID,
	)
	if err != nil {
		logger.Error("Failed to query items from DB", zap.Error(err), zap.String("order_uid", orderUID))
		return model.Order{}, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			logger.Error("Failed to close rows", zap.Error(err))
		}
	}()

	for rows.Next() {
		var item model.Item
		err := rows.Scan(&item.ChrtID, &item.TrackNumber, &item.Price, &item.Rid, &item.Name, &item.Sale, &item.Size, &item.TotalPrice, &item.NmID, &item.Brand, &item.Status)
		if err != nil {
			logger.Error("Failde to scan item from DB", zap.Error(err), zap.String("order_uid", orderUID))
			return model.Order{}, err
		}
		order.Items = append(order.Items, item)
	}
	logger.Info("Order retrieved from DB", zap.String("order_uid", orderUID))
	return order, nil
}
