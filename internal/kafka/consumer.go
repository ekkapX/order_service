package kafka

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"sync"
	"time"

	"l0/internal/applicaiton/validation"
	"l0/internal/cache"
	"l0/internal/db"
	"l0/internal/domain"

	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

func ConsumeOrders(ctx context.Context, wg *sync.WaitGroup, broker, topic, groupID string, dbConn *sql.DB, cache *cache.Cache, logger *zap.Logger) {
	defer wg.Done()
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  []string{broker},
		Topic:    topic,
		GroupID:  groupID,
		MinBytes: 10e3,
		MaxBytes: 10e6,
		MaxWait:  1 * time.Second,
	})
	defer func() {
		if err := reader.Close(); err != nil {
			logger.Error("Failed to close Kafka reader", zap.Error(err))
		}
	}()

	logger.Info("Starting Kafka consumer", zap.String("topic", topic), zap.String("groupID", groupID))

	orderValidator := validation.NewValidator()
	for {
		msg, err := reader.FetchMessage(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				logger.Info("Kafka consumer context canceled, stopping...")
				return
			}
			logger.Error("Fetch message from Kafka", zap.Error(err))
			continue
		}

		var order domain.Order
		if err := json.Unmarshal(msg.Value, &order); err != nil {
			logger.Error("Failed to unmarshal order", zap.Error(err), zap.String("message", string(msg.Value)))
			continue
		}

		if err := orderValidator.ValidateOrder(order); err != nil {
			logger.Warn("Invalid order data", zap.Error(err), zap.String("order_uid", order.OrderUID))
			if err := reader.CommitMessages(ctx, msg); err != nil {
				logger.Error("Failed to commit message", zap.Error(err), zap.String("order_uid", order.OrderUID))
			}
			continue
		}

		if err := cache.SaveOrder(ctx, order); err != nil {
			logger.Error("Failed to cache order", zap.Error(err), zap.String("order_uid", order.OrderUID))
		}

		if err := db.SaveOrder(dbConn, order, logger); err != nil {
			logger.Error("Failed to save order to DB", zap.Error(err), zap.String("order_uid", order.OrderUID))
			continue
		}

		if err := reader.CommitMessages(ctx, msg); err != nil {
			logger.Error("Failed to commit message", zap.Error(err), zap.String("order_uid", order.OrderUID))
		}

		logger.Info("Order pocessed from Kafka", zap.String("order_uid", order.OrderUID))
	}
}
