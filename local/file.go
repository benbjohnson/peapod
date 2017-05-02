package local

import (
	"context"
	"io"
	"os"
	"path/filepath"

	"github.com/middlemost/peapod"
)

// FileService represents a service for serving files from the local filesystem.
type FileService struct {
	Path          string
	GenerateToken func() string
}

// NewFileService returns a new instance of FileService.
func NewFileService() *FileService {
	return &FileService{
		GenerateToken: peapod.GenerateToken,
	}
}

// GenerateName returns a randomly generated name with the given extension.
func (s *FileService) GenerateName(ext string) string {
	return s.GenerateToken() + ext
}

// FindFileByName returns a file and a reader to its contents.
// The read must be closed by the caller.
func (s *FileService) FindFileByName(ctx context.Context, name string) (*peapod.File, io.ReadCloser, error) {
	if name == "" {
		return nil, nil, peapod.ErrFilenameRequired
	} else if !peapod.IsValidFilename(name) {
		return nil, nil, peapod.ErrInvalidFilename
	}

	// Stat file.
	path := filepath.Join(s.Path, name)
	fi, err := os.Stat(path)
	if os.IsNotExist(err) {
		return nil, nil, nil
	} else if err != nil {
		return nil, nil, err
	}

	// Open local file.
	file, err := os.Open(path)
	if os.IsNotExist(err) {
		return nil, nil, nil
	} else if err != nil {
		return nil, nil, err
	}

	// Generate file object.
	f := &peapod.File{Name: name, Size: fi.Size()}

	return f, file, nil
}

// CreateFile creates a new file with the contents of r.
func (s *FileService) CreateFile(ctx context.Context, f *peapod.File, r io.Reader) error {
	// Ensure parent path exists.
	if err := os.MkdirAll(s.Path, 0777); err != nil {
		return err
	}

	// Create file inside directory.
	path := filepath.Join(s.Path, f.Name)
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	// Copy contents.
	if _, err := io.Copy(file, r); err != nil {
		os.Remove(file.Name())
		return err
	}

	// Close file handle.
	if err := file.Close(); err != nil {
		return err
	}

	// Read size.
	fi, err := os.Stat(path)
	if err != nil {
		return err
	}
	f.Size = fi.Size()

	return nil
}
