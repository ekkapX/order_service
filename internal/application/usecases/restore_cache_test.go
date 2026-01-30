package usecases

import (
	"context"
	"errors"
	"testing"

	"l0/internal/domain/model"
	"l0/internal/domain/repository/mocks"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
)

func TestRestoreCacheUseCase_Success(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		setupMocks func(*mocks.MockOrderRepository, *mocks.MockOrderCache, context.Context)
	}{
		{
			name: "restores_all_orders",
			setupMocks: func(repo *mocks.MockOrderRepository, cache *mocks.MockOrderCache, ctx context.Context) {
				orders := []*model.Order{
					{OrderUID: "order-1", TrackNumber: "TRACK-1"},
					{OrderUID: "order-2", TrackNumber: "TRACK-2"},
					{OrderUID: "order-3", TrackNumber: "TRACK-3"},
				}
				repo.EXPECT().GetAll(ctx).Return(orders, nil)
				cache.EXPECT().Set(ctx, gomock.Any()).Return(nil).Times(3)
			},
		},
		{
			name: "empty_database",
			setupMocks: func(repo *mocks.MockOrderRepository, cache *mocks.MockOrderCache, ctx context.Context) {
				repo.EXPECT().GetAll(ctx).Return([]*model.Order{}, nil)
			},
		},
		{
			name: "partial_cache_failure",
			setupMocks: func(repo *mocks.MockOrderRepository, cache *mocks.MockOrderCache, ctx context.Context) {
				orders := []*model.Order{
					{OrderUID: "order-1"},
					{OrderUID: "order-2"},
					{OrderUID: "order-3"},
				}
				repo.EXPECT().GetAll(ctx).Return(orders, nil)
				cache.EXPECT().Set(ctx, gomock.Any()).Return(nil).Times(2)
				cache.EXPECT().Set(ctx, gomock.Any()).Return(errors.New("redis timeout")).Times(1)
			},
		},
		{
			name: "all_cache_writes_fail",
			setupMocks: func(repo *mocks.MockOrderRepository, cache *mocks.MockOrderCache, ctx context.Context) {
				orders := []*model.Order{
					{OrderUID: "order-1"},
					{OrderUID: "order-2"},
				}
				repo.EXPECT().GetAll(ctx).Return(orders, nil)
				cache.EXPECT().Set(ctx, gomock.Any()).Return(errors.New("redis down")).Times(2)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockRepo := mocks.NewMockOrderRepository(ctrl)
			mockCache := mocks.NewMockOrderCache(ctrl)
			logger := zap.NewNop()

			ctx := context.Background()
			tt.setupMocks(mockRepo, mockCache, ctx)

			uc := NewRestoreCacheUseCase(mockRepo, mockCache, logger)
			err := uc.Execute(ctx)

			assert.NoError(t, err)
		})
	}
}

func TestRestoreCacheUseCase_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		setupMocks func(*mocks.MockOrderRepository, *mocks.MockOrderCache, context.Context)
		ctx        context.Context
	}{
		{
			name: "database_unavailable",
			setupMocks: func(repo *mocks.MockOrderRepository, cache *mocks.MockOrderCache, ctx context.Context) {
				repo.EXPECT().GetAll(ctx).Return(nil, errors.New("database connection lost"))
			},
			ctx: context.Background(),
		},
		{
			name: "context_canceled",
			setupMocks: func(repo *mocks.MockOrderRepository, cache *mocks.MockOrderCache, ctx context.Context) {
				repo.EXPECT().GetAll(ctx).Return(nil, context.Canceled)
			},
			ctx: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx
			}(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockRepo := mocks.NewMockOrderRepository(ctrl)
			mockCache := mocks.NewMockOrderCache(ctrl)
			logger := zap.NewNop()

			tt.setupMocks(mockRepo, mockCache, tt.ctx)

			uc := NewRestoreCacheUseCase(mockRepo, mockCache, logger)
			err := uc.Execute(tt.ctx)

			assert.Error(t, err)
		})
	}
}
