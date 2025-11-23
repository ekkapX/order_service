package main

import (
	"context"
	"fmt"
	"os"

	"l0/internal/api"
	"l0/internal/cache"
	"l0/internal/db"
	"l0/internal/kafka"

	"go.uber.org/zap"
)

func main() {
	logger, err := zap.NewProduction()
	if err != nil {
		panic(err)
	}

	defer func() {
		if err := logger.Sync(); err != nil {
			fmt.Fprintf(os.Stderr, "logger sync failed: %v\n", err)
		}
	}()

	sqldb, err := db.NewDB(logger)
	if err != nil {
		logger.Fatal("DB connection failed", zap.Error(err))
	}

	defer func() {
		if err := sqldb.Close(); err != nil {
			logger.Error("Failed to close DB connection", zap.Error(err))
		}
	}()

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
