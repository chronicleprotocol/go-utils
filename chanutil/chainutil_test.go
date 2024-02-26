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

package chanutil

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFanIn(t *testing.T) {
	in1 := make(chan int)
	in2 := make(chan int)
	in3 := make(chan int)
	out := make(chan int)

	fi := NewFanIn(out)
	fi.Input(in1, in2, in3)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		var v []int

		for i := 0; i < 3; i++ {
			v = append(v, <-out)
		}
		assert.ElementsMatch(t, []int{1, 2, 3}, v)
		wg.Done()
	}()

	in1 <- 1
	in2 <- 2
	in3 <- 3

	close(in1)
	close(in2)
	close(in3)

	fi.Wait()
	fi.Close()

	// The output channel should be closed after closing all the input
	// channels.
	_, ok := <-out
	assert.False(t, ok)

	// Wait for the goroutine to finish.
	wg.Wait()
}

func TestFanIn_AutoClose(t *testing.T) {
	in1 := make(chan int)
	in2 := make(chan int)
	in3 := make(chan int)
	out := make(chan int)

	fi := NewFanIn(out)
	fi.Input(in1, in2, in3)

	close(in1)
	close(in2)
	close(in3)

	fi.AutoClose()

	// The output channel should be automatically closed because all input
	// channels are closed.
	_, ok := <-out
	assert.False(t, ok)
	fi.Close()
}

func TestFanIn_AutoClose_NoInputs(t *testing.T) {
	out := make(chan int)

	fi := NewFanIn(out)
	fi.AutoClose()

	time.Sleep(10 * time.Millisecond)

	// The output channel should be automatically closed because there are no
	// input channels.
	_, ok := <-out
	assert.False(t, ok)
	fi.Close()
}

func TestFanOut(t *testing.T) {
	in := make(chan int)
	fo := NewFanOut(in)

	out1 := fo.Output()
	out2 := fo.Output()
	out3 := fo.Output()

	go func() {
		in <- 1
		in <- 2
		in <- 3
		close(in)
	}()

	var mu sync.Mutex
	var wg sync.WaitGroup
	var vs [][]int
	for i, ch := range []<-chan int{out1, out2, out3} {
		wg.Add(1)
		mu.Lock()
		vs = append(vs, []int{})
		mu.Unlock()
		go func(ch <-chan int, i int) {
			for v := range ch {
				mu.Lock()
				vs[i] = append(vs[i], v)
				mu.Unlock()
			}
			wg.Done()
		}(ch, i)
	}

	// Wait for all the output channels to be closed. This should happen after
	// closing the input channel.
	wg.Wait()

	// Any of the output channels should receive copy of all the values sent to
	// the input channel.
	assert.ElementsMatch(t, []int{1, 2, 3}, vs[0])
	assert.ElementsMatch(t, []int{1, 2, 3}, vs[1])
	assert.ElementsMatch(t, []int{1, 2, 3}, vs[2])

	// Any output channel should be closed after closing the input channel.
	_, ok := <-fo.Output()
	assert.False(t, ok)
}

func TestFanOut_Chan_ChanContext(t *testing.T) {
	in := make(chan int)
	fo := NewFanOut(in)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	out := fo.OutputContext(ctx)
	cancel()

	// Channel should be closed after the context is canceled.
	_, ok := <-out
	assert.False(t, ok)

	// Sending to the input channel should not block after the output channel is
	// closed.
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		select {
		case in <- 1:
		case <-time.After(100 * time.Millisecond):
			assert.Fail(t, "sending to the input channel should not block after the output channel is closed")
		}
		wg.Done()
	}()
	wg.Wait()
}
