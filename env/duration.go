package env

import (
	"time"
)

// Duration returns a duration from the environment variable with the given key using time.ParseDuration.
func Duration(key string, def time.Duration) (v time.Duration) {
	return env(time.ParseDuration, key, def)
}
