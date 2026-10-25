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

// Package quotaresume resumes the tasks that wait for the free AI quota (esperando_cuota).
// A recurring job (every 15 minutes, which includes 00:00 UTC when the daily quota resets) lists the
// waiting tasks in priority order (p0, regressions, p1, the rest), resumes each one and re-runs the
// original work through the resumer registered for the task origin. When a resumer hits the quota
// again the task goes back to waiting and the run stops.
package quotaresume

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/harness/gitness/app/bootstrap"
	"github.com/harness/gitness/app/services/aigateway"
	"github.com/harness/gitness/app/services/insights/triage"
	"github.com/harness/gitness/lock"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

const (
	lockKey        = "quotaresume"
	resumeReason   = "AI quota available again"
	waitAgainText  = "AI quota exhausted again, waiting for the next reset"
	maxPerRun      = 50
	insightsNoLink = "task has no Insights issue link"
)

// Resumer re-runs the original work of a task after it left esperando_cuota.
// It returns an error satisfying IsQuotaError when the quota is exhausted again.
type Resumer interface {
	Resume(ctx context.Context, t *types.Task) error
}

// ResumerFunc adapts a function to a Resumer.
type ResumerFunc func(ctx context.Context, t *types.Task) error

// Resume implements Resumer.
func (f ResumerFunc) Resume(ctx context.Context, t *types.Task) error { return f(ctx, t) }

// TaskStore is the part of the task store used here.
type TaskStore interface {
	ListWaitingQuota(ctx context.Context, limit int) ([]*types.Task, error)
}

// TaskService is the part of the task service used here.
type TaskService interface {
	Resume(ctx context.Context, principalID, taskID int64, reason string) (*types.Task, error)
	Transition(ctx context.Context, principalID, taskID int64, to enum.TaskState, reason string) (*types.Task, error)
}

// Quota is the part of the planner used here.
type Quota interface {
	Remaining(class aigateway.Class) int
}

// LinkStore is the part of the task link store used here.
type LinkStore interface {
	ListByTask(ctx context.Context, taskID int64) ([]*types.TaskLink, error)
}

// Retriager re-runs the Insights triage of an issue.
type Retriager interface {
	Retriage(ctx context.Context, issueID int64) error
}

// IsQuotaError reports whether err says the AI quota is exhausted.
func IsQuotaError(err error) bool {
	return errors.Is(err, triage.ErrQuotaExhausted) || errors.Is(err, aigateway.ErrQuotaExhausted)
}

// Result summarizes a run.
type Result struct {
	// Resumed counts the tasks moved out of esperando_cuota and handed to their resumer.
	Resumed int
	// Failed counts the tasks whose resumer failed for a reason other than the quota.
	Failed int
	// Stopped is true when the run ended because the quota ran out.
	Stopped bool
	// Skipped is true when another instance holds the lock or no quota is left.
	Skipped bool
}

// String implements fmt.Stringer; it is the result text of the job.
func (r Result) String() string {
	return fmt.Sprintf("resumed=%d failed=%d stopped=%t skipped=%t", r.Resumed, r.Failed, r.Stopped, r.Skipped)
}

// Service resumes the tasks that wait for quota.
type Service struct {
	tasks     TaskStore
	svc       TaskService
	quota     Quota
	class     aigateway.Class
	mutexes   lock.MutexManager
	resumers  map[enum.TaskOrigin]Resumer
	principal func() int64
}

// New creates the service. Origins without a resumer are only resumed.
func New(
	tasks TaskStore,
	svc TaskService,
	quota Quota,
	mutexes lock.MutexManager,
	resumers map[enum.TaskOrigin]Resumer,
) *Service {
	if resumers == nil {
		resumers = map[enum.TaskOrigin]Resumer{}
	}
	return &Service{
		tasks: tasks, svc: svc, quota: quota, class: aigateway.ClassTriage, mutexes: mutexes, resumers: resumers,
		principal: func() int64 { return bootstrap.NewSystemServiceSession().Principal.ID },
	}
}

// Run resumes waiting tasks while quota is left. At most one run is active across replicas: a run that
// can't take the lock does nothing.
func (s *Service) Run(ctx context.Context) (Result, error) {
	mx, err := s.mutexes.NewMutex(lockKey, lock.WithTries(1))
	if err != nil {
		return Result{}, fmt.Errorf("failed to create quota resume mutex: %w", err)
	}
	if err = mx.Lock(ctx); err != nil {
		log.Ctx(ctx).Debug().Err(err).Msg("quota resume: another instance is running")
		return Result{Skipped: true}, nil
	}
	defer func() {
		if uerr := mx.Unlock(context.WithoutCancel(ctx)); uerr != nil {
			log.Ctx(ctx).Warn().Err(uerr).Msg("quota resume: failed to release lock")
		}
	}()

	budget := min(s.quota.Remaining(s.class), maxPerRun)
	if budget <= 0 {
		return Result{Skipped: true}, nil
	}
	list, err := s.tasks.ListWaitingQuota(ctx, budget)
	if err != nil {
		return Result{}, fmt.Errorf("failed to list tasks waiting for quota: %w", err)
	}

	var res Result
	pid := s.principal()
	for _, t := range list {
		if ctx.Err() != nil {
			return res, ctx.Err()
		}
		if s.quota.Remaining(s.class) <= 0 {
			res.Stopped = true
			return res, nil
		}
		if _, err = s.svc.Resume(ctx, pid, t.ID, resumeReason); err != nil {
			log.Ctx(ctx).Warn().Err(err).Int64("task_id", t.ID).Msg("quota resume: failed to resume task")
			res.Failed++
			continue
		}
		res.Resumed++

		r, ok := s.resumers[t.Origin]
		if !ok {
			continue
		}
		err = r.Resume(ctx, t)
		switch {
		case err == nil:
		case IsQuotaError(err):
			if _, terr := s.svc.Transition(ctx, pid, t.ID, enum.TaskStateWaitingQuota, waitAgainText); terr != nil {
				log.Ctx(ctx).Warn().Err(terr).Int64("task_id", t.ID).Msg("quota resume: failed to put task back to waiting")
			}
			res.Stopped = true
			return res, nil
		default:
			log.Ctx(ctx).Warn().Err(err).Int64("task_id", t.ID).Msg("quota resume: resumer failed")
			res.Failed++
		}
	}
	return res, nil
}

// InsightsResumer re-runs the triage of the Insights issue linked to the task.
func InsightsResumer(links LinkStore, retriager Retriager) Resumer {
	return ResumerFunc(func(ctx context.Context, t *types.Task) error {
		ls, err := links.ListByTask(ctx, t.ID)
		if err != nil {
			return fmt.Errorf("failed to list task links: %w", err)
		}
		for _, l := range ls {
			if l.Kind != enum.TaskLinkKindInsightsIssue {
				continue
			}
			id, perr := strconv.ParseInt(l.RefID, 10, 64)
			if perr != nil {
				continue
			}
			return retriager.Retriage(ctx, id)
		}
		return errors.New(insightsNoLink)
	})
}

// ShieldResumer is a no-op for now: it only records that the task was resumed.
func ShieldResumer() Resumer {
	return ResumerFunc(func(ctx context.Context, t *types.Task) error {
		log.Ctx(ctx).Info().Int64("task_id", t.ID).Msg("quota resume: shield task resumed, nothing to re-run yet")
		return nil
	})
}
