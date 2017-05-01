package bolt

import (
	"context"

	"github.com/gogo/protobuf/proto"
	"github.com/middlemost/peapod"
)

// Ensure service implements interface.
var _ peapod.PlaylistService = &PlaylistService{}

// PlaylistService represents a service to manage playlists.
type PlaylistService struct {
	db *DB
}

// NewPlaylistService returns a new instance of PlaylistService.
func NewPlaylistService(db *DB) *PlaylistService {
	return &PlaylistService{db: db}
}

// FindPlaylistByID returns a playlist and its tracks by id.
func (s *PlaylistService) FindPlaylistByID(ctx context.Context, id int) (*peapod.Playlist, error) {
	tx, err := s.db.BeginAuth(ctx, false)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// Retrieve playlist.
	playlist, err := findPlaylistByID(ctx, tx, id)
	if err != nil {
		return nil, err
	}

	// Attach tracks.
	tracks, err := playlistTracks(ctx, tx, playlist.ID)
	if err != nil {
		return nil, err
	}
	playlist.Tracks = tracks

	return playlist, nil
}

func findPlaylistByID(ctx context.Context, tx *Tx, id int) (*peapod.Playlist, error) {
	bkt := tx.Bucket([]byte("Playlists"))
	if bkt == nil {
		return nil, nil
	}

	var playlist peapod.Playlist
	if buf := bkt.Get(itob(id)); buf == nil {
		return nil, nil
	} else if err := unmarshalPlaylist(buf, &playlist); err != nil {
		return nil, err
	}
	return &playlist, nil
}

func playlistExists(ctx context.Context, tx *Tx, id int) bool {
	bkt := tx.Bucket([]byte("Playlists"))
	if bkt == nil {
		return false
	}
	return bkt.Get(itob(id)) != nil
}

func marshalPlaylist(v *peapod.Playlist) ([]byte, error) {
	return proto.Marshal(&Playlist{
		ID:        int64(v.ID),
		Name:      v.Name,
		CreatedAt: encodeTime(v.CreatedAt),
		UpdatedAt: encodeTime(v.UpdatedAt),
	})
}

func unmarshalPlaylist(data []byte, v *peapod.Playlist) error {
	var pb Playlist
	if err := proto.Unmarshal(data, &pb); err != nil {
		return err
	}
	*v = peapod.Playlist{
		ID:        int(pb.ID),
		Name:      pb.Name,
		CreatedAt: decodeTime(pb.CreatedAt),
		UpdatedAt: decodeTime(pb.UpdatedAt),
	}
	return nil
}
