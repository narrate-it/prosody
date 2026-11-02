package media

import (
	"encoding/binary"
	"io"
	"math"
	"time"
)

type UnverifiedVideoMetadata struct {
	MIME          string
	Duration      time.Duration
	Width, Height int
}
type atom struct {
	kind      string
	body, end int64
}

// ProbeMP4 reads metadata from a bounded non-fragmented MP4 container profile.
// It reads only small metadata atoms, never mdat into memory. This is structural
// inspection only: sample maps, timelines and codec configuration are not yet
// qualified. Never use this result as publication admission or a measured duration.
// Dimensions are header claims, not a bound on decoded codec dimensions.
// Original bytes remain unchanged. Fragmented/encrypted/other codecs fail closed.
func ProbeMP4(r io.ReaderAt, size, maxBytes int64, maxPixels int64) (UnverifiedVideoMetadata, error) {
	if r == nil || size < 16 || maxBytes <= 0 || size > maxBytes || maxPixels <= 0 {
		return UnverifiedVideoMetadata{}, ErrInvalid
	}
	budget := 4096
	top, err := atoms(r, 0, size, &budget)
	if err != nil {
		return UnverifiedVideoMetadata{}, err
	}
	var movie *atom
	hasType, hasData := false, false
	for _, a := range top {
		switch a.kind {
		case "ftyp":
			if hasType || a.end-a.body < 8 || a.end-a.body > 4096 {
				return UnverifiedVideoMetadata{}, ErrInvalid
			}
			hasType = true
		case "mdat":
			hasData = hasData || a.end > a.body
		case "moov":
			if movie != nil {
				return UnverifiedVideoMetadata{}, ErrInvalid
			}
			copy := a
			movie = &copy
		case "moof":
			return UnverifiedVideoMetadata{}, ErrInvalid
		}
	}
	if !hasType || !hasData || movie == nil {
		return UnverifiedVideoMetadata{}, ErrInvalid
	}
	children, err := atoms(r, movie.body, movie.end, &budget)
	if err != nil {
		return UnverifiedVideoMetadata{}, err
	}
	info := UnverifiedVideoMetadata{MIME: "video/mp4"}
	videos, tracks := 0, 0
	for _, a := range children {
		switch a.kind {
		case "mvex":
			return UnverifiedVideoMetadata{}, ErrInvalid
		case "mvhd":
			if info.Duration != 0 {
				return UnverifiedVideoMetadata{}, ErrInvalid
			}
			info.Duration, err = atomDuration(r, a)
			if err != nil {
				return UnverifiedVideoMetadata{}, err
			}
		case "trak":
			tracks++
			if tracks > 8 {
				return UnverifiedVideoMetadata{}, ErrInvalid
			}
			kind, width, height, err := inspectTrack(r, a, &budget, maxPixels)
			if err != nil {
				return UnverifiedVideoMetadata{}, err
			}
			if kind == "vide" {
				videos++
				info.Width, info.Height = width, height
			}
		}
	}
	if videos != 1 || info.Duration <= 0 {
		return UnverifiedVideoMetadata{}, ErrInvalid
	}
	return info, nil
}
func atoms(r io.ReaderAt, start, end int64, budget *int) ([]atom, error) {
	result := []atom{}
	for start < end {
		if *budget <= 0 || end-start < 8 {
			return nil, ErrInvalid
		}
		*budget--
		var header [16]byte
		if _, err := r.ReadAt(header[:8], start); err != nil {
			return nil, ErrInvalid
		}
		n := uint64(binary.BigEndian.Uint32(header[:4]))
		headerSize := int64(8)
		if n == 1 {
			if end-start < 16 {
				return nil, ErrInvalid
			}
			if _, err := r.ReadAt(header[8:], start+8); err != nil {
				return nil, ErrInvalid
			}
			n = binary.BigEndian.Uint64(header[8:])
			headerSize = 16
		}
		if n == 0 {
			n = uint64(end - start)
		}
		if n < uint64(headerSize) || n > uint64(end-start) {
			return nil, ErrInvalid
		}
		result = append(result, atom{string(header[4:8]), start + headerSize, start + int64(n)})
		start += int64(n)
	}
	return result, nil
}
func readAtom(r io.ReaderAt, a atom, limit int64) ([]byte, error) {
	if a.end-a.body < 0 || a.end-a.body > limit {
		return nil, ErrInvalid
	}
	b := make([]byte, a.end-a.body)
	if _, err := r.ReadAt(b, a.body); err != nil {
		return nil, ErrInvalid
	}
	return b, nil
}
func atomDuration(r io.ReaderAt, a atom) (time.Duration, error) {
	// Header atoms contain additional fixed fields after the time values. Read a
	// bounded prefix rather than trusting an attacker-controlled metadata size.
	n := a.end - a.body
	if n < 20 {
		return 0, ErrInvalid
	}
	if n > 32 {
		n = 32
	}
	b := make([]byte, n)
	if _, err := r.ReadAt(b, a.body); err != nil {
		return 0, ErrInvalid
	}
	var scale uint32
	var duration uint64
	switch b[0] {
	case 0:
		scale = binary.BigEndian.Uint32(b[12:16])
		duration = uint64(binary.BigEndian.Uint32(b[16:20]))
	case 1:
		if len(b) < 32 {
			return 0, ErrInvalid
		}
		scale = binary.BigEndian.Uint32(b[20:24])
		duration = binary.BigEndian.Uint64(b[24:32])
	default:
		return 0, ErrInvalid
	}
	if scale == 0 || duration == 0 {
		return 0, ErrInvalid
	}
	seconds, remainder := duration/uint64(scale), duration%uint64(scale)
	if seconds > math.MaxInt64/uint64(time.Second) {
		return 0, ErrInvalid
	}
	nanos := seconds * uint64(time.Second)
	fraction := remainder * uint64(time.Second) / uint64(scale)
	if fraction > math.MaxInt64-nanos {
		return 0, ErrInvalid
	}
	return time.Duration(nanos + fraction), nil
}
func inspectTrack(r io.ReaderAt, track atom, budget *int, maxPixels int64) (string, int, int, error) {
	children, err := atoms(r, track.body, track.end, budget)
	if err != nil {
		return "", 0, 0, err
	}
	var media *atom
	for _, a := range children {
		if a.kind == "mdia" {
			if media != nil {
				return "", 0, 0, ErrInvalid
			}
			copy := a
			media = &copy
		}
	}
	if media == nil {
		return "", 0, 0, ErrInvalid
	}
	children, err = atoms(r, media.body, media.end, budget)
	if err != nil {
		return "", 0, 0, err
	}
	kind := ""
	var descriptions []atom
	hasDuration := false
	for _, a := range children {
		switch a.kind {
		case "hdlr":
			b, err := readAtom(r, a, 4096)
			if err != nil || len(b) < 12 || kind != "" {
				return "", 0, 0, ErrInvalid
			}
			kind = string(b[8:12])
		case "mdhd":
			if hasDuration {
				return "", 0, 0, ErrInvalid
			}
			if _, err := atomDuration(r, a); err != nil {
				return "", 0, 0, err
			}
			hasDuration = true
		case "minf":
			minf, err := atoms(r, a.body, a.end, budget)
			if err != nil {
				return "", 0, 0, err
			}
			for _, m := range minf {
				if m.kind == "stbl" {
					stbl, err := atoms(r, m.body, m.end, budget)
					if err != nil {
						return "", 0, 0, err
					}
					for _, s := range stbl {
						if s.kind == "stsd" {
							descriptions = append(descriptions, s)
						}
					}
				}
			}
		}
	}
	if !hasDuration || len(descriptions) != 1 || kind != "vide" && kind != "soun" {
		return "", 0, 0, ErrInvalid
	}
	stsd := descriptions[0]
	if stsd.end-stsd.body < 8 {
		return "", 0, 0, ErrInvalid
	}
	var h [8]byte
	if _, err := r.ReadAt(h[:], stsd.body); err != nil {
		return "", 0, 0, ErrInvalid
	}
	if h[0] != 0 || binary.BigEndian.Uint32(h[4:]) != 1 {
		return "", 0, 0, ErrInvalid
	}
	entries, err := atoms(r, stsd.body+8, stsd.end, budget)
	if err != nil || len(entries) != 1 {
		return "", 0, 0, ErrInvalid
	}
	entry := entries[0]
	if kind == "soun" {
		if entry.kind != "mp4a" || entry.end-entry.body < 28 {
			return "", 0, 0, ErrInvalid
		}
		config, err := atoms(r, entry.body+28, entry.end, budget)
		if err != nil {
			return "", 0, 0, err
		}
		for _, a := range config {
			if a.kind == "sinf" {
				return "", 0, 0, ErrInvalid
			}
		}
		return kind, 0, 0, nil
	}
	if entry.kind != "avc1" || entry.end-entry.body < 78 {
		return "", 0, 0, ErrInvalid
	}
	var dimensions [4]byte
	if _, err := r.ReadAt(dimensions[:], entry.body+24); err != nil {
		return "", 0, 0, ErrInvalid
	}
	width, height := int(binary.BigEndian.Uint16(dimensions[:2])), int(binary.BigEndian.Uint16(dimensions[2:]))
	if width <= 0 || height <= 0 || int64(width) > maxPixels/int64(height) {
		return "", 0, 0, ErrInvalid
	}
	config, err := atoms(r, entry.body+78, entry.end, budget)
	if err != nil {
		return "", 0, 0, err
	}
	avc := false
	for _, a := range config {
		if a.kind == "sinf" {
			return "", 0, 0, ErrInvalid
		}
		if a.kind == "avcC" {
			if avc || a.end-a.body < 7 {
				return "", 0, 0, ErrInvalid
			}
			avc = true
		}
	}
	if !avc {
		return "", 0, 0, ErrInvalid
	}
	return kind, width, height, nil
}
