package env

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/defiweb/go-eth/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBool(t *testing.T) {
	tests := []struct {
		key string
		def bool

		want    bool
		envVars map[string]string
	}{
		{
			key:  "test",
			def:  true,
			want: true,
		}, {
			key:  "test",
			def:  false,
			want: false,
		}, {
			key:  "test",
			def:  false,
			want: true,
			envVars: map[string]string{
				"test": "1",
			},
		}, {
			key:  "test",
			def:  false,
			want: true,
			envVars: map[string]string{
				"test": "true",
			},
		}, {
			key:  "test",
			def:  true,
			want: false,
			envVars: map[string]string{
				"test": "0",
			},
		}, {
			key:  "test",
			def:  true,
			want: false,
			envVars: map[string]string{
				"test": "false",
			},
		},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf(`want %t when %s="%s" default is %t`, tt.want, tt.key, tt.envVars[tt.key], tt.def), func(t *testing.T) {
			for k, v := range tt.envVars {
				require.NoErrorf(t, os.Setenv(k, v), "failed to set %s to %v", k, v)
			}
			defer func() {
				for k := range tt.envVars {
					require.NoErrorf(t, os.Unsetenv(k), "failed to unset %s", k)
				}
			}()
			if gotV := Bool(tt.key, tt.def); gotV != tt.want {
				t.Errorf("Bool() = %v, want %v", gotV, tt.want)
			}
		})
	}
}

func Test_env(t *testing.T) {
	require.NoError(t, os.Unsetenv("TEST_KEY"))
	assert.Equal(t, "", String("TEST_KEY", ""))
	assert.Equal(t, "default", String("TEST_KEY", "default"))
	require.NoError(t, os.Setenv("TEST_KEY", "value"))
	assert.Equal(t, "value", String("TEST_KEY", "default"))

	require.NoError(t, os.Unsetenv("TEST_KEY"))
	assert.Equal(t, []byte{}, HexBytes("TEST_KEY", []byte{}))
	assert.Equal(t, []byte{1, 2, 15}, HexBytes("TEST_KEY", []byte{1, 2, 15}))
	require.NoError(t, os.Setenv("TEST_KEY", "0f0203"))
	assert.Equal(t, []byte{15, 2, 3}, HexBytes("TEST_KEY", []byte{1, 2, 15}))

	require.NoError(t, os.Unsetenv("TEST_KEY"))
	assert.Equal(t, types.Address{}, Address("TEST_KEY", types.Address{}))
	assert.Equal(t, types.MustAddressFromHex("0x0123456789012345678901234567890123456789"), Address("TEST_KEY", types.MustAddressFromHex("0x0123456789012345678901234567890123456789")))
	require.NoError(t, os.Setenv("TEST_KEY", "0x1234567890123456789012345678901234567890"))
	assert.Equal(t, types.MustAddressFromHex("0x1234567890123456789012345678901234567890"), Address("TEST_KEY", types.MustAddressFromHex("0x0123456789012345678901234567890123456789")))

	require.NoError(t, os.Unsetenv("TEST_KEY"))
	assert.Equal(t, false, Bool("TEST_KEY", false))
	assert.Equal(t, true, Bool("TEST_KEY", true))
	require.NoError(t, os.Setenv("TEST_KEY", "f"))
	assert.Equal(t, false, Bool("TEST_KEY", true))
	require.NoError(t, os.Setenv("TEST_KEY", "t"))
	assert.Equal(t, true, Bool("TEST_KEY", false))
	require.NoError(t, os.Setenv("TEST_KEY", "0"))
	assert.Equal(t, false, Bool("TEST_KEY", true))
	require.NoError(t, os.Setenv("TEST_KEY", "1"))
	assert.Equal(t, true, Bool("TEST_KEY", false))
	require.NoError(t, os.Setenv("TEST_KEY", "false"))
	assert.Equal(t, false, Bool("TEST_KEY", true))
	require.NoError(t, os.Setenv("TEST_KEY", "true"))
	assert.Equal(t, true, Bool("TEST_KEY", false))

	require.NoError(t, os.Unsetenv("TEST_KEY"))
	assert.Equal(t, time.Duration(0), Duration("TEST_KEY", 0))
	assert.Equal(t, time.Hour, Duration("TEST_KEY", time.Hour))
	require.NoError(t, os.Setenv("TEST_KEY", "1m"))
	assert.Equal(t, time.Minute, Duration("TEST_KEY", time.Hour))

	require.NoError(t, os.Unsetenv("TEST_KEY"))
	assert.Equal(t, 0, Int("TEST_KEY", 0))
	assert.Equal(t, 2137, Int("TEST_KEY", 2137))
	require.NoError(t, os.Setenv("TEST_KEY", "1701"))
	assert.Equal(t, 1701, Int("TEST_KEY", 2137))

	require.NoError(t, os.Unsetenv("TEST_KEY"))
	assert.Equal(t, uint64(0), Uint64("TEST_KEY", 0))
	assert.Equal(t, uint64(2137), Uint64("TEST_KEY", 2137))
	require.NoError(t, os.Setenv("TEST_KEY", "1701"))
	assert.Equal(t, uint64(1701), Uint64("TEST_KEY", 2137))

	require.NoError(t, os.Unsetenv("TEST_KEY"))
	assert.Equal(t, []string{}, Strings("TEST_KEY", []string{}))
	assert.Equal(t, []string{"defaultA", "defaultB"}, Strings("TEST_KEY", []string{"defaultA", "defaultB"}))
	require.NoError(t, os.Setenv("TEST_KEY", "value1\nvalue2"))
	assert.Equal(t, []string{"value1", "value2"}, Strings("TEST_KEY", []string{"defaultA", "defaultB"}))
	require.NoError(t, os.Setenv("ITEM_SEPARATOR", "|"))
	require.NoError(t, os.Setenv("TEST_KEY", "value3|value4"))
	assert.Equal(t, []string{"value3", "value4"}, Strings("TEST_KEY", []string{"defaultA", "defaultB"}))
}
