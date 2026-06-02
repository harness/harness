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

package pullreq

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"testing"

	"github.com/harness/gitness/git"
	gitenum "github.com/harness/gitness/git/enum"
	"github.com/harness/gitness/git/sha"
	"github.com/harness/gitness/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPullReqViewCreateInput_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		in      PullReqViewCreateInput
		errPart string
	}{
		{
			name: "empty title",
			in: PullReqViewCreateInput{Groups: []PullReqViewCreateInputGroup{{
				Title: "   ",
				Files: []string{"a.txt"},
			}}},
			errPart: "group title can't be empty",
		},
		{
			name: "group without files",
			in: PullReqViewCreateInput{Groups: []PullReqViewCreateInputGroup{{
				Title: "Backend",
				Files: nil,
			}}},
			errPart: "must contain at least one file",
		},
		{
			name: "duplicate titles",
			in: PullReqViewCreateInput{Groups: []PullReqViewCreateInputGroup{
				{Title: "Backend", Files: []string{"a.txt"}},
				{Title: "Backend", Files: []string{"b.txt"}},
			}},
			errPart: "duplicate group title",
		},
		{
			name: "valid input",
			in: PullReqViewCreateInput{Groups: []PullReqViewCreateInputGroup{
				{Title: "Backend", Files: []string{"a.txt"}},
				{Title: "Frontend", Files: []string{"b.txt"}},
			}},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			err := test.in.Validate()
			if test.errPart == "" {
				require.NoError(t, err)
				return
			}

			require.Error(t, err)
			assert.Contains(t, err.Error(), test.errPart)
		})
	}
}

func TestCollectRequestedGroupPaths(t *testing.T) {
	t.Parallel()

	got := collectRequestedGroupPaths([]PullReqViewCreateInputGroup{
		{Title: "Backend", Files: []string{"b.txt", "a.txt", "b.txt"}},
		{Title: "Frontend", Files: []string{"c.txt", "a.txt"}},
	})

	assert.Equal(t, []string{"a.txt", "b.txt", "c.txt"}, got)
}

func TestBuildFileGroupFiles(t *testing.T) {
	t.Parallel()

	t.Run("deduplicates paths and keeps first occurrence order", func(t *testing.T) {
		t.Parallel()

		files, err := buildFileGroupFiles(
			[]string{"b.txt", "a.txt", "b.txt"},
			map[string]fileGroupPathSHAs{
				"a.txt": {oldSHA: "old-a", newSHA: "new-a"},
				"b.txt": {oldSHA: "old-b", newSHA: "new-b"},
			},
		)

		require.NoError(t, err)
		require.Len(t, files, 2)
		assert.Equal(t, "b.txt", files[0].Path)
		assert.Equal(t, "old-b", files[0].OldSHA)
		assert.Equal(t, "new-b", files[0].NewSHA)
		assert.Equal(t, "a.txt", files[1].Path)
		assert.Equal(t, "old-a", files[1].OldSHA)
		assert.Equal(t, "new-a", files[1].NewSHA)
	})

	t.Run("rejects empty path", func(t *testing.T) {
		t.Parallel()

		_, err := buildFileGroupFiles(
			[]string{""},
			map[string]fileGroupPathSHAs{"a.txt": {oldSHA: "o", newSHA: "n"}},
		)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "files.path is required")
	})

	t.Run("rejects path not in diff", func(t *testing.T) {
		t.Parallel()

		_, err := buildFileGroupFiles(
			[]string{"missing.txt"},
			map[string]fileGroupPathSHAs{"a.txt": {oldSHA: "o", newSHA: "n"}},
		)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "is not part of the pull request diff")
	})
}

func TestResolveCurrentPRSHAs(t *testing.T) {
	t.Parallel()

	ctrl := &Controller{}
	pr := &types.PullReq{SourceSHA: "pr-source", MergeBaseSHA: "pr-merge-base"}

	t.Run("uses request shas when both are set", func(t *testing.T) {
		t.Parallel()

		sourceSHA, mergeBaseSHA, err := ctrl.resolveCurrentPRSHAs(
			context.Background(),
			&types.RepositoryCore{},
			pr,
			"req-source",
			"req-merge-base",
		)

		require.NoError(t, err)
		assert.Equal(t, "req-source", sourceSHA)
		assert.Equal(t, "req-merge-base", mergeBaseSHA)
	})

	t.Run("falls back to pull request shas", func(t *testing.T) {
		t.Parallel()

		sourceSHA, mergeBaseSHA, err := ctrl.resolveCurrentPRSHAs(
			context.Background(),
			&types.RepositoryCore{},
			pr,
			"",
			"",
		)

		require.NoError(t, err)
		assert.Equal(t, "pr-source", sourceSHA)
		assert.Equal(t, "pr-merge-base", mergeBaseSHA)
	})

	t.Run("resolves missing values from git", func(t *testing.T) {
		repo := &types.RepositoryCore{GitUID: "repo-git-uid"}
		prWithNoSHAs := &types.PullReq{Number: 7, TargetBranch: "main"}

		gitStub := &gitInterfaceStub{}
		gitStub.getRefFn = func(_ context.Context, params git.GetRefParams) (git.GetRefResponse, error) {
			require.Equal(t, "repo-git-uid", params.RepoUID)

			switch params.Type {
			case gitenum.RefTypePullReqHead:
				require.Equal(t, strconv.FormatInt(prWithNoSHAs.Number, 10), params.Name)
				return git.GetRefResponse{SHA: sha.Must("1111111111111111111111111111111111111111")}, nil
			case gitenum.RefTypeBranch:
				require.Equal(t, "main", params.Name)
				return git.GetRefResponse{SHA: sha.Must("2222222222222222222222222222222222222222")}, nil
			case gitenum.RefTypeRaw, gitenum.RefTypeTag,
				gitenum.RefTypePullReqMerge, gitenum.RefTypePullReqMergeQueue:
				t.Fatalf("unexpected ref type: %v", params.Type)
			}

			return git.GetRefResponse{}, nil
		}
		gitStub.mergeBaseFn = func(_ context.Context, params git.MergeBaseParams) (git.MergeBaseOutput, error) {
			require.Equal(t, "repo-git-uid", params.RepoUID)
			require.Equal(t, "1111111111111111111111111111111111111111", params.Ref1)
			require.Equal(t, "2222222222222222222222222222222222222222", params.Ref2)

			return git.MergeBaseOutput{MergeBaseSHA: sha.Must("3333333333333333333333333333333333333333")}, nil
		}

		ctrlWithGit := &Controller{git: gitStub}

		sourceSHA, mergeBaseSHA, err := ctrlWithGit.resolveCurrentPRSHAs(
			context.Background(),
			repo,
			prWithNoSHAs,
			"",
			"",
		)

		require.NoError(t, err)
		assert.Equal(t, "1111111111111111111111111111111111111111", sourceSHA)
		assert.Equal(t, "3333333333333333333333333333333333333333", mergeBaseSHA)
	})

	t.Run("returns wrapped error when git fallback fails", func(t *testing.T) {
		repo := &types.RepositoryCore{GitUID: "repo-git-uid"}
		prWithNoSHAs := &types.PullReq{Number: 7, TargetBranch: "main"}

		gitStub := &gitInterfaceStub{}
		gitStub.getRefFn = func(_ context.Context, params git.GetRefParams) (git.GetRefResponse, error) {
			if params.Type == gitenum.RefTypePullReqHead {
				return git.GetRefResponse{}, errors.New("head ref failed")
			}
			return git.GetRefResponse{SHA: sha.Must("2222222222222222222222222222222222222222")}, nil
		}

		ctrlWithGit := &Controller{git: gitStub}

		_, _, err := ctrlWithGit.resolveCurrentPRSHAs(
			context.Background(),
			repo,
			prWithNoSHAs,
			"",
			"",
		)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to resolve pull request SHAs from git")
		assert.Contains(t, err.Error(), "failed to resolve pull request head reference")
	})
}

func TestResolveFileGroupPathSHAsFromPRDiff_EmptyPaths(t *testing.T) {
	t.Parallel()

	ctrl := &Controller{}

	got, err := ctrl.resolveFileGroupPathSHAsFromPRDiff(
		context.Background(),
		gitRepositoryStub{},
		"source-sha",
		"merge-base-sha",
		nil,
	)

	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestPullReqViewCreate_InvalidInput(t *testing.T) {
	t.Parallel()

	ctrl := &Controller{}

	err := ctrl.PullReqViewCreate(
		context.Background(),
		testSession(),
		"repo",
		1,
		&PullReqViewCreateInput{Groups: []PullReqViewCreateInputGroup{{
			Title: "",
			Files: []string{"a.txt"},
		}}},
	)

	require.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "group title can't be empty"))
}

type gitRepositoryStub struct{}

func (gitRepositoryStub) GetGitUID() string { return "" }

type gitInterfaceStub struct {
	git.Interface

	getRefFn    func(ctx context.Context, params git.GetRefParams) (git.GetRefResponse, error)
	mergeBaseFn func(ctx context.Context, params git.MergeBaseParams) (git.MergeBaseOutput, error)
}

func (s *gitInterfaceStub) GetRef(ctx context.Context, params git.GetRefParams) (git.GetRefResponse, error) {
	if s.getRefFn == nil {
		return git.GetRefResponse{}, errors.New("getRefFn not configured")
	}

	return s.getRefFn(ctx, params)
}

func (s *gitInterfaceStub) MergeBase(ctx context.Context, params git.MergeBaseParams) (git.MergeBaseOutput, error) {
	if s.mergeBaseFn == nil {
		return git.MergeBaseOutput{}, errors.New("mergeBaseFn not configured")
	}

	return s.mergeBaseFn(ctx, params)
}
