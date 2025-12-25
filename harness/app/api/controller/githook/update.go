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

package githook

import (
	"context"
	"fmt"

	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/git/hook"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

// Update executes the update hook for a git repository.
func (c *Controller) Update(
	ctx context.Context,
	rgit RestrictedGIT,
	session *auth.Session,
	in types.GithookUpdateInput,
) (hook.Output, error) {
	repo, err := c.getRepoCheckAccess(ctx, session, in.RepoID, enum.PermissionRepoPush)
	if err != nil {
		return hook.Output{}, err
	}

	output := hook.Output{}

	err = c.updateExtender.Extend(ctx, rgit, session, repo, in, &output)
	if err != nil {
		return hook.Output{}, fmt.Errorf("failed to extend update hook: %w", err)
	}

	// We currently don't have any update action (nothing planned as of now)
	return hook.Output{}, nil
}
