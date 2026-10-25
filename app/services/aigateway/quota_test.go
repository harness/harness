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

package aigateway

import (
	"errors"
	"testing"
	"time"
)

func fixedClock(ts string) func() time.Time {
	t, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		panic(err)
	}
	return func() time.Time { return t }
}

type clock struct{ t time.Time }

func (c *clock) now() time.Time { return c.t }

func (c *clock) advance(d time.Duration) { c.t = c.t.Add(d) }

func newClock(ts string) *clock { return &clock{t: fixedClock(ts)()} }

func reserveN(p *Planner, c Class, n int) error {
	for range n {
		if err := p.Reserve(c); err != nil {
			return err
		}
	}
	return nil
}

func TestPlannerClassReservationsBeforeNoon(t *testing.T) {
	c := newClock("2026-10-25T09:00:00Z")
	p := NewPlanner(100, 1000, nil)
	p.now = c.now

	if err := reserveN(p, ClassDecision, 25); err != nil {
		t.Fatalf("decisions should have 25 requests: %v", err)
	}
	err := p.Reserve(ClassDecision)
	var qe *QuotaError
	if !errors.As(err, &qe) || !qe.Daily || qe.Class != ClassDecision {
		t.Fatalf("expected daily quota error for decisions, got %v", err)
	}
	if !errors.Is(err, ErrQuotaExhausted) {
		t.Errorf("quota errors must match ErrQuotaExhausted")
	}
	if want := time.Date(2026, 10, 26, 0, 0, 0, 0, time.UTC); !qe.RetryAt.Equal(want) {
		t.Errorf("expected retry at %s, got %s", want, qe.RetryAt)
	}

	// other classes keep their reservation
	if got := p.Remaining(ClassAgent); got != 40 {
		t.Errorf("expected 40 agent requests left, got %d", got)
	}
}

func TestPlannerSharedPoolAfterNoon(t *testing.T) {
	c := newClock("2026-10-25T09:00:00Z")
	p := NewPlanner(100, 1000, nil)
	p.now = c.now

	if err := reserveN(p, ClassDecision, 25); err != nil {
		t.Fatal(err)
	}
	c.advance(4 * time.Hour)

	if got := p.Remaining(ClassDecision); got != 75 {
		t.Errorf("unused reservations must be shared after noon, got %d", got)
	}
	if err := reserveN(p, ClassDecision, 75); err != nil {
		t.Fatalf("decisions should use the shared pool: %v", err)
	}
	if err := p.Reserve(ClassAgent); !errors.Is(err, ErrQuotaExhausted) {
		t.Errorf("daily limit must hold for every class, got %v", err)
	}
}

func TestPlannerResetsOnNewUTCDay(t *testing.T) {
	c := newClock("2026-10-25T23:59:00Z")
	p := NewPlanner(4, 1000, nil)
	p.now = c.now

	if err := reserveN(p, ClassAgent, 4); err != nil {
		t.Fatal(err)
	}
	if err := p.Reserve(ClassAgent); err == nil {
		t.Fatal("expected quota error")
	}

	c.advance(2 * time.Minute)
	if err := p.Reserve(ClassAgent); err != nil {
		t.Fatalf("quota must reset at UTC midnight: %v", err)
	}
	if s := p.Snapshot(); s.Day != "2026-10-26" || s.Used != 1 {
		t.Errorf("unexpected snapshot: %+v", s)
	}
}

func TestPlannerPerMinuteLimit(t *testing.T) {
	c := newClock("2026-10-25T13:00:00Z")
	p := NewPlanner(1000, 3, nil)
	p.now = c.now

	if err := reserveN(p, ClassTriage, 3); err != nil {
		t.Fatal(err)
	}
	err := p.Reserve(ClassTriage)
	var qe *QuotaError
	if !errors.As(err, &qe) || qe.Daily {
		t.Fatalf("expected per-minute quota error, got %v", err)
	}
	if want := c.t.Add(time.Minute); !qe.RetryAt.Equal(want) {
		t.Errorf("expected retry at %s, got %s", want, qe.RetryAt)
	}

	c.advance(time.Minute)
	if err := p.Reserve(ClassTriage); err != nil {
		t.Errorf("window must slide: %v", err)
	}
	// the request rejected by the per-minute limit must not have been charged
	if s := p.Snapshot(); s.Used != 4 {
		t.Errorf("expected 4 requests charged, got %d", s.Used)
	}
}

func TestPlannerSmallLimitsKeepOneDecision(t *testing.T) {
	p := NewPlanner(2, 1000, nil)
	p.now = fixedClock("2026-10-25T08:00:00Z")
	if err := p.Reserve(ClassManual); err != nil {
		t.Errorf("every class with a reservation gets at least one request: %v", err)
	}
}

func TestPlannerMarkExhausted(t *testing.T) {
	p := NewPlanner(1000, 1000, nil)
	p.now = fixedClock("2026-10-25T08:00:00Z")
	p.MarkExhausted()
	if err := p.Reserve(ClassDecision); !errors.Is(err, ErrQuotaExhausted) {
		t.Errorf("expected exhausted quota, got %v", err)
	}
}
