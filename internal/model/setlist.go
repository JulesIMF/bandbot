package model

import "time"

type Setlist struct {
	ID        int
	ChatID    int64
	Name      string
	CreatedAt time.Time

	Songs []SetlistEntry
}

type SetlistEntry struct {
	SetlistID int
	SongID    int
	Position  int
	Song      *Song
}
