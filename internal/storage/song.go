package storage

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v4"
	"github.com/julesimf/bandbot/internal/model"
)

func (pg *Postgres) CreateSong(ctx context.Context, song *model.Song) error {
	return pg.pool.QueryRow(ctx,
		`INSERT INTO songs (chat_id, name, tempo, key, responsible)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, created_at, last_accessed_at`,
		song.ChatID, song.Name, song.Tempo, song.Key, song.Responsible,
	).Scan(&song.ID, &song.CreatedAt, &song.LastAccessedAt)
}

func (pg *Postgres) GetSong(ctx context.Context, chatID int64, name string) (*model.Song, error) {
	song := &model.Song{}
	err := pg.pool.QueryRow(ctx,
		`SELECT id, chat_id, name, tempo, key, responsible, last_accessed_at, created_at
		 FROM songs WHERE chat_id = $1 AND name = $2`,
		chatID, name,
	).Scan(&song.ID, &song.ChatID, &song.Name, &song.Tempo, &song.Key,
		&song.Responsible, &song.LastAccessedAt, &song.CreatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if err := pg.loadSongRelations(ctx, song); err != nil {
		return nil, err
	}
	return song, nil
}

func (pg *Postgres) GetSongByID(ctx context.Context, id int) (*model.Song, error) {
	song := &model.Song{}
	err := pg.pool.QueryRow(ctx,
		`SELECT id, chat_id, name, tempo, key, responsible, last_accessed_at, created_at
		 FROM songs WHERE id = $1`, id,
	).Scan(&song.ID, &song.ChatID, &song.Name, &song.Tempo, &song.Key,
		&song.Responsible, &song.LastAccessedAt, &song.CreatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if err := pg.loadSongRelations(ctx, song); err != nil {
		return nil, err
	}
	return song, nil
}

func (pg *Postgres) loadSongRelations(ctx context.Context, song *model.Song) error {
	// Notes
	rows, err := pg.pool.Query(ctx,
		`SELECT id, song_id, content, created_by, created_at
		 FROM song_notes WHERE song_id = $1 ORDER BY created_at`, song.ID)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var n model.SongNote
		if err := rows.Scan(&n.ID, &n.SongID, &n.Content, &n.CreatedBy, &n.CreatedAt); err != nil {
			return err
		}
		song.Notes = append(song.Notes, n)
	}

	// Pins
	rows, err = pg.pool.Query(ctx,
		`SELECT id, song_id, label, message_id, chat_id, pinned_by, created_at
		 FROM song_pins WHERE song_id = $1 ORDER BY created_at`, song.ID)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var p model.SongPin
		if err := rows.Scan(&p.ID, &p.SongID, &p.Label, &p.MessageID, &p.ChatID, &p.PinnedBy, &p.CreatedAt); err != nil {
			return err
		}
		song.Pins = append(song.Pins, p)
	}

	// Subscribers
	rows, err = pg.pool.Query(ctx,
		`SELECT song_id, user_id, username FROM song_subscribers WHERE song_id = $1`, song.ID)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var s model.SongSubscriber
		if err := rows.Scan(&s.SongID, &s.UserID, &s.Username); err != nil {
			return err
		}
		song.Subscribers = append(song.Subscribers, s)
	}

	return nil
}

func (pg *Postgres) UpdateSong(ctx context.Context, song *model.Song) error {
	ct, err := pg.pool.Exec(ctx,
		`UPDATE songs SET name = $1, tempo = $2, key = $3, responsible = $4, last_accessed_at = now()
		 WHERE id = $5`,
		song.Name, song.Tempo, song.Key, song.Responsible, song.ID)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("song %d not found", song.ID)
	}
	return nil
}

func (pg *Postgres) DeleteSong(ctx context.Context, id int) error {
	_, err := pg.pool.Exec(ctx, `DELETE FROM songs WHERE id = $1`, id)
	return err
}

func (pg *Postgres) TouchSong(ctx context.Context, id int) error {
	_, err := pg.pool.Exec(ctx,
		`UPDATE songs SET last_accessed_at = now() WHERE id = $1`, id)
	return err
}

func (pg *Postgres) ListSongs(ctx context.Context, chatID int64) ([]model.Song, error) {
	rows, err := pg.pool.Query(ctx,
		`SELECT id, chat_id, name, tempo, key, responsible, last_accessed_at, created_at
		 FROM songs WHERE chat_id = $1 ORDER BY name`, chatID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var songs []model.Song
	for rows.Next() {
		var s model.Song
		if err := rows.Scan(&s.ID, &s.ChatID, &s.Name, &s.Tempo, &s.Key,
			&s.Responsible, &s.LastAccessedAt, &s.CreatedAt); err != nil {
			return nil, err
		}
		songs = append(songs, s)
	}
	return songs, nil
}

func (pg *Postgres) ListSongNames(ctx context.Context, chatID int64) ([]string, error) {
	rows, err := pg.pool.Query(ctx,
		`SELECT name FROM songs WHERE chat_id = $1 ORDER BY name`, chatID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		names = append(names, name)
	}
	return names, nil
}

func (pg *Postgres) SearchSongs(ctx context.Context, chatID int64, query string, limit int) ([]model.Song, error) {
	var q string
	var args []interface{}
	if chatID != 0 {
		q = `SELECT id, chat_id, name, tempo, key, responsible, last_accessed_at, created_at
		     FROM songs WHERE chat_id = $1 AND lower(name) LIKE '%' || lower($2) || '%'
		     ORDER BY last_accessed_at DESC LIMIT $3`
		args = []interface{}{chatID, query, limit}
	} else {
		q = `SELECT id, chat_id, name, tempo, key, responsible, last_accessed_at, created_at
		     FROM songs WHERE lower(name) LIKE '%' || lower($1) || '%'
		     ORDER BY last_accessed_at DESC LIMIT $2`
		args = []interface{}{query, limit}
	}
	rows, err := pg.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var songs []model.Song
	for rows.Next() {
		var s model.Song
		if err := rows.Scan(&s.ID, &s.ChatID, &s.Name, &s.Tempo, &s.Key,
			&s.Responsible, &s.LastAccessedAt, &s.CreatedAt); err != nil {
			return nil, err
		}
		songs = append(songs, s)
	}
	return songs, nil
}

func (pg *Postgres) AddNote(ctx context.Context, note *model.SongNote) error {
	return pg.pool.QueryRow(ctx,
		`INSERT INTO song_notes (song_id, content, created_by)
		 VALUES ($1, $2, $3) RETURNING id, created_at`,
		note.SongID, note.Content, note.CreatedBy,
	).Scan(&note.ID, &note.CreatedAt)
}

func (pg *Postgres) DeleteNote(ctx context.Context, id int) error {
	_, err := pg.pool.Exec(ctx, `DELETE FROM song_notes WHERE id = $1`, id)
	return err
}

func (pg *Postgres) ClearNotes(ctx context.Context, songID int) error {
	_, err := pg.pool.Exec(ctx, `DELETE FROM song_notes WHERE song_id = $1`, songID)
	return err
}

func (pg *Postgres) AddPin(ctx context.Context, pin *model.SongPin) error {
	return pg.pool.QueryRow(ctx,
		`INSERT INTO song_pins (song_id, label, message_id, chat_id, pinned_by)
		 VALUES ($1, $2, $3, $4, $5) RETURNING id, created_at`,
		pin.SongID, pin.Label, pin.MessageID, pin.ChatID, pin.PinnedBy,
	).Scan(&pin.ID, &pin.CreatedAt)
}

func (pg *Postgres) DeletePin(ctx context.Context, id int) error {
	_, err := pg.pool.Exec(ctx, `DELETE FROM song_pins WHERE id = $1`, id)
	return err
}

func (pg *Postgres) ClearPins(ctx context.Context, songID int) error {
	_, err := pg.pool.Exec(ctx, `DELETE FROM song_pins WHERE song_id = $1`, songID)
	return err
}
