package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/demo/medium-api/internal/model"
	"github.com/demo/medium-api/internal/service"
	"github.com/gin-gonic/gin"
)

// CustomerHandler provides HTTP handlers for managing customer resources.
type CustomerHandler struct {
	service service.CustomerService
}

// NewCustomerHandler creates a new CustomerHandler.
func NewCustomerHandler(service service.CustomerService) *CustomerHandler {
	return &CustomerHandler{service: service}
}

// Register handles POST /api/customers.
func (h *CustomerHandler) Register(c *gin.Context) {
	var req model.CreateCustomerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	customer, err := h.service.RegisterCustomer(c.Request.Context(), &req)
	if err != nil {
		if errors.Is(err, model.ErrDuplicateEmail) {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, customer)
}

// Get handles GET /api/customers/:id.
func (h *CustomerHandler) Get(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid customer id"})
		return
	}

	customer, err := h.service.GetCustomerByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, model.ErrCustomerNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch customer"})
		return
	}

	c.JSON(http.StatusOK, customer)
}

// List handles GET /api/customers.
func (h *CustomerHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.Query("page"))
	limit, _ := strconv.Atoi(c.Query("limit"))

	filter := model.CustomerListFilter{
		Search: c.Query("q"),
		Page:   page,
		Limit:  limit,
	}

	result, err := h.service.ListCustomers(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list customers"})
		return
	}

	c.JSON(http.StatusOK, result)
}

// Update handles PUT /api/customers/:id.
func (h *CustomerHandler) Update(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid customer id"})
		return
	}

	var req model.UpdateCustomerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	updated, err := h.service.UpdateCustomer(c.Request.Context(), id, &req)
	if err != nil {
		if errors.Is(err, model.ErrCustomerNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, updated)
}

// Delete handles DELETE /api/customers/:id.
func (h *CustomerHandler) Delete(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid customer id"})
		return
	}

	if err := h.service.DeleteCustomer(c.Request.Context(), id); err != nil {
		if errors.Is(err, model.ErrCustomerNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete customer"})
		return
	}

	c.Status(http.StatusNoContent)
}
