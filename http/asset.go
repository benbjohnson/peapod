package http

import (
	"mime"
	"net/http"
	"path"
	"strconv"

	"github.com/pressly/chi"
)

//go:generate go-bindata -o assets.gen.go -pkg http -prefix assets -ignore "\\.go$" assets

// assetHandler represents an HTTP handler for embedded assets.
type assetHandler struct {
	router chi.Router
}

// newAssetHandler returns a new instance of assetHandler.
func newAssetHandler() *assetHandler {
	h := &assetHandler{router: chi.NewRouter()}
	h.router.Get("/:name", h.handleGet)
	return h
}

// ServeHTTP implements http.Handler.
func (h *assetHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.router.ServeHTTP(w, r)
}

func (h *assetHandler) handleGet(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

	buf, _ := Asset(name)
	if len(buf) == 0 {
		Error(r.Context(), w, r, ErrAssetNotFound)
		return
	}

	// Set headers.
	w.Header().Set("Content-Type", mime.TypeByExtension(path.Ext(name)))
	w.Header().Set("Content-Length", strconv.Itoa(len(buf)))

	// Write contents.
	w.Write(buf)
}
