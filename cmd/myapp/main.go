package main

import (
	"context"
	"l0/internal/api"
	"l0/internal/cache"
	"l0/internal/db"
	"l0/internal/kafka"
	"l0/internal/model"
	"time"

	"go.uber.org/zap"
)

func main() {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	sqldb, err := db.NewDB(logger)
	if err != nil {
		logger.Fatal("DB connection failed", zap.Error(err))
	}
	defer sqldb.Close()

	redisCache := cache.NewCache("l0-redis:6379", logger)

	ctx := context.Background()
	if err := redisCache.RestoreFromDB(ctx, sqldb); err != nil {
		logger.Error("Failed to restore cache from DB", zap.Error(err))
	}
	logger.Info("Cache restoration attempted")

	order := model.Order{
		OrderUID:    "test789",
		TrackNumber: "TRACK789",
		OrderEntry:  "WBIL",
		Delivery: model.Delivery{
			Name:   "Jane Doe",
			Phone:  "+987654321",
			Zip:    "654321",
			City:   "Berlin",
			Adress: "456 Street",
			Region: "West",
			Email:  "jane@example.com",
		},
		Payment: model.Payment{
			Transaction:  "trans123",
			Currency:     "USD",
			Provider:     "wbpay",
			Amount:       1000,
			PaymentDt:    1637907727,
			Bank:         "alpha",
			DeliveryCost: 1500,
			GoodsTotal:   900,
			CustomFee:    0,
		},
		Items: []model.Item{
			{
				ChrtID:      9934930,
				TrackNumber: "TRACK789",
				Price:       453,
				Rid:         "item123",
				Name:        "Mascaras",
				Sale:        30,
				Size:        "0",
				TotalPrice:  317,
				NmID:        2389212,
				Brand:       "Vivienne Sabo",
				Status:      202,
			},
		},
		Locale:            "en",
		InternalSignature: "",
		CustomerID:        "test",
		DeliveryService:   "meest",
		Shardkey:          "9",
		SmID:              99,
		DateCreated:       "2021-11-26T06:22:19Z",
		OofShard:          "1",
	}
	if err := db.SaveOrder(sqldb, order, logger); err != nil {
		logger.Error("Failed to save order", zap.Error(err))
		return
	}
	if err := redisCache.SaveOrder(context.Background(), order); err != nil {
		logger.Warn("Failed to cache test order, continuing without cache", zap.Error(err))
	}
	logger.Info("Test order saved")

	time.Sleep(30 * time.Second)

	go kafka.ConsumeOrders(ctx, "kafka:9092", "orders", "order-group", sqldb, redisCache, logger)

	apiServer := api.NewServer(redisCache, sqldb, logger)
	if err := apiServer.Start(":8080"); err != nil {
		logger.Fatal("Failed to start HTTTP server", zap.Error(err))
	}
}
