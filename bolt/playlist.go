package bolt

import (
	"bytes"
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
	tx, err := s.db.Begin(ctx, false)
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

// FindPlaylistByToken returns a playlist and its tracks by token.
func (s *PlaylistService) FindPlaylistByToken(ctx context.Context, token string) (*peapod.Playlist, error) {
	tx, err := s.db.Begin(ctx, false)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// Retrieve id from the token.
	id := findPlaylistIDByToken(ctx, tx, token)
	if id == 0 {
		return nil, peapod.ErrPlaylistNotFound
	}

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

// FindPlaylistsByUserID returns a list of all playlists for a user.
func (s *PlaylistService) FindPlaylistsByUserID(ctx context.Context, id int) ([]*peapod.Playlist, error) {
	tx, err := s.db.Begin(ctx, false)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	return findPlaylistsByUserID(ctx, tx, id)
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

func findPlaylistIDByToken(ctx context.Context, tx *Tx, token string) int {
	bkt := tx.Bucket([]byte("Playlists.Token"))
	if bkt == nil {
		return 0
	}
	v := bkt.Get([]byte(token))
	if v == nil {
		return 0
	}
	return btoi(v)
}

func findPlaylistsByUserID(ctx context.Context, tx *Tx, id int) ([]*peapod.Playlist, error) {
	bkt := tx.Bucket([]byte("Users.Playlists"))
	if bkt == nil {
		return nil, nil
	}

	cur := bkt.Cursor()
	prefix := itob(id)
	a := make([]*peapod.Playlist, 0, 1)
	for k, _ := cur.Seek(prefix); bytes.HasPrefix(k, prefix); k, _ = cur.Next() {
		playlistID := btoi(k[8:])
		playlist, err := findPlaylistByID(ctx, tx, playlistID)
		if err != nil {
			return nil, err
		}
		assert(playlist != nil, "indexed playlist not found: id=%d", playlistID)
		a = append(a, playlist)
	}
	return a, nil
}

func createPlaylist(ctx context.Context, tx *Tx, playlist *peapod.Playlist) error {
	if playlist == nil {
		return peapod.ErrPlaylistRequired
	}

	bkt, err := tx.CreateBucketIfNotExists([]byte("Playlists"))
	if err != nil {
		return err
	}

	// Retrieve next sequence.
	id, _ := bkt.NextSequence()
	playlist.ID = int(id)

	// Generate external token.
	playlist.Token = tx.GenerateToken()

	// Update timestamps.
	playlist.CreatedAt = tx.Now

	// Save data.
	if err := savePlaylist(ctx, tx, playlist); err != nil {
		return err
	}

	// Index by owner.
	if err := updateIndex(ctx, tx, []byte("Users.Playlists"), 0, 0, playlist.OwnerID, playlist.ID); err != nil {
		return err
	}

	// Index by token.
	if bkt, err := tx.CreateBucketIfNotExists([]byte("Playlists.Token")); err != nil {
		return err
	} else if err := bkt.Put([]byte(playlist.Token), itob(playlist.ID)); err != nil {
		return err
	}

	return nil
}

func savePlaylist(ctx context.Context, tx *Tx, playlist *peapod.Playlist) error {
	// Validate record.
	if playlist.OwnerID == 0 {
		return peapod.ErrPlaylistOwnerRequired
	} else if !userExists(ctx, tx, playlist.OwnerID) {
		return peapod.ErrUserNotFound
	} else if playlist.Token == "" {
		return peapod.ErrPlaylistTokenRequired
	} else if playlist.Name == "" {
		return peapod.ErrPlaylistNameRequired
	}

	// Update timestamp.
	playlist.UpdatedAt = tx.Now

	// Marshal and update record.
	if buf, err := marshalPlaylist(playlist); err != nil {
		return err
	} else if bkt, err := tx.CreateBucketIfNotExists([]byte("Playlists")); err != nil {
		return err
	} else if err := bkt.Put(itob(playlist.ID), buf); err != nil {
		return err
	}
	return nil
}

func marshalPlaylist(v *peapod.Playlist) ([]byte, error) {
	return proto.Marshal(&Playlist{
		ID:        int64(v.ID),
		OwnerID:   int64(v.OwnerID),
		Token:     v.Token,
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
		OwnerID:   int(pb.OwnerID),
		Token:     pb.Token,
		Name:      pb.Name,
		CreatedAt: decodeTime(pb.CreatedAt),
		UpdatedAt: decodeTime(pb.UpdatedAt),
	}
	return nil
}
