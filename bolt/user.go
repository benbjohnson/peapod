package bolt

import (
	"context"

	"github.com/gogo/protobuf/proto"
	"github.com/middlemost/peapod"
)

// Ensure service implements interface.
var _ peapod.UserService = &UserService{}

// UserService represents a service to manage users.
type UserService struct {
	db *DB
}

// NewUserService returns a new instance of UserService.
func NewUserService(db *DB) *UserService {
	return &UserService{db: db}
}

// FindUserByID returns a user with a given id.
func (s *UserService) FindUserByID(ctx context.Context, id int) (*peapod.User, error) {
	tx, err := s.db.BeginAuth(ctx, false)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	user, err := findUserByID(ctx, tx, id)
	if err != nil {
		return nil, err
	}
	return user, nil
}

// FindUserByMobileNumber returns a user by mobile number.
func (s *UserService) FindUserByMobileNumber(ctx context.Context, mobileNumber string) (*peapod.User, error) {
	tx, err := s.db.BeginAuth(ctx, false)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	id := findUserIDByMobileNumber(ctx, tx, mobileNumber)
	if id == 0 {
		return nil, nil
	}

	user, err := findUserByID(ctx, tx, id)
	if err != nil {
		return nil, err
	}
	return user, nil
}

// FindOrCreateUserByMobileNumber returns an existing user by mobile number.
// If a user is not found then a new one is created.
func (s *UserService) FindOrCreateUserByMobileNumber(ctx context.Context, mobileNumber string) (*peapod.User, error) {
	tx, err := s.db.BeginAuth(ctx, true)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// Find existing user by number.
	if id := findUserIDByMobileNumber(ctx, tx, mobileNumber); id != 0 {
		if user, err := findUserByID(ctx, tx, id); err != nil {
			return nil, err
		} else if user != nil {
			return user, nil
		}
	}

	// Create a user if one doesn't exist.
	user := &peapod.User{MobileNumber: mobileNumber}
	if err := createUser(ctx, tx, user); err != nil {
		return nil, err
	}

	// Commit changes.
	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return user, nil
}

func findUserByID(ctx context.Context, tx *Tx, id int) (*peapod.User, error) {
	bkt := tx.Bucket([]byte("Users"))
	if bkt == nil {
		return nil, nil
	}

	var u peapod.User
	if buf := bkt.Get(itob(id)); buf == nil {
		return nil, nil
	} else if err := unmarshalUser(buf, &u); err != nil {
		return nil, err
	}
	return &u, nil
}

func userExists(ctx context.Context, tx *Tx, id int) bool {
	bkt := tx.Bucket([]byte("Users"))
	if bkt == nil {
		return false
	}
	return bkt.Get(itob(id)) != nil
}

func findUserIDByMobileNumber(ctx context.Context, tx *Tx, mobileNumber string) int {
	bkt := tx.Bucket([]byte("Users.MobileNumber"))
	if bkt == nil {
		return 0
	}
	v := bkt.Get([]byte(mobileNumber))
	if v == nil {
		return 0
	}
	return btoi(v)
}

func createUser(ctx context.Context, tx *Tx, user *peapod.User) error {
	if user == nil {
		return peapod.ErrUserRequired
	} else if id := findUserIDByMobileNumber(ctx, tx, user.MobileNumber); id != 0 {
		return peapod.ErrUserMobileNumberInUse
	}

	bkt, err := tx.CreateBucketIfNotExists([]byte("Users"))
	if err != nil {
		return err
	}

	// Retrieve next sequence.
	id, _ := bkt.NextSequence()
	user.ID = int(id)

	// Update timestamps.
	user.CreatedAt = tx.Now

	// Save data.
	if err := saveUser(ctx, tx, user); err != nil {
		return err
	}

	// Index by mobile number.
	if bkt, err := tx.CreateBucketIfNotExists([]byte("Users.MobileNumber")); err != nil {
		return err
	} else if err := bkt.Put([]byte(user.MobileNumber), itob(user.ID)); err != nil {
		return err
	}

	return nil
}

func saveUser(ctx context.Context, tx *Tx, user *peapod.User) error {
	// Validate record.
	if user.MobileNumber == "" {
		return peapod.ErrUserMobileNumberRequired
	}

	// Update timestamp.
	user.UpdatedAt = tx.Now

	// Marshal and update record.
	if buf, err := marshalUser(user); err != nil {
		return err
	} else if bkt, err := tx.CreateBucketIfNotExists([]byte("Users")); err != nil {
		return err
	} else if err := bkt.Put(itob(user.ID), buf); err != nil {
		return err
	}
	return nil
}

func marshalUser(v *peapod.User) ([]byte, error) {
	return proto.Marshal(&User{
		ID:           int64(v.ID),
		MobileNumber: v.MobileNumber,
		CreatedAt:    encodeTime(v.CreatedAt),
		UpdatedAt:    encodeTime(v.UpdatedAt),
	})
}

func unmarshalUser(data []byte, v *peapod.User) error {
	var pb User
	if err := proto.Unmarshal(data, &pb); err != nil {
		return err
	}
	*v = peapod.User{
		ID:           int(pb.ID),
		MobileNumber: pb.MobileNumber,
		CreatedAt:    decodeTime(pb.CreatedAt),
		UpdatedAt:    decodeTime(pb.UpdatedAt),
	}
	return nil
}
