package db

import (
	"context"
	"fmt"
	"go-react-template/backend/pkg/sqlcqueries"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type DBService struct {
	Pool *pgxpool.Pool
	*sqlcqueries.Queries 
}

func NewDatabase(ctx context.Context) (*DBService, error) {
	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		return nil, fmt.Errorf("DATABASE_URL environment variable not set")
	}

	config, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database URL: %w", err)
	}

	config.MaxConns = int32(25)
	config.MinConns = int32(5)
	config.MaxConnLifetime = 5 * time.Minute
	config.MaxConnIdleTime = 1 * time.Minute
	config.HealthCheckPeriod = 1 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close() // Close the pool if ping fails
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	log.Println("Database connection pool established successfully using pgxpool.")

	queries := sqlcqueries.New(pool)

	return &DBService{
		Pool:    pool,
		Queries: queries,
	}, nil
}

func (s *DBService) Close() {
	if s.Pool != nil {
		log.Println("Closing database connection pool...")
		s.Pool.Close()
	}
}


