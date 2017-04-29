package http

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/http"
	"strings"

	"github.com/middlemost/peapod"
)

// errorMap is a whitelist that maps errors to status codes.
var errorMap = map[error]int{
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
func Error(ctx context.Context, w http.ResponseWriter, r *http.Request, err error) {
	// Determine status code.
	code := ErrorStatusCode(err)

	// Log error.
	if logOutput := FromContext(ctx); logOutput != nil {
		fmt.Fprintf(logOutput, "http error: %d %s\n", code, err.Error())
	}

	// Mask unrecognized errors from end users.
	if _, ok := errorMap[err]; !ok {
		err = peapod.ErrInternal
	}

	// Write response.
	switch {
	case strings.Contains(r.Header.Get("Accept"), "text/xml"): // twilio only
		w.Header().Set("Context-Type", "text/xml")
		w.WriteHeader(code)
		xml.NewEncoder(w).Encode(&twilioResponse{Message: err.Error()})

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
