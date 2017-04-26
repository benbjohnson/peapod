package peapod

import (
	"context"
	"time"
)

// Playlist represents a time-ordered list of tracks.
type Playlist struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// PlaylistService represents a service for managing playlists.
type PlaylistService interface {
	FindPlaylistByID(ctx context.Context, id string) (*Playlist, error)
}
