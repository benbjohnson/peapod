package youtube_dl

import (
	"context"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/url"
	"os"
	"os/exec"
	"time"

	"github.com/middlemost/peapod"
)

// URLTrackGenerator generates audio tracks from a URL.
type URLTrackGenerator struct {
	Proxy string

	LogOutput io.Writer
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

	// Generate empty temporary file.
	f, err := ioutil.TempFile("", "peapod-youtube-dl-")
	if err != nil {
		return nil, nil, err
	} else if err := f.Close(); err != nil {
		return nil, nil, err
	} else if err := os.Remove(f.Name()); err != nil {
		return nil, nil, err
	}
	path := f.Name()

	// Build argument list.
	args := []string{
		"-v",
		"-f", "worstaudio",
		"--no-playlist",
		"--extract-audio",
		"--audio-format", "mp3",
		"--audio-quality", "32K",
		"-o", path + ".%(ext)s",
		"--write-info-json",
	}
	if g.Proxy != "" {
		args = append(args, "--proxy", g.Proxy)
	}
	args = append(args, u.String())

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
		Title:       info.Title,
		Description: info.Description,
		Duration:    time.Duration(info.Duration) * time.Second,
		ContentType: "audio/mp3",
		Size:        info.Size,
	}

	// Open file handle to return for reading.
	file, err := os.Open(path + ".mp3")
	if err != nil {
		return nil, nil, err
	}

	return track, &oneTimeReader{File: file}, nil
}

// infoFile represents a partial structure of the youtube-dl info JSON file.
type infoFile struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Duration    int    `json:"duration"`
	Size        int    `json:"filesize"`
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
