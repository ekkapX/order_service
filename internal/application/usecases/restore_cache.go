package usecases

import (
	"context"
	"fmt"
	"l0/internal/domain/repository"

	"go.uber.org/zap"
)

type RestoreCacheUseCase struct {
	orderRepo  repository.OrderRepository
	orderCache repository.OrderCache
	logger     *zap.Logger
}

func NewRestoreCacheUseCase(orderRepo repository.OrderRepository, orderCache repository.OrderCache, logger *zap.Logger) *RestoreCacheUseCase {
	return &RestoreCacheUseCase{orderRepo: orderRepo, orderCache: orderCache, logger: logger}
}

func (uc *RestoreCacheUseCase) Execute(ctx context.Context) error {
	orders, err := uc.orderRepo.GetAll(ctx)
	if err != nil {
		uc.logger.Error("Failed to get orders from DB for cache restore", zap.Error(err))
		return fmt.Errorf("failed to get orders from DB: %w", err)
	}

	successCount := 0
	for _, order := range orders {
		if err := uc.orderCache.Set(ctx, order); err != nil {
			uc.logger.Error("Failed to save order to cache", zap.Error(err), zap.String("order_uid", order.OrderUID))
			continue
		}
		successCount++
	}

	uc.logger.Info("Cache restored", zap.Int("success_count", successCount), zap.Int("total_count", len(orders)))
	return nil
}
