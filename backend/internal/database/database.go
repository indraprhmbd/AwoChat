package database

import (
	"context"
	"fmt"
	"time"

	"github.com/indraprhmbd/AwoChat/backend/internal/config"
	"github.com/jackc/pgx/v5/pgxpool"
)

type DB struct {
	Pool *pgxpool.Pool
}

func New(cfg config.DatabaseConfig) (*DB, error) {
	connStr := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.DBName,
	)

	poolConfig, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database config: %w", err)
	}

	poolConfig.MaxConns = int32(cfg.MaxOpenConns)
	poolConfig.MinConns = int32(cfg.MaxIdleConns)
	poolConfig.MaxConnLifetime = cfg.ConnMaxLifetime

	pool, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Verify connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &DB{Pool: pool}, nil
}

func (db *DB) Close() {
	db.Pool.Close()
}

// StartSessionCleanup runs a background goroutine that periodically removes expired sessions
func StartSessionCleanup(ctx context.Context, db *DB, sessionExpiration time.Duration) {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			cleanupExpiredSessions(ctx, db, sessionExpiration)
		}
	}
}

func cleanupExpiredSessions(ctx context.Context, db *DB, sessionExpiration time.Duration) {
	expiredTime := time.Now().Add(-sessionExpiration)

	_, err := db.Pool.Exec(
		ctx,
		"DELETE FROM sessions WHERE expires_at < $1",
		expiredTime,
	)
	if err != nil {
		// Log error but don't crash - this is a cleanup task
		fmt.Printf("Failed to cleanup expired sessions: %v\n", err)
	}
}
