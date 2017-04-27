package http

import (
	"io"
	"io/ioutil"
	"net"
	"net/http"

	"github.com/middlemost/peapod"
	"github.com/pressly/chi"
	"github.com/pressly/chi/middleware"
	"golang.org/x/crypto/acme/autocert"
)

// Server represents an HTTP server.
type Server struct {
	ln net.Listener

	// Services
	TrackService    peapod.TrackService
	PlaylistService peapod.PlaylistService
	FileService     peapod.FileService
	UserService     peapod.UserService

	// Server options.
	Addr        string // bind address
	Host        string // external hostname
	Autocert    bool   // ACME autocert
	Recoverable bool   // panic recovery
	LogOutput   io.Writer
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
func (s *Server) URL() string {
	if s.ln == nil {
		return ""
	}

	if s.Autocert {
		return "https://" + s.Host
	}
	return "http://" + s.ln.Addr().String()
}

func (s *Server) router() http.Handler {
	r := chi.NewRouter()

	// Attach router middleware.
	r.Use(middleware.RealIP)
	if s.Recoverable {
		r.Use(middleware.Recoverer)
	}
	r.Mount("/debug", middleware.Profiler())

	// Create API routes.
	r.Route("/", func(r chi.Router) {
		r.Use(middleware.DefaultCompress)
		r.Get("/ping", s.handlePing)
		// r.Mount("/files", s.fileHandler())
		// r.Mount("/playlists", s.playlistHandler())
		// r.Mount("/tracks", s.trackHandler())
		// r.Mount("/users", s.userHandler())
	})

	return r
}

// handlePing verifies the database connection and returns a success.
func (s *Server) handlePing(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte(`{"status:":"ok"}` + "\n"))
}
