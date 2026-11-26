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

package request

import (
	"testing"

	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/require"
)

func TestGitQueryOptions(t *testing.T) {
	req := newTestRequest(t, "/?git_ref=refs%2Fheads%2Fdev&include_commit=true&include_stats=true&include_git_stats=true"+
		"&include_pullreqs=true&include_merge_queue=true&include_directories=true&flatten_directories=true"+
		"&max_divergence=12&commit_sha=abc123")

	require.Equal(t, "refs/heads/dev", GetGitRefFromQueryOrDefault(req, "main"))
	require.Equal(t, "main", GetGitRefFromQueryOrDefault(newTestRequest(t, "/"), "main"))
	require.Equal(t, "abc123", GetCommitSHAFromQueryOrDefault(req))
	require.Empty(t, GetCommitSHAFromQueryOrDefault(newTestRequest(t, "/")))

	boolGetters := map[string]func(*testing.T, string, bool) (bool, error){
		"include_commit": func(t *testing.T, q string, d bool) (bool, error) {
			return GetIncludeCommitFromQueryOrDefault(newTestRequest(t, q), d)
		},
		"include_stats": func(t *testing.T, q string, d bool) (bool, error) {
			return GetIncludeStatsFromQueryOrDefault(newTestRequest(t, q), d)
		},
		"include_git_stats": func(t *testing.T, q string, d bool) (bool, error) {
			return GetIncludeGitStatsFromQueryOrDefault(newTestRequest(t, q), d)
		},
		"include_pullreqs": func(t *testing.T, q string, d bool) (bool, error) {
			return GetIncludePullReqsFromQueryOrDefault(newTestRequest(t, q), d)
		},
		"include_merge_queue": func(t *testing.T, q string, d bool) (bool, error) {
			return GetIncludeMergeQueueFromQueryOrDefault(newTestRequest(t, q), d)
		},
		"include_directories": func(t *testing.T, q string, d bool) (bool, error) {
			return GetIncludeDirectoriesFromQueryOrDefault(newTestRequest(t, q), d)
		},
		"flatten_directories": func(t *testing.T, q string, d bool) (bool, error) {
			return GetFlattenDirectoriesFromQueryOrDefault(newTestRequest(t, q), d)
		},
	}
	for param, get := range boolGetters {
		t.Run(param, func(t *testing.T) {
			got, err := get(t, "/?"+param+"=true", false)
			require.NoError(t, err)
			require.True(t, got)

			got, err = get(t, "/", true)
			require.NoError(t, err)
			require.True(t, got, "falls back to the default")

			_, err = get(t, "/?"+param+"=maybe", false)
			requireBadRequest(t, err, "must be a boolean")
		})
	}

	div, err := GetMaxDivergenceFromQueryOrDefault(req, 0)
	require.NoError(t, err)
	require.Equal(t, int64(12), div)
	div, err = GetMaxDivergenceFromQueryOrDefault(newTestRequest(t, "/"), 5)
	require.NoError(t, err)
	require.Equal(t, int64(5), div)
	_, err = GetMaxDivergenceFromQueryOrDefault(newTestRequest(t, "/?max_divergence=-2"), 0)
	requireBadRequest(t, err, "must be a positive integer")
}

func TestGetCommitSHAFromPath(t *testing.T) {
	got, err := GetCommitSHAFromPath(newTestRequest(t, "/", PathParamCommitSHA, "abc"))
	require.NoError(t, err)
	require.Equal(t, "abc", got)

	_, err = GetCommitSHAFromPath(newTestRequest(t, "/"))
	requireBadRequest(t, err, "not found")
}

func TestParseBranchFilter(t *testing.T) {
	t.Run("defaults", func(t *testing.T) {
		got, err := ParseBranchFilter(newTestRequest(t, "/"))
		require.NoError(t, err)
		require.Equal(t, &types.BranchFilter{
			Sort: enum.BranchSortOptionDefault, Order: enum.OrderDefault, Page: 1, Size: PerPageDefault,
		}, got)
	})

	t.Run("all options", func(t *testing.T) {
		got, err := ParseBranchFilter(newTestRequest(t,
			"/?query=feat&sort=date&order=desc&page=2&limit=5&include_commit=true"+
				"&include_checks=true&include_rules=true&include_pullreqs=true&include_merge_queue=true&max_divergence=9"))
		require.NoError(t, err)
		require.Equal(t, "feat", got.Query)
		require.Equal(t, enum.BranchSortOptionDate, got.Sort)
		require.Equal(t, enum.OrderDesc, got.Order)
		require.Equal(t, 2, got.Page)
		require.Equal(t, 5, got.Size)
		require.True(t, got.IncludeCommit)
		require.Equal(t, types.BranchMetadataOptions{
			IncludeChecks: true, IncludeRules: true, IncludePullReqs: true, IncludeMergeQueue: true, MaxDivergence: 9,
		}, got.BranchMetadataOptions)
	})

	t.Run("sort options", func(t *testing.T) {
		require.Equal(t, enum.BranchSortOptionName, ParseSortBranch(newTestRequest(t, "/?sort=name")))
		require.Equal(t, enum.BranchSortOptionDefault, ParseSortBranch(newTestRequest(t, "/?sort=other")))
	})

	t.Run("invalid values", func(t *testing.T) {
		for _, q := range []string{
			"include_commit=maybe", "include_checks=maybe", "include_rules=maybe", "include_pullreqs=maybe",
			"include_merge_queue=maybe", "max_divergence=0",
		} {
			_, err := ParseBranchFilter(newTestRequest(t, "/?"+q))
			require.Error(t, err, q)
		}
	})
}

func TestParseTagFilter(t *testing.T) {
	got := ParseTagFilter(newTestRequest(t, "/?query=v1&sort=name&order=asc&page=3&limit=8"))
	require.Equal(t, &types.TagFilter{
		Query: "v1", Sort: enum.TagSortOptionName, Order: enum.OrderAsc, Page: 3, Size: 8,
	}, got)

	require.Equal(t, enum.TagSortOptionDate, ParseSortTag(newTestRequest(t, "/?sort=date")))
	require.Equal(t, enum.TagSortOptionDefault, ParseSortTag(newTestRequest(t, "/")))
}

func TestParseCommitFilter(t *testing.T) {
	got, err := ParseCommitFilter(newTestRequest(t, "/?after=abc&page=2&limit=4&path=a%2Fb&since=10&until=20"+
		"&committer=jane&committer_id=1&committer_id=2&author=joe&author_id=3&include_stats=true"))
	require.NoError(t, err)
	require.Equal(t, &types.CommitFilter{
		After:            "abc",
		PaginationFilter: types.PaginationFilter{Page: 2, Limit: 4},
		Path:             "a/b",
		Since:            10,
		Until:            20,
		Committer:        "jane",
		CommitterIDs:     []int64{1, 2},
		Author:           "joe",
		AuthorIDs:        []int64{3},
		IncludeStats:     true,
	}, got)

	got, err = ParseCommitFilter(newTestRequest(t, "/"))
	require.NoError(t, err)
	require.Equal(t, 1, got.Page)
	require.Equal(t, PerPageDefault, got.Limit)

	for _, q := range []string{"since=0", "until=x", "include_stats=maybe", "committer_id=x", "author_id=-1"} {
		_, err := ParseCommitFilter(newTestRequest(t, "/?"+q))
		require.Error(t, err, q)
	}
}

func TestGitProtocolAndService(t *testing.T) {
	req := newTestRequest(t, "/?service=git-upload-pack")
	req.Header.Set(HeaderParamGitProtocol, "version=2")
	require.Equal(t, "version=2", GetGitProtocolFromHeadersOrDefault(req, ""))
	require.Equal(t, "dflt", GetGitProtocolFromHeadersOrDefault(newTestRequest(t, "/"), "dflt"))

	svc, err := GetGitServiceTypeFromQuery(req)
	require.NoError(t, err)
	require.Equal(t, enum.GitServiceTypeUploadPack, svc)

	svc, err = GetGitServiceTypeFromQuery(newTestRequest(t, "/?service=git-receive-pack"))
	require.NoError(t, err)
	require.Equal(t, enum.GitServiceTypeReceivePack, svc)

	_, err = GetGitServiceTypeFromQuery(newTestRequest(t, "/"))
	require.Error(t, err)
	_, err = GetGitServiceTypeFromQuery(newTestRequest(t, "/?service=upload-pack"))
	requireBadRequest(t, err, "Not a git service type")
	_, err = GetGitServiceTypeFromQuery(newTestRequest(t, "/?service=git-bogus"))
	require.Error(t, err)
}

func TestRepoRequestHelpers(t *testing.T) {
	got, err := GetRepoRefFromPath(newTestRequest(t, "/", PathParamRepoRef, "space%2Frepo"))
	require.NoError(t, err)
	require.Equal(t, "space/repo", got)
	_, err = GetRepoRefFromPath(newTestRequest(t, "/"))
	requireBadRequest(t, err, "not found")

	require.Equal(t, []string{"a/b", "c"}, GetRepoPathsFromQuery(newTestRequest(t, "/?repo_path=a%2Fb&repo_path=c")))
	require.Empty(t, GetRepoPathsFromQuery(newTestRequest(t, "/")))

	require.Equal(t, enum.RepoAttrCreated, ParseSortRepo(newTestRequest(t, "/?sort=created")))
	require.Equal(t, enum.RepoAttrNone, ParseSortRepo(newTestRequest(t, "/")))

	fav, err := ParseOnlyFavoritesFromQuery(newTestRequest(t, "/?only_favorites=true"))
	require.NoError(t, err)
	require.True(t, fav)
	_, err = ParseOnlyFavoritesFromQuery(newTestRequest(t, "/?only_favorites=x"))
	requireBadRequest(t, err, "must be a boolean")
}

func TestParseRepoTypes(t *testing.T) {
	require.Nil(t, parseRepoTypes(newTestRequest(t, "/")), "absent parameter means every type")
	require.Nil(t, parseRepoTypes(newTestRequest(t, "/?repo_type=bogus")))
	require.Equal(t, []enum.RepoType{enum.RepoTypeLinked},
		parseRepoTypes(newTestRequest(t, "/?repo_type=linked&repo_type=linked")))
	require.ElementsMatch(t, []enum.RepoType{enum.RepoTypeNormal, enum.RepoTypeLinked},
		parseRepoTypes(newTestRequest(t, "/?repo_type=&repo_type=linked")))
}

func TestParseRepoFilter(t *testing.T) {
	session := &auth.Session{Principal: types.Principal{ID: 42}}

	t.Run("defaults", func(t *testing.T) {
		got, err := ParseRepoFilter(newTestRequest(t, "/"), session)
		require.NoError(t, err)
		require.Equal(t, enum.OrderAsc, got.Order, "the default order is ascending")
		require.Equal(t, 1, got.Page)
		require.Equal(t, PerPageDefault, got.Size)
		require.False(t, got.Recursive)
		require.Nil(t, got.DeletedAt)
		require.Nil(t, got.DeletedBeforeOrAt)
		require.Nil(t, got.OnlyFavoritesFor)
	})

	t.Run("all options", func(t *testing.T) {
		got, err := ParseRepoFilter(newTestRequest(t, "/?query=x&order=desc&sort=updated&page=2&limit=3&recursive=true"+
			"&deleted_at=100&deleted_before_or_at=200&only_favorites=true&repo_type=linked&tag=env:prod"), session)
		require.NoError(t, err)
		require.Equal(t, "x", got.Query)
		require.Equal(t, enum.OrderDesc, got.Order)
		require.Equal(t, enum.RepoAttrUpdated, got.Sort)
		require.True(t, got.Recursive)
		require.Equal(t, int64(100), *got.DeletedAt)
		require.Equal(t, int64(200), *got.DeletedBeforeOrAt)
		require.Equal(t, int64(42), *got.OnlyFavoritesFor)
		require.Equal(t, []enum.RepoType{enum.RepoTypeLinked}, got.Types)
		require.Equal(t, map[string][]string{"env": {"prod"}}, got.Tags)
	})

	t.Run("invalid values", func(t *testing.T) {
		for _, q := range []string{"recursive=x", "deleted_before_or_at=0", "deleted_at=x", "only_favorites=x"} {
			_, err := ParseRepoFilter(newTestRequest(t, "/?"+q), session)
			require.Error(t, err, q)
		}
	})
}
