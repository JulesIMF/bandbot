package storage

import (
	"context"

	"github.com/jackc/pgx/v4"
	"github.com/julesimf/bandbot/internal/model"
)

func (pg *Postgres) Subscribe(ctx context.Context, sub *model.SongSubscriber) error {
	_, err := pg.pool.Exec(ctx,
		`INSERT INTO song_subscribers (song_id, user_id, username)
		 VALUES ($1, $2, $3) ON CONFLICT DO NOTHING`,
		sub.SongID, sub.UserID, sub.Username)
	return err
}

func (pg *Postgres) Unsubscribe(ctx context.Context, songID int, userID int64) error {
	_, err := pg.pool.Exec(ctx,
		`DELETE FROM song_subscribers WHERE song_id = $1 AND user_id = $2`,
		songID, userID)
	return err
}

func (pg *Postgres) IsSubscribed(ctx context.Context, songID int, userID int64) (bool, error) {
	var exists bool
	err := pg.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM song_subscribers WHERE song_id = $1 AND user_id = $2)`,
		songID, userID).Scan(&exists)
	return exists, err
}

func (pg *Postgres) GetNotifyList(ctx context.Context, song *model.Song) ([]string, error) {
	seen := make(map[string]bool)
	var usernames []string

	if song.Responsible != "" {
		seen[song.Responsible] = true
		usernames = append(usernames, song.Responsible)
	}

	rows, err := pg.pool.Query(ctx,
		`SELECT username FROM song_subscribers WHERE song_id = $1`, song.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var u string
		if err := rows.Scan(&u); err != nil {
			return nil, err
		}
		if !seen[u] {
			seen[u] = true
			usernames = append(usernames, u)
		}
	}

	rows, err = pg.pool.Query(ctx,
		`SELECT username FROM user_chat_settings
		 WHERE chat_id = $1 AND subscribe_all = true AND username != ''`, song.ChatID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var u string
		if err := rows.Scan(&u); err != nil {
			return nil, err
		}
		if !seen[u] {
			seen[u] = true
			usernames = append(usernames, u)
		}
	}

	return usernames, nil
}

func (pg *Postgres) GetUserChatSettings(ctx context.Context, userID, chatID int64) (*model.UserChatSettings, error) {
	s := &model.UserChatSettings{UserID: userID, ChatID: chatID}
	err := pg.pool.QueryRow(ctx,
		`SELECT subscribe_all FROM user_chat_settings WHERE user_id = $1 AND chat_id = $2`,
		userID, chatID).Scan(&s.SubscribeAll)
	if err == pgx.ErrNoRows {
		return s, nil
	}
	return s, err
}

func (pg *Postgres) ToggleSubscribeAll(ctx context.Context, userID, chatID int64, username string) (bool, error) {
	var newVal bool
	err := pg.pool.QueryRow(ctx,
		`INSERT INTO user_chat_settings (user_id, chat_id, username, subscribe_all)
		 VALUES ($1, $2, $3, true)
		 ON CONFLICT (user_id, chat_id) DO UPDATE
		 SET subscribe_all = NOT user_chat_settings.subscribe_all,
		     username = EXCLUDED.username
		 RETURNING subscribe_all`,
		userID, chatID, username).Scan(&newVal)
	return newVal, err
}

func (pg *Postgres) GetSubscribeAllUsers(ctx context.Context, chatID int64) ([]string, error) {
	rows, err := pg.pool.Query(ctx,
		`SELECT username FROM user_chat_settings
		 WHERE chat_id = $1 AND subscribe_all = true AND username != ''`, chatID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var usernames []string
	for rows.Next() {
		var u string
		if err := rows.Scan(&u); err != nil {
			return nil, err
		}
		usernames = append(usernames, u)
	}
	return usernames, nil
}
