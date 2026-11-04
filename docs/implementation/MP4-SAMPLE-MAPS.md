# Bounded MP4 structural sample maps

Source preparation from `c35934dcc1455c1a62bf4d6df41a683467727610` extends the
existing ProbeMP4 profile. It still returns **UnverifiedVideoMetadata**. Movie
Duration is the converted `mvhd` header claim and dimensions are `avc1` header
claims. Neither is a measured/decoded or presentation-timeline authority. No
production caller, publication path, media admission or native route is added.
No decodable video asset was downloaded.

## Structural profile

Each supported track must contain one mdia, mdhd, minf, stbl and dinf/dref chain,
one stsd with one supported sample entry, one stsz, stsc and stts, and exactly
one stco OR co64. Duplicate required containers/tables and unsupported stz2
are rejected. Existing one-video/optional-audio, fragmentation, protected-entry,
AVC minimum configuration and QuickTime audio-header version controls remain.
Tables stsd/stsz/stsc/stts/stco/co64 and dref use FullBox version 0/flags 0.
Movie/media time prefixes retain versions 0/1, zero flags and existing checked
conversion to time.Duration; raw media ticks remain available for stts checking.
This is a deliberately restrictive profile, not all-valid-MP4 compatibility.

stsz requires a nonzero sample count and nonzero sample sizes. Uniform-size
bodies are exactly 12 bytes; variable-size bodies are exactly 12+4*N, with no
zero-sized entries. stsc starts at chunk 1, has strictly increasing in-range
run starts, positive samples-per-chunk and sample-description ID 1. Every chunk
consumes the exact size-table sample count, with checked uint64 byte arithmetic.
No trailing run may silently start beyond the chunk table. stts counts/deltas
are positive, its total count equals stsz and its checked tick sum equals raw
mdhd ticks, including version-1 values above 32 bits. Nanosecond rounding cannot
make different raw tick totals equal.

Chunk offsets are absolute file offsets. Each nonempty half-open range must
fit wholly in ONE declared top-level mdat body, excluding box headers/gaps and
end overrun. Checked addition precedes any containment comparison; large co64
values cannot wrap into admitted ranges. All track ranges are sorted once to
reject cross-track and intra-track overlaps. Adjacent chunks, nonmonotonic file
offsets, interleaved tracks and multiple mdats remain possible. mdat lookup is
binary search over file-ordered declared bodies; overlap sorting is O(C log C),
not pairwise scans over up to 100,000 chunks.

Every sample entry must use data_reference_index 1. The sole dref entry must be
`url `, FullBox version 0/flags 1, with no URI/payload beyond that FullBox field.
External/ambiguous references fail; no URL is fetched or resolved.

## Shared resource and I/O limits

The private constants are shared across the entire ProbeMP4 call, not reset per
track: `maxMP4Samples=1,000,000`, `maxMP4Chunks=100,000`,
`maxMP4MetadataBytes=16<<20` (16 MiB). The existing track cap is eight and atom
budget is 4096. Advertised counts, exact table lengths and remaining allocation
capacity are checked before bulk allocation. Variable-size table storage and
chunk ranges are bounded by these limits; no sample-body buffer is created.

The metadata cap means cumulative REQUESTED ReaderAt bytes: all atom headers,
small prefixes, hdlr/stsd/codec reads and repeated table prefixes are charged.
A private per-call ReaderAt wrapper covers existing parser paths as well as new
tables, rejects short reads (even a caller returning nil error), and checks
remaining capacity before bulk metadata allocation. It does not promise to
interrupt a blocking ReaderAt or make an unstable reader into an immutable
snapshot. Those are separate object-adapter/qualification responsibilities.

No mdat body bytes are requested. Containment checks declared container ranges,
not physical sample existence, digest identity, immutable storage, decodable
frames or codec dimensions. A caller-supplied size/ReaderAt is not a durable
object proof. Edit lists, composition offsets/ctts, movie-vs-media mapping,
complete codec configuration and actual decoded timing remain UNQUALIFIED.
No presentation-duration, Nanos/storage, native or publication acceptance is
inferred, and this is not the fragmented W3C MSE profile.

## Structural tests and executed scoped receipts

Existing synthetic codec/protection fixtures now carry valid self-references,
size/chunk/time tables, mdat bytes and absolute offsets, preserving their
intended negative conditions. New guarded-reader positives cover uniform and
variable sizes, stco/co64, adjacent/nonmonotonic/interleaved/multiple-mdat chunks,
a change between at least two stsc runs, mdhd-v1 5,000,000,000 ticks and a
subnanosecond raw-tick control. A paired different-ticks/same-rounded-duration
negative checks exact raw comparison. These containers are NOT decodable media.

Negative cases cover malformed/count/size/FullBox table fields, missing/duplicate
containers, unsupported compact sizes, wrong description/reference, external
URLs, unconsumed/excess samples, timing/overflow and header/out-of-body/overlap
ranges. GuardReader denies any request overlapping mdat bodies. Disjoint
resource tests exercise exact/over aggregate sample/chunk limits, metadata
requested-byte boundary (including repeated prefixes), preallocation rejection,
short-nil-error readers and huge counts on small metadata inputs. Primitive
16-MiB reader tests are identified separately from public ProbeMP4 profiles;
they do not pretend to be decoded or 16-MiB valid movie inputs.

The coordinator tested exact source tree
`fbb983833fe270557afb6c74a35b1c46cec2763c` on base
`c35934dcc1455c1a62bf4d6df41a683467727610`. The focused
`^(TestProbeMP4|TestMP4)` scope passed all 129 named test/subtest records across
15 top-level tests: exit 0, no failures or skips.

A compiled overlay of the old mp4.go produced three failed records:
missing_stsz, time_ticks_mismatch and their parent. The uniform positive case
and its parent passed (two pass records), no skips, exit 1 as expected.
A separate compiled omission removed ONLY the containment guard while keeping
the other source/fixtures: mdat_header, cross_mdat_header and their parent
failed (three failure records). The same uniform control and its parent passed
(two pass records), no skips, exit 1 as expected. Working source was unchanged.

The containment subtest selector lacked a leading anchor and also selected
cross_mdat_header. The supervisor expected two failure records instead of the
actual three and FAILED its count assertion after the Go command completed.
An audit of exact events/assertion messages retained that supervisor failure
and the valid extra containment regression; no result was silently discarded.
The planned full-media step was NOT REACHED, so this sequence supplies no
full-media pass. The exact owned build lease was released. Lint, the full Go
unit gate and the required commit hook are separate from this scoped receipt;
Consult the PR verification record for those separate outcomes.

These are synthetic metadata/profile checks, not decoded media, actual object
bytes, native, DB, provider or device qualification. No provider call, external
media fixture, credential lookup, infrastructure, signing or deployment is part
of this slice. All full-codec/timeline/publication/I/O gates stay open.

## Primary format references

The [sample-to-chunk table](https://developer.apple.com/documentation/quicktime-file-format/sample-to-chunk_atom/sample-to-chunk_table),
[chunk offsets](https://developer.apple.com/documentation/quicktime-file-format/chunk_offset_atom),
[time-to-sample table](https://developer.apple.com/documentation/quicktime-file-format/time-to-sample_atom/time-to-sample_table)
and [sample-size atom](https://developer.apple.com/documentation/quicktime-file-format/sample_size_atom)
describe the field layouts used here. The bounds, zero-size rejection and single
self-contained data-reference mode are this probe's explicit narrower policy,
not a claim that those restrictions define all valid QuickTime/ISO BMFF files.
