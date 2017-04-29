package youtube_dl

import (
	"context"
	"net/url"
)

type TrackFileService struct{}

func (s *TrackFileService) TrackFileFromURL(ctx context.Context, u url.URL) (*Track, io.ReadCloser, error) {
	panic("TODO: Execute youtube-dl and extract audio.")
	panic("TODO: Extract metadata")
}
