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

// Package task provides the controller of the repository scoped Tasks API.
package task

import (
	"context"
	"fmt"

	apiauth "github.com/harness/gitness/app/api/auth"
	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/auth/authz"
	"github.com/harness/gitness/app/services/refcache"
	taskservice "github.com/harness/gitness/app/services/task"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/google/wire"
)

// WireSet provides a wire set for this package.
var WireSet = wire.NewSet(
	ProvideController,
)

type (
	repoFinder interface {
		FindByRef(ctx context.Context, repoRef string) (*types.RepositoryCore, error)
		FindByID(ctx context.Context, repoID int64) (*types.RepositoryCore, error)
	}

	spaceFinder interface {
		FindByRef(ctx context.Context, spaceRef string) (*types.SpaceCore, error)
	}

	repoLister interface {
		List(ctx context.Context, parentID int64, filter *types.RepoFilter) ([]*types.Repository, error)
	}

	// taskService is the part of the task domain service used by the controller.
	taskService interface {
		Create(ctx context.Context, principalID int64, in taskservice.CreateInput) (*types.Task, bool, error)
		Update(ctx context.Context, principalID int64, taskID int64, in taskservice.UpdateInput) (*types.Task, error)
		Transition(
			ctx context.Context, principalID int64, taskID int64, to enum.TaskState, reason string,
		) (*types.Task, error)
		Resume(ctx context.Context, principalID int64, taskID int64, reason string) (*types.Task, error)
		AddComment(ctx context.Context, principalID int64, taskID int64, text string) error
	}
)

// Controller handles the Tasks API.
type Controller struct {
	authorizer    authz.Authorizer
	repoFinder    repoFinder
	spaceFinder   spaceFinder
	repoStore     repoLister
	taskStore     store.TaskStore
	activityStore store.TaskActivityStore
	linkStore     store.TaskLinkStore
	service       taskService
}

// NewController returns a new Controller.
func NewController(
	authorizer authz.Authorizer,
	repoFinder repoFinder,
	spaceFinder spaceFinder,
	repoStore repoLister,
	taskStore store.TaskStore,
	activityStore store.TaskActivityStore,
	linkStore store.TaskLinkStore,
	service taskService,
) *Controller {
	return &Controller{
		authorizer:    authorizer,
		repoFinder:    repoFinder,
		spaceFinder:   spaceFinder,
		repoStore:     repoStore,
		taskStore:     taskStore,
		activityStore: activityStore,
		linkStore:     linkStore,
		service:       service,
	}
}

// ProvideController provides the Tasks controller.
func ProvideController(
	authorizer authz.Authorizer,
	repoFinder refcache.RepoFinder,
	spaceFinder refcache.SpaceFinder,
	repoStore store.RepoStore,
	taskStore store.TaskStore,
	activityStore store.TaskActivityStore,
	linkStore store.TaskLinkStore,
	service *taskservice.Service,
) *Controller {
	return NewController(
		authorizer, repoFinder, spaceFinder, repoStore, taskStore, activityStore, linkStore, service)
}

// Permissions of the Tasks API. Reading needs view access to the repository. Everything that
// changes a task needs push access, the same permission used to create and change pull requests.
const (
	permissionView = enum.PermissionRepoView
	permissionEdit = enum.PermissionRepoPush
)

func (c *Controller) getRepoCheckAccess(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	reqPermission enum.Permission,
) (*types.RepositoryCore, error) {
	if repoRef == "" {
		return nil, usererror.BadRequest("A valid repository reference must be provided.")
	}

	r, err := c.repoFinder.FindByRef(ctx, repoRef)
	if err != nil {
		return nil, fmt.Errorf("failed to find repository: %w", err)
	}

	if err = apiauth.CheckRepoState(ctx, session, r, reqPermission); err != nil {
		return nil, err
	}

	if err = apiauth.CheckRepo(ctx, c.authorizer, session, r, reqPermission); err != nil {
		return nil, fmt.Errorf("access check failed: %w", err)
	}

	return r, nil
}

// getTaskCheckAccess finds the repository and the task by number checking the permission on the repository.
func (c *Controller) getTaskCheckAccess(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	taskNumber int64,
	reqPermission enum.Permission,
) (*types.RepositoryCore, *types.Task, error) {
	r, err := c.getRepoCheckAccess(ctx, session, repoRef, reqPermission)
	if err != nil {
		return nil, nil, err
	}

	t, err := c.taskStore.FindByNumber(ctx, r.ID, taskNumber)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to find task %d: %w", taskNumber, err)
	}

	return r, t, nil
}
