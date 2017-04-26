package peapod

import (
	"context"
	"time"
)

// File represents an on-disk file.
type File struct {
	ID          string
	ContentType string
	CreatedAt   time.Time `json:"created_at"`
}

// FileService represents a service for managing file objects.
type FileService interface {
	FindFileByID(ctx context.Context, id string) (*File, io.Reader, error)
	CreateFile(ctx context.Context, f *File, r io.Reader) error
}
