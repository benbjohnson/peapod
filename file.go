package peapod

import (
	"context"
	"io"
	"time"
)

// File represents an on-disk file.
type File struct {
	ID        string    `json:"id"`
	Size      int64     `json:"size"`
	CreatedAt time.Time `json:"created_at"`
}

// FileService represents a service for managing file objects.
type FileService interface {
	FindFileByID(ctx context.Context, id string) (*File, io.Reader, error)
	CreateFile(ctx context.Context, f *File, r io.Reader) error
}
