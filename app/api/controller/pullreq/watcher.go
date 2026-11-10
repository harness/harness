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

package pullreq

import (
	"context"
	"fmt"

	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

// Watch subscribes the calling principal to notifications of the given pull request.
// The operation is idempotent.
func (c *Controller) Watch(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	prNum int64,
) error {
	prID, err := c.getWatchTargetPullReqID(ctx, session, repoRef, prNum)
	if err != nil {
		return err
	}

	if err = c.watcherStore.Create(ctx, prID, session.Principal.ID); err != nil {
		return fmt.Errorf("failed to create pull request watcher: %w", err)
	}

	return nil
}

// Unwatch unsubscribes the calling principal from notifications of the given pull request.
// The operation is idempotent.
func (c *Controller) Unwatch(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	prNum int64,
) error {
	prID, err := c.getWatchTargetPullReqID(ctx, session, repoRef, prNum)
	if err != nil {
		return err
	}

	if err = c.watcherStore.Delete(ctx, prID, session.Principal.ID); err != nil {
		return fmt.Errorf("failed to delete pull request watcher: %w", err)
	}

	return nil
}

// WatcherList returns the list of principals watching the given pull request.
func (c *Controller) WatcherList(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	prNum int64,
) ([]*types.PullReqWatcher, error) {
	repo, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoView)
	if err != nil {
		return nil, fmt.Errorf("failed to acquire access to repo: %w", err)
	}

	pr, err := c.pullreqStore.FindByNumber(ctx, repo.ID, prNum)
	if err != nil {
		return nil, fmt.Errorf("failed to find pull request: %w", err)
	}

	watchers, err := c.watcherStore.List(ctx, pr.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to list pull request watchers: %w", err)
	}

	return watchers, nil
}

// getWatchTargetPullReqID checks that the caller is allowed to watch pull requests of the repo
// and returns the internal ID of the pull request.
func (c *Controller) getWatchTargetPullReqID(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	prNum int64,
) (int64, error) {
	if auth.IsAnonymousSession(session) {
		return 0, usererror.ErrUnauthorized
	}

	repo, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoView)
	if err != nil {
		return 0, fmt.Errorf("failed to acquire access to repo: %w", err)
	}

	pr, err := c.pullreqStore.FindByNumber(ctx, repo.ID, prNum)
	if err != nil {
		return 0, fmt.Errorf("failed to find pull request: %w", err)
	}

	return pr.ID, nil
}
