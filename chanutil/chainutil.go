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
)

// FanIn is a fan-in channel multiplexer. It takes multiple input channels
// and merges them into a single output channel. The implementation is
// thread-safe.
type FanIn[T any] struct {
	mu     sync.Mutex
	wg     sync.WaitGroup
	once   sync.Once
	closed bool
	out    chan T
}

// NewFanIn creates a new FanIn instance.
func NewFanIn[T any](output chan T) *FanIn[T] {
	return &FanIn[T]{out: output}
}

// Input adds a new input channel.
// If the fan-in is already closed, this method panics.
func (fi *FanIn[T]) Input(chs ...<-chan T) {
	fi.mu.Lock()
	defer fi.mu.Unlock()
	if fi.closed {
		panic("input channels cannot be added to a closed fan-in")
	}
	for _, ch := range chs {
		fi.wg.Add(1)
		go fi.fanInRoutine(ch)
	}
}

// Wait blocks until all the input channels are closed.
func (fi *FanIn[T]) Wait() {
	fi.wg.Wait()
}

// Close closes the output channel. This method must be called only after all
// input channels are closed. Otherwise, the code may panic due to sending to a
// closed channel. To make sure that all input channels are closed, a call
// to this method can be preceded by a call to the Wait method. Alternatively,
// the AutoClose method can be used.
func (fi *FanIn[T]) Close() {
	fi.mu.Lock()
	defer fi.mu.Unlock()
	if fi.closed {
		return
	}
	close(fi.out)
	fi.closed = true
}

// AutoClose will automatically close the output channel when all input
// channels are closed. This method must be called only after at least one
// input channel has been added. Otherwise, it will immediately close the
// output channel. This method is idempotent and non-blocking.
func (fi *FanIn[T]) AutoClose() {
	fi.once.Do(func() {
		go func() {
			fi.Wait()
			fi.Close()
		}()
	})
}

func (fi *FanIn[T]) fanInRoutine(ch <-chan T) {
	for v := range ch {
		fi.out <- v
	}
	fi.wg.Done()
}

// FanOut is a fan-out channel demultiplexer. It takes a single input channel
// and distributes its values to multiple output channels. The input channel
// is emptied even if there are no output channels. Output channels are closed
// when the input channel is closed. The implementation is thread-safe.
type FanOut[T any] struct {
	mu  sync.Mutex
	in  <-chan T
	out map[chan T]context.Context
}

// NewFanOut creates a new FanOut instance.
func NewFanOut[T any](input <-chan T) *FanOut[T] {
	fo := &FanOut[T]{in: input, out: make(map[chan T]context.Context)}
	go fo.fanOutRoutine()
	return fo
}

// Output returns a new output channel.
func (fo *FanOut[T]) Output() <-chan T {
	return fo.OutputContext(nil) //nolint:staticcheck
}

// OutputContext returns a new output channel.
// The channel is closed when the given context is canceled.
func (fo *FanOut[T]) OutputContext(ctx context.Context) <-chan T {
	fo.mu.Lock()
	defer fo.mu.Unlock()
	ch := make(chan T)
	if fo.in == nil {
		// If the input channel is already closed, the output channel is closed
		// immediately.
		close(ch)
		return ch
	}
	if ctx != nil {
		go fo.closeRoutine(ctx, ch)
	}
	fo.out[ch] = ctx
	return ch
}

// Close stops sending values to the given output channel and closes it.
//
// If the output channel is already closed or it was not created by the
// Output or OutputContext method, this method does nothing.
func (fo *FanOut[T]) Close(ch <-chan T) {
	fo.mu.Lock()
	defer fo.mu.Unlock()
	for out := range fo.out {
		if ch == out {
			delete(fo.out, out)
			close(out)
			break
		}
	}
}

func (fo *FanOut[T]) closeRoutine(ctx context.Context, ch <-chan T) {
	<-ctx.Done()
	fo.Close(ch)
}

func (fo *FanOut[T]) fanOutRoutine() {
	for v := range fo.in {
		fo.mu.Lock()
		for ch := range fo.out {
			ch <- v
		}
		fo.mu.Unlock()
	}
	fo.mu.Lock()
	defer fo.mu.Unlock()
	for ch := range fo.out {
		close(ch)
	}
	// Remove references to the output channels to help the garbage collector
	// to free the memory. These channels are inaccessible at this point.
	fo.in = nil
	fo.out = nil
}
