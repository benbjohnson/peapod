package bolt

import (
	"context"

	"github.com/middlemost/peapod"
)

// Ensure service implements interface.
var _ peapod.PlaylistService = &PlaylistService{}

// PlaylistService represents a service to manage playlists.
type PlaylistService struct {
	db *DB
}

// NewPlaylistService returns a new instance of PlaylistService.
func NewPlaylistService(db *DB) *PlaylistService {
	return &PlaylistService{db: db}
}

func (s *PlaylistService) FindPlaylistByID(ctx context.Context, id int) (*peapod.Playlist, error) {
	panic("TODO")
}

func playlistExists(ctx context.Context, tx *Tx, id int) bool { panic("TODO") }
