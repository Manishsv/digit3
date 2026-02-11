package postgres

import (
	"fmt"

	"digit.org/workflow/config"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// NewDB creates a new GORM database connection.
func NewDB(cfg config.DBConfig) (*gorm.DB, error) {
	// Construct the connection string from the configuration.
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host,
		cfg.Port,
		cfg.User,
		cfg.Password,
		cfg.Name,
		cfg.SSLMode,
	)

	// Open a connection to the database using GORM.
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent), // Keep it quiet for production
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Get the underlying sql.DB to configure connection pool
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	// Configure connection pool settings
	sqlDB.SetMaxOpenConns(25)   // Maximum number of open connections
	sqlDB.SetMaxIdleConns(5)    // Maximum number of idle connections
	sqlDB.SetConnMaxLifetime(0) // No connection lifetime limit
	sqlDB.SetConnMaxIdleTime(0) // No idle time limit

	// Ping the database to verify the connection is alive.
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}
