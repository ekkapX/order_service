package usecases

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"l0/internal/application/validation"
	"l0/internal/domain/model"
	"l0/internal/domain/repository/mocks"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
)

func createValidOrder(t *testing.T) model.Order {
	t.Helper()
	uid := gofakeit.UUID()

	return model.Order{
		OrderUID:        uid,
		TrackNumber:     gofakeit.LetterN(10),
		Entry:           gofakeit.RandomString([]string{"WB", "OZON"}),
		Locale:          gofakeit.LanguageAbbreviation(),
		CustomerID:      gofakeit.UUID(),
		DeliveryService: "dostavka",
		Shardkey:        gofakeit.DigitN(1),
		SmID:            gofakeit.Number(1, 999),
		DateCreated:     time.Now().Format(time.RFC3339),
		OofShard:        "1",
		Delivery: model.Delivery{
			Name:    gofakeit.Name(),
			Phone:   fmt.Sprintf("+7%010d", gofakeit.Number(9000000000, 9999999999)),
			Zip:     gofakeit.Zip(),
			City:    gofakeit.City(),
			Address: gofakeit.Address().Address,
			Region:  gofakeit.State(),
			Email:   gofakeit.Email(),
		},
		Payment: model.Payment{
			Transaction:  uid,
			Currency:     "RUB",
			Provider:     gofakeit.CreditCardType(),
			Amount:       gofakeit.Number(100, 10000),
			PaymentDt:    time.Now().Unix(),
			Bank:         gofakeit.RandomString([]string{"sber", "vtb", "t-bank"}),
			DeliveryCost: gofakeit.Number(100, 1000),
			GoodsTotal:   gofakeit.Number(100, 10000),
			CustomFee:    0,
		},
		Items: []model.Item{
			{
				ChrtID:      gofakeit.Number(1000, 999999),
				TrackNumber: gofakeit.LetterN(8),
				Price:       gofakeit.Number(100, 5000),
				Rid:         gofakeit.UUID(),
				Name:        gofakeit.ProductName(),
				Sale:        gofakeit.Number(0, 50),
				Size:        gofakeit.RandomString([]string{"XS", "S", "M", "L", "XL"}),
				TotalPrice:  gofakeit.Number(100, 5000),
				NmID:        gofakeit.Number(10000, 999999),
				Brand:       gofakeit.Company(),
				Status:      202,
			},
		},
	}
}

func TestSaveOrderUseCase_Success(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := mocks.NewMockOrderRepository(ctrl)
	mockCache := mocks.NewMockOrderCache(ctrl)
	validator := validation.NewValidator()
	logger := zap.NewNop()

	uc := NewSaveOrderUseCase(mockRepo, mockCache, validator, logger)

	ctx := context.Background()
	order := createValidOrder(t)

	mockRepo.EXPECT().Exists(ctx, order.OrderUID).Return(false, nil)
	mockRepo.EXPECT().Save(ctx, &order).Return(nil)
	mockCache.EXPECT().Set(ctx, &order).Return(nil)

	err := uc.Execute(ctx, &order)

	assert.NoError(t, err)
}

func TestSaveOrderUseCase_BusinessErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		setupMocks  func(*mocks.MockOrderRepository, *mocks.MockOrderCache, context.Context, string)
		mutateOrder func(*model.Order)
		wantErr     error
	}{
		{
			name: "validation_failed",
			setupMocks: func(repo *mocks.MockOrderRepository, cache *mocks.MockOrderCache, ctx context.Context, uid string) {
				repo.EXPECT().Exists(gomock.Any(), gomock.Any()).Times(0)
				repo.EXPECT().Save(gomock.Any(), gomock.Any()).Times(0)
				cache.EXPECT().Set(gomock.Any(), gomock.Any()).Times(0)
			},
			mutateOrder: func(o *model.Order) {
				o.OrderUID = ""
			},
			wantErr: model.ErrInvalidOrderData,
		},
		{
			name: "order_already_exists",
			setupMocks: func(repo *mocks.MockOrderRepository, cache *mocks.MockOrderCache, ctx context.Context, uid string) {
				repo.EXPECT().Exists(ctx, uid).Return(true, nil)
				repo.EXPECT().Save(gomock.Any(), gomock.Any()).Times(0)
				cache.EXPECT().Set(gomock.Any(), gomock.Any()).Times(0)
			},
			mutateOrder: func(o *model.Order) {},
			wantErr:     model.ErrOrderAlreadyExists,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockRepo := mocks.NewMockOrderRepository(ctrl)
			mockCache := mocks.NewMockOrderCache(ctrl)
			validator := validation.NewValidator()
			logger := zap.NewNop()

			uc := NewSaveOrderUseCase(mockRepo, mockCache, validator, logger)

			ctx := context.Background()
			order := createValidOrder(t)
			tt.mutateOrder(&order)

			tt.setupMocks(mockRepo, mockCache, ctx, order.OrderUID)

			err := uc.Execute(ctx, &order)

			require.Error(t, err)
			assert.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestSaveOrderUseCase_InfrastructureErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		setupMocks func(*mocks.MockOrderRepository, *mocks.MockOrderCache, context.Context, string)
	}{
		{
			name: "exists_check_failed",
			setupMocks: func(repo *mocks.MockOrderRepository, cache *mocks.MockOrderCache, ctx context.Context, uid string) {
				repo.EXPECT().Exists(ctx, uid).Return(false, errors.New("db error"))
			},
		},
		{
			name: "db_save_failed",
			setupMocks: func(repo *mocks.MockOrderRepository, cache *mocks.MockOrderCache, ctx context.Context, uid string) {
				repo.EXPECT().Exists(ctx, uid).Return(false, nil)
				repo.EXPECT().Save(gomock.Any(), gomock.Any()).Return(errors.New("db connection lost"))
				cache.EXPECT().Set(gomock.Any(), gomock.Any()).Times(0)
			},
		},
		{
			name: "cache_set_failed",
			setupMocks: func(repo *mocks.MockOrderRepository, cache *mocks.MockOrderCache, ctx context.Context, uid string) {
				repo.EXPECT().Exists(ctx, uid).Return(false, nil)
				repo.EXPECT().Save(gomock.Any(), gomock.Any()).Return(nil)
				cache.EXPECT().Set(gomock.Any(), gomock.Any()).Return(errors.New("redis down"))
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
			validator := validation.NewValidator()
			logger := zap.NewNop()

			uc := NewSaveOrderUseCase(mockRepo, mockCache, validator, logger)

			ctx := context.Background()
			order := createValidOrder(t)

			tt.setupMocks(mockRepo, mockCache, ctx, order.OrderUID)

			err := uc.Execute(ctx, &order)

			assert.Error(t, err)
		})
	}
}
