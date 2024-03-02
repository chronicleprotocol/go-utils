package env

import (
	"github.com/defiweb/go-eth/types"
)

// Address returns an address from the environment variable with the given key using types.AddressFromHex.
func Address(key string, def types.Address) (v types.Address) {
	return env(types.AddressFromHex, key, def)
}
