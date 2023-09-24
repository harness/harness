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

package service

import (
	"context"
	"fmt"
	"os"

	"github.com/harness/gitness/gitrpc/internal/types"
	"github.com/harness/gitness/gitrpc/rpc"

	"github.com/drone/go-generate/builder"
	"github.com/drone/go-generate/chroot"
	"github.com/rs/zerolog/log"
)

func (s RepositoryService) GeneratePipeline(ctx context.Context,
	request *rpc.GeneratePipelineRequest,
) (*rpc.GeneratePipelineResponse, error) {
	base := request.GetBase()
	if base == nil {
		return nil, types.ErrBaseCannotBeEmpty
	}
	repoPath := getFullPathForRepo(s.reposRoot, base.GetRepoUid())

	tempDir, err := os.MkdirTemp(s.tmpDir, "*-"+base.GetRepoUid())
	if err != nil {
		return nil, fmt.Errorf("error creating temp dir for repo %s: %w", base.GetRepoUid(), err)
	}
	defer func(path string) {
		// when repo is successfully created remove temp dir
		errRm := os.RemoveAll(path)
		if errRm != nil {
			log.Err(errRm).Msg("failed to cleanup temporary dir.")
		}
	}(tempDir)

	// Clone repository to temp dir
	if err = s.adapter.Clone(ctx, repoPath, tempDir, types.CloneRepoOptions{Depth: 1}); err != nil {
		return nil, processGitErrorf(err, "failed to clone repo")
	}

	// create a chroot virtual filesystem that we
	// pass to the builder for isolation purposes.
	chroot, err := chroot.New(tempDir)
	if err != nil {
		return nil, fmt.Errorf("failed to set the temp directory as active directory: %w", err)
	}

	// builds the pipeline configuration based on
	// the contents of the virtual filesystem.
	builder := builder.New()
	out, err := builder.Build(chroot)
	if err != nil {
		return nil, fmt.Errorf("failed to build pipeline: %w", err)
	}

	return &rpc.GeneratePipelineResponse{
		PipelineYaml: out,
	}, nil
}
