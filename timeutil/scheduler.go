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

package timeutil

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/exp/constraints"

	"github.com/chronicleprotocol/go-utils/sliceutil"
)

const (
	secondsInMinute = 60
	minutesInHour   = 60
	hoursInDay      = 24
	daysInWeek      = 7
	maxDaysInMonth  = 31
	monthsInYear    = 12

	// scheduleMaxSleepTime is the maximum sleep time for the scheduler.
	//
	// If the sleep time is greater than this value, the scheduler will
	// sleep for the maximum sleep time and then recalculate the next
	// tick time to ensure that the scheduler is in sync with the wall
	// clock.
	scheduleMaxSleepTime = 15 * time.Minute
)

// NextTick provides the next tick time for the Scheduler.
type NextTick interface {
	// Next returns the next tick time after the given time.
	Next(at time.Time) time.Time
}

// Scheduler ticks at the specified schedule.
type Scheduler struct {
	mu  sync.RWMutex
	ctx context.Context

	n  NextTick
	c  chan time.Time
	tz *time.Location
}

// NewScheduler returns a new Scheduler.
//
// Scheduler guarantees that tick will be sent for every scheduled
// tick, even if the tick channel is blocked for a long enough to miss
// the next scheduled tick.
//
// Timezone is optional, if nil, the local timezone is used.
func NewScheduler(n NextTick, tz *time.Location) *Scheduler {
	if tz == nil {
		tz = time.Local
	}
	return &Scheduler{n: n, c: make(chan time.Time), tz: tz}
}

// Start starts the scheduler.
//
// It panics if the scheduler is already started or the context is nil.
func (s *Scheduler) Start(ctx context.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ctx != nil {
		panic("timeutil.Scheduler: scheduler is already started")
	}
	if ctx == nil {
		panic("timeutil.Scheduler: context is nil")
	}
	s.ctx = ctx
	go s.scheduler()
}

// Tick sends a tick to the ticker channel.
//
// Scheduler must be started before calling this method, otherwise it panics.
func (s *Scheduler) Tick() {
	s.TickAt(time.Now().In(s.tz))
}

// TickAt sends a tick to the ticker channel with the given time.
//
// Scheduler must be started before calling this method, otherwise it panics.
func (s *Scheduler) TickAt(at time.Time) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.ctx == nil || s.ctx.Err() != nil {
		panic("timeutil.Scheduler: scheduler is not started")
	}
	s.c <- at.In(s.tz)
}

// TickCh returns the scheduler channel.
//
// Channel is closed when the context is canceled.
func (s *Scheduler) TickCh() <-chan time.Time {
	return s.c
}

func (s *Scheduler) scheduler() {
	defer close(s.c)
	n := s.n.Next(time.Now().In(s.tz))
	for {
		d := time.Until(n)
		// Ticker uses runtime clock which may drift from the wall clock
		// over time. For this reason, the duration is recalculated every
		// minute to ensure that the ticker is in sync with the wall clock.
		if d >= scheduleMaxSleepTime {
			if !sleep(s.ctx, scheduleMaxSleepTime) {
				return
			}
			continue
		}
		if !sleep(s.ctx, d) {
			return
		}
		s.c <- n
		n = s.n.Next(n.In(s.tz))
	}
}

// ManualTicker ticks only when requested.
//
// Manual ticker CAN NOT be shared between multiple Scheduler instances.
type ManualTicker struct {
	c chan time.Time
}

// NewManualTicker returns a new ManualTicker.
func NewManualTicker() *ManualTicker {
	return &ManualTicker{c: make(chan time.Time)}
}

// Tick sends a tick to the ticker channel.
//
// Tick is blocking until the tick is consumed.
func (t *ManualTicker) Tick() {
	t.c <- time.Now()
}

// TickAt sends a tick to the ticker channel with the given time.
//
// TickAt is blocking until the tick is consumed.
func (t *ManualTicker) TickAt(at time.Time) {
	t.c <- at
}

// Next implements the NextTick interface.
func (t *ManualTicker) Next(_ time.Time) time.Time {
	return <-t.c
}

// IntervalTicker ticks at the specified interval.
//
// Interval ticker can be shared between multiple Scheduler instances.
type IntervalTicker struct {
	interval time.Duration
}

// NewIntervalTicker returns a new IntervalTicker.
func NewIntervalTicker(interval time.Duration) *IntervalTicker {
	return &IntervalTicker{interval: interval}
}

// UnmarshalText implements the encoding.TextUnmarshaler interface.
//
// The duration is parsed using time.ParseDuration.
//
// If a number is provided, it's assumed to be in seconds.
func (t *IntervalTicker) UnmarshalText(text []byte) error {
	if num, err := strconv.ParseFloat(string(text), 64); err == nil {
		t.interval = time.Duration(num) * time.Second
		return nil
	}
	interval, err := time.ParseDuration(string(text))
	if err != nil {
		return err
	}
	t.interval = interval
	return nil
}

// MarshalText implements the encoding.TextMarshaler interface.
func (t IntervalTicker) MarshalText() ([]byte, error) {
	return []byte(t.String()), nil
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (t *IntervalTicker) UnmarshalJSON(data []byte) error {
	spec, err := strconv.Unquote(string(data))
	if err != nil {
		return err
	}
	return t.UnmarshalText([]byte(spec))
}

// MarshalJSON implements the json.Marshaler interface.
func (t IntervalTicker) MarshalJSON() ([]byte, error) {
	return []byte(strconv.Quote(t.String())), nil
}

// String returns the string representation of the interval.
func (t *IntervalTicker) String() string {
	return t.interval.String()
}

// Next implements the NextTick interface.
func (t *IntervalTicker) Next(at time.Time) time.Time {
	return at.Add(t.interval)
}

// ScheduledTicker ticks at the specified schedule.
//
// Scheduled ticker can be shared between multiple Scheduler instances.
type ScheduledTicker struct {
	// seconds, minutes, hours, days, months, weekdays are sorted and unique.
	// If empty or nil, it means every value.

	seconds  []int
	minutes  []int
	hours    []int
	days     []int
	months   []time.Month
	weekdays []time.Weekday
}

// NewScheduledTicker returns a new ScheduledTicker.
//
// The spec is a string representing a schedule in the following format:
//
//	second minute hour day month weekday
//
// Where:
//   - Second and minute can be a number (0-59) or an asterisk (*). Optionally,
//     the suffix "s" or "m" can be added respectively.
//   - Hour can be a number (0-23) or an asterisk (*). Optionally, the suffix
//     "h" can be added.
//   - A day can be a number (1-31) or an asterisk (*). Optionally, the suffix
//     "d" can be added.
//   - The month can be a number (1-12), three-letter abbreviation
//     (jan, feb, ...) or the full name (january, february, ...).
//   - The weekday can be a three-letter abbreviation (sun, mon, ...) or the
//     full name (sunday, monday, ...).
//   - It is possible to specify multiple values separated by a comma.
//   - trailing asterisks can be omitted, except for the asterisk for seconds.
//
// Examples:
//   - *s - every second
//   - 0s *m - every minute
//   - 0s 0m *h *d * * - every hour
//   - 0s 0m 0h *d * * - every day
//   - 0s 0m 0h 1d * * - every month at the first day of the month
//   - 0s 0m 0h 1d jan * - every year at the first day of January
//   - 0s 0m 0h 1d jan wed - every year at the first day of January if it's Wednesday
//   - 0s *m *h 1d - every minute on the first day of the month
//   - 0s 0,15,30,45m - every 15 minutes
func NewScheduledTicker(spec string) (*ScheduledTicker, error) {
	s := &ScheduledTicker{}
	if err := s.parse(spec); err != nil {
		return nil, err
	}
	return s, nil
}

// UnmarshalText implements the encoding.TextUnmarshaler interface.
func (s *ScheduledTicker) UnmarshalText(text []byte) error {
	spec := string(text)
	if err := s.parse(spec); err != nil {
		return err
	}
	return nil
}

// MarshalText implements the encoding.TextMarshaler interface.
func (s ScheduledTicker) MarshalText() ([]byte, error) {
	return []byte(s.String()), nil
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (s *ScheduledTicker) UnmarshalJSON(data []byte) error {
	spec, err := strconv.Unquote(string(data))
	if err != nil {
		return err
	}
	if err := s.parse(spec); err != nil {
		return err
	}
	return nil
}

// MarshalJSON implements the json.Marshaler interface.
func (s ScheduledTicker) MarshalJSON() ([]byte, error) {
	return []byte(strconv.Quote(s.String())), nil
}

// String returns the string representation of the schedule.
func (s *ScheduledTicker) String() string {
	var b strings.Builder
	writeNums(&b, s.seconds)
	b.WriteString("s ")
	writeNums(&b, s.minutes)
	b.WriteString("m ")
	writeNums(&b, s.hours)
	b.WriteString("h ")
	writeNums(&b, s.days)
	b.WriteString("d ")
	writeMonths(&b, s.months)
	b.WriteByte(' ')
	writeWeekdays(&b, s.weekdays)
	return b.String()
}

// Next implements the NextTick interface.
func (s *ScheduledTicker) Next(at time.Time) time.Time {
	at = s.nextSecond(at)
	for {
		if !s.matchMonth(at) {
			at = s.nextMonth(at)
			continue
		}
		if !s.matchWeekday(at) || !s.matchDay(at) {
			at = s.nextDay(at)
			continue
		}
		if !s.matchHour(at) {
			at = s.nextHour(at)
			continue
		}
		if !s.matchMinute(at) {
			at = s.nextMinute(at)
			continue
		}
		if !s.matchSecond(at) {
			at = s.nextSecond(at)
			continue
		}
		return at
	}
}

func (s *ScheduledTicker) nextMonth(at time.Time) time.Time {
	y, m, _ := at.Date()
	if len(s.months) == 0 {
		return time.Date(y+1, 1, 1, 0, 0, 0, 0, at.Location())
	}
	for _, v := range s.months {
		if v > m {
			return time.Date(y, v, 1, 0, 0, 0, 0, at.Location())
		}
	}
	return time.Date(y+1, s.months[0], 1, 0, 0, 0, 0, at.Location())
}

func (s *ScheduledTicker) nextDay(at time.Time) time.Time {
	y, m, d := at.Date()
	if len(s.days) == 0 {
		return time.Date(y, m, d+1, 0, 0, 0, 0, at.Location())
	}
	for _, v := range s.days {
		if v > d {
			return time.Date(y, m, v, 0, 0, 0, 0, at.Location())
		}
	}
	return time.Date(y, m+1, s.days[0], 0, 0, 0, 0, at.Location())
}

func (s *ScheduledTicker) nextHour(at time.Time) time.Time {
	hrs, min, sec := at.Clock()
	off := time.Duration(min)*time.Minute + time.Duration(sec)*time.Second + time.Duration(at.Nanosecond())
	if len(s.hours) == 0 {
		return at.Add(time.Hour - off)
	}
	for _, v := range s.hours {
		if v > hrs {
			return at.Add(time.Duration(v-hrs)*time.Hour - off)
		}
	}
	return at.Add(time.Duration(hoursInDay-hrs+s.hours[0])*time.Hour - off)
}

func (s *ScheduledTicker) nextMinute(at time.Time) time.Time {
	off := time.Duration(at.Second())*time.Second + time.Duration(at.Nanosecond())
	if len(s.minutes) == 0 {
		return at.Add(time.Minute - off)
	}
	min := at.Minute()
	for _, v := range s.minutes {
		if v > min {
			return at.Add(time.Duration(v-min)*time.Minute - off)
		}
	}
	return at.Add(time.Duration(minutesInHour-min+s.minutes[0])*time.Minute - off)
}

func (s *ScheduledTicker) nextSecond(at time.Time) time.Time {
	off := time.Duration(at.Nanosecond())
	if len(s.seconds) == 0 {
		return at.Add(time.Second - off)
	}
	sec := at.Second()
	for _, v := range s.seconds {
		if v > sec {
			return at.Add(time.Duration(v-sec)*time.Second - off)
		}
	}
	return at.Add(time.Duration(secondsInMinute-sec+s.seconds[0])*time.Second - off)
}

func (s *ScheduledTicker) matchSecond(at time.Time) bool {
	if len(s.seconds) == 0 {
		return true
	}
	sec := at.Second()
	for _, v := range s.seconds {
		if v == sec {
			return true
		}
	}
	return false
}

func (s *ScheduledTicker) matchMinute(at time.Time) bool {
	if len(s.minutes) == 0 {
		return true
	}
	min := at.Minute()
	for _, v := range s.minutes {
		if v == min {
			return true
		}
	}
	return false
}

func (s *ScheduledTicker) matchHour(at time.Time) bool {
	if len(s.hours) == 0 {
		return true
	}
	hrs := at.Hour()
	for _, v := range s.hours {
		if v == hrs {
			return true
		}
	}
	return false
}

func (s *ScheduledTicker) matchDay(at time.Time) bool {
	if len(s.days) == 0 {
		return true
	}
	d := at.Day()
	for _, v := range s.days {
		if v == d {
			return true
		}
	}
	return false
}

func (s *ScheduledTicker) matchMonth(at time.Time) bool {
	if len(s.months) == 0 {
		return true
	}
	m := at.Month()
	for _, v := range s.months {
		if v == m {
			return true
		}
	}
	return false
}

func (s *ScheduledTicker) matchWeekday(at time.Time) bool {
	if len(s.weekdays) == 0 {
		return true
	}
	w := at.Weekday()
	for _, v := range s.weekdays {
		if v == w {
			return true
		}
	}
	return false
}

func (s *ScheduledTicker) parse(spec string) (err error) {
	if len(spec) == 0 {
		return fmt.Errorf("timeutil: invalid spec: spec is empty")
	}
	parts := strings.Split(spec, " ")
	if len(parts) > 6 {
		return fmt.Errorf("timeutil: invalid spec: %s", spec)
	}
	if s.seconds, err = parsePart(parts[0], "s", parseSecond); err != nil {
		return err
	}
	if len(parts) >= 2 {
		if s.minutes, err = parsePart(parts[1], "m", parseMinute); err != nil {
			return err
		}
	}
	if len(parts) >= 3 {
		if s.hours, err = parsePart(parts[2], "h", parseHour); err != nil {
			return err
		}
	}
	if len(parts) >= 4 {
		if s.days, err = parsePart(parts[3], "d", parseDay); err != nil {
			return err
		}
	}
	if len(parts) >= 5 {
		if s.months, err = parsePart(parts[4], "", parseMonth); err != nil {
			return err
		}
	}
	if len(parts) >= 6 {
		if s.weekdays, err = parsePart(parts[5], "", parseWeekday); err != nil {
			return err
		}
	}
	if len(s.seconds) == secondsInMinute {
		s.seconds = nil
	}
	if len(s.minutes) == minutesInHour {
		s.minutes = nil
	}
	if len(s.hours) == hoursInDay {
		s.hours = nil
	}
	if len(s.days) == maxDaysInMonth {
		s.days = nil
	}
	if len(s.months) == monthsInYear {
		s.months = nil
	}
	if len(s.weekdays) == daysInWeek {
		s.weekdays = nil
	}
	return nil
}

func parsePart[T constraints.Ordered](part, suffix string, fn func(s string) (T, error)) (nums []T, err error) {
	part = strings.TrimSuffix(part, suffix)
	if part == "*" {
		return nil, nil
	}
	parts := strings.Split(part, ",")
	for _, part := range parts {
		num, err := fn(part)
		if err != nil {
			return nil, err
		}
		nums = append(nums, num)
	}
	if len(nums) == 1 {
		return nums, nil
	}
	sort.Slice(nums, func(i, j int) bool { return nums[i] < nums[j] })
	return sliceutil.Unique(nums), nil
}

func parseSecond(s string) (int, error) {
	i, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("timeutil: invalid second: %w", err)
	}
	if i < 0 || i >= secondsInMinute {
		return 0, fmt.Errorf("timeutil: invalid second: %s", s)
	}
	return i, nil
}

func parseMinute(s string) (int, error) {
	i, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("timeutil: invalid minute: %w", err)
	}
	if i < 0 || i >= minutesInHour {
		return 0, fmt.Errorf("timeutil: invalid minute: %s", s)
	}
	return i, nil
}

func parseHour(s string) (int, error) {
	i, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("timeutil: invalid hour: %w", err)
	}
	if i < 0 || i >= hoursInDay {
		return 0, fmt.Errorf("timeutil: invalid hour: %s", s)
	}
	return i, nil
}

func parseDay(s string) (int, error) {
	i, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("timeutil: invalid day: %w", err)
	}
	if i <= 0 || i > maxDaysInMonth {
		return 0, fmt.Errorf("timeutil: invalid day: %s", s)
	}
	return i, nil
}

func parseMonth(s string) (time.Month, error) {
	switch s {
	case "1", "jan", "january":
		return time.January, nil
	case "2", "feb", "february":
		return time.February, nil
	case "3", "mar", "march":
		return time.March, nil
	case "4", "apr", "april":
		return time.April, nil
	case "5", "may":
		return time.May, nil
	case "6", "jun", "june":
		return time.June, nil
	case "7", "jul", "july":
		return time.July, nil
	case "8", "aug", "august":
		return time.August, nil
	case "9", "sep", "september":
		return time.September, nil
	case "10", "oct", "october":
		return time.October, nil
	case "11", "nov", "november":
		return time.November, nil
	case "12", "dec", "december":
		return time.December, nil
	}
	return 0, fmt.Errorf("timeutil: invalid month: %s", s)
}

func parseWeekday(s string) (time.Weekday, error) {
	switch s {
	case "sun", "sunday":
		return time.Sunday, nil
	case "mon", "monday":
		return time.Monday, nil
	case "tue", "tuesday":
		return time.Tuesday, nil
	case "wed", "wednesday":
		return time.Wednesday, nil
	case "thu", "thursday":
		return time.Thursday, nil
	case "fri", "friday":
		return time.Friday, nil
	case "sat", "saturday":
		return time.Saturday, nil
	}
	return 0, fmt.Errorf("timeutil: invalid weekday: %s", s)
}

func writeNums[T any](b *strings.Builder, nums []T) {
	if len(nums) == 0 {
		b.WriteString("*")
		return
	}
	for i, num := range nums {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(fmt.Sprint(num))
	}
}

func writeWeekdays(b *strings.Builder, weekdays []time.Weekday) {
	if len(weekdays) == 0 {
		b.WriteString("*")
		return
	}
	for i, weekday := range weekdays {
		if i > 0 {
			b.WriteByte(',')
		}
		switch weekday {
		case time.Sunday:
			b.WriteString("sun")
		case time.Monday:
			b.WriteString("mon")
		case time.Tuesday:
			b.WriteString("tue")
		case time.Wednesday:
			b.WriteString("wed")
		case time.Thursday:
			b.WriteString("thu")
		case time.Friday:
			b.WriteString("fri")
		case time.Saturday:
			b.WriteString("sat")
		}
	}
}

func writeMonths(b *strings.Builder, months []time.Month) {
	if len(months) == 0 {
		b.WriteString("*")
		return
	}
	for i, month := range months {
		if i > 0 {
			b.WriteByte(',')
		}
		switch month {
		case time.January:
			b.WriteString("jan")
		case time.February:
			b.WriteString("feb")
		case time.March:
			b.WriteString("mar")
		case time.April:
			b.WriteString("apr")
		case time.May:
			b.WriteString("may")
		case time.June:
			b.WriteString("jun")
		case time.July:
			b.WriteString("jul")
		case time.August:
			b.WriteString("aug")
		case time.September:
			b.WriteString("sep")
		case time.October:
			b.WriteString("oct")
		case time.November:
			b.WriteString("nov")
		case time.December:
			b.WriteString("dec")
		}
	}
}

// sleep sleeps for the given duration or until the context is canceled.
//
// It returns false if the context is canceled before the duration is over.
func sleep(ctx context.Context, d time.Duration) bool {
	if d <= 0 {
		return ctx.Err() == nil
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-t.C:
		return true
	}
}
