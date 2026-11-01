// Package strictjson rejects lossy Unicode decoding before encoding/json runs.
package strictjson

import (
	"encoding/json"
	"errors"
	"unicode/utf8"
)

var ErrUnicode = errors.New("JSON contains invalid Unicode")

// Validate requires JSON syntax, UTF-8 bytes, and paired UTF-16 escapes. Go's
// encoding/json otherwise replaces lone surrogates with U+FFFD, changing keys.
func Validate(raw []byte) error {
	if !utf8.Valid(raw) || !json.Valid(raw) {
		return ErrUnicode
	}
	for i := 0; i < len(raw); i++ {
		if raw[i] != '"' {
			continue
		}
		i++
		for i < len(raw) && raw[i] != '"' {
			if raw[i] != '\\' {
				i++
				continue
			}
			i++
			if raw[i] != 'u' {
				i++
				continue
			}
			n := hex4(raw[i+1 : i+5])
			i += 5
			if n >= 0xDC00 && n <= 0xDFFF {
				return ErrUnicode
			}
			if n >= 0xD800 && n <= 0xDBFF {
				if i+6 > len(raw) || raw[i] != '\\' || raw[i+1] != 'u' {
					return ErrUnicode
				}
				low := hex4(raw[i+2 : i+6])
				if low < 0xDC00 || low > 0xDFFF {
					return ErrUnicode
				}
				i += 6
			}
		}
	}
	return nil
}

func hex4(b []byte) uint16 {
	var n uint16
	for _, c := range b {
		n <<= 4
		switch {
		case c >= '0' && c <= '9':
			n += uint16(c - '0')
		case c >= 'a' && c <= 'f':
			n += uint16(c - 'a' + 10)
		case c >= 'A' && c <= 'F':
			n += uint16(c - 'A' + 10)
		}
	}
	return n
}
