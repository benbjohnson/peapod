package peapod

import (
	"context"
	"time"
)

// Playlist errors.
const (
	ErrPlaylistRequired = Error("playlist required")
	ErrPlaylistNotFound = Error("playlist not found")
)

// Playlist represents a time-ordered list of tracks.
type Playlist struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	Tracks []*Track `json:"tracks,omitempty"`
}

// PlaylistService represents a service for managing playlists.
type PlaylistService interface {
	FindPlaylistByID(ctx context.Context, id int) (*Playlist, error)
}
