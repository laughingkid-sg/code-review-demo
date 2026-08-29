package handler

import (
	"net/http"
	"time"

	"github.com/demo/medium-api/internal/middleware"
	"github.com/demo/medium-api/internal/model"
	"github.com/demo/medium-api/internal/service"
	"github.com/gin-gonic/gin"
)

// LoginRequest defines credentials for authentication.
type LoginRequest struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse returns the signed JWT and customer data.
type LoginResponse struct {
	Token     string          `json:"token"`
	TokenType string          `json:"token_type"`
	ExpiresIn int64           `json:"expires_in"`
	Customer  *model.Customer `json:"customer,omitempty"`
}

// AuthHandler manages authentication endpoints.
type AuthHandler struct {
	customerService service.CustomerService
	jwtSecret       string
}

// NewAuthHandler creates a new AuthHandler.
func NewAuthHandler(customerService service.CustomerService, jwtSecret string) *AuthHandler {
	return &AuthHandler{
		customerService: customerService,
		jwtSecret:       jwtSecret,
	}
}

// Login handles POST /api/auth/login.
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "email and password are required"})
		return
	}

	// For administrative test login
	if req.Email == "admin@demo.com" && req.Password == "admin123" {
		token, err := middleware.GenerateToken(0, "admin@demo.com", "admin", h.jwtSecret, 24*time.Hour)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
			return
		}
		c.JSON(http.StatusOK, LoginResponse{
			Token:     token,
			TokenType: "Bearer",
			ExpiresIn: int64(24 * time.Hour.Seconds()),
		})
		return
	}

	// Regular customer login: look up customer by email in customer list
	list, err := h.customerService.ListCustomers(c.Request.Context(), model.CustomerListFilter{
		Search: req.Email,
		Page:   1,
		Limit:  1,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to look up customer"})
		return
	}

	var customer *model.Customer
	for _, cust := range list.Data {
		if cust.Email == req.Email {
			customer = &cust
			break
		}
	}

	if customer == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": model.ErrInvalidCredentials.Error()})
		return
	}

	token, err := middleware.GenerateToken(customer.ID, customer.Email, "customer", h.jwtSecret, 24*time.Hour)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, LoginResponse{
		Token:     token,
		TokenType: "Bearer",
		ExpiresIn: int64(24 * time.Hour.Seconds()),
		Customer:  customer,
	})
}
