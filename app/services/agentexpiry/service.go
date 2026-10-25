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

// Package agentexpiry expires the agent attempts that stay running for too long, e.g. because the replica
// that launched them crashed or the agent never reported completion. A recurring job marks them failed
// ("expired") so they stop occupying a max active slot of the repo and, for attempts of a task, records an
// agent_run with outcome "expired" and moves the task to bloqueada.
package agentexpiry

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/harness/gitness/app/bootstrap"
	"github.com/harness/gitness/app/services/task"
	"github.com/harness/gitness/lock"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

const (
	lockKey = "agentexpiry"

	// OutcomeExpired is the agent_run outcome recorded for an expired attempt.
	OutcomeExpired = "expired"

	taskKeyPrefix = "task:"
	blockReason   = "Agent attempt expired without reporting completion"
)

// AttemptStore is the part of the attempt store used here.
type AttemptStore interface {
	ExpireRunning(ctx context.Context, olderThan, now int64) ([]*types.AgentAttempt, error)
}

// TaskService is the part of the task service used here.
type TaskService interface {
	RecordAgentRun(ctx context.Context, principalID, taskID int64, in task.AgentRunInput) error
	Transition(ctx context.Context, principalID, taskID int64, to enum.TaskState, reason string) (*types.Task, error)
}

// Result summarizes a run.
type Result struct {
	// Expired counts the attempts marked as expired.
	Expired int
	// Skipped is true when another instance holds the lock or expiry is disabled.
	Skipped bool
}

// String implements fmt.Stringer; it is the result text of the job.
func (r Result) String() string {
	return fmt.Sprintf("expired=%d skipped=%t", r.Expired, r.Skipped)
}

// Service expires stale running attempts.
type Service struct {
	attempts  AttemptStore
	tasks     TaskService
	mutexes   lock.MutexManager
	timeout   time.Duration
	now       func() time.Time
	principal func() int64
}

// New creates the service. Attempts running for longer than timeout are expired; 0 disables expiry.
func New(attempts AttemptStore, tasks TaskService, mutexes lock.MutexManager, timeout time.Duration) *Service {
	return &Service{
		attempts: attempts, tasks: tasks, mutexes: mutexes, timeout: timeout, now: time.Now,
		principal: func() int64 { return bootstrap.NewSystemServiceSession().Principal.ID },
	}
}

// Run expires the stale attempts. At most one run is active across replicas: a run that can't take the
// lock does nothing.
func (s *Service) Run(ctx context.Context) (Result, error) {
	if s.timeout <= 0 {
		return Result{Skipped: true}, nil
	}
	mx, err := s.mutexes.NewMutex(lockKey, lock.WithTries(1))
	if err != nil {
		return Result{}, fmt.Errorf("failed to create agent expiry mutex: %w", err)
	}
	if err = mx.Lock(ctx); err != nil {
		log.Ctx(ctx).Debug().Err(err).Msg("agent expiry: another instance is running")
		return Result{Skipped: true}, nil
	}
	defer func() {
		if uerr := mx.Unlock(context.WithoutCancel(ctx)); uerr != nil {
			log.Ctx(ctx).Warn().Err(uerr).Msg("agent expiry: failed to release lock")
		}
	}()

	now := s.now()
	expired, err := s.attempts.ExpireRunning(ctx, now.Add(-s.timeout).UnixMilli(), now.UnixMilli())
	if err != nil {
		return Result{}, fmt.Errorf("failed to expire running agent attempts: %w", err)
	}
	pid := s.principal()
	for _, a := range expired {
		log.Ctx(ctx).Warn().Int64("attempt_id", a.ID).Str("task_key", a.TaskKey).Msg("agent expiry: attempt expired")
		taskID, ok := parseTaskKey(a.TaskKey)
		if !ok {
			continue
		}
		s.blockTask(ctx, pid, taskID, a)
	}
	return Result{Expired: len(expired)}, nil
}

// blockTask records the expired run on the task and moves it to bloqueada. Errors are logged only.
func (s *Service) blockTask(ctx context.Context, pid, taskID int64, a *types.AgentAttempt) {
	if err := s.tasks.RecordAgentRun(ctx, pid, taskID, task.AgentRunInput{
		Model: a.Model, Outcome: OutcomeExpired,
	}); err != nil {
		log.Ctx(ctx).Warn().Err(err).Int64("task_id", taskID).Msg("agent expiry: failed to record agent run")
	}
	if _, err := s.tasks.Transition(ctx, pid, taskID, enum.TaskStateBlocked, blockReason); err != nil {
		log.Ctx(ctx).Warn().Err(err).Int64("task_id", taskID).Msg("agent expiry: failed to block task")
	}
}

// parseTaskKey returns the task id of an attempt task key of the form "task:<id>".
func parseTaskKey(key string) (int64, bool) {
	rest, ok := strings.CutPrefix(key, taskKeyPrefix)
	if !ok {
		return 0, false
	}
	id, err := strconv.ParseInt(rest, 10, 64)
	if err != nil || id <= 0 {
		return 0, false
	}
	return id, true
}
