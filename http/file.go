package http

import (
	"io"
	"mime"
	"net/http"
	"net/url"
	"path"
	"strconv"

	"github.com/middlemost/peapod"
	"github.com/pressly/chi"
)

// fileHandler represents an HTTP handler for files.
type fileHandler struct {
	router chi.Router

	baseURL     url.URL
	fileService peapod.FileService
}

// newFileHandler returns a new instance of fileHandler.
func newFileHandler() *fileHandler {
	h := &fileHandler{router: chi.NewRouter()}
	h.router.Get("/:name", h.handleGet)
	return h
}

// ServeHTTP implements http.Handler.
func (h *fileHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.router.ServeHTTP(w, r)
}

func (h *fileHandler) handleGet(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	name := chi.URLParam(r, "name")

	// Fetch file.
	f, rc, err := h.fileService.FindFileByName(ctx, name)
	if err != nil {
		Error(w, r, err)
		return
	}
	defer rc.Close()

	// Set headers.
	w.Header().Set("Content-Type", mime.TypeByExtension(path.Ext(f.Name)))
	w.Header().Set("Content-Length", strconv.FormatInt(f.Size, 10))

	// Write file contents to response.
	if _, err := io.Copy(w, rc); err != nil {
		Error(w, r, err)
		return
	}
}
