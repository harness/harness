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

package githook

import (
	"context"
	"testing"

	"github.com/harness/gitness/audit"
	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/git/hook"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/require"
)

func (e *cxrEnv) postInput(op enum.GitOpType, updates ...hook.ReferenceUpdate) types.GithookPostReceiveInput {
	return types.GithookPostReceiveInput{
		GithookInputBase: types.GithookInputBase{RepoID: e.repo.ID, PrincipalID: e.admin.ID, OperationType: op},
		PostReceiveInput: hook.PostReceiveInput{RefUpdates: updates},
	}
}

func (e *cxrEnv) postReceive(ctx context.Context, t *testing.T, updates ...hook.ReferenceUpdate) hook.Output {
	t.Helper()
	out, err := e.ctrl.PostReceive(ctx, e.git, e.session(), e.postInput(enum.GitOpTypeGitPush, updates...))
	require.NoError(t, err)
	return out
}

func cxrOpenPR(number int64, title string, sourceRepoID, targetRepoID int64) *types.PullReq {
	return &types.PullReq{
		Number: number, Title: title, SourceRepoID: &sourceRepoID, TargetRepoID: targetRepoID,
		TargetBranch: "main", SourceBranch: "feature",
	}
}

func TestCxrPostReceiveRepoLookup(t *testing.T) {
	ctx := t.Context()
	e := cxrNewEnv(ctx, t)

	in := e.postInput(enum.GitOpTypeGitPush)
	in.RepoID = 0
	_, err := e.ctrl.PostReceive(ctx, e.git, e.session(), in)
	require.ErrorContains(t, err, "valid repository reference")

	in.RepoID = 9999
	_, err = e.ctrl.PostReceive(ctx, e.git, e.session(), in)
	require.ErrorContains(t, err, "failed to find repo")
}

func TestCxrPostReceiveReferenceEvents(t *testing.T) {
	ctx := t.Context()

	t.Run("branch and tag updates", func(t *testing.T) {
		e := cxrNewEnv(ctx, t)
		out := e.postReceive(ctx, t,
			cxrCreate(cxrRefFeature), cxrDelete("refs/heads/old"), cxrUpdate("refs/heads/other"),
			cxrCreate(cxrRefTag), cxrDelete("refs/tags/v0"), cxrUpdate("refs/tags/v2"),
			cxrCreate("refs/notes/commits"))
		require.Nil(t, out.Error)
		require.Equal(t, 1, e.post.calls)
	})

	t.Run("extender runs and its failure is ignored", func(t *testing.T) {
		e := cxrNewEnv(ctx, t)
		e.post.messages = []string{"from extender"}
		e.post.err = errCxrBoom
		out := e.postReceive(ctx, t, cxrUpdate(cxrRefMain))
		require.Contains(t, out.Messages, "from extender")
	})

	t.Run("non-active repositories do not report ref events", func(t *testing.T) {
		e := cxrNewEnv(ctx, t)
		e.setRepoColumn(ctx, t, "repo_state", int(enum.RepoStateMigrateGitPush))
		e.git.ancestor = false
		e.postReceive(ctx, t, cxrUpdate(cxrRefMain))
		require.Empty(t, e.audit.actions)
	})

	t.Run("force push on the default branch is audited", func(t *testing.T) {
		e := cxrNewEnv(ctx, t)
		e.git.ancestor = false
		e.postReceive(ctx, t, cxrUpdate(cxrRefMain))
		require.Equal(t, []audit.Action{audit.ActionForcePush}, e.audit.actions)
	})

	t.Run("failed ancestor check counts as force push", func(t *testing.T) {
		e := cxrNewEnv(ctx, t)
		e.git.ancestorErr = errCxrBoom
		e.postReceive(ctx, t, cxrUpdate(cxrRefMain))
		require.Equal(t, []audit.Action{audit.ActionForcePush}, e.audit.actions)
	})

	t.Run("regular update on the default branch is not audited", func(t *testing.T) {
		e := cxrNewEnv(ctx, t)
		e.postReceive(ctx, t, cxrUpdate(cxrRefMain))
		require.Empty(t, e.audit.actions)
	})

	t.Run("force push on other branches is not audited", func(t *testing.T) {
		e := cxrNewEnv(ctx, t)
		e.git.ancestor = false
		e.postReceive(ctx, t, cxrUpdate(cxrRefFeature))
		require.Empty(t, e.audit.actions)
	})

	t.Run("audit failures and unknown principals are tolerated", func(t *testing.T) {
		e := cxrNewEnv(ctx, t)
		e.git.ancestor = false
		e.audit.err = errCxrBoom
		e.postReceive(ctx, t, cxrUpdate(cxrRefMain))
		require.Len(t, e.audit.actions, 1)

		in := e.postInput(enum.GitOpTypeGitPush, cxrUpdate(cxrRefMain))
		in.PrincipalID = 9999
		_, err := e.ctrl.PostReceive(ctx, e.git, e.session(), in)
		require.NoError(t, err)
		require.Len(t, e.audit.actions, 1)
	})
}

func TestCxrPostReceiveEmptyRepo(t *testing.T) {
	ctx := t.Context()

	t.Run("push without branches reports an error", func(t *testing.T) {
		e := cxrNewEnv(ctx, t)
		e.setRepoColumn(ctx, t, "repo_is_empty", true)
		out := e.postReceive(ctx, t, cxrCreate(cxrRefTag))
		require.NotNil(t, out.Error)
		require.Contains(t, *out.Error, "branch")
	})

	t.Run("default branch is created", func(t *testing.T) {
		e := cxrNewEnv(ctx, t)
		cxrBootstrap(ctx, t, e.cxrBase)
		e.setRepoColumn(ctx, t, "repo_is_empty", true)
		out := e.postReceive(ctx, t, cxrCreate(cxrRefFeature), cxrCreate(cxrRefMain))
		require.Nil(t, out.Error)

		repo, err := e.repos.Find(ctx, e.repo.ID)
		require.NoError(t, err)
		require.False(t, repo.IsEmpty)
		require.Equal(t, "main", repo.DefaultBranch)
	})

	t.Run("first pushed branch becomes the default branch", func(t *testing.T) {
		e := cxrNewEnv(ctx, t)
		cxrBootstrap(ctx, t, e.cxrBase)
		e.setRepoColumn(ctx, t, "repo_is_empty", true)
		out := e.postReceive(ctx, t, cxrCreate(cxrRefFeature))
		require.Nil(t, out.Error)

		repo, err := e.repos.Find(ctx, e.repo.ID)
		require.NoError(t, err)
		require.False(t, repo.IsEmpty)
		require.Equal(t, "feature", repo.DefaultBranch)
	})
}

func TestCxrPostReceivePullRequestMessages(t *testing.T) {
	ctx := t.Context()

	t.Run("new branch suggests a pull request", func(t *testing.T) {
		e := cxrNewEnv(ctx, t)
		out := e.postReceive(ctx, t, cxrCreate(cxrRefFeature))
		require.Len(t, out.Messages, 2)
		require.Contains(t, out.Messages[0], "Create a pull request")
		require.Contains(t, out.Messages[1], "https://ui.example.com/root/repo/compare/main...feature")
	})

	t.Run("default branch push suggests nothing", func(t *testing.T) {
		e := cxrNewEnv(ctx, t)
		require.Empty(t, e.postReceive(ctx, t, cxrUpdate(cxrRefMain)).Messages)
	})

	t.Run("batch pushes, tags and deletions are skipped", func(t *testing.T) {
		e := cxrNewEnv(ctx, t)
		require.Empty(t, e.postReceive(ctx, t, cxrCreate(cxrRefFeature), cxrCreate("refs/heads/b")).Messages)
		require.Empty(t, e.postReceive(ctx, t, cxrCreate(cxrRefTag)).Messages)
		require.Empty(t, e.postReceive(ctx, t, cxrDelete(cxrRefFeature)).Messages)
	})

	t.Run("existing open pull requests are listed", func(t *testing.T) {
		e := cxrNewEnv(ctx, t)
		e.prs.prs = []*types.PullReq{cxrOpenPR(1, "first", e.repo.ID, e.repo.ID)}
		out := e.postReceive(ctx, t, cxrUpdate(cxrRefFeature))
		require.Equal(t, []string{
			"Branch \"feature\" has an open PR:",
			"  (#1) first",
			"    https://ui.example.com/root/repo/pulls/1",
		}, out.Messages)

		e.prs.prs = []*types.PullReq{
			cxrOpenPR(2, "second", e.repo.ID, e.repo.ID), cxrOpenPR(1, "first", e.repo.ID, e.repo.ID),
		}
		out = e.postReceive(ctx, t, cxrUpdate(cxrRefFeature))
		require.Equal(t, "Branch \"feature\" has open PRs:", out.Messages[0])
		require.Len(t, out.Messages, 5)
	})

	t.Run("pull requests across forks link to the target repository", func(t *testing.T) {
		e := cxrNewEnv(ctx, t)
		target := e.createRepo(ctx, t, e.createSpace(ctx, t, "other"), "upstream")
		e.prs.prs = []*types.PullReq{cxrOpenPR(7, "fork pr", e.repo.ID, target.ID)}
		out := e.postReceive(ctx, t, cxrUpdate(cxrRefFeature))
		require.Contains(t, out.Messages[2], "https://ui.example.com/other/upstream/pulls/7")

		// An unknown target repository fails message generation and no message is printed.
		e.prs.prs = []*types.PullReq{cxrOpenPR(7, "fork pr", e.repo.ID, 9999)}
		out = e.postReceive(ctx, t, cxrUpdate(cxrRefFeature))
		require.Empty(t, out.Messages)
	})

	t.Run("pull requests with unrelated histories announce closure", func(t *testing.T) {
		e := cxrNewEnv(ctx, t)
		e.git.mergeBaseErr = errors.InvalidArgument("unrelated histories")
		e.prs.prs = []*types.PullReq{cxrOpenPR(1, "first", e.repo.ID, e.repo.ID)}
		out := e.postReceive(ctx, t, cxrUpdate(cxrRefFeature))
		require.Contains(t, out.Messages[0], "has an open PR that would be closed")

		e.prs.prs = []*types.PullReq{
			cxrOpenPR(1, "first", e.repo.ID, e.repo.ID), cxrOpenPR(2, "second", e.repo.ID, e.repo.ID),
		}
		out = e.postReceive(ctx, t, cxrUpdate(cxrRefFeature))
		require.Contains(t, out.Messages[0], "has open PRs that would be closed")
	})

	t.Run("merge base failures are tolerated", func(t *testing.T) {
		e := cxrNewEnv(ctx, t)
		e.git.mergeBaseErr = errCxrBoom
		e.prs.prs = []*types.PullReq{cxrOpenPR(1, "first", e.repo.ID, e.repo.ID)}
		out := e.postReceive(ctx, t, cxrUpdate(cxrRefFeature))
		require.Contains(t, out.Messages[0], "has an open PR")
	})

	t.Run("pull request listing failure is tolerated", func(t *testing.T) {
		e := cxrNewEnv(ctx, t)
		e.prs.err = errCxrBoom
		require.Empty(t, e.postReceive(ctx, t, cxrUpdate(cxrRefFeature)).Messages)
	})
}

func TestCxrPostReceiveMergeQueueEvents(t *testing.T) {
	ctx := t.Context()
	mqRule := cxrRule("branch", `{"pullreq":{"merge_queue":{"status_checks":{"require_identifiers":["ci"]},`+
		`"group_size":1,"checks_concurrency":1,"max_check_duration_seconds":60}}}`)

	t.Run("queue protected branches trigger events", func(t *testing.T) {
		e := cxrNewEnv(ctx, t)
		e.rules.rules = []types.RuleInfoInternal{mqRule}
		out := e.postReceive(ctx, t, cxrUpdate(cxrRefMain), cxrDelete(cxrRefFeature), cxrCreate("refs/heads/new"))
		require.Nil(t, out.Error)
	})

	t.Run("branches without queue are ignored", func(t *testing.T) {
		e := cxrNewEnv(ctx, t)
		require.Nil(t, e.postReceive(ctx, t, cxrUpdate(cxrRefMain), cxrDelete(cxrRefFeature)).Error)
	})

	t.Run("merge queue operations skip the trigger", func(t *testing.T) {
		e := cxrNewEnv(ctx, t)
		e.rules.onlyErr = errCxrBoom
		_, err := e.ctrl.PostReceive(ctx, e.git, e.session(),
			e.postInput(enum.GitOpTypeMergeQueue, cxrUpdate(cxrRefMain)))
		require.NoError(t, err)
	})

	t.Run("rule listing failure is tolerated", func(t *testing.T) {
		e := cxrNewEnv(ctx, t)
		e.rules.onlyErr = errCxrBoom
		require.Nil(t, e.postReceive(ctx, t, cxrUpdate(cxrRefMain)).Error)
	})

	t.Run("invalid rule definition is tolerated", func(t *testing.T) {
		e := cxrNewEnv(ctx, t)
		e.rules.rules = []types.RuleInfoInternal{cxrRule("branch", "not json")}
		require.Nil(t, e.postReceive(ctx, t, cxrUpdate(cxrRefMain), cxrDelete(cxrRefFeature)).Error)
	})
}
