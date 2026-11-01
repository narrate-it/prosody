// Package media validates original assets without rewriting their bytes.
package media

import (
	"bytes"
	"errors"
	"image"
	_ "image/jpeg"
	_ "image/png"
)

var ErrInvalid = errors.New("invalid or unsupported media")

type ImageInfo struct {
	MIME          string
	Width, Height int
}

// InspectImage admits bounded JPEG/PNG dimensions and a complete decode. It
// refuses decompression bombs before allocating the decoded pixel buffer.
func InspectImage(b []byte, maxBytes int, maxPixels int64) (ImageInfo, error) {
	if maxBytes <= 0 || maxPixels <= 0 || len(b) == 0 || len(b) > maxBytes {
		return ImageInfo{}, ErrInvalid
	}
	config, format, err := image.DecodeConfig(bytes.NewReader(b))
	if err != nil || config.Width <= 0 || config.Height <= 0 || int64(config.Width) > maxPixels/int64(config.Height) {
		return ImageInfo{}, ErrInvalid
	}
	if format != "jpeg" && format != "png" {
		return ImageInfo{}, ErrInvalid
	}
	decoded, actual, err := image.Decode(bytes.NewReader(b))
	if err != nil || actual != format || decoded.Bounds().Dx() != config.Width || decoded.Bounds().Dy() != config.Height {
		return ImageInfo{}, ErrInvalid
	}
	mime := "image/" + format
	return ImageInfo{mime, config.Width, config.Height}, nil
}
