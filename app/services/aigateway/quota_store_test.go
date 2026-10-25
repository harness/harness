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
	"context"
	"errors"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/harness/gitness/types"
)

// fakeQuotaStore is an in-memory AIQuotaStore that can be told to fail.
type fakeQuotaStore struct {
	mu   sync.Mutex
	used map[[2]string]int

	failIncrement map[string]bool // by class
	failDecrement bool
}

func newFakeQuotaStore() *fakeQuotaStore {
	return &fakeQuotaStore{used: map[[2]string]int{}, failIncrement: map[string]bool{}}
}

func (f *fakeQuotaStore) Find(_ context.Context, day, class string) (*types.AIQuota, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return &types.AIQuota{Day: day, Class: class, Used: f.used[[2]string{day, class}]}, nil
}

func (f *fakeQuotaStore) IncrementQuota(_ context.Context, day, class string, limit int) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.failIncrement[class] {
		return false, errors.New("boom")
	}
	k := [2]string{day, class}
	if limit <= 0 || f.used[k] >= limit {
		return false, nil
	}
	f.used[k]++
	return true, nil
}

func (f *fakeQuotaStore) DecrementQuota(_ context.Context, day, class string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.failDecrement {
		return errors.New("boom")
	}
	k := [2]string{day, class}
	if f.used[k] > 0 {
		f.used[k]--
	}
	return nil
}

func (f *fakeQuotaStore) RaiseQuota(_ context.Context, day, class string, used int) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	k := [2]string{day, class}
	f.used[k] = max(f.used[k], used)
	return nil
}

func (f *fakeQuotaStore) get(day, class string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.used[[2]string{day, class}]
}

func TestStorePlannerSharedBetweenReplicas(t *testing.T) {
	st := newFakeQuotaStore()
	c := newClock("2026-10-25T13:00:00Z") // shared pool: only the total holds
	a := NewStorePlanner(4, 1000, nil, st)
	b := NewStorePlanner(4, 1000, nil, st)
	a.now, b.now = c.now, c.now

	if err := reserveN(a, ClassAgent, 2); err != nil {
		t.Fatal(err)
	}
	if err := reserveN(b, ClassManual, 2); err != nil {
		t.Fatal(err)
	}
	for _, p := range []*Planner{a, b} {
		var qe *QuotaError
		if err := p.Reserve(ClassDecision); !errors.As(err, &qe) || !qe.Daily {
			t.Fatalf("both replicas must see the spent total, got %v", err)
		}
	}
	if got := a.Snapshot().Used; got != 4 {
		t.Fatalf("snapshot used = %d, want 4", got)
	}
}

func TestStorePlannerGivesClassUnitBackWhenTotalSpent(t *testing.T) {
	st := newFakeQuotaStore()
	c := newClock("2026-10-25T09:00:00Z")
	p := NewStorePlanner(4, 1000, nil, st) // decision reservation: 1, agent: 1
	p.now = c.now

	// someone else spent the whole total
	if err := st.RaiseQuota(context.Background(), "2026-10-25", totalClass, 4); err != nil {
		t.Fatal(err)
	}
	err := p.Reserve(ClassAgent)
	var qe *QuotaError
	if !errors.As(err, &qe) || !qe.Daily {
		t.Fatalf("expected daily quota error, got %v", err)
	}
	if got := st.get("2026-10-25", "agent"); got != 0 {
		t.Fatalf("class unit must be given back, agent used = %d", got)
	}
}

func TestStorePlannerLeaksClassUnitWhenGiveBackFails(t *testing.T) {
	st := newFakeQuotaStore()
	st.failDecrement = true
	c := newClock("2026-10-25T09:00:00Z")
	p := NewStorePlanner(4, 1000, nil, st)
	p.now = c.now

	_ = st.RaiseQuota(context.Background(), "2026-10-25", totalClass, 4)
	if err := p.Reserve(ClassAgent); !errors.Is(err, ErrQuotaExhausted) {
		t.Fatalf("expected quota error, got %v", err)
	}
	// conservative: the class loses a unit, the total is never overspent
	if got := st.get("2026-10-25", "agent"); got != 1 {
		t.Fatalf("agent used = %d, want 1", got)
	}
	if got := st.get("2026-10-25", totalClass); got != 4 {
		t.Fatalf("total used = %d, want 4", got)
	}
}

func TestStorePlannerFailsClosedOnStoreError(t *testing.T) {
	st := newFakeQuotaStore()
	st.failIncrement["agent"] = true
	c := newClock("2026-10-25T09:00:00Z")
	p := NewStorePlanner(100, 1000, nil, st)
	p.now = c.now

	err := p.Reserve(ClassAgent)
	if err == nil || errors.Is(err, ErrQuotaExhausted) {
		t.Fatalf("expected a plain store error, got %v", err)
	}
}

func TestStorePlannerMarkExhaustedStopsEveryReplica(t *testing.T) {
	st := newFakeQuotaStore()
	c := newClock("2026-10-25T13:00:00Z")
	a := NewStorePlanner(100, 1000, nil, st)
	b := NewStorePlanner(100, 1000, nil, st)
	a.now, b.now = c.now, c.now

	a.MarkExhausted()
	if err := b.Reserve(ClassAgent); !errors.Is(err, ErrQuotaExhausted) {
		t.Fatalf("other replica should be exhausted, got %v", err)
	}
	if got := b.Remaining(ClassAgent); got != 0 {
		t.Fatalf("remaining = %d, want 0", got)
	}

	// a new UTC day starts fresh
	c.advance(24 * time.Hour)
	if err := b.Reserve(ClassAgent); err != nil {
		t.Fatalf("next day should work: %v", err)
	}
}

type fakeUsageStore struct {
	mu   sync.Mutex
	rows []*types.AIUsage
	err  error
}

func (f *fakeUsageStore) Create(_ context.Context, u *types.AIUsage) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return f.err
	}
	f.rows = append(f.rows, u)
	return nil
}

func (f *fakeUsageStore) ListSince(context.Context, int64) ([]*types.AIUsage, error) {
	return f.rows, nil
}

func TestDecideRecordsUsageIncludingFailures(t *testing.T) {
	f := &fakeOpenRouter{decisions: map[string]func(http.ResponseWriter){
		"baseten/mammoth-1:free": answer("baseten/mammoth-1:free", 0),
	}}
	g := newTestGateway(t, f, 100)
	g.planner.now = fixedClock("2026-10-25T08:00:00Z")
	us := &fakeUsageStore{}
	g.WithStores(nil, us)

	if _, err := g.Decide(context.Background(), "s1", map[string]any{"a": 1}, fixQuestion); err != nil {
		t.Fatalf("decide failed: %v", err)
	}
	if len(us.rows) != 1 || us.rows[0].Feature != "decision" || us.rows[0].Requests != 1 ||
		us.rows[0].Model != "baseten/mammoth-1:free-build" {
		t.Fatalf("unexpected usage rows: %+v", us.rows)
	}

	// a failing request is recorded too
	f.mu.Lock()
	f.decisions["baseten/mammoth-1:free"] = status(http.StatusBadRequest)
	f.mu.Unlock()
	if _, err := g.Decide(context.Background(), "s2", map[string]any{"a": 2}, fixQuestion); err == nil {
		t.Fatal("expected the scripted failure")
	}
	if len(us.rows) < 2 {
		t.Fatalf("failed request not recorded: %+v", us.rows)
	}

	// a failing usage write never fails the call
	us.err = errors.New("db down")
	if _, err := g.Chat(context.Background(), ClassManual, ChatRequest{
		Model:    "vendor/coder:free",
		Messages: []ChatMessage{{Role: "user", Content: "hi"}},
	}); err != nil {
		t.Fatalf("chat must not fail on usage errors: %v", err)
	}
}
