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

func (s *UserService) FindUserByID(ctx context.Context, id int) (*peapod.User, error) {
	panic("TODO")
}

func (s *UserService) FindUserByMobileNumber(ctx context.Context, mobileNumber string) (*peapod.User, error) {
	panic("TODO")
}

func (s *UserService) FindOrCreateUserByMobileNumber(ctx context.Context, mobileNumber string) (*peapod.User, error) {
	panic("TODO: Create user")
	panic("TODO: Create default playlist")
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
