package peapod

import (
	"context"
	"time"
)

// User represents a user in the system.
type User struct {
	ID           int       `json:"id"`
	MobileNumber string    `json:"mobile_number,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// UserUpdate represents an update to a user.
type UserUpdate struct {
	MobileNumber *string `json:"mobile_number"`
}

// UserService represents a service for managing users.
type UserService interface {
	FindUserByID(ctx context.Context, id int) (*User, error)
	FindUserByMobileNumber(ctx context.Context, mobileNumber string) (*User, error)
	CreateUser(ctx context.Context, u *User) error
	UpdateUser(ctx context.Context, id int, upd UserUpdate) error
}
