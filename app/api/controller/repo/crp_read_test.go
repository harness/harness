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
	"bytes"
	"context"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/harness/gitness/app/api/request"
	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/git/api"
	mocksse "github.com/harness/gitness/mocks/sse"
	mockstore "github.com/harness/gitness/mocks/store"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// crpTx is a dbtx.Transactor that runs the callback directly.
type crpTx struct{}

func (crpTx) WithTx(ctx context.Context, fn func(ctx context.Context) error, _ ...any) error {
	return fn(ctx)
}

var _ dbtx.Transactor = crpTx{}

// crpPrincipalInfoCache is a store.PrincipalInfoCache backed by a map.
type crpPrincipalInfoCache struct {
	store.PrincipalInfoCache
	infos map[int64]*types.PrincipalInfo
	err   error
}

func (c crpPrincipalInfoCache) Map(context.Context, []int64) (map[int64]*types.PrincipalInfo, error) {
	return c.infos, c.err
}

// crpRepoActivityStore is a store.RepoActivityStore with preconfigured results.
type crpRepoActivityStore struct {
	store.RepoActivityStore
	list     []*types.RepoActivity
	count    int
	listErr  error
	countErr error
}

func (s crpRepoActivityStore) List(context.Context, int64, *types.RepoActivityFilter) ([]*types.RepoActivity, error) {
	return s.list, s.listErr
}

func (s crpRepoActivityStore) Count(context.Context, int64, *types.RepoActivityFilter) (int, error) {
	return s.count, s.countErr
}

func crpUserErrStatus(t *testing.T, err error) int {
	t.Helper()

	var userErr *usererror.Error
	require.ErrorAs(t, err, &userErr)

	return userErr.Status
}

func TestCrpGetCommit(t *testing.T) {
	ctx := context.Background()

	t.Run("maps the commit", func(t *testing.T) {
		e := crpNewEnv(t)
		e.git.On("GetCommit", mock.Anything, mock.MatchedBy(func(p *git.GetCommitParams) bool {
			return p.Revision == "abc"
		})).Return(&git.GetCommitOutput{Commit: git.Commit{
			SHA: crpTestSHA, Title: "title", Message: "body", ParentSHAs: nil,
			Author: git.Signature{Identity: git.Identity{Name: "A", Email: "a@example.com"}},
		}}, nil)

		out, err := e.c.GetCommit(ctx, crpSession(), crpRepoRef, "abc")

		require.NoError(t, err)
		assert.Equal(t, crpTestSHA, out.SHA)
		assert.Equal(t, "title", out.Title)
		assert.Equal(t, "a@example.com", out.Author.Identity.Email)
	})

	t.Run("git failure", func(t *testing.T) {
		e := crpNewEnv(t)
		e.git.On("GetCommit", mock.Anything, mock.Anything).Return(nil, errCrpBoom)

		_, err := e.c.GetCommit(ctx, crpSession(), crpRepoRef, "abc")

		require.ErrorIs(t, err, errCrpBoom)
	})
}

func TestCrpContributorsRegex(t *testing.T) {
	ctx := context.Background()
	infos := map[int64]*types.PrincipalInfo{1: {ID: 1, Email: "a.b@example.com"}}

	tests := []struct {
		name       string
		identifier string
		ids        []int64
		cache      crpPrincipalInfoCache
		want       string
		wantErr    bool
	}{
		{name: "nothing to match", want: ""},
		{name: "identifier only", identifier: "al.ice", want: `al\.ice`},
		{
			name: "ids only", ids: []int64{1}, cache: crpPrincipalInfoCache{infos: infos},
			want: `\<(a\.b@example\.com)\>`,
		},
		{
			name: "identifier and ids", identifier: "alice", ids: []int64{1},
			cache: crpPrincipalInfoCache{infos: infos},
			want:  `alice|\<(a\.b@example\.com)\>`,
		},
		{
			name: "ids resolving to nothing", ids: []int64{1},
			cache: crpPrincipalInfoCache{infos: map[int64]*types.PrincipalInfo{}},
			want:  "",
		},
		{
			name: "cache failure", ids: []int64{1}, cache: crpPrincipalInfoCache{err: errCrpBoom}, wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := crpNewEnv(t)
			e.c.principalInfoCache = tt.cache

			got, err := e.c.contributorsRegex(ctx, tt.identifier, tt.ids)

			if tt.wantErr {
				require.ErrorIs(t, err, errCrpBoom)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCrpListCommits(t *testing.T) {
	ctx := context.Background()
	when := time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)

	t.Run("defaults to the default branch and maps results", func(t *testing.T) {
		e := crpNewEnv(t)
		e.git.On("ListCommits", mock.Anything, mock.MatchedBy(func(p *git.ListCommitsParams) bool {
			return p.GitREF == crpDefaultBranch && p.After == crpBaseRef && p.Page == 2 && p.Limit == 20 &&
				p.Path == "dir" && p.Since == 10 && p.Until == 20 && p.Committer == `al\.ice` &&
				p.Author == "" && p.IncludeStats && p.Regex
		})).Return(&git.ListCommitsOutput{
			Commits: []git.Commit{
				{SHA: crpTestSHA, Title: "one", Committer: git.Signature{When: when}},
				{SHA: crpTestSHAOther, Title: "two"},
			},
			RenameDetails: []*git.RenameDetails{{OldPath: "a", NewPath: "b", CommitShaBefore: crpTestSHA}},
			TotalCommits:  42,
		}, nil)

		out, err := e.c.ListCommits(ctx, crpSession(), crpRepoRef, "", &types.CommitFilter{
			PaginationFilter: types.PaginationFilter{Page: 2, Limit: 20},
			After:            crpBaseRef, Path: "dir", Since: 10, Until: 20, Committer: "al.ice", IncludeStats: true,
		})

		require.NoError(t, err)
		assert.Equal(t, 42, out.TotalCommits)
		require.Len(t, out.Commits, 2)
		assert.Equal(t, "one", out.Commits[0].Title)
		assert.Equal(t, when, out.Commits[0].Committer.When)
		require.Len(t, out.RenameDetails, 1)
		assert.Equal(t, "b", out.RenameDetails[0].NewPath)
	})

	t.Run("explicit ref is used", func(t *testing.T) {
		e := crpNewEnv(t)
		e.git.On("ListCommits", mock.Anything, mock.MatchedBy(func(p *git.ListCommitsParams) bool {
			return p.GitREF == crpFeatureBranch
		})).Return(&git.ListCommitsOutput{}, nil)

		out, err := e.c.ListCommits(ctx, crpSession(), crpRepoRef, crpFeatureBranch, &types.CommitFilter{})

		require.NoError(t, err)
		assert.Empty(t, out.Commits)
	})

	t.Run("two upstream references are rejected", func(t *testing.T) {
		e := crpNewEnv(t)

		_, err := e.c.ListCommits(ctx, crpSession(), crpRepoRef, "upstream:b", &types.CommitFilter{
			After: "upstream:a",
		})

		require.Error(t, err)
		assert.Equal(t, http.StatusBadRequest, crpUserErrStatus(t, err))
	})

	t.Run("committer lookup failure", func(t *testing.T) {
		e := crpNewEnv(t)
		e.c.principalInfoCache = crpPrincipalInfoCache{err: errCrpBoom}

		_, err := e.c.ListCommits(ctx, crpSession(), crpRepoRef, "", &types.CommitFilter{CommitterIDs: []int64{1}})

		require.ErrorIs(t, err, errCrpBoom)
	})

	t.Run("author lookup failure", func(t *testing.T) {
		e := crpNewEnv(t)
		e.c.principalInfoCache = crpPrincipalInfoCache{err: errCrpBoom}

		_, err := e.c.ListCommits(ctx, crpSession(), crpRepoRef, "", &types.CommitFilter{AuthorIDs: []int64{1}})

		require.ErrorIs(t, err, errCrpBoom)
	})

	t.Run("git failure", func(t *testing.T) {
		e := crpNewEnv(t)
		e.git.On("ListCommits", mock.Anything, mock.Anything).Return(nil, errCrpBoom)

		_, err := e.c.ListCommits(ctx, crpSession(), crpRepoRef, "", &types.CommitFilter{})

		require.ErrorIs(t, err, errCrpBoom)
	})

	t.Run("nil rename details are rejected", func(t *testing.T) {
		e := crpNewEnv(t)
		e.git.On("ListCommits", mock.Anything, mock.Anything).Return(&git.ListCommitsOutput{
			RenameDetails: []*git.RenameDetails{nil},
		}, nil)

		_, err := e.c.ListCommits(ctx, crpSession(), crpRepoRef, "", &types.CommitFilter{})

		require.ErrorContains(t, err, "rename details was nil")
	})
}

func TestCrpGetCommitDivergences(t *testing.T) {
	ctx := context.Background()

	t.Run("no requests yield an empty list", func(t *testing.T) {
		e := crpNewEnv(t)

		for _, in := range []*GetCommitDivergencesInput{nil, {}} {
			out, err := e.c.GetCommitDivergences(ctx, crpSession(), crpRepoRef, in)

			require.NoError(t, err)
			assert.Empty(t, out)
		}
		e.git.AssertNotCalled(t, "GetCommitDivergences", mock.Anything, mock.Anything)
	})

	t.Run("too many requests", func(t *testing.T) {
		e := crpNewEnv(t)

		_, err := e.c.GetCommitDivergences(ctx, crpSession(), crpRepoRef, &GetCommitDivergencesInput{
			Requests: make([]CommitDivergenceRequest, request.PerPageMax+1),
		})

		require.ErrorIs(t, err, usererror.ErrRequestTooLarge)
	})

	t.Run("backfills the default branch and maps divergences", func(t *testing.T) {
		e := crpNewEnv(t)
		e.git.On("GetCommitDivergences", mock.Anything, mock.MatchedBy(func(p *git.GetCommitDivergencesParams) bool {
			return p.MaxCount == 10 && len(p.Requests) == 2 &&
				p.Requests[0].From == "a" && p.Requests[0].To == crpDefaultBranch &&
				p.Requests[1].From == "b" && p.Requests[1].To == "c"
		})).Return(&git.GetCommitDivergencesOutput{
			Divergences: []api.CommitDivergence{{Ahead: 1, Behind: 2}, {Ahead: 3, Behind: 4}},
		}, nil)

		out, err := e.c.GetCommitDivergences(ctx, crpSession(), crpRepoRef, &GetCommitDivergencesInput{
			MaxCount: 10,
			Requests: []CommitDivergenceRequest{{From: "a"}, {From: "b", To: "c"}},
		})

		require.NoError(t, err)
		assert.Equal(t, []types.CommitDivergence{{Ahead: 1, Behind: 2}, {Ahead: 3, Behind: 4}}, out)
	})

	t.Run("git failure", func(t *testing.T) {
		e := crpNewEnv(t)
		e.git.On("GetCommitDivergences", mock.Anything, mock.Anything).Return(nil, errCrpBoom)

		_, err := e.c.GetCommitDivergences(ctx, crpSession(), crpRepoRef, &GetCommitDivergencesInput{
			Requests: []CommitDivergenceRequest{{From: "a"}},
		})

		require.ErrorIs(t, err, errCrpBoom)
	})
}

func TestCrpSummary(t *testing.T) {
	ctx := context.Background()

	newEnv := func(t *testing.T) (*crpEnv, *mockstore.RepoStore) {
		t.Helper()
		e := crpNewEnv(t)
		repoStore := &mockstore.RepoStore{}
		e.c.repoStore = repoStore

		return e, repoStore
	}

	t.Run("combines git and db data", func(t *testing.T) {
		e, repoStore := newEnv(t)
		repoStore.On("Find", crpRepoID).Return(&types.Repository{
			ID: crpRepoID, NumOpenPulls: 1, NumClosedPulls: 2, NumMergedPulls: 3, LOC: 400, Size: 500,
		}, nil)
		e.git.On("Summary", mock.Anything, mock.Anything).Return(
			git.SummaryOutput{CommitCount: 10, BranchCount: 4, TagCount: 2}, nil)

		out, err := e.c.Summary(ctx, crpSession(), crpRepoRef)

		require.NoError(t, err)
		assert.Equal(t, 10, out.DefaultBranchCommitCount)
		assert.Equal(t, 4, out.BranchCount)
		assert.Equal(t, 2, out.TagCount)
		assert.Equal(t, types.RepositoryPullReqSummary{OpenCount: 1, ClosedCount: 2, MergedCount: 3}, out.PullReqSummary)
		assert.EqualValues(t, 400, out.LOC)
		assert.EqualValues(t, 500, out.Size)
	})

	t.Run("store failure", func(t *testing.T) {
		e, repoStore := newEnv(t)
		repoStore.On("Find", crpRepoID).Return(nil, errCrpBoom)

		_, err := e.c.Summary(ctx, crpSession(), crpRepoRef)

		require.ErrorIs(t, err, errCrpBoom)
		e.git.AssertNotCalled(t, "Summary", mock.Anything, mock.Anything)
	})

	t.Run("git failure", func(t *testing.T) {
		e, repoStore := newEnv(t)
		repoStore.On("Find", crpRepoID).Return(&types.Repository{ID: crpRepoID}, nil)
		e.git.On("Summary", mock.Anything, mock.Anything).Return(git.SummaryOutput{}, errCrpBoom)

		_, err := e.c.Summary(ctx, crpSession(), crpRepoRef)

		require.ErrorIs(t, err, errCrpBoom)
	})
}

func TestCrpListPaths(t *testing.T) {
	ctx := context.Background()

	t.Run("defaults the ref and returns paths", func(t *testing.T) {
		e := crpNewEnv(t)
		e.git.On("ListPaths", mock.Anything, mock.MatchedBy(func(p *git.ListPathsParams) bool {
			return p.GitREF == crpDefaultBranch && p.IncludeDirectories
		})).Return(&git.ListPathsOutput{Files: []string{"a.txt"}, Directories: []string{"dir"}}, nil)

		out, err := e.c.ListPaths(ctx, crpSession(), crpRepoRef, "", true)

		require.NoError(t, err)
		assert.Equal(t, ListPathsOutput{Files: []string{"a.txt"}, Directories: []string{"dir"}}, out)
	})

	t.Run("git failure", func(t *testing.T) {
		e := crpNewEnv(t)
		e.git.On("ListPaths", mock.Anything, mock.Anything).Return(nil, errCrpBoom)

		_, err := e.c.ListPaths(ctx, crpSession(), crpRepoRef, crpFeatureBranch, false)

		require.ErrorIs(t, err, errCrpBoom)
	})
}

func TestCrpArchive(t *testing.T) {
	ctx := context.Background()

	t.Run("delegates to git", func(t *testing.T) {
		e := crpNewEnv(t)
		var buf bytes.Buffer
		e.git.On("Archive", mock.Anything, mock.MatchedBy(func(p git.ArchiveParams) bool {
			return p.Prefix == "pre" && p.RepoUID == crpGitUID
		}), &buf).Return(nil)

		err := e.c.Archive(ctx, crpSession(), crpRepoRef, api.ArchiveParams{Prefix: "pre"}, &buf)

		require.NoError(t, err)
		e.git.AssertNumberOfCalls(t, "Archive", 1)
	})

	t.Run("git failure", func(t *testing.T) {
		e := crpNewEnv(t)
		e.git.On("Archive", mock.Anything, mock.Anything, mock.Anything).Return(errCrpBoom)

		err := e.c.Archive(ctx, crpSession(), crpRepoRef, api.ArchiveParams{}, &bytes.Buffer{})

		require.ErrorIs(t, err, errCrpBoom)
	})
}

func TestCrpBlame(t *testing.T) {
	ctx := context.Background()

	validation := []struct {
		name     string
		path     string
		from, to int
	}{
		{name: "empty path", path: ""},
		{name: "blank path", path: "   "},
		{name: "inverted range", path: "f.txt", from: 5, to: 2},
	}
	for _, tt := range validation {
		t.Run(tt.name, func(t *testing.T) {
			e := crpNewEnv(t)

			_, err := e.c.Blame(ctx, crpSession(), crpRepoRef, "", tt.path, tt.from, tt.to)

			require.Error(t, err)
			assert.Equal(t, http.StatusBadRequest, crpUserErrStatus(t, err))
		})
	}

	t.Run("streams blame parts", func(t *testing.T) {
		e := crpNewEnv(t)
		parts := make(chan *git.BlamePart, 1)
		parts <- &git.BlamePart{Lines: []string{"line"}}
		close(parts)
		errs := make(chan error)
		e.git.On("Blame", mock.Anything, mock.MatchedBy(func(p *git.BlameParams) bool {
			return p.GitRef == crpDefaultBranch && p.Path == "f.txt" && p.LineFrom == 1 && p.LineTo == 3
		})).Return((<-chan *git.BlamePart)(parts), (<-chan error)(errs))

		stream, err := e.c.Blame(ctx, crpSession(), crpRepoRef, "", " f.txt ", 1, 3)

		require.NoError(t, err)
		part, err := stream.Next()
		require.NoError(t, err)
		assert.Equal(t, []string{"line"}, part.Lines)
		_, err = stream.Next()
		assert.ErrorIs(t, err, io.EOF)
	})
}

func TestCrpDiff(t *testing.T) {
	ctx := context.Background()

	t.Run("RawDiff invalid path", func(t *testing.T) {
		e := crpNewEnv(t)

		err := e.c.RawDiff(ctx, &bytes.Buffer{}, crpSession(), crpRepoRef, "invalid", false)

		require.Error(t, err)
		assert.Equal(t, http.StatusBadRequest, crpUserErrStatus(t, err))
	})

	t.Run("RawDiff delegates the parsed range", func(t *testing.T) {
		e := crpNewEnv(t)
		e.git.On("RawDiff", mock.Anything, mock.Anything, mock.MatchedBy(func(p *git.DiffParams) bool {
			return p.BaseRef == crpBaseRef && p.HeadRef == crpHeadBranch && p.MergeBase && p.IgnoreWhitespace
		})).Return(nil)

		err := e.c.RawDiff(ctx, &bytes.Buffer{}, crpSession(), crpRepoRef, "base...head", true)

		require.NoError(t, err)
	})

	t.Run("RawDiff with two upstream refs", func(t *testing.T) {
		e := crpNewEnv(t)

		err := e.c.RawDiff(ctx, &bytes.Buffer{}, crpSession(), crpRepoRef, "upstream:a..upstream:b", false)

		require.Error(t, err)
	})

	t.Run("CommitDiff", func(t *testing.T) {
		e := crpNewEnv(t)
		e.git.On("CommitDiff", mock.Anything, mock.MatchedBy(func(p *git.GetCommitParams) bool {
			return p.Revision == "abc" && p.IgnoreWhitespace
		}), mock.Anything).Return(nil).Once()
		e.git.On("CommitDiff", mock.Anything, mock.Anything, mock.Anything).Return(errCrpBoom).Once()

		require.NoError(t, e.c.CommitDiff(ctx, crpSession(), crpRepoRef, "abc", true, &bytes.Buffer{}))
		require.ErrorIs(t, e.c.CommitDiff(ctx, crpSession(), crpRepoRef, "abc", true, &bytes.Buffer{}), errCrpBoom)
	})

	t.Run("DiffStats", func(t *testing.T) {
		e := crpNewEnv(t)
		e.git.On("DiffStats", mock.Anything, mock.MatchedBy(func(p *git.DiffParams) bool {
			return p.BaseRef == "a" && p.HeadRef == "b" && !p.MergeBase
		})).Return(git.DiffStatsOutput{Commits: 1, FilesChanged: 2, Additions: 3, Deletions: 4}, nil).Once()
		e.git.On("DiffStats", mock.Anything, mock.Anything).Return(git.DiffStatsOutput{}, errCrpBoom).Once()

		out, err := e.c.DiffStats(ctx, crpSession(), crpRepoRef, "a..b", false)
		require.NoError(t, err)
		assert.Equal(t, types.NewDiffStats(1, 2, 3, 4), out)

		_, err = e.c.DiffStats(ctx, crpSession(), crpRepoRef, "a..b", false)
		require.ErrorIs(t, err, errCrpBoom)

		_, err = e.c.DiffStats(ctx, crpSession(), crpRepoRef, "bad", false)
		require.Error(t, err)

		_, err = e.c.DiffStats(ctx, crpSession(), crpUnknownRepoRef, "a..b", false)
		require.Error(t, err)
	})

	t.Run("DiffStats denied", func(t *testing.T) {
		e := crpNewEnv(t)
		e.setAuthorizer(crpErrAuthorizer{err: errCrpBoom})

		_, err := e.c.DiffStats(ctx, crpSession(), crpRepoRef, "a..b", false)

		require.ErrorIs(t, err, errCrpBoom)
	})

	t.Run("Diff streams file diffs", func(t *testing.T) {
		e := crpNewEnv(t)
		files := make(chan *git.FileDiff, 1)
		files <- &git.FileDiff{Path: "a.txt", Additions: 1}
		close(files)
		e.git.On("Diff", mock.Anything, mock.MatchedBy(func(p *git.DiffParams) bool {
			return p.BaseRef == "a" && p.HeadRef == "b" && p.IncludePatch && p.IgnoreWhitespace
		})).Return((<-chan *git.FileDiff)(files), (<-chan error)(make(chan error)))

		stream, err := e.c.Diff(ctx, crpSession(), crpRepoRef, "a..b", true, true)

		require.NoError(t, err)
		diff, err := stream.Next()
		require.NoError(t, err)
		assert.Equal(t, "a.txt", diff.Path)
		_, err = stream.Next()
		assert.ErrorIs(t, err, io.EOF)
	})

	t.Run("Diff failures", func(t *testing.T) {
		e := crpNewEnv(t)

		_, err := e.c.Diff(ctx, crpSession(), crpUnknownRepoRef, "a..b", false, false)
		require.Error(t, err)

		_, err = e.c.Diff(ctx, crpSession(), crpRepoRef, "bad", false, false)
		require.Error(t, err)

		e.setAuthorizer(crpErrAuthorizer{err: errCrpBoom})
		_, err = e.c.Diff(ctx, crpSession(), crpRepoRef, "a..b", false, false)
		require.ErrorIs(t, err, errCrpBoom)
	})
}

func TestCrpMergeCheck(t *testing.T) {
	ctx := context.Background()

	setup := func(t *testing.T) *crpEnv {
		t.Helper()
		e := crpNewEnv(t)
		e.git.On("ResolveRevision", mock.Anything, mock.MatchedBy(func(p git.ResolveRevisionParams) bool {
			return p.Revision == crpBaseRef
		})).Return(git.ResolveRevisionOutput{SHA: crpTestSHA}, nil)
		e.git.On("ResolveRevision", mock.Anything, mock.MatchedBy(func(p git.ResolveRevisionParams) bool {
			return p.Revision == crpHeadBranch
		})).Return(git.ResolveRevisionOutput{SHA: crpTestSHAOther}, nil)

		return e
	}

	t.Run("invalid path", func(t *testing.T) {
		e := crpNewEnv(t)

		_, err := e.c.MergeCheck(ctx, crpSession(), crpRepoRef, "invalid")

		require.Error(t, err)
	})

	t.Run("mergeable publishes an event", func(t *testing.T) {
		e := setup(t)
		streamer := &mocksse.Streamer{}
		e.c.sseStreamer = streamer
		verify := git.MergeVerifyOutput{CommitCount: 2}
		e.git.On("MergeVerify", mock.Anything, mock.MatchedBy(func(p *git.MergeVerifyParams) bool {
			return p.BaseSHA == crpTestSHA && p.HeadSHA == crpTestSHAOther
		})).Return(verify, nil)
		streamer.On("Publish", e.repo.ParentID, enum.SSETypeBranchMergableUpdated, verify).Return()

		out, err := e.c.MergeCheck(ctx, crpSession(), crpRepoRef, "base..head")

		require.NoError(t, err)
		assert.True(t, out.Mergeable)
		assert.Empty(t, out.ConflictFiles)
		streamer.AssertNumberOfCalls(t, "Publish", 1)
	})

	t.Run("conflicts are reported without an event", func(t *testing.T) {
		e := setup(t)
		e.git.On("MergeVerify", mock.Anything, mock.Anything).Return(
			git.MergeVerifyOutput{ConflictFiles: []string{"a.txt"}}, nil)

		out, err := e.c.MergeCheck(ctx, crpSession(), crpRepoRef, "base..head")

		require.NoError(t, err)
		assert.False(t, out.Mergeable)
		assert.Equal(t, []string{"a.txt"}, out.ConflictFiles)
	})

	t.Run("unrelated histories use the user supplied refs", func(t *testing.T) {
		e := setup(t)
		e.git.On("MergeVerify", mock.Anything, mock.Anything).Return(
			git.MergeVerifyOutput{}, &api.UnrelatedHistoriesError{BaseRef: "sha1", HeadRef: "sha2"})

		_, err := e.c.MergeCheck(ctx, crpSession(), crpRepoRef, "base..head")

		var unrelated *api.UnrelatedHistoriesError
		require.ErrorAs(t, err, &unrelated)
		assert.Equal(t, crpBaseRef, unrelated.BaseRef)
		assert.Equal(t, crpHeadBranch, unrelated.HeadRef)
	})

	t.Run("merge verify failure", func(t *testing.T) {
		e := setup(t)
		e.git.On("MergeVerify", mock.Anything, mock.Anything).Return(git.MergeVerifyOutput{}, errCrpBoom)

		_, err := e.c.MergeCheck(ctx, crpSession(), crpRepoRef, "base..head")

		require.ErrorIs(t, err, errCrpBoom)
	})

	t.Run("base resolve failure", func(t *testing.T) {
		e := crpNewEnv(t)
		e.git.On("ResolveRevision", mock.Anything, mock.Anything).Return(git.ResolveRevisionOutput{}, errCrpBoom)

		_, err := e.c.MergeCheck(ctx, crpSession(), crpRepoRef, "base..head")

		require.ErrorIs(t, err, errCrpBoom)
		assert.ErrorContains(t, err, "base revision")
	})

	t.Run("head resolve failure", func(t *testing.T) {
		e := crpNewEnv(t)
		e.git.On("ResolveRevision", mock.Anything, mock.MatchedBy(func(p git.ResolveRevisionParams) bool {
			return p.Revision == crpBaseRef
		})).Return(git.ResolveRevisionOutput{SHA: crpTestSHA}, nil)
		e.git.On("ResolveRevision", mock.Anything, mock.MatchedBy(func(p git.ResolveRevisionParams) bool {
			return p.Revision == crpHeadBranch
		})).Return(git.ResolveRevisionOutput{}, errCrpBoom)

		_, err := e.c.MergeCheck(ctx, crpSession(), crpRepoRef, "base..head")

		require.ErrorIs(t, err, errCrpBoom)
		assert.ErrorContains(t, err, "head revision")
	})
}

func TestCrpListActivities(t *testing.T) {
	ctx := context.Background()
	activities := []*types.RepoActivity{{RepoID: crpRepoID, Key: "k"}}

	tests := []struct {
		name      string
		store     crpRepoActivityStore
		filter    types.RepoActivityFilter
		wantCount int
		wantErr   error
	}{
		{
			name:      "short first page skips the count query",
			store:     crpRepoActivityStore{list: activities, count: 99},
			filter:    types.RepoActivityFilter{Page: 1, Size: 10},
			wantCount: 1,
		},
		{
			name:      "full page counts",
			store:     crpRepoActivityStore{list: activities, count: 99},
			filter:    types.RepoActivityFilter{Page: 2, Size: 1},
			wantCount: 99,
		},
		{
			name:    "list failure",
			store:   crpRepoActivityStore{listErr: errCrpBoom},
			filter:  types.RepoActivityFilter{Page: 1, Size: 10},
			wantErr: errCrpBoom,
		},
		{
			name:    "count failure",
			store:   crpRepoActivityStore{list: activities, countErr: errCrpBoom},
			filter:  types.RepoActivityFilter{Page: 2, Size: 1},
			wantErr: errCrpBoom,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := crpNewEnv(t)
			e.c.tx = crpTx{}
			e.c.repoActivityStore = tt.store

			out, count, err := e.c.ListActivities(ctx, crpSession(), crpRepoRef, &tt.filter)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantCount, count)
			assert.Equal(t, activities, out)
		})
	}
}

func TestCrpPipelineGenerate(t *testing.T) {
	ctx := context.Background()

	e := crpNewEnv(t)
	e.git.On("GeneratePipeline", mock.Anything, mock.Anything).
		Return(git.GeneratePipelinesOutput{PipelineYAML: []byte("yaml")}, nil).Once()
	e.git.On("GeneratePipeline", mock.Anything, mock.Anything).
		Return(git.GeneratePipelinesOutput{}, errCrpBoom).Once()

	out, err := e.c.PipelineGenerate(ctx, crpSession(), crpRepoRef)
	require.NoError(t, err)
	assert.Equal(t, []byte("yaml"), out)

	_, err = e.c.PipelineGenerate(ctx, crpSession(), crpRepoRef)
	require.ErrorIs(t, err, errCrpBoom)
}

func TestCrpGitInfoRefsAndServicePack(t *testing.T) {
	ctx := context.Background()

	t.Run("info refs", func(t *testing.T) {
		e := crpNewEnv(t)
		e.git.On("GetInfoRefs", mock.Anything, mock.Anything, mock.MatchedBy(func(p *git.InfoRefsParams) bool {
			return p.Service == string(enum.GitServiceTypeUploadPack) && p.GitProtocol == "version=2"
		})).Return(nil).Once()
		e.git.On("GetInfoRefs", mock.Anything, mock.Anything, mock.Anything).Return(errCrpBoom).Once()

		err := e.c.GitInfoRefs(ctx, crpSession(), crpRepoRef, enum.GitServiceTypeUploadPack, "version=2", io.Discard)
		require.NoError(t, err)

		err = e.c.GitInfoRefs(ctx, crpSession(), crpRepoRef, enum.GitServiceTypeUploadPack, "", io.Discard)
		require.ErrorIs(t, err, errCrpBoom)
	})

	t.Run("info refs is allowed while importing", func(t *testing.T) {
		e := crpNewEnv(t)
		e.setRepo(func(r *types.RepositoryCore) { r.State = enum.RepoStateGitImport })
		e.git.On("GetInfoRefs", mock.Anything, mock.Anything, mock.Anything).Return(nil)

		err := e.c.GitInfoRefs(ctx, crpSession(), crpRepoRef, enum.GitServiceTypeUploadPack, "", io.Discard)

		require.NoError(t, err)
	})

	t.Run("upload pack uses read params", func(t *testing.T) {
		e := crpNewEnv(t)
		e.git.On("ServicePack", mock.Anything, mock.MatchedBy(func(p *git.ServicePackParams) bool {
			return p.ReadParams != nil && p.WriteParams == nil && p.ReadParams.RepoUID == crpGitUID
		})).Return(nil)

		err := e.c.GitServicePack(ctx, crpSession(), crpRepoRef,
			api.ServicePackOptions{Service: enum.GitServiceTypeUploadPack})

		require.NoError(t, err)
	})

	t.Run("receive pack uses write params", func(t *testing.T) {
		e := crpNewEnv(t)
		e.git.On("ServicePack", mock.Anything, mock.MatchedBy(func(p *git.ServicePackParams) bool {
			return p.WriteParams != nil && p.ReadParams == nil && p.WriteParams.Actor.Email == crpAliceEmail
		})).Return(nil)

		err := e.c.GitServicePack(ctx, crpSession(), crpRepoRef,
			api.ServicePackOptions{Service: enum.GitServiceTypeReceivePack})

		require.NoError(t, err)
	})

	t.Run("service pack failure", func(t *testing.T) {
		e := crpNewEnv(t)
		e.git.On("ServicePack", mock.Anything, mock.Anything).Return(errCrpBoom)

		err := e.c.GitServicePack(ctx, crpSession(), crpRepoRef,
			api.ServicePackOptions{Service: enum.GitServiceTypeUploadPack})

		require.ErrorIs(t, err, errCrpBoom)
	})

	t.Run("receive pack needs push permission", func(t *testing.T) {
		e := crpNewEnv(t)
		e.setAuthorizer(crpDenyPermAuthorizer{denied: map[enum.Permission]bool{enum.PermissionRepoPush: true}})
		e.git.On("ServicePack", mock.Anything, mock.Anything).Return(nil)

		require.NoError(t, e.c.GitServicePack(ctx, crpSession(), crpRepoRef,
			api.ServicePackOptions{Service: enum.GitServiceTypeUploadPack}))
		require.Error(t, e.c.GitServicePack(ctx, crpSession(), crpRepoRef,
			api.ServicePackOptions{Service: enum.GitServiceTypeReceivePack}))
	})
}

type crpRepoLangStore struct {
	store.RepoLangStore
	stats []*types.RepoLangStat
	err   error
}

func (s crpRepoLangStore) ListByRepoID(context.Context, int64) ([]*types.RepoLangStat, error) {
	return s.stats, s.err
}

func TestCrpGetLanguages(t *testing.T) {
	ctx := context.Background()
	stats := []*types.RepoLangStat{{Language: "Go"}}

	t.Run("returns stats", func(t *testing.T) {
		e := crpNewEnv(t)
		e.c.repoLangStore = crpRepoLangStore{stats: stats}

		out, err := e.c.GetLanguages(ctx, crpSession(), crpRepoRef)

		require.NoError(t, err)
		assert.Equal(t, stats, out)
	})

	t.Run("store failure", func(t *testing.T) {
		e := crpNewEnv(t)
		e.c.repoLangStore = crpRepoLangStore{err: errCrpBoom}

		_, err := e.c.GetLanguages(ctx, crpSession(), crpRepoRef)

		require.ErrorIs(t, err, errCrpBoom)
	})
}
