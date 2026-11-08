package media

import (
	"bytes"
	"context"
	"encoding/binary"
)

// mp3Frames rejects incomplete frames before decoding: the decoder intentionally
// treats some truncated frame errors as EOF. It also rejects sample-rate changes
// so a first-frame rate cannot understate the recording duration.
func mp3Frames(ctx context.Context, b []byte) (rate int, samples int64, err error) {
	offset := 0
	if len(b) >= 10 && bytes.Equal(b[:3], []byte("ID3")) {
		if b[3] < 2 || b[3] > 4 {
			return 0, 0, ErrInvalid
		}
		size := 0
		for _, v := range b[6:10] {
			if v&0x80 != 0 {
				return 0, 0, ErrInvalid
			}
			size = size<<7 | int(v)
		}
		offset = 10 + size
		if b[3] == 4 && b[5]&0x10 != 0 {
			offset += 10
		}
		if offset > len(b) {
			return 0, 0, ErrInvalid
		}
	}
	end := len(b)
	if end-offset >= 128 && bytes.Equal(b[end-128:end-125], []byte("TAG")) {
		end -= 128
	}
	for offset < end {
		if err := ctx.Err(); err != nil {
			return 0, 0, err
		}
		if end-offset < 4 {
			return 0, 0, ErrInvalid
		}
		h := binary.BigEndian.Uint32(b[offset : offset+4])
		version, layer, bitrateIndex, frequency := int(h>>19&3), int(h>>17&3), int(h>>12&15), int(h>>10&3)
		if h>>21 != 0x7ff || version == 1 || layer != 1 || bitrateIndex == 0 || bitrateIndex == 15 || frequency == 3 {
			return 0, 0, ErrInvalid
		}
		currentRate := []int{44100, 48000, 32000}[frequency]
		bitrate := []int{0, 32, 40, 48, 56, 64, 80, 96, 112, 128, 160, 192, 224, 256, 320}[bitrateIndex]
		coefficient, frameSamples := 144000, int64(1152)
		if version != 3 {
			currentRate /= 2
			if version == 0 {
				currentRate /= 2
			}
			bitrate = []int{0, 8, 16, 24, 32, 40, 48, 56, 64, 80, 96, 112, 128, 144, 160}[bitrateIndex]
			coefficient, frameSamples = 72000, 576
		}
		if rate != 0 && rate != currentRate {
			return 0, 0, ErrInvalid
		}
		rate = currentRate
		size := coefficient*bitrate/currentRate + int(h>>9&1)
		if size < 4 || size > end-offset {
			return 0, 0, ErrInvalid
		}
		offset += size
		samples += frameSamples
	}
	if samples == 0 {
		return 0, 0, ErrInvalid
	}
	if err := ctx.Err(); err != nil {
		return 0, 0, err
	}
	return rate, samples, nil
}
