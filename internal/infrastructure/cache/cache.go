package cache

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"l0/internal/domain/model"

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
	for i := range 3 {
		_, err := client.Ping(ctx).Result()
		if err == nil {
			logger.Info("Connected to Redis", zap.String("addr", addr))
			break
		}
		logger.Warn("Failed to connect to Redis, retrying...", zap.Error(err), zap.Int("attempt", i+1))
		time.Sleep(2 * time.Second)
	}
	return &Cache{client: client, logger: logger}
}

func (c *Cache) SaveOrder(ctx context.Context, order model.Order) error {
	data, err := json.Marshal(order)
	if err != nil {
		c.logger.Error("Failed to marshal order for cache", zap.Error(err), zap.String("order_uid", order.OrderUID))
		return err
	}
	if err := c.client.Set(ctx, order.OrderUID, data, 0).Err(); err != nil {
		c.logger.Error("Failed to save order to Redis", zap.Error(err), zap.String("order_uid", order.OrderUID))
		return err
	}

	c.logger.Info("Order cached in Redis", zap.String("order_uid", order.OrderUID))
	return nil
}

func (c *Cache) GetOrder(ctx context.Context, orderUID string) (*model.Order, error) {
	data, err := c.client.Get(ctx, orderUID).Bytes()
	if errors.Is(err, redis.Nil) {
		c.logger.Info("Order not found in cache", zap.String("order_uid", orderUID))
		return nil, nil
	}
	if err != nil {
		c.logger.Error("Failed to get order from Redis", zap.Error(err), zap.String("order_uid", orderUID))
		return nil, fmt.Errorf("redis get failed: %w", err)
	}

	var order model.Order
	if err := json.Unmarshal(data, &order); err != nil {
		c.logger.Error("Failde to unmarshal order from cache", zap.Error(err), zap.String("order_uid", orderUID))
		return nil, fmt.Errorf("unmarshal failed: %w", err)
	}
	c.logger.Info("Order retrieved from Redis", zap.String("order_uid", orderUID))
	return &order, nil
}

func (c *Cache) RestoreFromDB(ctx context.Context, dbConn *sql.DB) error {
	query := `
        SELECT 
            o.order_uid, o.track_number, o.entry, o.locale, o.internal_signature,
            o.customer_id, o.delivery_service, o.shardkey, o.sm_id, o.date_created, o.oof_shard,
            d.name, d.phone, d.zip, d.city, d.address, d.region, d.email,
            p.transaction, p.request_id, p.currency, p.provider, p.amount,
            p.payment_dt, p.bank, p.delivery_cost, p.goods_total, p.custom_fee
        FROM orders o
        JOIN delivery d ON o.order_uid = d.order_uid
        JOIN payment p ON o.order_uid = p.order_uid
        ORDER BY o.order_uid`

	rows, err := dbConn.QueryContext(ctx, query)
	if err != nil {
		c.logger.Error("Failed to query orders for cache restore", zap.Error(err))
		return err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			c.logger.Error("Failed to close rows", zap.Error(err))
		}
	}()

	ordersMap := make(map[string]*model.Order)
	var orderUIDs []string

	for rows.Next() {
		var order model.Order
		err := rows.Scan(
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
		)
		if err != nil {
			c.logger.Error("Failed to scan order", zap.Error(err))
			continue
		}
		order.Items = []model.Item{}
		ordersMap[order.OrderUID] = &order
		orderUIDs = append(orderUIDs, order.OrderUID)
	}

	if len(orderUIDs) == 0 {
		c.logger.Info("No orders to restore")
		return nil
	}

	itemsQuery := `
        SELECT order_uid, chrt_id, track_number, price, rid, name, sale, 
               size, total_price, nm_id, brand, status
        FROM items
        WHERE order_uid = ANY($1)
        ORDER BY order_uid, chrt_id`

	itemRows, err := dbConn.QueryContext(ctx, itemsQuery, orderUIDs)
	if err != nil {
		c.logger.Error("Failed to query items", zap.Error(err))
		return err
	}
	defer func() {
		if err := itemRows.Close(); err != nil {
			c.logger.Error("Failed to close item rows", zap.Error(err))
		}
	}()

	for itemRows.Next() {
		var item model.Item
		var orderUID string
		if err := itemRows.Scan(
			&orderUID, &item.ChrtID, &item.TrackNumber, &item.Price, &item.Rid,
			&item.Name, &item.Sale, &item.Size, &item.TotalPrice, &item.NmID,
			&item.Brand, &item.Status,
		); err != nil {
			c.logger.Error("Failed to scan item", zap.Error(err))
			continue
		}
		if order, exists := ordersMap[orderUID]; exists {
			order.Items = append(order.Items, item)
		}
	}

	if err := itemRows.Err(); err != nil {
		c.logger.Error("Failed to scan items", zap.Error(err))
		return err
	}

	for _, order := range ordersMap {
		if err := c.SaveOrder(ctx, *order); err != nil {
			c.logger.Error("Failed to cache order during restore",
				zap.Error(err), zap.String("order_uid", order.OrderUID))
		}
	}

	c.logger.Info("Cache restored from DB", zap.Int("orders_count", len(ordersMap)))
	return nil
}

func (c *Cache) Close() error {
	if err := c.client.Close(); err != nil {
		c.logger.Error("Failed to close Redis client", zap.Error(err))
		return err
	}
	return nil
}
