package peapod

import (
	"context"
	"time"
)

// Track errors.
const (
	ErrTrackRequired         = Error("track required")
	ErrTrackNotFound         = Error("track not found")
	ErrTrackPlaylistRequired = Error("track playlist required")
	ErrTrackFileRequired     = Error("track file required")
)

// Track represents an audio track.
type Track struct {
	ID         int       `json:"id"`
	PlaylistID int       `json:"playlist_id"`
	FileID     string    `json:"file_id"`
	Title      string    `json:"title"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// TrackService represents a service for managing audio tracks.
type TrackService interface {
	CreateTrack(ctx context.Context, track *Track) error
}
