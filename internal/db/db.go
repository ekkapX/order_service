package db

import (
	"database/sql"
	"fmt"
	"l0/internal/model"

	_ "github.com/lib/pq"
	"go.uber.org/zap"
)

func NewDB(logger *zap.Logger) (*sql.DB, error) {
	db, err := sql.Open("postgres", "host=postgres port=5432 user=user password=password dbname=orders_db sslmode=disable")
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
	defer tx.Rollback()

	_, err = tx.Exec(`
        INSERT INTO orders (
            order_uid, track_number, order_entry, locale, internal_signature, 
            customer_id, delivery_service, shardkey, sm_id, date_created, oof_shard
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		order.OrderUID, order.TrackNumber, order.OrderEntry, order.Locale, order.InternalSignature,
		order.CustomerID, order.DeliveryService, order.Shardkey, order.SmID, order.DateCreated, order.OofShard)
	if err != nil {
		logger.Error("Failed to insert into orders", zap.Error(err), zap.String("order_uid", order.OrderUID))
		return err
	}

	_, err = tx.Exec(`
        INSERT INTO delivery (
            order_uid, name, phone, zip, city, adress, region, email
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		order.OrderUID, order.Delivery.Name, order.Delivery.Phone, order.Delivery.Zip, order.Delivery.City,
		order.Delivery.Adress, order.Delivery.Region, order.Delivery.Email)
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
