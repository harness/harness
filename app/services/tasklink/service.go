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

// Package tasklink links pull requests to tasks ("Fixes TASK-12") and moves the tasks
// through their lifecycle when the pull request is opened, edited, merged or closed.
package tasklink

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/app/bootstrap"
	pullreqevents "github.com/harness/gitness/app/events/pullreq"
	"github.com/harness/gitness/app/services/task"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/events"
	gitness_store "github.com/harness/gitness/store"
	"github.com/harness/gitness/stream"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

const groupTaskLink = "gitness:tasklink"

// taskService is the subset of task.Service used by this service.
type taskService interface {
	Transition(ctx context.Context, principalID, taskID int64, to enum.TaskState, reason string) (*types.Task, error)
	Link(ctx context.Context, principalID, taskID int64, kind enum.TaskLinkKind, refID string) error
	Unlink(ctx context.Context, principalID, taskID int64, kind enum.TaskLinkKind, refID string) error
}

type taskFinder interface {
	Find(ctx context.Context, id int64) (*types.Task, error)
	FindByNumber(ctx context.Context, repoID, number int64) (*types.Task, error)
}

type linkFinder interface {
	ListByTask(ctx context.Context, taskID int64) ([]*types.TaskLink, error)
	FindTasksByRef(ctx context.Context, kind enum.TaskLinkKind, refID string) ([]*types.Task, error)
}

type pullReqFinder interface {
	Find(ctx context.Context, id int64) (*types.PullReq, error)
}

type principalFinder interface {
	Find(ctx context.Context, id int64) (*types.Principal, error)
}

// Service links pull requests to tasks based on the references found in their title and description
// and moves the tasks through their lifecycle when the pull request changes.
//
// The pull request is linked with kind pullreq and ref id equal to the (global) pull request id.
// All actions are executed as the system service principal, the pull request author is recorded
// in the reason of the state changes.
type Service struct {
	taskSvc    taskService
	taskStore  taskFinder
	linkStore  linkFinder
	pullreqs   pullReqFinder
	principals principalFinder

	// principalID returns the principal used for the actions, by default the system service principal.
	principalID func() int64
}

// NewService creates the service and registers the pull request event handlers.
func NewService(
	ctx context.Context,
	config *types.Config,
	pullreqEvReaderFactory *events.ReaderFactory[*pullreqevents.Reader],
	taskSvc *task.Service,
	taskStore store.TaskStore,
	linkStore store.TaskLinkStore,
	pullreqStore store.PullReqStore,
	principalStore store.PrincipalStore,
) (*Service, error) {
	s := newService(taskSvc, taskStore, linkStore, pullreqStore, principalStore)

	_, err := pullreqEvReaderFactory.Launch(ctx, groupTaskLink, config.InstanceID,
		func(r *pullreqevents.Reader) error {
			const idleTimeout = 30 * time.Second
			r.Configure(
				stream.WithConcurrency(1),
				stream.WithHandlerOptions(
					stream.WithIdleTimeout(idleTimeout),
					stream.WithMaxRetries(3),
				))

			_ = r.RegisterCreated(s.onCreated)
			_ = r.RegisterUpdated(s.onUpdated)
			_ = r.RegisterReopened(s.onReopened)
			_ = r.RegisterMerged(s.onMerged)
			_ = r.RegisterClosed(s.onClosed)

			return nil
		})
	if err != nil {
		return nil, fmt.Errorf("failed to launch task link event readers: %w", err)
	}

	return s, nil
}

func newService(
	taskSvc taskService,
	taskStore taskFinder,
	linkStore linkFinder,
	pullreqs pullReqFinder,
	principals principalFinder,
) *Service {
	return &Service{
		taskSvc:    taskSvc,
		taskStore:  taskStore,
		linkStore:  linkStore,
		pullreqs:   pullreqs,
		principals: principals,
		principalID: func() int64 {
			return bootstrap.NewSystemServiceSession().Principal.ID
		},
	}
}

func prRef(pr *types.PullReq) string { return strconv.FormatInt(pr.ID, 10) }

func (s *Service) onCreated(
	ctx context.Context,
	e *events.Event[*pullreqevents.CreatedPayload],
) error {
	return s.syncPullReq(ctx, e.Payload.PullReqID)
}

func (s *Service) onReopened(
	ctx context.Context,
	e *events.Event[*pullreqevents.ReopenedPayload],
) error {
	return s.syncPullReq(ctx, e.Payload.PullReqID)
}

func (s *Service) onUpdated(
	ctx context.Context,
	e *events.Event[*pullreqevents.UpdatedPayload],
) error {
	if !e.Payload.TitleChanged && !e.Payload.DescriptionChanged {
		return nil
	}
	return s.syncPullReq(ctx, e.Payload.PullReqID)
}

// syncPullReq makes the links of an open pull request match the references in its text
// and moves the tasks closed by the pull request to en_revision.
func (s *Service) syncPullReq(ctx context.Context, pullReqID int64) error {
	pr, err := s.pullreqs.Find(ctx, pullReqID)
	if err != nil {
		return fmt.Errorf("failed to find pull request %d: %w", pullReqID, err)
	}
	if pr.State != enum.PullReqStateOpen {
		return nil
	}

	author := s.authorName(ctx, pr)
	principalID := s.principalID()
	wanted := map[int64]bool{}

	for _, ref := range ParseReferences(pr.Title, pr.Description) {
		t, err := s.taskStore.FindByNumber(ctx, pr.TargetRepoID, ref.Number)
		if errors.Is(err, gitness_store.ErrResourceNotFound) {
			continue // not a task of this repository
		}
		if err != nil {
			return fmt.Errorf("failed to find task %d: %w", ref.Number, err)
		}
		wanted[t.ID] = true

		if err = s.taskSvc.Link(ctx, principalID, t.ID, enum.TaskLinkKindPullReq, prRef(pr)); err != nil {
			return fmt.Errorf("failed to link task %d: %w", ref.Number, err)
		}
		if !ref.Closes {
			continue
		}
		reason := fmt.Sprintf("pull request #%d opened for the task (author: %s)", pr.Number, author)
		if err = s.moveToReview(ctx, principalID, t.ID, reason); err != nil {
			return err
		}
	}

	// References removed from the text.
	linked, err := s.linkStore.FindTasksByRef(ctx, enum.TaskLinkKindPullReq, prRef(pr))
	if err != nil {
		return fmt.Errorf("failed to list tasks linked to pull request %d: %w", pr.ID, err)
	}
	for _, t := range linked {
		if t.RepoID != pr.TargetRepoID || wanted[t.ID] {
			continue
		}
		if err = s.taskSvc.Unlink(ctx, principalID, t.ID, enum.TaskLinkKindPullReq, prRef(pr)); err != nil {
			return fmt.Errorf("failed to unlink task %d: %w", t.Number, err)
		}
	}

	return nil
}

// onMerged resolves the tasks that the merged pull request closes.
func (s *Service) onMerged(
	ctx context.Context,
	e *events.Event[*pullreqevents.MergedPayload],
) error {
	pr, err := s.pullreqs.Find(ctx, e.Payload.PullReqID)
	if err != nil {
		return fmt.Errorf("failed to find pull request %d: %w", e.Payload.PullReqID, err)
	}

	author := s.authorName(ctx, pr)
	principalID := s.principalID()

	for _, ref := range ParseReferences(pr.Title, pr.Description) {
		if !ref.Closes {
			continue
		}
		t, err := s.taskStore.FindByNumber(ctx, pr.TargetRepoID, ref.Number)
		if errors.Is(err, gitness_store.ErrResourceNotFound) {
			continue
		}
		if err != nil {
			return fmt.Errorf("failed to find task %d: %w", ref.Number, err)
		}

		if err = s.taskSvc.Link(ctx, principalID, t.ID, enum.TaskLinkKindPullReq, prRef(pr)); err != nil {
			return fmt.Errorf("failed to link task %d: %w", ref.Number, err)
		}

		reason := fmt.Sprintf("pull request #%d merged (author: %s)", pr.Number, author)
		if err = s.moveToReview(ctx, principalID, t.ID, reason); err != nil {
			return err
		}
		if err = s.transition(ctx, principalID, t.ID, enum.TaskStateResolved, reason); err != nil {
			return err
		}
	}

	return nil
}

// onClosed sends the tasks in review back to en_curso if no other open pull request links them.
func (s *Service) onClosed(
	ctx context.Context,
	e *events.Event[*pullreqevents.ClosedPayload],
) error {
	pr, err := s.pullreqs.Find(ctx, e.Payload.PullReqID)
	if err != nil {
		return fmt.Errorf("failed to find pull request %d: %w", e.Payload.PullReqID, err)
	}
	if pr.State != enum.PullReqStateClosed {
		return nil // reopened or merged in the meantime
	}

	linked, err := s.linkStore.FindTasksByRef(ctx, enum.TaskLinkKindPullReq, prRef(pr))
	if err != nil {
		return fmt.Errorf("failed to list tasks linked to pull request %d: %w", pr.ID, err)
	}

	author := s.authorName(ctx, pr)
	principalID := s.principalID()

	for _, t := range linked {
		if t.RepoID != pr.TargetRepoID || t.State != enum.TaskStateInReview {
			continue
		}
		other, err := s.hasOtherOpenPullReq(ctx, t.ID, pr.ID)
		if err != nil {
			return err
		}
		if other {
			continue
		}
		reason := fmt.Sprintf("pull request #%d closed without merge (author: %s)", pr.Number, author)
		if err = s.transition(ctx, principalID, t.ID, enum.TaskStateInProgress, reason); err != nil {
			return err
		}
	}

	return nil
}

// hasOtherOpenPullReq reports whether the task is linked to an open pull request other than the given one.
func (s *Service) hasOtherOpenPullReq(ctx context.Context, taskID, exceptPullReqID int64) (bool, error) {
	links, err := s.linkStore.ListByTask(ctx, taskID)
	if err != nil {
		return false, fmt.Errorf("failed to list links of task %d: %w", taskID, err)
	}
	for _, l := range links {
		if l.Kind != enum.TaskLinkKindPullReq {
			continue
		}
		id, err := strconv.ParseInt(l.RefID, 10, 64)
		if err != nil || id == exceptPullReqID {
			continue
		}
		pr, err := s.pullreqs.Find(ctx, id)
		if errors.Is(err, gitness_store.ErrResourceNotFound) {
			continue
		}
		if err != nil {
			return false, fmt.Errorf("failed to find pull request %d: %w", id, err)
		}
		if pr.State == enum.PullReqStateOpen {
			return true, nil
		}
	}
	return false, nil
}

// moveToReview moves a task in abierta, triage, en_curso or reabierta to en_revision
// (going through en_curso when needed). Tasks in any other state are left untouched.
func (s *Service) moveToReview(ctx context.Context, principalID, taskID int64, reason string) error {
	// Read the task again, the state could have changed since it was loaded.
	cur, err := s.taskStore.Find(ctx, taskID)
	if err != nil {
		return fmt.Errorf("failed to find task %d: %w", taskID, err)
	}

	switch cur.State { //nolint:exhaustive
	case enum.TaskStateOpen, enum.TaskStateTriage, enum.TaskStateReopened:
		if err = s.transition(ctx, principalID, taskID, enum.TaskStateInProgress, reason); err != nil {
			return err
		}
		return s.transition(ctx, principalID, taskID, enum.TaskStateInReview, reason)
	case enum.TaskStateInProgress:
		return s.transition(ctx, principalID, taskID, enum.TaskStateInReview, reason)
	}
	return nil
}

// transition applies a state change. Invalid transitions are logged and ignored.
func (s *Service) transition(
	ctx context.Context,
	principalID, taskID int64,
	to enum.TaskState,
	reason string,
) error {
	cur, err := s.taskStore.Find(ctx, taskID)
	if err != nil {
		return fmt.Errorf("failed to find task %d: %w", taskID, err)
	}
	if cur.State == to {
		return nil // already there (event replay)
	}

	_, err = s.taskSvc.Transition(ctx, principalID, taskID, to, reason)
	if err == nil {
		return nil
	}
	var ue *usererror.Error
	if errors.Is(err, task.ErrInvalidTransition) || errors.As(err, &ue) {
		log.Ctx(ctx).Warn().Err(err).Msgf("task link: ignoring invalid transition of task %d from %q to %q",
			taskID, cur.State, to)
		return nil
	}
	return fmt.Errorf("failed to transition task %d to %q: %w", taskID, to, err)
}

func (s *Service) authorName(ctx context.Context, pr *types.PullReq) string {
	p, err := s.principals.Find(ctx, pr.CreatedBy)
	if err != nil || p == nil {
		return strconv.FormatInt(pr.CreatedBy, 10)
	}
	return p.UID
}
