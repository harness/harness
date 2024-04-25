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
	"errors"
	"fmt"

	apiauth "github.com/harness/gitness/app/api/auth"
	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/services/importer"
	"github.com/harness/gitness/job"
	"github.com/harness/gitness/types/enum"
)

// ImportProgress returns progress of the import job.
func (c *Controller) ImportProgress(ctx context.Context,
	session *auth.Session,
	repoRef string,
) (job.Progress, error) {
	// note: can't use c.getRepoCheckAccess because this needs to fetch a repo being imported.
	repo, err := c.repoStore.FindByRef(ctx, repoRef)
	if err != nil {
		return job.Progress{}, err
	}

	if err = apiauth.CheckRepo(ctx, c.authorizer, session, repo, enum.PermissionRepoView); err != nil {
		return job.Progress{}, err
	}

	progress, err := c.importer.GetProgress(ctx, repo)
	if errors.Is(err, importer.ErrNotFound) {
		return job.Progress{}, usererror.NotFound("No recent or ongoing import found for repository.")
	}
	if err != nil {
		return job.Progress{}, fmt.Errorf("failed to retrieve import progress: %w", err)
	}

	return progress, err
}
