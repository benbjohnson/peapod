package http

import (
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/middlemost/peapod"
	"github.com/pressly/chi"
)

const (
	ErrTTSTextRequired = peapod.Error("tts text required")
)

// trackHandler represents an HTTP handler for managing tracks.
type trackHandler struct {
	router chi.Router

	// Services
	jobService      peapod.JobService
	playlistService peapod.PlaylistService
	trackService    peapod.TrackService
	userService     peapod.UserService
}

// newTrackHandler returns a new instance of trackHandler.
func newTrackHandler() *trackHandler {
	h := &trackHandler{router: chi.NewRouter()}
	h.router.Post("/tts", h.handlePostTTS)
	return h
}

// ServeHTTP implements http.Handler.
func (h *trackHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.router.ServeHTTP(w, r)
}

func (h *trackHandler) handlePostTTS(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Read body text.
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		Error(w, r, err)
		return
	}
	text := strings.TrimSpace(string(body))
	if len(text) == 0 {
		Error(w, r, ErrTTSTextRequired)
		return
	}

	// Read title.
	title := r.URL.Query().Get("title")
	if title == "" {
		Error(w, r, peapod.ErrTrackTitleRequired)
		return
	}

	// Retrieve phone number from header.
	// TODO: Use token auth system instead.
	mobileNumber := r.Header.Get("X-MOBILE-NUMBER")

	// Lookup user.
	u, err := h.userService.FindUserByMobileNumber(ctx, mobileNumber)
	if err != nil {
		Error(w, r, err)
		return
	} else if u == nil {
		Error(w, r, peapod.ErrUserNotFound)
		return
	}

	// Lookup default playlist.
	playlists, err := h.playlistService.FindPlaylistsByUserID(ctx, u.ID)
	if err != nil {
		Error(w, r, err)
		return
	}

	// Add text to job processing queue.
	job := peapod.Job{
		OwnerID:    u.ID,
		Type:       peapod.JobTypeCreateTrackFromTTS,
		PlaylistID: playlists[0].ID,
		Title:      title,
		Text:       text,
	}
	if err := h.jobService.CreateJob(ctx, &job); err != nil {
		Error(w, r, err)
		return
	}

	w.WriteHeader(http.StatusOK)
}
