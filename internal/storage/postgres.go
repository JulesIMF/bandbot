package storage

import (
	"context"
	"embed"
	"fmt"

	"github.com/jackc/pgx/v4/pgxpool"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

type Postgres struct {
	pool *pgxpool.Pool
}

func NewPostgres(ctx context.Context, databaseURL string) (*Postgres, error) {
	pool, err := pgxpool.Connect(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("connect to postgres: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	pg := &Postgres{pool: pool}
	if err := pg.migrate(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("run migrations: %w", err)
	}

	return pg, nil
}

func (pg *Postgres) migrate(ctx context.Context) error {
	sql, err := migrationsFS.ReadFile("migrations/001_init.sql")
	if err != nil {
		return fmt.Errorf("read migration: %w", err)
	}
	_, err = pg.pool.Exec(ctx, string(sql))
	if err != nil {
		return fmt.Errorf("exec migration: %w", err)
	}
	return nil
}

func (pg *Postgres) Close() {
	pg.pool.Close()
}

func (pg *Postgres) EnsureChat(ctx context.Context, chatID int64) error {
	_, err := pg.pool.Exec(ctx,
		`INSERT INTO chats (id) VALUES ($1) ON CONFLICT DO NOTHING`, chatID)
	return err
}
