package peapod

import (
	"context"
	"io"
	"net/url"
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
	ID          int           `json:"id"`
	PlaylistID  int           `json:"playlist_id"`
	FileID      string        `json:"file_id"`
	Title       string        `json:"title"`
	Duration    time.Duration `json:"duration"`
	ContentType string        `json:"content_type"`
	Size        int           `json:"size"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`
}

// TrackService represents a service for managing audio tracks.
type TrackService interface {
	CreateTrack(ctx context.Context, track *Track) error
}

// URLTrackGenerator returns a track and file contents from a URL.
type URLTrackGenerator interface {
	GenerateTrackFromURL(ctx context.Context, url url.URL) (*Track, io.ReadCloser, error)
}
