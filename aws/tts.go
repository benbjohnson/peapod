package aws

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/polly"
	"github.com/middlemost/peapod"
	"golang.org/x/sync/errgroup"
)

// MaxCharactersPerRequest is the maximum number of characters allowed by Polly.
const MaxCharactersPerRequest = 1500

// DefaultVoiceID is the default voice to use when synthesizing speech.
const DefaultVoiceID = "Emma"

// Ensure service implements interface.
var _ peapod.TTSService = &TTSService{}

// TTSService represents a service for performing text-to-speech.
type TTSService struct {
	Session   *Session
	VoiceID   string
	LogOutput io.Writer
}

// NewTTSService returns a new instance of TTSService.
func NewTTSService() *TTSService {
	return &TTSService{
		VoiceID:   DefaultVoiceID,
		LogOutput: ioutil.Discard,
	}
}

// SynthesizeSpeech encodes text to speech.
func (s *TTSService) SynthesizeSpeech(ctx context.Context, text string) (io.ReadCloser, error) {
	// Split into chunks.
	chunks := splitTextOnParagraphs(text, MaxCharactersPerRequest)

	// Synthesize chunks in parallel.
	paths := make([]string, len(chunks))
	var wg errgroup.Group
	for i, chunk := range chunks {
		i, chunk := i, chunk
		fmt.Fprintf(s.LogOutput, "tts: synthesizing chunk: index=%d, len=%d\n", i, len(chunk))

		wg.Go(func() error {
			path, err := s.synthesizeChunk(ctx, i, chunk)
			paths[i] = path
			return err
		})
	}

	// Wait for the chunks to complete.
	if err := wg.Wait(); err != nil {
		return nil, err
	}

	// Combine chunks.
	combinedPath, err := s.concatenateFiles(ctx, paths)
	if err != nil {
		return nil, err
	}

	// Open file handle to return for reading.
	file, err := os.Open(combinedPath)
	if err != nil {
		return nil, err
	}
	return &oneTimeReader{File: file}, nil
}

// synthesizeChunk synthesizes a single chunk of text to a temp file.
// Returns a path to the temporary file.
func (s *TTSService) synthesizeChunk(ctx context.Context, index int, text string) (string, error) {
	svc := polly.New(s.Session.session)

	resp, err := svc.SynthesizeSpeech(&polly.SynthesizeSpeechInput{
		OutputFormat: aws.String("mp3"),
		VoiceId:      aws.String(s.VoiceID),
		Text:         aws.String(text),
	})
	if resp != nil {
		fmt.Fprintf(s.LogOutput, "tts: response: chars=%d\n", resp.RequestCharacters)
	}
	if err != nil {
		return "", err
	}
	defer resp.AudioStream.Close()

	// Write audio to a temporary file.
	f, err := ioutil.TempFile("", "peapod-polly-chunk-")
	if err != nil {
		return "", err
	} else if _, err := io.Copy(f, resp.AudioStream); err != nil {
		return "", err
	} else if err := f.Close(); err != nil {
		return "", err
	}

	// Rename with extension.
	path := f.Name() + ".mp3"
	if err := os.Rename(f.Name(), path); err != nil {
		return "", err
	}
	return path, nil
}

func (s *TTSService) concatenateFiles(ctx context.Context, paths []string) (string, error) {
	// Create a temporary path.
	f, err := ioutil.TempFile("", "peapod-polly-")
	if err != nil {
		return "", err
	} else if err := f.Close(); err != nil {
		return "", err
	} else if err := os.Remove(f.Name()); err != nil {
		return "", err
	}
	path := f.Name() + ".mp3"

	// Execute command.
	args := []string{"-i", "concat:" + strings.Join(paths, "|"), "-c", "copy", path}
	cmd := exec.Command("ffmpeg", args...)
	cmd.Stdout = s.LogOutput
	cmd.Stderr = s.LogOutput
	if err := cmd.Run(); err != nil {
		return "", err
	}

	return path, nil
}

// oneTimeReader allows the reader to read once and then it deletes on close.
type oneTimeReader struct {
	*os.File
}

// Close closes the file handle and deletes the file.
func (r *oneTimeReader) Close() error {
	if err := r.File.Close(); err != nil {
		return err
	}
	return os.Remove(r.File.Name())
}

// splitTextOnParagraphs splits into chunks of maxChars-length chunks.
func splitTextOnParagraphs(text string, maxChars int) []string {
	lines := regexp.MustCompile(`\n+`).Split(text, -1)

	var chunks []string
	for _, line := range lines {
		line += "\n"

		// If line is too large for one chunk then split on words.
		if len(line) > maxChars {
			chunks = append(chunks, splitTextOnWords(line, maxChars)...)
			continue
		}

		// Add if this is the first line.
		if len(chunks) == 0 {
			chunks = append(chunks, line)
			continue
		}

		// Add new chunk if adding line will exceed max.
		if len(chunks[len(chunks)-1])+len(line) > maxChars {
			chunks = append(chunks, line)
			continue
		}

		// Append to last chunk.
		chunks[len(chunks)-1] = chunks[len(chunks)-1] + "\n" + line
	}

	return chunks
}

// splitTextOnWords splits into max length chunks at word boundries.
func splitTextOnWords(text string, maxChars int) []string {
	words := regexp.MustCompile(` +`).Split(text, -1)

	chunks := make([]string, 1)
	chunks[0] = words[0]
	for _, word := range words[1:] {
		if len(chunks[len(chunks)-1])+len(word) > maxChars {
			chunks = append(chunks, word)
			continue
		}

		chunks[len(chunks)-1] = chunks[len(chunks)-1] + " " + word
	}

	return chunks
}
