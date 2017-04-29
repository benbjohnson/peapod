package bolt

import (
	"context"

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
	panic("TODO")
}

func (s *UserService) CreateUser(ctx context.Context, u *peapod.User) error {
	panic("TODO")
}

func (s *UserService) UpdateUser(ctx context.Context, id int, upd peapod.UserUpdate) error {
	panic("TODO")
}
