package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/demo/simple-api/internal/handler"
	"github.com/demo/simple-api/internal/middleware"
	"github.com/demo/simple-api/internal/store"
)

type config struct {
	port   string
	dbPath string
	apiKey string
}

func loadConfig() config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "data/catalog.db"
	}

	apiKey := os.Getenv("API_KEY")
	if apiKey == "" {
		apiKey = "dev-api-key-12345"
	}

	return config{
		port:   port,
		dbPath: dbPath,
		apiKey: apiKey,
	}
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
	})
}

func main() {
	cfg := loadConfig()

	// Ensure database directory exists if not in-memory
	if cfg.dbPath != ":memory:" {
		dir := filepath.Dir(cfg.dbPath)
		if dir != "." && dir != "/" {
			if err := os.MkdirAll(dir, 0755); err != nil {
				log.Fatalf("failed to create database directory %s: %v", dir, err)
			}
		}
	}

	// Initialize SQLite store
	dbStore, err := store.NewSQLiteStore(cfg.dbPath)
	if err != nil {
		log.Fatalf("failed to initialize sqlite store: %v", err)
	}
	defer func() {
		if err := dbStore.Close(); err != nil {
			log.Printf("error closing database: %v", err)
		}
	}()

	// Initialize middlewares and handlers
	authMiddleware := middleware.NewAPIKeyMiddleware(cfg.apiKey)
	healthH := handler.NewHealthHandler(dbStore)
	productH := handler.NewProductHandler(dbStore)

	mux := http.NewServeMux()
	mux.Handle("GET /api/health", healthH)
	mux.HandleFunc("GET /api/products", productH.List)
	mux.HandleFunc("GET /api/products/{id}", productH.Get)
	mux.HandleFunc("POST /api/products", authMiddleware.RequireKeyFunc(productH.Create))
	mux.HandleFunc("PUT /api/products/{id}", authMiddleware.RequireKeyFunc(productH.Update))
	mux.HandleFunc("DELETE /api/products/{id}", authMiddleware.RequireKeyFunc(productH.Delete))

	server := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.port),
		Handler:      loggingMiddleware(mux),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Channel for graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		log.Printf("Simple API server listening on :%s (database: %s)", cfg.port, cfg.dbPath)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("listen error: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("Shutting down server gracefully...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("server forced to shutdown: %v", err)
	}

	log.Println("Server exited cleanly")
}
