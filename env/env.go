//  Copyright (C) 2021-2023 Chronicle Labs, Inc.
//
//  This program is free software: you can redistribute it and/or modify
//  it under the terms of the GNU Affero General Public License as
//  published by the Free Software Foundation, either version 3 of the
//  License, or (at your option) any later version.
//
//  This program is distributed in the hope that it will be useful,
//  but WITHOUT ANY WARRANTY; without even the implied warranty of
//  MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
//  GNU Affero General Public License for more details.
//
//  You should have received a copy of the GNU Affero General Public License
//  along with this program.  If not, see <http://www.gnu.org/licenses/>.

package env

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/defiweb/go-eth/types"
)

// separator is used to split the environment variable values.
// It is taken from CFG_ITEM_SEPARATOR environment variable and defaults to a newline.
var separator = String(KeyItemSeparator, "\n")

// env is a helper function to read environment variables.
// It returns the default value if the environment variable is not set or if the value cannot be parsed.
func env[T any](f fn[T], key string, def T) (v T) {
	defer func() {
		log.Debugw(reflect.TypeOf(v).String(), key, v)
	}()
	s, ok := os.LookupEnv(key)
	if !ok {
		v = def
		return
	}
	var err error
	v, err = f(s)
	if err != nil {
		v = def
		return
	}
	return
}

type fn[T any] func(string) (T, error)

// pass is of type fn[T] and returns the value passed to it.
func pass[T any](s T) (T, error) {
	return s, nil
}

// String returns a string from the environment variable with the given key.
// If the variable is not set, the default value is returned.
// Empty values are allowed and valid.
func String(key, def string) string {
	return env(pass[string], key, def)
}

// Strings returns a slice of strings from the environment variable with the
// given key. If the variable is not set, the default value is returned.
// The value is split by the separator defined in the CFG_ITEM_SEPARATOR.
// Values are trimmed of the separator before splitting.
// If the environment variable exists but is empty, an empty slice is returned.
func Strings(key string, def []string) []string {
	v, ok := os.LookupEnv(key)
	if !ok {
		log.Debugw("[]string", key, def)
		return def
	}
	v = strings.Trim(v, separator)
	if v == "" {
		log.Debugw("[]string", key, "[]")
		return []string{}
	}
	log.Debugw("[]string", key, v)
	return strings.Split(v, separator)
}

// Address returns an address from the environment variable with the given key using types.AddressFromHex.
func Address(key string, def types.Address) (v types.Address) {
	return env(types.AddressFromHex, key, def)
}

// Duration returns a duration from the environment variable with the given key using time.ParseDuration.
func Duration(key string, def time.Duration) (v time.Duration) {
	return env(time.ParseDuration, key, def)
}

// scanDec is of type fn[T] and returns the decimal value scanned (using fmt.Sscanf) from the string passed to it.
func scanDec[T any](s string) (T, error) {
	var ret T
	_, err := fmt.Sscanf(s, "%d", &ret)
	return ret, err
}

// // scan is of type fn[T] and returns the decimal value scanned (using fmt.Sscan) from the string passed to it.
// func scan[T any](s string) (T, error) {
// 	var ret T
// 	_, err := fmt.Sscan(s, &ret)
// 	return ret, err
// }

// Bool returns a bool from the environment variable with the given key.
func Bool(key string, def bool) bool {
	return env(strconv.ParseBool, key, def)
}

// Int returns an int from the environment variable with the given key.
func Int(key string, def int) (v int) {
	return env(scanDec[int], key, def)
}

// Uint64 returns a uint64 from the environment variable with the given key.
func Uint64(key string, def uint64) uint64 {
	return env(scanDec[uint64], key, def)
}

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
