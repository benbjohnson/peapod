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
	Token     string    `json:"token"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	Tracks []*Track `json:"tracks,omitempty"`
}

// LastTrackUpdatedAt returns maximum track time.
func (p *Playlist) LastTrackUpdatedAt() time.Time {
	var max time.Time
	for _, track := range p.Tracks {
		if track.UpdatedAt.After(max) {
			max = track.UpdatedAt
		}
	}
	return max
}

// PlaylistService represents a service for managing playlists.
type PlaylistService interface {
	FindPlaylistByID(ctx context.Context, id int) (*Playlist, error)
	FindPlaylistByToken(ctx context.Context, token string) (*Playlist, error)
}
