package media

import (
	"bytes"
	"encoding/binary"
	"errors"
	"testing"
	"time"
)

// These synthetic containers exercise metadata structure, not decodable video.
func mp4Box(kind string, body ...[]byte) []byte {
	b := make([]byte, 8)
	copy(b[4:], kind)
	for _, part := range body {
		b = append(b, part...)
	}
	binary.BigEndian.PutUint32(b, uint32(len(b)))
	return b
}

func probeFixture(videoExtras, audioExtras []byte, audio bool) []byte {
	duration := make([]byte, 20)
	binary.BigEndian.PutUint32(duration[12:], 1000)
	binary.BigEndian.PutUint32(duration[16:], 2500)
	track := func(kind string, entry []byte) []byte {
		handler := make([]byte, 12)
		copy(handler[8:], kind)
		stsd := make([]byte, 8)
		binary.BigEndian.PutUint32(stsd[4:], 1)
		return mp4Box("trak", mp4Box("mdia", mp4Box("hdlr", handler),
			mp4Box("mdhd", duration), mp4Box("minf", mp4Box("stbl", mp4Box("stsd", stsd, entry)))))
	}
	video := make([]byte, 78)
	binary.BigEndian.PutUint16(video[24:], 640)
	binary.BigEndian.PutUint16(video[26:], 360)
	tracks := track("vide", mp4Box("avc1", video, mp4Box("avcC", make([]byte, 7)), videoExtras))
	if audio {
		tracks = append(tracks, track("soun", mp4Box("mp4a", make([]byte, 28), audioExtras))...)
	}
	return bytes.Join([][]byte{mp4Box("ftyp", []byte("isom\x00\x00\x00\x00")),
		mp4Box("moov", mp4Box("mvhd", duration), tracks), mp4Box("mdat", []byte{1})}, nil)
}

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

func TestProbeMP4RejectsEveryTruncatedPrefix(t *testing.T) {
	t.Parallel()
	data := probeFixture(nil, nil, true)
	for size := 0; size < len(data); size++ {
		if _, err := ProbeMP4(bytes.NewReader(data[:size]), int64(size), 1<<20, 640*360); !errors.Is(err, ErrInvalid) {
			t.Fatalf("accepted truncated prefix of %d bytes: %v", size, err)
		}
	}
}
