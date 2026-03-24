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
	"github.com/harness/gitness/app/services/refcache"
	storecache "github.com/harness/gitness/app/store/cache"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

// alwaysAllowAuthorizer is a test authorizer that grants every permission check.
type alwaysAllowAuthorizer struct{}

func (alwaysAllowAuthorizer) Check(
	_ context.Context, _ *auth.Session, _ *types.Scope, _ *types.Resource, _ enum.Permission,
) (bool, error) {
	return true, nil
}

func (alwaysAllowAuthorizer) CheckAll(
	_ context.Context, _ *auth.Session, _ ...types.PermissionCheck,
) (bool, error) {
	return true, nil
}

// staticRepoIDCache is an in-memory cache.Cache[int64, *types.RepositoryCore] for tests.
type staticRepoIDCache struct {
	repos map[int64]*types.RepositoryCore
}

func (c *staticRepoIDCache) Stats() (int64, int64)            { return 0, 0 }
func (c *staticRepoIDCache) Evict(_ context.Context, _ int64) {}
func (c *staticRepoIDCache) Get(_ context.Context, id int64) (*types.RepositoryCore, error) {
	if r, ok := c.repos[id]; ok {
		return r, nil
	}
	return nil, fmt.Errorf("repo %d not found", id)
}

// errPublicAccess is a publicaccess.Service stub that always returns an error on Get/Set/Delete
// but reports public access as supported. This lets tests verify that the fork guard fires (or
// does not fire) without needing a real public-access store.
type errPublicAccess struct{}

func (errPublicAccess) Get(_ context.Context, _ enum.PublicResourceType, _ string) (bool, error) {
	return false, errors.New("mock: public access Get not implemented")
}

func (errPublicAccess) Set(_ context.Context, _ enum.PublicResourceType, _ string, _ bool) error {
	return errors.New("mock: public access Set not implemented")
}

func (errPublicAccess) Delete(_ context.Context, _ enum.PublicResourceType, _ string) error {
	return errors.New("mock: public access Delete not implemented")
}

func (errPublicAccess) IsPublicAccessSupported(
	_ context.Context, _ enum.PublicResourceType, _ string,
) (bool, error) {
	return true, nil
}

// newTestController builds a minimal Controller wired with an in-memory repo cache and
// the always-allow authorizer. Only the fields exercised by UpdatePublicAccess up to
// (and including) the fork guard are populated.
func newTestController(repos map[int64]*types.RepositoryCore) *Controller {
	repoFinder := refcache.NewRepoFinder(
		nil, // repoStore — not reached for numeric refs
		nil, // spacePathCache — not reached for numeric refs
		&staticRepoIDCache{repos: repos},
		nil, // repoRefCache — not reached for numeric refs
		storecache.Evictor[*types.RepositoryCore]{}, // zero-value is a no-op
	)
	return &Controller{
		repoFinder:   repoFinder,
		authorizer:   alwaysAllowAuthorizer{},
		publicAccess: errPublicAccess{},
	}
}

func TestUpdatePublicAccess_ForkRepoCannotBePublic(t *testing.T) {
	const (
		forkRepoID     int64 = 1
		upstreamRepoID int64 = 42
		plainRepoID    int64 = 2
	)

	repos := map[int64]*types.RepositoryCore{
		forkRepoID: {
			ID:     forkRepoID,
			ForkID: upstreamRepoID,
			Path:   "myspace/fork-repo",
		},
		plainRepoID: {
			ID:   plainRepoID,
			Path: "myspace/plain-repo",
		},
	}

	c := newTestController(repos)
	session := &auth.Session{}

	tests := []struct {
		name        string
		repoRef     string // numeric ref → direct repoIDCache lookup, no DB needed
		isPublic    bool
		wantForkErr bool // true when the fork guard should block the request
	}{
		{
			name:        "fork repository cannot be made public",
			repoRef:     fmt.Sprintf("%d", forkRepoID),
			isPublic:    true,
			wantForkErr: true,
		},
		{
			name:        "fork repository can be made private",
			repoRef:     fmt.Sprintf("%d", forkRepoID),
			isPublic:    false,
			wantForkErr: false,
		},
		{
			name:        "non-fork repository is not blocked by the fork guard",
			repoRef:     fmt.Sprintf("%d", plainRepoID),
			isPublic:    true,
			wantForkErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := c.UpdatePublicAccess(
				context.Background(),
				session,
				tt.repoRef,
				&UpdatePublicAccessInput{IsPublic: tt.isPublic},
			)

			if tt.wantForkErr {
				// The fork guard must fire and return a 400 Bad Request that mentions "fork".
				if err == nil {
					t.Fatal("expected fork error, got nil")
				}

				var userErr *usererror.Error
				if !errors.As(err, &userErr) {
					t.Fatalf("expected *usererror.Error, got %T: %v", err, err)
				}
				if userErr.Status != http.StatusBadRequest {
					t.Errorf("expected HTTP %d, got %d", http.StatusBadRequest, userErr.Status)
				}
				if !strings.Contains(strings.ToLower(userErr.Message), "fork") {
					t.Errorf("expected error message to mention %q, got: %q", "fork", userErr.Message)
				}
				return
			}

			// For cases where the fork guard must NOT fire: an error can still occur due to
			// the minimal mock setup (errPublicAccess returns an error on Get), but it must
			// not be the fork restriction error.
			if err != nil {
				var userErr *usererror.Error
				if errors.As(err, &userErr) && strings.Contains(strings.ToLower(userErr.Message), "fork") {
					t.Errorf("got unexpected fork restriction error: %v", err)
				}
			}
		})
	}
}
