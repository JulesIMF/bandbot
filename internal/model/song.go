package model

import "time"

type Song struct {
	ID             int
	ChatID         int64
	Name           string
	Tempo          *int
	Key            *string
	Responsible    string
	LastAccessedAt time.Time
	CreatedAt      time.Time

	Notes       []SongNote
	Pins        []SongPin
	Subscribers []SongSubscriber
}

type SongNote struct {
	ID        int
	SongID    int
	Content   string
	CreatedBy string
	CreatedAt time.Time
}

type SongPin struct {
	ID        int
	SongID    int
	Label     string
	MessageID int64
	ChatID    int64
	PinnedBy  string
	CreatedAt time.Time
}

type SongSubscriber struct {
	SongID   int
	UserID   int64
	Username string
}

type SongHistory struct {
	ID        int
	SongID    int
	Field     string
	OldValue  *string
	NewValue  string
	ChangedBy string
	ChangedAt time.Time
}
