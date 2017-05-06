package bolt

import (
	"context"

	"github.com/gogo/protobuf/proto"
	"github.com/middlemost/peapod"
)

// Ensure service implement interface.
var _ peapod.JobService = &JobService{}

// JobService represents a service for creating and processing jobs.
type JobService struct {
	db *DB

	c chan struct{}
}

// NewJobService returns a new instance of JobService.
func NewJobService(db *DB) *JobService {
	return &JobService{
		db: db,
		c:  make(chan struct{}, 1),
	}
}

// C returns a channel that sends notifications of new jobs.
func (s *JobService) C() <-chan struct{} { return s.c }

// CreateJob creates adds a job to the job queue.
func (s *JobService) CreateJob(ctx context.Context, job *peapod.Job) error {
	tx, err := s.db.Begin(ctx, true)
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
		return err
	}

	// Signal change notification.
	select {
	case s.c <- struct{}{}:
	default:
	}

	return nil
}

// NextJob returns the next job in the job queue and marks it as started.
func (s *JobService) NextJob(ctx context.Context) (*peapod.Job, error) {
	tx, err := s.db.Begin(ctx, true)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// Retrieve next job id.
	job, err := nextJob(ctx, tx)
	if err != nil {
		return nil, err
	} else if job == nil {
		return nil, nil
	}

	// Mark job as started.
	if err := setJobStatus(ctx, tx, job.ID, peapod.JobStatusProcessing, nil); err != nil {
		return nil, err
	}

	// Re-fetch job.
	job, err = findJobByID(ctx, tx, job.ID)
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
	tx, err := s.db.Begin(ctx, true)
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
	tx, err := s.db.Begin(ctx, true)
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
	if !peapod.IsValidJobType(job.Type) {
		return peapod.ErrInvalidJobType
	} else if !peapod.IsValidJobStatus(job.Status) {
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

// nextJob returns the next pending job in the job queue.
func nextJob(ctx context.Context, tx *Tx) (*peapod.Job, error) {
	bkt := tx.Bucket([]byte("JobQueue"))
	if bkt == nil {
		return nil, nil
	}

	cur := bkt.Cursor()
	for k, v := cur.First(); k != nil; k, v = cur.Next() {
		job, err := findJobByID(ctx, tx, btoi(v))
		if err != nil {
			return nil, err
		} else if job.Status == peapod.JobStatusPending {
			return job, nil
		}
	}

	return nil, nil
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

func marshalJob(v *peapod.Job) ([]byte, error) {
	return proto.Marshal(&Job{
		ID:         int64(v.ID),
		OwnerID:    int64(v.OwnerID),
		Type:       v.Type,
		Status:     v.Status,
		PlaylistID: int64(v.PlaylistID),
		Title:      v.Title,
		URL:        v.URL,
		Text:       v.Text,
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
		Title:      pb.Title,
		URL:        pb.URL,
		Text:       pb.Text,
		Error:      pb.Error,
		CreatedAt:  decodeTime(pb.CreatedAt),
		UpdatedAt:  decodeTime(pb.UpdatedAt),
	}
	return nil
}
