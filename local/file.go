package local

import (
	"context"
	"io"

	"github.com/middlemost/peapod"
)

// FileService represents a service for serving files from the local filesystem.
type FileService struct {
	path string
}

// NewFileService returns a new instance of FileService.
func NewFileService(path string) *FileService {
	return &FileService{path: path}
}

func (s *FileService) FindFileByID(ctx context.Context, id string) (*peapod.File, io.Reader, error) {
	panic("TODO")
}

func (s *FileService) CreateFile(ctx context.Context, f *peapod.File, r io.Reader) error {
	panic("TODO")
}
