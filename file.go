package peapod

import (
	"context"
	"io"
	"regexp"
)

// File errors
const (
	ErrFilenameRequired = Error("filename required")
	ErrInvalidFilename  = Error("invalid filename")
)

// File represents an on-disk file.
type File struct {
	Name string `json:"name"`
	Size int64  `json:"size"`
}

// FileService represents a service for managing file objects.
type FileService interface {
	GenerateName(ext string) string
	FindFileByName(ctx context.Context, name string) (*File, io.ReadCloser, error)
	CreateFile(ctx context.Context, f *File, r io.Reader) error
}

// IsValidFilename returns true if the name is in a valid format.
func IsValidFilename(name string) bool {
	return fileIDRegex.MatchString(name)
}

var fileIDRegex = regexp.MustCompile(`^[a-z0-9]+(\.[a-z0-9]+)?$`)
