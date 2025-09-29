package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib" // postgres driver
	_ "modernc.org/sqlite"             // sqlite driver (pure Go)
)

// Open opens a database connection with appropriate settings
func Open(ctx context.Context, driver, dsn string) (*sql.DB, error) {
	var db *sql.DB
	var err error

	switch driver {
	case "sqlite":
		db, err = openSQLite(dsn)
	case "postgres":
		db, err = openPostgres(dsn)
	default:
		return nil, fmt.Errorf("unsupported database driver: %s", driver)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Test the connection
	if err := Ping(ctx, db); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}

// openSQLite opens a SQLite database with optimized settings
func openSQLite(dsn string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}

	// Enable WAL mode and foreign keys
	_, err = db.Exec("PRAGMA journal_mode=WAL")
	if err != nil {
		return nil, fmt.Errorf("failed to enable WAL mode: %w", err)
	}

	_, err = db.Exec("PRAGMA busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("failed to set busy timeout: %w", err)
	}

	_, err = db.Exec("PRAGMA foreign_keys=ON")
	if err != nil {
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	return db, nil
}

// openPostgres opens a PostgreSQL database using pgx stdlib driver
func openPostgres(dsn string) (*sql.DB, error) {
	return sql.Open("pgx", dsn)
}

// Close closes the database connection
func Close(db *sql.DB) error {
	if db != nil {
		return db.Close()
	}
	return nil
}

// Ping tests the database connection
func Ping(ctx context.Context, db *sql.DB) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	return db.PingContext(ctx)
}
