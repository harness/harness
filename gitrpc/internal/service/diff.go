// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package service

import (
	"fmt"
	"os"
	"time"

	"github.com/harness/gitness/gitrpc/internal/streamio"
	"github.com/harness/gitness/gitrpc/internal/types"
	"github.com/harness/gitness/gitrpc/rpc"

	"code.gitea.io/gitea/modules/git"
	"code.gitea.io/gitea/modules/setting"
)

type DiffService struct {
	rpc.UnimplementedDiffServiceServer
	adapter   GitAdapter
	reposRoot string
}

func NewDiffService(adapter GitAdapter, reposRoot string) (*DiffService, error) {
	return &DiffService{
		adapter:   adapter,
		reposRoot: reposRoot,
	}, nil
}

func (s DiffService) RawDiff(req *rpc.RawDiffRequest, stream rpc.DiffService_RawDiffServer) error {
	err := validateDiffRequest(req)
	if err != nil {
		return err
	}

	ctx := stream.Context()

	sw := streamio.NewWriter(func(p []byte) error {
		return stream.Send(&rpc.RawDiffResponse{Data: p})
	})

	repoPath := getFullPathForRepo(s.reposRoot, req.GetRepoId())

	cmd := git.NewCommand(ctx, "diff", "--full-index", req.LeftCommitId, req.RightCommitId)
	cmd.SetDescription(fmt.Sprintf("GetDiffRange [repo_path: %s]", repoPath))
	return cmd.Run(&git.RunOpts{
		Timeout: time.Duration(setting.Git.Timeout.Default) * time.Second,
		Dir:     repoPath,
		Stderr:  os.Stderr,
		Stdout:  sw,
	})
}

type requestWithLeftRightCommitIds interface {
	GetLeftCommitId() string
	GetRightCommitId() string
}

func validateDiffRequest(in requestWithLeftRightCommitIds) error {
	if in.GetLeftCommitId() == "" {
		return types.ErrEmptyLeftCommitID
	}
	if in.GetRightCommitId() == "" {
		return types.ErrEmptyRightCommitID
	}

	return nil
}
