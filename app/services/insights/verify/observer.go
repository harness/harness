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

package verify

import (
	"context"
	"errors"
	"fmt"

	"github.com/harness/gitness/app/services/insights"
	"github.com/harness/gitness/app/services/task"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

// TaskFinder finds the tasks linked to an entity; it is implemented by store.TaskLinkStore.
type TaskFinder interface {
	FindTasksByRef(ctx context.Context, kind enum.TaskLinkKind, refID string) ([]*types.Task, error)
}

var _ insights.IssueObserver = (*ReopenObserver)(nil)

// ReopenObserver reopens the resolved or verified task of an issue that regressed.
type ReopenObserver struct {
	insights.NoopObserver

	tasks       TaskFinder
	taskByID    func(ctx context.Context, id int64) (*types.Task, error)
	transitions Transitioner
	principal   func() int64
}

// NewReopenObserver creates the observer. taskByID may be nil to only use task links.
func NewReopenObserver(
	tasks TaskFinder,
	taskByID func(ctx context.Context, id int64) (*types.Task, error),
	transitions Transitioner,
	principal func() int64,
) *ReopenObserver {
	return &ReopenObserver{tasks: tasks, taskByID: taskByID, transitions: transitions, principal: principal}
}

// OnIssueRegressed moves the issue's resolved or verified tasks to reabierta.
func (o *ReopenObserver) OnIssueRegressed(
	ctx context.Context,
	_ *types.InsightsProject,
	issue *types.InsightsIssue,
	event *types.InsightsEvent,
) error {
	linked, err := o.tasks.FindTasksByRef(ctx, enum.TaskLinkKindInsightsIssue, IssueRef(issue.ID))
	if err != nil {
		return fmt.Errorf("failed to find tasks of issue %d: %w", issue.ID, err)
	}
	seen := map[int64]bool{}
	for _, t := range linked {
		seen[t.ID] = true
	}
	if issue.TaskID != nil && !seen[*issue.TaskID] && o.taskByID != nil {
		if t, ferr := o.taskByID(ctx, *issue.TaskID); ferr == nil {
			linked = append(linked, t)
		}
	}

	release := issue.LastRelease
	if event != nil && event.Release != "" {
		release = event.Release
	}
	reason := "error reappeared in release " + release
	if release == "" {
		reason = "error reappeared"
	}

	var errs []error
	for _, t := range linked {
		if t.State != enum.TaskStateResolved && t.State != enum.TaskStateVerified {
			continue
		}
		_, terr := o.transitions.Transition(ctx, o.principal(), t.ID, enum.TaskStateReopened, reason)
		if terr != nil && !errors.Is(terr, task.ErrInvalidTransition) {
			errs = append(errs, fmt.Errorf("failed to reopen task %d: %w", t.ID, terr))
		}
	}
	return errors.Join(errs...)
}
