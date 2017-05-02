package twilio

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"

	"github.com/middlemost/peapod"
	"github.com/subosito/twilio"
)

// Ensure service implements interface.
var _ peapod.SMSService = &SMSService{}

// SMSService represents a service for sending SMS text messages over Twilio.
type SMSService struct {
	// API settings.
	AccountSID string
	AuthToken  string

	// Sender phone number.
	From string

	LogOutput io.Writer
}

// NewSMSService returns a new instance of SMSService.
func NewSMSService() *SMSService {
	return &SMSService{LogOutput: ioutil.Discard}
}

// SendSMS sends an SMS message.
func (s *SMSService) SendSMS(ctx context.Context, msg *peapod.SMS) error {
	client := twilio.NewClient(s.AccountSID, s.AuthToken, nil)

	// Send message.
	ret, _, err := client.Messages.SendSMS(s.From, msg.To, msg.Body)
	if err != nil {
		return err
	}
	msg.ID = ret.Sid

	// Log returned message.
	buf, _ := json.Marshal(ret)
	fmt.Fprintf(s.LogOutput, "twilio: send: %s\n", buf)

	return nil
}
