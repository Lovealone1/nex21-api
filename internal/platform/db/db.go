package db

import (
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
