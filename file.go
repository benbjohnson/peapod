package peapod

import (
	"context"
	"io"
	"regexp"
)

// File errors
const (
	ErrFileIDRequired = Error("file id required")
	ErrInvalidFileID  = Error("invalid file id")
)

// File represents an on-disk file.
type File struct {
	ID string `json:"id"`
}

// FileService represents a service for managing file objects.
type FileService interface {
	FindFileByID(ctx context.Context, id string) (*File, io.ReadCloser, error)
	CreateFile(ctx context.Context, f *File, r io.Reader) error
}

// IsValidFileID returns true if the id is in a valid format.
func IsValidFileID(id string) bool {
	return fileIDRegex.MatchString(id)
}

var fileIDRegex = regexp.MustCompile(`^[a-zA-Z0-9]+$`)
