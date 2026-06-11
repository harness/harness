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
	"fmt"
	"strings"
	"testing"

	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/services/refcache"
	storecache "github.com/harness/gitness/app/store/cache"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

type createTestAuthorizer struct{}

func (createTestAuthorizer) Check(
	_ context.Context, _ *auth.Session, _ *types.Scope, _ *types.Resource, _ enum.Permission,
) (bool, error) {
	return true, nil
}

func (createTestAuthorizer) CheckAll(
	_ context.Context, _ *auth.Session, _ ...types.PermissionCheck,
) (bool, error) {
	return true, nil
}

type createTestRepoCache struct {
	repos map[int64]*types.RepositoryCore
}

func (c *createTestRepoCache) Stats() (int64, int64)            { return 0, 0 }
func (c *createTestRepoCache) Evict(_ context.Context, _ int64) {}
func (c *createTestRepoCache) Get(_ context.Context, id int64) (*types.RepositoryCore, error) {
	if r, ok := c.repos[id]; ok {
		return r, nil
	}
	return nil, fmt.Errorf("repo %d not found", id)
}

func newCreateLinkedTestController(repos map[int64]*types.RepositoryCore) *Controller {
	repoFinder := refcache.NewRepoFinder(
		nil,
		nil,
		&createTestRepoCache{repos: repos},
		nil,
		storecache.Evictor[*types.RepositoryCore]{},
	)
	return &Controller{
		repoFinder: repoFinder,
		authorizer: createTestAuthorizer{},
	}
}

func TestCreate_LinkedRepoForbidden(t *testing.T) {
	t.Parallel()

	const linkedRepoID int64 = 1

	c := newCreateLinkedTestController(map[int64]*types.RepositoryCore{
		linkedRepoID: {
			ID:            linkedRepoID,
			Path:          "myspace/linked-repo",
			Type:          enum.RepoTypeLinked,
			DefaultBranch: "main",
		},
	})

	_, err := c.Create(context.Background(), &auth.Session{}, fmt.Sprintf("%d", linkedRepoID), &CreateInput{
		Title:        "Test PR",
		SourceBranch: "feature",
		TargetBranch: "main",
	})
	if err == nil {
		t.Fatal("expected error creating pull request in linked repo")
	}
	if !strings.Contains(err.Error(), "linked repository") {
		t.Fatalf("expected linked repository error, got: %v", err)
	}
}
