// Copyright 2026 Harness, Inc.
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

package commit

import (
	"context"
	"errors"
	"testing"

	"github.com/harness/gitness/git"
	"github.com/harness/gitness/git/sha"
	"github.com/harness/gitness/types"

	"github.com/stretchr/testify/require"
)

type infGit struct {
	git.Interface

	commit    git.Commit
	branchErr error
	commitErr error
	gotBranch *git.GetBranchParams
	gotCommit *git.GetCommitParams
}

func (g *infGit) GetBranch(_ context.Context, p *git.GetBranchParams) (*git.GetBranchOutput, error) {
	g.gotBranch = p
	if g.branchErr != nil {
		return nil, g.branchErr
	}
	return &git.GetBranchOutput{Branch: git.Branch{Name: p.BranchName, Commit: &g.commit}}, nil
}

func (g *infGit) GetCommit(_ context.Context, p *git.GetCommitParams) (*git.GetCommitOutput, error) {
	g.gotCommit = p
	if g.commitErr != nil {
		return nil, g.commitErr
	}
	return &git.GetCommitOutput{Commit: g.commit}, nil
}

func TestInfService_FindRefAndFindCommit(t *testing.T) {
	ctx := context.Background()
	repo := &types.RepositoryCore{GitUID: "git-uid"}
	g := &infGit{commit: git.Commit{
		SHA:     sha.Must("0123456789abcdef0123456789abcdef01234567"),
		Title:   "the title",
		Message: "the message",
	}}
	svc := newService(g)

	c, err := svc.FindRef(ctx, repo, "main")
	require.NoError(t, err)
	require.Equal(t, "the title", c.Title)
	require.Equal(t, "0123456789abcdef0123456789abcdef01234567", c.SHA.String())
	require.Equal(t, "git-uid", g.gotBranch.RepoUID)
	require.Equal(t, "main", g.gotBranch.BranchName)

	c, err = svc.FindCommit(ctx, repo, "abc123")
	require.NoError(t, err)
	require.Equal(t, "the message", c.Message)
	require.Equal(t, "git-uid", g.gotCommit.RepoUID)
	require.Equal(t, "abc123", g.gotCommit.Revision)

	g.branchErr = errors.New("no branch")
	_, err = svc.FindRef(ctx, repo, "main")
	require.ErrorContains(t, err, "no branch")

	g.commitErr = errors.New("no commit")
	_, err = svc.FindCommit(ctx, repo, "abc")
	require.ErrorContains(t, err, "no commit")
}
