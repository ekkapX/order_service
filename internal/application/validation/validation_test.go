package validation

import (
	"fmt"
	"testing"
	"time"

	"l0/internal/domain/model"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func validOrder(t *testing.T) model.Order {
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

func TestValidateOrder_Success(t *testing.T) {
	t.Parallel()

	v := NewValidator()
	order := validOrder(t)

	err := v.ValidateOrder(order)

	assert.NoError(t, err)
}

func TestValidateOrder_ValidationErrors(t *testing.T) {
	t.Parallel()

	v := NewValidator()

	tests := []struct {
		name          string
		mutateOrder   func(*model.Order)
		expectedField string
	}{
		{
			name: "missing_order_uid",
			mutateOrder: func(o *model.Order) {
				o.OrderUID = ""
			},
			expectedField: "OrderUID",
		},
		{
			name: "invalid_email",
			mutateOrder: func(o *model.Order) {
				o.Delivery.Email = "not-an-email"
			},
			expectedField: "Email",
		},
		{
			name: "invalid_phone",
			mutateOrder: func(o *model.Order) {
				o.Delivery.Phone = "123"
			},
			expectedField: "Phone",
		},
		{
			name: "empty_items",
			mutateOrder: func(o *model.Order) {
				o.Items = []model.Item{}
			},
			expectedField: "Items",
		},
		{
			name: "negative_amount",
			mutateOrder: func(o *model.Order) {
				o.Payment.Amount = -100
			},
			expectedField: "Amount",
		},
		{
			name: "invalid_date",
			mutateOrder: func(o *model.Order) {
				o.DateCreated = "not-a-date"
			},
			expectedField: "DateCreated",
		},
		{
			name: "zero_price",
			mutateOrder: func(o *model.Order) {
				o.Items[0].Price = 0
			},
			expectedField: "Price",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			order := validOrder(t)
			tt.mutateOrder(&order)

			err := v.ValidateOrder(order)

			require.Error(t, err)

			var validationErrs validator.ValidationErrors
			require.ErrorAs(t, err, &validationErrs)

			assert.Equal(t, tt.expectedField, validationErrs[0].Field())
		})
	}
}
