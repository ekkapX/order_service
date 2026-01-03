package usecases

import (
	"context"
	"fmt"
	"l0/internal/applicaiton/validation"
	"l0/internal/domain/model"
	"l0/internal/domain/repository"

	"go.uber.org/zap"
)

type SaveOrderUseCase struct {
	orderRepo  repository.OrderRepository
	orderCache repository.OrderCache
	validator  *validation.Validator
	logger     *zap.Logger
}

func NewSaveOrderUseCase(orderRepo repository.OrderRepository, orderCache repository.OrderCache, validator *validation.Validator, logger *zap.Logger) *SaveOrderUseCase {
	return &SaveOrderUseCase{orderRepo: orderRepo, orderCache: orderCache, validator: validator, logger: logger}
}

func (uc *SaveOrderUseCase) Execute(ctx context.Context, order *model.Order) error {
	if err := uc.validator.ValidateOrder(*order); err != nil {
		uc.logger.Warn("Order validation failed", zap.String("order_uid", order.OrderUID), zap.Error(err))
		return fmt.Errorf("validation failed: %w", model.ErrInvalidOrderData)
	}

	exists, err := uc.orderRepo.Exists(ctx, order.OrderUID)
	if err != nil {
		return fmt.Errorf("failed to check order existence: %w", err)
	}
	if exists {
		uc.logger.Info("Order already exists, skipping", zap.String("order_uid", order.OrderUID))
		return model.ErrOrderAlreadyExists
	}

	if err := uc.orderRepo.Save(ctx, order); err != nil {
		uc.logger.Error("Failed to save order to DB", zap.Error(err), zap.String("order_uid", order.OrderUID))
		return fmt.Errorf("failed to save order to DB: %w", err)
	}

	if err := uc.orderCache.Set(ctx, order); err != nil {
		uc.logger.Error("Failed to save order to cache", zap.Error(err), zap.String("order_uid", order.OrderUID))
		return fmt.Errorf("failed to save order to cache: %w", err)
	}

	uc.logger.Info("Order saved", zap.String("order_uid", order.OrderUID))
	return nil
}
