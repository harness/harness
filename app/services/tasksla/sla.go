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

// Package tasksla provides the recurring job that notifies about tasks that are about to be due or overdue.
package tasksla

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

const (
	markerWarning = "warning"
	markerOverdue = "overdue"
	batchSize     = 100
)

// Notifier sends the SLA notifications. *notification.Service implements it.
type Notifier interface {
	NotifyTaskSLA(ctx context.Context, t *types.Task, overdue bool) error
}

// markerPayload is the payload of enum.TaskActivityKindSLAWarning activities. They record which
// notification was sent for a task so it is sent only once.
type markerPayload struct {
	Type string `json:"type"`
	Due  int64  `json:"due"`
}

// Result summarizes a run.
type Result struct {
	Warned  int
	Overdue int
	Failed  int
}

func (r Result) String() string {
	return fmt.Sprintf("warned=%d overdue=%d failed=%d", r.Warned, r.Overdue, r.Failed)
}

// Runner selects the tasks to notify and sends the notifications.
type Runner struct {
	window     time.Duration
	tasks      store.TaskStore
	activities store.TaskActivityStore
	notifier   Notifier
	now        func() time.Time
}

// NewRunner creates a runner notifying about tasks due within the window.
func NewRunner(
	window time.Duration,
	tasks store.TaskStore,
	activities store.TaskActivityStore,
	notifier Notifier,
) *Runner {
	return &Runner{window: window, tasks: tasks, activities: activities, notifier: notifier, now: time.Now}
}

// decide returns the marker type to notify for the task, or "" if nothing is to be sent.
// markers are the marker types already recorded for the task.
func decide(t *types.Task, nowMs int64, markers map[string]bool) string {
	if t.Due == nil {
		return ""
	}
	if *t.Due <= nowMs {
		if markers[markerOverdue] {
			return ""
		}
		return markerOverdue
	}
	if markers[markerWarning] || markers[markerOverdue] {
		return ""
	}
	return markerWarning
}

// Run notifies about the due tasks once.
func (r *Runner) Run(ctx context.Context) (Result, error) {
	var res Result
	now := r.now()
	nowMs := now.UnixMilli()
	until := now.Add(r.window).UnixMilli()

	var after int64
	for {
		list, err := r.tasks.ListDueBefore(ctx, until, after, batchSize)
		if err != nil {
			return res, fmt.Errorf("failed to list tasks due before %d: %w", until, err)
		}
		for _, t := range list {
			after = t.ID
			markers, err := r.markers(ctx, t.ID)
			if err != nil {
				res.Failed++
				continue
			}
			typ := decide(t, nowMs, markers)
			if typ == "" {
				continue
			}
			if err = r.notifier.NotifyTaskSLA(ctx, t, typ == markerOverdue); err != nil {
				res.Failed++
				continue
			}
			raw, _ := json.Marshal(markerPayload{Type: typ, Due: *t.Due})
			err = r.activities.Create(ctx, &types.TaskActivity{
				TaskID:  t.ID,
				Kind:    enum.TaskActivityKindSLAWarning,
				Payload: raw,
				Created: nowMs,
			})
			if err != nil {
				res.Failed++
				continue
			}
			if typ == markerOverdue {
				res.Overdue++
			} else {
				res.Warned++
			}
		}
		if len(list) < batchSize {
			return res, nil
		}
	}
}

func (r *Runner) markers(ctx context.Context, taskID int64) (map[string]bool, error) {
	acts, err := r.activities.ListByKind(ctx, taskID, enum.TaskActivityKindSLAWarning)
	if err != nil {
		return nil, err
	}
	out := make(map[string]bool, 2)
	for _, a := range acts {
		var p markerPayload
		if json.Unmarshal(a.Payload, &p) == nil {
			out[p.Type] = true
		}
	}
	return out, nil
}
