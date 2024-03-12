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

const testLayout = "2006-01-02 15:04:05"

func TestScheduler(t *testing.T) {
	ctx, cancelCtx := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancelCtx()

	start := time.Now()

	schedule, _ := NewScheduledTicker("* * * * * *")
	scheduler := NewScheduler(schedule, time.UTC)

	scheduler.Start(ctx)

	n := 0
	for n < 3 {
		<-scheduler.TickCh()
		n++
	}

	cancelCtx()

	elapsed := time.Since(start)
	assert.True(t, elapsed >= 2*time.Second)
}

func TestScheduledTicker(t *testing.T) {
	tt := []struct {
		in      string
		at      string
		out     string
		wantErr bool
	}{
		// Asterisks
		{"", "2020-01-01 00:00:00", "", true},
		{"*", "2020-01-01 00:00:00", "2020-01-01 00:00:01", false},
		{"* *", "2020-01-01 00:00:00", "2020-01-01 00:00:01", false},
		{"* * *", "2020-01-01 00:00:00", "2020-01-01 00:00:01", false},
		{"* * * *", "2020-01-01 00:00:00", "2020-01-01 00:00:01", false},
		{"* * * * *", "2020-01-01 00:00:00", "2020-01-01 00:00:01", false},
		{"* * * * * *", "2020-01-01 00:00:00", "2020-01-01 00:00:01", false},
		{"* * * * * * *", "2020-01-01 00:00:00", "", true},

		// Single value
		{"0 * * * * *", "2020-01-01 00:00:00", "2020-01-01 00:01:00", false},
		{"0,30 * * * * *", "2020-01-01 00:00:00", "2020-01-01 00:00:30", false},
		{"* 30 * * * *", "2020-01-01 00:00:01", "2020-01-01 00:30:00", false},
		{"* * 6 * * *", "2020-01-01 00:01:01", "2020-01-01 06:00:00", false},
		{"* * * 15 * *", "2020-01-01 06:30:01", "2020-01-15 00:00:00", false},
		{"* * * * jul *", "2020-01-15 06:30:01", "2020-07-01 00:00:00", false},
		{"* * * * * wed", "2020-01-01 00:00:00", "2020-01-01 00:00:01", false},

		// At the match
		{"0 0 0 1 * *", "2020-01-01 00:00:00", "2020-02-01 00:00:00", false},
		{"0 0 0 1 jan *", "2020-01-01 00:00:00", "2021-01-01 00:00:00", false},
		{"0 0 0 1 jan wed", "2020-01-01 00:00:00", "2025-01-01 00:00:00", false},

		// Just after the match
		{"0 0 0 1 * *", "2020-01-01 00:00:01", "2020-02-01 00:00:00", false},
		{"0 0 0 1 jan *", "2020-01-01 00:00:01", "2021-01-01 00:00:00", false},
		{"0 0 0 1 jan wed", "2020-01-01 00:00:01", "2025-01-01 00:00:00", false},

		// Months
		{"0 0 0 1 jan *", "2020-01-01 00:00:00", "2021-01-01 00:00:00", false},
		{"0 0 0 1 feb *", "2020-01-01 00:00:00", "2020-02-01 00:00:00", false},
		{"0 0 0 1 mar *", "2020-01-01 00:00:00", "2020-03-01 00:00:00", false},
		{"0 0 0 1 apr *", "2020-01-01 00:00:00", "2020-04-01 00:00:00", false},
		{"0 0 0 1 may *", "2020-01-01 00:00:00", "2020-05-01 00:00:00", false},
		{"0 0 0 1 jun *", "2020-01-01 00:00:00", "2020-06-01 00:00:00", false},
		{"0 0 0 1 jul *", "2020-01-01 00:00:00", "2020-07-01 00:00:00", false},
		{"0 0 0 1 aug *", "2020-01-01 00:00:00", "2020-08-01 00:00:00", false},
		{"0 0 0 1 sep *", "2020-01-01 00:00:00", "2020-09-01 00:00:00", false},
		{"0 0 0 1 oct *", "2020-01-01 00:00:00", "2020-10-01 00:00:00", false},
		{"0 0 0 1 nov *", "2020-01-01 00:00:00", "2020-11-01 00:00:00", false},
		{"0 0 0 1 january *", "2020-01-01 00:00:00", "2021-01-01 00:00:00", false},
		{"0 0 0 1 february *", "2020-01-01 00:00:00", "2020-02-01 00:00:00", false},
		{"0 0 0 1 march *", "2020-01-01 00:00:00", "2020-03-01 00:00:00", false},
		{"0 0 0 1 april *", "2020-01-01 00:00:00", "2020-04-01 00:00:00", false},
		{"0 0 0 1 may *", "2020-01-01 00:00:00", "2020-05-01 00:00:00", false},
		{"0 0 0 1 june *", "2020-01-01 00:00:00", "2020-06-01 00:00:00", false},
		{"0 0 0 1 july *", "2020-01-01 00:00:00", "2020-07-01 00:00:00", false},
		{"0 0 0 1 august *", "2020-01-01 00:00:00", "2020-08-01 00:00:00", false},
		{"0 0 0 1 september *", "2020-01-01 00:00:00", "2020-09-01 00:00:00", false},
		{"0 0 0 1 october *", "2020-01-01 00:00:00", "2020-10-01 00:00:00", false},
		{"0 0 0 1 november *", "2020-01-01 00:00:00", "2020-11-01 00:00:00", false},

		// Weekdays
		{"0 0 0 * * mon", "2020-01-01 00:00:00", "2020-01-06 00:00:00", false},
		{"0 0 0 * * tue", "2020-01-01 00:00:00", "2020-01-07 00:00:00", false},
		{"0 0 0 * * wed", "2020-01-01 00:00:00", "2020-01-08 00:00:00", false},
		{"0 0 0 * * thu", "2020-01-01 00:00:00", "2020-01-02 00:00:00", false},
		{"0 0 0 * * fri", "2020-01-01 00:00:00", "2020-01-03 00:00:00", false},
		{"0 0 0 * * sat", "2020-01-01 00:00:00", "2020-01-04 00:00:00", false},
		{"0 0 0 * * sun", "2020-01-01 00:00:00", "2020-01-05 00:00:00", false},
		{"0 0 0 * * monday", "2020-01-01 00:00:00", "2020-01-06 00:00:00", false},
		{"0 0 0 * * tuesday", "2020-01-01 00:00:00", "2020-01-07 00:00:00", false},
		{"0 0 0 * * wednesday", "2020-01-01 00:00:00", "2020-01-08 00:00:00", false},
		{"0 0 0 * * thursday", "2020-01-01 00:00:00", "2020-01-02 00:00:00", false},
		{"0 0 0 * * friday", "2020-01-01 00:00:00", "2020-01-03 00:00:00", false},
		{"0 0 0 * * saturday", "2020-01-01 00:00:00", "2020-01-04 00:00:00", false},
		{"0 0 0 * * sunday", "2020-01-01 00:00:00", "2020-01-05 00:00:00", false},

		// Multiple values
		{"15,45 * * * * *", "2020-01-01 00:00:00", "2020-01-01 00:00:15", false},
		{"0 15,45 * * * *", "2020-01-01 00:00:00", "2020-01-01 00:15:00", false},
		{"0 0 6,12 * * *", "2020-01-01 00:00:00", "2020-01-01 06:00:00", false},
		{"0 0 0 15,30 * *", "2020-01-01 00:00:00", "2020-01-15 00:00:00", false},
		{"0 0 0 * jun,sep *", "2020-01-01 00:00:00", "2020-06-01 00:00:00", false},
		{"0 0 0 * * mon,tue", "2020-01-01 00:00:00", "2020-01-06 00:00:00", false},

		// Reverse order
		{"45,15 * * * * *", "2020-01-01 00:00:00", "2020-01-01 00:00:15", false},
		{"0 45,15 * * * *", "2020-01-01 00:00:00", "2020-01-01 00:15:00", false},
		{"0 0 12,6 * * *", "2020-01-01 00:00:00", "2020-01-01 06:00:00", false},
		{"0 0 0 30,15 * *", "2020-01-01 00:00:00", "2020-01-15 00:00:00", false},
		{"0 0 0 * sep,jun *", "2020-01-01 00:00:00", "2020-06-01 00:00:00", false},
		{"0 0 0 * * tue,mon", "2020-01-01 00:00:00", "2020-01-06 00:00:00", false},

		// Invalid values
		{"-1 * * * * *", "2020-01-01 00:00:00", "", true},
		{"60 * * * * *", "2020-01-01 00:00:00", "", true},
		{"0 -1 * * * *", "2020-01-01 00:00:00", "", true},
		{"0 60 * * * *", "2020-01-01 00:00:00", "", true},
		{"0 0 -1 * * *", "2020-01-01 00:00:00", "", true},
		{"0 0 25 * * *", "2020-01-01 00:00:00", "", true},
		{"0 0 0 -1 * *", "2020-01-01 00:00:00", "", true},
		{"0 0 0 32 * *", "2020-01-01 00:00:00", "", true},
		{"0 0 0 1 0 *", "2020-01-01 00:00:00", "", true},
		{"0 0 0 1 13 *", "2020-01-01 00:00:00", "", true},
		{"0 0 0 1 1 1", "2020-01-01 00:00:00", "", true},
		{"0 0 0 1 1 a", "2020-01-01 00:00:00", "", true},
	}
	for _, tc := range tt {
		t.Run(tc.in, func(t *testing.T) {
			s, err := NewScheduledTicker(tc.in)
			if tc.wantErr {
				assert.NotNil(t, err)
			} else {
				at, _ := time.ParseInLocation(testLayout, tc.at, time.UTC)
				assert.Nil(t, err)
				assert.Equal(t, tc.out, s.Next(at).Format(testLayout))
			}
		})
	}
}

func TestScheduledTicker_String(t *testing.T) {
	tt := []struct {
		in   string
		want string
	}{
		{"* * * * * *", "*s *m *h *d * *"},
		{"0 * * * * *", "0s *m *h *d * *"},
		{"0 0 * * * *", "0s 0m *h *d * *"},
		{"0 0 0 * * *", "0s 0m 0h *d * *"},
		{"0 0 0 1 * *", "0s 0m 0h 1d * *"},
		{"0 0 0 1 jan *", "0s 0m 0h 1d jan *"},
		{"0 0 0 * * mon", "0s 0m 0h *d * mon"},
		{"0,1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,16,17,18,19,20,21,22,23,24,25,26,27,28,29,30,31,32,33,34,35,36,37,38,39,40,41,42,43,44,45,46,47,48,49,50,51,52,53,54,55,56,57,58,59 * * * * *", "*s *m *h *d * *"},
		{"0 0,1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,16,17,18,19,20,21,22,23,24,25,26,27,28,29,30,31,32,33,34,35,36,37,38,39,40,41,42,43,44,45,46,47,48,49,50,51,52,53,54,55,56,57,58,59 * * * *", "0s *m *h *d * *"},
		{"0 0 0,1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,16,17,18,19,20,21,22,23 * * *", "0s 0m *h *d * *"},
		{"0 0 0 1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,16,17,18,19,20,21,22,23,24,25,26,27,28,29,30,31 * *", "0s 0m 0h *d * *"},
		{"0 0 0 1 jan,feb,mar,apr,may,jun,jul,aug,sep,oct,nov,dec *", "0s 0m 0h 1d * *"},
		{"0 0 0 1 january,february,march,april,may,june,july,august,september,october,november,december *", "0s 0m 0h 1d * *"},
		{"0 0 0 * * mon,tue,wed,thu,fri,sat,sun", "0s 0m 0h *d * *"},
		{"0 0 0 * * monday,tuesday,wednesday,thursday,friday,saturday,sunday", "0s 0m 0h *d * *"},
		{"*", "*s *m *h *d * *"},
	}
	for _, tc := range tt {
		t.Run(tc.in, func(t *testing.T) {
			s, _ := NewScheduledTicker(tc.in)
			assert.Equal(t, tc.want, s.String())
		})
	}
}

func TestScheduledTicker_SummerTime(t *testing.T) {
	cet, _ := time.LoadLocation("CET")
	at := time.Date(2020, 10, 25, 1, 0, 0, 0, cet)
	s, _ := NewScheduledTicker("0 0,30 0,1,2,3,4 * * *")
	at = s.Next(at)
	assert.Equal(t, "2020-10-25 01:30:00 +0200 CEST", at.String())
	at = s.Next(at)
	assert.Equal(t, "2020-10-25 02:00:00 +0200 CEST", at.String())
	at = s.Next(at)
	assert.Equal(t, "2020-10-25 02:30:00 +0200 CEST", at.String())
	at = s.Next(at)
	assert.Equal(t, "2020-10-25 02:00:00 +0100 CET", at.String())
	at = s.Next(at)
	assert.Equal(t, "2020-10-25 02:30:00 +0100 CET", at.String())
}
