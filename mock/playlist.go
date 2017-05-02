package mock

import (
	"context"

	"github.com/middlemost/peapod"
)

var _ peapod.PlaylistService = &PlaylistService{}

type PlaylistService struct {
	FindPlaylistByIDFn      func(ctx context.Context, id int) (*peapod.Playlist, error)
	FindPlaylistByTokenFn   func(ctx context.Context, token string) (*peapod.Playlist, error)
	FindPlaylistsByUserIDFn func(ctx context.Context, id int) ([]*peapod.Playlist, error)
}

func (s *PlaylistService) FindPlaylistByID(ctx context.Context, id int) (*peapod.Playlist, error) {
	return s.FindPlaylistByIDFn(ctx, id)
}

func (s *PlaylistService) FindPlaylistByToken(ctx context.Context, token string) (*peapod.Playlist, error) {
	return s.FindPlaylistByTokenFn(ctx, token)
}

func (s *PlaylistService) FindPlaylistsByUserID(ctx context.Context, id int) ([]*peapod.Playlist, error) {
	return s.FindPlaylistsByUserIDFn(ctx, id)
}
