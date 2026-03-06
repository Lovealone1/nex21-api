package db

import (
	"database/sql"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/Lovealone1/nex21-api/internal/platform/config"
	"github.com/Lovealone1/nex21-api/internal/platform/observability"
)

// Database wraps the gorm.DB instance
type Database struct {
	*gorm.DB
}

// Connect initializes the GORM PostgreSQL connection
func Connect(cfg *config.Config) (*Database, error) {
	if cfg.DBUrl == "" {
		return nil, gorm.ErrInvalidDB
	}

	// We can customize the GORM logger here, but default is fine to start
	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	}

	db, err := gorm.Open(postgres.Open(cfg.DBUrl), gormConfig)
	if err != nil {
		observability.Log.Errorf("GORM Failed to connect to database: %v", err)
		return nil, err
	}

	observability.Log.Info("Connected to PostgreSQL via GORM successfully")

	return &Database{DB: db}, nil
}

// ConnectSimple opens a dedicated *sql.DB using pgx with QueryExecModeSimpleProtocol.
//
// Simple protocol sends every query as a plain-text message without server-side
// prepared statements. This prevents the "prepared statement already exists"
// (SQLSTATE 42P05) conflict that occurs when pgx's default statement cache
// is used across pooled connections.
//
// Use this connection ONLY for admin repository operations that bypass RLS and
// must not deal with prepared-statement collisions.
func ConnectSimple(dsn string) (*sql.DB, error) {
	connCfg, err := pgx.ParseConfig(dsn)
	if err != nil {
		return nil, err
	}

	// Simple protocol: queries are sent as text, no named prepared statements.
	connCfg.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol

	db := stdlib.OpenDB(*connCfg)
	return db, nil
}
