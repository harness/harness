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
	"slices"
	"strings"
	"testing"

	"github.com/harness/gitness/app/auth"
	mockstore "github.com/harness/gitness/mocks/store"
	"github.com/harness/gitness/types"
)

func TestSearchSpace_NoRepoPathsSearchesWholeSpace(t *testing.T) {
	t.Parallel()

	spaceFinder := newSpaceFinder(
		map[int64]*types.SpaceCore{
			10: {ID: 10, Path: "alpha"},
		},
		map[string]int64{"alpha": 10},
	)

	repoStore := &mockstore.RepoStore{}
	repoStore.On("MapOfAllRepos", int64(10), false).
		Return(map[int64]string{
			1: "alpha/one",
			2: "alpha/two",
		}, nil).Once()

	searcher := &fakeSearcher{result: types.SearchResult{}}

	ctrl := &Controller{
		searcher:    searcher,
		repoStore:   repoStore,
		repoFinder:  newRepoFinder(nil, nil, nil),
		spaceFinder: spaceFinder,
		repoFilter:  filterAll{},
	}

	_, err := ctrl.SearchSpace(context.Background(), &auth.Session{}, "alpha", SearchSpaceInput{
		Query: "needle",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []int64{1, 2}
	if !slices.Equal(searcher.calledWithRepoIDs, want) {
		t.Fatalf("expected searcher called with %v, got %v", want, searcher.calledWithRepoIDs)
	}
	repoStore.AssertExpectations(t)
}

func TestSearchSpace_NumericRefSearchesWholeSpace(t *testing.T) {
	t.Parallel()

	// A numeric space ref resolves to a path, then the whole space is listed.
	spaceFinder := newSpaceFinder(
		map[int64]*types.SpaceCore{
			20: {ID: 20, Path: "beta"},
		},
		map[string]int64{"beta": 20},
	)

	repoStore := &mockstore.RepoStore{}
	repoStore.On("MapOfAllRepos", int64(20), false).
		Return(map[int64]string{3: "beta/svc"}, nil).Once()

	searcher := &fakeSearcher{result: types.SearchResult{}}

	ctrl := &Controller{
		searcher:    searcher,
		repoStore:   repoStore,
		repoFinder:  newRepoFinder(nil, nil, nil),
		spaceFinder: spaceFinder,
		repoFilter:  filterAll{},
	}

	_, err := ctrl.SearchSpace(context.Background(), &auth.Session{}, "20", SearchSpaceInput{
		Query: "token",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []int64{3}
	if !slices.Equal(searcher.calledWithRepoIDs, want) {
		t.Fatalf("expected searcher called with %v, got %v", want, searcher.calledWithRepoIDs)
	}
	repoStore.AssertExpectations(t)
}

func TestSearchSpace_RecursiveForwardedToStore(t *testing.T) {
	t.Parallel()

	spaceFinder := newSpaceFinder(
		map[int64]*types.SpaceCore{
			30: {ID: 30, Path: "gamma"},
		},
		map[string]int64{"gamma": 30},
	)

	repoStore := &mockstore.RepoStore{}
	// Recursive=true must reach the store as the second argument.
	repoStore.On("MapOfAllRepos", int64(30), true).
		Return(map[int64]string{
			4: "gamma/lib",
			5: "gamma/nested/tool",
		}, nil).Once()

	searcher := &fakeSearcher{result: types.SearchResult{}}

	ctrl := &Controller{
		searcher:    searcher,
		repoStore:   repoStore,
		repoFinder:  newRepoFinder(nil, nil, nil),
		spaceFinder: spaceFinder,
		repoFilter:  filterAll{},
	}

	_, err := ctrl.SearchSpace(context.Background(), &auth.Session{}, "gamma", SearchSpaceInput{
		Query:     "pattern",
		Recursive: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []int64{4, 5}
	if !slices.Equal(searcher.calledWithRepoIDs, want) {
		t.Fatalf("expected searcher called with %v, got %v", want, searcher.calledWithRepoIDs)
	}
	repoStore.AssertExpectations(t)
}

func TestSearchSpace_RepoPathsInSpaceSearched(t *testing.T) {
	t.Parallel()

	// When explicit repo paths are given, the space is only resolved to validate
	// containment; MapOfAllRepos must NOT be called.
	spaceFinder := newSpaceFinder(
		map[int64]*types.SpaceCore{
			40: {ID: 40, Path: "delta"},
		},
		map[string]int64{"delta": 40},
	)

	repoFinder := newRepoFinder(
		map[int64]*types.RepositoryCore{
			6: {ID: 6, ParentID: 40, Path: "delta/worker"},
		},
		map[string]int64{"delta": 40},
		map[types.RepoCacheKey]int64{
			{SpaceID: 40, RepoIdentifier: "worker"}: 6,
		},
	)

	searcher := &fakeSearcher{result: types.SearchResult{}}

	repoStore := &mockstore.RepoStore{} // MapOfAllRepos not expected

	ctrl := &Controller{
		searcher:    searcher,
		repoStore:   repoStore,
		repoFinder:  repoFinder,
		spaceFinder: spaceFinder,
		repoFilter:  filterAll{},
	}

	_, err := ctrl.SearchSpace(context.Background(), &auth.Session{}, "delta", SearchSpaceInput{
		Query:     "phrase",
		RepoPaths: []string{"delta/worker"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []int64{6}
	if !slices.Equal(searcher.calledWithRepoIDs, want) {
		t.Fatalf("expected searcher called with %v, got %v", want, searcher.calledWithRepoIDs)
	}
	repoStore.AssertExpectations(t)
}

func TestSearchSpace_RepoPathOutsideSpaceRejected(t *testing.T) {
	t.Parallel()

	spaceFinder := newSpaceFinder(
		map[int64]*types.SpaceCore{
			50: {ID: 50, Path: "epsilon"},
		},
		map[string]int64{"epsilon": 50},
	)

	ctrl := &Controller{
		searcher:    &fakeSearcher{},
		repoStore:   &mockstore.RepoStore{},
		repoFinder:  newRepoFinder(nil, nil, nil),
		spaceFinder: spaceFinder,
		repoFilter:  filterAll{},
	}

	_, err := ctrl.SearchSpace(context.Background(), &auth.Session{}, "epsilon", SearchSpaceInput{
		Query:     "query",
		RepoPaths: []string{"zeta/stray"},
	})
	if err == nil {
		t.Fatal("expected error for repo outside space")
	}
	if !strings.Contains(err.Error(), "must be in space") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSearchSpace_RepoPathPrefixSiblingRejected(t *testing.T) {
	t.Parallel()

	// "kappa-2/..." shares the "kappa" prefix but is a sibling, not a child — the
	// path-separator boundary must reject it.
	spaceFinder := newSpaceFinder(
		map[int64]*types.SpaceCore{
			60: {ID: 60, Path: "kappa"},
		},
		map[string]int64{"kappa": 60},
	)

	ctrl := &Controller{
		searcher:    &fakeSearcher{},
		repoStore:   &mockstore.RepoStore{},
		repoFinder:  newRepoFinder(nil, nil, nil),
		spaceFinder: spaceFinder,
		repoFilter:  filterAll{},
	}

	_, err := ctrl.SearchSpace(context.Background(), &auth.Session{}, "kappa", SearchSpaceInput{
		Query:     "query",
		RepoPaths: []string{"kappa-2/sibling"},
	})
	if err == nil {
		t.Fatal("expected error for sibling-space repo sharing the name prefix")
	}
	if !strings.Contains(err.Error(), "must be in space") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSearchSpace_RecursiveWithRepoPathsRejected(t *testing.T) {
	t.Parallel()

	spaceFinder := newSpaceFinder(
		map[int64]*types.SpaceCore{
			70: {ID: 70, Path: "lambda"},
		},
		map[string]int64{"lambda": 70},
	)

	ctrl := &Controller{
		searcher:    &fakeSearcher{},
		repoStore:   &mockstore.RepoStore{},
		repoFinder:  newRepoFinder(nil, nil, nil),
		spaceFinder: spaceFinder,
		repoFilter:  filterAll{},
	}

	_, err := ctrl.SearchSpace(context.Background(), &auth.Session{}, "lambda", SearchSpaceInput{
		Query:     "query",
		RepoPaths: []string{"lambda/thing"},
		Recursive: true,
	})
	if err == nil {
		t.Fatal("expected error when combining recursive with explicit repo paths")
	}
	if !strings.Contains(err.Error(), "recursive search") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSearchSpace_UnknownSpaceRefErrors(t *testing.T) {
	t.Parallel()

	ctrl := &Controller{
		searcher:    &fakeSearcher{},
		repoStore:   &mockstore.RepoStore{},
		repoFinder:  newRepoFinder(nil, nil, nil),
		spaceFinder: newSpaceFinder(nil, nil), // FindByRef will fail
		repoFilter:  filterAll{},
	}

	_, err := ctrl.SearchSpace(context.Background(), &auth.Session{}, "phantom", SearchSpaceInput{
		Query: "query",
	})
	if err == nil {
		t.Fatal("expected error for unknown space ref")
	}
	if !strings.Contains(err.Error(), "failed to find space by ref") {
		t.Fatalf("unexpected error: %v", err)
	}
}
