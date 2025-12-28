package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"l0/internal/api"
	"l0/internal/infrastructure/cache"
	"l0/internal/infrastructure/config"
	"l0/internal/infrastructure/db"
	"l0/internal/infrastructure/messaging/kafka"

	"go.uber.org/zap"
)

func main() {
	logger, err := zap.NewProduction()
	if err != nil {
		panic(err)
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		logger.Fatal("Failed to load config", zap.Error(err))
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	defer func() {
		if err := logger.Sync(); err != nil {
			fmt.Fprintf(os.Stderr, "logger sync failed: %v\n", err)
		}
	}()

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", cfg.Database.Host, cfg.Database.Port, cfg.Database.User, cfg.Database.Password, cfg.Database.Name)
	sqldb, err := db.NewDB(dsn, logger)
	if err != nil {
		logger.Fatal("DB connection failed", zap.Error(err))
	}

	db.RunMigrations(sqldb, logger)

	redisCache := cache.NewCache(cfg.Redis.Addr, logger)

	if err := redisCache.RestoreFromDB(ctx, sqldb); err != nil {
		logger.Error("Failed to restore cache from DB", zap.Error(err))
	}
	logger.Info("Cache restoration attempted")

	wg := &sync.WaitGroup{}
	wg.Add(1)
	go kafka.ConsumeOrders(ctx, wg, cfg.Kafka.Broker, cfg.Kafka.Topic, cfg.Kafka.GroupID, sqldb, redisCache, logger)

	apiServer := api.NewServer(redisCache, sqldb, logger)

	go func() {
		if err := apiServer.Start(cfg.HTTPServerPort); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Fatal("Failed to start HTTTP server", zap.Error(err))
		}
	}()

	logger.Info("Application started. Waiting for signals...")
	<-ctx.Done()

	logger.Info("Shutdown signal received, shutting down...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
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
