package http

import (
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"path"

	"github.com/middlemost/peapod"
	"github.com/pressly/chi"
	"github.com/pressly/chi/middleware"
	"golang.org/x/crypto/acme/autocert"
)

// Server represents an HTTP server.
type Server struct {
	ln net.Listener

	// Services
	FileService     peapod.FileService
	JobService      peapod.JobService
	PlaylistService peapod.PlaylistService
	SMSService      peapod.SMSService
	TrackService    peapod.TrackService
	UserService     peapod.UserService

	// Server options.
	Addr        string // bind address
	Host        string // external hostname
	Autocert    bool   // ACME autocert
	Recoverable bool   // panic recovery

	// Twilio specific options.
	Twilio struct {
		AccountSID string // twilio account number
	}

	LogOutput io.Writer
}

// NewServer returns a new instance of Server.
func NewServer() *Server {
	return &Server{
		Recoverable: true,
		LogOutput:   ioutil.Discard,
	}
}

// Open opens the server.
func (s *Server) Open() error {
	// Open listener on specified bind address.
	// Use HTTPS port if autocert is enabled.
	if s.Autocert {
		s.ln = autocert.NewListener(s.Host)
	} else {
		ln, err := net.Listen("tcp", s.Addr)
		if err != nil {
			return err
		}
		s.ln = ln
	}

	// Start HTTP server.
	go http.Serve(s.ln, s.router())

	return nil
}

// Close closes the socket.
func (s *Server) Close() error {
	if s.ln != nil {
		s.ln.Close()
	}
	return nil
}

// URL returns a base URL string with the scheme and host.
// This is available after the server has been opened.
func (s *Server) URL() url.URL {
	if s.ln == nil {
		return url.URL{}
	}

	if s.Autocert {
		return url.URL{Scheme: "https", Host: s.Host}
	}
	return url.URL{Scheme: "http", Host: s.ln.Addr().String()}
}

func (s *Server) router() http.Handler {
	r := chi.NewRouter()

	// Attach router middleware.
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	if s.Recoverable {
		r.Use(middleware.Recoverer)
	}
	// r.Mount("/debug", middleware.Profiler())
	r.Use(s.attachLogOutputToContext)
	r.Use(s.detectAccept)

	// Create API routes.
	r.Route("/", func(r chi.Router) {
		r.Use(middleware.DefaultCompress)
		r.Get("/ping", s.handlePing)
		r.Mount("/assets", newAssetHandler())
		r.Mount("/p", s.playlistHandler()) // alias
		r.Mount("/playlists", s.playlistHandler())
		r.Mount("/files", s.fileHandler())
		r.Mount("/tracks", s.trackHandler())
		r.Mount("/twilio", s.twilioHandler())
	})

	return r
}

// handlePing verifies the database connection and returns a success.
func (s *Server) handlePing(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte(`{"status:":"ok"}` + "\n"))
}

func (s *Server) playlistHandler() *playlistHandler {
	h := newPlaylistHandler()
	h.baseURL = s.URL()
	h.playlistService = s.PlaylistService
	return h
}

func (s *Server) fileHandler() *fileHandler {
	h := newFileHandler()
	h.fileService = s.FileService
	return h
}

func (s *Server) trackHandler() *trackHandler {
	h := newTrackHandler()
	h.jobService = s.JobService
	h.playlistService = s.PlaylistService
	h.trackService = s.TrackService
	h.userService = s.UserService
	return h
}

func (s *Server) twilioHandler() *twilioHandler {
	h := newTwilioHandler()
	h.baseURL = s.URL()
	h.accountSID = s.Twilio.AccountSID
	h.jobService = s.JobService
	h.playlistService = s.PlaylistService
	h.smsService = s.SMSService
	h.trackService = s.TrackService
	h.userService = s.UserService
	return h
}

func (s *Server) attachLogOutputToContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r.WithContext(NewContext(r.Context(), s.LogOutput)))
	})
}

func (s *Server) detectAccept(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch path.Ext(r.URL.Path) {
		case ".json":
			r.Header.Set("Accept", "application/json")
		case ".rss":
			r.Header.Set("Accept", "text/xml")
		}

		next.ServeHTTP(w, r)
	})
}
