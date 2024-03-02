package env

import (
	"strings"
)

// String returns a string from the environment variable with the given key.
// If the variable is not set, the default value is returned.
// Empty values are allowed and valid.
func String(key, def string) string {
	return env(pass[string], key, def)
}

func stringSlice(s string) ([]string, error) {
	sep := separator()
	s = strings.Trim(s, sep)
	if len(s) == 0 {
		return nil, nil
	}
	return strings.Split(s, sep), nil
}

// Strings returns a slice of strings from the environment variable with the
// given key. If the variable is not set, the default value is returned.
// The value is split by the separator defined in the CFG_ITEM_SEPARATOR.
// Values are trimmed of the separator before splitting.
// If the environment variable exists but is empty, an empty slice is returned.
func Strings(key string, def []string) []string {
	return env(stringSlice, key, def)
}
