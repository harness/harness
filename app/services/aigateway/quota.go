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
	"fmt"
	"sync"
	"time"

	"github.com/harness/gitness/app/store"

	"github.com/rs/zerolog/log"
)

// Class groups requests that share a daily reservation.
type Class string

const (
	ClassDecision Class = "decision"
	ClassTriage   Class = "triage"
	ClassAgent    Class = "agent"
	ClassManual   Class = "manual"
)

// DefaultReservations is the share of the daily quota reserved for each class.
var DefaultReservations = map[Class]float64{
	ClassDecision: 0.25,
	ClassTriage:   0.25,
	ClassAgent:    0.40,
	ClassManual:   0.10,
}

// sharedPoolHourUTC is the hour after which unused reservations are released to every class.
const sharedPoolHourUTC = 12

// totalClass is the pseudo-class whose counter holds the daily total across all classes.
const totalClass = "total"

// quotaStoreTimeout bounds a single planner call to the shared counters.
const quotaStoreTimeout = 5 * time.Second

var ErrQuotaExhausted = errors.New("free model quota exhausted")

// QuotaError tells the caller when the request can be retried.
type QuotaError struct {
	Class   Class
	Daily   bool
	RetryAt time.Time
}

func (e *QuotaError) Error() string {
	scope := "per-minute"
	if e.Daily {
		scope = "daily"
	}
	return fmt.Sprintf("%s: %s limit reached for class %s, retry at %s",
		ErrQuotaExhausted, scope, e.Class, e.RetryAt.UTC().Format(time.RFC3339))
}

func (e *QuotaError) Is(target error) bool {
	return target == ErrQuotaExhausted
}

// QuotaSnapshot is the state of the planner for the current UTC day.
type QuotaSnapshot struct {
	Day       string
	Limit     int
	Used      int
	UsedClass map[Class]int
}

// Planner spends the account-wide free request quota. Failed requests count too,
// so a request is charged when it is reserved, not when it succeeds.
//
// Daily counters are shared between replicas when the planner is backed by the quota store:
// every counter is moved by one atomic conditional upsert, so no replica can overspend.
// A reservation touches two counters: the class counter (limited by the class reservation
// before noon UTC) and the "total" counter (limited by the daily limit). They can't be
// updated in one statement, so the class counter is charged first and given back with
// Decrement if the total is already spent. Giving it back is best effort: if that write fails
// the class loses one unit for the day, which under-spends but never overspends.
// The per-minute window is kept in memory, so the rpm given here is a per-replica limit.
// The gateway passes Config.RequestsPerMinute divided by Config.Replicas
// (GITNESS_AI_GATEWAY_REPLICAS), so N replicas together stay below the account limit.
type Planner struct {
	mu sync.Mutex

	now          func() time.Time
	dailyLimit   int
	rpm          int
	reservations map[Class]float64
	counters     counters

	window []time.Time
}

// NewPlanner returns a planner with in-memory counters.
func NewPlanner(dailyLimit, rpm int, reservations map[Class]float64) *Planner {
	return newPlanner(dailyLimit, rpm, reservations, newMemCounters())
}

// NewStorePlanner returns a planner whose daily counters are kept in the quota store.
func NewStorePlanner(
	dailyLimit, rpm int,
	reservations map[Class]float64,
	quotaStore store.AIQuotaStore,
) *Planner {
	return newPlanner(dailyLimit, rpm, reservations, storeCounters{store: quotaStore})
}

func newPlanner(dailyLimit, rpm int, reservations map[Class]float64, c counters) *Planner {
	if reservations == nil {
		reservations = DefaultReservations
	}
	return &Planner{
		now:          time.Now,
		dailyLimit:   dailyLimit,
		rpm:          rpm,
		reservations: reservations,
		counters:     c,
	}
}

// Reserve charges one request to the class or returns a *QuotaError.
// If the shared counters can't be reached the request is refused, so a broken
// database can't turn into unmetered requests.
func (p *Planner) Reserve(class Class) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), quotaStoreTimeout)
	defer cancel()

	now := p.now().UTC()
	day := now.Format(time.DateOnly)

	// the window check is free, so it goes first and never wastes a daily unit
	p.trimWindow(now)
	if len(p.window) >= p.rpm {
		return &QuotaError{Class: class, RetryAt: p.window[0].Add(time.Minute)}
	}

	if err := p.charge(ctx, day, class, now); err != nil {
		return err
	}
	p.window = append(p.window, now)
	return nil
}

func (p *Planner) charge(ctx context.Context, day string, class Class, now time.Time) error {
	exhausted := &QuotaError{Class: class, Daily: true, RetryAt: nextUTCDay(now)}

	if now.Hour() >= sharedPoolHourUTC {
		// after noon unused reservations are released: only the total holds
		ok, err := p.counters.Increment(ctx, day, totalClass, p.dailyLimit)
		if err != nil {
			return fmt.Errorf("failed to charge ai quota: %w", err)
		}
		if !ok {
			return exhausted
		}
		// keep the class counter for the snapshot; it can't fail the request
		if _, err := p.counters.Increment(ctx, day, string(class), p.dailyLimit); err != nil {
			log.Ctx(ctx).Warn().Err(err).Str("class", string(class)).Msg("ai gateway: failed to count class quota")
		}
		return nil
	}

	ok, err := p.counters.Increment(ctx, day, string(class), p.classLimit(class))
	if err != nil {
		return fmt.Errorf("failed to charge ai quota: %w", err)
	}
	if !ok {
		return exhausted
	}

	ok, err = p.counters.Increment(ctx, day, totalClass, p.dailyLimit)
	if err == nil && ok {
		return nil
	}

	// the total is spent (or unreachable): give the class unit back
	if derr := p.counters.Decrement(ctx, day, string(class)); derr != nil {
		log.Ctx(ctx).Warn().Err(derr).Str("class", string(class)).Msg("ai gateway: failed to give back class quota")
	}
	if err != nil {
		return fmt.Errorf("failed to charge ai quota: %w", err)
	}
	return exhausted
}

// MarkExhausted is called when OpenRouter reports the daily limit before the planner does,
// for example because another client shares the account. It is persisted, so every replica stops.
func (p *Planner) MarkExhausted() {
	p.mu.Lock()
	defer p.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), quotaStoreTimeout)
	defer cancel()

	day := p.now().UTC().Format(time.DateOnly)
	if err := p.counters.Raise(ctx, day, totalClass, p.dailyLimit); err != nil {
		log.Ctx(ctx).Warn().Err(err).Msg("ai gateway: failed to persist exhausted quota")
	}
}

// Remaining returns how many requests the class can still make today.
// If the counters can't be read it reports zero.
func (p *Planner) Remaining(class Class) int {
	p.mu.Lock()
	defer p.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), quotaStoreTimeout)
	defer cancel()

	now := p.now().UTC()
	day := now.Format(time.DateOnly)

	total, err := p.counters.Used(ctx, day, totalClass)
	if err != nil {
		log.Ctx(ctx).Warn().Err(err).Msg("ai gateway: failed to read quota")
		return 0
	}
	left := p.dailyLimit - total
	if now.Hour() >= sharedPoolHourUTC {
		return max(left, 0)
	}
	used, err := p.counters.Used(ctx, day, string(class))
	if err != nil {
		log.Ctx(ctx).Warn().Err(err).Msg("ai gateway: failed to read quota")
		return 0
	}
	return max(min(left, p.classLimit(class)-used), 0)
}

func (p *Planner) Snapshot() QuotaSnapshot {
	p.mu.Lock()
	defer p.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), quotaStoreTimeout)
	defer cancel()

	day := p.now().UTC().Format(time.DateOnly)
	snap := QuotaSnapshot{Day: day, Limit: p.dailyLimit, UsedClass: make(map[Class]int, len(p.reservations))}

	total, err := p.counters.Used(ctx, day, totalClass)
	if err != nil {
		log.Ctx(ctx).Warn().Err(err).Msg("ai gateway: failed to read quota")
		return snap
	}
	snap.Used = total
	for class := range p.reservations {
		used, err := p.counters.Used(ctx, day, string(class))
		if err != nil {
			log.Ctx(ctx).Warn().Err(err).Msg("ai gateway: failed to read quota")
			continue
		}
		if used > 0 {
			snap.UsedClass[class] = used
		}
	}
	return snap
}

func (p *Planner) classLimit(class Class) int {
	limit := int(float64(p.dailyLimit) * p.reservations[class])
	// every configured class gets at least one request so decisions never starve completely
	if limit == 0 && p.reservations[class] > 0 {
		limit = 1
	}
	return limit
}

func (p *Planner) trimWindow(now time.Time) {
	cut := 0
	for cut < len(p.window) && now.Sub(p.window[cut]) >= time.Minute {
		cut++
	}
	p.window = p.window[cut:]
}

func nextUTCDay(now time.Time) time.Time {
	y, m, d := now.UTC().Date()
	return time.Date(y, m, d+1, 0, 0, 0, 0, time.UTC)
}
