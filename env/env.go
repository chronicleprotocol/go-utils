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
	"os"
	"reflect"
)

type fn[T any] func(string) (T, error)

// env is a helper function to read environment variables.
// It returns the default value if the environment variable is not set or if the value cannot be parsed.
func env[T any](f fn[T], key string, def T) (v T) {
	defer func() {
		log.Debugw(reflect.TypeOf(v).String(), key, v)
	}()
	s, ok := os.LookupEnv(key)
	if !ok {
		return def
	}
	val, err := f(s)
	if err != nil {
		return def
	}
	return val
}

// pass is of type fn[T] and returns the value passed to it.
func pass[T any](s T) (T, error) {
	return s, nil
}
