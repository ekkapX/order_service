package cache

import (
	"context"
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
