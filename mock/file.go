package mock

import (
	"context"
	"io"

	"github.com/middlemost/peapod"
)

var _ peapod.FileService = &FileService{}

type FileService struct {
	FindFileByIDFn func(ctx context.Context, id string) (*peapod.File, io.ReadCloser, error)
	CreateFileFn   func(ctx context.Context, f *peapod.File, r io.Reader) error
}

func (s *FileService) FindFileByID(ctx context.Context, id string) (*peapod.File, io.ReadCloser, error) {
	return s.FindFileByIDFn(ctx, id)
}

func (s *FileService) CreateFile(ctx context.Context, f *peapod.File, r io.Reader) error {
	return s.CreateFileFn(ctx, f, r)
}
