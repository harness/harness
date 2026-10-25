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

package agentguard

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/types"
)

// ErrAttemptDenied is returned (wrapped) by StartAttempt if the atomic reservation was refused.
var ErrAttemptDenied = errors.New("agent attempt denied")

// TryStartParams are the inputs of an atomic attempt reservation.
type TryStartParams struct {
	RepoID  int64
	TaskKey string
	// MaxActive is the maximum number of running attempts of the repo.
	MaxActive int
	// Cooldown is the minimum time between attempts of the same task.
	Cooldown time.Duration
	// MaxFailed is the number of failed attempts after which the task is refused.
	MaxFailed int
	// At is the start time of the attempt.
	At time.Time
	// ActiveTimeout, if positive, excludes running attempts older than it from the max active count.
	ActiveTimeout time.Duration
}

// StartOptions tune the reservation done by TryStartAttempt.
type StartOptions struct {
	// SkipCooldown reserves without enforcing the cooldown (e.g. for an immediate retry after a failure).
	SkipCooldown bool
}

// TryStartAttempt atomically checks max active, cooldown and failed count and records a running attempt,
// so concurrent launches cannot exceed the max active agents of the repo. If the reservation is refused
// the returned error wraps ErrAttemptDenied and the deny reason.
func (g *Guard) TryStartAttempt(ctx context.Context, req Request, opts StartOptions) (Attempt, error) {
	maxActive := math.MaxInt32
	if g.settings != nil {
		s, err := g.settings.AgentSettings(ctx, req.RepoID)
		if err != nil {
			return Attempt{}, fmt.Errorf("failed to load repo agent settings: %w", err)
		}
		if s.MaxActive != nil {
			maxActive = *s.MaxActive
		}
	}
	cooldown := g.config.Cooldown
	if opts.SkipCooldown {
		cooldown = 0
	}

	at := g.now()
	id, reason, err := g.attempts.TryStart(ctx, TryStartParams{
		RepoID:    req.RepoID,
		TaskKey:   req.TaskKey,
		MaxActive: maxActive,
		Cooldown:  cooldown,
		MaxFailed: g.config.MaxFailedAttempts,
		At:        at,

		ActiveTimeout: g.config.AttemptTimeout,
	})
	if err != nil {
		return Attempt{}, err
	}
	if reason != "" {
		return Attempt{}, fmt.Errorf("%w: %s", ErrAttemptDenied, reason)
	}
	return Attempt{ID: id, TaskKey: req.TaskKey, RepoID: req.RepoID, StartedAt: at, Status: AttemptRunning}, nil
}

// DBAttemptStore is an AttemptStore persisted in the agent_attempts table. Attempts survive restarts
// and are shared by all replicas.
type DBAttemptStore struct {
	store store.AgentAttemptStore
}

// NewDBAttemptStore creates an AttemptStore backed by the database.
func NewDBAttemptStore(s store.AgentAttemptStore) *DBAttemptStore {
	return &DBAttemptStore{store: s}
}

var _ AttemptStore = (*DBAttemptStore)(nil)

func (s *DBAttemptStore) TryStart(ctx context.Context, p TryStartParams) (int64, Reason, error) {
	id, deny, err := s.store.TryStart(ctx, types.AgentAttemptStart{
		RepoID:    p.RepoID,
		TaskKey:   p.TaskKey,
		MaxActive: p.MaxActive,
		Cooldown:  p.Cooldown,
		MaxFailed: p.MaxFailed,
		At:        p.At,

		ActiveTimeout: p.ActiveTimeout,
	})
	return id, Reason(deny), err
}

func (s *DBAttemptStore) Start(ctx context.Context, taskKey string, repoID int64, at time.Time) (Attempt, error) {
	id, deny, err := s.store.TryStart(ctx, types.AgentAttemptStart{
		RepoID: repoID, TaskKey: taskKey, MaxActive: math.MaxInt32, MaxFailed: 0, At: at,
	})
	if err != nil {
		return Attempt{}, err
	}
	if deny != "" {
		return Attempt{}, fmt.Errorf("%w: %s", ErrAttemptDenied, deny)
	}
	return Attempt{ID: id, TaskKey: taskKey, RepoID: repoID, StartedAt: at, Status: AttemptRunning}, nil
}

func (s *DBAttemptStore) Finish(ctx context.Context, attemptID int64, status AttemptStatus) error {
	return s.store.Finish(ctx, attemptID, types.AgentAttemptState(status), "")
}

func (s *DBAttemptStore) ListByTask(ctx context.Context, taskKey string) ([]Attempt, error) {
	list, err := s.store.ListByTask(ctx, taskKey)
	if err != nil {
		return nil, err
	}
	out := make([]Attempt, len(list))
	for i, a := range list {
		out[i] = Attempt{
			ID: a.ID, TaskKey: a.TaskKey, RepoID: a.RepoID,
			StartedAt: time.UnixMilli(a.Started), Status: AttemptStatus(a.State),
		}
	}
	return out, nil
}

func (s *DBAttemptStore) CountActive(ctx context.Context, repoID int64) (int, error) {
	return s.store.CountActive(ctx, repoID)
}
