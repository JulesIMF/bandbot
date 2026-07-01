package auth

import (
	"context"

	"github.com/julesimf/bandbot/internal/model"
)

type Authorizer interface {
	CanEditSong(ctx context.Context, userID int64, song *model.Song) bool
	CanDeleteSong(ctx context.Context, userID int64, song *model.Song) bool
	CanEditSetlist(ctx context.Context, userID int64, setlist *model.Setlist) bool
	CanDeleteSetlist(ctx context.Context, userID int64, setlist *model.Setlist) bool
}

type AllowAll struct{}

func (AllowAll) CanEditSong(_ context.Context, _ int64, _ *model.Song) bool       { return true }
func (AllowAll) CanDeleteSong(_ context.Context, _ int64, _ *model.Song) bool      { return true }
func (AllowAll) CanEditSetlist(_ context.Context, _ int64, _ *model.Setlist) bool   { return true }
func (AllowAll) CanDeleteSetlist(_ context.Context, _ int64, _ *model.Setlist) bool { return true }
