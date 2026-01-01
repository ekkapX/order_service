package handlers

import (
	"errors"
	"l0/internal/applicaiton/usecases"
	"l0/internal/domain/model"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type OrderHandler struct {
	getOrderUC *usecases.GetOrderUseCase
	logger     *zap.Logger
}

func NewOrderHandler(getOrderUC *usecases.GetOrderUseCase, logger *zap.Logger) *OrderHandler {
	return &OrderHandler{getOrderUC: getOrderUC, logger: logger}
}

func (h *OrderHandler) GetByUID(c *gin.Context) {
	orderUID := c.Param("order_uid")
	if orderUID == "" {
		h.logger.Warn("Missing order_uid parameter")
		c.JSON(http.StatusBadRequest, gin.H{"error": "order_uid is required"})
		return
	}

	order, err := h.getOrderUC.Execute(c.Request.Context(), orderUID)
	if err != nil {
		if errors.Is(err, model.ErrOrderNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "order not found"})
			return
		}
		h.logger.Error("Failed to get order", zap.Error(err), zap.String("order_uid", orderUID))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get order"})
		return
	}
	c.JSON(http.StatusOK, order)
}
