package env

var separatorKey = "ITEM_SEPARATOR"

// separator is used to split the environment variable values.
// It is taken from ENV_ITEM_SEPARATOR environment variable and defaults to a newline.
func separator() string { return String(separatorKey, "\n") }

// SetSeparatorKey sets the environment variable key to read the separator from.
func SetSeparatorKey(s string) {
	separatorKey = s
}
