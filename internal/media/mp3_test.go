package media

import (
	"encoding/binary"
	"testing"
)

func TestFrameBoundaryAndSampleRate(t *testing.T) {
	// MPEG1 LayerIII, 128 kb/s, 44.1 kHz: 417-byte frame. Structural checks do not
	// claim these synthetic bytes are valid decoded audio.
	frame := make([]byte, 417)
	binary.BigEndian.PutUint32(frame, 0xfffb9000)
	if rate, samples, err := mp3Frames(frame); err != nil || rate != 44100 || samples != 1152 {
		t.Fatalf("rate=%d samples=%d error=%v", rate, samples, err)
	}
	if _, _, err := mp3Frames(frame[:416]); err == nil {
		t.Fatal("accepted incomplete frame")
	}
	second := make([]byte, 384)
	binary.BigEndian.PutUint32(second, 0xfffb9400)
	if _, _, err := mp3Frames(append(frame, second...)); err == nil {
		t.Fatal("accepted changing sample rate")
	}
}
