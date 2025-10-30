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
	"github.com/harness/gitness/types/enum"
)

// FileViewDelete removes a file from being marked as viewed.
func (c *Controller) FileViewDelete(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	prNum int64,
	filePath string,
) error {
	repo, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoReview)
	if err != nil {
		return fmt.Errorf("failed to acquire access to repo: %w", err)
	}

	pr, err := c.pullreqStore.FindByNumber(ctx, repo.ID, prNum)
	if err != nil {
		return fmt.Errorf("failed to find pull request by number: %w", err)
	}

	if filePath == "" {
		return usererror.BadRequest("File path can't be empty")
	}

	err = c.fileViewStore.DeleteByFileForPrincipal(ctx, pr.ID, session.Principal.ID, filePath)
	if err != nil {
		return fmt.Errorf("failed to delete file view entry in db: %w", err)
	}

	return nil
}
