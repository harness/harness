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

package pullreq

import (
	"context"
	"testing"

	gitevents "github.com/harness/gitness/app/events/git"
	"github.com/harness/gitness/app/store"
	gitnesserrors "github.com/harness/gitness/errors"
	"github.com/harness/gitness/events"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/git/sha"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/require"
)

type sv5BranchGit struct {
	*sv5Git

	commitErr    error
	fetchErr     error
	fetched      []string
	updateRefErr error
	headRefs     []git.UpdateRefParams
	getRefErr    error
	mergeBaseErr error
	mergeBaseSHA string
	diffErr      error
}

func (g *sv5BranchGit) GetCommit(context.Context, *git.GetCommitParams) (*git.GetCommitOutput, error) {
	if g.commitErr != nil {
		return nil, g.commitErr
	}
	return &git.GetCommitOutput{Commit: git.Commit{Title: "commit title"}}, nil
}

func (g *sv5BranchGit) FetchObjects(_ context.Context, p *git.FetchObjectsParams) (git.FetchObjectsOutput, error) {
	g.fetched = append(g.fetched, p.Source)
	return git.FetchObjectsOutput{}, g.fetchErr
}

func (g *sv5BranchGit) UpdateRef(_ context.Context, p git.UpdateRefParams) error {
	g.headRefs = append(g.headRefs, p)
	return g.updateRefErr
}

func (g *sv5BranchGit) GetRef(context.Context, git.GetRefParams) (git.GetRefResponse, error) {
	return git.GetRefResponse{SHA: sha.Must(sv5SHAc)}, g.getRefErr
}

func (g *sv5BranchGit) MergeBase(context.Context, git.MergeBaseParams) (git.MergeBaseOutput, error) {
	if g.mergeBaseErr != nil {
		return git.MergeBaseOutput{}, g.mergeBaseErr
	}
	if g.mergeBaseSHA != "" {
		return git.MergeBaseOutput{MergeBaseSHA: sha.Must(g.mergeBaseSHA)}, nil
	}
	return git.MergeBaseOutput{MergeBaseSHA: sha.Must(sv5SHAb)}, nil
}

func (g *sv5BranchGit) DiffStats(context.Context, *git.DiffParams) (git.DiffStatsOutput, error) {
	return git.DiffStatsOutput{Commits: 4, FilesChanged: 5, Additions: 6, Deletions: 7}, g.diffErr
}

type sv5ListPullReqStore struct {
	*sv5PullReqStore

	list      []*types.PullReq
	resetErr  error
	resetRepo int64
	resetRef  string
	seqErr    error
}

func (s *sv5ListPullReqStore) List(context.Context, *types.PullReqFilter) ([]*types.PullReq, error) {
	return s.list, nil
}

func (s *sv5ListPullReqStore) UpdateActivitySeq(_ context.Context, pr *types.PullReq) (*types.PullReq, error) {
	if s.seqErr != nil {
		return nil, s.seqErr
	}
	cp := *pr
	cp.ActivitySeq++
	return &cp, nil
}

func (s *sv5ListPullReqStore) ResetMergeCheckStatus(_ context.Context, repoID int64, ref string) error {
	s.resetRepo, s.resetRef = repoID, ref
	return s.resetErr
}

type sv5AutoMergeStore struct {
	store.AutoMergeStore

	deleted []int64
}

func (s *sv5AutoMergeStore) Delete(_ context.Context, id int64) (bool, error) {
	s.deleted = append(s.deleted, id)
	return true, nil
}

type sv5BranchEnv struct {
	*sv5Env

	bgit  *sv5BranchGit
	lists *sv5ListPullReqStore
	auto  *sv5AutoMergeStore
}

func sv5NewBranchEnv(t *testing.T) *sv5BranchEnv {
	t.Helper()
	env := sv5NewEnv(t)
	bgit := &sv5BranchGit{sv5Git: env.git}
	lists := &sv5ListPullReqStore{sv5PullReqStore: env.prs}
	auto := &sv5AutoMergeStore{}

	env.svc.git = bgit
	env.svc.pullreqStore = lists
	env.svc.autoMergeStore = auto

	pr := *env.prs.pr
	lists.list = []*types.PullReq{&pr}

	return &sv5BranchEnv{sv5Env: env, bgit: bgit, lists: lists, auto: auto}
}

func sv5BranchEvent() *events.Event[*gitevents.BranchUpdatedPayload] {
	return &events.Event[*gitevents.BranchUpdatedPayload]{Payload: &gitevents.BranchUpdatedPayload{
		RepoID: 10, PrincipalID: 9, Ref: "refs/heads/feature", OldSHA: sv5SHAa, NewSHA: sv5SHAc, Forced: true,
	}}
}

func TestSv5UpdatePullReqOnBranchUpdate(t *testing.T) {
	ctx := context.Background()

	t.Run("updates the pull request", func(t *testing.T) {
		env := sv5NewBranchEnv(t)
		require.NoError(t, env.svc.updatePullReqOnBranchUpdate(ctx, sv5BranchEvent()))

		require.Equal(t, int64(10), env.lists.resetRepo)
		require.Equal(t, "feature", env.lists.resetRef)

		pr := env.prs.closed
		require.NotNil(t, pr)
		require.Equal(t, sv5SHAc, pr.SourceSHA)
		require.Equal(t, sv5SHAb, pr.MergeBaseSHA)
		require.Equal(t, sv5SHAc, *pr.MergeTargetSHA)
		require.Equal(t, enum.MergeCheckStatusUnchecked, pr.MergeCheckStatus)

		require.Len(t, env.bgit.headRefs, 1)
		require.Equal(t, sha.Must(sv5SHAc), env.bgit.headRefs[0].NewValue)
		require.Equal(t, sha.Must(sv5SHAa), env.bgit.headRefs[0].OldValue)
		require.Empty(t, env.bgit.fetched, "same repository pull requests need no fetch")

		require.Len(t, env.activity.payloads, 1)
		require.Equal(t, &types.PullRequestActivityPayloadBranchUpdate{
			Old: sv5SHAa, New: sv5SHAc, Forced: true, CommitTitle: "commit title",
		}, env.activity.payloads[0])
		require.Equal(t, 1, env.producer.count())
		require.Equal(t, []enum.SSEType{enum.SSETypePullReqUpdated}, env.sse.published)
	})

	t.Run("auto merge is disabled by a new commit", func(t *testing.T) {
		env := sv5NewBranchEnv(t)
		env.lists.list[0].SubState = enum.PullReqSubStateAutoMerge

		require.NoError(t, env.svc.updatePullReqOnBranchUpdate(ctx, sv5BranchEvent()))
		require.Equal(t, enum.PullReqSubStateNone, env.prs.closed.SubState)
		require.Equal(t, int64(6), env.prs.closed.ActivitySeq)
		require.Equal(t, []int64{7}, env.auto.deleted)
		require.Len(t, env.activity.payloads, 2)
		require.IsType(t, &types.PullRequestActivityPayloadAutoMergeDisabledBranchUpdate{}, env.activity.payloads[1])
	})

	t.Run("cross repository pull request fetches the objects", func(t *testing.T) {
		env := sv5NewBranchEnv(t)
		fork := int64(11)
		env.lists.list[0].SourceRepoID = &fork

		require.NoError(t, env.svc.updatePullReqOnBranchUpdate(ctx, sv5BranchEvent()))
		require.Equal(t, []string{"git-uid"}, env.bgit.fetched)
		require.NotNil(t, env.prs.closed)
	})

	t.Run("activity failures are tolerated", func(t *testing.T) {
		env := sv5NewBranchEnv(t)
		env.lists.list[0].SubState = enum.PullReqSubStateAutoMerge
		env.activity.err = errSv5Boom
		env.bgit.commitErr = errSv5Boom // title is optional

		require.NoError(t, env.svc.updatePullReqOnBranchUpdate(ctx, sv5BranchEvent()))
		require.Len(t, env.activity.payloads, 2)
		require.NotNil(t, env.prs.closed)
	})

	t.Run("merge status reset failure aborts the handler", func(t *testing.T) {
		env := sv5NewBranchEnv(t)
		env.lists.resetErr = errSv5Boom
		require.ErrorIs(t, env.svc.updatePullReqOnBranchUpdate(ctx, sv5BranchEvent()), errSv5Boom)
		require.Nil(t, env.prs.closed)
	})

	t.Run("unrelated histories close the pull request", func(t *testing.T) {
		env := sv5NewBranchEnv(t)
		env.bgit.mergeBaseErr = gitnesserrors.InvalidArgument("unrelated")

		require.NoError(t, env.svc.updatePullReqOnBranchUpdate(ctx, sv5BranchEvent()))
		require.NotNil(t, env.prs.closed)
		require.Equal(t, enum.PullReqStateClosed, env.prs.closed.State)
	})

	t.Run("pull request closed concurrently", func(t *testing.T) {
		env := sv5NewBranchEnv(t)
		env.lists.list[0].State = enum.PullReqStateMerged
		require.NoError(t, env.svc.updatePullReqOnBranchUpdate(ctx, sv5BranchEvent()))
		require.Nil(t, env.prs.closed)
		require.Empty(t, env.activity.payloads)
	})

	t.Run("ref that is not a branch", func(t *testing.T) {
		env := sv5NewBranchEnv(t)
		ev := sv5BranchEvent()
		ev.Payload.Ref = "refs/tags/v1"
		require.NoError(t, env.svc.updatePullReqOnBranchUpdate(ctx, ev))
		require.Empty(t, env.lists.resetRef)
		require.Nil(t, env.prs.closed)
	})

	failures := map[string]func(env *sv5BranchEnv, ev *events.Event[*gitevents.BranchUpdatedPayload]){
		"invalid old sha": func(_ *sv5BranchEnv, ev *events.Event[*gitevents.BranchUpdatedPayload]) {
			ev.Payload.OldSHA = "zz"
		},
		"invalid new sha": func(_ *sv5BranchEnv, ev *events.Event[*gitevents.BranchUpdatedPayload]) {
			ev.Payload.NewSHA = "zz"
		},
		"no source repo": func(env *sv5BranchEnv, _ *events.Event[*gitevents.BranchUpdatedPayload]) {
			env.lists.list[0].SourceRepoID = nil
		},
		"target repo lookup": func(env *sv5BranchEnv, _ *events.Event[*gitevents.BranchUpdatedPayload]) {
			env.lists.list[0].TargetRepoID = 404
		},
		"source repo missing": func(env *sv5BranchEnv, _ *events.Event[*gitevents.BranchUpdatedPayload]) {
			id := int64(405)
			env.lists.list[0].SourceRepoID = &id
		},
		"source repo lookup": func(env *sv5BranchEnv, _ *events.Event[*gitevents.BranchUpdatedPayload]) {
			id := int64(404)
			env.lists.list[0].SourceRepoID = &id
		},
		"fetch": func(env *sv5BranchEnv, _ *events.Event[*gitevents.BranchUpdatedPayload]) {
			id := int64(11)
			env.lists.list[0].SourceRepoID = &id
			env.bgit.fetchErr = errSv5Boom
		},
		"head ref update": func(env *sv5BranchEnv, _ *events.Event[*gitevents.BranchUpdatedPayload]) {
			env.bgit.updateRefErr = errSv5Boom
		},
		"target branch": func(env *sv5BranchEnv, _ *events.Event[*gitevents.BranchUpdatedPayload]) {
			env.bgit.getRefErr = errSv5Boom
		},
		"merge base": func(env *sv5BranchEnv, _ *events.Event[*gitevents.BranchUpdatedPayload]) {
			env.bgit.mergeBaseErr = errSv5Boom
		},
		"stale source sha": func(env *sv5BranchEnv, _ *events.Event[*gitevents.BranchUpdatedPayload]) {
			env.lists.list[0].SourceSHA = sv5SHAb
		},
	}
	for name, mutate := range failures {
		t.Run("failure "+name, func(t *testing.T) {
			env := sv5NewBranchEnv(t)
			ev := sv5BranchEvent()
			mutate(env, ev)

			// Per pull request failures are logged and don't fail the handler.
			require.NoError(t, env.svc.updatePullReqOnBranchUpdate(ctx, ev))
			require.Nil(t, env.prs.closed)
			require.Empty(t, env.activity.payloads)
		})
	}
}

func sv5DeletedTargetPR(env *sv5BranchEnv) *types.PullReq {
	return env.lists.list[0]
}

func TestSv5UpdatePRToDefaultBranch(t *testing.T) {
	ctx := context.Background()
	repo := &types.Repository{ID: 10, ParentID: 3, GitUID: "git-uid", DefaultBranch: "main"}

	t.Run("retargets the pull request", func(t *testing.T) {
		env := sv5NewBranchEnv(t)
		pr := sv5DeletedTargetPR(env)
		pr.TargetBranch = "gone"

		require.NoError(t, env.svc.updatePRToDefaultBranch(ctx, pr, repo, "gone", 9))

		got := env.prs.closed
		require.Equal(t, "main", got.TargetBranch)
		require.Equal(t, sv5SHAb, got.MergeBaseSHA)
		require.Equal(t, sv5SHAc, *got.MergeTargetSHA)
		require.EqualValues(t, 4, *got.Stats.DiffStats.Commits)
		require.Len(t, env.activity.payloads, 1)
		movedPayload, ok := env.activity.payloads[0].(*types.PullRequestActivityPayloadTargetBranchDeleted)
		require.True(t, ok)
		require.Equal(t, "gone", movedPayload.OldTargetBranch)
		require.Len(t, env.bgit.headRefs, 1, "stale merge reference is deleted")
		require.Equal(t, "refs/pullreq/3/merge", env.bgit.headRefs[0].Name)
		require.Equal(t, 1, env.producer.count())
		require.Equal(t, []enum.SSEType{enum.SSETypePullReqUpdated}, env.sse.published)
	})

	t.Run("activity and reference failures are tolerated", func(t *testing.T) {
		env := sv5NewBranchEnv(t)
		env.activity.err = errSv5Boom
		env.bgit.updateRefErr = errSv5Boom
		require.NoError(t, env.svc.updatePRToDefaultBranch(ctx, sv5DeletedTargetPR(env), repo, "gone", 9))
		require.NotNil(t, env.prs.closed)
	})

	t.Run("pull request without changes is closed", func(t *testing.T) {
		env := sv5NewBranchEnv(t)
		pr := sv5DeletedTargetPR(env)
		env.bgit.mergeBaseSHA = pr.SourceSHA

		require.NoError(t, env.svc.updatePRToDefaultBranch(ctx, pr, repo, "gone", 9))

		got := env.prs.closed
		require.Equal(t, enum.PullReqStateClosed, got.State)
		require.Equal(t, "main", got.TargetBranch)
		require.Equal(t, 1, env.repos.updated.NumClosedPulls)
		require.Len(t, env.activity.payloads, 2)
		require.IsType(t, &types.PullRequestActivityPayloadTargetBranchDeleted{}, env.activity.payloads[0])
		require.IsType(t, &types.PullRequestActivityPayloadStateChange{}, env.activity.payloads[1])
		require.Len(t, env.bgit.headRefs, 1)
		require.Equal(t, 1, env.producer.count())
	})

	t.Run("closing tolerates activity and reference failures", func(t *testing.T) {
		env := sv5NewBranchEnv(t)
		pr := sv5DeletedTargetPR(env)
		env.bgit.mergeBaseSHA = pr.SourceSHA
		env.activity.err = errSv5Boom
		env.bgit.updateRefErr = errSv5Boom
		require.NoError(t, env.svc.updatePRToDefaultBranch(ctx, pr, repo, "gone", 9))
		require.NotNil(t, env.prs.closed)
	})

	t.Run("unrelated histories close the pull request", func(t *testing.T) {
		env := sv5NewBranchEnv(t)
		env.bgit.mergeBaseErr = gitnesserrors.InvalidArgument("unrelated")

		require.NoError(t, env.svc.updatePRToDefaultBranch(ctx, sv5DeletedTargetPR(env), repo, "gone", 9))
		require.Equal(t, enum.PullReqStateClosed, env.prs.closed.State)
		// Target branch deleted activity, non-unique merge base, state change.
		require.Len(t, env.activity.payloads, 3)
	})

	t.Run("unrelated histories with an invalid source sha", func(t *testing.T) {
		env := sv5NewBranchEnv(t)
		env.bgit.mergeBaseErr = gitnesserrors.InvalidArgument("unrelated")
		pr := sv5DeletedTargetPR(env)
		pr.SourceSHA = "zz"
		require.ErrorContains(t, env.svc.updatePRToDefaultBranch(ctx, pr, repo, "gone", 9), "parse PR source SHA")
	})

	t.Run("activity sequence reservation failure is tolerated", func(t *testing.T) {
		env := sv5NewBranchEnv(t)
		env.bgit.mergeBaseErr = gitnesserrors.InvalidArgument("unrelated")
		env.lists.seqErr = errSv5Boom
		require.NoError(t, env.svc.updatePRToDefaultBranch(ctx, sv5DeletedTargetPR(env), repo, "gone", 9))
		require.Len(t, env.activity.payloads, 2, "only the close activities are written")
	})

	t.Run("pull request closed concurrently", func(t *testing.T) {
		env := sv5NewBranchEnv(t)
		pr := sv5DeletedTargetPR(env)
		pr.State = enum.PullReqStateClosed
		require.NoError(t, env.svc.updatePRToDefaultBranch(ctx, pr, repo, "gone", 9))
		require.Nil(t, env.prs.closed)

		pr.SourceSHA = sv5SHAb
		env.bgit.mergeBaseSHA = sv5SHAb
		require.NoError(t, env.svc.updatePRToDefaultBranch(ctx, pr, repo, "gone", 9))
		require.Nil(t, env.prs.closed)
	})

	failures := map[string]func(env *sv5BranchEnv){
		"default branch lookup": func(env *sv5BranchEnv) { env.bgit.getRefErr = errSv5Boom },
		"merge base":            func(env *sv5BranchEnv) { env.bgit.mergeBaseErr = errSv5Boom },
		"diff stats":            func(env *sv5BranchEnv) { env.bgit.diffErr = errSv5Boom },
		"database": func(env *sv5BranchEnv) {
			env.prs.updateErr = errSv5Boom
		},
	}
	for name, mutate := range failures {
		t.Run("failure "+name, func(t *testing.T) {
			env := sv5NewBranchEnv(t)
			mutate(env)
			require.ErrorIs(t, env.svc.updatePRToDefaultBranch(ctx, sv5DeletedTargetPR(env), repo, "gone", 9), errSv5Boom)
		})
	}

	t.Run("closing without changes fails in the database", func(t *testing.T) {
		env := sv5NewBranchEnv(t)
		pr := sv5DeletedTargetPR(env)
		env.bgit.mergeBaseSHA = pr.SourceSHA
		env.prs.updateErr = errSv5Boom
		require.ErrorContains(t, env.svc.updatePRToDefaultBranch(ctx, pr, repo, "gone", 9), "close PR with no changes")
	})
}
