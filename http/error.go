package http

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/middlemost/peapod"
)

const (
	ErrNotAcceptable = peapod.Error("not acceptable")
	ErrAssetNotFound = peapod.Error("asset not found")
)

// errorMap is a whitelist that maps errors to status codes.
var errorMap = map[error]int{
	ErrNotAcceptable:         http.StatusNotAcceptable,
	ErrTwilioAccountMismatch: http.StatusBadRequest,
	ErrInvalidSMSRequestBody: http.StatusBadRequest,
}

// ErrorStatusCode returns the HTTP status code for an error object.
func ErrorStatusCode(err error) int {
	if code, ok := errorMap[err]; ok {
		return code
	}
	return http.StatusInternalServerError
}

// Error writes an error reponse to the writer.
func Error(w http.ResponseWriter, r *http.Request, err error) {
	// Determine status code.
	code := ErrorStatusCode(err)

	// Log error.
	if logOutput := FromContext(r.Context()); logOutput != nil {
		fmt.Fprintf(logOutput, "http error: %d %s\n", code, err.Error())
	}

	// Mask unrecognized errors from end users.
	if _, ok := errorMap[err]; !ok {
		err = peapod.ErrInternal
	}

	// Write response.
	switch {
	case strings.Contains(r.Header.Get("Accept"), "application/json"):
		w.Header().Set("Context-Type", "application/json")
		w.WriteHeader(code)
		json.NewEncoder(w).Encode(&errorResponse{Err: err.Error()})

	default:
		w.Header().Set("Context-Type", "text/plain")
		w.WriteHeader(code)
		w.Write([]byte(err.Error()))
	}
}

type errorResponse struct {
	Err string `json:"error,omitempty"`
}
