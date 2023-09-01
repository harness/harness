// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package job

import (
	"testing"
	"time"
)

func TestSchedulerTimer_ResetAt(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name string
		at   time.Time
		exp  time.Duration
	}{
		{
			name: "zero",
			at:   time.Time{},
			exp:  timerMinDur,
		},
		{
			name: "immediate",
			at:   now,
			exp:  timerMinDur,
		},
		{
			name: "30s",
			at:   now.Add(30 * time.Second),
			exp:  30 * time.Second,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			timer := newSchedulerTimer()
			dur := timer.resetAt(now, test.at, false)
			if want, got := test.exp, dur; want != dur {
				t.Errorf("want: %s, got: %s", want.String(), got.String())
			}
		})
	}
}

func TestSchedulerTimer_TryResetAt(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name string
		at   time.Time
		edgy bool
		exp  time.Duration
	}{
		{
			name: "past",
			at:   now.Add(-time.Second),
			exp:  timerMinDur,
		},
		{
			name: "30s",
			at:   now.Add(30 * time.Second),
			exp:  30 * time.Second,
		},
		{
			name: "90s",
			at:   now.Add(90 * time.Second),
			exp:  0,
		},
		{
			name: "30s-edgy",
			at:   now.Add(30 * time.Second),
			edgy: true,
			exp:  timerMinDur,
		},
		{
			name: "90s-edgy",
			at:   now.Add(90 * time.Second),
			edgy: true,
			exp:  timerMinDur,
		},
		{
			name: "zero",
			at:   time.Time{},
			exp:  0,
		},
		{
			name: "zero-edgy",
			at:   time.Time{},
			edgy: true,
			exp:  timerMinDur,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			timer := newSchedulerTimer()
			timer.resetAt(now, now.Add(time.Minute), test.edgy)
			dur := timer.rescheduleEarlier(now, test.at)
			if want, got := test.exp, dur; want != dur {
				t.Errorf("want: %s, got: %s", want.String(), got.String())
			}
		})
	}
}
