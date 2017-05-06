package aws

import (
	"errors"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
)

// Session represents a session to AWS.
type Session struct {
	session *session.Session
}

// NewSession returns a session with the given credentials.
func NewSession(accessKeyID, secretAccessKey, region string) (*Session, error) {
	if region == "" {
		return nil, errors.New("aws region required")
	}

	s, err := session.NewSession(&aws.Config{
		Region:      aws.String(region),
		Credentials: credentials.NewStaticCredentials(accessKeyID, secretAccessKey, ""),
	})
	if err != nil {
		return nil, err
	}
	return &Session{session: s}, nil
}
