package bolt

import (
	"context"
	"fmt"
	"io"
	"sync"

	"github.com/gogo/protobuf/proto"
	"github.com/middlemost/peapod"
)

// Ensure service implement interface.
var _ peapod.JobService = &JobService{}

// JobService represents a service for creating and processing jobs.
type JobService struct {
	db *DB

	notify chan struct{}
	wg     sync.WaitGroup

	FileService       peapod.FileService
	SMSService        peapod.SMSService
	TrackService      peapod.TrackService
	UserService       peapod.UserService
	URLTrackGenerator peapod.URLTrackGenerator

	LogOutput io.Writer
}

// NewJobService returns a new instance of JobService.
func NewJobService(db *DB) *JobService {
	return &JobService{db: db}
}

// Open initializes the job processing queue.
func (s *JobService) Open() error {
	s.notify = make(chan struct{}, 1)
	return nil
}

// Close stops the job processing queue and waits for outstanding workers.
func (s *JobService) Close() error {
	close(s.notify)
	s.wg.Wait()
	return nil
}

// CreateJob creates adds a job to the job queue.
func (s *JobService) CreateJob(ctx context.Context, job *peapod.Job) error {
	tx, err := s.db.BeginAuth(ctx, true)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Create job & commit.
	if err := func() error {
		if err := createJob(ctx, tx, job); err != nil {
			return err
		} else if err := tx.Commit(); err != nil {
			return err
		}
		return nil
	}(); err != nil {
		job.ID = 0
		return nil
	}

	// Signal change notification.
	select {
	case s.notify <- struct{}{}:
	default:
	}

	return nil
}

// NextJob returns the next job in the job queue and marks it as started.
func (s *JobService) NextJob(ctx context.Context) (*peapod.Job, error) {
	tx, err := s.db.BeginAuth(ctx, true)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// Retrieve next job id.
	id := nextJobID(ctx, tx)
	if id == 0 {
		return nil, nil
	}

	// Mark job as started.
	if err := setJobStatus(ctx, tx, id, peapod.JobStatusProcessing, nil); err != nil {
		return nil, err
	}

	// Fetch job.
	job, err := findJobByID(ctx, tx, id)
	if err != nil {
		return nil, err
	}

	// Attach owner.
	owner, err := findUserByID(ctx, tx, job.OwnerID)
	if err != nil {
		return nil, err
	} else if owner == nil {
		return nil, peapod.ErrJobOwnerNotFound
	}
	job.Owner = owner

	// Commit changes.
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return job, nil
}

// CompleteJob marks a job as completed or failed.
func (s *JobService) CompleteJob(ctx context.Context, id int, e error) error {
	tx, err := s.db.BeginAuth(ctx, true)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Determine status based on error.
	status := peapod.JobStatusCompleted
	if e != nil {
		status = peapod.JobStatusFailed
	}

	// Update status & commit.
	if err := setJobStatus(ctx, tx, id, status, e); err != nil {
		return err
	} else if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}

// ResetJobQueue resets all queued jobs to a pending status.
// This should be called when the process starts so that all jobs are restarted.
func (s *JobService) ResetJobQueue(ctx context.Context) error {
	tx, err := s.db.BeginAuth(ctx, true)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Fetch queue.
	bkt := tx.Bucket([]byte("JobQueue"))
	if bkt == nil {
		return nil
	}
	cur := bkt.Cursor()

	// Iterate over queue.
	for k, v := cur.First(); k != nil; k, v = cur.Next() {
		if err := setJobStatus(ctx, tx, btoi(v), peapod.JobStatusPending, nil); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func findJobByID(ctx context.Context, tx *Tx, id int) (*peapod.Job, error) {
	bkt := tx.Bucket([]byte("Jobs"))
	if bkt == nil {
		return nil, nil
	}

	var job peapod.Job
	if buf := bkt.Get(itob(id)); buf == nil {
		return nil, nil
	} else if err := unmarshalJob(buf, &job); err != nil {
		return nil, err
	}
	return &job, nil
}

func jobExists(ctx context.Context, tx *Tx, id int) bool {
	bkt := tx.Bucket([]byte("Jobs"))
	if bkt == nil {
		return false
	}
	return bkt.Get(itob(id)) != nil
}

func createJob(ctx context.Context, tx *Tx, job *peapod.Job) error {
	bkt, err := tx.CreateBucketIfNotExists([]byte("Jobs"))
	if err != nil {
		return err
	}

	// Retrieve next sequence.
	id, _ := bkt.NextSequence()
	job.ID = int(id)

	// Mark as pending.
	job.Status = peapod.JobStatusPending

	// Update timestamps.
	job.CreatedAt = tx.Now

	// Save data & add to end of job queue.
	if err := saveJob(ctx, tx, job); err != nil {
		return err
	} else if addJobToQueue(ctx, tx, job.ID); err != nil {
		return err
	}

	return nil
}

func saveJob(ctx context.Context, tx *Tx, job *peapod.Job) error {
	// Validate record.
	if peapod.IsValidJobType(job.Type) {
		return peapod.ErrInvalidJobType
	} else if peapod.IsValidJobStatus(job.Status) {
		return peapod.ErrInvalidJobStatus
	} else if job.OwnerID == 0 {
		return peapod.ErrJobOwnerRequired
	} else if !userExists(ctx, tx, job.OwnerID) {
		return peapod.ErrUserNotFound
	}

	// Marshal and update record.
	if buf, err := marshalJob(job); err != nil {
		return err
	} else if bkt, err := tx.CreateBucketIfNotExists([]byte("Jobs")); err != nil {
		return err
	} else if err := bkt.Put(itob(job.ID), buf); err != nil {
		return err
	}
	return nil
}

func setJobStatus(ctx context.Context, tx *Tx, id int, status string, e error) error {
	// Fetch job.
	job, err := findJobByID(ctx, tx, id)
	if err != nil {
		return err
	} else if job == nil {
		return peapod.ErrJobNotFound
	}

	// Ignore if status unchanged.
	if job.Status == status {
		return nil
	}

	// If status is a completion status then remove from the job queue.
	switch status {
	case peapod.JobStatusCompleted, peapod.JobStatusFailed:
		if err := removeJobFromQueue(ctx, tx, job.ID); err != nil {
			return err
		}
	}

	// Update status and save job.
	job.Status = status
	job.Error = errorString(e)
	if err := saveJob(ctx, tx, job); err != nil {
		return err
	}

	return nil
}

// nextJobID returns the next job in the job queue. Returns zero if queue is empty.
func nextJobID(ctx context.Context, tx *Tx) int {
	bkt := tx.Bucket([]byte("JobQueue"))
	if bkt == nil {
		return 0
	}

	_, v := bkt.Cursor().First()
	if v == nil {
		return 0
	}
	return btoi(v)
}

// addJobToQueue appends a job to the end of the queue.
func addJobToQueue(ctx context.Context, tx *Tx, id int) error {
	bkt, err := tx.CreateBucketIfNotExists([]byte("JobQueue"))
	if err != nil {
		return err
	}
	seq, _ := bkt.NextSequence()
	return bkt.Put(itob(int(seq)), itob(id))
}

// removeJobFromQueue finds a job in the queue and removes it.
func removeJobFromQueue(ctx context.Context, tx *Tx, id int) error {
	bkt := tx.Bucket([]byte("JobQueue"))
	if bkt == nil {
		return nil
	}

	cur := bkt.Cursor()
	for k, v := cur.First(); k != nil; k, v = cur.Next() {
		if id == btoi(v) {
			if err := cur.Delete(); err != nil {
				return err
			}
		}
	}
	return nil
}

// monitorJobQueue waits for notifications
func (s *JobService) monitorJobQueue() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	for {
		// Wait for next change notification. Exit on close.
		if _, ok := <-s.notify; !ok {
			return
		}

		// Read next job.
		job, err := s.NextJob(ctx)
		if err != nil {
			fmt.Fprintf(s.LogOutput, "error: next job: err=%s\n", err)
			continue
		} else if job == nil {
			continue
		}

		// Launch job processing in a separate goroutine.
		s.wg.Add(1)
		go func(ctx context.Context, job *peapod.Job) {
			defer s.wg.Done()
			s.executeJob(ctx, job)
		}(ctx, job)
	}
}

// executeJob processes a job in a separate goroutine.
func (s *JobService) executeJob(ctx context.Context, job *peapod.Job) {
	// Log job start.
	fmt.Fprintf(s.LogOutput, "job started: id=%d user=%d\n", job.ID, job.OwnerID)

	// Execute job.
	p := peapod.JobExecutor{
		FileService:  s.FileService,
		SMSService:   s.SMSService,
		TrackService: s.TrackService,
		UserService:  s.UserService,

		URLTrackGenerator: s.URLTrackGenerator,
	}
	err := p.ExecuteJob(ctx, job)

	// Mark job as completed.
	if e := s.CompleteJob(ctx, job.ID, err); err != nil {
		fmt.Fprintf(s.LogOutput, "error: complete job: id=%d err=%s\n", job.ID, e)
		return
	}

	// Log job completion.
	fmt.Fprintf(s.LogOutput, "job completed: id=%d user=%d err=%q\n", job.ID, job.OwnerID, errorString(err))
}

func marshalJob(v *peapod.Job) ([]byte, error) {
	return proto.Marshal(&Job{
		ID:         int64(v.ID),
		OwnerID:    int64(v.OwnerID),
		Type:       v.Type,
		Status:     v.Status,
		PlaylistID: int64(v.PlaylistID),
		URL:        v.URL,
		Error:      v.Error,
		CreatedAt:  encodeTime(v.CreatedAt),
		UpdatedAt:  encodeTime(v.UpdatedAt),
	})
}

func unmarshalJob(data []byte, v *peapod.Job) error {
	var pb Job
	if err := proto.Unmarshal(data, &pb); err != nil {
		return err
	}
	*v = peapod.Job{
		ID:         int(pb.ID),
		OwnerID:    int(pb.OwnerID),
		Type:       pb.Type,
		Status:     pb.Status,
		PlaylistID: int(pb.PlaylistID),
		URL:        pb.URL,
		Error:      pb.Error,
		CreatedAt:  decodeTime(pb.CreatedAt),
		UpdatedAt:  decodeTime(pb.UpdatedAt),
	}
	return nil
}
