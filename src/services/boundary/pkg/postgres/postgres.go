package postgres

import (
	"boundary/internal/config"
	"fmt"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// NewConnection creates a new PostgreSQL database connection using GORM
func NewConnection(cfg *config.DatabaseConfig) (*gorm.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName, cfg.SSLMode,
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("error opening database connection: %v", err)
	}

	// Get underlying sql.DB to test connection
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("error getting underlying database: %v", err)
	}

	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("error connecting to database: %v", err)
	}

	return db, nil
}
