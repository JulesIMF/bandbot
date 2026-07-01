package storage

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v4"
	"github.com/julesimf/bandbot/internal/model"
)

func (pg *Postgres) CreateSetlist(ctx context.Context, setlist *model.Setlist, songIDs []int) error {
	tx, err := pg.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	err = tx.QueryRow(ctx,
		`INSERT INTO setlists (chat_id, name) VALUES ($1, $2) RETURNING id, created_at`,
		setlist.ChatID, setlist.Name,
	).Scan(&setlist.ID, &setlist.CreatedAt)
	if err != nil {
		return err
	}

	for i, songID := range songIDs {
		_, err := tx.Exec(ctx,
			`INSERT INTO setlist_songs (setlist_id, song_id, position) VALUES ($1, $2, $3)`,
			setlist.ID, songID, i+1)
		if err != nil {
			return err
		}
	}

	_, err = tx.Exec(ctx,
		`UPDATE chats SET active_setlist_id = $1 WHERE id = $2`,
		setlist.ID, setlist.ChatID)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (pg *Postgres) GetSetlist(ctx context.Context, chatID int64, name string) (*model.Setlist, error) {
	sl := &model.Setlist{}
	err := pg.pool.QueryRow(ctx,
		`SELECT id, chat_id, name, created_at FROM setlists WHERE chat_id = $1 AND name = $2`,
		chatID, name,
	).Scan(&sl.ID, &sl.ChatID, &sl.Name, &sl.CreatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return pg.loadSetlistSongs(ctx, sl)
}

func (pg *Postgres) GetSetlistByNameInChats(ctx context.Context, chatIDs []int64, name string) ([]model.Setlist, error) {
	if len(chatIDs) == 0 {
		return nil, nil
	}
	rows, err := pg.pool.Query(ctx,
		`SELECT id, chat_id, name, created_at FROM setlists WHERE chat_id = ANY($1) AND name = $2`,
		chatIDs, name)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var setlists []model.Setlist
	for rows.Next() {
		var sl model.Setlist
		if err := rows.Scan(&sl.ID, &sl.ChatID, &sl.Name, &sl.CreatedAt); err != nil {
			return nil, err
		}
		loaded, err := pg.loadSetlistSongs(ctx, &sl)
		if err != nil {
			return nil, err
		}
		setlists = append(setlists, *loaded)
	}
	return setlists, nil
}

func (pg *Postgres) GetSetlistByID(ctx context.Context, id int) (*model.Setlist, error) {
	sl := &model.Setlist{}
	err := pg.pool.QueryRow(ctx,
		`SELECT id, chat_id, name, created_at FROM setlists WHERE id = $1`, id,
	).Scan(&sl.ID, &sl.ChatID, &sl.Name, &sl.CreatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return pg.loadSetlistSongs(ctx, sl)
}

func (pg *Postgres) GetActiveSetlist(ctx context.Context, chatID int64) (*model.Setlist, error) {
	var setlistID *int
	err := pg.pool.QueryRow(ctx,
		`SELECT active_setlist_id FROM chats WHERE id = $1`, chatID,
	).Scan(&setlistID)
	if err == pgx.ErrNoRows || setlistID == nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return pg.GetSetlistByID(ctx, *setlistID)
}

func (pg *Postgres) GetActiveSetlistsInChats(ctx context.Context, chatIDs []int64) ([]model.Setlist, error) {
	if len(chatIDs) == 0 {
		return nil, nil
	}
	rows, err := pg.pool.Query(ctx,
		`SELECT s.id, s.chat_id, s.name, s.created_at
		 FROM setlists s
		 JOIN chats c ON c.active_setlist_id = s.id
		 WHERE c.id = ANY($1)`, chatIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var setlists []model.Setlist
	for rows.Next() {
		var sl model.Setlist
		if err := rows.Scan(&sl.ID, &sl.ChatID, &sl.Name, &sl.CreatedAt); err != nil {
			return nil, err
		}
		loaded, err := pg.loadSetlistSongs(ctx, &sl)
		if err != nil {
			return nil, err
		}
		setlists = append(setlists, *loaded)
	}
	return setlists, nil
}

func (pg *Postgres) SetActiveSetlist(ctx context.Context, chatID int64, setlistID int) error {
	_, err := pg.pool.Exec(ctx,
		`UPDATE chats SET active_setlist_id = $1 WHERE id = $2`, setlistID, chatID)
	return err
}

func (pg *Postgres) UpdateSetlistName(ctx context.Context, id int, name string) error {
	ct, err := pg.pool.Exec(ctx,
		`UPDATE setlists SET name = $1 WHERE id = $2`, name, id)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("setlist %d not found", id)
	}
	return nil
}

func (pg *Postgres) UpdateSetlistSongs(ctx context.Context, setlistID int, songIDs []int) error {
	tx, err := pg.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `DELETE FROM setlist_songs WHERE setlist_id = $1`, setlistID)
	if err != nil {
		return err
	}

	for i, songID := range songIDs {
		_, err := tx.Exec(ctx,
			`INSERT INTO setlist_songs (setlist_id, song_id, position) VALUES ($1, $2, $3)`,
			setlistID, songID, i+1)
		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func (pg *Postgres) DeleteSetlist(ctx context.Context, id int) error {
	_, err := pg.pool.Exec(ctx, `DELETE FROM setlists WHERE id = $1`, id)
	return err
}

func (pg *Postgres) ListSetlists(ctx context.Context, chatID int64) ([]model.Setlist, error) {
	rows, err := pg.pool.Query(ctx,
		`SELECT id, chat_id, name, created_at FROM setlists WHERE chat_id = $1 ORDER BY name`,
		chatID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var setlists []model.Setlist
	for rows.Next() {
		var sl model.Setlist
		if err := rows.Scan(&sl.ID, &sl.ChatID, &sl.Name, &sl.CreatedAt); err != nil {
			return nil, err
		}
		setlists = append(setlists, sl)
	}
	return setlists, nil
}

func (pg *Postgres) ListSetlistsInChats(ctx context.Context, chatIDs []int64) ([]model.Setlist, error) {
	if len(chatIDs) == 0 {
		return nil, nil
	}
	rows, err := pg.pool.Query(ctx,
		`SELECT id, chat_id, name, created_at FROM setlists WHERE chat_id = ANY($1) ORDER BY name`,
		chatIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var setlists []model.Setlist
	for rows.Next() {
		var sl model.Setlist
		if err := rows.Scan(&sl.ID, &sl.ChatID, &sl.Name, &sl.CreatedAt); err != nil {
			return nil, err
		}
		setlists = append(setlists, sl)
	}
	return setlists, nil
}

func (pg *Postgres) loadSetlistSongs(ctx context.Context, sl *model.Setlist) (*model.Setlist, error) {
	rows, err := pg.pool.Query(ctx,
		`SELECT ss.setlist_id, ss.song_id, ss.position,
		        s.id, s.chat_id, s.name, s.tempo, s.key, s.responsible, s.last_accessed_at, s.created_at
		 FROM setlist_songs ss
		 JOIN songs s ON s.id = ss.song_id
		 WHERE ss.setlist_id = $1
		 ORDER BY ss.position`, sl.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var e model.SetlistEntry
		song := &model.Song{}
		if err := rows.Scan(
			&e.SetlistID, &e.SongID, &e.Position,
			&song.ID, &song.ChatID, &song.Name, &song.Tempo, &song.Key,
			&song.Responsible, &song.LastAccessedAt, &song.CreatedAt,
		); err != nil {
			return nil, err
		}
		e.Song = song
		sl.Songs = append(sl.Songs, e)
	}
	return sl, nil
}
