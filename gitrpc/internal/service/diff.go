// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package service

import (
	"context"
	"fmt"

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

func (s DiffService) RawDiff(request *rpc.DiffRequest, stream rpc.DiffService_RawDiffServer) error {
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

func validateDiffRequest(in *rpc.DiffRequest) error {
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

func (s DiffService) DiffShortStat(ctx context.Context, r *rpc.DiffRequest) (*rpc.DiffShortStatResponse, error) {
	err := validateDiffRequest(r)
	if err != nil {
		return nil, fmt.Errorf("failed to validate request for short diff statistic "+
			"between %s and %s with err: %w", r.GetBaseRef(), r.GetHeadRef(), err)
	}

	base := r.GetBase()
	repoPath := getFullPathForRepo(s.reposRoot, base.GetRepoUid())

	// direct comparison
	// when direct is false then its like you use --merge-base
	// to find best common ancestor(s) between two refs
	direct := !r.GetMergeBase()

	stat, err := s.adapter.DiffShortStat(ctx, repoPath, r.GetBaseRef(), r.GetHeadRef(), direct)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch short statistics "+
			"between %s and %s with err: %w", r.GetBaseRef(), r.GetHeadRef(), err)
	}

	return &rpc.DiffShortStatResponse{
		Files:     int32(stat.Files),
		Additions: int32(stat.Additions),
		Deletions: int32(stat.Deletions),
	}, nil
}
