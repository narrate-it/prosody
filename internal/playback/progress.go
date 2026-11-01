// Package playback models consumption of one immutable rendition by one person.
// The persistence boundary must bind person, workspace, item and rendition from
// current authenticated authority; these pure functions do not authorize access.
package playback

import (
	"errors"
	"sort"
)

var (
	ErrInvalid  = errors.New("invalid playback progress")
	ErrConflict = errors.New("playback revision conflict")
)

const MaxIntervals = 4096

type Interval struct {
	StartMS int64 `json:"startMs"`
	EndMS   int64 `json:"endMs"`
}

type Segment struct {
	// Zero duration identifies visual-only content.
	DurationMS int64      `json:"durationMs"`
	Coverage   []Interval `json:"coverage"`
	Viewed     bool       `json:"viewed"`
}

type Progress struct {
	Revision        uint64    `json:"revision"`
	Segments        []Segment `json:"segments"`
	PositionSegment int       `json:"positionSegment"`
	PositionMS      int64     `json:"positionMs"`
	MarkedRead      bool      `json:"markedRead"`
}

type Update struct {
	ExpectedRevision uint64 `json:"expectedRevision"`
	Segment          int    `json:"segment"`
	PositionMS       int64  `json:"positionMs"`
	// Played reports intervals actually played, never the seek distance.
	Played     []Interval `json:"played"`
	Viewed     bool       `json:"viewed"`
	MarkedRead *bool      `json:"markedRead,omitempty"`
}

func New(durationsMS []int64) (Progress, error) {
	if len(durationsMS) == 0 || len(durationsMS) > 100 {
		return Progress{}, ErrInvalid
	}
	p := Progress{Segments: make([]Segment, len(durationsMS))}
	for i, d := range durationsMS {
		if d < 0 {
			return Progress{}, ErrInvalid
		}
		p.Segments[i].DurationMS = d
	}
	return p, nil
}

func normalize(intervals []Interval, duration int64) ([]Interval, error) {
	if duration < 0 || len(intervals) > MaxIntervals+256 {
		return nil, ErrInvalid
	}
	ordered := append([]Interval(nil), intervals...)
	for _, v := range ordered {
		if v.StartMS < 0 || v.EndMS <= v.StartMS || v.EndMS > duration {
			return nil, ErrInvalid
		}
	}
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].StartMS < ordered[j].StartMS })
	merged := make([]Interval, 0, len(ordered))
	for _, v := range ordered {
		if len(merged) == 0 || v.StartMS > merged[len(merged)-1].EndMS {
			merged = append(merged, v)
			continue
		}
		if v.EndMS > merged[len(merged)-1].EndMS {
			merged[len(merged)-1].EndMS = v.EndMS
		}
	}
	if len(merged) > MaxIntervals {
		return nil, ErrInvalid
	}
	return merged, nil
}

// Apply returns a new value and never mutates the previous revision, including
// on failure. The caller must persist with a compare-and-swap on Revision.
func Apply(current Progress, u Update) (Progress, error) {
	if u.ExpectedRevision != current.Revision {
		return Progress{}, ErrConflict
	}
	if current.Revision == ^uint64(0) || len(current.Segments) == 0 || u.Segment < 0 || u.Segment >= len(current.Segments) || len(u.Played) > 256 {
		return Progress{}, ErrInvalid
	}
	duration := current.Segments[u.Segment].DurationMS
	if u.PositionMS < 0 || u.PositionMS > duration {
		return Progress{}, ErrInvalid
	}
	next := current
	next.Segments = make([]Segment, len(current.Segments))
	for i, s := range current.Segments {
		coverage, err := normalize(s.Coverage, s.DurationMS)
		if err != nil {
			return Progress{}, err
		}
		s.Coverage = coverage
		next.Segments[i] = s
	}
	s := &next.Segments[u.Segment]
	coverage, err := normalize(append(s.Coverage, u.Played...), duration)
	if err != nil {
		return Progress{}, err
	}
	s.Coverage = coverage
	s.Viewed = s.Viewed || u.Viewed
	next.PositionSegment = u.Segment
	next.PositionMS = u.PositionMS
	if u.MarkedRead != nil {
		next.MarkedRead = *u.MarkedRead
	}
	next.Revision++
	return next, nil
}

// Complete requires actual coverage of at least 90% and playback reaching the
// end of every timed segment. Visual-only segments need a foreground view.
// Explicit mark-read is a separate user action that overrides consumption.
func (p Progress) Complete() bool {
	if len(p.Segments) == 0 {
		return false
	}
	if p.MarkedRead {
		return true
	}
	for _, s := range p.Segments {
		if s.DurationMS == 0 {
			if !s.Viewed {
				return false
			}
			continue
		}
		coverage, err := normalize(s.Coverage, s.DurationMS)
		if err != nil || len(coverage) == 0 {
			return false
		}
		var played int64
		for _, v := range coverage {
			played += v.EndMS - v.StartMS
		}
		if played < s.DurationMS-s.DurationMS/10 || coverage[len(coverage)-1].EndMS != s.DurationMS {
			return false
		}
	}
	return true
}
