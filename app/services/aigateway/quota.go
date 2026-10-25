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
	"fmt"
	"sync"
	"time"
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
type Planner struct {
	mu sync.Mutex

	now          func() time.Time
	dailyLimit   int
	rpm          int
	reservations map[Class]float64

	day    string
	used   int
	byCls  map[Class]int
	window []time.Time
}

func NewPlanner(dailyLimit, rpm int, reservations map[Class]float64) *Planner {
	if reservations == nil {
		reservations = DefaultReservations
	}
	return &Planner{
		now:          time.Now,
		dailyLimit:   dailyLimit,
		rpm:          rpm,
		reservations: reservations,
		byCls:        map[Class]int{},
	}
}

// Reserve charges one request to the class or returns a *QuotaError.
func (p *Planner) Reserve(class Class) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	now := p.now().UTC()
	p.rollDay(now)

	if p.used >= p.dailyLimit || !p.classHasRoom(class, now) {
		return &QuotaError{Class: class, Daily: true, RetryAt: nextUTCDay(now)}
	}

	p.trimWindow(now)
	if len(p.window) >= p.rpm {
		return &QuotaError{Class: class, RetryAt: p.window[0].Add(time.Minute)}
	}

	p.used++
	p.byCls[class]++
	p.window = append(p.window, now)
	return nil
}

// MarkExhausted is called when OpenRouter reports the daily limit before the planner does,
// for example because another client shares the account.
func (p *Planner) MarkExhausted() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.rollDay(p.now().UTC())
	p.used = p.dailyLimit
}

// Remaining returns how many requests the class can still make today.
func (p *Planner) Remaining(class Class) int {
	p.mu.Lock()
	defer p.mu.Unlock()

	now := p.now().UTC()
	p.rollDay(now)

	left := p.dailyLimit - p.used
	if now.Hour() >= sharedPoolHourUTC {
		return max(left, 0)
	}
	return max(min(left, p.classLimit(class)-p.byCls[class]), 0)
}

func (p *Planner) Snapshot() QuotaSnapshot {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.rollDay(p.now().UTC())

	byCls := make(map[Class]int, len(p.byCls))
	for k, v := range p.byCls {
		byCls[k] = v
	}
	return QuotaSnapshot{Day: p.day, Limit: p.dailyLimit, Used: p.used, UsedClass: byCls}
}

func (p *Planner) classHasRoom(class Class, now time.Time) bool {
	if now.Hour() >= sharedPoolHourUTC {
		return true
	}
	return p.byCls[class] < p.classLimit(class)
}

func (p *Planner) classLimit(class Class) int {
	limit := int(float64(p.dailyLimit) * p.reservations[class])
	// every configured class gets at least one request so decisions never starve completely
	if limit == 0 && p.reservations[class] > 0 {
		limit = 1
	}
	return limit
}

func (p *Planner) rollDay(now time.Time) {
	day := now.Format(time.DateOnly)
	if day == p.day {
		return
	}
	p.day = day
	p.used = 0
	p.byCls = map[Class]int{}
	p.window = nil
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
