package env

import (
	"fmt"
	"os"
	"testing"

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
