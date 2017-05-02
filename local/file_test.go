package local_test

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/middlemost/peapod"
	"github.com/middlemost/peapod/local"
)

// Ensure file service can create and fetch a file.
func TestFileService(t *testing.T) {
	s := NewFileService()
	defer s.MustClose()

	// Create file.
	var f peapod.File
	if err := s.CreateFile(context.Background(), &peapod.File{Name: "0001"}, strings.NewReader("ABC")); err != nil {
		t.Fatal(err)
	}

	// Fetch file & verify.
	if other, rc, err := s.FindFileByName(context.Background(), "0001"); err != nil {
		t.Fatal(err)
	} else if !reflect.DeepEqual(other, &peapod.File{Name: "0001", Size: 3}) {
		t.Fatalf("unexpected file: %#v", f)
	} else if buf, err := ioutil.ReadAll(rc); err != nil {
		t.Fatal(err)
	} else if string(buf) != "ABC" {
		t.Fatalf("unexpected file data: %q", buf)
	} else if err := rc.Close(); err != nil {
		t.Fatal(err)
	}
}

// FileService is a test wrapper for local.FileService.
type FileService struct {
	*local.FileService
}

// NewFileService returns a file service in a temporary directory.
func NewFileService() *FileService {
	path, err := ioutil.TempDir("", "peapod-")
	if err != nil {
		panic(err)
	}

	s := &FileService{FileService: local.NewFileService()}
	s.Path = path
	return s
}

// MustClose cleans up the temporary directory used by the service.
func (s *FileService) MustClose() {
	if err := os.RemoveAll(s.Path); err != nil {
		panic(err)
	}
}

// SequentialTokenGenerator returns an autoincrementing token.
func SequentialTokenGenerator() func() string {
	var i int
	return func() string {
		i++
		return fmt.Sprintf("%04x", i)
	}
}
