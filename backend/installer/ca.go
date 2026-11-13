package main

import (
	"encoding/base64"
	"encoding/pem"
	"fmt"
)

// decodeCA accepts either a raw PEM or a base64-encoded PEM custody value.
func decodeCA(value string) ([]byte, error) {
	if block, _ := pem.Decode([]byte(value)); block != nil {
		return []byte(value), nil
	}
	decoded, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		return nil, fmt.Errorf("CA custody is neither PEM nor base64")
	}
	if block, _ := pem.Decode(decoded); block == nil {
		return nil, fmt.Errorf("decoded CA is not a PEM certificate")
	}
	return decoded, nil
}
