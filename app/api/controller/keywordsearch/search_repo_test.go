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

package keywordsearch

import (
	"context"
	"strings"
	"testing"

	"github.com/harness/gitness/app/auth"
	mockstore "github.com/harness/gitness/mocks/store"
	"github.com/harness/gitness/types"
)

func TestSearchRepo_PathRefHappyPath(t *testing.T) {
	t.Parallel()

	// repoFinder needs the full path→ID→core wiring: SearchRepo forwards the ref and
	// the underlying Search resolves it to the canonical path.
	repoFinder := newRepoFinder(
		map[int64]*types.RepositoryCore{
			42: {ID: 42, ParentID: 7, Path: "alpha/svc"},
		},
		map[string]int64{"alpha": 7},
		map[types.RepoCacheKey]int64{
			{SpaceID: 7, RepoIdentifier: "svc"}: 42,
		},
	)

	searcher := &fakeSearcher{result: types.SearchResult{
		FileMatches: []types.FileMatch{{RepoID: 42, FileName: "main.go"}},
	}}

	ctrl := &Controller{
		searcher:    searcher,
		repoStore:   &mockstore.RepoStore{}, // never called: no space paths
		repoFinder:  repoFinder,
		spaceFinder: newSpaceFinder(nil, nil),
		repoFilter:  filterAll{},
	}

	got, err := ctrl.SearchRepo(context.Background(), &auth.Session{}, "alpha/svc", SearchRepoInput{
		Query: "needle",
		Limit: 10,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(searcher.calledWithRepoIDs) != 1 || searcher.calledWithRepoIDs[0] != 42 {
		t.Fatalf("expected searcher called with [42], got: %v", searcher.calledWithRepoIDs)
	}
	if len(got.FileMatches) != 1 || got.FileMatches[0].RepoPath != "alpha/svc" {
		t.Fatalf("expected single match with canonical repo path, got: %+v", got.FileMatches)
	}
}

func TestSearchRepo_NumericRefResolved(t *testing.T) {
	t.Parallel()

	// A numeric ref must resolve through the repo (not space) finder.
	repoFinder := newRepoFinder(
		map[int64]*types.RepositoryCore{
			55: {ID: 55, ParentID: 8, Path: "beta/api"},
		},
		map[string]int64{"beta": 8},
		map[types.RepoCacheKey]int64{
			{SpaceID: 8, RepoIdentifier: "api"}: 55,
		},
	)

	searcher := &fakeSearcher{result: types.SearchResult{}}

	ctrl := &Controller{
		searcher:    searcher,
		repoStore:   &mockstore.RepoStore{},
		repoFinder:  repoFinder,
		spaceFinder: newSpaceFinder(nil, nil),
		repoFilter:  filterAll{},
	}

	_, err := ctrl.SearchRepo(context.Background(), &auth.Session{}, "55", SearchRepoInput{
		Query: "token",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(searcher.calledWithRepoIDs) != 1 || searcher.calledWithRepoIDs[0] != 55 {
		t.Fatalf("expected searcher called with [55], got: %v", searcher.calledWithRepoIDs)
	}
}

func TestSearchRepo_UnknownRefErrors(t *testing.T) {
	t.Parallel()

	ctrl := &Controller{
		searcher:    &fakeSearcher{},
		repoStore:   &mockstore.RepoStore{},
		repoFinder:  newRepoFinder(nil, nil, nil), // FindByRef will fail
		spaceFinder: newSpaceFinder(nil, nil),
		repoFilter:  filterAll{},
	}

	_, err := ctrl.SearchRepo(context.Background(), &auth.Session{}, "ghost/gone", SearchRepoInput{
		Query: "spark",
	})
	if err == nil {
		t.Fatal("expected error for unknown repo ref")
	}
	if !strings.Contains(err.Error(), "failed to find repo by ref") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSearchRepo_RegexAndLimitForwarded(t *testing.T) {
	t.Parallel()

	repoFinder := newRepoFinder(
		map[int64]*types.RepositoryCore{
			60: {ID: 60, ParentID: 9, Path: "gamma/web"},
		},
		map[string]int64{"gamma": 9},
		map[types.RepoCacheKey]int64{
			{SpaceID: 9, RepoIdentifier: "web"}: 60,
		},
	)

	searcher := &recordingSearcher{}

	ctrl := &Controller{
		searcher:    searcher,
		repoStore:   &mockstore.RepoStore{},
		repoFinder:  repoFinder,
		spaceFinder: newSpaceFinder(nil, nil),
		repoFilter:  filterAll{},
	}

	_, err := ctrl.SearchRepo(context.Background(), &auth.Session{}, "gamma/web", SearchRepoInput{
		Query: "match",
		Limit: 25,
		Regex: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if searcher.query != "match" || searcher.maxResultCount != 25 || !searcher.enableRegex {
		t.Fatalf("expected query/limit/regex forwarded, got query=%q limit=%d regex=%v",
			searcher.query, searcher.maxResultCount, searcher.enableRegex)
	}
}

// recordingSearcher captures every scalar argument passed to Search so tests can
// assert that controller inputs are forwarded verbatim.
type recordingSearcher struct {
	repoIDs        []int64
	query          string
	enableRegex    bool
	maxResultCount int
}

func (s *recordingSearcher) Search(
	_ context.Context,
	repoIDs []int64,
	query string,
	enableRegex bool,
	maxResultCount int,
) (types.SearchResult, error) {
	s.repoIDs = append([]int64(nil), repoIDs...)
	s.query = query
	s.enableRegex = enableRegex
	s.maxResultCount = maxResultCount
	return types.SearchResult{}, nil
}
