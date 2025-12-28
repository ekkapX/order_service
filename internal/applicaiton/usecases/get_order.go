package usecases

import (
	"context"
	"errors"
	"fmt"
	"l0/internal/domain/model"
	"l0/internal/domain/repository"

	"go.uber.org/zap"
)

type GetOrderUseCase struct {
	orderRepo  repository.OrderRepository
	orderCache repository.OrderCache
	logger     *zap.Logger
}

func NewGetOrderUseCase(repo repository.OrderRepository, cache repository.OrderCache, logger *zap.Logger) *GetOrderUseCase {
	return &GetOrderUseCase{orderRepo: repo, orderCache: cache, logger: logger}
}

func (uc *GetOrderUseCase) Execute(ctx context.Context, orderUID string) (*model.Order, error) {
	order, err := uc.orderCache.Get(ctx, orderUID)
	if err == nil && order != nil {
		uc.logger.Debug("Order retrieved from cache", zap.String("order_uid", orderUID))
		return order, nil
	}

	order, err = uc.orderRepo.GetByUID(ctx, orderUID)
	if err != nil {
		if errors.Is(err, model.ErrOrderNotFound) {
			uc.logger.Info("Order not found", zap.String("order_uid", orderUID))
			return nil, model.ErrOrderNotFound
		}
		uc.logger.Error("Failed to get order from DB", zap.Error(err), zap.String("order_uid", orderUID))
		return nil, fmt.Errorf("failed to get order from DB: %w", err)
	}

	if err := uc.orderCache.Set(ctx, order); err != nil {
		uc.logger.Error("Failed to save order to cache", zap.Error(err), zap.String("order_uid", orderUID))
		return nil, fmt.Errorf("failed to save order to cache: %w", err)
	}

	uc.logger.Debug("Order retrieved from DB", zap.String("order_uid", orderUID))
	return order, nil
}
