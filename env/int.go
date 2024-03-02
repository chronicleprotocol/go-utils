package env

import (
	"fmt"
)

// scanDec is of type fn[T] and returns the decimal value scanned (using fmt.Sscanf) from the string passed to it.
func scanDec[T any](s string) (T, error) {
	var ret T
	_, err := fmt.Sscanf(s, "%d", &ret)
	return ret, err
}

// Int returns an int from the environment variable with the given key.
func Int(key string, def int) (v int) {
	return env(scanDec[int], key, def)
}

// Uint64 returns a uint64 from the environment variable with the given key.
func Uint64(key string, def uint64) uint64 {
	return env(scanDec[uint64], key, def)
}
