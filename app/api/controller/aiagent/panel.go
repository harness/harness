// Copyright 2026 Harness, Inc.
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

package aiagent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	apiauth "github.com/harness/gitness/app/api/auth"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/services/aigateway"
	gitness_store "github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

const (
	// maxWaitingShown is the number of tasks of the esperando_cuota queue returned.
	maxWaitingShown = 20
	// maxRunsShown is the number of latest agent runs returned.
	maxRunsShown = 20
	// maxActiveShown bounds the number of active agent tasks returned.
	maxActiveShown = 100
	// waitingScanLimit bounds the global queue scanned to find the tasks of the space.
	waitingScanLimit = 1000
	// repoPageSize is the page size used when walking the repositories of a space.
	repoPageSize = 100
	// countKeySuffix marks the observation count keys in the success rate map of a model.
	countKeySuffix = "#n"
)

var aiClasses = []aigateway.Class{
	aigateway.ClassDecision,
	aigateway.ClassTriage,
	aigateway.ClassAgent,
	aigateway.ClassManual,
}

// QuotaPanel is the quota snapshot of the AI gateway.
type QuotaPanel struct {
	Day              string         `json:"day,omitempty"`
	Limit            int            `json:"limit"`
	Used             int            `json:"used"`
	UsedByClass      map[string]int `json:"used_by_class"`
	RemainingByClass map[string]int `json:"remaining_by_class"`
	// SpaceUsedToday is the number of requests used today by the space (and its repositories).
	SpaceUsedToday int `json:"space_used_today"`
}

// AgentTask is a task handled by the agent.
type AgentTask struct {
	Number   int64             `json:"number"`
	RepoPath string            `json:"repo_path"`
	Title    string            `json:"title"`
	State    enum.TaskState    `json:"state"`
	Priority enum.TaskPriority `json:"priority"`
	Updated  int64             `json:"updated"`
	Model    string            `json:"model,omitempty"`
}

// WaitingQueue is the esperando_cuota queue.
type WaitingQueue struct {
	Count int64 `json:"count"`
	// Tasks are the first tasks of the space in the order they resume.
	Tasks []AgentTask `json:"tasks"`
}

// KindSuccess is the success rate of a model for a kind of work.
type KindSuccess struct {
	Kind    string  `json:"kind"`
	Rate    float64 `json:"rate"`
	Samples int     `json:"samples"`
}

// ModelSuccess is the success rate of a model per kind of work.
type ModelSuccess struct {
	Model string        `json:"model"`
	Kinds []KindSuccess `json:"kinds"`
}

// AgentRun is a run of the agent on a task.
type AgentRun struct {
	TaskNumber   int64  `json:"task_number"`
	RepoPath     string `json:"repo_path"`
	Title        string `json:"title"`
	Model        string `json:"model"`
	Outcome      string `json:"outcome"`
	RequestsUsed int    `json:"requests_used"`
	Created      int64  `json:"created"`
}

// Panel is the response of the AI agent panel endpoint.
type Panel struct {
	Enabled      bool           `json:"enabled"`
	Quota        QuotaPanel     `json:"quota"`
	ActiveTasks  []AgentTask    `json:"active_tasks"`
	WaitingQuota WaitingQueue   `json:"waiting_quota"`
	ModelSuccess []ModelSuccess `json:"model_success"`
	LatestRuns   []AgentRun     `json:"latest_runs"`
}

// agentRunPayload is the payload of an agent_run task activity (task.AgentRunInput).
type agentRunPayload struct {
	Model        string `json:"model"`
	RequestsUsed int    `json:"requests_used"`
	Outcome      string `json:"outcome"`
}

// launcherMetadata is the part of AITask.OutputMetadata written by the agent launcher that is used here.
type launcherMetadata struct {
	Model string `json:"agent_model"`
}

// Panel returns the AI agent panel of the space. It needs view access to the space; only the
// repositories the principal can view contribute to the tasks, runs and usage.
func (c *Controller) Panel(ctx context.Context, session *auth.Session, spaceRef string) (*Panel, error) {
	space, err := c.spaceFinder.FindByRef(ctx, spaceRef)
	if err != nil {
		return nil, fmt.Errorf("failed to find space: %w", err)
	}
	if err = apiauth.CheckSpace(ctx, c.authorizer, session, space, enum.PermissionSpaceView); err != nil {
		return nil, fmt.Errorf("access check failed: %w", err)
	}

	repos, err := c.viewableRepos(ctx, session, space.ID)
	if err != nil {
		return nil, err
	}
	repoIDs := make([]int64, 0, len(repos))
	paths := make(map[int64]string, len(repos))
	for _, r := range repos {
		repoIDs = append(repoIDs, r.ID)
		paths[r.ID] = r.Path
	}

	panel := &Panel{
		Enabled:      c.quota.Enabled(),
		ActiveTasks:  []AgentTask{},
		WaitingQuota: WaitingQueue{Tasks: []AgentTask{}},
		ModelSuccess: []ModelSuccess{},
		LatestRuns:   []AgentRun{},
	}

	if panel.Quota, err = c.quotaPanel(ctx, space.ID, repoIDs); err != nil {
		return nil, err
	}
	if panel.ActiveTasks, err = c.activeTasks(ctx, repoIDs, paths); err != nil {
		return nil, err
	}
	if panel.WaitingQuota, err = c.waitingQueue(ctx, repoIDs, paths); err != nil {
		return nil, err
	}
	if panel.ModelSuccess, err = c.modelSuccess(ctx); err != nil {
		return nil, err
	}
	if panel.LatestRuns, err = c.latestRuns(ctx, repoIDs, paths); err != nil {
		return nil, err
	}

	return panel, nil
}

func (c *Controller) quotaPanel(ctx context.Context, spaceID int64, repoIDs []int64) (QuotaPanel, error) {
	out := QuotaPanel{
		UsedByClass:      make(map[string]int, len(aiClasses)),
		RemainingByClass: make(map[string]int, len(aiClasses)),
	}

	dayStart := time.Now().UTC().Truncate(24 * time.Hour)
	if c.quota.Enabled() {
		snap := c.quota.Snapshot()
		out.Day, out.Limit, out.Used = snap.Day, snap.Limit, snap.Used
		for _, class := range aiClasses {
			out.UsedByClass[string(class)] = snap.UsedClass[class]
			out.RemainingByClass[string(class)] = c.quota.Remaining(class)
		}
		if d, err := time.Parse(time.DateOnly, snap.Day); err == nil {
			dayStart = d
		}
	}

	used, err := c.usageStore.SumRequestsSince(ctx, spaceID, repoIDs, dayStart.UnixMilli())
	if err != nil {
		return out, fmt.Errorf("failed to sum space ai usage: %w", err)
	}
	out.SpaceUsedToday = used

	return out, nil
}

func (c *Controller) activeTasks(
	ctx context.Context,
	repoIDs []int64,
	paths map[int64]string,
) ([]AgentTask, error) {
	isAgent := true
	tasks, err := c.taskStore.ListByRepos(ctx, repoIDs, &types.TaskFilter{
		Page:    1,
		Size:    maxActiveShown,
		States:  []enum.TaskState{enum.TaskStateInProgress, enum.TaskStateInReview},
		IsAgent: &isAgent,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list agent tasks: %w", err)
	}

	models, err := c.modelsOfTasks(ctx, tasks)
	if err != nil {
		return nil, err
	}

	out := make([]AgentTask, 0, len(tasks))
	for _, t := range tasks {
		at := mapTask(t, paths)
		at.Model = models[t.ID]
		out = append(out, at)
	}
	return out, nil
}

// modelsOfTasks returns the model of the latest AI task of every task, read from the launcher metadata.
func (c *Controller) modelsOfTasks(ctx context.Context, tasks []*types.Task) (map[int64]string, error) {
	out := make(map[int64]string, len(tasks))
	if len(tasks) == 0 {
		return out, nil
	}
	ids := make([]int64, len(tasks))
	for i, t := range tasks {
		ids[i] = t.ID
	}
	aiTasks, err := c.aiTaskStore.ListByTaskIDs(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("failed to list ai tasks of agent tasks: %w", err)
	}
	// the store returns the oldest first, so later AI tasks overwrite earlier ones.
	for _, at := range aiTasks {
		if at.TaskID == nil || len(at.OutputMetadata) == 0 {
			continue
		}
		var meta launcherMetadata
		if json.Unmarshal(at.OutputMetadata, &meta) == nil && meta.Model != "" {
			out[*at.TaskID] = meta.Model
		}
	}
	return out, nil
}

func (c *Controller) waitingQueue(
	ctx context.Context,
	repoIDs []int64,
	paths map[int64]string,
) (WaitingQueue, error) {
	out := WaitingQueue{Tasks: []AgentTask{}}

	count, err := c.taskStore.CountByRepos(ctx, repoIDs, &types.TaskFilter{
		States: []enum.TaskState{enum.TaskStateWaitingQuota},
	})
	if err != nil {
		return out, fmt.Errorf("failed to count tasks waiting for quota: %w", err)
	}
	out.Count = count
	if count == 0 {
		return out, nil
	}

	// the queue is global and ordered by resume order; keep the tasks of this space.
	waiting, err := c.taskStore.ListWaitingQuota(ctx, waitingScanLimit)
	if err != nil {
		return out, fmt.Errorf("failed to list tasks waiting for quota: %w", err)
	}
	for _, t := range waiting {
		if _, ok := paths[t.RepoID]; !ok {
			continue
		}
		out.Tasks = append(out.Tasks, mapTask(t, paths))
		if len(out.Tasks) == maxWaitingShown {
			break
		}
	}
	return out, nil
}

func (c *Controller) modelSuccess(ctx context.Context) ([]ModelSuccess, error) {
	models, err := c.modelStore.List(ctx, types.AIModelFilter{})
	if err != nil {
		return nil, fmt.Errorf("failed to list ai models: %w", err)
	}

	out := make([]ModelSuccess, 0, len(models))
	for _, m := range models {
		ms := ModelSuccess{Model: m.ID, Kinds: []KindSuccess{}}
		for kind, rate := range m.SuccessRateByTask {
			if strings.HasSuffix(kind, countKeySuffix) {
				continue
			}
			ms.Kinds = append(ms.Kinds, KindSuccess{
				Kind:    kind,
				Rate:    rate,
				Samples: int(m.SuccessRateByTask[kind+countKeySuffix]),
			})
		}
		if len(ms.Kinds) == 0 {
			continue
		}
		sort.Slice(ms.Kinds, func(i, j int) bool { return ms.Kinds[i].Kind < ms.Kinds[j].Kind })
		out = append(out, ms)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Model < out[j].Model })
	return out, nil
}

func (c *Controller) latestRuns(
	ctx context.Context,
	repoIDs []int64,
	paths map[int64]string,
) ([]AgentRun, error) {
	acts, err := c.activityStore.ListLatestByKindForRepos(
		ctx, repoIDs, enum.TaskActivityKindAgentRun, maxRunsShown)
	if err != nil {
		return nil, fmt.Errorf("failed to list agent runs: %w", err)
	}

	out := make([]AgentRun, 0, len(acts))
	tasks := make(map[int64]*types.Task, len(acts))
	for _, a := range acts {
		var p agentRunPayload
		if err = json.Unmarshal(a.Payload, &p); err != nil {
			continue
		}
		t, ok := tasks[a.TaskID]
		if !ok {
			if t, err = c.taskStore.Find(ctx, a.TaskID); err != nil {
				if errors.Is(err, gitness_store.ErrResourceNotFound) {
					continue
				}
				return nil, fmt.Errorf("failed to find task %d of agent run: %w", a.TaskID, err)
			}
			tasks[a.TaskID] = t
		}
		out = append(out, AgentRun{
			TaskNumber:   t.Number,
			RepoPath:     paths[t.RepoID],
			Title:        t.Title,
			Model:        p.Model,
			Outcome:      p.Outcome,
			RequestsUsed: p.RequestsUsed,
			Created:      a.Created,
		})
	}
	return out, nil
}

func mapTask(t *types.Task, paths map[int64]string) AgentTask {
	return AgentTask{
		Number:   t.Number,
		RepoPath: paths[t.RepoID],
		Title:    t.Title,
		State:    t.State,
		Priority: t.Priority,
		Updated:  t.Updated,
	}
}

// viewableRepos returns the repositories of the space and its descendants the principal can view.
func (c *Controller) viewableRepos(
	ctx context.Context,
	session *auth.Session,
	spaceID int64,
) ([]*types.RepositoryCore, error) {
	var out []*types.RepositoryCore

	for page := 1; ; page++ {
		repos, err := c.repoStore.List(ctx, spaceID, &types.RepoFilter{
			Page:      page,
			Size:      repoPageSize,
			Recursive: true,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to list repositories of space: %w", err)
		}

		for _, repo := range repos {
			core, findErr := c.repoFinder.FindByID(ctx, repo.ID)
			if errors.Is(findErr, gitness_store.ErrResourceNotFound) {
				continue
			}
			if findErr != nil {
				return nil, fmt.Errorf("failed to find repository %d: %w", repo.ID, findErr)
			}

			err = apiauth.CheckRepoState(ctx, session, core, enum.PermissionRepoView)
			if err == nil {
				err = apiauth.CheckRepo(ctx, c.authorizer, session, core, enum.PermissionRepoView)
			}
			switch {
			case err == nil:
				out = append(out, core)
			case errors.Is(err, apiauth.ErrForbidden):
				continue
			default:
				return nil, fmt.Errorf("failed to check access to repository %d: %w", repo.ID, err)
			}
		}

		if len(repos) < repoPageSize {
			return out, nil
		}
	}
}
