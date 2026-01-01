package cache

import (
	"context"
	"l0/internal/domain/model"
)

type OrderCache struct {
	cache *Cache
}

func NewOrderCache(cache *Cache) *OrderCache {
	return &OrderCache{cache: cache}
}

func (c *OrderCache) Get(ctx context.Context, orderUID string) (*model.Order, error) {
	return c.cache.GetOrder(ctx, orderUID)
}

func (c *OrderCache) Set(ctx context.Context, order *model.Order) error {
	return c.cache.SaveOrder(ctx, *order)
}

func (r *OrderCache) Delete(ctx context.Context, orderUID string) error {
	return r.cache.client.Del(ctx, orderUID).Err()
}

func (c *OrderCache) Close() error {
	return c.cache.Close()
}
