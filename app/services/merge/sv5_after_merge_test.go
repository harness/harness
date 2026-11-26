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
	"testing"

	"github.com/harness/gitness/app/api/controller/service"
	"github.com/harness/gitness/app/bootstrap"
	pullreqevents "github.com/harness/gitness/app/events/pullreq"
	"github.com/harness/gitness/app/services/instrument"
	"github.com/harness/gitness/app/services/refcache"
	"github.com/harness/gitness/app/sse"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/app/store/cache"
	gitnessurl "github.com/harness/gitness/app/url"
	"github.com/harness/gitness/events"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/git/sha"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/check"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/require"
)

const sv5SystemPrincipalID = 77

type sv5PrincipalStore struct{ store.PrincipalStore }

func (sv5PrincipalStore) FindServiceByUID(context.Context, string) (*types.Service, error) {
	return &types.Service{ID: sv5SystemPrincipalID, UID: "sys", DisplayName: "System", Admin: true}, nil
}

// sv5Booted is set once at package initialization: the system service principal is package-level state.
var sv5Booted = sv5InitBootstrap()

func sv5InitBootstrap() bool {
	cfg := &types.Config{}
	cfg.Principal.System.UID = "sys"
	ctrl := service.NewController(check.PrincipalUIDDefault, nil, sv5PrincipalStore{})
	if err := bootstrap.SystemService(context.Background(), cfg, ctrl); err != nil {
		panic(err)
	}
	return true
}

func sv5Bootstrap(t *testing.T) {
	t.Helper()
	require.True(t, sv5Booted)
}

type sv5URLProvider struct{ gitnessurl.Provider }

func (sv5URLProvider) GetInternalAPIURL(context.Context) string { return "http://internal.test" }

type sv5DeleteGit struct {
	git.Interface

	err     error
	deleted []string
}

func (g *sv5DeleteGit) DeleteBranch(_ context.Context, p *git.DeleteBranchParams) error {
	if g.err != nil {
		return g.err
	}
	g.deleted = append(g.deleted, p.BranchName)
	return nil
}

type sv5RepoIDCache struct{ store.RepoIDCache }

func (sv5RepoIDCache) Get(_ context.Context, id int64) (*types.RepositoryCore, error) {
	if id == 404 {
		return nil, errSv5Boom
	}
	return &types.RepositoryCore{ID: id, GitUID: "git-uid", ParentID: 3, Path: "space/repo", Identifier: "repo"}, nil
}

type sv5ActivityStore struct {
	store.PullReqActivityStore

	err      error
	payloads []types.PullReqActivityPayload
}

func (s *sv5ActivityStore) CreateWithPayload(
	_ context.Context, _ *types.PullReq, _ int64, payload types.PullReqActivityPayload, _ *types.PullReqActivityMetadata,
) (*types.PullReqActivity, error) {
	s.payloads = append(s.payloads, payload)
	return &types.PullReqActivity{}, s.err
}

type sv5SSE struct {
	sse.Streamer

	published []enum.SSEType
}

func (s *sv5SSE) Publish(_ context.Context, _ int64, t enum.SSEType, _ any) {
	s.published = append(s.published, t)
}

type sv5Instrument struct {
	instrument.Service

	err    error
	events []instrument.Event
}

func (i *sv5Instrument) Track(_ context.Context, e instrument.Event) error {
	i.events = append(i.events, e)
	return i.err
}

type sv5Producer struct{ sent int }

func (p *sv5Producer) Send(context.Context, string, map[string]any) (string, error) {
	p.sent++
	return "1", nil
}

type sv5PullReqStore struct {
	store.PullReqStore

	updated []*types.PullReq
}

func (s *sv5PullReqStore) UpdateOptLock(
	_ context.Context, pr *types.PullReq, fn func(*types.PullReq) error,
) (*types.PullReq, error) {
	cp := *pr
	if err := fn(&cp); err != nil {
		return nil, err
	}
	s.updated = append(s.updated, &cp)
	return &cp, nil
}

type sv5AutoMergeStore struct{ store.AutoMergeStore }

func (sv5AutoMergeStore) Delete(context.Context, int64) (bool, error) { return false, errSv5Boom }

type sv5After struct {
	svc      *Service
	git      *sv5DeleteGit
	activity *sv5ActivityStore
	sse      *sv5SSE
	instr    *sv5Instrument
	producer *sv5Producer
	prs      *sv5PullReqStore
}

func sv5NewAfter(t *testing.T) *sv5After {
	t.Helper()
	sv5Bootstrap(t)
	env := &sv5After{
		git:      &sv5DeleteGit{},
		activity: &sv5ActivityStore{},
		sse:      &sv5SSE{},
		instr:    &sv5Instrument{},
		producer: &sv5Producer{},
		prs:      &sv5PullReqStore{},
	}
	system, err := events.NewSystem(
		func(string, string) (events.StreamConsumer, error) { return nil, errSv5Boom },
		env.producer,
		events.NewNoopCollector(),
	)
	require.NoError(t, err)
	reporter, err := pullreqevents.NewReporter(system)
	require.NoError(t, err)

	env.svc = &Service{
		git:           env.git,
		eventReporter: reporter,
		repoFinder: refcache.NewRepoFinder(
			nil, nil, sv5RepoIDCache{}, nil, cache.Evictor[*types.RepositoryCore]{}),
		pullreqStore:    env.prs,
		activityStore:   env.activity,
		autoMergeStore:  sv5AutoMergeStore{},
		urlProvider:     sv5URLProvider{},
		sseStreamer:     env.sse,
		instrumentation: env.instr,
	}
	return env
}

func sv5MergedPR() *types.PullReq {
	srcRepoID := int64(1)
	return &types.PullReq{
		ID: 7, Number: 12, SourceRepoID: &srcRepoID, TargetRepoID: 1,
		SourceBranch: "feature", TargetBranch: "main", SourceSHA: sv5SHAa,
		State: enum.PullReqStateOpen, ActivitySeq: 4,
	}
}

func sv5MergeOutput() git.MergeOutput {
	return git.MergeOutput{
		HeadSHA:  sha.Must(sv5SHAa),
		BaseSHA:  sha.Must(sv5SHAb),
		MergeSHA: sha.Must(sv5SHAc),
	}
}

func TestSv5Publish(t *testing.T) {
	env := sv5NewAfter(t)
	mergedBy := &types.PrincipalInfo{ID: 9}
	target := &types.RepositoryCore{ID: 1, ParentID: 3, Path: "space/repo", Identifier: "repo"}

	env.svc.Publish(context.Background(), sv5MergedPR(), target, enum.MergeMethodSquash, sv5MergeOutput(), mergedBy)

	require.Equal(t, 1, env.producer.sent)
	require.Equal(t, []enum.SSEType{enum.SSETypePullReqUpdated}, env.sse.published)
	require.Len(t, env.instr.events, 1)
	require.Equal(t, instrument.EventTypeMergePullRequest, env.instr.events[0].Type)
	require.Equal(t, "space/repo", env.instr.events[0].Path)
	require.Equal(t, enum.MergeMethodSquash, env.instr.events[0].Properties[instrument.PropertyMergeStrategy])

	// A failing instrumentation is only logged.
	env.instr.err = errSv5Boom
	env.svc.Publish(context.Background(), sv5MergedPR(), target, enum.MergeMethodMerge, sv5MergeOutput(), mergedBy)
	require.Len(t, env.instr.events, 2)
}

func TestSv5DeleteBranchTry(t *testing.T) {
	ctx := context.Background()
	mergedBy := &types.PrincipalInfo{ID: 9, DisplayName: "Merger", Email: "m@example.com"}

	t.Run("deleted with activity", func(t *testing.T) {
		env := sv5NewAfter(t)
		pr := sv5MergedPR()
		require.True(t, env.svc.DeleteBranchTry(ctx, pr, mergedBy, 11))
		require.Equal(t, []string{"feature"}, env.git.deleted)
		require.Len(t, env.activity.payloads, 1)
		require.Equal(t, int64(11), pr.ActivitySeq)
		require.Equal(t, &types.PullRequestActivityPayloadBranchDelete{SHA: sv5SHAa}, env.activity.payloads[0])
	})

	t.Run("activity failure is tolerated", func(t *testing.T) {
		env := sv5NewAfter(t)
		env.activity.err = errSv5Boom
		require.True(t, env.svc.DeleteBranchTry(ctx, sv5MergedPR(), mergedBy, 11))
	})

	t.Run("no source repo", func(t *testing.T) {
		env := sv5NewAfter(t)
		pr := sv5MergedPR()
		pr.SourceRepoID = nil
		require.False(t, env.svc.DeleteBranchTry(ctx, pr, mergedBy, 11))
		require.Empty(t, env.git.deleted)
	})

	t.Run("source repo lookup failure", func(t *testing.T) {
		env := sv5NewAfter(t)
		pr := sv5MergedPR()
		missing := int64(404)
		pr.SourceRepoID = &missing
		require.False(t, env.svc.DeleteBranchTry(ctx, pr, mergedBy, 11))
	})

	t.Run("git failure", func(t *testing.T) {
		env := sv5NewAfter(t)
		env.git.err = errSv5Boom
		require.False(t, env.svc.DeleteBranchTry(ctx, sv5MergedPR(), mergedBy, 11))
		require.Empty(t, env.activity.payloads)
	})
}

func TestSv5CreateRPCWriteParams(t *testing.T) {
	env := sv5NewAfter(t)
	ctx := context.Background()

	params, err := env.svc.CreateRPCWriteParams(
		ctx, &types.PrincipalInfo{ID: 9, DisplayName: "Merger", Email: "m@example.com"}, 1)
	require.NoError(t, err)
	require.Equal(t, "git-uid", params.RepoUID)
	require.Equal(t, git.Identity{Name: "Merger", Email: "m@example.com"}, params.Actor)
	require.NotEmpty(t, params.EnvVars)

	_, err = env.svc.CreateRPCWriteParams(ctx, &types.PrincipalInfo{ID: 9}, 404)
	require.ErrorContains(t, err, "failed to find repo")
}

func TestSv5DatabaseUpdate(t *testing.T) {
	env := sv5NewAfter(t)
	mergedBy := &types.PrincipalInfo{ID: 9}

	pr, seqBranchDeleted, err := env.svc.DatabaseUpdate(
		context.Background(), sv5MergedPR(), enum.MergeMethodRebase, sv5MergeOutput(), mergedBy, false, "")
	require.NoError(t, err)
	require.NotNil(t, pr.Merged)
	require.Equal(t, enum.PullReqStateMerged, pr.State)
	require.Positive(t, seqBranchDeleted)
	require.Len(t, env.prs.updated, 1)
	require.Len(t, env.activity.payloads, 1)

	env.activity.err = errSv5Boom
	_, _, err = env.svc.DatabaseUpdate(
		context.Background(), sv5MergedPR(), enum.MergeMethodMerge, sv5MergeOutput(), mergedBy, true, "msg")
	require.NoError(t, err)
}

func TestSv5PrepareInputSystemCommitter(t *testing.T) {
	sv5Bootstrap(t)
	svc := &Service{}
	principal := &types.PrincipalInfo{ID: 9, DisplayName: "Me", Email: "me@example.com"}
	repo := &types.RepositoryCore{Path: "space/repo"}

	merge := svc.PreparePullReqMergeInputNoRefUpdates(sv5PR(), repo, principal, enum.MergeMethodMerge, "", "msg")
	require.Equal(t, "me@example.com", merge.Author.Email)
	require.Equal(t, int64(0), int64(len(merge.RefUpdates)))
	require.NotNil(t, merge.Committer)
	require.Equal(t, git.CommitMessage("Merge branch 'feature' of space/repo (#12)", "msg"), merge.CommitMessage)

	fork := svc.PreparePullReqMergeInputNoRefUpdates(sv5PR(), nil, principal, enum.MergeMethodMerge, "", "")
	require.Equal(t, git.CommitMessage("Merge branch 'feature' of deleted fork (#12)", ""), fork.CommitMessage)

	squash := svc.PreparePullReqMergeInputNoRefUpdates(sv5PR(), repo, principal, enum.MergeMethodSquash, "", "")
	require.Equal(t, "author@example.com", squash.Author.Email)
	require.NotNil(t, squash.Committer)
	require.Equal(t, git.CommitMessage("Add feature (#12)", ""), squash.CommitMessage)
}
