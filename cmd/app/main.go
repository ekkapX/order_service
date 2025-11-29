package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

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

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	defer func() {
		if err := logger.Sync(); err != nil {
			fmt.Fprintf(os.Stderr, "logger sync failed: %v\n", err)
		}
	}()

	sqldb, err := db.NewDB(logger)
	if err != nil {
		logger.Fatal("DB connection failed", zap.Error(err))
	}

	redisCache := cache.NewCache("l0-redis:6379", logger)

	if err := redisCache.RestoreFromDB(ctx, sqldb); err != nil {
		logger.Error("Failed to restore cache from DB", zap.Error(err))
	}
	logger.Info("Cache restoration attempted")

	wg := &sync.WaitGroup{}
	wg.Add(1)
	go kafka.ConsumeOrders(ctx, wg, "kafka:9092", "orders", "order-group", sqldb, redisCache, logger)

	apiServer := api.NewServer(redisCache, sqldb, logger)

	go func() {
		if err := apiServer.Start(":8080"); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Failed to start HTTTP server", zap.Error(err))
		}
	}()

	logger.Info("Application started. Waiting for signals...")
	<-ctx.Done()

	logger.Info("Shutdown signal received, shutting down...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := apiServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("Failed to shutdown HTTP server", zap.Error(err))
	} else {
		logger.Info("HTTP server stopped gracefully")
	}

	logger.Info("Waiting for Kafka consumer to stop...")
	wg.Wait()
	logger.Info("Kafka consumer stopped")

	if err := sqldb.Close(); err != nil {
		logger.Error("Failed to close DB connection", zap.Error(err))
	}
	logger.Info("DB connection closed")

	if err := redisCache.Close(); err != nil {
		logger.Error("Failed to close Redis connection", zap.Error(err))
	}
	logger.Info("Redis connection closed")

	logger.Info("Application stopped successfully")
}
