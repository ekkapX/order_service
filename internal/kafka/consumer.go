package kafka

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"l0/internal/cache"
	"l0/internal/db"
	"l0/internal/model"
	"time"

	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

func validateOrder(order model.Order) error {
	if order.OrderUID == "" {
		return fmt.Errorf("order_uid is empty")
	}
	if len(order.Items) == 0 {
		return fmt.Errorf("items list is empty")
	}
	if order.Payment.Amount <= 0 {
		return fmt.Errorf("payment amount must be postivie")
	}
	return nil
}

func ConsumeOrders(ctx context.Context, broker, topic, groupID string, dbConn *sql.DB, cache *cache.Cache, logger *zap.Logger) {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  []string{broker},
		Topic:    topic,
		GroupID:  groupID,
		MinBytes: 10e3,
		MaxBytes: 10e6,
		MaxWait:  1 * time.Second,
	})
	defer reader.Close()

	logger.Info("Starting Kafka consumer", zap.String("topic", topic), zap.String("groupID", groupID))

	for {
		msg, err := reader.ReadMessage(ctx)
		if err != nil {
			logger.Error("Failed to read message from Kafka", zap.Error(err))
			continue
		}

		var order model.Order
		if err := json.Unmarshal(msg.Value, &order); err != nil {
			logger.Error("Failed to unmarshal order", zap.Error(err), zap.String("message", string(msg.Value)))
			continue
		}

		if err := validateOrder(order); err != nil {
			logger.Warn("Invalid order data", zap.Error(err), zap.String("order_uid", order.OrderUID))
			reader.CommitMessages(ctx, msg)
			continue
		}

		if err := cache.SaveOrder(ctx, order); err != nil {
			logger.Error("Failed to cache order", zap.Error(err), zap.String("order_uid", order.OrderUID))
		}

		if err := db.SaveOrder(dbConn, order, logger); err != nil {
			logger.Error("Failed to save order to DB", zap.Error(err), zap.String("order_uid", order.OrderUID))
			continue
		}

		logger.Info("Order pocessed from Kafka", zap.String("order_uid", order.OrderUID))
	}
}
