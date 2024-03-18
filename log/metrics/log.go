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
	"net/http"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/chronicleprotocol/go-utils/dump"
	"github.com/chronicleprotocol/go-utils/httpserver"
	"github.com/chronicleprotocol/go-utils/interpolate"
	"github.com/chronicleprotocol/go-utils/log/null"
	"github.com/chronicleprotocol/go-utils/maputil"

	"github.com/chronicleprotocol/go-utils/log"
)

const LoggerTag = "METRICS"

const httpServerTimeout = 100 * time.Millisecond

// Config is the configuration for the Grafana logger.
type Config struct {
	// Metrics is a list of metric definitions.
	Metrics []Metric

	// Listen address for the HTTP server that will be serving metrics.
	ListenAddr string

	// Logger used to log errors related to this logger, such as connection errors.
	Logger log.Logger
}

// Metric describes one Grafana metric.
type Metric struct {
	// MatchMessage is a regexp that must match the log message.
	MatchMessage *regexp.Regexp

	// MatchFields is a list of regexp's that must match the values of the
	// fields defined in the map keys.
	MatchFields map[string]*regexp.Regexp

	// Value is the dot-separated path of the field with the metric value.
	// If empty, the value 1 will be used as the metric value.
	Value string

	// Name is the name of the metric. It can contain references to log fields
	// in the format ${path}, where path is the dot-separated path to the field.
	Name string

	// WindowLengthMin is the length of the window in minutes. If 0, the last
	// value is always used.
	WindowLengthMin uint

	// Aggregate specifies how to aggregate values in the time window.
	Aggregate Aggregate

	// TransformFunc defines the function applied to the value before setting it.
	TransformFunc func(float64) float64

	// ParserFunc is going to be applied to transform the value reflection to an actual float64 value
	ParserFunc func(reflect.Value) (float64, bool)

	parsedName interpolate.Parsed
}

// New creates a new logger that can extract parameters from log messages and
// send them to Grafana Cloud using the Graphite endpoint. It starts
// a background goroutine that will be sending metrics to the Grafana Cloud
// server as often as described in Config.Interval parameter.
func New(level log.Level, cfg Config) (log.Logger, error) {
	if cfg.Logger == nil {
		cfg.Logger = null.New()
	}
	// Parse names in advance to improve performance.
	for n := range cfg.Metrics {
		m := &cfg.Metrics[n]
		m.parsedName = interpolate.ParsePercent(m.Name)
	}
	l := &logger{
		shared: &shared{
			metrics:      cfg.Metrics,
			logger:       cfg.Logger.WithField("tag", LoggerTag),
			metricPoints: make(map[string]*metricAggregated),
			httpServer: httpserver.New(&http.Server{
				Addr:              cfg.ListenAddr,
				ReadTimeout:       httpServerTimeout,
				ReadHeaderTimeout: httpServerTimeout,
				WriteTimeout:      httpServerTimeout,
				IdleTimeout:       httpServerTimeout,
			}),
		},
		level:  level,
		fields: log.Fields{},
	}
	return l, nil
}

type logger struct {
	*shared

	level  log.Level
	fields log.Fields
}

type shared struct {
	mu  sync.Mutex
	ctx context.Context

	logger       log.Logger
	metrics      []Metric
	metricPoints map[string]*metricAggregated
	httpServer   *httpserver.HTTPServer
}

type Aggregate int

const (
	Replace Aggregate = iota // Replace the current value.
	Ignore                   // Use previous value.
	Sum                      // Add to the current value.
	Max                      // Use higher value.
	Min                      // Use lower value.
)

func (o Aggregate) aggregate(a, b float64) float64 {
	switch o {
	case Replace:
		return b
	case Ignore:
		return a
	case Sum:
		return a + b
	case Max:
		if a > b {
			return a
		}
		return b
	case Min:
		if a < b {
			return a
		}
		return b
	default:
		return b
	}
}

// Level implements the log.Logger interface.
func (c *logger) Level() log.Level {
	return c.level
}

// WithField implements the log.Logger interface.
func (c *logger) WithField(key string, value any) log.Logger {
	f := log.Fields{}
	for k, v := range c.fields {
		f[k] = v
	}
	f[key] = value
	return &logger{
		shared: c.shared,
		level:  c.level,
		fields: f,
	}
}

// WithFields implements the log.Logger interface.
func (c *logger) WithFields(fields log.Fields) log.Logger {
	f := log.Fields{}
	for k, v := range c.fields {
		f[k] = v
	}
	for k, v := range fields {
		f[k] = v
	}
	return &logger{
		shared: c.shared,
		level:  c.level,
		fields: f,
	}
}

// WithError implements the log.Logger interface.
func (c *logger) WithError(err error) log.Logger {
	return c.WithField("err", err.Error())
}

// WithAdvice implements the log.Logger interface.
func (c *logger) WithAdvice(advice string) log.Logger {
	return c.WithField("advice", advice)
}

// Debug implements the log.Logger interface.
func (c *logger) Debug(args ...any) {
	if c.level >= log.Debug {
		c.collect(fmt.Sprint(args...), c.fields)
	}
}

// Info implements the log.Logger interface.
func (c *logger) Info(args ...any) {
	if c.level >= log.Info {
		c.collect(fmt.Sprint(args...), c.fields)
	}
}

// Warn implements the log.Logger interface.
func (c *logger) Warn(args ...any) {
	if c.level >= log.Warn {
		c.collect(fmt.Sprint(args...), c.fields)
	}
}

// Error implements the log.Logger interface.
func (c *logger) Error(args ...any) {
	if c.level >= log.Error {
		c.collect(fmt.Sprint(args...), c.fields)
	}
}

// Panic implements the log.Logger interface.
func (c *logger) Panic(args ...any) {
	if c.level >= log.Error {
		c.collect(fmt.Sprint(args...), c.fields)
	}
	panic(fmt.Sprint(args...))
}

// Start implements the supervisor.Service interface.
func (c *logger) Start(ctx context.Context) error {
	c.logger.Debug("Starting")
	if c.ctx != nil {
		return fmt.Errorf("service can be started only once")
	}
	if ctx == nil {
		return fmt.Errorf("context is nil")
	}
	c.ctx = ctx
	if err := c.httpServer.Start(ctx); err != nil {
		return fmt.Errorf("failed to start HTTP server: %w", err)
	}
	c.httpServer.SetHandler("/metrics", http.HandlerFunc(c.handle))
	return nil
}

// Wait implements the supervisor.Service interface.
func (c *logger) Wait() <-chan error {
	return c.httpServer.Wait()
}

// collect checks if a log matches any of predefined metrics and if so,
// extracts a metric value from it.
func (c *logger) collect(msg string, fields log.Fields) {
	c.mu.Lock()
	defer c.mu.Unlock()

	rfields := reflect.ValueOf(fields)
	for _, metric := range c.metrics {
		if !match(metric, msg, rfields) {
			continue
		}
		var (
			ok bool
			mk string
			mp metricPoint
		)
		mp.value = 1
		mp.time = time.Now()
		mk, ok = replaceVars(metric.parsedName, rfields)
		if !ok {
			c.logger.
				WithField("path", metric.Name).
				Warn("Invalid path in the name field")
			continue
		}
		if len(metric.Value) > 0 {
			value := byPath(rfields, metric.Value)
			if metric.ParserFunc != nil {
				mp.value, ok = metric.ParserFunc(value)
			} else {
				mp.value, ok = toFloat(value)
			}
			if !ok {
				c.logger.
					WithField("path", metric.Value).
					WithField("value", value).
					Warn("There is no such field or it is not a number")
				continue
			}
		}
		if metric.TransformFunc != nil {
			mp.value = metric.TransformFunc(mp.value)
		}
		if c.metricPoints[mk] == nil {
			c.metricPoints[mk] = &metricAggregated{
				Aggregate:       metric.Aggregate,
				windowLengthMin: metric.WindowLengthMin,
			}
		}
		c.metricPoints[mk].add(time.Now(), mp)
		c.logger.
			WithFields(log.Fields{
				"name":  mk,
				"value": mp.value,
			}).
			Debug("New metric point")
	}
}

// handler returns an HTTP handler that serves metrics in JSON format.
func (c *logger) handle(w http.ResponseWriter, _ *http.Request) {
	c.mu.Lock()
	defer c.mu.Unlock()

	t := time.Now()
	m := &metricJSON{Metrics: make(map[string]metricPointJSON)}
	for _, mk := range maputil.SortedKeys(c.metricPoints, sort.Strings) {
		mp := c.metricPoints[mk].get(t)
		m.Metrics[mk] = metricPointJSON{
			Value: mp.value,
			Time:  mp.time.Format(time.RFC3339),
		}
	}
	j, err := json.Marshal(m)
	if err != nil {
		c.logger.
			WithError(err).
			WithAdvice("This is a bug and must be investigated").
			Error("Failed to marshal metrics")

		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if _, err := w.Write(j); err != nil {
		c.logger.
			WithError(err).
			WithAdvice("This is a bug and must be investigated").
			Error("Failed to write metrics")
	}
}

// match checks if log message and log fields matches metric definition.
func match(metric Metric, msg string, fields reflect.Value) bool {
	for path, rx := range metric.MatchFields {
		field, ok := toString(byPath(fields, path))
		if !ok || !rx.MatchString(field) {
			return false
		}
	}
	return metric.MatchMessage == nil || metric.MatchMessage.MatchString(msg)
}

// replaceVars replaces vars provided as ${field} with values from log fields.
func replaceVars(s interpolate.Parsed, fields reflect.Value) (string, bool) {
	valid := true
	return s.Interpolate(func(v interpolate.Variable) string {
		name, ok := toString(byPath(fields, v.Name))
		if !ok {
			if v.HasDefault {
				return v.Default
			}
			valid = false
			return ""
		}
		return name
	}), valid
}

func byPath(value reflect.Value, path string) reflect.Value {
	if value.Kind() == reflect.Interface || value.Kind() == reflect.Ptr {
		return byPath(value.Elem(), path)
	}
	if len(path) == 0 {
		return value
	}
	switch value.Kind() {
	case reflect.Slice:
		elem, path := splitPath(path)
		i, err := strconv.Atoi(elem)
		if err != nil {
			return reflect.Value{}
		}
		f := value.Index(i)
		if !f.IsValid() {
			return reflect.Value{}
		}
		return byPath(f, path)
	case reflect.Map:
		elem, path := splitPath(path)
		if value.Type().Key().Kind() != reflect.String {
			return reflect.Value{}
		}
		f := value.MapIndex(reflect.ValueOf(elem))
		if !f.IsValid() {
			return reflect.Value{}
		}
		return byPath(f, path)
	case reflect.Struct:
		elem, path := splitPath(path)
		f := value.FieldByName(elem)
		if !f.IsValid() {
			return reflect.Value{}
		}
		return byPath(f, path)
	default:
		return reflect.Value{}
	}
}

func splitPath(path string) (a, b string) {
	p := strings.SplitN(path, ".", 2)
	switch len(p) {
	case 1:
		return p[0], ""
	case 2:
		return p[0], p[1]
	default:
		return "", ""
	}
}

func toFloat(value reflect.Value) (float64, bool) {
	if !value.IsValid() {
		return 0, false
	}
	if t, ok := value.Interface().(time.Duration); ok {
		return t.Seconds(), true
	}
	switch value.Type().Kind() {
	case reflect.String:
		f, err := strconv.ParseFloat(value.String(), 64)
		return f, err == nil
	case reflect.Float32, reflect.Float64:
		return value.Float(), true
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return float64(value.Int()), true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return float64(value.Uint()), true
	default:
		return 0, false
	}
}

func toString(value reflect.Value) (string, bool) {
	if !value.IsValid() {
		return "", false
	}
	return fmt.Sprint(dump.Dump(value.Interface())), true
}

type metricJSON struct {
	Metrics map[string]metricPointJSON `json:"metrics"`
}

type metricPointJSON struct {
	Value float64 `json:"value"`
	Time  string  `json:"time"`
}

type metricPoint struct {
	value float64
	time  time.Time
}

type metricAggregated struct {
	values          []metricPoint
	Aggregate       Aggregate
	windowLengthMin uint
}

func (m *metricAggregated) add(now time.Time, mp metricPoint) {
	if m.windowLengthMin == 0 {
		m.values = []metricPoint{mp}
		return
	}
	if len(m.values) != int(m.windowLengthMin) {
		m.values = make([]metricPoint, m.windowLengthMin)
	}
	s := mp.time.Unix() / 60 % int64(m.windowLengthMin)
	t := now.Add(time.Duration(-m.windowLengthMin) * time.Minute)
	if mp.time.Before(t) {
		return
	}
	if m.values[s].time.Before(t) {
		m.values[s] = mp
		return
	}
	if m.values[s].time.Before(mp.time) {
		m.values[s].time = mp.time
	}
	m.values[s].value = m.Aggregate.aggregate(m.values[s].value, mp.value)
}

func (m *metricAggregated) get(now time.Time) (mp metricPoint) {
	if len(m.values) == 0 {
		return mp
	}
	if m.windowLengthMin == 0 {
		return m.values[0]
	}
	t := now.Add(time.Duration(-m.windowLengthMin) * time.Minute)
	for i := len(m.values) - 1; i >= 0; i-- {
		if m.values[i].time.Before(t) {
			continue
		}
		if mp.time.IsZero() {
			mp = m.values[i]
			continue
		}
		if m.values[i].time.After(mp.time) {
			mp.time = m.values[i].time
		}
		mp.value = m.Aggregate.aggregate(mp.value, m.values[i].value)
	}
	return mp
}
