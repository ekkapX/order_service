package postgres

import (
	"context"
	"database/sql"
	"strings"
	"testing"

	"l0/internal/domain/model"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	tcPostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
)

func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()
	ctx := context.Background()

	pgContainer, err := tcPostgres.Run(ctx,
		"postgres:18-alpine",
		tcPostgres.WithDatabase("test_db"),
		tcPostgres.WithUsername("user"),
		tcPostgres.WithPassword("password"),
		tcPostgres.BasicWaitStrategies(),
	)
	require.NoError(t, err, "failed to start postgres container")

	t.Cleanup(func() {
		if err := pgContainer.Terminate(ctx); err != nil {
			t.Logf("failed to terminate container: %v", err)
		}
	})

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err, "failed to get connection string")

	db, err := sql.Open("pgx", connStr)
	require.NoError(t, err, "failed to open db connection")

	err = db.Ping()
	require.NoError(t, err, "failed to ping db")

	err = goose.Up(db, "../../../../migrations")
	require.NoError(t, err, "failed to apply migrations")

	t.Cleanup(func() {
		db.Close()
	})

	return db
}

func createTestOrder(t *testing.T) model.Order {
	t.Helper()
	uid := gofakeit.UUID()

	return model.Order{
		OrderUID:        uid,
		TrackNumber:     gofakeit.LetterN(10),
		Entry:           "WBIL",
		Locale:          "en",
		CustomerID:      gofakeit.UUID(),
		DeliveryService: gofakeit.Company(),
		Shardkey:        "9",
		SmID:            gofakeit.Number(1, 999),
		DateCreated:     gofakeit.Date().Format("2006-01-02T15:04:05Z"),
		OofShard:        "1",
		Delivery: model.Delivery{
			Name:    gofakeit.Name(),
			Phone:   gofakeit.Phone(),
			Zip:     gofakeit.Zip(),
			City:    gofakeit.City(),
			Address: gofakeit.Street(),
			Region:  gofakeit.State(),
			Email:   gofakeit.Email(),
		},
		Payment: model.Payment{
			Transaction:  uid,
			Currency:     "USD",
			Provider:     "wbpay",
			Amount:       gofakeit.Number(100, 10000),
			PaymentDt:    gofakeit.Date().Unix(),
			Bank:         gofakeit.Company(),
			DeliveryCost: gofakeit.Number(100, 1000),
			GoodsTotal:   gofakeit.Number(100, 10000),
			CustomFee:    gofakeit.Number(0, 100),
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
				Status:      gofakeit.Number(200, 299),
			},
		},
	}
}

func TestOrderRepository_CRUD(t *testing.T) {
	db := setupTestDB(t)
	repo := NewOrderRepository(db)
	ctx := context.Background()

	order := createTestOrder(t)

	err := repo.Save(ctx, &order)
	require.NoError(t, err)

	exists, err := repo.Exists(ctx, order.OrderUID)
	require.NoError(t, err)
	assert.True(t, exists)

	retrieved, err := repo.GetByUID(ctx, order.OrderUID)
	require.NoError(t, err)
	assert.Equal(t, order.OrderUID, retrieved.OrderUID)
	assert.Equal(t, order.Delivery.Name, retrieved.Delivery.Name)
	require.NotEmpty(t, retrieved.Items)
}

func TestOrderRepository_GetByUID_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewOrderRepository(db)
	ctx := context.Background()

	_, err := repo.GetByUID(ctx, "non-existent-uid")
	assert.ErrorIs(t, err, model.ErrOrderNotFound)
}

func TestOrderRepository_Save_DuplicateKey(t *testing.T) {
	db := setupTestDB(t)

	repo := NewOrderRepository(db)
	ctx := context.Background()

	order := createTestOrder(t)
	err := repo.Save(ctx, &order)
	require.NoError(t, err)

	err = repo.Save(ctx, &order)
	assert.Error(t, err)
}

func TestOrderRepository_Save_MultipleItems(t *testing.T) {
	db := setupTestDB(t)

	repo := NewOrderRepository(db)
	ctx := context.Background()

	order := createTestOrder(t)
	extraItems := []model.Item{
		{
			ChrtID:      12345,
			TrackNumber: "TRACK-2",
			Price:       1000,
			Rid:         gofakeit.UUID(),
			Name:        "Item 2",
			Sale:        10,
			Size:        "M",
			TotalPrice:  900,
			NmID:        54321,
			Brand:       "Nike",
			Status:      202,
		},
		{
			ChrtID:      67890,
			TrackNumber: "TRACK-3",
			Price:       2000,
			Rid:         gofakeit.UUID(),
			Name:        "Item 3",
			Sale:        0,
			Size:        "L",
			TotalPrice:  2000,
			NmID:        98765,
			Brand:       "Adidas",
			Status:      200,
		},
	}
	order.Items = append(order.Items, extraItems...)

	err := repo.Save(ctx, &order)
	require.NoError(t, err)

	retrieved, err := repo.GetByUID(ctx, order.OrderUID)
	require.NoError(t, err)

	require.Len(t, retrieved.Items, 3)

	assert.ElementsMatch(t, order.Items, retrieved.Items)
}

func TestOrderRepository_Save_RollbackOnFailure(t *testing.T) {
	db := setupTestDB(t)

	repo := NewOrderRepository(db)
	ctx := context.Background()

	order := createTestOrder(t)

	order.Items[0].Name = strings.Repeat("x", 200)

	err := repo.Save(ctx, &order)

	require.Error(t, err)

	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM orders WHERE order_uid = $1", order.OrderUID).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count, "Orders table should be empty after rollback")

	err = db.QueryRow("SELECT COUNT(*) FROM payment WHERE order_uid = $1", order.OrderUID).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count, "Payment table should be empty after rollback")

	err = db.QueryRow("SELECT COUNT(*) FROM delivery WHERE order_uid = $1", order.OrderUID).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count, "Delivery table should be empty after rollback")
}

func TestOrderRepository_GetAll_Success(t *testing.T) {
	db := setupTestDB(t)

	repo := NewOrderRepository(db)
	ctx := context.Background()

	orders := []model.Order{
		createTestOrder(t),
		createTestOrder(t),
		createTestOrder(t),
	}

	for _, o := range orders {
		require.NoError(t, repo.Save(ctx, &o))
	}

	retrievedOrders, err := repo.GetAll(ctx)
	require.NoError(t, err)

	require.Len(t, retrievedOrders, 3)

	firstRetrieved := retrievedOrders[0]
	assert.NotEmpty(t, firstRetrieved.OrderUID)
	assert.NotEmpty(t, firstRetrieved.Delivery.Name)
	assert.NotEmpty(t, firstRetrieved.Payment.Transaction)
	assert.NotEmpty(t, firstRetrieved.Items)
}

func TestOrderRepository_GetAll_EmptyDatabase(t *testing.T) {
	db := setupTestDB(t)

	repo := NewOrderRepository(db)
	ctx := context.Background()

	retrievedOrders, err := repo.GetAll(ctx)
	require.NoError(t, err)
	assert.Empty(t, retrievedOrders)
}

func TestOrderRepository_Save_ContextCanceled(t *testing.T) {
	db := setupTestDB(t)

	repo := NewOrderRepository(db)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	order := createTestOrder(t)

	err := repo.Save(ctx, &order)

	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)

	exists, _ := repo.Exists(context.Background(), order.OrderUID)
	assert.False(t, exists)
}

func TestOrderRepository_Save_BoundaryValues(t *testing.T) {
	db := setupTestDB(t)

	repo := NewOrderRepository(db)
	ctx := context.Background()

	order := createTestOrder(t)

	order.OrderUID = strings.Repeat("a", 50)
	order.TrackNumber = strings.Repeat("b", 50)
	order.Payment.Transaction = strings.Repeat("c", 50)

	err := repo.Save(ctx, &order)
	require.NoError(t, err)

	retrieved, err := repo.GetByUID(ctx, order.OrderUID)
	require.NoError(t, err)

	assert.Equal(t, 50, len(retrieved.OrderUID))
	assert.Equal(t, 50, len(retrieved.TrackNumber))
}
