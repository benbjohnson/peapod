package peapod

import (
	"context"
)

// Job types.
const (
	JobTypeCreateTrackFromURL = "create_track_from_url"
)

// Job represents an task to be performed by a worker.
type Job struct {
	ID     int    `json:"id"`
	UserID int    `json:"user_id,omitempty"` // requesting user
	Type   string `json:"type"`              // type of work to perform
	URL    string `json:"url,omitempty"`     // related URL
}

// JobService manages jobs in a job queue.
type JobService interface {
	CreateJob(ctx context.Context, job *Job) error
}
