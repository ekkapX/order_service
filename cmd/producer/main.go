package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"math/rand/v2"
	"os"
	"os/signal"
	"syscall"
	"time"

	"l0/internal/domain/model"
	"l0/internal/infrastructure/config"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

var (
	intervalFlag = flag.Duration("interval", 5*time.Second, "Time interval between messages")
	badDataRate  = flag.Float64("bad-rate", 0.2, "Rate of bad data messages")
)

func main() {
	flag.Parse()

	logger, err := zap.NewProduction()
	if err != nil {
		panic(err)
	}
	defer func() {
		if err := logger.Sync(); err != nil {
			fmt.Fprintf(os.Stderr, "logger sync failed: %v\n", err)
		}
	}()

	cfg, err := config.LoadProducerConfig()
	if err != nil {
		logger.Fatal("Failed to load config", zap.Error(err))
	}

	writer := &kafka.Writer{
		Addr:                   kafka.TCP(cfg.Kafka.Broker),
		Topic:                  cfg.Kafka.Topic,
		Balancer:               &kafka.Hash{},
		AllowAutoTopicCreation: true,
	}
	defer func() {
		if err := writer.Close(); err != nil {
			logger.Error("Failed to close Kafka writer", zap.Error(err))
		}
	}()

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()
	logger.Info("Producer started", zap.String("broker", cfg.Kafka.Broker), zap.Float64("bad_rate", *badDataRate), zap.Duration("interval", *intervalFlag))

	ticker := time.NewTicker(*intervalFlag)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.Info("Stopping producer...")
			return
		case <-ticker.C:
			if err := produce(ctx, writer, logger); err != nil {
				logger.Error("Failed to produce message", zap.Error(err))
			}
		}
	}
}

func produce(ctx context.Context, writer *kafka.Writer, logger *zap.Logger) error {
	if rand.Float64() < *badDataRate {
		return sendGarbage(ctx, writer)
	}
	return sendOrder(ctx, writer, logger)
}

func sendOrder(ctx context.Context, writer *kafka.Writer, logger *zap.Logger) error {
	order := generateRealOrder()

	payload, err := json.Marshal(order)
	if err != nil {
		return fmt.Errorf("failed to marshal order: %w", err)
	}

	if err := writer.WriteMessages(ctx, kafka.Message{Key: []byte(order.OrderUID), Value: payload}); err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}

	logger.Info("Order sent", zap.String("order_uid", order.OrderUID))
	return nil
}

func sendGarbage(ctx context.Context, writer *kafka.Writer) error {
	garbage := []byte(fmt.Sprintf(`{"uid: "%s", "broken": true,`, gofakeit.UUID()))

	return writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(gofakeit.UUID()),
		Value: garbage,
	})
}

func generateRealOrder() model.Order {
	uid := gofakeit.UUID()
	price := gofakeit.Price(100, 5000)

	return model.Order{
		OrderUID:          uid,
		TrackNumber:       gofakeit.UUID(),
		Entry:             gofakeit.RandomString([]string{"WBIL", "OZON", "SBER"}),
		Locale:            "ru",
		InternalSignature: "",
		CustomerID:        gofakeit.UUID(),
		DeliveryService:   gofakeit.RandomString([]string{"meest", "cdek", "boxberry", "dhl"}),
		Shardkey:          fmt.Sprintf("%d", gofakeit.Number(1, 10)),
		SmID:              gofakeit.Number(1, 99999999),
		DateCreated:       time.Now().Format(time.RFC3339),
		OofShard:          fmt.Sprintf("%d", gofakeit.Number(1, 10)),

		Delivery: model.Delivery{
			Name:    gofakeit.Name(),
			Phone:   fmt.Sprintf("+7%010d", gofakeit.Number(9000000000, 9999999999)),
			Zip:     gofakeit.Zip(),
			City:    gofakeit.City(),
			Address: gofakeit.Address().Address,
			Region:  gofakeit.State(),
			Email:   gofakeit.Email(),
		},

		Payment: model.Payment{
			Transaction:  uid,
			RequestID:    gofakeit.UUID(),
			Currency:     "RUB",
			Provider:     gofakeit.RandomString([]string{"alfabank", "sberbank", "tinkoff"}),
			Amount:       int(price * 100),
			PaymentDt:    time.Now().Unix(),
			Bank:         "alfabank",
			DeliveryCost: 1500,
			GoodsTotal:   int(price * 100),
			CustomFee:    0,
		},

		Items: []model.Item{
			{
				ChrtID:      gofakeit.Number(1000000, 9999999),
				TrackNumber: gofakeit.UUID(),
				Price:       int(price * 100),
				Rid:         gofakeit.UUID(),
				Name:        gofakeit.ProductName(),
				Sale:        gofakeit.Number(0, 90),
				Size:        gofakeit.RandomString([]string{"S", "M", "L", "XL", "XXL", "0"}),
				TotalPrice:  int(price * 100),
				NmID:        gofakeit.Number(100000, 99999999),
				Brand:       gofakeit.Company(),
				Status:      202,
			},
		},
	}
}
