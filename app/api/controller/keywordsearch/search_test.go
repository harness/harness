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
	"errors"
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/services/refcache"
	storecache "github.com/harness/gitness/app/store/cache"
	mockstore "github.com/harness/gitness/mocks/store"
	"github.com/harness/gitness/types"
)

// ---------- refcache stubs (cache.Cache implementations) ----------

type spaceIDCacheStub struct {
	spaces map[int64]*types.SpaceCore
}

func (c *spaceIDCacheStub) Stats() (int64, int64)            { return 0, 0 }
func (c *spaceIDCacheStub) Evict(_ context.Context, _ int64) {}
func (c *spaceIDCacheStub) Get(_ context.Context, id int64) (*types.SpaceCore, error) {
	if s, ok := c.spaces[id]; ok {
		return s, nil
	}
	return nil, fmt.Errorf("space %d not found", id)
}

type spacePathCacheStub struct {
	pathToID map[string]int64
}

func (c *spacePathCacheStub) Stats() (int64, int64)             { return 0, 0 }
func (c *spacePathCacheStub) Evict(_ context.Context, _ string) {}
func (c *spacePathCacheStub) Get(_ context.Context, path string) (*types.SpacePath, error) {
	id, ok := c.pathToID[path]
	if !ok {
		return nil, fmt.Errorf("space path %q not found", path)
	}
	return &types.SpacePath{Value: path, IsPrimary: true, SpaceID: id}, nil
}

type repoIDCacheStub struct {
	repos map[int64]*types.RepositoryCore
}

func (c *repoIDCacheStub) Stats() (int64, int64)            { return 0, 0 }
func (c *repoIDCacheStub) Evict(_ context.Context, _ int64) {}
func (c *repoIDCacheStub) Get(_ context.Context, id int64) (*types.RepositoryCore, error) {
	if r, ok := c.repos[id]; ok {
		return r, nil
	}
	return nil, fmt.Errorf("repo %d not found", id)
}

type repoRefCacheStub struct {
	byKey map[types.RepoCacheKey]int64
}

func (c *repoRefCacheStub) Stats() (int64, int64)                         { return 0, 0 }
func (c *repoRefCacheStub) Evict(_ context.Context, _ types.RepoCacheKey) {}
func (c *repoRefCacheStub) Get(_ context.Context, key types.RepoCacheKey) (int64, error) {
	if id, ok := c.byKey[key]; ok {
		return id, nil
	}
	return 0, fmt.Errorf("repo ref %+v not found", key)
}

// ---------- test doubles for injected collaborators ----------

// fakeSearcher records the repoIDs it was called with and returns canned results.
type fakeSearcher struct {
	calledWithRepoIDs []int64
	result            types.SearchResult
	err               error
}

func (s *fakeSearcher) Search(
	_ context.Context,
	repoIDs []int64,
	_ string,
	_ bool,
	_ int,
) (types.SearchResult, error) {
	s.calledWithRepoIDs = append([]int64(nil), repoIDs...)
	sort.Slice(s.calledWithRepoIDs, func(i, j int) bool {
		return s.calledWithRepoIDs[i] < s.calledWithRepoIDs[j]
	})
	if s.err != nil {
		return types.SearchResult{}, s.err
	}
	return s.result, nil
}

// filterAll returns every repoID in the map, sorted ascending.
type filterAll struct{}

func (filterAll) ViewAccessFilter(
	_ context.Context, _ *auth.Session, repoMap map[int64]string,
) ([]int64, error) {
	ids := make([]int64, 0, len(repoMap))
	for id := range repoMap {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids, nil
}

// filterNone denies every repo.
type filterNone struct{}

func (filterNone) ViewAccessFilter(
	_ context.Context, _ *auth.Session, _ map[int64]string,
) ([]int64, error) {
	return nil, nil
}

type filterOnly struct{ allowed map[int64]bool }

func (f filterOnly) ViewAccessFilter(
	_ context.Context, _ *auth.Session, repoMap map[int64]string,
) ([]int64, error) {
	ids := make([]int64, 0, len(repoMap))
	for id := range repoMap {
		if f.allowed[id] {
			ids = append(ids, id)
		}
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids, nil
}

type filterErr struct{ err error }

func (f filterErr) ViewAccessFilter(
	_ context.Context, _ *auth.Session, _ map[int64]string,
) ([]int64, error) {
	return nil, f.err
}

// ---------- helpers ----------

func newRepoFinder(
	repos map[int64]*types.RepositoryCore,
	pathToID map[string]int64,
	refCache map[types.RepoCacheKey]int64,
) refcache.RepoFinder {
	return refcache.NewRepoFinder(
		nil,
		&spacePathCacheStub{pathToID: pathToID},
		&repoIDCacheStub{repos: repos},
		&repoRefCacheStub{byKey: refCache},
		storecache.Evictor[*types.RepositoryCore]{},
	)
}

func newSpaceFinder(
	spaces map[int64]*types.SpaceCore,
	pathToID map[string]int64,
) refcache.SpaceFinder {
	return refcache.NewSpaceFinder(
		&spaceIDCacheStub{spaces: spaces},
		&spacePathCacheStub{pathToID: pathToID},
		nil,
		storecache.Evictor[*types.SpaceCore]{},
	)
}

// ---------- tests ----------

func TestSearch_EmptyQueryRejected(t *testing.T) {
	t.Parallel()

	ctrl := &Controller{}
	_, err := ctrl.Search(context.Background(), &auth.Session{}, types.SearchInput{
		RepoPaths: []string{"space/repo"},
	})
	if err == nil {
		t.Fatal("expected error for empty query")
	}
	if !strings.Contains(err.Error(), "Query cannot be empty") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSearch_NeitherRepoNorSpacePathsRejected(t *testing.T) {
	t.Parallel()

	ctrl := &Controller{}
	_, err := ctrl.Search(context.Background(), &auth.Session{}, types.SearchInput{
		Query: "foo",
	})
	if err == nil {
		t.Fatal("expected error when neither repo nor space paths are set")
	}
	if !strings.Contains(err.Error(), "either repo paths or space paths need to be set") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSearch_RepoPathHappyPath(t *testing.T) {
	t.Parallel()

	// A single repo path resolves via repoFinder, no space traversal.
	repoFinder := newRepoFinder(
		map[int64]*types.RepositoryCore{
			42: {ID: 42, ParentID: 7, Path: "space/repo-a", Identifier: "repo-a"},
		},
		map[string]int64{"space": 7},
		map[types.RepoCacheKey]int64{
			{SpaceID: 7, RepoIdentifier: "repo-a"}: 42,
		},
	)

	searcher := &fakeSearcher{result: types.SearchResult{
		FileMatches: []types.FileMatch{{RepoID: 42, FileName: "main.go"}},
		Stats:       types.SearchStats{TotalFiles: 1, TotalMatches: 1},
	}}

	ctrl := &Controller{
		searcher:    searcher,
		repoStore:   &mockstore.RepoStore{}, // never called: no space paths
		repoFinder:  repoFinder,
		spaceFinder: newSpaceFinder(nil, nil),
		repoFilter:  filterAll{},
	}

	got, err := ctrl.Search(context.Background(), &auth.Session{}, types.SearchInput{
		Query:     "foo",
		RepoPaths: []string{"space/repo-a"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got.FileMatches) != 1 {
		t.Fatalf("expected 1 file match, got %d", len(got.FileMatches))
	}
	if got.FileMatches[0].RepoPath != "space/repo-a" {
		t.Fatalf("expected RepoPath to be set from canonical repo path, got %q", got.FileMatches[0].RepoPath)
	}
	if len(searcher.calledWithRepoIDs) != 1 || searcher.calledWithRepoIDs[0] != 42 {
		t.Fatalf("expected searcher to be called with [42], got: %v", searcher.calledWithRepoIDs)
	}
}

func TestSearch_RepoFinderErrorPropagates(t *testing.T) {
	t.Parallel()

	repoFinder := newRepoFinder(nil, nil, nil) // FindByRef will fail

	ctrl := &Controller{
		searcher:    &fakeSearcher{},
		repoStore:   &mockstore.RepoStore{},
		repoFinder:  repoFinder,
		spaceFinder: newSpaceFinder(nil, nil),
		repoFilter:  filterAll{},
	}

	_, err := ctrl.Search(context.Background(), &auth.Session{}, types.SearchInput{
		Query:     "foo",
		RepoPaths: []string{"missing/repo"},
	})
	if err == nil {
		t.Fatal("expected error from repo finder")
	}
	if !strings.Contains(err.Error(), "failed to find repo by path") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSearch_SpacePathNonRecursive(t *testing.T) {
	t.Parallel()

	spaceFinder := newSpaceFinder(
		map[int64]*types.SpaceCore{
			10: {ID: 10, Path: "space", Identifier: "space"},
		},
		map[string]int64{"space": 10},
	)

	repoStore := &mockstore.RepoStore{}
	// Only the direct space (10) is queried; Recursive=false → not expanded.
	repoStore.On("MapOfAllRepos", int64(10), false).
		Return(map[int64]string{
			1: "space/repo-1",
			2: "space/repo-2",
		}, nil).Once()

	searcher := &fakeSearcher{result: types.SearchResult{
		FileMatches: []types.FileMatch{
			{RepoID: 1, FileName: "a.go"},
			{RepoID: 2, FileName: "b.go"},
		},
	}}

	ctrl := &Controller{
		searcher:    searcher,
		repoStore:   repoStore,
		repoFinder:  newRepoFinder(nil, nil, nil),
		spaceFinder: spaceFinder,
		repoFilter:  filterAll{},
	}

	got, err := ctrl.Search(context.Background(), &auth.Session{}, types.SearchInput{
		Query:      "foo",
		SpacePaths: []string{"space"},
		Recursive:  false,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got.FileMatches) != 2 {
		t.Fatalf("expected 2 matches, got %d", len(got.FileMatches))
	}

	repoPathsByID := map[int64]string{}
	for _, m := range got.FileMatches {
		repoPathsByID[m.RepoID] = m.RepoPath
	}
	if repoPathsByID[1] != "space/repo-1" || repoPathsByID[2] != "space/repo-2" {
		t.Fatalf("unexpected repo paths in result: %v", repoPathsByID)
	}
	repoStore.AssertExpectations(t)
}

func TestSearch_SpacePathRecursiveExpandsDescendants(t *testing.T) {
	t.Parallel()

	spaceFinder := newSpaceFinder(
		map[int64]*types.SpaceCore{
			10: {ID: 10, Path: "space", Identifier: "space"},
		},
		map[string]int64{"space": 10},
	)

	// Recursive=true → the controller delegates descendant expansion to the store
	// by calling MapOfAllRepos once for the space with recursive=true.
	repoStore := &mockstore.RepoStore{}
	repoStore.On("MapOfAllRepos", int64(10), true).
		Return(map[int64]string{
			1: "space/repo-a",
			2: "space/child/repo-b",
			3: "space/other/repo-c",
		}, nil).Once()

	searcher := &fakeSearcher{result: types.SearchResult{}}

	ctrl := &Controller{
		searcher:    searcher,
		repoStore:   repoStore,
		repoFinder:  newRepoFinder(nil, nil, nil),
		spaceFinder: spaceFinder,
		repoFilter:  filterAll{},
	}

	_, err := ctrl.Search(context.Background(), &auth.Session{}, types.SearchInput{
		Query:      "foo",
		SpacePaths: []string{"space"},
		Recursive:  true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []int64{1, 2, 3}
	if !equalSortedInt64s(searcher.calledWithRepoIDs, want) {
		t.Fatalf("expected searcher called with %v, got %v", want, searcher.calledWithRepoIDs)
	}
	repoStore.AssertExpectations(t)
}

func TestSearch_DedupsAcrossRepoAndSpacePaths(t *testing.T) {
	t.Parallel()

	// The same repo (id=1) is reachable via both a direct repo path and via the space listing.
	repoFinder := newRepoFinder(
		map[int64]*types.RepositoryCore{
			1: {ID: 1, ParentID: 10, Path: "space/repo-a", Identifier: "repo-a"},
		},
		map[string]int64{"space": 10},
		map[types.RepoCacheKey]int64{
			{SpaceID: 10, RepoIdentifier: "repo-a"}: 1,
		},
	)
	spaceFinder := newSpaceFinder(
		map[int64]*types.SpaceCore{
			10: {ID: 10, Path: "space", Identifier: "space"},
		},
		map[string]int64{"space": 10},
	)

	repoStore := &mockstore.RepoStore{}
	repoStore.On("MapOfAllRepos", int64(10), false).
		Return(map[int64]string{
			1: "space/repo-a",
			2: "space/repo-b",
		}, nil).Once()

	searcher := &fakeSearcher{result: types.SearchResult{}}

	ctrl := &Controller{
		searcher:    searcher,
		repoStore:   repoStore,
		repoFinder:  repoFinder,
		spaceFinder: spaceFinder,
		repoFilter:  filterAll{},
	}

	_, err := ctrl.Search(context.Background(), &auth.Session{}, types.SearchInput{
		Query:      "foo",
		RepoPaths:  []string{"space/repo-a"},
		SpacePaths: []string{"space"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []int64{1, 2}
	if !equalSortedInt64s(searcher.calledWithRepoIDs, want) {
		t.Fatalf("expected searcher called with %v, got %v", want, searcher.calledWithRepoIDs)
	}
}

func TestSearch_DuplicateSpacePathsQueriedOnce(t *testing.T) {
	t.Parallel()

	// Both distinct space paths resolve to the same space ID; MapOfAllRepos must be called only once.
	spaceFinder := newSpaceFinder(
		map[int64]*types.SpaceCore{
			10: {ID: 10, Path: "space", Identifier: "space"},
		},
		map[string]int64{
			"space":       10,
			"space-alias": 10,
		},
	)

	repoStore := &mockstore.RepoStore{}
	repoStore.On("MapOfAllRepos", int64(10), false).
		Return(map[int64]string{1: "space/repo-a"}, nil).Once()

	ctrl := &Controller{
		searcher:    &fakeSearcher{},
		repoStore:   repoStore,
		repoFinder:  newRepoFinder(nil, nil, nil),
		spaceFinder: spaceFinder,
		repoFilter:  filterAll{},
	}

	_, err := ctrl.Search(context.Background(), &auth.Session{}, types.SearchInput{
		Query:      "foo",
		SpacePaths: []string{"space", "space-alias"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	repoStore.AssertExpectations(t)
}

func TestSearch_NoReposFoundReturns404(t *testing.T) {
	t.Parallel()

	spaceFinder := newSpaceFinder(
		map[int64]*types.SpaceCore{
			10: {ID: 10, Path: "empty-space", Identifier: "empty-space"},
		},
		map[string]int64{"empty-space": 10},
	)

	repoStore := &mockstore.RepoStore{}
	repoStore.On("MapOfAllRepos", int64(10), false).
		Return(map[int64]string{}, nil).Once()

	ctrl := &Controller{
		searcher:    &fakeSearcher{},
		repoStore:   repoStore,
		repoFinder:  newRepoFinder(nil, nil, nil),
		spaceFinder: spaceFinder,
		repoFilter:  filterAll{},
	}

	_, err := ctrl.Search(context.Background(), &auth.Session{}, types.SearchInput{
		Query:      "foo",
		SpacePaths: []string{"empty-space"},
	})
	if err == nil {
		t.Fatal("expected NotFound error")
	}
	if !strings.Contains(err.Error(), "No repositories found") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSearch_AccessDeniedReturns404(t *testing.T) {
	t.Parallel()

	spaceFinder := newSpaceFinder(
		map[int64]*types.SpaceCore{
			10: {ID: 10, Path: "space", Identifier: "space"},
		},
		map[string]int64{"space": 10},
	)

	repoStore := &mockstore.RepoStore{}
	repoStore.On("MapOfAllRepos", int64(10), false).
		Return(map[int64]string{
			1: "space/repo-a",
			2: "space/repo-b",
		}, nil).Once()

	ctrl := &Controller{
		searcher:    &fakeSearcher{},
		repoStore:   repoStore,
		repoFinder:  newRepoFinder(nil, nil, nil),
		spaceFinder: spaceFinder,
		repoFilter:  filterNone{},
	}

	_, err := ctrl.Search(context.Background(), &auth.Session{}, types.SearchInput{
		Query:      "foo",
		SpacePaths: []string{"space"},
	})
	if err == nil {
		t.Fatal("expected NotFound error when access filter yields empty set")
	}
	if !strings.Contains(err.Error(), "No repositories found") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSearch_OnlyAccessibleReposPassedToSearcher(t *testing.T) {
	t.Parallel()

	spaceFinder := newSpaceFinder(
		map[int64]*types.SpaceCore{
			10: {ID: 10, Path: "space", Identifier: "space"},
		},
		map[string]int64{"space": 10},
	)

	repoStore := &mockstore.RepoStore{}
	repoStore.On("MapOfAllRepos", int64(10), false).
		Return(map[int64]string{
			1: "space/repo-a",
			2: "space/repo-b",
			3: "space/repo-c",
		}, nil).Once()

	searcher := &fakeSearcher{result: types.SearchResult{
		FileMatches: []types.FileMatch{{RepoID: 1, FileName: "a.go"}},
	}}

	ctrl := &Controller{
		searcher:    searcher,
		repoStore:   repoStore,
		repoFinder:  newRepoFinder(nil, nil, nil),
		spaceFinder: spaceFinder,
		repoFilter:  filterOnly{allowed: map[int64]bool{1: true, 3: true}},
	}

	_, err := ctrl.Search(context.Background(), &auth.Session{}, types.SearchInput{
		Query:      "foo",
		SpacePaths: []string{"space"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []int64{1, 3}
	if !equalSortedInt64s(searcher.calledWithRepoIDs, want) {
		t.Fatalf("expected searcher called with %v, got %v", want, searcher.calledWithRepoIDs)
	}
}

func TestSearch_FilterErrorPropagates(t *testing.T) {
	t.Parallel()

	spaceFinder := newSpaceFinder(
		map[int64]*types.SpaceCore{
			10: {ID: 10, Path: "space", Identifier: "space"},
		},
		map[string]int64{"space": 10},
	)

	repoStore := &mockstore.RepoStore{}
	repoStore.On("MapOfAllRepos", int64(10), false).
		Return(map[int64]string{1: "space/repo-a"}, nil).Once()

	boom := errors.New("authz down")
	ctrl := &Controller{
		searcher:    &fakeSearcher{},
		repoStore:   repoStore,
		repoFinder:  newRepoFinder(nil, nil, nil),
		spaceFinder: spaceFinder,
		repoFilter:  filterErr{err: boom},
	}

	_, err := ctrl.Search(context.Background(), &auth.Session{}, types.SearchInput{
		Query:      "foo",
		SpacePaths: []string{"space"},
	})
	if err == nil {
		t.Fatal("expected error from filter")
	}
	if !errors.Is(err, boom) {
		t.Fatalf("expected wrapped filter error, got: %v", err)
	}
}

func TestSearch_UnknownRepoIDInResultIsSkipped(t *testing.T) {
	t.Parallel()

	spaceFinder := newSpaceFinder(
		map[int64]*types.SpaceCore{
			10: {ID: 10, Path: "space", Identifier: "space"},
		},
		map[string]int64{"space": 10},
	)

	repoStore := &mockstore.RepoStore{}
	repoStore.On("MapOfAllRepos", int64(10), false).
		Return(map[int64]string{1: "space/repo-a"}, nil).Once()

	// Searcher returns a match for a repo the map doesn't know about → RepoPath left empty.
	searcher := &fakeSearcher{result: types.SearchResult{
		FileMatches: []types.FileMatch{
			{RepoID: 1, FileName: "a.go"},
			{RepoID: 999, FileName: "stray.go"},
		},
	}}

	ctrl := &Controller{
		searcher:    searcher,
		repoStore:   repoStore,
		repoFinder:  newRepoFinder(nil, nil, nil),
		spaceFinder: spaceFinder,
		repoFilter:  filterAll{},
	}

	got, err := ctrl.Search(context.Background(), &auth.Session{}, types.SearchInput{
		Query:      "foo",
		SpacePaths: []string{"space"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got.FileMatches) != 2 {
		t.Fatalf("expected 2 file matches, got %d", len(got.FileMatches))
	}
	byID := map[int64]string{}
	for _, m := range got.FileMatches {
		byID[m.RepoID] = m.RepoPath
	}
	if byID[1] != "space/repo-a" {
		t.Fatalf("expected known repo path to be set, got %q", byID[1])
	}
	if byID[999] != "" {
		t.Fatalf("expected unknown repo path to remain empty, got %q", byID[999])
	}
}

func TestSearch_EmptyPathEntriesAreSkipped(t *testing.T) {
	t.Parallel()

	spaceFinder := newSpaceFinder(
		map[int64]*types.SpaceCore{
			10: {ID: 10, Path: "space", Identifier: "space"},
		},
		map[string]int64{"space": 10},
	)
	repoFinder := newRepoFinder(
		map[int64]*types.RepositoryCore{
			1: {ID: 1, ParentID: 10, Path: "space/repo-a", Identifier: "repo-a"},
		},
		map[string]int64{"space": 10},
		map[types.RepoCacheKey]int64{
			{SpaceID: 10, RepoIdentifier: "repo-a"}: 1,
		},
	)

	repoStore := &mockstore.RepoStore{}
	repoStore.On("MapOfAllRepos", int64(10), false).
		Return(map[int64]string{2: "space/repo-b"}, nil).Once()

	searcher := &fakeSearcher{result: types.SearchResult{}}

	ctrl := &Controller{
		searcher:    searcher,
		repoStore:   repoStore,
		repoFinder:  repoFinder,
		spaceFinder: spaceFinder,
		repoFilter:  filterAll{},
	}

	_, err := ctrl.Search(context.Background(), &auth.Session{}, types.SearchInput{
		Query:      "foo",
		RepoPaths:  []string{"", "space/repo-a"},
		SpacePaths: []string{"", "space"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []int64{1, 2}
	if !equalSortedInt64s(searcher.calledWithRepoIDs, want) {
		t.Fatalf("expected searcher called with %v, got %v", want, searcher.calledWithRepoIDs)
	}
	repoStore.AssertExpectations(t)
}

// ---------- shared helpers ----------

func equalSortedInt64s(a, b []int64) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
