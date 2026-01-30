package cache

import (
	"context"
	"testing"

	"l0/internal/domain/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	tcRedis "github.com/testcontainers/testcontainers-go/modules/redis"
	"go.uber.org/zap"
)

func setupTestCache(t *testing.T) *Cache {
	t.Helper()
	ctx := context.Background()

	redisContainer, err := tcRedis.Run(ctx,
		"redis:7-alpine",
		tcRedis.WithSnapshotting(0, 0),
		tcRedis.WithLogLevel(tcRedis.LogLevelVerbose),
	)
	require.NoError(t, err, "failed to start redis container")

	t.Cleanup(func() {
		if err := redisContainer.Terminate(ctx); err != nil {
			t.Logf("failed to terminate redis container: %v", err)
		}
	})
	endpoint, err := redisContainer.Endpoint(ctx, "")
	require.NoError(t, err, "failed to get redis endpoint")

	logger := zap.NewNop()
	c := NewCache(endpoint, logger)

	t.Cleanup(func() {
		_ = c.Close()
	})

	return c
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
	c := setupTestCache(t)

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
	c := setupTestCache(t)

	ctx := context.Background()

	cachedOrder, err := c.GetOrder(ctx, "non-existent-key")

	require.NoError(t, err)
	assert.Nil(t, cachedOrder)
}

func TestOrderCache_Overwrite(t *testing.T) {
	c := setupTestCache(t)

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

func TestOrderCache_Get_CorruptedData(t *testing.T) {
	c := setupTestCache(t)
	ctx := context.Background()
	uid := "corrupted-uid"

	err := c.client.Set(ctx, uid, "{ invalid-json-data ...", 0).Err()
	require.NoError(t, err)

	order, err := c.GetOrder(ctx, uid)

	require.Error(t, err)
	assert.Nil(t, order)
	assert.Contains(t, err.Error(), "invalid character")
}

func TestOrderCache_Save_ContextCanceled(t *testing.T) {
	c := setupTestCache(t)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	order := createTestOrder("ctx-test")

	err := c.SaveOrder(ctx, *order)

	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestOrderCache_Serialization_EdgeCases(t *testing.T) {
	c := setupTestCache(t)
	ctx := context.Background()

	orderEmptyItems := createTestOrder("empty-items")
	orderEmptyItems.Items = []model.Item{}

	err := c.SaveOrder(ctx, *orderEmptyItems)
	require.NoError(t, err)

	retrieved, err := c.GetOrder(ctx, orderEmptyItems.OrderUID)
	require.NoError(t, err)

	assert.Empty(t, retrieved.Items)
}
