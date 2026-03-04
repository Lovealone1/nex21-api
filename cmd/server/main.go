package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/Lovealone1/nex21-api/internal/platform/config"
	"github.com/Lovealone1/nex21-api/internal/platform/db"
	appMiddleware "github.com/Lovealone1/nex21-api/internal/platform/httpserver/middleware"
	"github.com/Lovealone1/nex21-api/internal/platform/observability"
)

func main() {
	// Load config
	cfg := config.Load()

	// Init logger (true = development, false = production)
	observability.Init(true)
	log := observability.Log

	log.Info("Starting Nex21 API...")

	// Initialize Database via GORM
	database, err := db.Connect(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	// Extract the underlying *sql.DB if we want to defer close, but GORM handles pooling.
	// sqlDB, _ := database.DB.DB()
	// defer sqlDB.Close()

	// Router
	r := chi.NewRouter()

	// Core middlewares
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	// Health check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Nex21 API running"))
	})

	// Private routes (Require Tenant and Auth)
	r.Route("/api/v1", func(r chi.Router) {
		// En el futuro aquí irá el AuthMiddleware
		// r.Use(appMiddleware.AuthMiddleware)

		// 1. Inyectamos el TenantMiddleware para validar membresía
		r.Use(appMiddleware.TenantMiddleware(database))

		r.Get("/test-tenant", func(w http.ResponseWriter, r *http.Request) {

			// Si llegó hasta aquí, el middleware ya validó que el User tiene acceso al Tenant
			// y adjuntó el TenantID seguro al Request Context.
			tenantID := db.ExtractTenant(r.Context())

			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"message": "Acceso concedido al tenant ` + tenantID + `"}`))
		})
	})

	// HTTP Server
	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		log.Infof("🚀 Server running on :%s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	// Graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	log.Info("Shutting down server...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Errorf("Shutdown error: %v", err)
	}

	log.Info("Server stopped gracefully")
}
