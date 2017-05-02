package mock

import (
	"context"

	"github.com/middlemost/peapod"
)

var _ peapod.SMSService = &SMSService{}

type SMSService struct {
	SendSMSFn func(ctx context.Context, msg *peapod.SMS) error
}

func (s *SMSService) SendSMS(ctx context.Context, msg *peapod.SMS) error {
	return s.SendSMSFn(ctx, msg)
}
