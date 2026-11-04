package media

import (
	"encoding/binary"
	"io"
	"math"
	"sort"
	"time"
)

const (
	maxMP4Samples       = uint64(1_000_000)
	maxMP4Chunks        = uint64(100_000)
	maxMP4MetadataBytes = int64(16 << 20)
)

type mp4MediaTime struct {
	ticks    uint64
	duration time.Duration
}
type mp4ByteRange struct{ start, end uint64 }
type mp4SampleBudget struct {
	samples, chunks uint64
	ranges          []mp4ByteRange
}
type mp4MapAtoms struct{ stsd, stsz, stsc, stts, offsets atom }

// mp4MetadataReader bounds all requested metadata reads, including headers and
// repeated prefixes. It never changes the underlying ReaderAt's blocking rules.
type mp4MetadataReader struct {
	source    io.ReaderAt
	remaining int64
}

// ReadAt rejects exhausted budgets and short reads before callers use metadata.
func (r *mp4MetadataReader) ReadAt(p []byte, off int64) (int, error) {
	if off < 0 || int64(len(p)) > r.remaining {
		return 0, ErrInvalid
	}
	r.remaining -= int64(len(p))
	n, err := r.source.ReadAt(p, off)
	if n < 0 || n > len(p) {
		return 0, ErrInvalid
	}
	if err != nil || n != len(p) {
		return n, ErrInvalid
	}
	return n, nil
}

// mp4MetadataFits checks remaining read capacity before a bulk allocation.
func mp4MetadataFits(r io.ReaderAt, n int64) bool {
	if n < 0 || n > maxMP4MetadataBytes {
		return false
	}
	if bounded, ok := r.(*mp4MetadataReader); ok {
		return n <= bounded.remaining
	}
	return true
}

// mp4TrackTables selects each required table once and a single self-contained
// data reference. Other sample-size encodings and duplicate containers fail.
func mp4TrackTables(r io.ReaderAt, minf atom, budget *int) (mp4MapAtoms, error) {
	var result mp4MapAtoms
	children, err := atoms(r, minf.body, minf.end, budget)
	if err != nil {
		return result, err
	}
	var stbl, dinf *atom
	for _, child := range children {
		switch child.kind {
		case "stbl":
			if stbl != nil {
				return result, ErrInvalid
			}
			copy := child
			stbl = &copy
		case "dinf":
			if dinf != nil {
				return result, ErrInvalid
			}
			copy := child
			dinf = &copy
		}
	}
	if stbl == nil || dinf == nil {
		return result, ErrInvalid
	}
	refs, err := atoms(r, dinf.body, dinf.end, budget)
	if err != nil || len(refs) != 1 || refs[0].kind != "dref" {
		return result, ErrInvalid
	}
	dref := refs[0]
	var header [8]byte
	if dref.end-dref.body < 8 {
		return result, ErrInvalid
	}
	if _, err := r.ReadAt(header[:], dref.body); err != nil || binary.BigEndian.Uint32(header[:4]) != 0 || binary.BigEndian.Uint32(header[4:]) != 1 {
		return result, ErrInvalid
	}
	entries, err := atoms(r, dref.body+8, dref.end, budget)
	if err != nil || len(entries) != 1 || entries[0].kind != "url " || entries[0].end-entries[0].body != 4 {
		return result, ErrInvalid
	}
	var flags [4]byte
	if _, err := r.ReadAt(flags[:], entries[0].body); err != nil || binary.BigEndian.Uint32(flags[:]) != 1 {
		return result, ErrInvalid
	}
	tables, err := atoms(r, stbl.body, stbl.end, budget)
	if err != nil {
		return result, err
	}
	for _, table := range tables {
		var target *atom
		switch table.kind {
		case "stsd":
			target = &result.stsd
		case "stsz":
			target = &result.stsz
		case "stsc":
			target = &result.stsc
		case "stts":
			target = &result.stts
		case "stco", "co64":
			target = &result.offsets
		case "stz2":
			return result, ErrInvalid
		}
		if target != nil {
			if target.kind != "" {
				return result, ErrInvalid
			}
			*target = table
		}
	}
	if result.stsd.kind == "" || result.stsz.kind == "" || result.stsc.kind == "" || result.stts.kind == "" || result.offsets.kind == "" {
		return result, ErrInvalid
	}
	return result, nil
}

// mp4EntryTable validates FullBox/count/exact byte shape before reading entries.
func mp4EntryTable(r io.ReaderAt, a atom, stride, maxEntries uint64) ([]byte, uint64, error) {
	var header [8]byte
	if a.end-a.body < 8 {
		return nil, 0, ErrInvalid
	}
	if _, err := r.ReadAt(header[:], a.body); err != nil || binary.BigEndian.Uint32(header[:4]) != 0 {
		return nil, 0, ErrInvalid
	}
	count := uint64(binary.BigEndian.Uint32(header[4:]))
	if count == 0 || count > maxEntries || stride == 0 || count > (math.MaxUint64-8)/stride || uint64(a.end-a.body) != 8+count*stride {
		return nil, 0, ErrInvalid
	}
	data, err := readAtom(r, a, maxMP4MetadataBytes)
	return data, count, err
}

type mp4Sizes struct {
	count, uniform uint64
	data           []byte
}

// readMP4Sizes bounds sample count globally and rejects zero-sized samples.
func readMP4Sizes(r io.ReaderAt, a atom, budget *mp4SampleBudget) (mp4Sizes, error) {
	var header [12]byte
	if a.end-a.body < 12 {
		return mp4Sizes{}, ErrInvalid
	}
	if _, err := r.ReadAt(header[:], a.body); err != nil || binary.BigEndian.Uint32(header[:4]) != 0 {
		return mp4Sizes{}, ErrInvalid
	}
	sizes := mp4Sizes{uniform: uint64(binary.BigEndian.Uint32(header[4:8])), count: uint64(binary.BigEndian.Uint32(header[8:]))}
	if sizes.count == 0 || sizes.count > maxMP4Samples-budget.samples {
		return mp4Sizes{}, ErrInvalid
	}
	expected := uint64(12)
	if sizes.uniform == 0 {
		expected += 4 * sizes.count
	}
	if uint64(a.end-a.body) != expected {
		return mp4Sizes{}, ErrInvalid
	}
	if sizes.uniform == 0 {
		var err error
		sizes.data, err = readAtom(r, a, maxMP4MetadataBytes)
		if err != nil {
			return mp4Sizes{}, err
		}
		for i := uint64(0); i < sizes.count; i++ {
			if sizes.at(i) == 0 {
				return mp4Sizes{}, ErrInvalid
			}
		}
	}
	budget.samples += sizes.count
	return sizes, nil
}

// at reads a validated size without allocating or touching sample-body bytes.
func (s mp4Sizes) at(index uint64) uint64 {
	if s.uniform != 0 {
		return s.uniform
	}
	return uint64(binary.BigEndian.Uint32(s.data[12+index*4:]))
}

// inspectMP4Timing compares exact raw mdhd ticks, not rounded nanoseconds.
func inspectMP4Timing(r io.ReaderAt, a atom, samples, ticks uint64) error {
	data, runs, err := mp4EntryTable(r, a, 8, samples)
	if err != nil {
		return err
	}
	var count, total uint64
	for i := uint64(0); i < runs; i++ {
		at := 8 + i*8
		n := uint64(binary.BigEndian.Uint32(data[at : at+4]))
		delta := uint64(binary.BigEndian.Uint32(data[at+4 : at+8]))
		if n == 0 || delta == 0 || n > samples-count || delta > math.MaxUint64/n {
			return ErrInvalid
		}
		duration := n * delta
		if duration > math.MaxUint64-total {
			return ErrInvalid
		}
		count += n
		total += duration
	}
	if count != samples || total != ticks {
		return ErrInvalid
	}
	return nil
}

// inspectMP4Samples consumes exactly the size-table samples and records each
// chunk's half-open declared byte range for containment/overlap validation.
func inspectMP4Samples(r io.ReaderAt, tables mp4MapAtoms, ticks uint64, budget *mp4SampleBudget, bodies []mp4ByteRange) error {
	sizes, err := readMP4Sizes(r, tables.stsz, budget)
	if err != nil {
		return err
	}
	stride := uint64(4)
	if tables.offsets.kind == "co64" {
		stride = 8
	}
	offsets, chunks, err := mp4EntryTable(r, tables.offsets, stride, maxMP4Chunks-budget.chunks)
	if err != nil {
		return err
	}
	budget.chunks += chunks
	mapping, runs, err := mp4EntryTable(r, tables.stsc, 12, chunks)
	if err != nil {
		return err
	}
	var previous uint64
	for i := uint64(0); i < runs; i++ {
		at := 8 + i*12
		first := uint64(binary.BigEndian.Uint32(mapping[at : at+4]))
		perChunk := uint64(binary.BigEndian.Uint32(mapping[at+4 : at+8]))
		description := binary.BigEndian.Uint32(mapping[at+8 : at+12])
		if first == 0 || first > chunks || first <= previous || i == 0 && first != 1 || perChunk == 0 || perChunk > sizes.count || description != 1 {
			return ErrInvalid
		}
		previous = first
	}
	var used uint64
	for i := uint64(0); i < runs; i++ {
		at := 8 + i*12
		first := uint64(binary.BigEndian.Uint32(mapping[at : at+4]))
		perChunk := uint64(binary.BigEndian.Uint32(mapping[at+4 : at+8]))
		end := chunks
		if i+1 < runs {
			end = uint64(binary.BigEndian.Uint32(mapping[at+12:at+16])) - 1
		}
		for chunk := first - 1; chunk < end; chunk++ {
			if perChunk > sizes.count-used {
				return ErrInvalid
			}
			var length uint64
			if sizes.uniform != 0 {
				if perChunk > math.MaxUint64/sizes.uniform {
					return ErrInvalid
				}
				length = perChunk * sizes.uniform
			} else {
				for n := uint64(0); n < perChunk; n++ {
					size := sizes.at(used + n)
					if size > math.MaxUint64-length {
						return ErrInvalid
					}
					length += size
				}
			}
			used += perChunk
			var offset uint64
			if stride == 4 {
				offset = uint64(binary.BigEndian.Uint32(offsets[8+chunk*stride:]))
			} else {
				offset = binary.BigEndian.Uint64(offsets[8+chunk*stride:])
			}
			if length == 0 || offset > math.MaxUint64-length {
				return ErrInvalid
			}
			last := offset + length
			if !mp4Contained(offset, last, bodies) {
				return ErrInvalid
			}
			budget.ranges = append(budget.ranges, mp4ByteRange{offset, last})
		}
	}
	if used != sizes.count {
		return ErrInvalid
	}
	return inspectMP4Timing(r, tables.stts, sizes.count, ticks)
}

// mp4Contained uses top-level mdat bodies in file order; headers are excluded.
func mp4Contained(start, end uint64, bodies []mp4ByteRange) bool {
	i := sort.Search(len(bodies), func(i int) bool { return bodies[i].end > start })
	return end > start && i < len(bodies) && start >= bodies[i].start && end <= bodies[i].end
}

// mp4RangesDisjoint sorts all track ranges once; adjacent chunks are valid.
func mp4RangesDisjoint(ranges []mp4ByteRange) bool {
	sort.Slice(ranges, func(i, j int) bool {
		if ranges[i].start == ranges[j].start {
			return ranges[i].end < ranges[j].end
		}
		return ranges[i].start < ranges[j].start
	})
	for i := 1; i < len(ranges); i++ {
		if ranges[i].start < ranges[i-1].end {
			return false
		}
	}
	return true
}
