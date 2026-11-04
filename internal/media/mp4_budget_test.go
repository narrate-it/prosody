package media

import (
	"bytes"
	"errors"
	"io"
	"math"
	"testing"
)

// mp4BudgetTrack reuses the map fixture with either one contiguous chunk or
// one-byte chunks. Synthetic sample bodies do not establish decodable media.
func mp4BudgetTrack(kind string, samples, chunks uint32, start uint64) mp4TestTrack {
	track := mp4MappedTrack(kind)
	track.count = samples
	track.chunks = make([]mp4TestChunk, chunks)
	perChunk := samples / chunks
	for i := range track.chunks {
		track.chunks[i] = mp4TestChunk{body: 0, offset: start + uint64(i)*uint64(perChunk)}
	}
	track.mapping = []uint32{1, perChunk, 1}
	track.timing = []uint32{samples, 1}
	track.ticks = uint64(samples)
	return track
}

// mp4BudgetProbe checks public profile admission and sample-body read isolation
// for a bounded synthetic fixture without asserting decoder or playback support.
func mp4BudgetProbe(t *testing.T, tracks []mp4TestTrack, payloadBytes int, valid bool) {
	t.Helper()
	data, bodies := mp4SampleFixture(tracks, make([]byte, payloadBytes))
	reader := &mp4GuardReader{data: data, bodies: bodies}
	out, err := ProbeMP4(reader, int64(len(data)), int64(len(data)), 640*360)
	if valid {
		if err != nil || out.MIME != "video/mp4" || out.Width != 640 || out.Height != 360 {
			t.Fatalf("bounded coherent profile rejected: %+v %v", out, err)
		}
	} else if !errors.Is(err, ErrInvalid) || out != (UnverifiedVideoMetadata{}) {
		t.Fatalf("aggregate resource excess returned metadata: %+v %v", out, err)
	}
	if reader.bodyRead || reader.reads > maxMP4MetadataBytes {
		t.Fatal("probe read a sample body or exceeded requested metadata bytes")
	}
}

// TestProbeMP4SharedSampleBudget exercises the public profile across tracks.
// Audio controls retain a one-sample video because ProbeMP4 requires one video.
func TestProbeMP4SharedSampleBudget(t *testing.T) {
	for _, extra := range []uint32{0, 1} {
		name := "exact aggregate limit"
		if extra != 0 {
			name = "one sample above aggregate limit"
		}
		t.Run(name, func(t *testing.T) {
			videoCount := uint32(maxMP4Samples / 2)
			audioCount := videoCount + extra
			video := mp4BudgetTrack("vide", videoCount, 1, 0)
			audio := mp4BudgetTrack("soun", audioCount, 1, uint64(videoCount))
			// Each synthetic track map passes a separate structural check.
			mp4BudgetProbe(t, []mp4TestTrack{video}, int(videoCount), true)
			mp4BudgetProbe(t, []mp4TestTrack{
				mp4BudgetTrack("vide", 1, 1, 0),
				mp4BudgetTrack("soun", audioCount, 1, 1),
			}, int(audioCount)+1, true)
			mp4BudgetProbe(t, []mp4TestTrack{video, audio}, int(videoCount+audioCount), extra == 0)
		})
	}
}

// TestProbeMP4SharedChunkBudget keeps sample counts far below their own cap.
// Every chunk is one byte and both tracks occupy disjoint parts of one mdat.
func TestProbeMP4SharedChunkBudget(t *testing.T) {
	for _, extra := range []uint32{0, 1} {
		name := "exact aggregate limit"
		if extra != 0 {
			name = "one chunk above aggregate limit"
		}
		t.Run(name, func(t *testing.T) {
			videoCount := uint32(maxMP4Chunks / 2)
			audioCount := videoCount + extra
			video := mp4BudgetTrack("vide", videoCount, videoCount, 0)
			audio := mp4BudgetTrack("soun", audioCount, audioCount, uint64(videoCount))
			mp4BudgetProbe(t, []mp4TestTrack{video}, int(videoCount), true)
			mp4BudgetProbe(t, []mp4TestTrack{
				mp4BudgetTrack("vide", 1, 1, 0),
				mp4BudgetTrack("soun", audioCount, audioCount, 1),
			}, int(audioCount)+1, true)
			mp4BudgetProbe(t, []mp4TestTrack{video, audio}, int(videoCount+audioCount), extra == 0)
		})
	}
}

type mp4BudgetCountingReader struct {
	source    io.ReaderAt
	calls     int
	requested int64
}

// ReadAt records every requested byte, including repeated prefixes, before
// forwarding to the fixed fixture source.
func (r *mp4BudgetCountingReader) ReadAt(p []byte, off int64) (int, error) {
	r.calls++
	r.requested += int64(len(p))
	return r.source.ReadAt(p, off)
}

// TestMP4MetadataReaderRequestedBudget is a primitive test, not a 16-MiB public
// metadata profile. It reuses one 64-KiB prefix so unique bytes cannot mask repeated
// requests. Payload and scratch buffers together occupy 128 KiB.
func TestMP4MetadataReaderRequestedBudget(t *testing.T) {
	prefix := bytes.Repeat([]byte{0x5a}, 64<<10)
	underlying := &mp4BudgetCountingReader{source: bytes.NewReader(prefix)}
	reader := &mp4MetadataReader{source: underlying, remaining: maxMP4MetadataBytes}
	scratch := make([]byte, len(prefix))
	for requested := int64(0); requested < maxMP4MetadataBytes; requested += int64(len(scratch)) {
		n, err := reader.ReadAt(scratch, 0)
		if err != nil || n != len(scratch) || !bytes.Equal(scratch, prefix) {
			t.Fatalf("read within exact cumulative budget failed: n=%d err=%v", n, err)
		}
	}
	if reader.remaining != 0 || underlying.requested != maxMP4MetadataBytes {
		t.Fatal("repeated prefix requests were not charged cumulatively")
	}
	calls := underlying.calls
	if n, err := reader.ReadAt(scratch[:1], 0); n != 0 || !errors.Is(err, ErrInvalid) || underlying.calls != calls {
		t.Fatal("exhausted reader called underlying source or accepted more metadata")
	}
}

// TestMP4MetadataBudgetRefusesBeforeRead distinguishes the allocation precheck
// from real parser profiles. Small remaining capacity avoids a huge test buffer.
func TestMP4MetadataBudgetRefusesBeforeRead(t *testing.T) {
	underlying := &mp4BudgetCountingReader{source: bytes.NewReader(make([]byte, 32))}
	reader := &mp4MetadataReader{source: underlying, remaining: 16}
	if n, err := reader.ReadAt(make([]byte, 17), 0); n != 0 || !errors.Is(err, ErrInvalid) || underlying.calls != 0 || reader.remaining != 16 {
		t.Fatal("oversized request reached source or consumed unrequested capacity")
	}
	if data, err := readAtom(reader, atom{body: 0, end: 17}, maxMP4MetadataBytes); data != nil || !errors.Is(err, ErrInvalid) || underlying.calls != 0 {
		t.Fatal("bulk metadata allocation/read ignored remaining capacity")
	}
	if mp4MetadataFits(reader, 17) || mp4MetadataFits(reader, -1) || mp4MetadataFits(underlying, maxMP4MetadataBytes+1) {
		t.Fatal("preallocation predicate accepted invalid metadata size")
	}
	if n, err := reader.ReadAt(make([]byte, 16), 0); n != 16 || err != nil || underlying.calls != 1 || reader.remaining != 0 {
		t.Fatal("exact remaining request was not permitted")
	}
}

// mp4BudgetShortReader deliberately violates ReaderAt's requirement to report an
// error with a short read. The unfilled last byte must never become metadata.
type mp4BudgetShortReader struct {
	source io.ReaderAt
	at     int64
	hits   int
}

// ReadAt returns a short nil-error result only at the selected metadata offset;
// all other reads retain the fixture source's ordinary behavior.
func (r *mp4BudgetShortReader) ReadAt(p []byte, off int64) (int, error) {
	if off == r.at && len(p) > 0 {
		r.hits++
		n, err := r.source.ReadAt(p[:len(p)-1], off)
		if err != nil {
			return n, err
		}
		return n, nil
	}
	return r.source.ReadAt(p, off)
}

// TestMP4MetadataReaderRejectsShortNilError verifies the reader primitive rejects
// malformed ReaderAt results and charges requested rather than returned bytes.
func TestMP4MetadataReaderRejectsShortNilError(t *testing.T) {
	underlying := &mp4BudgetShortReader{source: bytes.NewReader(bytes.Repeat([]byte{0x7f}, 32))}
	reader := &mp4MetadataReader{source: underlying, remaining: 16}
	buffer := make([]byte, 8)
	n, err := reader.ReadAt(buffer, 0)
	if n != 7 || !errors.Is(err, ErrInvalid) || underlying.hits != 1 || reader.remaining != 8 {
		t.Fatal("short nil-error result accepted or charged by actual instead of requested bytes")
	}
}

// TestProbeMP4RejectsShortMetadataWithoutZeroFill checks the public probe does
// not turn an unfilled header byte into usable claimed metadata.
func TestProbeMP4RejectsShortMetadataWithoutZeroFill(t *testing.T) {
	data, bodies := mp4SampleFixture([]mp4TestTrack{mp4MappedTrack("vide")}, []byte{1, 2})
	// Dropping the final movie-duration byte still leaves a positive duration.
	// Ignoring n would therefore admit zero-filled metadata rather than fail.
	body := bytes.Index(data, []byte("mvhd")) + 4
	if body < 4 {
		t.Fatal("missing fixture movie header")
	}
	guard := &mp4GuardReader{data: data, bodies: bodies}
	short := &mp4BudgetShortReader{source: guard, at: int64(body)}
	out, err := ProbeMP4(short, int64(len(data)), int64(len(data)), 640*360)
	if !errors.Is(err, ErrInvalid) || out != (UnverifiedVideoMetadata{}) || short.hits != 1 || guard.bodyRead {
		t.Fatal("short metadata was accepted as zero-filled headers or sample bodies were read")
	}
}

// TestProbeMP4HugeCountsStayBounded uses tiny malformed table declarations;
// neither the fixture nor this test allocates advertised attacker-sized arrays.
func TestProbeMP4HugeCountsStayBounded(t *testing.T) {
	for _, tc := range []struct {
		name  string
		table string
		off   int
	}{
		{"sample sizes", "stsz", 8},
		{"chunk offsets", "stco", 4},
		{"chunk runs", "stsc", 4},
		{"timing runs", "stts", 4},
	} {
		t.Run(tc.name, func(t *testing.T) {
			track := mp4MappedTrack("vide")
			track.edit = mp4EditWord(tc.table, tc.off, math.MaxUint32)
			data, bodies := mp4SampleFixture([]mp4TestTrack{track}, []byte{1, 2})
			guard := &mp4GuardReader{data: data, bodies: bodies}
			out, err := ProbeMP4(guard, int64(len(data)), int64(len(data)), 640*360)
			if !errors.Is(err, ErrInvalid) || out != (UnverifiedVideoMetadata{}) || guard.bodyRead || guard.reads > 4096 {
				t.Fatal("huge count accepted or tiny fixture exceeded bounded metadata requests")
			}
		})
	}
}
