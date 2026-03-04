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
	httpSwagger "github.com/swaggo/http-swagger/v2"

	_ "github.com/Lovealone1/nex21-api/docs" // Must be imported for Swagger init
	"github.com/Lovealone1/nex21-api/internal/platform/config"
	observability "github.com/Lovealone1/nex21-api/internal/platform/logger"

	"github.com/Lovealone1/nex21-api/internal/modules/auth/application"
	"github.com/Lovealone1/nex21-api/internal/modules/auth/infra"
	authhttp "github.com/Lovealone1/nex21-api/internal/modules/auth/transport/http"
)

// @title NEX21 .
// @version 1.0
// @description Your entire business, in one platform
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
func main() {
	// Load config
	cfg := config.Load()

	// Init logger (true = development, false = production)
	observability.Init(true)
	log := observability.Log

	log.Info("Starting Nex21 API...")

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

	// Setup Modules
	supabaseClient := infra.NewSupabaseClient(cfg.Auth.SupabaseURL, cfg.Auth.SupabaseAnonKey)
	authService := application.NewAuthService(supabaseClient)
	authHandler := authhttp.NewAuthHandler(authService)

	// Swagger Docs
	r.Get("/api/docs/*", httpSwagger.Handler(
		httpSwagger.URL("/api/docs/doc.json"), // The url pointing to API definition
	))

	// Mount Routes
	r.Route("/auth", func(r chi.Router) {
		authHandler.RegisterRoutes(r)
	})

	// HTTP Server
	srv := &http.Server{
		Addr:         ":" + cfg.HTTP.Port,
		Handler:      r,
		ReadTimeout:  cfg.HTTP.ReadTimeout,
		WriteTimeout: cfg.HTTP.WriteTimeout,
		IdleTimeout:  cfg.HTTP.IdleTimeout,
	}

	// Start server in goroutine
	go func() {
		log.Infof("Server running on: http://localhost:%s/health", cfg.HTTP.Port)
		log.Infof("Swagger Docs available at: http://localhost:%s/api/docs/index.html", cfg.HTTP.Port)
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
