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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTicker(t *testing.T) {
	ctx, cancelCtx := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancelCtx()

	start := time.Now()

	ticker := NewTicker(10 * time.Millisecond)
	ticker.Start(ctx)

	n := 0
	for n < 10 {
		<-ticker.TickCh()
		n++
	}

	cancelCtx()

	elapsed := time.Since(start)
	assert.True(t, elapsed >= 100*time.Millisecond)
}

func TestVarTicker_Tick(t *testing.T) {
	ctx, cancelCtx := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancelCtx()

	start := time.Now()

	ticker := NewVarTicker(10*time.Millisecond, 10*time.Millisecond)
	ticker.Start(ctx)

	n := 0
	for n < 10 {
		<-ticker.TickCh()
		n++
	}

	cancelCtx()

	elapsed := time.Since(start)
	assert.True(t, elapsed >= 100*time.Millisecond)
}
