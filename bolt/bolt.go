package bolt

import (
	"context"
	"encoding/binary"
	"fmt"
	"time"
)

//go:generate protoc --gogo_out=. bolt.proto

// itob returns an 8-byte big-endian encoded byte slice of v.
func itob(v int) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(v))
	return b
}

// btoi returns an integer decoded from an 8-byte big-endian encoded byte slice.
func btoi(b []byte) int {
	return int(binary.BigEndian.Uint64(b))
}

func encodeTime(t time.Time) int64 {
	if t.IsZero() {
		return 0
	}
	return t.UnixNano()
}

func decodeTime(v int64) time.Time {
	if v == 0 {
		return time.Time{}
	}
	return time.Unix(0, v).UTC()
}

// updateIndex removes an index at <oldParentID,oldChildID> and adds <newParentID,newChildID>.
func updateIndex(ctx context.Context, tx *Tx, name []byte, oldParentID, oldChildID, newParentID, newChildID int) error {
	// Ignore if index is unchanged.
	if oldParentID == newParentID && oldChildID == newChildID {
		return nil
	}

	// Find index bucket.
	bkt, err := tx.CreateBucketIfNotExists(name)
	if err != nil {
		return err
	}

	// Remove old index entry, if specified.
	if oldParentID != 0 || oldChildID != 0 {
		if err := bkt.Delete(makeIndexKey(oldParentID, oldChildID)); err != nil {
			return err
		}
	}

	// Add new index entry, if specified.
	if newParentID != 0 || newChildID != 0 {
		if err := bkt.Put(makeIndexKey(newParentID, newChildID), nil); err != nil {
			return err
		}
	}

	return nil
}

func makeIndexKey(v0, v1 int) []byte {
	key := make([]byte, 16)
	binary.BigEndian.PutUint64(key[0:8], uint64(v0))
	binary.BigEndian.PutUint64(key[8:16], uint64(v1))
	return key

}

// assert panics with a formatted message if condition is false.
func assert(condition bool, format string, a ...interface{}) {
	if !condition {
		panic(fmt.Sprintf(format, a...))
	}
}
