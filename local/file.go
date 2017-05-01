package local

import (
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/middlemost/peapod"
)

// FileService represents a service for serving files from the local filesystem.
type FileService struct {
	Path string

	GenerateToken func() string
}

// NewFileService returns a new instance of FileService.
func NewFileService() *FileService {
	return &FileService{
		GenerateToken: MustGenerateToken,
	}
}

// FindFileByID returns a file and a reader to its contents.
// The read must be closed by the caller.
func (s *FileService) FindFileByID(ctx context.Context, id string) (*peapod.File, io.ReadCloser, error) {
	if id == "" {
		return nil, nil, peapod.ErrFileIDRequired
	} else if !peapod.IsValidFileID(id) {
		return nil, nil, peapod.ErrInvalidFileID
	}

	// Open local file.
	file, err := os.Open(filepath.Join(s.Path, id))
	if os.IsNotExist(err) {
		return nil, nil, nil
	} else if err != nil {
		return nil, nil, err
	}

	// Generate file object.
	f := &peapod.File{ID: id}

	return f, file, nil
}

// CreateFile creates a new file with the contents of r. Returns the ID to f.ID.
func (s *FileService) CreateFile(ctx context.Context, f *peapod.File, r io.Reader) error {
	// Generate random ID.
	id := s.GenerateToken()

	// Ensure parent path exists.
	if err := os.MkdirAll(s.Path, 0777); err != nil {
		return err
	}

	// Create file inside directory.
	file, err := os.Create(filepath.Join(s.Path, id))
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

	// Assign id to file.
	f.ID = id

	return nil
}

// MustGenerateToken returns a random string.
func MustGenerateToken() string {
	buf := make([]byte, 20)
	if _, err := rand.Read(buf); err != nil {
		panic(err)
	}
	return fmt.Sprintf("%x", buf)
}
