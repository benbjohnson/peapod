package http

import (
	"encoding/xml"
	"os"
	"testing"
)

// Ensure playlist can encode to RSS.
func TestPlaylistRSS(t *testing.T) {
	rss := &playlistRSS{
		Title:         "TITLE",
		LastBuildDate: "LASTBUILDDATE",
		Items: []itemRSS{
			{
				Title:           "TITLE",
				Link:            "LINK",
				PubDate:         "PUBDATE",
				Duration:        "00:00:00",
				EnclosureURL:    "URL",
				EnclosureType:   "TYPE",
				EnclosureLength: 100,
			},
		},
	}

	if err := xml.NewEncoder(os.Stdout).Encode(rss); err != nil {
		t.Fatal(err)
	}
}
