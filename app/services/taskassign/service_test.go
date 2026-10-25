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

package taskassign

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/harness/gitness/app/services/codeowners"
	"github.com/harness/gitness/app/services/usergroup"
	"github.com/harness/gitness/app/store"
	gitness_store "github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
)

type fakeMatcher struct {
	entries []codeowners.Entry
	err     error
}

// MatchPaths treats the entry patterns as plain path prefixes.
func (f *fakeMatcher) MatchPaths(
	_ context.Context, _ *types.RepositoryCore, ref string, paths []string,
) (*codeowners.PathMatches, error) {
	if ref != "" {
		return nil, fmt.Errorf("expected default branch, got ref %q", ref)
	}
	if f.err != nil {
		return nil, f.err
	}
	res := &codeowners.PathMatches{FileSHA: "sha"}
	for _, p := range paths {
		for _, e := range f.entries {
			if strings.HasPrefix(p, e.Pattern) {
				res.Matches = append(res.Matches, codeowners.PathMatch{Path: p, Entry: e})
			}
		}
	}
	return res, nil
}

type fakePrincipals struct {
	store.PrincipalStore
	byEmail map[string]int64
}

func (f *fakePrincipals) FindByEmail(_ context.Context, email string) (*types.Principal, error) {
	id, ok := f.byEmail[email]
	if !ok {
		return nil, gitness_store.ErrResourceNotFound
	}
	return &types.Principal{ID: id, Email: email, DisplayName: email}, nil
}

type fakeGroups struct {
	byID map[string]int64
}

func (f *fakeGroups) Resolve(_ context.Context, id string) (*types.UserGroup, error) {
	gid, ok := f.byID[id]
	if !ok {
		return nil, usergroup.ErrNotFound
	}
	return &types.UserGroup{ID: gid, Identifier: id, Name: id}, nil
}

type fakeRepos struct {
	store.RepoStore
}

func (fakeRepos) Find(_ context.Context, id int64) (*types.Repository, error) {
	return &types.Repository{ID: id, DefaultBranch: "main"}, nil
}

func newTestService(m *fakeMatcher) *Service {
	return NewService(
		m,
		fakeRepos{},
		&fakePrincipals{byEmail: map[string]int64{"a@x.io": 1, "b@x.io": 2}},
		&fakeGroups{byID: map[string]int64{"team": 10}},
	)
}

func TestSuggest(t *testing.T) {
	repo := &types.RepositoryCore{DefaultBranch: "main"}
	entries := []codeowners.Entry{
		{LineNumber: 1, Pattern: "api/", Owners: []string{"@team", "a@x.io"}},
		{LineNumber: 2, Pattern: "web/", Owners: []string{"b@x.io", "ghost@x.io", "@nogroup"}},
	}

	tests := []struct {
		name      string
		matcher   *fakeMatcher
		paths     []string
		wantOwner string // empty: no top candidate
		wantOrder []string
		wantRules int
	}{
		{
			name:    "no codeowners file",
			matcher: &fakeMatcher{err: codeowners.ErrNotFound},
			paths:   []string{"api/x.go"},
		},
		{
			name:    "no matching rule",
			matcher: &fakeMatcher{entries: entries},
			paths:   []string{"docs/x.md"},
		},
		{
			name:    "no paths",
			matcher: &fakeMatcher{entries: entries},
		},
		{
			name:      "most paths wins over order",
			matcher:   &fakeMatcher{entries: entries},
			paths:     []string{"api/x.go", "web/a.ts", "web/b.ts"},
			wantOwner: "b@x.io",
			wantOrder: []string{"b@x.io", "a@x.io", "@team"},
			wantRules: 2,
		},
		{
			name:      "tie prefers user over group",
			matcher:   &fakeMatcher{entries: entries},
			paths:     []string{"api/x.go"},
			wantOwner: "a@x.io",
			wantOrder: []string{"a@x.io", "@team"},
			wantRules: 1,
		},
		{
			name:      "tie between users keeps codeowners order",
			matcher:   &fakeMatcher{entries: entries},
			paths:     []string{"web/a.ts", "api/x.go"},
			wantOwner: "a@x.io",
			wantOrder: []string{"a@x.io", "b@x.io", "@team"},
			wantRules: 2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := newTestService(tt.matcher).Suggest(context.Background(), repo, tt.paths)
			if err != nil {
				t.Fatalf("Suggest() error = %v", err)
			}
			if tt.wantOwner == "" {
				if got.Top != nil || len(got.Candidates) != 0 {
					t.Fatalf("Suggest() = %+v, want empty", got)
				}
				return
			}
			if got.Top == nil || got.Top.Owner != tt.wantOwner {
				t.Fatalf("Suggest() top = %+v, want %q", got.Top, tt.wantOwner)
			}
			var order []string
			for _, c := range got.Candidates {
				order = append(order, c.Owner)
			}
			if fmt.Sprint(order) != fmt.Sprint(tt.wantOrder) {
				t.Errorf("candidates = %v, want %v", order, tt.wantOrder)
			}
			if len(got.Rules) != tt.wantRules {
				t.Errorf("rules = %d, want %d", len(got.Rules), tt.wantRules)
			}
			if got.FileSHA != "sha" {
				t.Errorf("file sha = %q", got.FileSHA)
			}
		})
	}
}

func TestSuggestForTask(t *testing.T) {
	entries := []codeowners.Entry{
		{LineNumber: 1, Pattern: "api/", Owners: []string{"@team"}},
		{LineNumber: 2, Pattern: "web/", Owners: []string{"b@x.io"}},
	}
	s := newTestService(&fakeMatcher{entries: entries})
	task := &types.Task{RepoID: 5}

	_, in, err := s.SuggestForTask(context.Background(), task, []string{"web/a.ts"})
	if err != nil {
		t.Fatal(err)
	}
	if in == nil || in.PrincipalID == nil || *in.PrincipalID != 2 || in.GroupID != nil || in.IsAgent {
		t.Fatalf("user input = %+v", in)
	}

	_, in, err = s.SuggestForTask(context.Background(), task, []string{"api/a.go"})
	if err != nil {
		t.Fatal(err)
	}
	if in == nil || in.GroupID == nil || *in.GroupID != 10 || in.PrincipalID != nil || in.IsAgent {
		t.Fatalf("group input = %+v", in)
	}

	_, in, err = s.SuggestForTask(context.Background(), task, []string{"none/a.go"})
	if err != nil || in != nil {
		t.Fatalf("empty: in = %+v, err = %v", in, err)
	}
}
