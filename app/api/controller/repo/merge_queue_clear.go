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

package repo

import (
	"context"
	"fmt"
	"strings"

	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/store"
	"github.com/harness/gitness/types/enum"
)

// MergeQueueClear clears the merge queue for the given branch.
func (c *Controller) MergeQueueClear(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	branch string,
) error {
	branch = strings.TrimSpace(branch)
	if branch == "" {
		return usererror.BadRequest("Branch must be provided.")
	}

	repo, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoEdit)
	if err != nil {
		return fmt.Errorf("failed to acquire access to repo: %w", err)
	}

	err = c.mergeQueueService.RemoveAll(ctx, repo, branch, enum.MergeQueueRemovalReasonManual)
	if err != nil && !errors.Is(err, store.ErrResourceNotFound) {
		return fmt.Errorf("failed to remove all merge queue entries: %w", err)
	}

	return nil
}
