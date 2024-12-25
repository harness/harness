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

package pipeline

import (
	"context"
	"fmt"

	apiauth "github.com/harness/gitness/app/api/auth"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/types/enum"
)

func (c *Controller) Delete(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	identifier string,
) error {
	repo, err := c.repoFinder.FindByRef(ctx, repoRef)
	if err != nil {
		return fmt.Errorf("failed to find repo by ref: %w", err)
	}

	err = apiauth.CheckPipeline(ctx, c.authorizer, session, repo.Path, identifier, enum.PermissionPipelineDelete)
	if err != nil {
		return fmt.Errorf("failed to authorize pipeline: %w", err)
	}

	err = c.pipelineStore.DeleteByIdentifier(ctx, repo.ID, identifier)
	if err != nil {
		return fmt.Errorf("could not delete pipeline: %w", err)
	}
	return nil
}
