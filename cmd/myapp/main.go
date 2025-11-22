package main

import (
	"context"

	"l0/internal/api"
	"l0/internal/cache"
	"l0/internal/db"
	"l0/internal/kafka"

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

	go kafka.ConsumeOrders(ctx, "kafka:9092", "orders", "order-group", sqldb, redisCache, logger)

	apiServer := api.NewServer(redisCache, sqldb, logger)
	if err := apiServer.Start(":8080"); err != nil {
		logger.Fatal("Failed to start HTTTP server", zap.Error(err))
	}
}
