package env

import (
	"strconv"
)

// Bool returns a bool from the environment variable with the given key.
func Bool(key string, def bool) bool {
	return env(strconv.ParseBool, key, def)
}
