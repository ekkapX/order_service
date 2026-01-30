package handlers_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"l0/internal/domain/model"
	"l0/internal/domain/repository/mocks"
	"l0/internal/infrastructure/http/handlers"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
)

func setupTest(t *testing.T) (*gomock.Controller, *mocks.MockOrderUseCaseProvider, *gin.Engine, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)

	ctrl := gomock.NewController(t)
	mockUC := mocks.NewMockOrderUseCaseProvider(ctrl)
	logger := zap.NewNop()

	h := handlers.NewOrderHandler(mockUC, logger)

	r := gin.New()
	r.GET("/order/:order_uid", h.GetByUID)

	w := httptest.NewRecorder()

	return ctrl, mockUC, r, w
}

func TestOrderHandler_GetByUID_Success(t *testing.T) {
	ctrl, mockUC, router, w := setupTest(t)
	defer ctrl.Finish()

	uid := "test-order-1"
	expectedOrder := &model.Order{
		OrderUID:    uid,
		TrackNumber: "TRACK-SUCCESS",
	}

	mockUC.EXPECT().
		Execute(gomock.Any(), uid).
		Return(expectedOrder, nil).
		Times(1)

	ctx := context.Background()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "/order/"+uid, nil)
	require.NoError(t, err)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var result model.Order
	err = json.Unmarshal(w.Body.Bytes(), &result)
	require.NoError(t, err)
	assert.Equal(t, uid, result.OrderUID)
	assert.Equal(t, "TRACK-SUCCESS", result.TrackNumber)
}

func TestOrderHandler_GetByUID_NotFound(t *testing.T) {
	ctrl, mockUC, router, w := setupTest(t)
	defer ctrl.Finish()

	uid := "unknown-id"

	mockUC.EXPECT().
		Execute(gomock.Any(), uid).
		Return(nil, model.ErrOrderNotFound).
		Times(1)

	ctx := context.Background()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "/order/"+uid, nil)
	require.NoError(t, err)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.JSONEq(t, `{"error": "order not found"}`, w.Body.String())
}

func TestOrderHandler_GetByUID_InternalError(t *testing.T) {
	ctrl, mockUC, router, w := setupTest(t)
	defer ctrl.Finish()

	uid := "crash-id"

	mockUC.EXPECT().
		Execute(gomock.Any(), uid).
		Return(nil, errors.New("db connection lost")).
		Times(1)

	ctx := context.Background()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "/order/"+uid, nil)
	require.NoError(t, err)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.JSONEq(t, `{"error": "failed to get order"}`, w.Body.String())
}

func TestOrderHandler_GetByUID_EmptyParam(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockUC := mocks.NewMockOrderUseCaseProvider(ctrl)
	logger := zap.NewNop()
	h := handlers.NewOrderHandler(mockUC, logger)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	c.Params = []gin.Param{{Key: "order_uid", Value: ""}}

	h.GetByUID(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.JSONEq(t, `{"error": "order_uid is required"}`, w.Body.String())
}
