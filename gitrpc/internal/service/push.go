// Copyright 2023 Harness, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package service

import (
	"context"

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
		Remote: request.RemoteUrl,
		Force:  false,
		Env:    nil,
		Mirror: true,
	})
	if err != nil {
		return nil, err
	}
	return &rpc.PushRemoteResponse{}, nil
}
