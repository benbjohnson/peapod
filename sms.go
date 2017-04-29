package peapod

import (
	"context"
)

// SMS represents a text message.
type SMS struct {
	ID   string
	To   string
	Body string
}

// SMSService sends a text message to a recipient.
type SMSService interface {
	SendSMS(ctx context.Context, msg *SMS) error
}
