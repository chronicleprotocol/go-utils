package env

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
)

// HexBytes returns a byte slice from the environment variable with the given key using hex.DecodeString.
func HexBytes(key string, def []byte) []byte {
	return env(hex.DecodeString, key, def)
}

func HexBytesOfSize(key string, l int) ([]byte, error) {
	b := HexBytes(key, nil)
	if len(b) != l {
		return nil, fmt.Errorf("invalid bytes length - want: %d, got: %d", l, len(b))
	}
	return b, nil
}

func ReaderFromHexOrRandomOfSize(key string, size int) io.Reader {
	seedReader := rand.Reader
	if seed, err := HexBytesOfSize(key, size); err != nil {
		log.Warn(err, " - using random seed")
	} else {
		seedReader = bytes.NewReader(seed)
	}
	return seedReader
}
