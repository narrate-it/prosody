package media

import (
	"bytes"
	"encoding/binary"
	"errors"
	"math"
	"testing"
)

// mp4Words encodes fixed big-endian fixture fields without interpreting media.
func mp4Words(values ...uint32) []byte {
	data := make([]byte, len(values)*4)
	for i, v := range values {
		binary.BigEndian.PutUint32(data[i*4:], v)
	}
	return data
}

// mp4SelfReference encodes the deliberately narrow one-url self-contained profile.
func mp4SelfReference() []byte {
	return mp4Box("dinf", mp4Box("dref", mp4Words(0, 1), mp4Box("url ", mp4Words(1))))
}

type mp4TestChunk struct {
	body   int
	offset uint64
}
type mp4TestTrack struct {
	kind                                    string
	uniform, count                          uint32
	sizes                                   []uint32
	chunks                                  []mp4TestChunk
	absolute                                []uint64
	mapping, timing                         []uint32
	ticks                                   uint64
	scale                                   uint32
	version                                 byte
	co64                                    bool
	edit                                    func(string, []byte) []byte
	extraTables, extraMinf, extraMdia, dinf []byte
	omit                                    string
	reference                               uint16
}

// mp4MappedTrack supplies a two-sample/two-chunk baseline with exact raw ticks.
func mp4MappedTrack(kind string) mp4TestTrack {
	return mp4TestTrack{kind: kind, uniform: 1, count: 2, chunks: []mp4TestChunk{{0, 0}, {0, 1}}, mapping: []uint32{1, 1, 1}, timing: []uint32{2, 1250}, ticks: 2500, scale: 1000, reference: 1}
}

// mp4TrackFixture encodes independent synthetic tables, including deliberate
// malformed bodies supplied by edit. No generated samples are decodable media.
func mp4TrackFixture(spec mp4TestTrack, bodies []mp4ByteRange) []byte {
	box := func(kind string, body []byte) []byte {
		if spec.edit != nil {
			body = spec.edit(kind, body)
		}
		if kind == spec.omit {
			return nil
		}
		return mp4Box(kind, body)
	}
	handler := make([]byte, 12)
	copy(handler[8:], spec.kind)
	mdhd := mp4Words(0, 0, 0, spec.scale, uint32(spec.ticks))
	if spec.version == 1 {
		mdhd = make([]byte, 32)
		mdhd[0] = 1
		binary.BigEndian.PutUint32(mdhd[20:], spec.scale)
		binary.BigEndian.PutUint64(mdhd[24:], spec.ticks)
	} else {
		mdhd[0] = spec.version
	}
	entry := make([]byte, 78)
	entryKind := "avc1"
	children := mp4Box("avcC", make([]byte, 7))
	if spec.kind == "soun" {
		entry = make([]byte, 28)
		entryKind = "mp4a"
		children = nil
	} else {
		binary.BigEndian.PutUint16(entry[24:], 640)
		binary.BigEndian.PutUint16(entry[26:], 360)
	}
	binary.BigEndian.PutUint16(entry[6:], spec.reference)
	stsd := append(mp4Words(0, 1), mp4Box(entryKind, entry, children)...)
	sizes := mp4Words(0, spec.uniform, spec.count)
	if spec.uniform == 0 {
		sizes = append(sizes, mp4Words(spec.sizes...)...)
	}
	offsetKind := "stco"
	stride := 4
	if spec.co64 {
		offsetKind = "co64"
		stride = 8
	}
	offsets := make([]byte, 8+len(spec.chunks)*stride)
	binary.BigEndian.PutUint32(offsets[4:], uint32(len(spec.chunks)))
	for i, chunk := range spec.chunks {
		var offset uint64
		if len(spec.absolute) > i {
			offset = spec.absolute[i]
		} else if len(bodies) > chunk.body {
			offset = bodies[chunk.body].start + chunk.offset
		}
		if spec.co64 {
			binary.BigEndian.PutUint64(offsets[8+i*stride:], offset)
		} else {
			binary.BigEndian.PutUint32(offsets[8+i*stride:], uint32(offset))
		}
	}
	tables := bytes.Join([][]byte{box("stsd", stsd), box("stsz", sizes), box("stsc", append(mp4Words(0, uint32(len(spec.mapping)/3)), mp4Words(spec.mapping...)...)), box("stts", append(mp4Words(0, uint32(len(spec.timing)/2)), mp4Words(spec.timing...)...)), box(offsetKind, offsets), spec.extraTables}, nil)
	dinf := spec.dinf
	if dinf == nil {
		dinf = mp4SelfReference()
	}
	if spec.omit == "dinf" {
		dinf = nil
	}
	return mp4Box("trak", mp4Box("mdia", box("hdlr", handler), box("mdhd", mdhd), box("minf", bytes.Join([][]byte{dinf, box("stbl", tables), spec.extraMinf}, nil)), spec.extraMdia))
}

// mp4SampleFixture resolves absolute offsets after metadata layout is fixed.
func mp4SampleFixture(specs []mp4TestTrack, payloads ...[]byte) ([]byte, []mp4ByteRange) {
	build := func(bodies []mp4ByteRange) []byte {
		tracks := []byte{}
		for _, spec := range specs {
			tracks = append(tracks, mp4TrackFixture(spec, bodies)...)
		}
		return bytes.Join([][]byte{mp4Box("ftyp", []byte("isom\x00\x00\x00\x00")), mp4Box("moov", mp4Box("mvhd", mp4Words(0, 0, 0, 1000, 2500)), tracks)}, nil)
	}
	head := build(nil)
	bodies := make([]mp4ByteRange, len(payloads))
	position := uint64(len(head))
	for i, payload := range payloads {
		bodies[i] = mp4ByteRange{position + 8, position + 8 + uint64(len(payload))}
		position += 8 + uint64(len(payload))
	}
	result := build(bodies)
	for _, payload := range payloads {
		result = append(result, mp4Box("mdat", payload)...)
	}
	return result, bodies
}

// mp4GuardReader denies ANY sample-body read and records requested metadata bytes.
type mp4GuardReader struct {
	data     []byte
	bodies   []mp4ByteRange
	reads    int64
	bodyRead bool
}

// ReadAt checks each requested interval, not only its first byte, against mdat.
func (r *mp4GuardReader) ReadAt(p []byte, off int64) (int, error) {
	r.reads += int64(len(p))
	for _, body := range r.bodies {
		if off >= 0 && uint64(off) < body.end && uint64(off)+uint64(len(p)) > body.start {
			r.bodyRead = true
			return 0, errors.New("sample body read forbidden")
		}
	}
	return bytes.NewReader(r.data).ReadAt(p, off)
}

// TestProbeMP4SampleMapPositiveProfiles admits coherent structural maps without
// reading payloads, including reordered/interleaved chunks and multiple mdats.
func TestProbeMP4SampleMapPositiveProfiles(t *testing.T) {
	t.Parallel()
	for _, name := range []string{"uniform stco adjacent", "variable co64 multiple mdat", "variable chunk runs", "nonmonotonic chunks", "two track interleaving", "mdhd v1 wide ticks", "subnanosecond raw ticks"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			video := mp4MappedTrack("vide")
			specs := []mp4TestTrack{video}
			payloads := [][]byte{{1, 2}}
			switch name {
			case "variable co64 multiple mdat":
				video.uniform = 0
				video.count = 3
				video.sizes = []uint32{1, 2, 3}
				video.co64 = true
				video.chunks = []mp4TestChunk{{0, 0}, {0, 1}, {1, 0}}
				video.timing = []uint32{1, 500, 2, 1000}
				specs = []mp4TestTrack{video}
				payloads = [][]byte{{1, 2, 3}, {4, 5, 6}}
			case "variable chunk runs":
				video.uniform = 0
				video.count = 3
				video.sizes = []uint32{1, 2, 3}
				video.mapping = []uint32{1, 2, 1, 2, 1, 1}
				video.chunks = []mp4TestChunk{{0, 0}, {0, 3}}
				video.timing = []uint32{1, 500, 2, 1000}
				specs = []mp4TestTrack{video}
				payloads = [][]byte{{1, 2, 3, 4, 5, 6}}
			case "mdhd v1 wide ticks":
				video.version, video.scale, video.ticks = 1, 2_000_000_000, 5_000_000_000
				video.timing = []uint32{2, 2_500_000_000}
				specs = []mp4TestTrack{video}
			case "subnanosecond raw ticks":
				video.scale, video.ticks, video.timing = math.MaxUint32, 2, []uint32{2, 1}
				specs = []mp4TestTrack{video}
			case "nonmonotonic chunks":
				video.chunks = []mp4TestChunk{{0, 1}, {0, 0}}
				specs = []mp4TestTrack{video}
			case "two track interleaving":
				video.chunks = []mp4TestChunk{{0, 0}, {0, 2}}
				audio := mp4MappedTrack("soun")
				audio.co64 = true
				audio.chunks = []mp4TestChunk{{0, 1}, {0, 3}}
				specs = []mp4TestTrack{video, audio}
				payloads = [][]byte{{1, 2, 3, 4}}
			}
			data, bodies := mp4SampleFixture(specs, payloads...)
			reader := &mp4GuardReader{data: data, bodies: bodies}
			info, err := ProbeMP4(reader, int64(len(data)), int64(len(data)), 640*360)
			if err != nil || info.Width != 640 || info.Height != 360 || info.MIME != "video/mp4" {
				t.Fatalf("coherent synthetic map rejected: %+v %v", info, err)
			}
			if reader.bodyRead || reader.reads > maxMP4MetadataBytes {
				t.Fatal("probe read sample bodies or exceeded shared metadata budget")
			}
		})
	}
}

// mp4EditWord changes one encoded field while preserving other fixture bytes.
func mp4EditWord(target string, offset int, value uint32) func(string, []byte) []byte {
	return func(kind string, body []byte) []byte {
		if kind == target {
			body = bytes.Clone(body)
			binary.BigEndian.PutUint32(body[offset:], value)
		}
		return body
	}
}

// TestProbeMP4RejectsInconsistentSampleMaps keeps all unrelated metadata valid
// while violating one sample-count, chunk-run, timing or closed-table rule.
func TestProbeMP4RejectsInconsistentSampleMaps(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		change func(*mp4TestTrack)
	}{
		{"zero chunk count", func(s *mp4TestTrack) { s.chunks = nil }},
		{"zero sample count", func(s *mp4TestTrack) { s.count = 0 }},
		{"huge sample count", func(s *mp4TestTrack) { s.count = math.MaxUint32 }},
		{"uniform trailing entries", func(s *mp4TestTrack) {
			s.edit = func(k string, b []byte) []byte {
				if k == "stsz" {
					return append(b, 0, 0, 0, 1)
				}
				return b
			}
		}},
		{"short stsz header", func(s *mp4TestTrack) {
			s.edit = func(k string, b []byte) []byte {
				if k == "stsz" {
					return b[:11]
				}
				return b
			}
		}},
		{"variable zero sample", func(s *mp4TestTrack) { s.uniform = 0; s.sizes = []uint32{1, 0} }},
		{"variable missing sample", func(s *mp4TestTrack) { s.uniform = 0; s.sizes = []uint32{1} }},
		{"variable extra sample", func(s *mp4TestTrack) { s.uniform = 0; s.sizes = []uint32{1, 1, 1} }},
		{"first chunk zero", func(s *mp4TestTrack) { s.mapping = []uint32{0, 1, 1} }},
		{"first chunk two", func(s *mp4TestTrack) { s.mapping = []uint32{2, 1, 1} }},
		{"repeated run start", func(s *mp4TestTrack) { s.mapping = []uint32{1, 1, 1, 1, 1, 1} }},
		{"decreasing run start", func(s *mp4TestTrack) {
			s.chunks = append(s.chunks, mp4TestChunk{0, 2})
			s.count = 3
			s.timing, s.ticks = []uint32{3, 1250}, 3750
			s.mapping = []uint32{1, 1, 1, 3, 1, 1, 2, 1, 1}
		}},
		{"tail run outside chunks", func(s *mp4TestTrack) { s.mapping = []uint32{1, 1, 1, 3, 1, 1} }},
		{"zero samples per chunk", func(s *mp4TestTrack) { s.mapping = []uint32{1, 0, 1} }},
		{"wrong sample description", func(s *mp4TestTrack) { s.mapping = []uint32{1, 1, 2} }},
		{"underconsumed sizes", func(s *mp4TestTrack) { s.chunks = s.chunks[:1] }},
		{"overconsumed sizes", func(s *mp4TestTrack) { s.mapping = []uint32{1, 2, 1} }},
		{"zero time count", func(s *mp4TestTrack) { s.timing = []uint32{0, 1250} }},
		{"zero time delta", func(s *mp4TestTrack) { s.timing = []uint32{2, 0} }},
		{"time count mismatch", func(s *mp4TestTrack) { s.timing = []uint32{1, 2500} }},
		{"time ticks mismatch", func(s *mp4TestTrack) { s.timing = []uint32{2, 1249} }},
		{"same rounded duration different ticks", func(s *mp4TestTrack) { s.scale = math.MaxUint32; s.ticks = 3; s.timing = []uint32{2, 1} }},
		{"mdhd v1 conversion overflow", func(s *mp4TestTrack) { s.version = 1; s.scale = 1; s.ticks = math.MaxUint64 }},
		{"mdhd v1 huge representable ticks mismatch", func(s *mp4TestTrack) { s.version = 1; s.scale = math.MaxUint32; s.ticks = math.MaxUint64 }},
		{"stsc huge entry count", func(s *mp4TestTrack) { s.edit = mp4EditWord("stsc", 4, math.MaxUint32) }},
		{"stts huge entry count", func(s *mp4TestTrack) { s.edit = mp4EditWord("stts", 4, math.MaxUint32) }},
		{"offset huge entry count", func(s *mp4TestTrack) { s.edit = mp4EditWord("stco", 4, math.MaxUint32) }},
		{"short offset body", func(s *mp4TestTrack) {
			s.edit = func(k string, b []byte) []byte {
				if k == "stco" {
					return b[:len(b)-1]
				}
				return b
			}
		}},
		{"uint64 chunk end overflow", func(s *mp4TestTrack) { s.co64 = true; s.absolute = []uint64{math.MaxUint64, math.MaxUint64 - 1} }},
		{"compact size conflict", func(s *mp4TestTrack) { s.extraTables = mp4Box("stz2", make([]byte, 12)) }},
		{"duplicate stsd", func(s *mp4TestTrack) {
			header := make([]byte, 78)
			binary.BigEndian.PutUint16(header[6:], 1)
			binary.BigEndian.PutUint16(header[24:], 640)
			binary.BigEndian.PutUint16(header[26:], 360)
			s.extraTables = mp4Box("stsd", mp4Words(0, 1), mp4Box("avc1", header, mp4Box("avcC", make([]byte, 7))))
		}},
		{"duplicate stsz", func(s *mp4TestTrack) { s.extraTables = mp4Box("stsz", mp4Words(0, 1, 2)) }},
		{"duplicate stsc", func(s *mp4TestTrack) { s.extraTables = mp4Box("stsc", mp4Words(0, 1, 1, 1, 1)) }},
		{"duplicate stts", func(s *mp4TestTrack) { s.extraTables = mp4Box("stts", mp4Words(0, 1, 2, 1250)) }},
		{"stco co64 conflict", func(s *mp4TestTrack) { s.extraTables = mp4Box("co64", mp4Words(0, 2, 0, 0, 0, 1)) }},
		{"duplicate offsets", func(s *mp4TestTrack) { s.extraTables = mp4Box("stco", mp4Words(0, 2, 0, 1)) }},
		{"duplicate minf", func(s *mp4TestTrack) { s.extraMdia = mp4Box("minf") }},
		{"duplicate mdhd", func(s *mp4TestTrack) { s.extraMdia = mp4Box("mdhd", mp4Words(0, 0, 0, 1000, 2500)) }},
		{"duplicate stbl", func(s *mp4TestTrack) { s.extraMinf = mp4Box("stbl") }},
		{"duplicate dinf", func(s *mp4TestTrack) { s.extraMinf = mp4SelfReference() }},
		{"truncated child header", func(s *mp4TestTrack) { s.extraTables = []byte{0, 0, 0, 8, 1, 2, 3} }},
		{"child escapes parent", func(s *mp4TestTrack) { s.extraTables = mp4Words(math.MaxUint32, 0x66726565) }},
	}
	for _, kind := range []string{"stsd", "stsz", "stsc", "stts", "stco", "co64"} {
		for _, version := range []bool{false, true} {
			kind, version := kind, version
			name := "flags"
			value := uint32(1)
			if version {
				name = "version"
				value = 1 << 24
			}
			cases = append(cases, struct {
				name   string
				change func(*mp4TestTrack)
			}{kind + " " + name, func(s *mp4TestTrack) { s.co64 = kind == "co64"; s.edit = mp4EditWord(kind, 0, value) }})
		}
	}
	for _, kind := range []string{"stsd", "stsz", "stsc", "stts", "stco", "mdhd", "minf", "stbl", "dinf"} {
		kind := kind
		cases = append(cases, struct {
			name   string
			change func(*mp4TestTrack)
		}{"missing " + kind, func(s *mp4TestTrack) { s.omit = kind }})
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			spec := mp4MappedTrack("vide")
			tc.change(&spec)
			data, bodies := mp4SampleFixture([]mp4TestTrack{spec}, make([]byte, 8))
			reader := &mp4GuardReader{data: data, bodies: bodies}
			out, err := ProbeMP4(reader, int64(len(data)), int64(len(data)), 640*360)
			if !errors.Is(err, ErrInvalid) || out != (UnverifiedVideoMetadata{}) {
				t.Fatalf("inconsistent map accepted: %+v %v", out, err)
			}
			if reader.bodyRead {
				t.Fatal("malformed metadata read mdat body")
			}
		})
	}
}

// TestProbeMP4RejectsExternalAndAmbiguousReferences enforces the single index1,
// version0/flags1/url-without-payload reference profile, not general ISO support.
func TestProbeMP4RejectsExternalAndAmbiguousReferences(t *testing.T) {
	t.Parallel()
	refs := []struct {
		name      string
		reference uint16
		dinf      []byte
	}{
		{"sample reference zero", 0, nil}, {"sample reference two", 2, nil},
		{"dref missing", 1, mp4Box("dinf")},
		{"duplicate dref", 1, mp4Box("dinf", mp4Box("dref", mp4Words(0, 1), mp4Box("url ", mp4Words(1))), mp4Box("dref", mp4Words(0, 1), mp4Box("url ", mp4Words(1))))},
		{"external url", 1, mp4Box("dinf", mp4Box("dref", mp4Words(0, 1), mp4Box("url ", append(mp4Words(0), []byte("https://example.invalid\x00")...))))},
		{"external empty url", 1, mp4Box("dinf", mp4Box("dref", mp4Words(0, 1), mp4Box("url ", mp4Words(0))))},
		{"self url payload", 1, mp4Box("dinf", mp4Box("dref", mp4Words(0, 1), mp4Box("url ", mp4Words(1, 0))))},
		{"url version", 1, mp4Box("dinf", mp4Box("dref", mp4Words(0, 1), mp4Box("url ", mp4Words(0x01000001))))},
		{"wrong entry kind", 1, mp4Box("dinf", mp4Box("dref", mp4Words(0, 1), mp4Box("urn ", mp4Words(1))))},
		{"duplicate url entries", 1, mp4Box("dinf", mp4Box("dref", mp4Words(0, 1), mp4Box("url ", mp4Words(1)), mp4Box("url ", mp4Words(1))))},
		{"wrong dref count", 1, mp4Box("dinf", mp4Box("dref", mp4Words(0, 2), mp4Box("url ", mp4Words(1))))},
		{"dref version", 1, mp4Box("dinf", mp4Box("dref", mp4Words(0x01000000, 1), mp4Box("url ", mp4Words(1))))},
		{"dref flags", 1, mp4Box("dinf", mp4Box("dref", mp4Words(1, 1), mp4Box("url ", mp4Words(1))))},
	}
	for _, tc := range refs {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			spec := mp4MappedTrack("vide")
			spec.reference, spec.dinf = tc.reference, tc.dinf
			data, _ := mp4SampleFixture([]mp4TestTrack{spec}, []byte{1, 2})
			if _, err := ProbeMP4(bytes.NewReader(data), int64(len(data)), int64(len(data)), 640*360); !errors.Is(err, ErrInvalid) {
				t.Fatal("unsupported reference accepted")
			}
		})
	}
}

// TestProbeMP4ChunkRangeDenials distinguishes declared mdat bodies from headers,
// gaps and overlap; physical sample existence/codec validity remains untested.
func TestProbeMP4ChunkRangeDenials(t *testing.T) {
	t.Parallel()
	for _, name := range []string{"ftyp header", "moov bytes", "mdat header", "mdat end", "outside file", "overrun body", "cross mdat header", "intra track overlap", "cross track overlap"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			video := mp4MappedTrack("vide")
			payloads := [][]byte{make([]byte, 8), make([]byte, 8)}
			_, bodies := mp4SampleFixture([]mp4TestTrack{video}, payloads...)
			video.absolute = []uint64{bodies[0].start, bodies[0].start + 1}
			specs := []mp4TestTrack{video}
			switch name {
			case "ftyp header":
				video.absolute[0] = 0
			case "moov bytes":
				video.absolute[0] = 24
			case "mdat header":
				video.absolute[0] = bodies[0].start - 1
			case "mdat end":
				video.absolute[0] = bodies[1].end
			case "outside file":
				video.absolute[0] = math.MaxUint32
			case "overrun body":
				video.uniform = 2
				video.absolute[0] = bodies[0].end - 1
			case "cross mdat header":
				video.uniform = 2
				video.absolute[0] = bodies[0].end - 1
				video.absolute[1] = bodies[1].start
			case "intra track overlap":
				video.absolute[1] = video.absolute[0]
			case "cross track overlap":
				audio := mp4MappedTrack("soun")
				audio.chunks = []mp4TestChunk{{0, 1}, {0, 2}}
				specs = append(specs, audio)
			}
			specs[0] = video
			// Adding an audio track changes metadata layout: use relative offsets for
			// overlap, while single-track absolute-header tests keep the original layout.
			if name == "cross track overlap" {
				specs[0].absolute = nil
			}
			data, actual := mp4SampleFixture(specs, payloads...)
			reader := &mp4GuardReader{data: data, bodies: actual}
			if _, err := ProbeMP4(reader, int64(len(data)), int64(len(data)), 640*360); !errors.Is(err, ErrInvalid) {
				t.Fatal("invalid declared chunk range accepted")
			}
			if reader.bodyRead {
				t.Fatal("range validation touched sample bodies")
			}
		})
	}
}
