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

package task

import (
	"context"
	"errors"
	"fmt"

	apiauth "github.com/harness/gitness/app/api/auth"
	"github.com/harness/gitness/app/auth"
	taskservice "github.com/harness/gitness/app/services/task"
	gitness_store "github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

// CreateInput is the input of Create. Tasks created through the API are manual tasks.
type CreateInput struct {
	Title       string                     `json:"title"`
	Description string                     `json:"description"`
	Priority    enum.TaskPriority          `json:"priority"`
	Due         *int64                     `json:"due"`
	Assignee    *taskservice.AssigneeInput `json:"assignee"`
	// ParentNumber is the number of a task of the same repository.
	ParentNumber *int64 `json:"parent_number"`
}

// UpdateInput is the input of Update. Fields that are not provided are left unchanged.
type UpdateInput struct {
	Title       *string            `json:"title"`
	Description *string            `json:"description"`
	Priority    *enum.TaskPriority `json:"priority"`
	Due         *int64             `json:"due"`
	ClearDue    bool               `json:"clear_due"`
	// Assignee replaces the assignee, an empty object unassigns the task.
	Assignee *taskservice.AssigneeInput `json:"assignee"`
}

// StateInput is the input of State.
type StateInput struct {
	State  enum.TaskState `json:"state"`
	Reason string         `json:"reason"`
}

// ResumeInput is the input of Resume.
type ResumeInput struct {
	Reason string `json:"reason"`
}

// CommentInput is the input of CreateComment.
type CommentInput struct {
	Text string `json:"text"`
}

// Create creates a manual task in the repository.
func (c *Controller) Create(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	in *CreateInput,
) (*types.Task, error) {
	r, err := c.getRepoCheckAccess(ctx, session, repoRef, permissionEdit)
	if err != nil {
		return nil, err
	}

	svcIn := taskservice.CreateInput{
		RepoID:      r.ID,
		Title:       in.Title,
		Description: in.Description,
		Origin:      enum.TaskOriginManual,
		Priority:    in.Priority,
		Due:         in.Due,
		Assignee:    in.Assignee,
	}

	if in.ParentNumber != nil {
		parent, findErr := c.taskStore.FindByNumber(ctx, r.ID, *in.ParentNumber)
		if findErr != nil {
			return nil, fmt.Errorf("failed to find parent task %d: %w", *in.ParentNumber, findErr)
		}
		svcIn.ParentID = &parent.ID
	}

	t, _, err := c.service.Create(ctx, session.Principal.ID, svcIn)
	if err != nil {
		return nil, err
	}

	return t, nil
}

// Find returns the task with the given number.
func (c *Controller) Find(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	taskNumber int64,
) (*types.Task, error) {
	_, t, err := c.getTaskCheckAccess(ctx, session, repoRef, taskNumber, permissionView)
	return t, err
}

// Update changes the editable fields of a task.
func (c *Controller) Update(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	taskNumber int64,
	in *UpdateInput,
) (*types.Task, error) {
	_, t, err := c.getTaskCheckAccess(ctx, session, repoRef, taskNumber, permissionEdit)
	if err != nil {
		return nil, err
	}

	return c.service.Update(ctx, session.Principal.ID, t.ID, taskservice.UpdateInput{
		Title:       in.Title,
		Description: in.Description,
		Priority:    in.Priority,
		Due:         in.Due,
		ClearDue:    in.ClearDue,
		Assignee:    in.Assignee,
	})
}

// State moves the task to a new state.
func (c *Controller) State(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	taskNumber int64,
	in *StateInput,
) (*types.Task, error) {
	_, t, err := c.getTaskCheckAccess(ctx, session, repoRef, taskNumber, permissionEdit)
	if err != nil {
		return nil, err
	}

	return c.service.Transition(ctx, session.Principal.ID, t.ID, in.State, in.Reason)
}

// Resume moves a waiting or blocked task back to the state it had before.
func (c *Controller) Resume(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	taskNumber int64,
	in *ResumeInput,
) (*types.Task, error) {
	_, t, err := c.getTaskCheckAccess(ctx, session, repoRef, taskNumber, permissionEdit)
	if err != nil {
		return nil, err
	}

	return c.service.Resume(ctx, session.Principal.ID, t.ID, in.Reason)
}

// CreateComment adds a comment to the history of the task.
func (c *Controller) CreateComment(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	taskNumber int64,
	in *CommentInput,
) error {
	_, t, err := c.getTaskCheckAccess(ctx, session, repoRef, taskNumber, permissionEdit)
	if err != nil {
		return err
	}

	return c.service.AddComment(ctx, session.Principal.ID, t.ID, in.Text)
}

// ListActivities returns a page of the history of the task, oldest first.
func (c *Controller) ListActivities(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	taskNumber int64,
	pagination types.Pagination,
) ([]*types.TaskActivity, error) {
	_, t, err := c.getTaskCheckAccess(ctx, session, repoRef, taskNumber, permissionView)
	if err != nil {
		return nil, err
	}

	list, err := c.activityStore.List(ctx, t.ID, pagination)
	if err != nil {
		return nil, fmt.Errorf("failed to list task activities: %w", err)
	}

	return list, nil
}

// ListLinks returns the links of the task.
func (c *Controller) ListLinks(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	taskNumber int64,
) ([]*types.TaskLink, error) {
	_, t, err := c.getTaskCheckAccess(ctx, session, repoRef, taskNumber, permissionView)
	if err != nil {
		return nil, err
	}

	list, err := c.linkStore.ListByTask(ctx, t.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to list task links: %w", err)
	}

	return list, nil
}

// List returns a page of the tasks of the repository and the total count.
func (c *Controller) List(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	filter *types.TaskFilter,
) ([]*types.Task, int64, error) {
	r, err := c.getRepoCheckAccess(ctx, session, repoRef, permissionView)
	if err != nil {
		return nil, 0, err
	}

	return c.listByRepos(ctx, []int64{r.ID}, filter)
}

func (c *Controller) listByRepos(
	ctx context.Context,
	repoIDs []int64,
	filter *types.TaskFilter,
) ([]*types.Task, int64, error) {
	list, err := c.taskStore.ListByRepos(ctx, repoIDs, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list tasks: %w", err)
	}

	if filter.Page == 1 && len(list) < filter.Size {
		return list, int64(len(list)), nil
	}

	count, err := c.taskStore.CountByRepos(ctx, repoIDs, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count tasks: %w", err)
	}

	return list, count, nil
}

// repoPageSize is the page size used when walking the repositories of a space.
const repoPageSize = 100

// ListForSpace returns a page of the tasks of the repositories of the space the principal can view,
// optionally including the repositories of the descendant spaces, and the total count.
func (c *Controller) ListForSpace(
	ctx context.Context,
	session *auth.Session,
	spaceRef string,
	recursive bool,
	filter *types.TaskFilter,
) ([]*types.TaskRepo, int64, error) {
	space, err := c.spaceFinder.FindByRef(ctx, spaceRef)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to find space: %w", err)
	}

	if err = apiauth.CheckSpaceScope(
		ctx, c.authorizer, session, space, enum.ResourceTypeRepo, permissionView,
	); err != nil {
		return nil, 0, fmt.Errorf("access check failed: %w", err)
	}

	repos, err := c.viewableRepos(ctx, session, space.ID, recursive)
	if err != nil {
		return nil, 0, err
	}

	repoIDs := make([]int64, 0, len(repos))
	paths := make(map[int64]string, len(repos))
	for _, r := range repos {
		repoIDs = append(repoIDs, r.ID)
		paths[r.ID] = r.Path
	}

	list, count, err := c.listByRepos(ctx, repoIDs, filter)
	if err != nil {
		return nil, 0, err
	}

	out := make([]*types.TaskRepo, len(list))
	for i, t := range list {
		out[i] = &types.TaskRepo{Task: t, RepoPath: paths[t.RepoID]}
	}

	return out, count, nil
}

// viewableRepos returns the repositories of the space the principal has view access to.
func (c *Controller) viewableRepos(
	ctx context.Context,
	session *auth.Session,
	spaceID int64,
	recursive bool,
) ([]*types.RepositoryCore, error) {
	var out []*types.RepositoryCore

	for page := 1; ; page++ {
		repos, err := c.repoStore.List(ctx, spaceID, &types.RepoFilter{
			Page:      page,
			Size:      repoPageSize,
			Recursive: recursive,
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

			err = apiauth.CheckRepoState(ctx, session, core, permissionView)
			if err == nil {
				err = apiauth.CheckRepo(ctx, c.authorizer, session, core, permissionView)
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
