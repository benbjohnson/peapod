package peapod

import (
	"context"
	"io"
)

type TTSService interface {
	SynthesizeSpeech(ctx context.Context, text string) (io.ReadCloser, error)
}
