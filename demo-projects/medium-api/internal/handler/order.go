package handler

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/demo/medium-api/internal/model"
	"github.com/demo/medium-api/internal/service"
	"github.com/gin-gonic/gin"
)

// OrderHandler provides HTTP endpoints for order operations.
type OrderHandler struct {
	service service.OrderService
}

// NewOrderHandler creates a new OrderHandler.
func NewOrderHandler(service service.OrderService) *OrderHandler {
	return &OrderHandler{service: service}
}

// Create handles POST /api/orders.
func (h *OrderHandler) Create(c *gin.Context) {
	var req model.CreateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	order, err := h.service.CreateOrder(c.Request.Context(), &req)
	if err != nil {
		if errors.Is(err, model.ErrCustomerNotFound) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "customer does not exist"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.Header("Location", fmt.Sprintf("/api/orders/%d", order.ID))
	c.JSON(http.StatusCreated, order)
}

// Get handles GET /api/orders/:id.
func (h *OrderHandler) Get(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid order id"})
		return
	}

	order, err := h.service.GetOrderByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, model.ErrOrderNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch order"})
		return
	}

	c.JSON(http.StatusOK, order)
}

// List handles GET /api/orders.
func (h *OrderHandler) List(c *gin.Context) {
	customerID, _ := strconv.ParseInt(c.Query("customer_id"), 10, 64)
	page, _ := strconv.Atoi(c.Query("page"))
	limit, _ := strconv.Atoi(c.Query("limit"))

	filter := model.OrderListFilter{
		CustomerID: customerID,
		Status:     c.Query("status"),
		Page:       page,
		Limit:      limit,
	}

	result, err := h.service.ListOrders(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list orders"})
		return
	}

	c.JSON(http.StatusOK, result)
}

// UpdateStatus handles PATCH /api/orders/:id/status.
func (h *OrderHandler) UpdateStatus(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid order id"})
		return
	}

	var req model.UpdateOrderStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	updated, err := h.service.UpdateOrderStatus(c.Request.Context(), id, &req)
	if err != nil {
		if errors.Is(err, model.ErrOrderNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		if errors.Is(err, model.ErrInvalidStatusTransition) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, updated)
}

// Cancel handles DELETE /api/orders/:id.
func (h *OrderHandler) Cancel(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid order id"})
		return
	}

	cancelled, err := h.service.CancelOrder(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, model.ErrOrderNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		if errors.Is(err, model.ErrOrderCannotBeCancelled) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to cancel order"})
		return
	}

	c.JSON(http.StatusOK, cancelled)
}
