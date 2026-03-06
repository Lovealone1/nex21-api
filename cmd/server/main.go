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
	"github.com/Lovealone1/nex21-api/internal/infrastructure/postgres"
	authInfra "github.com/Lovealone1/nex21-api/internal/modules/auth/infra"
	profileRepo "github.com/Lovealone1/nex21-api/internal/modules/profiles/repo"
	profileService "github.com/Lovealone1/nex21-api/internal/modules/profiles/service"
	profileHttp "github.com/Lovealone1/nex21-api/internal/modules/profiles/transport/http"
	tenantRepo "github.com/Lovealone1/nex21-api/internal/modules/tenant/repo"
	tenantService "github.com/Lovealone1/nex21-api/internal/modules/tenant/service"
	tenantHttp "github.com/Lovealone1/nex21-api/internal/modules/tenant/transport/http"
	"github.com/Lovealone1/nex21-api/internal/platform/config"
	"github.com/Lovealone1/nex21-api/internal/platform/db"
	appMiddleware "github.com/Lovealone1/nex21-api/internal/platform/httpserver/middleware"
	"github.com/Lovealone1/nex21-api/internal/platform/observability"
)

// @title NEX21 API
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

	// Initialize Database via GORM
	database, err := db.Connect(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	// Extract the underlying *sql.DB to pass to our custom Tenant Infrastructure
	sqlDB, err := database.DB.DB()
	if err != nil {
		log.Fatalf("Failed to extract sql.DB from gorm: %v", err)
	}

	// Create the core TenantStore implementing the Repo Contract
	tenantStore := postgres.NewTenantStore(sqlDB)

	// Open a dedicated simple-protocol connection for admin repo operations.
	// pgx's default statement cache (QueryExecModeCacheStatement) causes
	// "prepared statement already exists" (42P05) when the same physical
	// connection is reused across queries. Simple protocol sends queries as
	// plain text — no server-side prepared statements, no collisions.
	adminDB, err := db.ConnectSimple(cfg.DBUrl)
	if err != nil {
		log.Fatalf("Failed to open admin (simple-protocol) DB connection: %v", err)
	}

	// Initialize Identity Provider (Supabase Auth)
	authProvider := authInfra.NewSupabaseClient(cfg.SupabaseURL, cfg.SupabaseAnonKey, cfg.SupabaseServiceKey)

	// Initialize Profiles Module (Domain Repo + Service + Handler)
	profRepo := profileRepo.NewProfileRepo(tenantStore, adminDB)
	profService := profileService.NewProfileService(profRepo, authProvider)
	profHandler := profileHttp.NewProfileHandler(profService)

	// Initialize Tenant Module (Repo + Service + Handler)
	tenRepo := tenantRepo.NewTenantRepo(database.DB)
	tenService := tenantService.NewTenantService(tenRepo)
	tenHandler := tenantHttp.NewTenantHandler(tenService)

	// Initialize Tenant Members Module
	memberRepo := tenantRepo.NewMemberRepo(database.DB)
	memberService := tenantService.NewMemberService(memberRepo)
	memberHandler := tenantHttp.NewMemberHandler(memberService)

	// Router
	r := chi.NewRouter()

	// Core middlewares
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	// Health check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Nex21 API running"))
	})

	// Swagger Docs
	r.Get("/api/docs/*", httpSwagger.Handler(
		httpSwagger.URL("/api/docs/doc.json"), // The url pointing to API definition
	))

	// Private routes (Require Tenant and Auth)
	r.Route("/api/v1", func(r chi.Router) {
		// En el futuro aquí irá el AuthMiddleware
		// r.Use(appMiddleware.AuthMiddleware)

		// 1. Inyectamos el TenantMiddleware para validar membresía
		r.Use(appMiddleware.TenantMiddleware(database))

		r.Get("/test-tenant", func(w http.ResponseWriter, r *http.Request) {
			// El middleware ya validó la membresía e inyectó un Actor. Si no, MustActor hara panic que será atrapado por Recoverer.
			actor := db.MustActor(r.Context())

			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"message": "Acceso concedido al tenant ` + actor.TenantID + ` para el usuario ` + actor.UserID + ` con rol ` + actor.Role + `"}`))
		})

	})

	// Admin isolated routes — no tenant middleware required.
	// The caller provides tenant_id in the JSON body, handled inside the service layer.
	r.Route("/api/admin/v1", func(r chi.Router) {
		r.Route("/profiles", profHandler.RegisterRoutes)
		r.Route("/tenants", func(r chi.Router) {
			// Mount base tenant CRUD
			tenHandler.RegisterRoutes(r)
			// Mount membership sub-routes
			r.Route("/{id}/members", memberHandler.RegisterRoutes)
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
		log.Infof("Server running on :%s", cfg.Port)
		log.Infof("Swagger Docs available at: http://localhost:%s/api/docs/index.html", cfg.Port)
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
