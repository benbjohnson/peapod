package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/signal"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/middlemost/peapod"
	"github.com/middlemost/peapod/bolt"
	"github.com/middlemost/peapod/http"
	"github.com/middlemost/peapod/local"
	"github.com/middlemost/peapod/twilio"
	"github.com/middlemost/peapod/youtube_dl"
)

func main() {
	m := NewMain()

	// Parse command line flags.
	if err := m.ParseFlags(os.Args[1:]); err == flag.ErrHelp {
		os.Exit(1)
	} else if err != nil {
		fmt.Fprintln(m.Stderr, err)
		os.Exit(1)
	}

	// Load configuration.
	if err := m.LoadConfig(); err != nil {
		fmt.Fprintln(m.Stderr, err)
		os.Exit(1)
	}

	// Execute program.
	if err := m.Run(); err != nil {
		fmt.Fprintln(m.Stderr, err)
		os.Exit(1)
	}

	// Shutdown on SIGINT (CTRL-C).
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	<-c
	fmt.Fprintln(m.Stdout, "received interrupt, shutting down...")
}

// Main represents the main program execution.
type Main struct {
	ConfigPath string
	Config     Config

	// Input/output streams
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer

	closeFn func() error
}

// NewMain returns a new instance of Main.
func NewMain() *Main {
	return &Main{
		ConfigPath: DefaultConfigPath,
		Config:     DefaultConfig(),

		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,

		closeFn: func() error { return nil },
	}
}

// Close cleans up the program.
func (m *Main) Close() error { return m.closeFn() }

// Usage returns the usage message.
func (m *Main) Usage() string {
	return strings.TrimSpace(`
usage: peapod [flags]

The daemon process for managing peapod API requests and processing.

The following flags are available:

	-config PATH
		Specifies the configuration file to read.
		Defaults to ~/.peapod/config

`)
}

// ParseFlags parses the command line flags.
func (m *Main) ParseFlags(args []string) error {
	fs := flag.NewFlagSet("peapod", flag.ContinueOnError)
	fs.SetOutput(ioutil.Discard)
	fs.StringVar(&m.ConfigPath, "config", "", "config file")
	return fs.Parse(args)
}

// LoadConfig parses the configuration file.
func (m *Main) LoadConfig() error {
	// Default configuration path if not specified.
	path := m.ConfigPath
	if path == "" {
		path = DefaultConfigPath
	}

	// Interpolate path.
	if err := InterpolatePaths(&path); err != nil {
		return err
	}

	// Read configuration file.
	if _, err := toml.DecodeFile(path, &m.Config); os.IsNotExist(err) {
		if m.ConfigPath != "" {
			return err
		}
	} else if err != nil {
		return err
	}
	return nil
}

// Run executes the program.
func (m *Main) Run() error {
	// Interpolate config paths.
	dbPath := m.Config.Database.Path
	filePath := m.Config.File.Path
	if err := InterpolatePaths(&dbPath, &filePath); err != nil {
		return err
	}

	// Initialize file service.
	fileService := local.NewFileService()
	fileService.Path = filePath
	fmt.Fprintf(m.Stdout, "file storage: path=%s\n", m.Config.File.Path)

	// Initialize Twilio service.
	smsService := twilio.NewSMSService()
	smsService.AccountSID = m.Config.Twilio.AccountSID
	smsService.AuthToken = m.Config.Twilio.AuthToken
	smsService.From = m.Config.Twilio.From
	smsService.LogOutput = m.Stdout

	// Initialize youtube-dl.
	urlTrackGenerator := youtube_dl.NewURLTrackGenerator()
	urlTrackGenerator.Proxy = m.Config.YoutubeDL.Proxy

	// Open database.
	db := bolt.NewDB()
	db.Path = dbPath
	if err := db.Open(); err != nil {
		return err
	}
	fmt.Fprintf(m.Stdout, "database initialized: path=%s\n", m.Config.Database.Path)

	// Instantiate bolt services.
	jobService := bolt.NewJobService(db)
	playlistService := bolt.NewPlaylistService(db)
	trackService := bolt.NewTrackService(db)
	userService := bolt.NewUserService(db)

	// Reset job queue.
	if err := jobService.ResetJobQueue(context.Background()); err != nil {
		return fmt.Errorf("error: reset job queue: %s", err)
	}

	// Start job scheduler.
	jobScheduler := peapod.NewJobScheduler()
	jobScheduler.FileService = fileService
	jobScheduler.JobService = jobService
	jobScheduler.SMSService = smsService
	jobScheduler.TrackService = trackService
	jobScheduler.UserService = userService
	jobScheduler.URLTrackGenerator = urlTrackGenerator
	jobScheduler.LogOutput = m.Stdout

	if err := jobScheduler.Open(); err != nil {
		return fmt.Errorf("error: open job scheduler: %s", err)
	}

	// Initialize HTTP server.
	httpServer := http.NewServer()
	httpServer.Addr = m.Config.HTTP.Addr
	httpServer.Host = m.Config.HTTP.Host
	httpServer.Autocert = m.Config.HTTP.Autocert
	httpServer.Twilio.AccountSID = m.Config.Twilio.AccountSID
	httpServer.LogOutput = m.Stdout

	httpServer.FileService = fileService
	httpServer.TrackService = trackService
	httpServer.PlaylistService = playlistService
	httpServer.UserService = userService

	// Open HTTP server.
	if err := httpServer.Open(); err != nil {
		return err
	}
	fmt.Fprintf(m.Stdout, "http listening: %s\n", httpServer.URL())

	// Assign close function.
	m.closeFn = func() error {
		httpServer.Close()
		jobScheduler.Close()
		db.Close()
		return nil
	}

	return nil
}

// DefaultConfigPath is the default configuration path.
const DefaultConfigPath = "~/.peapod/config"

// Config represents a configuration file.
type Config struct {
	Database struct {
		Path string `toml:"path"`
	} `toml:"database"`

	File struct {
		Path string `toml:"path"`
	} `toml:"file"`

	HTTP struct {
		Addr     string `toml:"addr"`
		Host     string `toml:"host"`
		Autocert bool   `toml:"autocert"`
	} `toml:"http"`

	Twilio struct {
		AccountSID string `toml:"account-sid"`
		AuthToken  string `toml:"auth-token"`
		From       string `toml:"from"`
	} `toml:"twilio"`

	YoutubeDL struct {
		Proxy string `toml:"proxy"`
	} `toml:"youtube-dl"`
}

// NewConfig returns a configuration with default settings.
func DefaultConfig() Config {
	var c Config
	c.Database.Path = "~/.peapod/db"
	c.File.Path = "~/.peapod/file"
	c.HTTP.Addr = ":3000"
	return c
}

// InterpolatePaths replaces the tilde prefix with the user's home directory.
func InterpolatePaths(a ...*string) error {
	for _, s := range a {
		if !strings.HasPrefix(*s, "~/") {
			continue
		}

		u, err := user.Current()
		if err != nil {
			return err
		} else if u.HomeDir == "" {
			return errors.New("home directory not found")
		}
		*s = filepath.Join(u.HomeDir, strings.TrimPrefix(*s, "~/"))
	}
	return nil
}
