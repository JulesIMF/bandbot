package storage

import (
	"context"

	"github.com/julesimf/bandbot/internal/model"
)

type Storage interface {
	EnsureChat(ctx context.Context, chatID int64, name string) error
	ListAllChatIDs(ctx context.Context) ([]int64, error)
	GetChatName(ctx context.Context, chatID int64) (*string, error)

	// Songs
	CreateSong(ctx context.Context, song *model.Song) error
	GetSong(ctx context.Context, chatID int64, name string) (*model.Song, error)
	GetSongByID(ctx context.Context, id int) (*model.Song, error)
	UpdateSong(ctx context.Context, song *model.Song) error
	DeleteSong(ctx context.Context, id int) error
	TouchSong(ctx context.Context, id int) error
	ListSongs(ctx context.Context, chatID int64) ([]model.Song, error)
	ListSongNames(ctx context.Context, chatID int64) ([]string, error)
	SearchSongs(ctx context.Context, chatID int64, query string, limit int) ([]model.Song, error)
	SearchSongsInChats(ctx context.Context, chatIDs []int64, query string, limit int) ([]model.Song, error)
	ListSongsInChats(ctx context.Context, chatIDs []int64) ([]model.Song, error)
	ListSetlistsInChats(ctx context.Context, chatIDs []int64) ([]model.Setlist, error)
	GetSongByNameInChats(ctx context.Context, chatIDs []int64, name string) ([]model.Song, error)

	// Notes
	AddNote(ctx context.Context, note *model.SongNote) error
	DeleteNote(ctx context.Context, id int) error
	ClearNotes(ctx context.Context, songID int) error

	// Pins
	AddPin(ctx context.Context, pin *model.SongPin) error
	DeletePin(ctx context.Context, id int) error
	ClearPins(ctx context.Context, songID int) error

	// Subscribers
	Subscribe(ctx context.Context, sub *model.SongSubscriber) error
	Unsubscribe(ctx context.Context, songID int, userID int64) error
	IsSubscribed(ctx context.Context, songID int, userID int64) (bool, error)
	GetNotifyList(ctx context.Context, song *model.Song) ([]string, error)
	GetSubscribeAllUsers(ctx context.Context, chatID int64) ([]string, error)

	// User settings
	GetUserChatSettings(ctx context.Context, userID, chatID int64) (*model.UserChatSettings, error)
	ToggleSubscribeAll(ctx context.Context, userID, chatID int64, username string) (bool, error)

	// History
	AddHistory(ctx context.Context, h *model.SongHistory) error
	GetHistory(ctx context.Context, songID int) ([]model.SongHistory, error)

	// Setlists
	CreateSetlist(ctx context.Context, setlist *model.Setlist, songIDs []int) error
	GetSetlist(ctx context.Context, chatID int64, name string) (*model.Setlist, error)
	GetSetlistByNameInChats(ctx context.Context, chatIDs []int64, name string) ([]model.Setlist, error)
	GetSetlistByID(ctx context.Context, id int) (*model.Setlist, error)
	GetActiveSetlist(ctx context.Context, chatID int64) (*model.Setlist, error)
	GetActiveSetlistsInChats(ctx context.Context, chatIDs []int64) ([]model.Setlist, error)
	SetActiveSetlist(ctx context.Context, chatID int64, setlistID int) error
	UpdateSetlistName(ctx context.Context, id int, name string) error
	UpdateSetlistSongs(ctx context.Context, setlistID int, songIDs []int) error
	DeleteSetlist(ctx context.Context, id int) error
	ListSetlists(ctx context.Context, chatID int64) ([]model.Setlist, error)
}
