package bolt

import (
	"bytes"
	"context"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/middlemost/peapod"
)

// Ensure service implements interface.
var _ peapod.TrackService = &TrackService{}

// TrackService represents a service to manage tracks.
type TrackService struct {
	db *DB
}

// NewTrackService returns a new instance of TrackService.
func NewTrackService(db *DB) *TrackService {
	return &TrackService{db: db}
}

// FindTrackByID returns a track by id.
func (s *TrackService) FindTrackByID(ctx context.Context, id int) (*peapod.Track, error) {
	tx, err := s.db.Begin(ctx, false)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	return findTrackByID(ctx, tx, id)
}

// CreateTrack creates a new track on a playlist.
func (s *TrackService) CreateTrack(ctx context.Context, track *peapod.Track) error {
	tx, err := s.db.BeginAuth(ctx, true)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Create track & commit.
	if err := func() error {
		if err := createTrack(ctx, tx, track); err != nil {
			return err
		} else if err := tx.Commit(); err != nil {
			return err
		}
		return nil
	}(); err != nil {
		track.ID = 0
		return nil
	}

	return nil
}

func findTrackByID(ctx context.Context, tx *Tx, id int) (*peapod.Track, error) {
	bkt := tx.Bucket([]byte("Tracks"))
	if bkt == nil {
		return nil, nil
	}

	var track peapod.Track
	if buf := bkt.Get(itob(id)); buf == nil {
		return nil, nil
	} else if err := unmarshalTrack(buf, &track); err != nil {
		return nil, err
	}
	return &track, nil
}

func trackExists(ctx context.Context, tx *Tx, id int) bool {
	bkt := tx.Bucket([]byte("Tracks"))
	if bkt == nil {
		return false
	}
	return bkt.Get(itob(id)) != nil
}

func createTrack(ctx context.Context, tx *Tx, track *peapod.Track) error {
	bkt, err := tx.CreateBucketIfNotExists([]byte("Tracks"))
	if err != nil {
		return err
	}

	// Retrieve next sequence.
	id, _ := bkt.NextSequence()
	track.ID = int(id)

	// Update timestamps.
	track.CreatedAt = tx.Now

	// Save data & add to index.
	if err := saveTrack(ctx, tx, track); err != nil {
		return err
	} else if err := updateIndex(ctx, tx, []byte("Playlists.Tracks"), 0, 0, track.PlaylistID, track.ID); err != nil {
		return err
	}
	return nil
}

func saveTrack(ctx context.Context, tx *Tx, track *peapod.Track) error {
	// Validate record.
	if track.PlaylistID == 0 {
		return peapod.ErrTrackPlaylistRequired
	} else if !playlistExists(ctx, tx, track.PlaylistID) {
		return peapod.ErrPlaylistNotFound
	} else if track.Filename == "" {
		return peapod.ErrTrackFilenameRequired
	}

	// Update timestamps.
	track.UpdatedAt = tx.Now

	// Marshal and update record.
	if buf, err := marshalTrack(track); err != nil {
		return err
	} else if bkt, err := tx.CreateBucketIfNotExists([]byte("Tracks")); err != nil {
		return err
	} else if err := bkt.Put(itob(track.ID), buf); err != nil {
		return err
	}
	return nil
}

func playlistTracks(ctx context.Context, tx *Tx, playlistID int) ([]*peapod.Track, error) {
	bkt := tx.Bucket([]byte("Playlists.Tracks"))
	if bkt == nil {
		return nil, nil
	}

	// Iterate over index.
	a := make([]*peapod.Track, 0, 10)
	cur := bkt.Cursor()
	prefix := itob(playlistID)
	for k, _ := cur.Seek(prefix); bytes.HasPrefix(k, prefix); k, _ = cur.Next() {
		track, err := findTrackByID(ctx, tx, btoi(k[8:]))
		if err != nil {
			return nil, err
		}
		a = append(a, track)
	}
	return a, nil
}

func marshalTrack(v *peapod.Track) ([]byte, error) {
	return proto.Marshal(&Track{
		ID:          int64(v.ID),
		PlaylistID:  int64(v.PlaylistID),
		Filename:    v.Filename,
		ContentType: v.ContentType,
		Title:       v.Title,
		Description: v.Description,
		Duration:    int64(v.Duration),
		FileSize:    int64(v.Size),
		CreatedAt:   encodeTime(v.CreatedAt),
		UpdatedAt:   encodeTime(v.UpdatedAt),
	})
}

func unmarshalTrack(data []byte, v *peapod.Track) error {
	var pb Track
	if err := proto.Unmarshal(data, &pb); err != nil {
		return err
	}
	*v = peapod.Track{
		ID:          int(pb.ID),
		PlaylistID:  int(pb.PlaylistID),
		Filename:    pb.Filename,
		ContentType: pb.ContentType,
		Title:       pb.Title,
		Description: pb.Description,
		Duration:    time.Duration(pb.Duration),
		Size:        int(pb.FileSize),
		CreatedAt:   decodeTime(pb.CreatedAt),
		UpdatedAt:   decodeTime(pb.UpdatedAt),
	}
	return nil
}
