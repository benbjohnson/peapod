package mock

import (
	"context"

	"github.com/middlemost/peapod"
)

var _ peapod.JobService = &JobService{}

// JobService manages jobs in a job queue.
type JobService struct {
	CFn           func() <-chan struct{}
	CreateJobFn   func(ctx context.Context, job *peapod.Job) error
	NextJobFn     func(ctx context.Context) (*peapod.Job, error)
	CompleteJobFn func(ctx context.Context, id int, err error) error
}

func (s *JobService) C() <-chan struct{} {
	return s.CFn()
}

func (s *JobService) CreateJob(ctx context.Context, job *peapod.Job) error {
	return s.CreateJobFn(ctx, job)
}

func (s *JobService) NextJob(ctx context.Context) (*peapod.Job, error) {
	return s.NextJobFn(ctx)
}

func (s *JobService) CompleteJob(ctx context.Context, id int, err error) error {
	return s.CompleteJobFn(ctx, id, err)
}
