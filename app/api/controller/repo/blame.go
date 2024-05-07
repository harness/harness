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
	"strings"

	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

func (c *Controller) Blame(ctx context.Context,
	session *auth.Session,
	repoRef, gitRef, path string,
	lineFrom, lineTo int,
) (types.Stream[*git.BlamePart], error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, usererror.BadRequest("File path needs to specified.")
	}

	if lineTo > 0 && lineFrom > lineTo {
		return nil, usererror.BadRequest("Line range must be valid.")
	}

	repo, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoView)
	if err != nil {
		return nil, err
	}

	if gitRef == "" {
		gitRef = repo.DefaultBranch
	}

	reader := git.NewStreamReader(
		c.git.Blame(ctx, &git.BlameParams{
			ReadParams: git.CreateReadParams(repo),
			GitRef:     gitRef,
			Path:       path,
			LineFrom:   lineFrom,
			LineTo:     lineTo,
		}))

	return reader, nil
}
