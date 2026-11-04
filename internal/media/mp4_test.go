package media

import (
	"bytes"
	"encoding/binary"
	"errors"
	"testing"
	"time"
)

// mp4Box builds synthetic container atoms, not decodable video.
func mp4Box(kind string, body ...[]byte) []byte {
	b := make([]byte, 8)
	copy(b[4:], kind)
	for _, part := range body {
		b = append(b, part...)
	}
	binary.BigEndian.PutUint32(b, uint32(len(b)))
	return b
}

// probeFixture builds structurally mapped synthetic video/audio, not codecs.
func probeFixture(videoExtras, audioExtras []byte, audio bool) []byte {
	return probeAudioFixture(videoExtras, audioExtras, audio, make([]byte, 28))
}

// probeAudioFixture keeps legacy codec/version negatives behind valid maps and
// self-contained references, so those tests continue to reach their target.
func probeAudioFixture(videoExtras, audioExtras []byte, audio bool, audioHeader []byte, videoConfigs ...[]byte) []byte {
	config := mp4Box("avcC", make([]byte, 7))
	if len(videoConfigs) > 0 {
		config = bytes.Join(videoConfigs, nil)
	}
	duration := mp4Words(0, 0, 0, 1000, 2500)
	track := func(kind string, entry []byte, offset uint32) []byte {
		handler := make([]byte, 12)
		copy(handler[8:], kind)
		stbl := mp4Box("stbl", mp4Box("stsd", mp4Words(0, 1), entry),
			mp4Box("stsz", mp4Words(0, 1, 1)), mp4Box("stsc", mp4Words(0, 1, 1, 1, 1)),
			mp4Box("stts", mp4Words(0, 1, 1, 2500)), mp4Box("stco", mp4Words(0, 1, offset)))
		return mp4Box("trak", mp4Box("mdia", mp4Box("hdlr", handler),
			mp4Box("mdhd", duration), mp4Box("minf", mp4SelfReference(), stbl)))
	}
	video := make([]byte, 78)
	binary.BigEndian.PutUint16(video[6:], 1)
	binary.BigEndian.PutUint16(video[24:], 640)
	binary.BigEndian.PutUint16(video[26:], 360)
	audioHeader = append([]byte(nil), audioHeader...)
	if len(audioHeader) >= 8 {
		binary.BigEndian.PutUint16(audioHeader[6:], 1)
	}
	data := []byte{1}
	if audio {
		data = append(data, 2)
	}
	build := func(videoOffset, audioOffset uint32) []byte {
		tracks := track("vide", mp4Box("avc1", video, config, videoExtras), videoOffset)
		if audio {
			tracks = append(tracks, track("soun", mp4Box("mp4a", audioHeader, audioExtras), audioOffset)...)
		}
		return bytes.Join([][]byte{mp4Box("ftyp", []byte("isom\x00\x00\x00\x00")),
			mp4Box("moov", mp4Box("mvhd", duration), tracks), mp4Box("mdat", data)}, nil)
	}
	initial := build(0, 0)
	body := uint32(len(initial) - len(data))
	return build(body, body+1)
}

// TestProbeMP4SampleEntryProtection keeps protected/duplicate codec entries denied.
func TestProbeMP4SampleEntryProtection(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name         string
		video, audio []byte
		wantError    bool
	}{
		{"clear", nil, nil, false},
		{"clear audio child", nil, mp4Box("esds", make([]byte, 4)), false},
		{"video protection", mp4Box("sinf"), nil, true},
		{"audio protection", nil, mp4Box("sinf"), true},
		{"duplicate codec config", mp4Box("avcC", make([]byte, 7)), nil, true},
		{"truncated audio child", nil, []byte{1, 2, 3}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			data := probeFixture(tc.video, tc.audio, true)
			info, err := ProbeMP4(bytes.NewReader(data), int64(len(data)), 1<<20, 640*360)
			if tc.wantError {
				if !errors.Is(err, ErrInvalid) {
					t.Fatalf("expected invalid metadata, got %+v, %v", info, err)
				}
				return
			}
			if err != nil || info.Duration != 2500*time.Millisecond || info.Width != 640 || info.Height != 360 {
				t.Fatalf("unexpected metadata: %+v, %v", info, err)
			}
		})
	}
}

// TestProbeMP4RejectsEveryTruncatedPrefix requires complete declared metadata.
func TestProbeMP4RejectsEveryTruncatedPrefix(t *testing.T) {
	t.Parallel()
	data := probeFixture(nil, nil, true)
	for size := 0; size < len(data); size++ {
		if _, err := ProbeMP4(bytes.NewReader(data[:size]), int64(size), 1<<20, 640*360); !errors.Is(err, ErrInvalid) {
			t.Fatalf("accepted truncated prefix of %d bytes: %v", size, err)
		}
	}
}

// TestProbeMP4AudioVersions preserves supported audio-header/protection boundaries.
func TestProbeMP4AudioVersions(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name       string
		version    uint16
		headerSize int
		samples    uint32
		child      []byte
		invalid    bool
	}{
		{"clear version 1", 1, 44, 1024, mp4Box("esds", make([]byte, 4)), false},
		{"protected version 1 zero samples", 1, 44, 0, mp4Box("sinf"), true},
		{"protected version 1 nonzero samples", 1, 44, 1024, mp4Box("sinf"), true},
		{"short version 1", 1, 28, 0, nil, true},
		{"truncated extension", 1, 43, 0, nil, true},
		{"unknown version", 3, 28, 0, nil, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			header := make([]byte, tc.headerSize)
			binary.BigEndian.PutUint16(header[8:], tc.version)
			if len(header) >= 32 {
				binary.BigEndian.PutUint32(header[28:], tc.samples)
			}
			data := probeAudioFixture(nil, tc.child, true, header)
			info, err := ProbeMP4(bytes.NewReader(data), int64(len(data)), 1<<20, 640*360)
			if tc.invalid {
				if !errors.Is(err, ErrInvalid) {
					t.Fatalf("expected rejection, got %+v, %v", info, err)
				}
			} else if err != nil || info.Duration != 2500*time.Millisecond {
				t.Fatalf("clear version 1 rejected: %+v, %v", info, err)
			}
		})
	}
}

// TestProbeMP4UndersizedAVCConfig denies short codec records, even beside a valid one.
func TestProbeMP4UndersizedAVCConfig(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name    string
		configs [][]byte
	}{
		{"sole undersized", [][]byte{mp4Box("avcC", make([]byte, 6))}},
		{"undersized then valid", [][]byte{mp4Box("avcC", make([]byte, 6)), mp4Box("avcC", make([]byte, 7))}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			data := probeAudioFixture(nil, nil, false, nil, tc.configs...)
			if _, err := ProbeMP4(bytes.NewReader(data), int64(len(data)), 1<<20, 640*360); !errors.Is(err, ErrInvalid) {
				t.Fatalf("expected invalid AVC configuration, got %v", err)
			}
		})
	}
}
