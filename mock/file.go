package mock

import (
	"context"
	"io"

	"github.com/middlemost/peapod"
)

var _ peapod.FileService = &FileService{}

type FileService struct {
	GenerateNameFn   func(ext string) string
	FindFileByNameFn func(ctx context.Context, name string) (*peapod.File, io.ReadCloser, error)
	CreateFileFn     func(ctx context.Context, f *peapod.File, r io.Reader) error
}

func (s *FileService) GenerateName(ext string) string {
	return s.GenerateNameFn(ext)
}

func (s *FileService) FindFileByName(ctx context.Context, name string) (*peapod.File, io.ReadCloser, error) {
	return s.FindFileByNameFn(ctx, name)
}

func (s *FileService) CreateFile(ctx context.Context, f *peapod.File, r io.Reader) error {
	return s.CreateFileFn(ctx, f, r)
}
