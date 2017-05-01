package peapod

import (
	"context"
	"time"
)

// User errors.
const (
	ErrUserRequired = Error("user required")
	ErrUserNotFound = Error("user not found")
)

// User represents a user in the system.
type User struct {
	ID           int       `json:"id"`
	MobileNumber string    `json:"mobile_number,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// UserService represents a service for managing users.
type UserService interface {
	FindUserByID(ctx context.Context, id int) (*User, error)
	FindUserByMobileNumber(ctx context.Context, mobileNumber string) (*User, error)
	FindOrCreateUserByMobileNumber(ctx context.Context, mobileNumber string) (*User, error)
	UserPlaylists(ctx context.Context, id int) ([]*Playlist, error)
}
