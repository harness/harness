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

	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/types/enum"
)

// PipelineGenerate returns automatically generate pipeline YAML for a repository.
func (c *Controller) PipelineGenerate(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
) ([]byte, error) {
	repo, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoView)
	if err != nil {
		return nil, err
	}

	result, err := c.git.GeneratePipeline(ctx, &git.GeneratePipelineParams{
		ReadParams: git.CreateReadParams(repo),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to generate pipeline: %w", err)
	}

	return result.PipelineYAML, nil
}
