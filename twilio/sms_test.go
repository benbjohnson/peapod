// +build integration

package twilio_test

import (
	"bytes"
	"context"
	"flag"
	"testing"

	"github.com/middlemost/peapod"
	"github.com/middlemost/peapod/twilio"
)

var (
	accountSID = flag.String("account-sid", "", "Account SID")
	authToken  = flag.String("auth-token", "", "Auth Token")
	from       = flag.String("from", "", "From")
	to         = flag.String("to", "", "To")
)

// Ensure service can send an SMS over Twilio.
func TestSMSService_SendSMS(t *testing.T) {
	if *accountSID == "" {
		t.Fatal("account sid required")
	} else if *authToken == "" {
		t.Fatal("auth token required")
	} else if *from == "" {
		t.Fatal("from required")
	} else if *to == "" {
		t.Fatal("to required")
	}

	// Initialize service.
	var buf bytes.Buffer
	s := twilio.NewSMSService()
	s.AccountSID = *accountSID
	s.AuthToken = *authToken
	s.From = *from
	s.LogOutput = &buf

	// Send text.
	sms := &peapod.SMS{To: *to, Body: "TEST"}
	if err := s.SendSMS(context.Background(), sms); err != nil {
		t.Fatal(err)
	}

	// Verify message id is set.
	if sms.ID == "" {
		t.Fatal("expected message sid")
	}

	// Show log.
	t.Log(buf.String())
}
