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
	files := []string{"migrations/001_init.sql", "migrations/002_chat_name.sql"}
	for _, f := range files {
		sql, err := migrationsFS.ReadFile(f)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", f, err)
		}
		if _, err = pg.pool.Exec(ctx, string(sql)); err != nil {
			return fmt.Errorf("exec migration %s: %w", f, err)
		}
	}
	return nil
}

func (pg *Postgres) Close() {
	pg.pool.Close()
}

func (pg *Postgres) EnsureChat(ctx context.Context, chatID int64, name string) error {
	_, err := pg.pool.Exec(ctx,
		`INSERT INTO chats (id, name) VALUES ($1, $2)
		 ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name`,
		chatID, nilIfEmpty(name))
	return err
}

func nilIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func (pg *Postgres) ListAllChatIDs(ctx context.Context) ([]int64, error) {
	rows, err := pg.pool.Query(ctx, `SELECT id FROM chats`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func (pg *Postgres) GetChatName(ctx context.Context, chatID int64) (*string, error) {
	var name *string
	err := pg.pool.QueryRow(ctx, `SELECT name FROM chats WHERE id = $1`, chatID).Scan(&name)
	if err != nil {
		return nil, err
	}
	return name, nil
}
