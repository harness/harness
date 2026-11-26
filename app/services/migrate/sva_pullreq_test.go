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

package migrate

import (
	"context"
	"testing"
	"time"

	"github.com/harness/gitness/app/services/refcache"
	"github.com/harness/gitness/app/store"
	storecache "github.com/harness/gitness/app/store/cache"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/git/sha"
	gitness_store "github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	migratetypes "github.com/harness/harness-migrate/types"
	"github.com/stretchr/testify/require"
)

const (
	svaHeadSHA      = "d79d48937903b01dd8e72224dc6989d910fe8fe8"
	svaBaseSHA      = "c1f79b793fc423705a85ddd9ab9f218b23e0caa9"
	svaSourceTip    = "1111111111111111111111111111111111111111"
	svaTargetTip    = "2222222222222222222222222222222222222222"
	svaMergeBaseSHA = "3333333333333333333333333333333333333333"

	svaAuthorEmail   = "author@example.com"
	svaReviewerEmail = "reviewer@example.com"
	svaCommenter     = "commenter@example.com"
	svaStranger      = "stranger@example.com"
)

type svaPullReqStore struct {
	store.PullReqStore

	created      []*types.PullReq
	updated      []*types.PullReq
	createErr    error
	updateErr    error
	activitySeqE error
	nextID       int64
}

func (s *svaPullReqStore) Create(_ context.Context, pr *types.PullReq) error {
	if s.createErr != nil {
		return s.createErr
	}
	s.nextID++
	pr.ID = 1000 + s.nextID
	s.created = append(s.created, pr)
	return nil
}

func (s *svaPullReqStore) Update(_ context.Context, pr *types.PullReq) error {
	if s.updateErr != nil {
		return s.updateErr
	}
	s.updated = append(s.updated, pr)
	return nil
}

func (s *svaPullReqStore) UpdateActivitySeq(_ context.Context, pr *types.PullReq) (*types.PullReq, error) {
	if s.activitySeqE != nil {
		return nil, s.activitySeqE
	}
	updated := *pr
	updated.ActivitySeq++
	return &updated, nil
}

type svaActivityStore struct {
	store.PullReqActivityStore

	created []*types.PullReqActivity
	err     error
	failOn  int // 1-based index of the call that fails when err is set; 0 fails every call.
	calls   int
	nextID  int64
}

func (s *svaActivityStore) Create(_ context.Context, act *types.PullReqActivity) error {
	s.calls++
	if s.err != nil && (s.failOn == 0 || s.failOn == s.calls) {
		return s.err
	}
	s.nextID++
	act.ID = 2000 + s.nextID
	s.created = append(s.created, act)
	return nil
}

type svaReviewStore struct {
	store.PullReqReviewStore

	created []*types.PullReqReview
	err     error
}

func (s *svaReviewStore) Create(_ context.Context, review *types.PullReqReview) error {
	if s.err != nil {
		return s.err
	}
	review.ID = int64(3000 + len(s.created))
	s.created = append(s.created, review)
	return nil
}

type svaReviewerStore struct {
	store.PullReqReviewerStore

	existing  map[int64]*types.PullReqReviewer
	created   []*types.PullReqReviewer
	updated   []*types.PullReqReviewer
	createErr error
	findErr   error
	updateErr error
}

func (s *svaReviewerStore) Find(_ context.Context, _, principalID int64) (*types.PullReqReviewer, error) {
	if s.findErr != nil {
		return nil, s.findErr
	}
	if reviewer, ok := s.existing[principalID]; ok {
		return reviewer, nil
	}
	return nil, gitness_store.ErrResourceNotFound
}

func (s *svaReviewerStore) Create(_ context.Context, reviewer *types.PullReqReviewer) error {
	if s.createErr != nil {
		return s.createErr
	}
	s.created = append(s.created, reviewer)
	s.existing[reviewer.PrincipalID] = reviewer
	return nil
}

func (s *svaReviewerStore) Update(_ context.Context, reviewer *types.PullReqReviewer) error {
	if s.updateErr != nil {
		return s.updateErr
	}
	s.updated = append(s.updated, reviewer)
	return nil
}

type svaAssignStore struct {
	store.PullReqLabelAssignmentStore

	assigned []*types.PullReqLabel
	err      error
}

func (s *svaAssignStore) Assign(_ context.Context, label *types.PullReqLabel) error {
	if s.err != nil {
		return s.err
	}
	s.assigned = append(s.assigned, label)
	return nil
}

type svaRepoStore struct {
	store.RepoStore

	repo      *types.Repository
	findErr   error
	updateErr error
	updated   *types.Repository
}

func (s *svaRepoStore) Find(context.Context, int64) (*types.Repository, error) {
	if s.findErr != nil {
		return nil, s.findErr
	}
	repoCopy := *s.repo
	return &repoCopy, nil
}

func (s *svaRepoStore) Update(_ context.Context, repo *types.Repository) error {
	if s.updateErr != nil {
		return s.updateErr
	}
	s.updated = repo
	return nil
}

type svaGit struct {
	git.Interface

	branches     map[string]string
	mergeBaseErr error
}

func (g *svaGit) GetBranch(_ context.Context, params *git.GetBranchParams) (*git.GetBranchOutput, error) {
	tip, ok := g.branches[params.BranchName]
	if !ok {
		return nil, gitness_store.ErrResourceNotFound
	}
	return &git.GetBranchOutput{Branch: git.Branch{Name: params.BranchName, SHA: sha.Must(tip)}}, nil
}

func (g *svaGit) MergeBase(context.Context, git.MergeBaseParams) (git.MergeBaseOutput, error) {
	return git.MergeBaseOutput{MergeBaseSHA: sha.Must(svaMergeBaseSHA)}, g.mergeBaseErr
}

type svaRepoIDCache struct{}

func (svaRepoIDCache) Stats() (int64, int64) { return 0, 0 }

func (svaRepoIDCache) Get(context.Context, int64) (*types.RepositoryCore, error) {
	return nil, gitness_store.ErrResourceNotFound
}

func (svaRepoIDCache) Evict(context.Context, int64) {}

// svaFixture bundles the fakes used to exercise PullReq.Import.
type svaFixture struct {
	prs        *svaPullReqStore
	acts       *svaActivityStore
	reviews    *svaReviewStore
	reviewers  *svaReviewerStore
	assigns    *svaAssignStore
	repos      *svaRepoStore
	labels     *svaLabelStore
	labelVals  *svaLabelValueStore
	spaces     *svaSpaceStore
	principals *svaPrincipalStore
	git        *svaGit
	tx         svaTx
}

func svaNewFixture() *svaFixture {
	return &svaFixture{
		prs:       &svaPullReqStore{},
		acts:      &svaActivityStore{},
		reviews:   &svaReviewStore{},
		reviewers: &svaReviewerStore{existing: map[int64]*types.PullReqReviewer{}},
		assigns:   &svaAssignStore{},
		repos: &svaRepoStore{repo: &types.Repository{
			ID: 1, RootSpaceID: 5, RootSpaceIdentifier: "root", PullReqSeq: 2, NumPulls: 2, NumOpenPulls: 1,
		}},
		labels:    svaNewLabelStore(),
		labelVals: svaNewLabelValueStore(),
		spaces:    &svaSpaceStore{ancestors: []int64{1, 2}},
		principals: &svaPrincipalStore{byEmail: map[string]*types.Principal{
			svaAuthorEmail:   {ID: 10, UID: "author"},
			svaReviewerEmail: {ID: 11, UID: "reviewer"},
			svaCommenter:     {ID: 12, UID: "commenter"},
		}},
		git: &svaGit{branches: map[string]string{"feature": svaSourceTip, "main": svaTargetTip}},
	}
}

func (f *svaFixture) importer() *PullReq {
	finder := refcache.NewRepoFinder(nil, nil, svaRepoIDCache{}, nil, storecache.Evictor[*types.RepositoryCore]{})

	return NewPullReq(
		nil, f.git, f.principals, f.spaces, f.repos, f.prs, f.acts, f.labels, f.labelVals,
		f.assigns, f.reviewers, f.reviews, finder, f.tx, nil,
	)
}

func (f *svaFixture) state() *repoImportState {
	return &repoImportState{
		git:                         f.git,
		principalStore:              f.principals,
		spaceStore:                  f.spaces,
		pullReqStore:                f.prs,
		pullReqActivityStore:        f.acts,
		labelStore:                  f.labels,
		labelValueStore:             f.labelVals,
		pullReqLabelAssignmentStore: f.assigns,
		pullReqReviewerStore:        f.reviewers,
		pullReqReviewStore:          f.reviews,
		branchCheck:                 map[string]*git.Branch{},
		principals:                  map[string]*types.Principal{},
		unknownEmails:               map[int]map[string]bool{},
		labels:                      map[string]int64{},
		labelValues:                 map[int64]map[string]*int64{},
		migrator:                    types.Principal{ID: 1, UID: "migrator"},
		rootSpaceID:                 5,
		rootSpaceIdentifier:         "root",
	}
}

var svaMigrator = types.Principal{ID: 1, UID: "migrator"}

func svaExtPR(number int, mutate func(pr *ExternalPullRequest)) *ExternalPullRequest {
	ext := &ExternalPullRequest{}
	ext.PullRequest.Number = number
	ext.PullRequest.Title = "Title"
	ext.PullRequest.Author.Email = svaAuthorEmail
	ext.PullRequest.Head = migratetypes.Reference{Name: "feature", SHA: svaHeadSHA}
	ext.PullRequest.Base = migratetypes.Reference{Name: "main", SHA: svaBaseSHA}
	ext.PullRequest.Created = time.UnixMilli(1_700_000_000_000)
	ext.PullRequest.Updated = time.UnixMilli(1_700_000_100_000)
	if mutate != nil {
		mutate(ext)
	}
	return ext
}

func svaRepoCore() *types.RepositoryCore {
	return &types.RepositoryCore{ID: 1, ParentID: 4, GitUID: "git-uid", Identifier: "repo"}
}

func TestSvaPullReqImportFullFlow(t *testing.T) {
	t.Parallel()

	fix := svaNewFixture()
	created := time.UnixMilli(1_700_000_050_000)

	merged := svaExtPR(3, func(ext *ExternalPullRequest) {
		ext.PullRequest.Merged = true
		ext.PullRequest.Title = string(make([]byte, 300))
		ext.PullRequest.Labels = []migratetypes.Label{
			{Name: "bug", Color: "#C7292F", Value: "high"},
			{Name: "bug", Color: "#C7292F", Value: "high"},
			{Name: "docs"},
		}
		ext.Comments = []ExternalComment{
			{ID: 1, Body: "top level", Author: migratetypes.User{Email: svaCommenter}, Created: created},
			{ID: 2, Body: "reply", ParentID: 1, Author: migratetypes.User{Email: svaStranger}, Created: created},
			{
				ID: 3, Body: "code", Author: migratetypes.User{Email: svaReviewerEmail}, Created: created,
				CodeComment: &migratetypes.CodeComment{
					Path: "a.go", HunkHeader: "@@ -1,2 +3,4 @@", SourceSHA: svaHeadSHA, Side: "OLD",
					CodeSnippet: migratetypes.Hunk{Header: "@@ -1,2 +3,4 @@", Lines: []string{"x"}},
				},
			},
		}
		ext.Reviewers = []ExternalReviewer{
			{User: migratetypes.User{Email: svaReviewerEmail}},
			{User: migratetypes.User{Email: svaAuthorEmail}}, // The author is never a reviewer of its own PR.
		}
		ext.Reviews = []ExternalReview{
			{
				Author: migratetypes.User{Email: svaReviewerEmail}, SHA: svaHeadSHA, Decision: "approved",
				Created: created, Updated: created,
			},
			{Author: migratetypes.User{Email: svaCommenter}, SHA: svaHeadSHA, Decision: "changereq"},
			{Author: migratetypes.User{Email: svaAuthorEmail}, SHA: svaHeadSHA, Decision: "approved"},
		}
	})
	closed := svaExtPR(1, func(ext *ExternalPullRequest) {
		ext.PullRequest.Closed = true
		ext.PullRequest.Draft = true
	})
	open := svaExtPR(2, nil)

	got, err := fix.importer().Import(
		context.Background(), svaMigrator, svaRepoCore(), []*ExternalPullRequest{merged, closed, open})
	require.NoError(t, err)
	require.Len(t, got, 3)

	byNumber := map[int64]*types.PullReq{}
	for _, pr := range got {
		byNumber[pr.Number] = pr
	}

	// States and per-state data.
	mergedPR, closedPR, openPR := byNumber[3], byNumber[1], byNumber[2]
	require.Equal(t, enum.PullReqStateMerged, mergedPR.State)
	require.Equal(t, enum.PullReqStateClosed, closedPR.State)
	require.Equal(t, enum.PullReqStateOpen, openPR.State)
	require.Len(t, mergedPR.Title, 256, "overlong titles are truncated")
	require.True(t, closedPR.IsDraft)
	require.NotNil(t, closedPR.Closed)
	require.Equal(t, svaSourceTip, openPR.SourceSHA, "open PRs use the live source branch tip")
	require.Equal(t, svaMergeBaseSHA, openPR.MergeBaseSHA)
	require.Equal(t, int64(10), mergedPR.CreatedBy)
	require.Equal(t, int64(5), mergedPR.RootSpaceID)

	// Repository counters.
	require.NotNil(t, fix.repos.updated)
	require.Equal(t, int64(3), fix.repos.updated.PullReqSeq)
	require.Equal(t, 5, fix.repos.updated.NumPulls)
	require.Equal(t, 2, fix.repos.updated.NumOpenPulls)
	require.Equal(t, 1, fix.repos.updated.NumClosedPulls)
	require.Equal(t, 1, fix.repos.updated.NumMergedPulls)

	// Only the PR that received comments/labels is updated after the import.
	require.Len(t, fix.prs.updated, 1)
	require.Equal(t, int64(3), fix.prs.updated[0].Number)

	// Labels: "bug" and "docs" are defined once, the value only once.
	require.Len(t, fix.labels.defined, 2)
	require.Len(t, fix.assigns.assigned, 3)
	bug := fix.labels.labels["bug"]
	require.Len(t, fix.labelVals.values[bug.ID], 1)
	require.Equal(t, 1, fix.labels.increments[bug.ID])
	require.NotNil(t, fix.assigns.assigned[0].ValueID)
	require.Nil(t, fix.assigns.assigned[2].ValueID)

	// Reviewers: the reviewer is requested, the commenter is added as self-assigned through the review.
	require.Len(t, fix.reviewers.created, 2)
	require.Equal(t, enum.PullReqReviewerTypeRequested, fix.reviewers.created[0].Type)
	require.Equal(t, enum.PullReqReviewerTypeSelfAssigned, fix.reviewers.created[1].Type)
	require.Equal(t, int64(12), fix.reviewers.created[1].PrincipalID)
	require.Len(t, fix.reviews.created, 2)
	require.Equal(t, enum.PullReqReviewDecision("changereq"), fix.reviews.created[1].Decision)
	require.Len(t, fix.reviewers.updated, 1, "the requested reviewer gets its latest review recorded")
	require.Equal(t, int64(11), fix.reviewers.updated[0].PrincipalID)
	require.Equal(t, enum.PullReqReviewDecision("approved"), fix.reviewers.updated[0].ReviewDecision)

	// Comments: thread of two + code comment + info comment about the replaced unknown user.
	var comments, codeComments, replies, systemActs, infoComments int
	for _, act := range fix.acts.created {
		switch {
		case act.Type == enum.PullReqActivityTypeCodeComment:
			codeComments++
			require.NotNil(t, act.CodeComment)
			require.False(t, act.CodeComment.Outdated)
			require.Equal(t, 3, act.CodeComment.LineNew)
			require.Equal(t, 4, act.CodeComment.SpanNew)
			require.Equal(t, 1, act.CodeComment.LineOld)
			require.Equal(t, 2, act.CodeComment.SpanOld)
			require.NotNil(t, act.Resolved, "comments of merged PRs are resolved")
		case act.Type == enum.PullReqActivityTypeComment && act.ParentID != nil:
			replies++
			require.Equal(t, int64(1), act.CreatedBy, "unknown authors are replaced by the migrator")
		case act.Type == enum.PullReqActivityTypeComment && act.CreatedBy == 1:
			infoComments++
			require.Contains(t, act.Text, svaStranger)
		case act.Type == enum.PullReqActivityTypeComment:
			comments++
		case act.Kind == enum.PullReqActivityKindSystem:
			systemActs++
		}
	}
	require.Equal(t, 1, comments)
	require.Equal(t, 1, codeComments)
	require.Equal(t, 1, replies)
	require.Equal(t, 1, infoComments)
	require.Equal(t, 3, systemActs, "one reviewer-added activity and two review-submitted activities")
}

func TestSvaPullReqImportEmptyAndDuplicates(t *testing.T) {
	t.Parallel()

	t.Run("empty list is a no-op", func(t *testing.T) {
		t.Parallel()

		fix := svaNewFixture()
		got, err := fix.importer().Import(context.Background(), svaMigrator, svaRepoCore(), nil)
		require.NoError(t, err)
		require.Nil(t, got)
		require.Empty(t, fix.prs.created)
		require.Nil(t, fix.repos.updated)
	})

	t.Run("duplicate numbers conflict", func(t *testing.T) {
		t.Parallel()

		fix := svaNewFixture()
		closed := func(ext *ExternalPullRequest) { ext.PullRequest.Closed = true }
		_, err := fix.importer().Import(context.Background(), svaMigrator, svaRepoCore(),
			[]*ExternalPullRequest{svaExtPR(1, closed), svaExtPR(1, closed)})
		require.ErrorContains(t, err, "duplicate pull request number 1")
		require.Empty(t, fix.prs.created)
	})
}

func TestSvaPullReqImportErrors(t *testing.T) {
	t.Parallel()

	closed := func(ext *ExternalPullRequest) { ext.PullRequest.Closed = true }
	withComment := func(ext *ExternalPullRequest) {
		ext.PullRequest.Closed = true
		ext.Comments = []ExternalComment{{ID: 1, Body: "hi", Author: migratetypes.User{Email: svaCommenter}}}
	}

	tests := []struct {
		name  string
		pr    func(ext *ExternalPullRequest)
		setup func(f *svaFixture)
	}{
		{name: "repository lookup", pr: closed, setup: func(f *svaFixture) { f.repos.findErr = errSvaBoom }},
		{name: "author lookup", pr: closed, setup: func(f *svaFixture) { f.principals.err = errSvaBoom }},
		{name: "source branch missing", setup: func(f *svaFixture) { delete(f.git.branches, "feature") }},
		{name: "target branch missing", setup: func(f *svaFixture) { delete(f.git.branches, "main") }},
		{name: "merge base failure", setup: func(f *svaFixture) { f.git.mergeBaseErr = errSvaBoom }},
		{name: "create pull request", pr: closed, setup: func(f *svaFixture) { f.prs.createErr = errSvaBoom }},
		{
			name: "create reviewers",
			pr: func(ext *ExternalPullRequest) {
				closed(ext)
				ext.Reviewers = []ExternalReviewer{{User: migratetypes.User{Email: svaReviewerEmail}}}
			},
			setup: func(f *svaFixture) { f.reviewers.createErr = errSvaBoom },
		},
		{
			name: "create reviews",
			pr: func(ext *ExternalPullRequest) {
				closed(ext)
				ext.Reviews = []ExternalReview{{Author: migratetypes.User{Email: svaReviewerEmail}, Decision: "approved"}}
			},
			setup: func(f *svaFixture) { f.reviews.err = errSvaBoom },
		},
		{name: "create comments", pr: withComment, setup: func(f *svaFixture) { f.acts.err = errSvaBoom }},
		{
			name: "assign labels",
			pr: func(ext *ExternalPullRequest) {
				closed(ext)
				ext.PullRequest.Labels = []migratetypes.Label{{Name: "bug"}}
			},
			setup: func(f *svaFixture) { f.assigns.err = errSvaBoom },
		},
		{name: "update pull request", pr: withComment, setup: func(f *svaFixture) { f.prs.updateErr = errSvaBoom }},
		{name: "update repository", pr: closed, setup: func(f *svaFixture) { f.repos.updateErr = errSvaBoom }},
		{name: "transaction", pr: closed, setup: func(f *svaFixture) { f.tx = svaTx{err: errSvaBoom} }},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			fix := svaNewFixture()
			test.setup(fix)

			_, err := fix.importer().Import(context.Background(), svaMigrator, svaRepoCore(),
				[]*ExternalPullRequest{svaExtPR(1, test.pr)})
			require.Error(t, err)
		})
	}
}

func TestSvaCreateCommentErrors(t *testing.T) {
	t.Parallel()

	repo := svaRepoCore()

	t.Run("invalid hunk header", func(t *testing.T) {
		t.Parallel()

		state := svaNewFixture().state()
		_, err := state.createComment(context.Background(), repo, &types.PullReq{Number: 1}, nil, 1, 0, 0,
			&ExternalComment{
				ID: 9, Author: migratetypes.User{Email: svaCommenter},
				CodeComment: &migratetypes.CodeComment{HunkHeader: "not a hunk"},
			})
		require.ErrorContains(t, err, "Invalid hunk header")
	})

	t.Run("replies never become code comments", func(t *testing.T) {
		t.Parallel()

		fix := svaNewFixture()
		state := fix.state()
		parent := int64(77)
		act, err := state.createComment(context.Background(), repo, &types.PullReq{Number: 1}, &parent, 1, 1, 0,
			&ExternalComment{
				ID: 9, ParentID: 5, Body: "reply", Author: migratetypes.User{Email: svaCommenter},
				CodeComment: &migratetypes.CodeComment{HunkHeader: "@@ -1 +1 @@", SourceSHA: "abc"},
			})
		require.NoError(t, err)
		require.Equal(t, enum.PullReqActivityTypeComment, act.Type)
		require.Nil(t, act.CodeComment)
	})

	t.Run("outdated code comments and new side", func(t *testing.T) {
		t.Parallel()

		fix := svaNewFixture()
		state := fix.state()
		pr := &types.PullReq{Number: 1, SourceSHA: "current"}
		act, err := state.createComment(context.Background(), repo, pr, nil, 1, 0, 0,
			&ExternalComment{
				ID: 9, Body: "long " + string(make([]byte, 70<<10)), Author: migratetypes.User{Email: svaCommenter},
				CodeComment: &migratetypes.CodeComment{HunkHeader: "@@ -1 +1 @@", SourceSHA: "older", Side: "NEW"},
			})
		require.NoError(t, err)
		require.True(t, act.CodeComment.Outdated)
		require.Len(t, act.Text, 64<<10)
		require.Equal(t, 1, pr.CommentCount)
		require.Equal(t, 1, pr.UnresolvedCount)
		require.Equal(t, int64(1), pr.ActivitySeq)
	})

	t.Run("author lookup", func(t *testing.T) {
		t.Parallel()

		fix := svaNewFixture()
		fix.principals.err = errSvaBoom
		_, err := fix.state().createComment(context.Background(), repo, &types.PullReq{}, nil, 1, 0, 0,
			&ExternalComment{Author: migratetypes.User{Email: svaCommenter}})
		require.ErrorIs(t, err, errSvaBoom)
	})
}

func TestSvaCreateInfoCommentAndPrincipals(t *testing.T) {
	t.Parallel()

	t.Run("info comment is created", func(t *testing.T) {
		t.Parallel()

		fix := svaNewFixture()
		state := fix.state()
		state.unknownEmails[4] = map[string]bool{svaStranger: true}
		pr := &types.PullReq{ID: 9, Number: 4, ActivitySeq: 2}

		act, err := state.createInfoComment(context.Background(), svaRepoCore(), pr)
		require.NoError(t, err)
		require.Equal(t, int64(3), act.Order)
		require.Equal(t, int64(3), pr.ActivitySeq)
		require.Equal(t, 1, pr.CommentCount)
		require.Contains(t, act.Text, "migrator")
	})

	t.Run("info comment store failure", func(t *testing.T) {
		t.Parallel()

		fix := svaNewFixture()
		fix.acts.err = errSvaBoom
		_, err := fix.state().createInfoComment(context.Background(), svaRepoCore(), &types.PullReq{})
		require.ErrorIs(t, err, errSvaBoom)
	})

	t.Run("principal resolution", func(t *testing.T) {
		t.Parallel()

		fix := svaNewFixture()
		state := fix.state()
		ctx := context.Background()

		known, err := state.getPrincipalByEmail(ctx, svaAuthorEmail, 1, true)
		require.NoError(t, err)
		require.Equal(t, int64(10), known.ID)

		// A second lookup is served from the cache.
		fix.principals.err = errSvaBoom
		cached, err := state.getPrincipalByEmail(ctx, svaAuthorEmail, 1, true)
		require.NoError(t, err)
		require.Same(t, known, cached)

		_, err = state.getPrincipalByEmail(ctx, svaStranger, 1, false)
		require.ErrorIs(t, err, errSvaBoom)

		fix.principals.err = nil
		_, err = state.getPrincipalByEmail(ctx, svaStranger, 1, true)
		require.ErrorIs(t, err, gitness_store.ErrResourceNotFound)

		fallback, err := state.getPrincipalByEmail(ctx, svaStranger, 1, false)
		require.NoError(t, err)
		require.Equal(t, int64(1), fallback.ID)
		require.True(t, state.unknownEmails[1][svaStranger])
	})

	t.Run("unknown emails are capped", func(t *testing.T) {
		t.Parallel()

		state := svaNewFixture().state()
		state.unknownEmails[1] = map[string]bool{}
		for i := range MaxNumberOfUnknownEmails {
			state.unknownEmails[1][string(rune('a'+i%26))+string(rune('a'+i/26))+"@x.io"] = true
		}
		require.Len(t, state.unknownEmails[1], MaxNumberOfUnknownEmails)

		principal, err := state.getPrincipalByEmail(context.Background(), svaStranger, 1, false)
		require.NoError(t, err)
		require.Equal(t, int64(1), principal.ID)
		require.Len(t, state.unknownEmails[1], MaxNumberOfUnknownEmails)
	})
}

func TestSvaAssignLabels(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("no labels", func(t *testing.T) {
		t.Parallel()

		fix := svaNewFixture()
		require.NoError(t, fix.state().assignLabels(ctx, 4, &types.PullReq{}, nil))
		require.Empty(t, fix.assigns.assigned)
	})

	t.Run("reuses cached and stored labels and values", func(t *testing.T) {
		t.Parallel()

		fix := svaNewFixture()
		fix.labels.labels["bug"] = &types.Label{ID: 42, Key: "bug"}
		fix.labelVals.values[42] = []*types.LabelValue{{ID: 43, LabelID: 42, Value: "high"}}
		pr := &types.PullReq{ID: 9}

		require.NoError(t, fix.state().assignLabels(ctx, 4, pr, []ExternalLabel{
			{Name: "bug", Value: "high"},
			{Name: "bug", Value: "high"},
		}))
		require.Empty(t, fix.labels.defined)
		require.Len(t, fix.assigns.assigned, 2)
		require.Equal(t, int64(42), fix.assigns.assigned[0].LabelID)
		require.Equal(t, int64(43), *fix.assigns.assigned[0].ValueID)
		require.Equal(t, int64(2), pr.ActivitySeq)
	})

	errTests := []struct {
		name  string
		setup func(f *svaFixture)
	}{
		{name: "label lookup", setup: func(f *svaFixture) { f.labels.findErr = errSvaBoom }},
		{name: "ancestors lookup", setup: func(f *svaFixture) { f.spaces.err = errSvaBoom }},
		{name: "label define", setup: func(f *svaFixture) { f.labels.defineErr = errSvaBoom }},
		{name: "value lookup", setup: func(f *svaFixture) { f.labelVals.findErr = errSvaBoom }},
		{name: "value define", setup: func(f *svaFixture) { f.labelVals.defineErr = errSvaBoom }},
		{name: "value count", setup: func(f *svaFixture) { f.labels.incrementErr = errSvaBoom }},
		{name: "assignment", setup: func(f *svaFixture) { f.assigns.err = errSvaBoom }},
	}
	for _, test := range errTests {
		t.Run("error "+test.name, func(t *testing.T) {
			t.Parallel()

			fix := svaNewFixture()
			test.setup(fix)
			err := fix.state().assignLabels(ctx, 4, &types.PullReq{}, []ExternalLabel{{Name: "bug", Value: "high"}})
			require.Error(t, err)
		})
	}
}

func TestSvaReviewerAndReviewActivities(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := svaRepoCore()

	t.Run("reviewer activity increments the sequence and records the reviewers", func(t *testing.T) {
		t.Parallel()

		fix := svaNewFixture()
		pr := &types.PullReq{ID: 9, TargetRepoID: 1, ActivitySeq: 4}

		fix.state().createReviewerActivity(ctx, pr, []int64{11, 12}, enum.PullReqReviewerTypeRequested)
		require.Equal(t, int64(5), pr.ActivitySeq)
		require.Len(t, fix.acts.created, 1)
		require.Equal(t, int64(5), fix.acts.created[0].Order)
		require.Equal(t, enum.PullReqActivityTypeReviewerAdd, fix.acts.created[0].Type)
		require.Equal(t, []int64{11, 12}, fix.acts.created[0].Metadata.Mentions.IDs)
	})

	t.Run("self assigned reviewers are recorded with their type", func(t *testing.T) {
		t.Parallel()

		fix := svaNewFixture()
		fix.state().createReviewerActivity(ctx, &types.PullReq{ID: 9}, []int64{11}, enum.PullReqReviewerTypeSelfAssigned)
		require.Len(t, fix.acts.created, 1)
		require.Contains(t, string(fix.acts.created[0].PayloadRaw), string(enum.PullReqReviewerTypeSelfAssigned))
	})

	t.Run("requested reviewers are created", func(t *testing.T) {
		t.Parallel()

		fix := svaNewFixture()
		reviewers, err := fix.state().createReviewers(ctx, repo, &types.PullReq{ID: 9, CreatedBy: 10},
			[]ExternalReviewer{
				{User: migratetypes.User{Email: svaReviewerEmail}},
				{User: migratetypes.User{Email: svaAuthorEmail}},
			})
		require.NoError(t, err)
		require.Len(t, reviewers, 1)
		require.Equal(t, int64(11), reviewers[0].PrincipalID)
	})

	t.Run("reviewer activity without reviewers is a no-op", func(t *testing.T) {
		t.Parallel()

		fix := svaNewFixture()
		fix.state().createReviewerActivity(ctx, &types.PullReq{}, nil, enum.PullReqReviewerTypeRequested)
		require.Empty(t, fix.acts.created)
	})

	t.Run("activity failures are tolerated", func(t *testing.T) {
		t.Parallel()

		fix := svaNewFixture()
		fix.prs.activitySeqE = errSvaBoom
		pr := &types.PullReq{ActivitySeq: 4}
		fix.state().createReviewerActivity(ctx, pr, []int64{1}, enum.PullReqReviewerTypeRequested)
		fix.state().createReviewSubmitActivity(ctx, pr, &types.PullReqReview{})
		require.Equal(t, int64(4), pr.ActivitySeq)
		require.Empty(t, fix.acts.created)

		fix = svaNewFixture()
		fix.acts.err = errSvaBoom
		fix.state().createReviewerActivity(ctx, &types.PullReq{}, []int64{1}, enum.PullReqReviewerTypeRequested)
		fix.state().createReviewSubmitActivity(ctx, &types.PullReq{}, &types.PullReqReview{})
		require.Empty(t, fix.acts.created)
	})

	t.Run("review submit activity", func(t *testing.T) {
		t.Parallel()

		fix := svaNewFixture()
		pr := &types.PullReq{ID: 9, TargetRepoID: 1, ActivitySeq: 1}
		review := &types.PullReqReview{ID: 3, CreatedBy: 11, SHA: "abc", Decision: enum.PullReqReviewDecisionApproved}

		fix.state().createReviewSubmitActivity(ctx, pr, review)
		require.Equal(t, int64(2), pr.ActivitySeq)
		require.Len(t, fix.acts.created, 1)
		require.Equal(t, enum.PullReqActivityTypeReviewSubmit, fix.acts.created[0].Type)
		require.Equal(t, int64(11), fix.acts.created[0].CreatedBy)
	})

	t.Run("reviewer errors", func(t *testing.T) {
		t.Parallel()

		fix := svaNewFixture()
		fix.principals.err = errSvaBoom
		_, err := fix.state().createReviewers(ctx, repo, &types.PullReq{},
			[]ExternalReviewer{{User: migratetypes.User{Email: svaReviewerEmail}}})
		require.ErrorIs(t, err, errSvaBoom)

		_, err = fix.state().createReviews(ctx, repo, &types.PullReq{},
			[]ExternalReview{{Author: migratetypes.User{Email: svaReviewerEmail}}})
		require.ErrorIs(t, err, errSvaBoom)
	})

	t.Run("review store errors", func(t *testing.T) {
		t.Parallel()

		review := []ExternalReview{{Author: migratetypes.User{Email: svaReviewerEmail}, Decision: "approved"}}
		pr := &types.PullReq{ID: 9, CreatedBy: 10}

		fix := svaNewFixture()
		fix.reviewers.findErr = errSvaBoom
		_, err := fix.state().createReviews(ctx, repo, pr, review)
		require.ErrorIs(t, err, errSvaBoom)

		fix = svaNewFixture()
		fix.reviewers.createErr = errSvaBoom
		_, err = fix.state().createReviews(ctx, repo, pr, review)
		require.ErrorIs(t, err, errSvaBoom)
	})

	t.Run("failure to update an existing reviewer is not fatal", func(t *testing.T) {
		t.Parallel()

		fix := svaNewFixture()
		fix.reviewers.existing[11] = &types.PullReqReviewer{PrincipalID: 11}
		fix.reviewers.updateErr = errSvaBoom

		reviews, err := fix.state().createReviews(ctx, repo, &types.PullReq{ID: 9, CreatedBy: 10},
			[]ExternalReview{{Author: migratetypes.User{Email: svaReviewerEmail}, Decision: "approved"}})
		require.NoError(t, err)
		require.Len(t, reviews, 1)
	})
}

func TestSvaConvertPullReqClosedAndMerged(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	repo := svaRepoCore()

	t.Run("merged", func(t *testing.T) {
		t.Parallel()

		pr, err := svaNewFixture().state().convertPullReq(ctx, repo, svaExtPR(1, func(e *ExternalPullRequest) {
			e.PullRequest.Merged = true
			e.PullRequest.Body = "body"
		}))
		require.NoError(t, err)
		require.Equal(t, enum.PullReqStateMerged, pr.State)
		require.Equal(t, svaBaseSHA, pr.MergeBaseSHA)
		require.Equal(t, svaBaseSHA, *pr.MergeTargetSHA)
		require.Nil(t, pr.MergeSHA)
		require.Equal(t, int64(10), *pr.MergedBy)
		require.Equal(t, enum.MergeMethodMerge, *pr.MergeMethod)
		require.Equal(t, int64(1_700_000_100_000), *pr.Merged)
		require.Equal(t, "body", pr.Description)
	})

	t.Run("closed without timestamps uses the current time", func(t *testing.T) {
		t.Parallel()

		before := time.Now().UnixMilli()
		pr, err := svaNewFixture().state().convertPullReq(ctx, repo, svaExtPR(1, func(e *ExternalPullRequest) {
			e.PullRequest.Closed = true
			e.PullRequest.Created = time.Time{}
			e.PullRequest.Updated = time.Time{}
		}))
		require.NoError(t, err)
		require.GreaterOrEqual(t, pr.Created, before)
		require.Equal(t, pr.Updated, *pr.Closed)
		require.Nil(t, pr.MergeTargetSHA)
	})

	t.Run("overlong description is truncated", func(t *testing.T) {
		t.Parallel()

		pr, err := svaNewFixture().state().convertPullReq(ctx, repo, svaExtPR(1, func(e *ExternalPullRequest) {
			e.PullRequest.Closed = true
			e.PullRequest.Body = string(make([]byte, 100001))
		}))
		require.NoError(t, err)
		require.Len(t, pr.Description, 100000)
	})
}

func TestSvaPrincipalStoreFake(t *testing.T) {
	t.Parallel()

	// Guards the fakes used above: unknown emails map to ErrResourceNotFound, like the real store.
	_, err := (&svaPrincipalStore{}).FindByEmail(context.Background(), "x")
	require.ErrorIs(t, err, gitness_store.ErrResourceNotFound)
}
