package http

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/middlemost/peapod"
	"github.com/pressly/chi"
)

const (
	ErrTwilioAccountMismatch = peapod.Error("twilio account mismatch")
	ErrInvalidSMSRequestBody = peapod.Error("invalid sms request body")
)

// twilioHandler represents an HTTP handler for Twilio webhooks.
type twilioHandler struct {
	router chi.Router

	// The server's base URL.
	baseURL url.URL

	// Account identifier. Used to verify incoming messages.
	accountSID string

	// Services
	jobService      peapod.JobService
	playlistService peapod.PlaylistService
	smsService      peapod.SMSService
	trackService    peapod.TrackService
	userService     peapod.UserService
}

// newTwilioHandler returns a new instance of Twilio handler.
func newTwilioHandler() *twilioHandler {
	h := &twilioHandler{router: chi.NewRouter()}
	h.router.Post("/voice", h.handlePostVoice)
	h.router.Post("/sms", h.handlePostSMS)
	return h
}

// ServeHTTP implements http.Handler.
func (h *twilioHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.router.ServeHTTP(w, r)
}

func (h *twilioHandler) handlePostVoice(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Context-Type", "text/plain")
	w.WriteHeader(http.StatusNotImplemented)
	w.Write([]byte(`Peapod does not support voice calls. Please text me instead.`))
}

func (h *twilioHandler) handlePostSMS(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Verify incoming message matches account.
	accountSID := r.PostFormValue("AccountSid")
	if accountSID != h.accountSID {
		Error(ctx, w, r, ErrTwilioAccountMismatch)
		return
	}

	// Read incoming parameters.
	from := r.PostFormValue("From")
	body := strings.TrimSpace(r.PostFormValue("Body"))

	// Parse message as URL & ensure it doesn't point locally.
	u, err := url.Parse(body)
	if err != nil {
		Error(ctx, w, r, ErrInvalidSMSRequestBody)
		return
	} else if peapod.IsLocal(u.Hostname()) {
		Error(ctx, w, r, peapod.ErrInvalidURL)
		return
	}

	// Lookup user by mobile number.
	user, err := h.userService.FindUserByMobileNumber(ctx, from)
	if err != nil {
		Error(ctx, w, r, err)
		return
	}

	// Create the user if they don't exist.
	var isNewUser bool
	if user == nil {
		isNewUser = true
		user = &peapod.User{MobileNumber: from}
		if err := h.userService.CreateUser(ctx, user); err != nil {
			Error(ctx, w, r, err)
			return
		}
	}

	// Update context.
	ctx = peapod.NewContext(r.Context(), user)

	// Fetch user playlists.
	playlists, err := h.playlistService.FindPlaylistsByUserID(ctx, user.ID)
	if err != nil {
		Error(ctx, w, r, err)
		return
	} else if len(playlists) == 0 {
		Error(ctx, w, r, peapod.ErrPlaylistNotFound)
		return
	}

	// TODO: Ask user which playlist if there are multiple. Currently only one can exist.
	playlist := playlists[0]

	// If the user is new then send them their playlist feed URL.
	if isNewUser {
		feedURL := h.baseURL
		feedURL.Path = fmt.Sprintf("/p/%s.rss", playlist.Token)

		sms := &peapod.SMS{
			To:   user.MobileNumber,
			Body: fmt.Sprintf("Welcome to Peapod! Your personal podcast feed is:\n\n%s", feedURL.String()),
		}
		if err := h.smsService.SendSMS(ctx, sms); err != nil {
			Error(ctx, w, r, err)
			return
		}
	}

	// Add URL to job processing queue.
	job := peapod.Job{
		OwnerID:    user.ID,
		Type:       peapod.JobTypeCreateTrackFromURL,
		PlaylistID: playlist.ID,
		URL:        u.String(),
	}
	if err := h.jobService.CreateJob(ctx, &job); err != nil {
		Error(ctx, w, r, err)
		return
	}

	// Reply to user that job is being processed.
	w.Header().Set("Context-Type", "text/plain")
	w.Write([]byte(`I'll get that processed and let you know when it's ready.`))
}
