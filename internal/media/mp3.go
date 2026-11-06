package media

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	mp3 "github.com/hajimehoshi/go-mp3"
)

var ErrDuration = errors.New("audio exceeds admitted duration")

// MeasureMP3 decodes the entire bounded recording to count PCM samples. It does
// not trust MIME, bitrate estimates, ID3 duration fields or an agent's metadata.
// This adapter only admits the decoder's supported MP3 profiles. Provider and
// malformed-frame qualification are required before publishing external audio.
func MeasureMP3(ctx context.Context, b []byte, maxBytes int, maxDuration time.Duration) (time.Duration, error) {
	samples, rate, err := measureMP3Samples(ctx, b, maxBytes, maxDuration)
	if err != nil {
		return 0, err
	}
	return time.Duration(samples * int64(time.Second) / rate), nil
}

// measureMP3Samples retains exact validated timing for aggregate comparisons.
func measureMP3Samples(ctx context.Context, b []byte, maxBytes int, maxDuration time.Duration) (int64, int64, error) {
	if len(b) == 0 || maxBytes <= 0 || len(b) > maxBytes || maxDuration <= 0 || maxDuration > 24*time.Hour {
		return 0, 0, ErrInvalid
	}
	expectedRate, expectedSamples, err := mp3Frames(b)
	if err != nil {
		return 0, 0, err
	}
	decoder, err := mp3.NewDecoder(bytes.NewReader(b))
	if err != nil {
		return 0, 0, ErrInvalid
	}
	rate := int64(decoder.SampleRate())
	if rate <= 0 || rate > 192000 || rate != int64(expectedRate) {
		return 0, 0, ErrInvalid
	}
	// The decoder emits signed 16-bit stereo even for mono source: 4 bytes/sample.
	limit := int64(maxDuration/time.Second)*rate*4 + int64(maxDuration%time.Second)*rate*4/int64(time.Second)
	n, err := io.Copy(io.Discard, io.LimitReader(contextReader{ctx, decoder}, limit+4))
	if err != nil {
		return 0, 0, fmt.Errorf("decode audio duration: %w", err)
	}
	if n > limit {
		return 0, 0, ErrDuration
	}
	if n == 0 || n%4 != 0 || n/4 != expectedSamples {
		return 0, 0, ErrInvalid
	}
	return n / 4, rate, nil
}
