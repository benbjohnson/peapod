// +build integration

package youtube_dl_test

import (
	"bytes"
	"context"
	"flag"
	"io"
	"io/ioutil"
	"net/url"
	"testing"

	"github.com/middlemost/peapod/youtube_dl"
)

var (
	proxy    = flag.String("proxy", "", "Proxy")
	videoURL = flag.String("url", "", "Video URL")
)

// Ensure service can generate a track from a URL.
func TestURLTrackGenerator_GenerateTrackFromURL(t *testing.T) {
	if *videoURL == "" {
		t.Fatal("url required")
	}

	// Initialize service.
	var buf bytes.Buffer
	g := youtube_dl.NewURLTrackGenerator()
	g.Proxy = *proxy
	g.LogOutput = &buf

	// Parse URL.
	u, err := url.Parse(*videoURL)
	if err != nil {
		t.Fatal(err)
	}

	// Fetch URL.
	track, rc, err := g.GenerateTrackFromURL(context.Background(), *u)
	if err != nil {
		t.Log(buf.String())
		t.Fatal(err)
	}

	// Copy to a temporary file.
	f, err := ioutil.TempFile("", "peapod-test-")
	if err != nil {
		t.Fatal(err)
	} else if _, err := io.Copy(f, rc); err != nil {
		t.Fatal(err)
	} else if err := f.Close(); err != nil {
		t.Fatal(err)
	}

	// Print output.
	t.Logf("Title: %s", track.Title)
	t.Logf("Duration: %s", track.Duration)
	t.Logf("ContentType: %s", track.ContentType)
	t.Logf("Size: %d", track.Size)
	t.Logf("File: %s", f.Name())
	t.Log("===")

	// Show log.
	t.Log(buf.String())
}
