// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package service

import (
	"github.com/harness/gitness/gitrpc/internal/streamio"
	"github.com/harness/gitness/gitrpc/internal/types"
	"github.com/harness/gitness/gitrpc/rpc"
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

func (s DiffService) RawDiff(request *rpc.RawDiffRequest, stream rpc.DiffService_RawDiffServer) error {
	err := validateDiffRequest(request)
	if err != nil {
		return err
	}

	ctx := stream.Context()
	base := request.GetBase()

	sw := streamio.NewWriter(func(p []byte) error {
		return stream.Send(&rpc.RawDiffResponse{Data: p})
	})

	repoPath := getFullPathForRepo(s.reposRoot, base.GetRepoUid())

	args := []string{}
	if request.GetMergeBase() {
		args = []string{
			"--merge-base",
		}
	}

	return s.adapter.RawDiff(ctx, repoPath, request.GetBaseRef(), request.GetHeadRef(), sw, args...)
}

func validateDiffRequest(in *rpc.RawDiffRequest) error {
	if in.GetBase() == nil {
		return types.ErrBaseCannotBeEmpty
	}
	if in.GetBaseRef() == "" {
		return types.ErrEmptyBaseRef
	}
	if in.GetHeadRef() == "" {
		return types.ErrEmptyHeadRef
	}

	return nil
}
