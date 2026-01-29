package cache

import (
	"context"
	"testing"
	"time"

	"l0/internal/domain/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const testRedisAddr = "localhost:6380"

func setupTestCache(t *testing.T) (*Cache, func()) {
	t.Helper()

	logger := zap.NewNop()

	c := NewCache(testRedisAddr, logger)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := c.client.Ping(ctx).Err()
	require.NoError(t, err, "Redis is not reachable")

	cleanup := func() {
		_ = c.client.FlushDB(context.Background()).Err()
		_ = c.Close()
	}

	return c, cleanup
}

func createTestOrder(uid string) *model.Order {
	return &model.Order{
		OrderUID:    uid,
		TrackNumber: "TRACK-" + uid,
		Entry:       "WB",
		Delivery: model.Delivery{
			Name:  "Test User",
			Phone: "+79991234567",
		},
		Payment: model.Payment{
			Transaction: uid,
			Amount:      1000,
		},
		Items: []model.Item{
			{
				ChrtID: 123,
				Name:   "Item 1",
			},
		},
	}
}

func TestOrderCache_SetAndGet_Success(t *testing.T) {
	c, cleanup := setupTestCache(t)
	defer cleanup()

	ctx := context.Background()
	order := createTestOrder("cache-test-1")

	err := c.SaveOrder(ctx, *order)
	require.NoError(t, err)

	cachedOrder, err := c.GetOrder(ctx, order.OrderUID)

	require.NoError(t, err)
	assert.NotNil(t, cachedOrder)
	assert.Equal(t, order.OrderUID, cachedOrder.OrderUID)
	assert.Equal(t, order.TrackNumber, cachedOrder.TrackNumber)
	assert.Equal(t, order.Delivery.Phone, cachedOrder.Delivery.Phone)
	assert.Equal(t, order.Items[0].Name, cachedOrder.Items[0].Name)
}

func TestOrderCache_Get_NotFound(t *testing.T) {
	c, cleanup := setupTestCache(t)
	defer cleanup()

	ctx := context.Background()

	cachedOrder, err := c.GetOrder(ctx, "non-existent-key")

	require.NoError(t, err)
	assert.Nil(t, cachedOrder)
}

func TestOrderCache_Overwrite(t *testing.T) {
	c, cleanup := setupTestCache(t)
	defer cleanup()

	ctx := context.Background()
	order := createTestOrder("overwrite-test")

	order.TrackNumber = "V1"
	err := c.SaveOrder(ctx, *order)
	require.NoError(t, err)

	order.TrackNumber = "V2"
	err = c.SaveOrder(ctx, *order)
	require.NoError(t, err)

	cachedOrder, err := c.GetOrder(ctx, order.OrderUID)

	require.NoError(t, err)
	assert.Equal(t, "V2", cachedOrder.TrackNumber)
}
