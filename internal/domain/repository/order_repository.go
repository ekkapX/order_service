package repository

import (
	"context"

	"l0/internal/domain/model"
)

//go:generate mockgen -source=order_repository.go -destination=mocks/order_repository.go -package=mocks
type OrderRepository interface {
	Save(ctx context.Context, order *model.Order) error
	GetByUID(ctx context.Context, orderUID string) (*model.Order, error)
	GetAll(ctx context.Context) ([]*model.Order, error)
	Exists(ctx context.Context, orderUID string) (bool, error)
}

type OrderCache interface {
	Get(ctx context.Context, orderUID string) (*model.Order, error)
	Set(ctx context.Context, order *model.Order) error
	Delete(ctx context.Context, orderUID string) error
	Close() error
}

type OrderUseCaseProvider interface {
	Execute(ctx context.Context, orderUID string) (*model.Order, error)
}
