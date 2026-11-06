package media

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/narrate-it/prosody/internal/content"
)

// NarrationMeasurement describes decoded generated audio, not native artifact
// custody or publication authority. Duration is the exact sum rounded down once
// to nanoseconds AFTER comparing that exact sum against the admitted ceiling.
type NarrationMeasurement struct {
	Duration time.Duration
	Parts    []time.Duration
}

// MeasureNarration decodes all generated MP3 parts under one aggregate ceiling.
// The caller selects generated parts from its admitted manifest; supplied media
// must not enter this budget. maxBytes bounds their combined encoded bytes and
// uses the existing generated-speech codec ceiling, not an RFC supplied-media
// limit. Any failure returns no partial measurement. This function neither
// dispatches speech nor retries, repairs, persists or publishes output.
func MeasureNarration(ctx context.Context, parts [][]byte, maxBytes int, maxDuration time.Duration) (NarrationMeasurement, error) {
	if err := ctx.Err(); err != nil {
		return NarrationMeasurement{}, err
	}
	if len(parts) == 0 || len(parts) > 100 || maxBytes <= 0 || maxBytes > 64<<20 || maxDuration <= 0 || maxDuration > content.MaxNarrationDuration {
		return NarrationMeasurement{}, ErrInvalid
	}
	remainingBytes := maxBytes
	for _, part := range parts {
		if len(part) == 0 || len(part) > remainingBytes {
			return NarrationMeasurement{}, ErrInvalid
		}
		remainingBytes -= len(part)
	}
	measurement := NarrationMeasurement{Parts: make([]time.Duration, 0, len(parts))}
	total := new(big.Rat)
	ceiling := big.NewRat(int64(maxDuration), 1)
	for _, part := range parts {
		// Earlier accepted parts have passed the shared ceiling. At most one
		// further duration-limited decode runs before overflow refusal, with
		// decoder frame/look-ahead overhead, not a precise CPU-time bound.
		samples, rate, err := measureMP3Samples(ctx, part, maxBytes, maxDuration)
		if err != nil {
			return NarrationMeasurement{}, fmt.Errorf("measure narration: %w", err)
		}
		elapsed := big.NewRat(samples*int64(time.Second), rate)
		total.Add(total, elapsed)
		if total.Cmp(ceiling) > 0 {
			return NarrationMeasurement{}, ErrDuration
		}
		measurement.Parts = append(measurement.Parts, time.Duration(new(big.Int).Quo(elapsed.Num(), elapsed.Denom()).Int64()))
	}
	if err := ctx.Err(); err != nil {
		return NarrationMeasurement{}, err
	}
	measurement.Duration = time.Duration(new(big.Int).Quo(total.Num(), total.Denom()).Int64())
	return measurement, nil
}
