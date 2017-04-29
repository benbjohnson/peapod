package http

import (
	"encoding/xml"
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

	// Account identifier. Used to verify incoming messages.
	AccountSID string

	// Services
	JobService      peapod.JobService
	PlaylistService peapod.PlaylistService
	TrackService    peapod.TrackService
	UserService     peapod.UserService
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
	ctx := r.Context()
	w.Header().Set("Context-Type", "text/xml")
	w.WriteHeader(http.StatusNotImplemented)
	if err := xml.NewEncoder(w).Encode(&twilioResponse{Message: "Peapod does not support voice calls. Please text me instead."}); err != nil {
		Error(ctx, w, r, err)
		return
	}
}

func (h *twilioHandler) handlePostSMS(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Verify incoming message matches account.
	accountSID := r.PostFormValue("AccountSid")
	if accountSID != h.AccountSID {
		Error(ctx, w, r, ErrTwilioAccountMismatch)
		return
	}

	// Read incoming parameters.
	from := r.PostFormValue("From")
	body := strings.TrimSpace(r.PostFormValue("Body"))

	// Parse message as URL.
	u, err := url.Parse(body)
	if err != nil {
		Error(ctx, w, r, ErrInvalidSMSRequestBody)
		return
	}

	// Create or find user by mobile number.
	user, err := h.UserService.FindOrCreateUserByMobileNumber(ctx, from)
	if err != nil {
		Error(ctx, w, r, err)
		return
	}

	// Add URL to job processing queue.
	job := peapod.Job{
		UserID: user.ID,
		Type:   peapod.JobTypeCreateTrackFromURL,
		URL:    u.String(),
	}
	if err := h.JobService.CreateJob(ctx, &job); err != nil {
		Error(ctx, w, r, err)
		return
	}

	// Reply to user that job is being processed.
	w.Header().Set("Context-Type", "text/xml")
	w.WriteHeader(http.StatusNotImplemented)
	if err := xml.NewEncoder(w).Encode(&twilioResponse{Message: "I'll get that processed and let you know when it's ready."}); err != nil {
		Error(ctx, w, r, err)
		return
	}
}

type twilioResponse struct {
	Message string `xml:"Response>Message"`
}
