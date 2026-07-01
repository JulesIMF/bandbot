package storage

import (
	"context"

	"github.com/julesimf/bandbot/internal/model"
)

func (pg *Postgres) AddHistory(ctx context.Context, h *model.SongHistory) error {
	return pg.pool.QueryRow(ctx,
		`INSERT INTO song_history (song_id, field, old_value, new_value, changed_by)
		 VALUES ($1, $2, $3, $4, $5) RETURNING id, changed_at`,
		h.SongID, h.Field, h.OldValue, h.NewValue, h.ChangedBy,
	).Scan(&h.ID, &h.ChangedAt)
}

func (pg *Postgres) GetHistory(ctx context.Context, songID int) ([]model.SongHistory, error) {
	rows, err := pg.pool.Query(ctx,
		`SELECT id, song_id, field, old_value, new_value, changed_by, changed_at
		 FROM song_history WHERE song_id = $1 ORDER BY changed_at DESC LIMIT 50`, songID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var history []model.SongHistory
	for rows.Next() {
		var h model.SongHistory
		if err := rows.Scan(&h.ID, &h.SongID, &h.Field, &h.OldValue, &h.NewValue,
			&h.ChangedBy, &h.ChangedAt); err != nil {
			return nil, err
		}
		history = append(history, h)
	}
	return history, nil
}
