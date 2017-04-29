package peapod

import (
	"context"
	"net/url"
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
	CreateJob(ctx context.Context, job *Job) error
	NextJob(ctx context.Context) (*Job, error)
	CompleteJob(ctx context.Context, id int, err error) error
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
