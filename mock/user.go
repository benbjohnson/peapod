package mock

import (
	"context"

	"github.com/middlemost/peapod"
)

var _ peapod.UserService = &UserService{}

type UserService struct {
	FindUserByIDFn                   func(ctx context.Context, id int) (*peapod.User, error)
	FindUserByMobileNumberFn         func(ctx context.Context, mobileNumber string) (*peapod.User, error)
	FindOrCreateUserByMobileNumberFn func(ctx context.Context, mobileNumber string) (*peapod.User, error)
}

func (s *UserService) FindUserByID(ctx context.Context, id int) (*peapod.User, error) {
	return s.FindUserByIDFn(ctx, id)
}

func (s *UserService) FindUserByMobileNumber(ctx context.Context, mobileNumber string) (*peapod.User, error) {
	return s.FindUserByMobileNumberFn(ctx, mobileNumber)
}

func (s *UserService) FindOrCreateUserByMobileNumber(ctx context.Context, mobileNumber string) (*peapod.User, error) {
	return s.FindOrCreateUserByMobileNumberFn(ctx, mobileNumber)
}
