// Copyright 2023 Harness, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package job

import (
	"time"
)

const timerMaxDur = 30 * time.Minute
const timerMinDur = time.Nanosecond

type schedulerTimer struct {
	timerAt time.Time
	timer   *time.Timer
	edgy    bool // if true, the next RescheduleEarlier call will trigger the timer immediately.
}

// newSchedulerTimer created new timer for the Scheduler. It is created to fire immediately.
func newSchedulerTimer() *schedulerTimer {
	return &schedulerTimer{
		timerAt: time.Now().Add(timerMinDur),
		timer:   time.NewTimer(timerMinDur),
	}
}

// ResetAt resets the internal timer to trigger at the provided time.
// If the provided time is zero, it will schedule it to after the max duration.
func (t *schedulerTimer) ResetAt(next time.Time, edgy bool) time.Duration {
	return t.resetAt(time.Now(), next, edgy)
}

func (t *schedulerTimer) resetAt(now, next time.Time, edgy bool) time.Duration {
	var dur time.Duration

	dur = next.Sub(now)
	if dur < timerMinDur {
		dur = timerMinDur
		next = now.Add(dur)
	} else if dur > timerMaxDur {
		dur = timerMaxDur
		next = now.Add(dur)
	}

	t.Stop()
	t.edgy = edgy
	t.timerAt = next
	t.timer.Reset(dur)

	return dur
}

// RescheduleEarlier will reset the timer if the new time is earlier than the previous time.
// Otherwise, the function does nothing and returns 0.
// Providing zero time triggers the timer if it's edgy, otherwise does nothing.
func (t *schedulerTimer) RescheduleEarlier(next time.Time) time.Duration {
	return t.rescheduleEarlier(time.Now(), next)
}

func (t *schedulerTimer) rescheduleEarlier(now, next time.Time) time.Duration {
	var dur time.Duration

	switch {
	case t.edgy:
		// if the timer is edgy trigger it immediately
		dur = timerMinDur

	case next.IsZero():
		// if the provided time is zero: trigger the timer if it's edgy otherwise do nothing
		if !t.edgy {
			return 0
		}
		dur = timerMinDur

	case !next.Before(t.timerAt):
		// do nothing if the timer is already scheduled to run sooner than the provided time
		return 0

	default:
		dur = next.Sub(now)
		if dur < timerMinDur {
			dur = timerMinDur
		}
	}

	next = now.Add(dur)

	t.Stop()
	t.timerAt = next
	t.timer.Reset(dur)

	return dur
}

func (t *schedulerTimer) Ch() <-chan time.Time {
	return t.timer.C
}

func (t *schedulerTimer) Stop() {
	// stop the timer
	t.timer.Stop()

	// consume the timer's tick if any
	select {
	case <-t.timer.C:
	default:
	}

	t.timerAt = time.Time{}
}
