package youtube_dl

import (
	"context"
	"encoding/json"
	"io"
	"io/ioutil"
	"mime"
	"net/url"
	"os"
	"os/exec"
	"time"

	"github.com/middlemost/peapod"
)

// URLTrackGenerator generates audio tracks from a URL.
type URLTrackGenerator struct {
	Format string
	Proxy  string

	LogOutput io.Writer
}

// NewURLTrackGenerator returns a new instance of URLTrackGenerator.
func NewURLTrackGenerator() *URLTrackGenerator {
	return &URLTrackGenerator{Format: "m4a"}
}

// GenerateTrackFromURL fetches an audio stream from a given URL.
func (g *URLTrackGenerator) GenerateTrackFromURL(ctx context.Context, u url.URL) (*peapod.Track, io.ReadCloser, error) {
	// Ensure URL does not point to the local machine.
	if peapod.IsLocal(u.Hostname()) {
		return nil, nil, peapod.ErrInvalidURL
	}

	// Generate empty temporary file.
	f, err := ioutil.TempFile("", "peapod-youtube-dl-")
	if err != nil {
		return nil, nil, err
	} else if err := f.Close(); err != nil {
		return nil, nil, err
	}
	path := f.Name()

	// Build argument list.
	args := []string{"-v", "-f", g.Format, "-o", f.Name(), "--write-info-json"}
	if g.Proxy != "" {
		args = append(args, "--proxy", g.Proxy)
	}

	// Execute command.
	cmd := exec.Command("youtube-dl", args...)
	cmd.Stdout = g.LogOutput
	cmd.Stderr = g.LogOutput
	if err := cmd.Run(); err != nil {
		return nil, nil, err
	}

	// Read info file.
	var info infoFile
	if buf, err := ioutil.ReadFile(path + ".info.json"); err != nil {
		return nil, nil, err
	} else if err := json.Unmarshal(buf, &info); err != nil {
		return nil, nil, err
	}

	// Build track.
	track := &peapod.Track{
		Title:       info.FullTitle,
		Duration:    time.Duration(info.Duration) * time.Second,
		ContentType: mime.TypeByExtension("." + g.Format),
		Size:        info.Size,
	}

	// Open file handle to return for reading.
	file, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}

	return track, &oneTimeReader{File: file}, nil
}

// infoFile represents a partial structure of the youtube-dl info JSON file.
type infoFile struct {
	FullTitle string `json:"fulltitle"`
	Duration  int    `json:"duration"`
	Size      int    `json:"filesize"`
}

// oneTimeReader allows the reader to read once and then it deletes on close.
type oneTimeReader struct {
	*os.File
}

// Close closes the file handle and deletes the file.
func (r *oneTimeReader) Close() error {
	if err := r.File.Close(); err != nil {
		return err
	}
	return os.Remove(r.File.Name())
}
