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

package git

import (
	"context"
	"fmt"

	"github.com/harness/gitness/git/api"

	"github.com/drone/go-generate/builder"
)

type GeneratePipelineParams struct {
	ReadParams
}

type GeneratePipelinesOutput struct {
	PipelineYAML []byte
}

func (s *Service) GeneratePipeline(ctx context.Context,
	params *GeneratePipelineParams,
) (GeneratePipelinesOutput, error) {
	if err := params.Validate(); err != nil {
		return GeneratePipelinesOutput{}, err
	}
	repoPath := getFullPathForRepo(s.reposRoot, params.RepoUID)

	sha, err := s.git.ResolveRev(ctx, repoPath, "HEAD")
	if err != nil {
		return GeneratePipelinesOutput{}, fmt.Errorf("failed to resolve HEAD revision: %w", err)
	}

	ctxFS, cancelFn := context.WithCancel(ctx)
	defer cancelFn()

	gitFS := api.NewFS(ctxFS, sha.String(), repoPath)

	// builds the pipeline configuration based on
	// the contents of the virtual filesystem.
	builder := builder.New()
	out, err := builder.Build(gitFS)
	if err != nil {
		return GeneratePipelinesOutput{}, fmt.Errorf("failed to build pipeline: %w", err)
	}

	return GeneratePipelinesOutput{
		PipelineYAML: out,
	}, nil
}
