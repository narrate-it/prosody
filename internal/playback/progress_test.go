package playback_test

import (
	"errors"
	"testing"

	"github.com/narrate-it/prosody/internal/playback"
)

func TestSeekAndSkipDoNotCountAsListening(t *testing.T) {
	p, err := playback.New([]int64{10000})
	if err != nil {
		t.Fatal(err)
	}
	p, err = playback.Apply(p, playback.Update{Segment: 0, PositionMS: 10000})
	if err != nil {
		t.Fatal(err)
	}
	if p.Complete() {
		t.Fatal("seek counted as listening")
	}
	p, err = playback.Apply(p, playback.Update{ExpectedRevision: 1, Segment: 0, PositionMS: 10000, Played: []playback.Interval{{StartMS: 9000, EndMS: 10000}}})
	if err != nil {
		t.Fatal(err)
	}
	if p.Complete() {
		t.Fatal("last second counted as full consumption")
	}
}

func TestCoverageMergesWithoutDoubleCounting(t *testing.T) {
	p, err := playback.New([]int64{10000})
	if err != nil {
		t.Fatal(err)
	}
	u := playback.Update{Played: []playback.Interval{{StartMS: 1000, EndMS: 6000}, {StartMS: 5000, EndMS: 10000}}, PositionMS: 10000}
	next, err := playback.Apply(p, u)
	if err != nil {
		t.Fatal(err)
	}
	if !next.Complete() || len(next.Segments[0].Coverage) != 1 {
		t.Fatal("coverage not merged")
	}
	if p.Complete() || len(p.Segments[0].Coverage) != 0 {
		t.Fatal("previous revision mutated")
	}
	if _, err = playback.Apply(next, u); !errors.Is(err, playback.ErrConflict) {
		t.Fatal("stale update accepted")
	}
}

func TestVisualExposureAndPerPersonState(t *testing.T) {
	first, err := playback.New([]int64{0, 100})
	if err != nil {
		t.Fatal(err)
	}
	second := first
	first, err = playback.Apply(first, playback.Update{Segment: 1, Played: []playback.Interval{{StartMS: 0, EndMS: 100}}})
	if err != nil {
		t.Fatal(err)
	}
	if first.Complete() {
		t.Fatal("background-only image counted as viewed")
	}
	first, err = playback.Apply(first, playback.Update{ExpectedRevision: 1, Segment: 0, Viewed: true})
	if err != nil {
		t.Fatal(err)
	}
	if !first.Complete() || second.Complete() {
		t.Fatal("per-person progress not isolated")
	}
}

func TestInvalidIntervals(t *testing.T) {
	for _, tc := range []struct {
		name     string
		interval playback.Interval
	}{{"negative", playback.Interval{StartMS: -1, EndMS: 1}}, {"reversed", playback.Interval{StartMS: 20, EndMS: 10}}, {"outside", playback.Interval{StartMS: 0, EndMS: 101}}} {
		t.Run(tc.name, func(t *testing.T) {
			p, err := playback.New([]int64{100})
			if err != nil {
				t.Fatal(err)
			}
			if _, err = playback.Apply(p, playback.Update{Played: []playback.Interval{tc.interval}}); !errors.Is(err, playback.ErrInvalid) {
				t.Fatal("invalid interval accepted")
			}
		})
	}
}
