package http

import (
	"encoding/xml"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/middlemost/peapod"
	"github.com/pressly/chi"
)

// playlistHandler represents an HTTP handler for playlists.
type playlistHandler struct {
	router chi.Router

	baseURL         url.URL
	playlistService peapod.PlaylistService
}

// newPlaylistHandler returns a new instance of playlistHandler.
func newPlaylistHandler() *playlistHandler {
	h := &playlistHandler{router: chi.NewRouter()}
	h.router.Get("/:token", h.handleGet)
	return h
}

// ServeHTTP implements http.Handler.
func (h *playlistHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.router.ServeHTTP(w, r)
}

func (h *playlistHandler) handleGet(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	token := strings.TrimSuffix(chi.URLParam(r, "token"), ".rss")

	// Fetch playlist by token.
	playlist, err := h.playlistService.FindPlaylistByToken(ctx, token)

	// Encode response.
	switch {
	case strings.Contains(r.Header.Get("Accept"), "text/xml"):
		if err != nil {
			Error(ctx, w, r, err)
			return
		}

		// Convert playlist to RSS feed.
		rss := playlistRSS{
			Title: playlist.Name,
			Items: make([]itemRSS, len(playlist.Tracks)),
		}
		if t := playlist.LastTrackUpdatedAt(); !t.IsZero() {
			rss.LastBuildDate = t.Format(time.RFC1123Z)
		}

		// Conver tracks to RSS.
		for i, track := range playlist.Tracks {
			enclosureURL := h.baseURL
			enclosureURL.Path = fmt.Sprintf("/files/%s", track.FileID)

			rss.Items[i] = itemRSS{
				Title:           track.Title,
				PubDate:         track.CreatedAt.Format(time.RFC1123Z),
				Duration:        formatDuration(track.Duration),
				EnclosureURL:    enclosureURL.String(),
				EnclosureType:   track.ContentType,
				EnclosureLength: track.Size,
			}
		}

		w.Header().Set("Context-Type", "text/xml")
		if err := xml.NewEncoder(w).EncodeElement(
			rss,
			xml.StartElement{
				Name: xml.Name{Local: "rss"},
				Attr: []xml.Attr{
					{Name: xml.Name{Local: "version"}, Value: "2.0"},
				},
			},
		); err != nil {
			Error(ctx, w, r, err)
			return
		}

	default:
		Error(ctx, w, r, ErrNotAcceptable)
	}
}

// playlistRSS represents an RSS feed for a playlist.
type playlistRSS struct {
	Title         string    `xml:"channel>title"`
	LastBuildDate string    `xml:"channel>lastBuildDate"`
	Items         []itemRSS `xml:"channel>item"`
}

type itemRSS struct {
	Title           string `xml:"title"`
	Link            string `xml:"link"`
	PubDate         string `xml:"pubDate"`
	Duration        string `xml:"duration"`
	EnclosureURL    string `xml:"enclosure>url"`
	EnclosureType   string `xml:"enclosure>type"`
	EnclosureLength int    `xml:"enclosure>length"`
}

// formatDuration formats d in HH:MM:SS format.
func formatDuration(d time.Duration) string {
	s := (d / time.Second) % 60
	m := (d / time.Minute) % 60
	h := d / time.Hour
	return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
}
