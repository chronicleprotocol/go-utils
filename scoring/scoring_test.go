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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/chronicleprotocol/go-utils/timeutil"
)

func TestNew(t *testing.T) {
	decayTicker := timeutil.NewTicker(0)
	decayFunc := func(score float64) float64 { return score * 0.9 }

	s := New[int](decayTicker, decayFunc)
	assert.NotNil(t, s)
}

func TestAddAndGet(t *testing.T) {
	decayTicker := timeutil.NewTicker(0)
	decayFunc := func(score float64) float64 { return score * 0.9 }
	s := New[int](decayTicker, decayFunc)

	t.Run("add and get a score", func(t *testing.T) {
		s.Add(10, 1)
		score, ok := s.Get(1)
		assert.True(t, ok)
		assert.Equal(t, 10.0, score)
	})

	t.Run("get non-existent score", func(t *testing.T) {
		_, ok := s.Get(2)
		assert.False(t, ok)
	})
}

func TestRemove(t *testing.T) {
	decayTicker := timeutil.NewTicker(0)
	decayFunc := func(score float64) float64 { return score * 0.9 }
	s := New[int](decayTicker, decayFunc)
	s.Add(10, 1)

	t.Run("remove existing item", func(t *testing.T) {
		s.Remove(1)
		_, ok := s.Get(1)
		assert.False(t, ok)
	})

	t.Run("remove non-existent item", func(t *testing.T) {
		s.Remove(2) // no error should be thrown
		_, ok := s.Get(2)
		assert.False(t, ok)
	})
}

func TestIncreaseAndDecrease(t *testing.T) {
	decayTicker := timeutil.NewTicker(0)
	decayFunc := func(score float64) float64 { return score * 0.9 }
	s := New[int](decayTicker, decayFunc)
	s.Add(10, 1)

	t.Run("increase score", func(t *testing.T) {
		s.Increase(1, 5)
		score, _ := s.Get(1)
		assert.Equal(t, 15.0, score)
	})

	t.Run("decrease score", func(t *testing.T) {
		s.Decrease(1, 3)
		score, _ := s.Get(1)
		assert.Equal(t, 12.0, score)
	})

	t.Run("increase non-existent score", func(t *testing.T) {
		s.Increase(2, 5)
		_, ok := s.Get(2)
		assert.False(t, ok)
	})

	t.Run("decrease non-existent score", func(t *testing.T) {
		s.Decrease(2, 5)
		_, ok := s.Get(2)
		assert.False(t, ok)
	})
}

func TestSorted(t *testing.T) {
	decayTicker := timeutil.NewTicker(0)
	decayFunc := func(score float64) float64 { return score * 0.9 }
	s := New[int](decayTicker, decayFunc)

	s.Add(10, 1)
	s.Add(20, 2)
	s.Add(15, 3)

	sorted := s.Sorted()
	assert.Equal(t, []int{2, 3, 1}, sorted)
}

func TestSorted_RandomIfSameScore(t *testing.T) {
	decayTicker := timeutil.NewTicker(0)
	decayFunc := func(score float64) float64 { return score * 0.9 }
	s := New[int](decayTicker, decayFunc)
	require.NoError(t, s.Start(context.Background()))

	s.Add(10, 1)
	s.Add(10, 2)
	s.Add(10, 3)
	m := make(map[int]int)
	for i := 0; i < 1000; i++ {
		decayTicker.Tick()
		sorted := s.Sorted()
		m[sorted[0]+sorted[1]*10+sorted[2]*100]++
	}
	assert.Len(t, m, 6)
}

func TestStartScoring(t *testing.T) {
	decayTicker := timeutil.NewTicker(0)
	decayFunc := func(score float64) float64 { return score * 0.9 }

	s := New[int](decayTicker, decayFunc)
	err := s.Start(context.Background())
	assert.NoError(t, err)

	t.Run("start already started", func(t *testing.T) {
		err := s.Start(context.Background())
		assert.Error(t, err)
	})

	t.Run("decay", func(t *testing.T) {
		s.Add(10, 1)
		s.Add(20, 2)
		s.Add(15, 3)

		s1, _ := s.Get(1)
		s2, _ := s.Get(2)
		s3, _ := s.Get(3)
		assert.Equal(t, 10.0, s1)
		assert.Equal(t, 20.0, s2)
		assert.Equal(t, 15.0, s3)

		decayTicker.Tick()

		assert.Eventually(t, func() bool {
			s1, _ = s.Get(1)
			s2, _ = s.Get(2)
			s3, _ = s.Get(3)
			return s1 == 9.0 && s2 == 18.0 && s3 == 13.5
		}, 100*time.Millisecond, 10*time.Millisecond)
	})
}
