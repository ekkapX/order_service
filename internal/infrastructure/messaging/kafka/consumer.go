package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"time"

	"l0/internal/application/usecases"
	"l0/internal/domain/model"

	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

func ConsumeOrders(ctx context.Context, wg *sync.WaitGroup, broker, topic, groupID string, saveOrderUC *usecases.SaveOrderUseCase, logger *zap.Logger) {
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

	for {
		msg, err := reader.FetchMessage(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				logger.Info("Kafka consumer context canceled, stopping...")
				return
			}
			logger.Error("Failed to fetch message from Kafka", zap.Error(err))
			continue
		}

		var order model.Order
		if err := json.Unmarshal(msg.Value, &order); err != nil {
			logger.Error("Failed to unmarshal order", zap.Error(err), zap.String("message", string(msg.Value)))
			if err := reader.CommitMessages(ctx, msg); err != nil {
				logger.Error("Failed to commit message", zap.Error(err), zap.String("order_uid", order.OrderUID))
			}
			continue
		}

		shouldCommit := true
		if err := saveOrderUC.Execute(ctx, &order); err != nil {
			switch {
			case errors.Is(err, model.ErrOrderAlreadyExists):
				logger.Info("Order already exists, skipping", zap.String("order_uid", order.OrderUID))
			case errors.Is(err, model.ErrInvalidOrderData):
				logger.Info("Invalid order data, skipping", zap.String("order_uid", order.OrderUID))
			default:
				logger.Error("Failed to save order", zap.Error(err), zap.String("order_uid", order.OrderUID))
				shouldCommit = false
			}
		}

		if shouldCommit {
			if err := reader.CommitMessages(ctx, msg); err != nil {
				logger.Error("Failed to commit message", zap.Error(err), zap.String("order_uid", order.OrderUID))
			} else {
				logger.Info("Order processed from Kafka", zap.String("order_uid", order.OrderUID))
			}
		}
	}
}
