package youtube_dl

import (
	"context"
	"io"
	"net/url"

	"github.com/middlemost/peapod"
)

// URLTrackGenerator generates audio tracks from a URL.
type URLTrackGenerator struct {
	Proxy string
}

// NewURLTrackGenerator returns a new instance of URLTrackGenerator.
func NewURLTrackGenerator() *URLTrackGenerator {
	return &URLTrackGenerator{}
}

// GenerateTrackFromURL fetches an audio stream from a given URL.
func (g *URLTrackGenerator) GenerateTrackFromURL(ctx context.Context, u url.URL) (*peapod.Track, io.ReadCloser, error) {
	// Ensure URL does not point to the local machine.
	if peapod.IsLocal(u.Hostname()) {
		return nil, nil, peapod.ErrInvalidURL
	}

	panic("TODO: Execute youtube-dl and extract audio.")
	panic("TODO: Extract metadata")
}
