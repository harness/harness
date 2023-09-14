// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package service

import (
	"context"
	"time"

	"github.com/harness/gitness/gitrpc/internal/types"
	"github.com/harness/gitness/gitrpc/rpc"

	"code.gitea.io/gitea/modules/git"
)

type PushService struct {
	rpc.UnimplementedPushServiceServer
	adapter   GitAdapter
	reposRoot string
}

var _ rpc.PushServiceServer = (*PushService)(nil)

func NewPushService(adapter GitAdapter, reposRoot string) *PushService {
	return &PushService{
		adapter:   adapter,
		reposRoot: reposRoot,
	}
}

func (s PushService) PushRemote(
	ctx context.Context,
	request *rpc.PushRemoteRequest,
) (*rpc.PushRemoteResponse, error) {
	base := request.GetBase()
	repoPath := getFullPathForRepo(s.reposRoot, base.GetRepoUid())
	repo, err := git.OpenRepository(ctx, repoPath)
	if err != nil {
		return nil, processGitErrorf(err, "failed to open repo")
	}
	if ok, err := repo.IsEmpty(); ok {
		return nil, ErrInvalidArgumentf("cannot push empty repo", err)
	}

	err = s.adapter.Push(ctx, repoPath, types.PushOptions{
		Remote:  request.RemoteUrlWithToken,
		Force:   false,
		Env:     nil,
		Timeout: time.Duration(request.Timeout),
		Mirror:  true,
	})
	if err != nil {
		return nil, err
	}
	return &rpc.PushRemoteResponse{}, nil
}
