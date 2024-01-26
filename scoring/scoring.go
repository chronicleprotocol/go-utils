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

package scoring

import (
	"context"
	"errors"
	"math/rand"
	"sort"
	"sync"

	"github.com/chronicleprotocol/suite/pkg/util/maputil"
	"github.com/chronicleprotocol/suite/pkg/util/timeutil"
)

// Scoring is a simple scoring system that allows to track scores for a set of
// items.
//
// Scoring is thread-safe.
type Scoring[T comparable] struct {
	mu  sync.Mutex
	ctx context.Context

	items       map[T]float64
	decayTicker *timeutil.Ticker
	decayFunc   func(float64) float64
}

// New creates a new Scoring instance.
func New[T comparable](decayTicker *timeutil.Ticker, decayFunc func(float64) float64) *Scoring[T] {
	return &Scoring[T]{
		items:       make(map[T]float64),
		decayTicker: decayTicker,
		decayFunc:   decayFunc,
	}
}

// Start starts the decay routine.
func (s *Scoring[T]) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.ctx != nil {
		return errors.New("already started")
	}
	s.ctx = ctx
	s.decayTicker.Start(s.ctx)
	go s.decayRoutine()
	return nil
}

// Add adds the given values to the scoring system with the given score.
// If the item already exists, it is ignored.
func (s *Scoring[T]) Add(score float64, values ...T) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, value := range values {
		if _, ok := s.items[value]; ok {
			continue
		}
		s.items[value] = score
	}
}

// Remove removes the given values from the scoring system.
func (s *Scoring[T]) Remove(values ...T) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, value := range values {
		delete(s.items, value)
	}
}

// Get returns the score for the given value.
func (s *Scoring[T]) Get(value T) (score float64, ok bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	score, ok = s.items[value]
	return
}

// Sorted returns the items sorted by score.
func (s *Scoring[T]) Sorted() []T {
	s.mu.Lock()
	defer s.mu.Unlock()

	sorted := maputil.Keys(s.items)
	sort.Slice(sorted, func(i, j int) bool {
		if s.items[sorted[i]] == s.items[sorted[j]] {
			return rand.Intn(2) == 0 //nolint:gosec
		}
		return s.items[sorted[i]] > s.items[sorted[j]]
	})
	return sorted
}

// Increase increases the score for the given value by the given amount.
func (s *Scoring[T]) Increase(value T, score float64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.items[value]; !ok {
		return
	}
	s.items[value] += score
}

// Decrease decreases the score for the given value by the given amount.
func (s *Scoring[T]) Decrease(value T, score float64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.items[value]; !ok {
		return
	}
	s.items[value] -= score
}

func (s *Scoring[T]) decay() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i := range s.items {
		s.items[i] = s.decayFunc(s.items[i])
	}
}

func (s *Scoring[T]) decayRoutine() {
	for {
		select {
		case <-s.ctx.Done():
			return
		case <-s.decayTicker.TickCh():
			s.decay()
		}
	}
}
