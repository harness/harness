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
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/auth/authz/authztest"
	repoevents "github.com/harness/gitness/app/events/repo"
	"github.com/harness/gitness/app/services/publicaccess"
	"github.com/harness/gitness/app/services/refcache"
	"github.com/harness/gitness/app/store"
	storecache "github.com/harness/gitness/app/store/cache"
	"github.com/harness/gitness/app/url"
	"github.com/harness/gitness/audit"
	"github.com/harness/gitness/events"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

// mockRepoStore is a test stub for store.RepoStore serving repos from memory.
// Only the methods used by Update are implemented, the embedded interface covers
// the rest of the method set.
type mockRepoStore struct {
	store.RepoStore
	repos map[int64]*types.Repository
}

func (m *mockRepoStore) Find(_ context.Context, id int64) (*types.Repository, error) {
	repo, ok := m.repos[id]
	if !ok {
		return nil, fmt.Errorf("repo %d not found", id)
	}

	clone := repo.Clone()

	return &clone, nil
}

func (m *mockRepoStore) UpdateOptLock(
	_ context.Context,
	repo *types.Repository,
	mutateFn func(repository *types.Repository) error,
) (*types.Repository, error) {
	clone := repo.Clone()
	if err := mutateFn(&clone); err != nil {
		return nil, err
	}

	m.repos[clone.ID] = &clone

	return &clone, nil
}

// stubURLProvider is a test stub for url.Provider covering the git URL backfill.
type stubURLProvider struct {
	url.Provider
}

func (stubURLProvider) GenerateGITCloneURL(_ context.Context, repoPath string) string {
	return "https://git.example.com/" + repoPath + ".git"
}

func (stubURLProvider) GenerateGITCloneSSHURL(_ context.Context, repoPath string) string {
	return "ssh://git.example.com/" + repoPath + ".git"
}

// stubPublicAccess is a test stub for publicaccess.Service reporting all resources as private.
type stubPublicAccess struct {
	publicaccess.Service
}

func (stubPublicAccess) Get(_ context.Context, _ enum.PublicResourceType, _ string) (bool, error) {
	return false, nil
}

// newUpdateTestController builds a Controller wired with in-memory stores and an in-memory
// event system, sufficient to run Update end to end for description and state changes.
func newUpdateTestController(t *testing.T, repos map[int64]*types.Repository) (*Controller, *mockRepoStore) {
	t.Helper()

	eventSystem, err := events.ProvideSystem(events.Config{
		Mode:            events.ModeInMemory,
		MaxStreamLength: 100,
	}, nil, events.NewNoopCollector())
	if err != nil {
		t.Fatalf("failed to create event system: %v", err)
	}

	eventReporter, err := repoevents.NewReporter(eventSystem)
	if err != nil {
		t.Fatalf("failed to create repo event reporter: %v", err)
	}

	cores := make(map[int64]*types.RepositoryCore, len(repos))
	for id, repo := range repos {
		cores[id] = repo.Core()
	}

	repoFinder := refcache.NewRepoFinder(
		nil, // repoStore — not reached for numeric refs
		nil, // spacePathCache — not reached for numeric refs
		&staticRepoIDCache{repos: cores},
		nil, // repoRefCache — not reached for numeric refs
		storecache.Evictor[*types.RepositoryCore]{}, // zero-value is a no-op
	)

	repoStore := &mockRepoStore{repos: repos}

	return &Controller{
		repoFinder:    repoFinder,
		authorizer:    authztest.AllowAuthorizer{},
		repoStore:     repoStore,
		repoCheck:     NewNoOpRepoChecks(),
		publicAccess:  stubPublicAccess{},
		urlProvider:   stubURLProvider{},
		auditService:  audit.New(),
		eventReporter: eventReporter,
	}, repoStore
}

// TestUpdate_LinkedRepoDescriptionIsNotEditable verifies the ticket requirement that the
// description of a linked repository cannot be edited through the repo update endpoint,
// because it mirrors the provider-side repo. The cases cover every condition of that
// restriction: the repo type, a description being supplied at all, and it differing from
// the stored one.
func TestUpdate_LinkedRepoDescriptionIsNotEditable(t *testing.T) {
	const (
		linkedRepoID int64 = 1
		normalRepoID int64 = 2
	)

	newRepos := func() map[int64]*types.Repository {
		return map[int64]*types.Repository{
			linkedRepoID: {
				ID:          linkedRepoID,
				Path:        "myspace/linked-repo",
				Description: "linked description",
				State:       enum.RepoStateActive,
				Type:        enum.RepoTypeLinked,
			},
			normalRepoID: {
				ID:          normalRepoID,
				Path:        "myspace/normal-repo",
				Description: "normal description",
				State:       enum.RepoStateActive,
				Type:        enum.RepoTypeNormal,
			},
		}
	}

	archived := enum.RepoStateArchived
	newDescription := "edited description"
	sameLinkedDescription := "linked description"
	// A description that only differs by surrounding whitespace is still a change: the
	// restriction is applied after the input has been sanitized.
	paddedDescription := "  edited description  "

	tests := []struct {
		name      string
		repoID    int64
		in        *UpdateInput
		wantError bool
		// expected persisted values after the call.
		wantDescription string
		wantState       enum.RepoState
	}{
		{
			name:            "description of a linked repo cannot be edited",
			repoID:          linkedRepoID,
			in:              &UpdateInput{Description: &newDescription},
			wantError:       true,
			wantDescription: "linked description",
			wantState:       enum.RepoStateActive,
		},
		{
			name:            "padded description of a linked repo cannot be edited",
			repoID:          linkedRepoID,
			in:              &UpdateInput{Description: &paddedDescription},
			wantError:       true,
			wantDescription: "linked description",
			wantState:       enum.RepoStateActive,
		},
		// The restriction is checked before anything is written, so a payload that also
		// carries an allowed field is rejected as a whole - the state must not change.
		{
			name:            "description of a linked repo cannot be edited alongside its state",
			repoID:          linkedRepoID,
			in:              &UpdateInput{Description: &newDescription, State: &archived},
			wantError:       true,
			wantDescription: "linked description",
			wantState:       enum.RepoStateActive,
		},
		// Only an actual change is rejected: a client echoing back the current description
		// while editing something else is not trying to edit the description.
		{
			name:            "no-op description update on a linked repo is accepted",
			repoID:          linkedRepoID,
			in:              &UpdateInput{Description: &sameLinkedDescription},
			wantDescription: "linked description",
			wantState:       enum.RepoStateActive,
		},
		// No description supplied - the restriction does not apply and a linked repo stays
		// editable in every other respect.
		{
			name:            "state of a linked repo can still be edited",
			repoID:          linkedRepoID,
			in:              &UpdateInput{State: &archived},
			wantDescription: "linked description",
			wantState:       enum.RepoStateArchived,
		},
		// A normal repo is not a mirror of anything, so its description stays editable.
		{
			name:            "description of a normal repo can be edited",
			repoID:          normalRepoID,
			in:              &UpdateInput{Description: &newDescription},
			wantDescription: newDescription,
			wantState:       enum.RepoStateActive,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repos := newRepos()
			c, repoStore := newUpdateTestController(t, repos)

			out, err := c.Update(context.Background(), &auth.Session{}, fmt.Sprintf("%d", tt.repoID), tt.in)

			switch {
			case tt.wantError:
				if err == nil {
					t.Fatalf("expected error for linked repo, got output: %+v", out)
				}

				var userErr *usererror.Error
				if !errors.As(err, &userErr) {
					t.Fatalf("expected *usererror.Error, got %T: %v", err, err)
				}
				if userErr.Status != http.StatusForbidden {
					t.Errorf("expected HTTP %d, got %d", http.StatusForbidden, userErr.Status)
				}
				for _, want := range []string{"description", "linked"} {
					if !strings.Contains(strings.ToLower(userErr.Message), want) {
						t.Errorf("expected error message to mention %q, got: %q", want, userErr.Message)
					}
				}

			default:
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if out == nil {
					t.Fatal("expected repository output, got nil")
				}
			}

			// Whatever the outcome, the persisted repo must match the expectation - a
			// rejected update must not have written anything to the store.
			stored, err := repoStore.Find(context.Background(), tt.repoID)
			if err != nil {
				t.Fatalf("failed to find repo in store: %v", err)
			}
			if stored.Description != tt.wantDescription {
				t.Errorf("expected stored description %q, got %q", tt.wantDescription, stored.Description)
			}
			if stored.State != tt.wantState {
				t.Errorf("expected stored state %q, got %q", tt.wantState, stored.State)
			}
		})
	}
}
