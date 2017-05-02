package http

import (
	"encoding/xml"
	"os"
	"testing"
	"time"
)

// Ensure playlist can encode to RSS.
func TestPlaylistRSS(t *testing.T) {
	rss := &playlistRSS{
		Channel: channelRSS{
			Title:         "TITLE",
			Description:   cdata{"DESC"},
			LastBuildDate: "LASTBUILDDATE",
			Image:         imageRSS{Href: "IMAGE"},
			Items: []itemRSS{
				{
					Title:    "TITLE",
					Link:     "LINK",
					PubDate:  "PUBDATE",
					Duration: formatDuration(63742 * time.Second),
					Enclosure: enclosureRSS{
						URL:    "URL",
						Type:   "TYPE",
						Length: 100,
					},
				},
			},
		},
	}

	if err := xml.NewEncoder(os.Stdout).EncodeElement(
		rss,
		xml.StartElement{
			Name: xml.Name{Local: "rss"},
			Attr: []xml.Attr{
				{Name: xml.Name{Local: "xmlns:itunes"}, Value: "http://www.itunes.com/dtds/podcast-1.0.dtd"},
				{Name: xml.Name{Local: "xmlns:atom"}, Value: "http://www.w3.org/2005/Atom"},
				{Name: xml.Name{Local: "version"}, Value: "2.0"},
			},
		},
	); err != nil {
		t.Fatal(err)
	}
}
