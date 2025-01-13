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

package trigger

import (
	"context"
	"fmt"

	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/types/enum"
)

func (c *Controller) Delete(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	pipelineIdentifier string,
	triggerIdentifier string,
) error {
	repo, err := c.getRepoCheckPipelineAccess(ctx, session, repoRef, pipelineIdentifier, enum.PermissionPipelineEdit)
	if err != nil {
		return err
	}

	pipeline, err := c.pipelineStore.FindByIdentifier(ctx, repo.ID, pipelineIdentifier)
	if err != nil {
		return fmt.Errorf("failed to find pipeline: %w", err)
	}

	err = c.triggerStore.DeleteByIdentifier(ctx, pipeline.ID, triggerIdentifier)
	if err != nil {
		return fmt.Errorf("could not delete trigger: %w", err)
	}
	return nil
}
