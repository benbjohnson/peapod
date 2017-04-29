package bolt

import (
	"context"

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
	} else if track.FileID == "" {
		return peapod.ErrTrackFileRequired
	}

	// Marshal and update record.
	if buf, err := MarshalTrack(track); err != nil {
		return err
	} else if bkt, err := tx.CreateBucketIfNotExists([]byte("Tracks")); err != nil {
		return err
	} else if err := bkt.Put(itob(track.ID), buf); err != nil {
		return err
	}
	return nil

}

func MarshalTrack(v *peapod.Track) ([]byte, error) {
	return proto.Marshal(&Track{
		ID:         int64(v.ID),
		PlaylistID: int64(v.PlaylistID),
		FileID:     v.FileID,
		Title:      v.Title,
		CreatedAt:  encodeTime(v.CreatedAt),
		UpdatedAt:  encodeTime(v.UpdatedAt),
	})
}
