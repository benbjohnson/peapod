package peapod

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"sync"
	"time"
)

// Job errors.
const (
	ErrJobRequired      = Error("job required")
	ErrJobNotFound      = Error("job not found")
	ErrJobOwnerRequired = Error("job owner required")
	ErrJobOwnerNotFound = Error("job owner not found")
	ErrInvalidJobType   = Error("invalid job type")
	ErrInvalidJobStatus = Error("invalid job status")
)

// Job types.
const (
	JobTypeCreateTrackFromURL = "create_track_from_url"
)

// IsValidJobType returns true if v is a valid type.
func IsValidJobType(v string) bool {
	switch v {
	case JobTypeCreateTrackFromURL:
		return true
	default:
		return false
	}
}

// Job statuses.
const (
	JobStatusPending    = "pending"
	JobStatusProcessing = "processing"
	JobStatusCompleted  = "completed"
	JobStatusFailed     = "failed"
)

// IsValidJobType returns true if v is a valid type.
func IsValidJobStatus(v string) bool {
	switch v {
	case JobStatusPending, JobStatusProcessing, JobStatusCompleted, JobStatusFailed:
		return true
	default:
		return false
	}
}

// Job represents an task to be performed by a worker.
type Job struct {
	ID         int       `json:"id"`
	OwnerID    int       `json:"owner_id"`
	Owner      *User     `json:"owner,omitempty"`
	Type       string    `json:"type"`
	Status     string    `json:"status"`
	PlaylistID int       `json:"playlist_id,omitempty"`
	URL        string    `json:"url,omitempty"`
	Error      string    `json:"error,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// JobService manages jobs in a job queue.
type JobService interface {
	// Notification channel when a new job is ready.
	C() <-chan struct{}

	CreateJob(ctx context.Context, job *Job) error
	NextJob(ctx context.Context) (*Job, error)
	CompleteJob(ctx context.Context, id int, err error) error
}

// JobScheduler receives new jobs and schedules them for execution.
type JobScheduler struct {
	once    sync.Once
	closing chan struct{}
	wg      sync.WaitGroup

	FileService       FileService
	JobService        JobService
	SMSService        SMSService
	TrackService      TrackService
	UserService       UserService
	URLTrackGenerator URLTrackGenerator

	LogOutput io.Writer
}

// NewJobScheduler returns a new instance of JobScheduler.
func NewJobScheduler() *JobScheduler {
	return &JobScheduler{
		closing:   make(chan struct{}),
		LogOutput: ioutil.Discard,
	}
}

// Open initializes the job processing queue.
func (s *JobScheduler) Open() error {
	s.wg.Add(1)
	go func() { defer s.wg.Done(); s.monitor() }()
	return nil
}

// Close stops the job processing queue and waits for outstanding workers.
func (s *JobScheduler) Close() error {
	s.once.Do(func() { close(s.closing) })
	s.wg.Wait()
	return nil
}

// monitor waits for notifications from the job service and starts jobs.
func (s *JobScheduler) monitor() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	for {
		// Wait for next job or for the scheduler to close.
		select {
		case <-s.closing:
			return
		case <-s.JobService.C():
		}

		// Read next job.
		job, err := s.JobService.NextJob(ctx)
		if err != nil {
			fmt.Fprintf(s.LogOutput, "error: next job: err=%s\n", err)
			continue
		} else if job == nil {
			continue
		}

		// Launch job processing in a separate goroutine.
		s.wg.Add(1)
		go func(ctx context.Context, job *Job) {
			defer s.wg.Done()
			s.executeJob(ctx, job)
		}(ctx, job)
	}
}

// executeJob processes a job in a separate goroutine.
func (s *JobScheduler) executeJob(ctx context.Context, job *Job) {
	// Log job start.
	fmt.Fprintf(s.LogOutput, "job started: id=%d user=%d\n", job.ID, job.OwnerID)

	// Execute job.
	ex := JobExecutor{
		FileService:  s.FileService,
		SMSService:   s.SMSService,
		TrackService: s.TrackService,
		UserService:  s.UserService,

		URLTrackGenerator: s.URLTrackGenerator,
	}
	err := ex.ExecuteJob(ctx, job)

	// Mark job as completed.
	if e := s.JobService.CompleteJob(ctx, job.ID, err); e != nil {
		fmt.Fprintf(s.LogOutput, "error: complete job: id=%d err=%s\n", job.ID, e)
		return
	}

	// Log job completion.
	fmt.Fprintf(s.LogOutput, "job completed: id=%d user=%d err=%q\n", job.ID, job.OwnerID, errorString(err))
}

// JobExecutor represents a worker that executes a job.
type JobExecutor struct {
	FileService  FileService
	SMSService   SMSService
	TrackService TrackService
	UserService  UserService

	URLTrackGenerator URLTrackGenerator
}

// ExecuteJob processes a single job.
func (e *JobExecutor) ExecuteJob(ctx context.Context, job *Job) error {
	switch job.Type {
	case JobTypeCreateTrackFromURL:
		return e.createTrackFromURL(ctx, job)
	default:
		return ErrInvalidJobType
	}
}

// createTrackFromURL generates a new track based on a URL.
func (e *JobExecutor) createTrackFromURL(ctx context.Context, job *Job) error {
	// Parse URL.
	u, err := url.Parse(job.URL)
	if err != nil {
		return ErrInvalidURL
	}

	// Lookup user.
	user, err := e.UserService.FindUserByID(ctx, job.OwnerID)
	if err != nil {
		return err
	} else if user == nil {
		return ErrUserNotFound
	}

	// Generate track & file contents from a URL.
	track, rc, err := e.URLTrackGenerator.GenerateTrackFromURL(ctx, *u)
	if err != nil {
		return err
	}
	defer rc.Close()

	// Create a file from the reader.
	var file File
	if err := e.FileService.CreateFile(ctx, &file, rc); err != nil {
		return err
	}

	// Attach playlist & file to track.
	track.PlaylistID = job.PlaylistID
	track.FileID = file.ID

	// Create new track.
	if err := e.TrackService.CreateTrack(ctx, track); err != nil {
		return err
	}

	// Notify user of success.
	msg := &SMS{
		To:   user.MobileNumber,
		Body: `Finished processing. You track has been added to your playlist.`,
	}
	if err := e.SMSService.SendSMS(ctx, msg); err != nil {
		return err
	}

	return nil
}

func errorString(err error) string {
	if err != nil {
		return err.Error()
	}
	return ""
}
