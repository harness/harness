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

package repo

import (
	"context"

	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/services/codeowners"
	"github.com/harness/gitness/types/enum"
)

// CodeOwnersRaw returns the CODEOWNERS file at the given ref exactly as committed.
func (c *Controller) CodeOwnersRaw(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	gitRef string,
) (*codeowners.File, error) {
	repo, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoView)
	if err != nil {
		return nil, err
	}

	return c.codeOwners.GetRaw(ctx, repo, gitRef)
}
