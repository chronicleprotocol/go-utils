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

package metrics

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"reflect"
	"regexp"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/chronicleprotocol/go-utils/log"
)

type RoundTripFunc func(req *http.Request) *http.Response

func (f RoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req), nil
}

func TestLogger(t *testing.T) {
	type want struct {
		name  string
		value float64
	}
	tests := []struct {
		metrics []Metric
		want    []want
		logs    func(l log.Logger)
	}{
		// MatchMessage test:
		{
			metrics: []Metric{{MatchMessage: regexp.MustCompile("foo"), Name: "test"}},
			want: []want{
				{name: "test", value: 1},
			},
			logs: func(l log.Logger) {
				l.Info("foo")
				l.Info("bar")
			},
		},
		// MatchFields test:
		{
			metrics: []Metric{{MatchFields: map[string]*regexp.Regexp{"key": regexp.MustCompile("foo")}, Name: "test"}},
			want: []want{
				{name: "test", value: 1},
			},
			logs: func(l log.Logger) {
				l.WithField("key", "foo").Info("test")
				l.WithField("key", "bar").Info("test")
			},
		},
		// Value form string:
		{
			metrics: []Metric{{MatchMessage: regexp.MustCompile("foo"), Name: "test", Value: "key"}},
			want: []want{
				{name: "test", value: 1.5},
			},
			logs: func(l log.Logger) {
				l.WithField("key", "1.5").Info("foo")
			},
		},
		// Value from float:
		{
			metrics: []Metric{{MatchMessage: regexp.MustCompile("foo"), Name: "test", Value: "key"}},
			want: []want{
				{name: "test", value: 1.5},
			},
			logs: func(l log.Logger) {
				l.WithField("key", 1.5).Info("foo")
			},
		},
		// Value from time duration as seconds:
		{
			metrics: []Metric{{MatchMessage: regexp.MustCompile("foo"), Name: "test", Value: "key"}},
			want: []want{
				{name: "test", value: 1.5},
			},
			logs: func(l log.Logger) {
				l.WithField("key", time.Millisecond*1500).Info("foo")
			},
		},
		// Replace variables:
		{
			metrics: []Metric{{
				MatchMessage: regexp.MustCompile("foo"),
				Name:         "test.%{key1}.%{key2}.%{key3.a}",
			}},
			want: []want{
				{
					name:  "test.a.42.b",
					value: 1,
				},
			},
			logs: func(l log.Logger) {
				l.WithFields(log.Fields{
					"key1": "a",
					"key2": 42,
					"key3": map[string]string{"a": "b"},
				}).Info("foo")
			},
		},
		// Multiple matches:
		{
			metrics: []Metric{
				{MatchMessage: regexp.MustCompile("foo"), Name: "a"},
				{MatchMessage: regexp.MustCompile("foo"), Name: "b"},
			},
			want: []want{
				{name: "a", value: 1},
				{name: "b", value: 1},
			},
			logs: func(l log.Logger) {
				l.Info("foo")
				l.Info("foo")
			},
		},
		// Replace duplicated values:
		{
			metrics: []Metric{
				{MatchMessage: regexp.MustCompile("foo"), Name: "a"},
			},
			want: []want{
				{name: "a", value: 1},
			},
			logs: func(l log.Logger) {
				l.Info("foo")
				l.Info("foo")
			},
		},
		// Sum duplicated values:
		{
			metrics: []Metric{
				{MatchMessage: regexp.MustCompile("foo"), Name: "a", Aggregate: Sum, WindowLengthMin: 2},
			},
			want: []want{
				{name: "a", value: 2},
			},
			logs: func(l log.Logger) {
				l.Info("foo")
				l.Info("foo")
			},
		},
		// Use lower value:
		{
			metrics: []Metric{
				{MatchMessage: regexp.MustCompile("foo"), Name: "a", Value: "val", Aggregate: Min, WindowLengthMin: 2},
			},
			want: []want{
				{name: "a", value: 1},
			},
			logs: func(l log.Logger) {
				l.WithField("val", 2).Info("foo")
				l.WithField("val", 1).Info("foo")
				l.WithField("val", 2).Info("foo")
			},
		},
		// Use higher value:
		{
			metrics: []Metric{
				{MatchMessage: regexp.MustCompile("foo"), Name: "a", Value: "val", Aggregate: Max, WindowLengthMin: 2},
			},
			want: []want{
				{name: "a", value: 2},
			},
			logs: func(l log.Logger) {
				l.WithField("val", 1).Info("foo")
				l.WithField("val", 2).Info("foo")
				l.WithField("val", 1).Info("foo")
			},
		},
		// Ignore duplicated logs:
		{
			metrics: []Metric{
				{MatchMessage: regexp.MustCompile("foo"), Name: "a", Value: "val", Aggregate: Ignore, WindowLengthMin: 2},
			},
			want: []want{
				{name: "a", value: 1},
			},
			logs: func(l log.Logger) {
				l.WithField("val", 1).Info("foo")
				l.WithField("val", 2).Info("foo")
			},
		},
		// Scale value by 10^2:
		{
			metrics: []Metric{
				{MatchMessage: regexp.MustCompile("foo"), Name: "a", Value: "val", TransformFunc: func(v float64) float64 { return v / math.Pow(10, 2) }},
			},
			want: []want{
				{name: "a", value: 0.01},
			},
			logs: func(l log.Logger) {
				l.WithField("val", 1).Info("foo")
			},
		},
		// Ignore metrics that uses invalid path in a name:
		{
			metrics: []Metric{
				{MatchMessage: regexp.MustCompile("foo"), Name: "test.%{invalid}"},
				{MatchMessage: regexp.MustCompile("foo"), Name: "test.valid"},
			},
			want: []want{
				{name: "test.valid", value: 1},
			},
			logs: func(l log.Logger) {
				l.Info("foo")
			},
		},
		// Ignore tags with that uses invalid path:
		{
			metrics: []Metric{
				{MatchMessage: regexp.MustCompile("foo"), Name: "a"},
			},
			want: []want{
				{name: "a", value: 1},
			},
			logs: func(l log.Logger) {
				l.Info("foo")
			},
		},
		// Test all log types except panics:
		{
			metrics: []Metric{
				{MatchMessage: regexp.MustCompile(".*"), Name: "%{name}"},
			},
			want: []want{
				{name: "debug", value: 1},
				{name: "error", value: 1},
				{name: "info", value: 1},
				{name: "warn", value: 1},
			},
			logs: func(l log.Logger) {
				l.WithField("name", "debug").Debug("debug")
				l.WithField("name", "error").Error("error")
				l.WithField("name", "info").Info("info")
				l.WithField("name", "warn").Warn("warn")
			},
		},
		// Test panic:
		{
			metrics: []Metric{
				{MatchMessage: regexp.MustCompile(".*"), Name: "%{name}"},
			},
			want: []want{
				{name: "panic", value: 1},
			},
			logs: func(l log.Logger) {
				l.WithField("name", "panic").Panic("panic")
			},
		},
	}
	for n, tt := range tests {
		t.Run(fmt.Sprintf("case-%d", n+1), func(t *testing.T) {
			ctx, ctxCancel := context.WithCancel(context.Background())
			defer ctxCancel()
			l, _ := New(log.Debug, Config{
				Metrics:    tt.metrics,
				ListenAddr: ":0",
			})
			if l, ok := l.(log.LoggerService); ok {
				require.NoError(t, l.Start(ctx))
			}

			// Execute logs:
			func() {
				defer func() { recover() }()
				tt.logs(l)
			}()

			// Log metrics:
			r, _ := http.Get(fmt.Sprintf("http://%s/metrics", l.(*logger).httpServer.Addr()))
			require.Equal(t, http.StatusOK, r.StatusCode)
			j, err := io.ReadAll(r.Body)
			defer r.Body.Close()
			require.NoError(t, err)

			// Unmarshal metrics:
			m := &metricJSON{}
			require.NoError(t, json.Unmarshal(j, m))

			// Check if all metrics are present:
			for _, w := range tt.want {
				var found bool
				for n, v := range m.Metrics {
					if n == w.name {
						found = true
						require.Equal(t, w.value, v.Value, "metric value mismatch")
					}
				}
				require.True(t, found, "metric not found")
			}
		})
	}
}

func Test_byPath(t *testing.T) {
	tests := []struct {
		value   any
		path    string
		want    any
		invalid bool
	}{
		{
			value: "test",
			path:  "",
			want:  "test",
		},
		{
			value:   "test",
			path:    "abc",
			invalid: true,
		},
		{
			value: struct {
				Field string
			}{
				Field: "test",
			},
			path: "Field",
			want: "test",
		},
		{
			value: map[string]string{"Key": "test"},
			path:  "Key",
			want:  "test",
		},
		{
			value:   map[int]string{42: "test"},
			path:    "42",
			invalid: true,
		},
		{
			value: []string{"test"},
			path:  "0",
			want:  "test",
		},
		{
			value: struct {
				Field map[string][]int
			}{
				Field: map[string][]int{"Field2": {42}},
			},
			path: "Field.Field2.0",
			want: 42,
		},
	}
	for n, tt := range tests {
		t.Run(fmt.Sprintf("case-%d", n+1), func(t *testing.T) {
			v := byPath(reflect.ValueOf(tt.value), tt.path)
			if tt.invalid {
				assert.False(t, v.IsValid())
			} else {
				assert.Equal(t, tt.want, v.Interface())
			}
		})
	}
}
