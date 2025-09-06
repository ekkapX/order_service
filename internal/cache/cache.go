package cache

import (
	"context"
	"database/sql"
	"encoding/json"
	"l0/internal/model"
	"time"

	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
)

type Cache struct {
	client *redis.Client
	logger *zap.Logger
}

func NewCache(addr string, logger *zap.Logger) *Cache {
	client := redis.NewClient(&redis.Options{
		Addr:         addr,
		Password:     "",
		DB:           0,
		DialTimeout:  10 * time.Second,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		MaxRetries:   3,
	})

	ctx := context.Background()
	for i := 0; i < 3; i++ {
		_, err := client.Ping(ctx).Result()
		if err == nil {
			logger.Info("Connected to Redis", zap.String("addr", addr))
			break
		}
		logger.Warn("Failed to connet to Redis, retryng...", zap.Error(err), zap.Int("attempt", i+1))
		time.Sleep(2 * time.Second)
	}
	return &Cache{client: client, logger: logger}
}

func (c *Cache) SaveOrder(ctx context.Context, order model.Order) error {
	data, err := json.Marshal(order)
	if err != nil {
		c.logger.Error("Faiiled to marshal order for cache", zap.Error(err), zap.String("order_uid", order.OrderUID))
		return err
	}
	if err := c.client.Set(ctx, order.OrderUID, data, 0).Err(); err != nil {
		c.logger.Error("Failde to save order to Redis", zap.Error(err), zap.String("order_uid", order.OrderUID))
		return err
	}

	c.logger.Info("Order cached in Redis", zap.String("order_uid", order.OrderUID))
	return nil
}

func (c *Cache) GetOrder(ctx context.Context, orderUID string) (*model.Order, error) {
	data, err := c.client.Get(ctx, orderUID).Bytes()
	if err == redis.Nil {
		c.logger.Info("Order not found in cache", zap.String("order_uid", orderUID))
		return nil, nil
	}
	if err != nil {
		c.logger.Error("Failed to get order from Redis", zap.Error(err), zap.String("order_uid", orderUID))
		return nil, nil
	}

	var order model.Order
	if err := json.Unmarshal(data, &order); err != nil {
		c.logger.Error("Failde to unmarshal order from cache", zap.Error(err), zap.String("order_uid", orderUID))
		return nil, err
	}
	c.logger.Info("Order retrieved from Redis", zap.String("order_uid", orderUID))
	return &order, nil
}

func (c *Cache) RestoreFromDB(ctx context.Context, dbConn *sql.DB) error {
	rows, err := dbConn.QueryContext(ctx, `
	SELECT order_uid, track_number, order_entry, locale, internal_signature, customer_id, delivery_service, shardkey, sm_id, date_created, oof_shard
	FROM orders`)
	if err != nil {
		c.logger.Error("Failed to query orders for cache restore", zap.Error(err))
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var order model.Order
		err := rows.Scan(
			&order.OrderUID, &order.TrackNumber, &order.OrderEntry, &order.Locale,
			&order.InternalSignature, &order.CustomerID, &order.DeliveryService,
			&order.Shardkey, &order.SmID, &order.DateCreated, &order.OofShard,
		)
		if err != nil {
			c.logger.Error("Failde to scan order for cache restore", zap.Error(err))
			continue
		}

		err = dbConn.QueryRowContext(ctx, `
			SELECT name, phone, zip, city, adress, region, email
			FROM delivery
			WHERE order_uid = $1`, order.OrderUID).Scan(
			&order.Delivery.Name, &order.Delivery.Phone, &order.Delivery.Zip,
			&order.Delivery.City, &order.Delivery.Adress, &order.Delivery.Region, &order.Delivery.Email,
		)
		if err != nil {
			c.logger.Error("Failed to query delivery for cache restore", zap.Error(err), zap.String("order_uid", order.OrderUID))
			continue
		}

		err = dbConn.QueryRowContext(ctx, `
			SELECT transaction, request_id, currency, provider, amount, payment_dt, bank, delivery_cost, goods_total, custom_fee
			FROM payment
			WHERE order_uid = $1`, order.OrderUID).Scan(
			&order.Payment.Transaction, &order.Payment.RequestID, &order.Payment.Currency,
			&order.Payment.Provider, &order.Payment.Amount, &order.Payment.PaymentDt,
			&order.Payment.Bank, &order.Payment.DeliveryCost, &order.Payment.GoodsTotal, &order.Payment.CustomFee,
		)
		if err != nil {
			c.logger.Error("Failed to query payment for cache restore", zap.Error(err), zap.String("order_uid", order.OrderUID))
			continue
		}

		itemRows, err := dbConn.QueryContext(ctx, `
			SELECT chrt_id, track_number, price, rid, name, sale, size, total_price, nm_id, brand, status
			FROM items
			WHERE order_uid = $1`, order.OrderUID)
		if err != nil {
			c.logger.Error("Failed to query items for cache restore", zap.Error(err), zap.String("order_uid", order.OrderUID))
			continue
		}
		for itemRows.Next() {
			var item model.Item
			err := itemRows.Scan(
				&item.ChrtID, &item.TrackNumber, &item.Price, &item.Rid, &item.Name,
				&item.Sale, &item.Size, &item.TotalPrice, &item.NmID, &item.Brand, &item.Status,
			)
			if err != nil {
				c.logger.Error("Failed to scan item for cache restore", zap.Error(err), zap.String("order_uid", order.OrderUID))
				continue
			}
			order.Items = append(order.Items, item)
		}
		itemRows.Close()

		if err := c.SaveOrder(ctx, order); err != nil {
			c.logger.Error("Failed to cache order during restore", zap.Error(err))
			continue
		}
	}

	if err := rows.Err(); err != nil {
		c.logger.Error("Error iterating orders for cache restore", zap.Error(err))
		return err
	}

	c.logger.Info("Cache restored from DB")
	return nil
}
