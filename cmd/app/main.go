package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"l0/internal/application/usecases"
	"l0/internal/application/validation"
	"l0/internal/infrastructure/cache"
	"l0/internal/infrastructure/config"
	"l0/internal/infrastructure/db"
	"l0/internal/infrastructure/http/handlers"
	"l0/internal/infrastructure/http/server"
	"l0/internal/infrastructure/messaging/kafka"
	"l0/internal/infrastructure/persistence/postgres"

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

	cfg, err := config.LoadConsumerConfig()
	if err != nil {
		logger.Fatal("Failed to load config", zap.Error(err))
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", cfg.Database.Host, cfg.Database.Port, cfg.Database.User, cfg.Database.Password, cfg.Database.Name)
	sqldb, err := db.NewDB(ctx, dsn, logger)
	if err != nil {
		logger.Fatal("DB connection failed", zap.Error(err))
	}

	db.RunMigrations(sqldb, logger)

	redisCache := cache.NewCache(cfg.Redis.Addr, logger)
	orderCache := cache.NewOrderCache(redisCache)
	defer func() {
		if err := orderCache.Close(); err != nil {
			logger.Error("Failed to close order cache", zap.Error(err))
		}
	}()

	if err := redisCache.RestoreFromDB(ctx, sqldb); err != nil {
		logger.Error("Failed to restore cache from DB", zap.Error(err))
	}
	logger.Info("Cache restoration attempted")

	orderRepo := postgres.NewOrderRepository(sqldb, logger)

	validator := validation.NewValidator()

	getOrderUC := usecases.NewGetOrderUseCase(orderRepo, orderCache, logger)
	saveOrderUC := usecases.NewSaveOrderUseCase(orderRepo, orderCache, validator, logger)

	wg := &sync.WaitGroup{}
	wg.Add(1)
	go kafka.ConsumeOrders(ctx, wg, cfg.Kafka.Broker, cfg.Kafka.Topic, cfg.Kafka.GroupID, saveOrderUC, logger)

	orderHandler := handlers.NewOrderHandler(getOrderUC, logger)
	serverHTTP := server.NewServer(orderHandler, logger)

	go func() {
		if err := serverHTTP.Start(cfg.HTTP.Port); err != nil {
			logger.Fatal("Failed to start HTTP server", zap.Error(err))
		}
	}()

	logger.Info("Application started. Waiting for signals...")
	<-ctx.Done()

	logger.Info("Shutdown signal received, shutting down...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.HTTP.ShutdownTimeout)
	defer shutdownCancel()

	if err := serverHTTP.Shutdown(shutdownCtx); err != nil {
		logger.Error("Failed to shutdown HTTP server", zap.Error(err))
	} else {
		logger.Info("HTTP server stopped gracefully")
	}

	logger.Info("Waiting for Kafka consumer to stop...")
	wg.Wait()
	logger.Info("Kafka consumer stopped")

	logger.Info("Application stopped successfully")
}
