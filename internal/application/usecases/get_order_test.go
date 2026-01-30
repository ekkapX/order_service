package usecases

import (
	"context"
	"errors"
	"testing"

	"l0/internal/domain/model"
	"l0/internal/domain/repository/mocks"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
)

func TestGetOrderUseCase_Execute_FoundInCache(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCache := mocks.NewMockOrderCache(ctrl)
	mockRepo := mocks.NewMockOrderRepository(ctrl)
	logger := zap.NewNop()

	uc := NewGetOrderUseCase(mockRepo, mockCache, logger)

	ctx := context.Background()
	uid := "cached-order-123"
	expectedOrder := &model.Order{
		OrderUID:    uid,
		TrackNumber: "TRACK-001",
		CustomerID:  "cust-1",
	}

	mockCache.EXPECT().
		Get(ctx, uid).
		Return(expectedOrder, nil)

	mockRepo.EXPECT().GetByUID(gomock.Any(), gomock.Any()).Times(0)

	order, err := uc.Execute(ctx, uid)

	require.NoError(t, err)
	assert.NotNil(t, order)
	assert.Equal(t, uid, order.OrderUID)
	assert.Equal(t, "TRACK-001", order.TrackNumber)
}

func TestGetOrderUseCase_Execute_CacheMiss_FoundInDB(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCache := mocks.NewMockOrderCache(ctrl)
	mockRepo := mocks.NewMockOrderRepository(ctrl)
	logger := zap.NewNop()

	uc := NewGetOrderUseCase(mockRepo, mockCache, logger)

	ctx := context.Background()
	uid := "db-order-456"
	expectedOrder := &model.Order{
		OrderUID:    uid,
		TrackNumber: "TRACK-002",
	}

	mockCache.EXPECT().
		Get(ctx, uid).
		Return(nil, errors.New("cache miss"))

	mockRepo.EXPECT().
		GetByUID(ctx, uid).
		Return(expectedOrder, nil)

	mockCache.EXPECT().
		Set(ctx, expectedOrder).
		Return(nil)

	order, err := uc.Execute(ctx, uid)

	require.NoError(t, err)
	assert.NotNil(t, order)
	assert.Equal(t, uid, order.OrderUID)
}

func TestGetOrderUseCase_Execute_NotFound(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCache := mocks.NewMockOrderCache(ctrl)
	mockRepo := mocks.NewMockOrderRepository(ctrl)
	logger := zap.NewNop()

	uc := NewGetOrderUseCase(mockRepo, mockCache, logger)

	ctx := context.Background()
	uid := "non-existent-order"

	mockCache.EXPECT().
		Get(ctx, uid).
		Return(nil, errors.New("not in cache"))

	mockRepo.EXPECT().
		GetByUID(ctx, uid).
		Return(nil, model.ErrOrderNotFound)

	order, err := uc.Execute(ctx, uid)

	require.ErrorIs(t, err, model.ErrOrderNotFound)
	assert.Nil(t, order)
}

func TestGetOrderUseCase_Execute_CacheSetFailure_ReturnsData(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCache := mocks.NewMockOrderCache(ctrl)
	mockRepo := mocks.NewMockOrderRepository(ctrl)
	logger := zap.NewNop()

	uc := NewGetOrderUseCase(mockRepo, mockCache, logger)

	ctx := context.Background()
	uid := "test-order"
	expectedOrder := &model.Order{OrderUID: uid}

	mockCache.EXPECT().Get(ctx, uid).Return(nil, errors.New("miss"))
	mockRepo.EXPECT().GetByUID(ctx, uid).Return(expectedOrder, nil)
	mockCache.EXPECT().Set(ctx, expectedOrder).Return(errors.New("redis down"))

	order, err := uc.Execute(ctx, uid)

	require.NoError(t, err)
	assert.NotNil(t, order)
	assert.Equal(t, uid, order.OrderUID)
}

func TestGetOrderUseCase_Execute_DBError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCache := mocks.NewMockOrderCache(ctrl)
	mockRepo := mocks.NewMockOrderRepository(ctrl)
	logger := zap.NewNop()

	uc := NewGetOrderUseCase(mockRepo, mockCache, logger)

	ctx := context.Background()
	uid := "test-order"

	mockCache.EXPECT().Get(ctx, uid).Return(nil, errors.New("miss"))
	mockRepo.EXPECT().GetByUID(ctx, uid).Return(nil, errors.New("db connection lost"))

	order, err := uc.Execute(ctx, uid)

	require.Error(t, err)
	require.Nil(t, order)
}
