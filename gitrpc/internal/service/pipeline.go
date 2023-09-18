// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

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
