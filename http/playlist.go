package http

import (
	"encoding/xml"
	"fmt"
	"mime"
	"net/http"
	"net/url"
	"path"
	"sort"
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

	// Reverse track order.
	if playlist != nil {
		sort.Slice(playlist.Tracks, func(i, j int) bool { return i >= j })
	}

	// Encode response.
	switch {
	case strings.Contains(r.Header.Get("Accept"), "text/xml"):
		if err != nil {
			Error(w, r, err)
			return
		}

		// Determine logo image.
		imageURL := h.baseURL
		imageURL.Path = "/assets/logo-1024x1024.png"

		// Convert playlist to RSS feed.
		rss := playlistRSS{
			Channel: channelRSS{
				Title:       playlist.Name,
				Description: cdata{"Your personal podcast."},
				Summary:     cdata{"Your personal podcast."},
				Image:       imageRSS{Href: imageURL.String()},
				Items:       make([]itemRSS, len(playlist.Tracks)),
			},
		}
		if t := playlist.LastTrackUpdatedAt(); !t.IsZero() {
			rss.Channel.LastBuildDate = t.Format(time.RFC1123Z)
		}

		// Conver tracks to RSS.
		for i, track := range playlist.Tracks {
			enclosureURL := h.baseURL
			enclosureURL.Path = fmt.Sprintf("/files/%s", track.Filename)

			rss.Channel.Items[i] = itemRSS{
				Title:       track.Title,
				Description: cdata{track.Description},
				Summary:     cdata{track.Description},
				PubDate:     track.CreatedAt.Format(time.RFC1123Z),
				Duration:    formatDuration(track.Duration),
				Enclosure: enclosureRSS{
					URL:    enclosureURL.String(),
					Type:   mime.TypeByExtension(path.Ext(track.Filename)),
					Length: track.Size,
				},
			}
		}

		w.Header().Set("Content-Type", "text/xml")
		if err := xml.NewEncoder(w).EncodeElement(
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
			Error(w, r, err)
			return
		}

	default:
		Error(w, r, ErrNotAcceptable)
	}
}

// playlistRSS represents an RSS feed for a playlist.
type playlistRSS struct {
	Channel channelRSS `xml:"channel"`
}

type channelRSS struct {
	Title         string    `xml:"title"`
	Description   cdata     `xml:"description"`
	Summary       cdata     `xml:"itunes:summary"`
	Image         imageRSS  `xml:"itunes:image"`
	LastBuildDate string    `xml:"lastBuildDate"`
	Items         []itemRSS `xml:"item"`
}

type imageRSS struct {
	Href string `xml:"href,attr"`
}

type itemRSS struct {
	Title       string       `xml:"title"`
	Description cdata        `xml:"description"`
	Summary     cdata        `xml:"itunes:summary"`
	Link        string       `xml:"link"`
	PubDate     string       `xml:"pubDate"`
	Duration    string       `xml:"itunes:duration,omitempty"`
	Enclosure   enclosureRSS `xml:"enclosure"`
}

type enclosureRSS struct {
	URL    string `xml:"url,attr"`
	Type   string `xml:"type,attr"`
	Length int    `xml:"length,attr"`
}

type cdata struct {
	Value string `xml:",cdata"`
}

// formatDuration formats d in HH:MM:SS format.
func formatDuration(d time.Duration) string {
	if d == 0 {
		return ""
	}

	s := (d / time.Second) % 60
	m := (d / time.Minute) % 60
	h := d / time.Hour
	return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
}
