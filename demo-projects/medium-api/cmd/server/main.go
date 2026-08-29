package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/demo/medium-api/internal/cache"
	"github.com/demo/medium-api/internal/config"
	"github.com/demo/medium-api/internal/handler"
	"github.com/demo/medium-api/internal/middleware"
	"github.com/demo/medium-api/internal/repository"
	"github.com/demo/medium-api/internal/service"
	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

func main() {
	cfg := config.Load()

	// -- Database --
	db, err := sql.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := waitForDatabase(db, 30*time.Second); err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	log.Println("database connected")

	// -- Redis --
	redisCache, err := connectRedisWithRetry(cfg.RedisURL, cfg.RedisPassword, cfg.RedisDB, 30*time.Second)
	if err != nil {
		log.Fatalf("failed to connect to redis: %v", err)
	}
	defer redisCache.Close()
	log.Println("redis connected")

	// -- Repositories --
	customerRepo := repository.NewCustomerPGRepository(db)
	orderRepo := repository.NewOrderPGRepository(db)

	// -- Services --
	customerSvc := service.NewCustomerService(customerRepo, redisCache)
	orderSvc := service.NewOrderService(orderRepo, customerRepo, redisCache)

	// -- Handlers --
	healthH := handler.NewHealthHandler(db, redisCache)
	authH := handler.NewAuthHandler(customerSvc, cfg.JWTSecret)
	customerH := handler.NewCustomerHandler(customerSvc)
	orderH := handler.NewOrderHandler(orderSvc)

	// -- Router --
	if cfg.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.New()
	r.Use(gin.Recovery())

	rateLimiter := middleware.NewRateLimiter(cfg.RateLimitRPM)
	r.Use(middleware.LoggerMiddleware())
	r.Use(middleware.RateLimitMiddleware(rateLimiter))

	// Health
	r.GET("/api/health", healthH.Check)
	r.GET("/health", healthH.Check)

	// Auth
	r.POST("/api/auth/login", authH.Login)

	// Public registration
	r.POST("/api/customers", customerH.Register)

	// Protected routes
	protected := r.Group("/api")
	protected.Use(middleware.JWTAuthMiddleware(cfg.JWTSecret))
	{
		// Customers
		protected.GET("/customers", customerH.List)
		protected.GET("/customers/:id", customerH.Get)
		protected.PUT("/customers/:id", customerH.Update)
		protected.DELETE("/customers/:id", customerH.Delete)

		// Orders
		protected.POST("/orders", orderH.Create)
		protected.GET("/orders", orderH.List)
		protected.GET("/orders/:id", orderH.Get)
		protected.PATCH("/orders/:id/status", orderH.UpdateStatus)
		protected.DELETE("/orders/:id", orderH.Cancel)
	}

	// -- Server --
	addr := fmt.Sprintf(":%s", cfg.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Printf("server starting on %s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	<-done
	log.Println("shutting down server...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("server forced to shutdown: %v", err)
	}

	log.Println("server stopped gracefully")
}

func waitForDatabase(db *sql.DB, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	var lastErr error

	for {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		err := db.PingContext(ctx)
		cancel()
		if err == nil {
			return nil
		}

		lastErr = err
		if time.Now().After(deadline) {
			return lastErr
		}

		time.Sleep(2 * time.Second)
	}
}

func connectRedisWithRetry(addr, password string, db int, timeout time.Duration) (*cache.RedisCache, error) {
	deadline := time.Now().Add(timeout)
	var lastErr error

	for {
		redisCache, err := cache.NewRedisCache(addr, password, db)
		if err == nil {
			return redisCache, nil
		}

		lastErr = err
		if time.Now().After(deadline) {
			return nil, lastErr
		}

		time.Sleep(2 * time.Second)
	}
}
