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

package merge

import (
	"context"
	"errors"
	"testing"

	"github.com/harness/gitness/app/services/checkreq"
	"github.com/harness/gitness/app/services/codeowners"
	"github.com/harness/gitness/app/services/protection"
	"github.com/harness/gitness/app/services/usergroup"
	"github.com/harness/gitness/app/store"
	sv5errs "github.com/harness/gitness/errors"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/git/sha"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/require"
)

var errSv5Boom = errors.New("boom")

const (
	sv5SHAa = "1111111111111111111111111111111111111111"
	sv5SHAb = "2222222222222222222222222222222222222222"
	sv5SHAc = "3333333333333333333333333333333333333333"
)

type sv5Git struct {
	git.Interface

	headSHA     sha.SHA
	getRefErr   error
	branchSHA   sha.SHA
	branchErr   error
	ancestor    bool
	ancestorErr error
	treeNodeErr error
}

func (g *sv5Git) GetRef(context.Context, git.GetRefParams) (git.GetRefResponse, error) {
	return git.GetRefResponse{SHA: g.headSHA}, g.getRefErr
}

func (g *sv5Git) GetBranch(context.Context, *git.GetBranchParams) (*git.GetBranchOutput, error) {
	if g.branchErr != nil {
		return nil, g.branchErr
	}
	return &git.GetBranchOutput{Branch: git.Branch{SHA: g.branchSHA}}, nil
}

func (g *sv5Git) IsAncestor(context.Context, git.IsAncestorParams) (git.IsAncestorOutput, error) {
	return git.IsAncestorOutput{Ancestor: g.ancestor}, g.ancestorErr
}

func (g *sv5Git) GetTreeNode(context.Context, *git.GetTreeNodeParams) (*git.GetTreeNodeOutput, error) {
	if g.treeNodeErr != nil {
		return nil, g.treeNodeErr
	}
	return nil, sv5errs.NotFound("no codeowners file")
}

func sv5PR() *types.PullReq {
	return &types.PullReq{
		Number:       12,
		Title:        "Add feature",
		SourceBranch: "feature",
		TargetBranch: "main",
		SourceSHA:    sv5SHAa,
		Author:       types.PrincipalInfo{ID: 5, Email: "author@example.com", DisplayName: "Author"},
	}
}

func TestSv5PrepareInputNoRefUpdates(t *testing.T) {
	svc := &Service{}
	principal := &types.PrincipalInfo{ID: 9, Email: "me@example.com", DisplayName: "Me"}
	repo := &types.RepositoryCore{Path: "space/repo"}

	t.Run("rebase uses principal as committer", func(t *testing.T) {
		in := svc.PreparePullReqMergeInputNoRefUpdates(sv5PR(), repo, principal, enum.MergeMethodRebase, "T", "M")
		require.Nil(t, in.Author)
		require.NotNil(t, in.Committer)
		require.Equal(t, "me@example.com", in.Committer.Email)
		require.Equal(t, git.CommitMessage("T", "M"), in.CommitMessage)
		require.Equal(t, "main", in.TargetBranch)
		require.Nil(t, in.RefUpdates)
		require.Equal(t, sha.None, in.SourceSHA)
	})

	t.Run("fast forward has no identities", func(t *testing.T) {
		in := svc.PreparePullReqMergeInputNoRefUpdates(sv5PR(), repo, principal, enum.MergeMethodFastForward, "", "")
		require.Nil(t, in.Author)
		require.Nil(t, in.Committer)
		require.Equal(t, git.CommitMessage("", ""), in.CommitMessage)
	})
}

func TestSv5PreparePullReqMergeInput(t *testing.T) {
	svc := &Service{}
	principal := &types.PrincipalInfo{ID: 9, Email: "me@example.com"}
	targetSHA := sha.Must(sv5SHAb)

	in, err := svc.PreparePullReqMergeInput(
		sv5PR(), nil, targetSHA, principal, enum.MergeMethodFastForward, "title", "")
	require.NoError(t, err)
	require.Equal(t, sha.Must(sv5SHAa), in.SourceSHA)
	require.Len(t, in.RefUpdates, 3)

	require.Equal(t, "refs/heads/main", in.RefUpdates[0].Name)
	require.Equal(t, targetSHA, in.RefUpdates[0].Old)
	require.Equal(t, sha.SHA{}, in.RefUpdates[0].New)

	require.Equal(t, "refs/pullreq/12/head", in.RefUpdates[1].Name)
	require.Equal(t, sha.Must(sv5SHAa), in.RefUpdates[1].New)

	require.Equal(t, "refs/pullreq/12/merge", in.RefUpdates[2].Name)
	require.Equal(t, sha.Nil, in.RefUpdates[2].New)

	pr := sv5PR()
	pr.SourceSHA = "not-a-sha"
	_, err = svc.PreparePullReqMergeInput(pr, nil, targetSHA, principal, enum.MergeMethodFastForward, "t", "")
	require.ErrorContains(t, err, "failed to convert source SHA")
}

func TestSv5GetTargetSourceSHAs(t *testing.T) {
	ctx := context.Background()
	repo := &types.RepositoryCore{GitUID: "uid"}
	ok := func() *sv5Git {
		return &sv5Git{
			headSHA:   sha.Must(sv5SHAa),
			branchSHA: sha.Must(sv5SHAb),
			ancestor:  true,
		}
	}

	tests := []struct {
		name   string
		mutate func(*sv5Git, *types.PullReq)
		msg    string
	}{
		{"head ref", func(g *sv5Git, _ *types.PullReq) { g.getRefErr = errSv5Boom }, "head ref"},
		{"bad source sha", func(_ *sv5Git, pr *types.PullReq) { pr.SourceSHA = "x" }, "source SHA not valid"},
		{"head mismatch", func(g *sv5Git, _ *types.PullReq) { g.headSHA = sha.Must(sv5SHAc) }, "doesn't match"},
		{"branch", func(g *sv5Git, _ *types.PullReq) { g.branchErr = errSv5Boom }, "target branch"},
		{"ancestry", func(g *sv5Git, _ *types.PullReq) { g.ancestorErr = errSv5Boom }, "ancestry"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			g, pr := ok(), sv5PR()
			tc.mutate(g, pr)
			_, _, _, err := (&Service{git: g}).GetTargetSourceSHAs(ctx, repo, pr)
			require.ErrorContains(t, err, tc.msg)
		})
	}

	target, source, anc, err := (&Service{git: ok()}).GetTargetSourceSHAs(ctx, repo, sv5PR())
	require.NoError(t, err)
	require.Equal(t, sha.Must(sv5SHAb), target)
	require.Equal(t, sha.Must(sv5SHAa), source)
	require.True(t, anc)
}

type sv5Reviewers struct {
	store.PullReqReviewerStore

	err error
}

func (r sv5Reviewers) List(context.Context, int64) ([]*types.PullReqReviewer, error) {
	return nil, r.err
}

type sv5Checks struct {
	store.CheckStore

	results []types.CheckResult
	err     error
}

func (c sv5Checks) ListResults(context.Context, int64, string) ([]types.CheckResult, error) {
	return c.results, c.err
}

type sv5UserGroups struct{ usergroup.Service }

type sv5Protection struct {
	protection.BranchProtection

	out        protection.MergeVerifyOutput
	violations []types.RuleViolations
	err        error
	gotInput   protection.MergeVerifyInput
}

func (p *sv5Protection) MergeVerify(
	_ context.Context, in protection.MergeVerifyInput,
) (protection.MergeVerifyOutput, []types.RuleViolations, error) {
	p.gotInput = in
	return p.out, p.violations, p.err
}

type sv5Provider struct {
	req checkreq.Requirements
	err error
}

func (p sv5Provider) RequiredChecks(context.Context, *types.RepositoryCore, *types.PullReq) (
	checkreq.Requirements, error,
) {
	return p.req, p.err
}

func sv5RulesService(g *sv5Git, rev sv5Reviewers, chk sv5Checks, prov checkreq.Provider) *Service {
	return &Service{
		git:               g,
		reviewerStore:     rev,
		checkStore:        chk,
		codeOwners:        codeowners.New(nil, g, codeowners.Config{FilePaths: []string{"CODEOWNERS"}}, nil, nil),
		userGroupService:  sv5UserGroups{},
		checkRequirements: prov,
	}
}

func TestSv5CheckRules(t *testing.T) {
	ctx := context.Background()
	in := CheckRulesInput{
		PullReq:     &types.PullReq{ID: 1, TargetRepoID: 2, SourceSHA: sv5SHAa},
		TargetRepo:  &types.RepositoryCore{ID: 2},
		MergeMethod: enum.MergeMethodSquash,
		IsAncestor:  true,
	}

	t.Run("reviewer error", func(t *testing.T) {
		s := sv5RulesService(&sv5Git{}, sv5Reviewers{err: errSv5Boom}, sv5Checks{}, nil)
		_, _, err := s.CheckRules(ctx, &sv5Protection{}, in)
		require.ErrorIs(t, err, errSv5Boom)
	})

	t.Run("check results error", func(t *testing.T) {
		s := sv5RulesService(&sv5Git{}, sv5Reviewers{}, sv5Checks{err: errSv5Boom}, nil)
		_, _, err := s.CheckRules(ctx, &sv5Protection{}, in)
		require.ErrorContains(t, err, "status checks")
	})

	t.Run("codeowners error", func(t *testing.T) {
		g := &sv5Git{treeNodeErr: errSv5Boom}
		s := sv5RulesService(g, sv5Reviewers{}, sv5Checks{}, nil)
		_, _, err := s.CheckRules(ctx, &sv5Protection{}, in)
		require.ErrorContains(t, err, "CODEOWNERS evaluation failed")
	})

	t.Run("protection error", func(t *testing.T) {
		s := sv5RulesService(&sv5Git{}, sv5Reviewers{}, sv5Checks{}, nil)
		_, _, err := s.CheckRules(ctx, &sv5Protection{err: errSv5Boom}, in)
		require.ErrorContains(t, err, "verify protection rules")
	})

	t.Run("requirements error", func(t *testing.T) {
		s := sv5RulesService(&sv5Git{}, sv5Reviewers{}, sv5Checks{}, sv5Provider{err: errSv5Boom})
		_, _, err := s.CheckRules(ctx, &sv5Protection{}, in)
		require.ErrorContains(t, err, "resolve external checks")
	})

	t.Run("success with unsatisfied external check", func(t *testing.T) {
		prot := &sv5Protection{
			out:        protection.MergeVerifyOutput{MinimumRequiredApprovalsCount: 2},
			violations: []types.RuleViolations{{Rule: types.RuleInfo{Identifier: "r"}}},
		}
		prov := sv5Provider{req: checkreq.Requirements{Required: []string{"ci/build"}}}
		s := sv5RulesService(&sv5Git{}, sv5Reviewers{}, sv5Checks{}, prov)

		out, violations, err := s.CheckRules(ctx, prot, in)
		require.NoError(t, err)
		require.Equal(t, 2, out.MinimumRequiredApprovalsCount)
		require.Len(t, violations, 2)
		require.False(t, prot.gotInput.OmitMQViolations)
		require.True(t, prot.gotInput.TargetIsAncestor)
		require.Equal(t, enum.MergeMethodSquash, prot.gotInput.Method)

		_, _, err = s.CheckRulesForMergeQueue(ctx, prot, in)
		require.NoError(t, err)
		require.True(t, prot.gotInput.OmitMQViolations)

		resolved, err := s.RequiredChecks(ctx, in.TargetRepo, in.PullReq)
		require.NoError(t, err)
		require.Equal(t, []string{"ci/build"}, resolved.Unsatisfied(nil))
	})

	t.Run("codeowners not found is tolerated", func(t *testing.T) {
		g := &sv5Git{}
		s := sv5RulesService(g, sv5Reviewers{}, sv5Checks{}, nil)
		_, _, err := s.CheckRules(ctx, &sv5Protection{}, in)
		require.NoError(t, err)
	})
}
