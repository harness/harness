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

package repo

import (
	"context"
	"net/http"
	"testing"

	"github.com/harness/gitness/git"
	"github.com/harness/gitness/git/sha"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const crpUpdateForbidden = `{"lifecycle":{"update_forbidden":true}}`

// crpMergeIn is the common input of the merge commit, rebase and squash operations.
type crpMergeIn struct {
	baseBranch  string
	baseSHA     sha.SHA
	headBranch  string
	headSHA     sha.SHA
	title       string
	message     string
	dryRun      bool
	dryRunRules bool
	bypass      bool
}

// crpMergeResult normalizes the outputs of the merge commit, rebase and squash operations.
type crpMergeResult struct {
	newSHA         sha.SHA
	alreadyAnc     bool
	dryRun         bool
	dryRunRules    bool
	conflicts      []string
	ruleViolations []types.RuleViolations
	blocked        *types.MergeViolations
	err            error
}

type crpMergeOp struct {
	name        string
	hasAncestor bool
	call        func(c *Controller, in crpMergeIn) crpMergeResult
}

func crpMergeOps() []crpMergeOp {
	ctx := context.Background()
	s := crpSession()

	return []crpMergeOp{
		{
			name:        "MergeCommit",
			hasAncestor: true,
			call: func(c *Controller, in crpMergeIn) crpMergeResult {
				out, blocked, err := c.MergeCommit(ctx, s, crpRepoRef, &MergeCommitInput{
					BaseBranch: in.baseBranch, BaseCommitSHA: in.baseSHA, HeadBranch: in.headBranch,
					HeadCommitSHA: in.headSHA, Title: in.title, Message: in.message,
					DryRun: in.dryRun, DryRunRules: in.dryRunRules, BypassRules: in.bypass,
				})
				if err != nil || blocked != nil {
					return crpMergeResult{err: err, blocked: blocked}
				}

				return crpMergeResult{
					newSHA: out.NewHeadBranchSHA, alreadyAnc: out.AlreadyAncestor, dryRun: out.DryRun,
					dryRunRules: out.DryRunRules, conflicts: out.ConflictFiles, ruleViolations: out.RuleViolations,
				}
			},
		},
		{
			name:        "Rebase",
			hasAncestor: true,
			call: func(c *Controller, in crpMergeIn) crpMergeResult {
				out, blocked, err := c.Rebase(ctx, s, crpRepoRef, &RebaseInput{
					BaseBranch: in.baseBranch, BaseCommitSHA: in.baseSHA, HeadBranch: in.headBranch,
					HeadCommitSHA: in.headSHA, DryRun: in.dryRun, DryRunRules: in.dryRunRules,
					BypassRules: in.bypass,
				})
				if err != nil || blocked != nil {
					return crpMergeResult{err: err, blocked: blocked}
				}

				return crpMergeResult{
					newSHA: out.NewHeadBranchSHA, alreadyAnc: out.AlreadyAncestor, dryRun: out.DryRun,
					dryRunRules: out.DryRunRules, conflicts: out.ConflictFiles, ruleViolations: out.RuleViolations,
				}
			},
		},
		{
			name: "Squash",
			call: func(c *Controller, in crpMergeIn) crpMergeResult {
				out, blocked, err := c.Squash(ctx, s, crpRepoRef, &SquashInput{
					BaseBranch: in.baseBranch, BaseCommitSHA: in.baseSHA, HeadBranch: in.headBranch,
					HeadCommitSHA: in.headSHA, Title: in.title, Message: in.message,
					DryRun: in.dryRun, DryRunRules: in.dryRunRules, BypassRules: in.bypass,
				})
				if err != nil || blocked != nil {
					return crpMergeResult{err: err, blocked: blocked}
				}

				return crpMergeResult{
					newSHA: out.NewHeadBranchSHA, dryRun: out.DryRun, dryRunRules: out.DryRunRules,
					conflicts: out.ConflictFiles, ruleViolations: out.RuleViolations,
				}
			},
		},
	}
}

func crpDefaultMergeIn() crpMergeIn {
	return crpMergeIn{baseBranch: crpDefaultBranch, headBranch: crpHeadBranch, headSHA: crpTestSHA}
}

// crpMergeEnv prepares an environment where the head branch points to crpTestSHA and main to crpTestSHAOther.
func crpMergeEnv(ctx context.Context, t *testing.T, op crpMergeOp) *crpEnv {
	t.Helper()

	e := crpNewEnv(t)
	e.withMergeQueue(ctx, t, &crpMQEntryStore{})
	e.git.On("GetBranch", mock.Anything, mock.MatchedBy(func(p *git.GetBranchParams) bool {
		return p.BranchName == crpHeadBranch
	})).Return(&git.GetBranchOutput{Branch: crpBranch(crpHeadBranch, crpTestSHA)}, nil).Maybe()
	e.git.On("GetBranch", mock.Anything, mock.MatchedBy(func(p *git.GetBranchParams) bool {
		return p.BranchName == crpDefaultBranch
	})).Return(&git.GetBranchOutput{Branch: crpBranch(crpDefaultBranch, crpTestSHAOther)}, nil).Maybe()

	if op.hasAncestor {
		e.git.On("IsAncestor", mock.Anything, mock.Anything).
			Return(git.IsAncestorOutput{Ancestor: false}, nil).Maybe()
	} else {
		e.git.On("MergeBase", mock.Anything, mock.Anything).
			Return(git.MergeBaseOutput{MergeBaseSHA: crpTestSHAOther}, nil).Maybe()
		e.git.On("ListCommits", mock.Anything, mock.Anything).Return(&git.ListCommitsOutput{
			Commits:      []git.Commit{{SHA: crpTestSHA, Title: "only", Message: "only\n\nbody text"}},
			TotalCommits: 1,
		}, nil).Maybe()
	}

	return e
}

func TestCrpMergeOps_Validation(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(in *crpMergeIn)
	}{
		{name: "missing base", mutate: func(in *crpMergeIn) { in.baseBranch = "" }},
		{name: "missing head branch", mutate: func(in *crpMergeIn) { in.headBranch = "" }},
		{name: "missing head sha", mutate: func(in *crpMergeIn) { in.headSHA = sha.SHA{} }},
	}

	for _, op := range crpMergeOps() {
		for _, tt := range tests {
			t.Run(op.name+"/"+tt.name, func(t *testing.T) {
				e := crpNewEnv(t)
				in := crpDefaultMergeIn()
				tt.mutate(&in)

				res := op.call(e.c, in)

				require.Error(t, res.err)
				assert.Equal(t, http.StatusBadRequest, crpUserErrStatus(t, res.err))
			})
		}
	}
}

func TestCrpMergeOps(t *testing.T) {
	ctx := context.Background()

	merged := git.MergeOutput{MergeSHA: crpTestSHAOther}

	for _, op := range crpMergeOps() {
		t.Run(op.name+"/merge queue lookup failure", func(t *testing.T) {
			e := crpMergeEnv(ctx, t, op)
			e.withMergeQueue(ctx, t, &crpMQEntryStore{err: errCrpBoom})

			res := op.call(e.c, crpDefaultMergeIn())

			require.ErrorIs(t, res.err, errCrpBoom)
		})

		t.Run(op.name+"/rule store failure", func(t *testing.T) {
			e := crpMergeEnv(ctx, t, op)
			e.ruleStore.err = errCrpBoom

			res := op.call(e.c, crpDefaultMergeIn())

			require.ErrorIs(t, res.err, errCrpBoom)
		})

		t.Run(op.name+"/dry run rules reports violations", func(t *testing.T) {
			e := crpMergeEnv(ctx, t, op)
			e.withMergeQueue(ctx, t, &crpMQEntryStore{queued: map[string]struct{}{crpHeadBranch: {}}})
			in := crpDefaultMergeIn()
			in.dryRunRules = true

			res := op.call(e.c, in)

			require.NoError(t, res.err)
			assert.True(t, res.dryRunRules)
			require.Len(t, res.ruleViolations, 1)
			assert.Equal(t, "merge_queue", res.ruleViolations[0].Rule.Identifier)
		})

		t.Run(op.name+"/queued branch is blocked", func(t *testing.T) {
			e := crpMergeEnv(ctx, t, op)
			e.withMergeQueue(ctx, t, &crpMQEntryStore{queued: map[string]struct{}{crpHeadBranch: {}}})

			res := op.call(e.c, crpDefaultMergeIn())

			require.NoError(t, res.err)
			require.NotNil(t, res.blocked)
			require.Len(t, res.blocked.RuleViolations, 1)
			assert.NotEmpty(t, res.blocked.Message)
			e.git.AssertNotCalled(t, "Merge", mock.Anything, mock.Anything)
		})

		t.Run(op.name+"/protection rule blocks", func(t *testing.T) {
			e := crpMergeEnv(ctx, t, op)
			e.setAuthorizer(crpDenyPermAuthorizer{denied: map[enum.Permission]bool{enum.PermissionRepoEdit: true}})
			e.ruleStore.rules = []types.RuleInfoInternal{crpBranchRule(crpUpdateForbidden)}

			res := op.call(e.c, crpDefaultMergeIn())

			require.NoError(t, res.err)
			require.NotNil(t, res.blocked)
			assert.Contains(t, res.blocked.RuleViolations[0].Violations[0].Code, "lifecycle.update")
			e.git.AssertNotCalled(t, "Merge", mock.Anything, mock.Anything)
		})

		t.Run(op.name+"/head branch lookup failure", func(t *testing.T) {
			e := crpNewEnv(t)
			e.withMergeQueue(ctx, t, &crpMQEntryStore{})
			e.git.On("GetBranch", mock.Anything, mock.Anything).Return(nil, errCrpBoom)

			res := op.call(e.c, crpDefaultMergeIn())

			require.ErrorIs(t, res.err, errCrpBoom)
		})

		t.Run(op.name+"/stale head commit", func(t *testing.T) {
			e := crpMergeEnv(ctx, t, op)
			in := crpDefaultMergeIn()
			in.headSHA = crpTestSHAOther

			res := op.call(e.c, in)

			require.Error(t, res.err)
			assert.Equal(t, http.StatusBadRequest, crpUserErrStatus(t, res.err))
			assert.ErrorContains(t, res.err, "isn't the latest commit")
		})

		t.Run(op.name+"/base branch lookup failure", func(t *testing.T) {
			e := crpNewEnv(t)
			e.withMergeQueue(ctx, t, &crpMQEntryStore{})
			e.git.On("GetBranch", mock.Anything, mock.MatchedBy(func(p *git.GetBranchParams) bool {
				return p.BranchName == crpHeadBranch
			})).Return(&git.GetBranchOutput{Branch: crpBranch(crpHeadBranch, crpTestSHA)}, nil)
			e.git.On("GetBranch", mock.Anything, mock.MatchedBy(func(p *git.GetBranchParams) bool {
				return p.BranchName == crpDefaultBranch
			})).Return(nil, errCrpBoom)

			res := op.call(e.c, crpDefaultMergeIn())

			require.ErrorIs(t, res.err, errCrpBoom)
			assert.ErrorContains(t, res.err, "base branch")
		})

		t.Run(op.name+"/dry run verifies the merge", func(t *testing.T) {
			e := crpMergeEnv(ctx, t, op)
			e.git.On("MergeVerify", mock.Anything, mock.MatchedBy(func(p *git.MergeVerifyParams) bool {
				return p.HeadSHA == crpTestSHA
			})).Return(git.MergeVerifyOutput{ConflictFiles: []string{"c.txt"}}, nil)
			in := crpDefaultMergeIn()
			in.dryRun = true

			res := op.call(e.c, in)

			require.NoError(t, res.err)
			assert.True(t, res.dryRun)
			assert.Equal(t, []string{"c.txt"}, res.conflicts)
			e.git.AssertNotCalled(t, "Merge", mock.Anything, mock.Anything)
		})

		t.Run(op.name+"/dry run failure", func(t *testing.T) {
			e := crpMergeEnv(ctx, t, op)
			e.git.On("MergeVerify", mock.Anything, mock.Anything).Return(git.MergeVerifyOutput{}, errCrpBoom)
			in := crpDefaultMergeIn()
			in.dryRun = true

			res := op.call(e.c, in)

			require.ErrorIs(t, res.err, errCrpBoom)
		})

		t.Run(op.name+"/merge success with explicit base commit", func(t *testing.T) {
			e := crpMergeEnv(ctx, t, op)
			e.git.On("Merge", mock.Anything, mock.MatchedBy(func(p *git.MergeParams) bool {
				return p.HeadBranch == crpHeadBranch && p.HeadBranchExpectedSHA == crpTestSHA &&
					len(p.Refs) == 1 && p.Refs[0].Name == "refs/heads/head" && p.Refs[0].Old == crpTestSHA
			})).Return(merged, nil)
			in := crpDefaultMergeIn()
			in.baseBranch = ""
			in.baseSHA = crpTestSHAOther

			res := op.call(e.c, in)

			require.NoError(t, res.err)
			assert.Nil(t, res.blocked)
			assert.Equal(t, crpTestSHAOther, res.newSHA)
			e.git.AssertNotCalled(t, "GetBranch", mock.Anything, mock.MatchedBy(func(p *git.GetBranchParams) bool {
				return p.BranchName == crpDefaultBranch
			}))
		})

		t.Run(op.name+"/conflicts block the merge", func(t *testing.T) {
			e := crpMergeEnv(ctx, t, op)
			e.git.On("Merge", mock.Anything, mock.Anything).Return(
				git.MergeOutput{ConflictFiles: []string{"c.txt"}}, nil)

			res := op.call(e.c, crpDefaultMergeIn())

			require.NoError(t, res.err)
			require.NotNil(t, res.blocked)
			assert.Equal(t, []string{"c.txt"}, res.blocked.ConflictFiles)
			assert.Contains(t, res.blocked.Message, "c.txt")
		})

		t.Run(op.name+"/merge failure", func(t *testing.T) {
			e := crpMergeEnv(ctx, t, op)
			e.git.On("Merge", mock.Anything, mock.Anything).Return(git.MergeOutput{}, errCrpBoom)

			res := op.call(e.c, crpDefaultMergeIn())

			require.ErrorIs(t, res.err, errCrpBoom)
		})
	}
}

func TestCrpMergeCommitAndRebase_AncestorHandling(t *testing.T) {
	ctx := context.Background()

	for _, op := range crpMergeOps() {
		if !op.hasAncestor {
			continue
		}

		t.Run(op.name+"/already an ancestor", func(t *testing.T) {
			e := crpNewEnv(t)
			e.withMergeQueue(ctx, t, &crpMQEntryStore{})
			e.git.On("GetBranch", mock.Anything, mock.Anything).Return(
				&git.GetBranchOutput{Branch: crpBranch(crpHeadBranch, crpTestSHA)}, nil)
			e.git.On("IsAncestor", mock.Anything, mock.Anything).Return(git.IsAncestorOutput{Ancestor: true}, nil)

			res := op.call(e.c, crpDefaultMergeIn())

			require.NoError(t, res.err)
			assert.True(t, res.alreadyAnc)
			e.git.AssertNotCalled(t, "Merge", mock.Anything, mock.Anything)
		})

		t.Run(op.name+"/ancestor check failure", func(t *testing.T) {
			e := crpNewEnv(t)
			e.withMergeQueue(ctx, t, &crpMQEntryStore{})
			e.git.On("GetBranch", mock.Anything, mock.Anything).Return(
				&git.GetBranchOutput{Branch: crpBranch(crpHeadBranch, crpTestSHA)}, nil)
			e.git.On("IsAncestor", mock.Anything, mock.Anything).Return(git.IsAncestorOutput{}, errCrpBoom)

			res := op.call(e.c, crpDefaultMergeIn())

			require.ErrorIs(t, res.err, errCrpBoom)
		})
	}
}

func TestCrpMergeCommit_DefaultMessages(t *testing.T) {
	ctx := context.Background()
	op := crpMergeOps()[0]

	tests := []struct {
		name    string
		in      func() crpMergeIn
		wantMsg string
	}{
		{
			name:    "title and message are combined",
			in:      func() crpMergeIn { in := crpDefaultMergeIn(); in.title = "T"; in.message = "M"; return in },
			wantMsg: "T\n\nM",
		},
		{
			name:    "default for a base branch",
			in:      crpDefaultMergeIn,
			wantMsg: "Merge branch 'main' into head",
		},
		{
			name: "default for a base commit",
			in: func() crpMergeIn {
				in := crpDefaultMergeIn()
				in.baseBranch = ""
				in.baseSHA = crpTestSHAOther

				return in
			},
			wantMsg: "Merge commit '" + crpTestSHAOther.String() + "' into head",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := crpMergeEnv(ctx, t, op)
			var got string
			e.git.On("Merge", mock.Anything, mock.MatchedBy(func(p *git.MergeParams) bool {
				got = p.Message

				return true
			})).Return(git.MergeOutput{MergeSHA: crpTestSHAOther}, nil)

			res := op.call(e.c, tt.in())

			require.NoError(t, res.err)
			assert.Equal(t, tt.wantMsg, got)
		})
	}
}

func TestCrpSquash_Preparation(t *testing.T) {
	ctx := context.Background()
	op := crpMergeOps()[2]

	t.Run("merge base failure", func(t *testing.T) {
		e := crpMergeEnv(ctx, t, op)
		e.git.ExpectedCalls = nil
		e.git.On("GetBranch", mock.Anything, mock.Anything).Return(
			&git.GetBranchOutput{Branch: crpBranch(crpHeadBranch, crpTestSHA)}, nil)
		e.git.On("MergeBase", mock.Anything, mock.Anything).Return(git.MergeBaseOutput{}, errCrpBoom)

		res := op.call(e.c, crpDefaultMergeIn())

		require.ErrorIs(t, res.err, errCrpBoom)
	})

	t.Run("commit listing failure", func(t *testing.T) {
		e := crpMergeEnv(ctx, t, op)
		e.git.ExpectedCalls = nil
		e.git.On("GetBranch", mock.Anything, mock.Anything).Return(
			&git.GetBranchOutput{Branch: crpBranch(crpHeadBranch, crpTestSHA)}, nil)
		e.git.On("MergeBase", mock.Anything, mock.Anything).Return(git.MergeBaseOutput{MergeBaseSHA: crpTestSHA}, nil)
		e.git.On("ListCommits", mock.Anything, mock.Anything).Return(nil, errCrpBoom)

		res := op.call(e.c, crpDefaultMergeIn())

		require.ErrorIs(t, res.err, errCrpBoom)
	})

	t.Run("no commits between head and base", func(t *testing.T) {
		e := crpMergeEnv(ctx, t, op)
		e.git.ExpectedCalls = nil
		e.git.On("GetBranch", mock.Anything, mock.Anything).Return(
			&git.GetBranchOutput{Branch: crpBranch(crpHeadBranch, crpTestSHA)}, nil)
		e.git.On("MergeBase", mock.Anything, mock.Anything).Return(git.MergeBaseOutput{MergeBaseSHA: crpTestSHA}, nil)
		e.git.On("ListCommits", mock.Anything, mock.Anything).Return(&git.ListCommitsOutput{}, nil)

		res := op.call(e.c, crpDefaultMergeIn())

		require.Error(t, res.err)
		assert.Equal(t, http.StatusBadRequest, crpUserErrStatus(t, res.err))
	})

	t.Run("single commit supplies title and message", func(t *testing.T) {
		e := crpMergeEnv(ctx, t, op)
		var got *git.MergeParams
		e.git.On("Merge", mock.Anything, mock.MatchedBy(func(p *git.MergeParams) bool {
			got = p

			return true
		})).Return(git.MergeOutput{MergeSHA: crpTestSHAOther}, nil)

		res := op.call(e.c, crpDefaultMergeIn())

		require.NoError(t, res.err)
		require.NotNil(t, got)
		assert.Equal(t, "only\n\nbody text\n", got.Message)
		assert.Equal(t, crpTestSHAOther, got.BaseSHA)
	})

	t.Run("several commits are summarized", func(t *testing.T) {
		e := crpMergeEnv(ctx, t, op)
		e.git.ExpectedCalls = nil
		e.git.On("GetBranch", mock.Anything, mock.Anything).Return(
			&git.GetBranchOutput{Branch: crpBranch(crpHeadBranch, crpTestSHA)}, nil)
		e.git.On("MergeBase", mock.Anything, mock.Anything).Return(git.MergeBaseOutput{MergeBaseSHA: crpTestSHA}, nil)
		e.git.On("ListCommits", mock.Anything, mock.Anything).Return(&git.ListCommitsOutput{
			Commits:      []git.Commit{{Title: "first", Message: "first"}, {Title: "second", Message: "second"}},
			TotalCommits: 5,
		}, nil)
		var got *git.MergeParams
		e.git.On("Merge", mock.Anything, mock.MatchedBy(func(p *git.MergeParams) bool {
			got = p

			return true
		})).Return(git.MergeOutput{MergeSHA: crpTestSHAOther}, nil)

		res := op.call(e.c, crpDefaultMergeIn())

		require.NoError(t, res.err)
		require.NotNil(t, got)
		assert.Equal(t, "Squashed 5 commits\n\n* first\n* second\n* and 3 more commits\n", got.Message)
	})
}

func TestCrpDeleteBranch_WithMergeQueue(t *testing.T) {
	ctx := context.Background()

	t.Run("deletes the branch", func(t *testing.T) {
		e := crpNewEnv(t)
		e.withMergeQueue(ctx, t, &crpMQEntryStore{})
		e.git.On("DeleteBranch", mock.Anything, mock.MatchedBy(func(p *git.DeleteBranchParams) bool {
			return p.BranchName == crpFeatureBranch
		})).Return(nil)

		out, violations, err := e.c.DeleteBranch(ctx, crpSession(), crpRepoRef, crpFeatureBranch, false, false)

		require.NoError(t, err)
		assert.Empty(t, violations)
		assert.Empty(t, out.RuleViolations)
		e.git.AssertNumberOfCalls(t, "DeleteBranch", 1)
	})

	t.Run("queued branch is protected", func(t *testing.T) {
		e := crpNewEnv(t)
		e.withMergeQueue(ctx, t, &crpMQEntryStore{queued: map[string]struct{}{crpFeatureBranch: {}}})

		_, violations, err := e.c.DeleteBranch(ctx, crpSession(), crpRepoRef, crpFeatureBranch, false, false)

		require.NoError(t, err)
		require.Len(t, violations, 1)
		e.git.AssertNotCalled(t, "DeleteBranch", mock.Anything, mock.Anything)
	})

	t.Run("merge queue lookup failure", func(t *testing.T) {
		e := crpNewEnv(t)
		e.withMergeQueue(ctx, t, &crpMQEntryStore{err: errCrpBoom})

		_, _, err := e.c.DeleteBranch(ctx, crpSession(), crpRepoRef, crpFeatureBranch, false, false)

		require.ErrorIs(t, err, errCrpBoom)
	})

	t.Run("dry run reports violations without touching git", func(t *testing.T) {
		e := crpNewEnv(t)
		e.withMergeQueue(ctx, t, &crpMQEntryStore{})
		e.ruleStore.rules = []types.RuleInfoInternal{crpBranchRule(crpDeleteForbidden)}

		out, _, err := e.c.DeleteBranch(ctx, crpSession(), crpRepoRef, crpFeatureBranch, false, true)

		require.NoError(t, err)
		assert.True(t, out.DryRunRules)
		require.Len(t, out.RuleViolations, 1)
		e.git.AssertNotCalled(t, "DeleteBranch", mock.Anything, mock.Anything)
	})

	t.Run("critical violation blocks deletion", func(t *testing.T) {
		e := crpNewEnv(t)
		e.withMergeQueue(ctx, t, &crpMQEntryStore{})
		e.setAuthorizer(crpDenyPermAuthorizer{denied: map[enum.Permission]bool{enum.PermissionRepoEdit: true}})
		e.ruleStore.rules = []types.RuleInfoInternal{crpBranchRule(crpDeleteForbidden)}

		_, violations, err := e.c.DeleteBranch(ctx, crpSession(), crpRepoRef, crpFeatureBranch, false, false)

		require.NoError(t, err)
		require.Len(t, violations, 1)
		e.git.AssertNotCalled(t, "DeleteBranch", mock.Anything, mock.Anything)
	})

	t.Run("repo owner can bypass the violation", func(t *testing.T) {
		e := crpNewEnv(t)
		e.withMergeQueue(ctx, t, &crpMQEntryStore{})
		e.ruleStore.rules = []types.RuleInfoInternal{crpBranchRule(crpDeleteOwnerAllowed)}
		e.git.On("DeleteBranch", mock.Anything, mock.Anything).Return(nil)

		out, violations, err := e.c.DeleteBranch(ctx, crpSession(), crpRepoRef, crpFeatureBranch, true, false)

		require.NoError(t, err)
		assert.Empty(t, violations)
		require.Len(t, out.RuleViolations, 1)
		assert.True(t, out.RuleViolations[0].Bypassed)
	})

	t.Run("git failure", func(t *testing.T) {
		e := crpNewEnv(t)
		e.withMergeQueue(ctx, t, &crpMQEntryStore{})
		e.git.On("DeleteBranch", mock.Anything, mock.Anything).Return(errCrpBoom)

		_, _, err := e.c.DeleteBranch(ctx, crpSession(), crpRepoRef, crpFeatureBranch, false, false)

		require.ErrorIs(t, err, errCrpBoom)
	})
}
