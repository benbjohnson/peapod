package bolt_test

import (
	"io/ioutil"
	"os"
	"time"

	"github.com/middlemost/peapod/bolt"
)

var Now = time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC)

// DB is a test wrapper for bolt.DB.
type DB struct {
	*bolt.DB
}

// NewDB returns a new instance of DB.
func NewDB() *DB {
	db := &DB{DB: bolt.NewDB()}
	db.Now = func() time.Time { return Now }
	return db
}

// MustOpenDB opens a DB at a temporary file path.
func MustOpenDB() *DB {
	f, err := ioutil.TempFile("", "")
	if err != nil {
		panic(err)
	} else if err := f.Close(); err != nil {
		panic(err)
	}

	db := NewDB()
	db.Path = f.Name()
	if err := db.Open(); err != nil {
		panic(err)
	}
	return db
}

// Close closes the database and removes the underlying data file.
func (db *DB) Close() error {
	defer os.Remove(db.Path)
	return db.DB.Close()
}

// MustClose closes the database. Panic on error.
func (db *DB) MustClose() {
	if err := db.Close(); err != nil {
		panic(err)
	}
}
