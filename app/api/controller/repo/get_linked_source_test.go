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
	"fmt"
	"strings"
	"testing"

	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/services/importer"
	"github.com/harness/gitness/app/services/refcache"
	"github.com/harness/gitness/app/store"
	storecache "github.com/harness/gitness/app/store/cache"
	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

// mockLinkedRepoStore is a test stub for store.LinkedRepoStore.
type mockLinkedRepoStore struct {
	store.LinkedRepoStore
	linked *types.LinkedRepo
	err    error
}

func (m *mockLinkedRepoStore) Find(_ context.Context, _ int64) (*types.LinkedRepo, error) {
	return m.linked, m.err
}

// mockConnectorService is a test stub for importer.ConnectorService that records the call.
type mockConnectorService struct {
	info        importer.AccessInfo
	err         error
	receivedDef importer.ConnectorDef
	called      bool
}

func (m *mockConnectorService) GetAccessInfo(_ context.Context, c importer.ConnectorDef) (importer.AccessInfo, error) {
	m.called = true
	m.receivedDef = c
	return m.info, m.err
}

func newLinkedSourceTestController(
	repos map[int64]*types.RepositoryCore,
	linkedStore store.LinkedRepoStore,
	connService importer.ConnectorService,
) *Controller {
	repoFinder := refcache.NewRepoFinder(
		nil,
		nil,
		&staticRepoIDCache{repos: repos},
		nil,
		storecache.Evictor[*types.RepositoryCore]{},
	)
	return &Controller{
		repoFinder:       repoFinder,
		authorizer:       alwaysAllowAuthorizer{},
		linkedRepoStore:  linkedStore,
		connectorService: connService,
	}
}

// TestGetLinkedSource_Success verifies the happy path: a linked repo resolves
// its connector and returns the plain source URL. Also verifies the correct
// connector path/identifier from the DB record is forwarded to the service.
func TestGetLinkedSource_Success(t *testing.T) {
	const linkedRepoID int64 = 1

	repos := map[int64]*types.RepositoryCore{
		linkedRepoID: {
			ID:   linkedRepoID,
			Path: "myspace/linked-repo",
			Type: enum.RepoTypeLinked,
		},
	}

	connSvc := &mockConnectorService{
		info: importer.AccessInfo{
			Username: "git",
			Password: "ghp_secrettoken",
			URL:      "https://github.com/myorg/myrepo",
		},
	}

	c := newLinkedSourceTestController(
		repos,
		&mockLinkedRepoStore{
			linked: &types.LinkedRepo{
				RepoID:              linkedRepoID,
				ConnectorPath:       "account.myOrg.myProject",
				ConnectorIdentifier: "myGithubConnector",
			},
		},
		connSvc,
	)

	result, err := c.GetLinkedSource(context.Background(), &auth.Session{}, fmt.Sprintf("%d", linkedRepoID))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.URL != "https://github.com/myorg/myrepo" {
		t.Errorf("expected URL %q, got %q", "https://github.com/myorg/myrepo", result.URL)
	}

	if !connSvc.called {
		t.Fatal("expected connectorService.GetAccessInfo to be called")
	}
	if connSvc.receivedDef.Path != "account.myOrg.myProject" {
		t.Errorf("expected connector path %q, got %q", "account.myOrg.myProject", connSvc.receivedDef.Path)
	}
	if connSvc.receivedDef.Identifier != "myGithubConnector" {
		t.Errorf("expected connector identifier %q, got %q", "myGithubConnector", connSvc.receivedDef.Identifier)
	}

	// The returned URL must be the plain URL, not URLWithCredentials() which would embed the token.
	if strings.Contains(result.URL, "ghp_secrettoken") {
		t.Error("returned URL must not contain the connector password")
	}
}

// TestGetLinkedSource_NonLinkedRepoReturns400 verifies the ticket requirement that
// the API should work only for linked repositories, otherwise it should return 400 status.
func TestGetLinkedSource_NonLinkedRepoReturns400(t *testing.T) {
	const normalRepoID int64 = 2

	repos := map[int64]*types.RepositoryCore{
		normalRepoID: {
			ID:   normalRepoID,
			Path: "myspace/normal-repo",
			Type: enum.RepoTypeNormal,
		},
	}

	connSvc := &mockConnectorService{}

	c := newLinkedSourceTestController(
		repos,
		&mockLinkedRepoStore{},
		connSvc,
	)

	_, err := c.GetLinkedSource(context.Background(), &auth.Session{}, fmt.Sprintf("%d", normalRepoID))
	if err == nil {
		t.Fatal("expected error for non-linked repo, got nil")
	}

	if !errors.IsInvalidArgument(err) {
		t.Fatalf("expected InvalidArgument error, got status %q: %v", errors.AsStatus(err), err)
	}

	if !strings.Contains(err.Error(), "not a linked repository") {
		t.Errorf("expected error about linked repository, got: %q", err.Error())
	}

	// Connector must never be called for a non-linked repo — we should fail before reaching it.
	if connSvc.called {
		t.Error("connectorService.GetAccessInfo should not be called for non-linked repo")
	}
}

// TestGetLinkedSource_ConnectorFailure verifies that when the Harness connector is
// unreachable (token revoked, connector deleted, etc.), the error propagates with context.
func TestGetLinkedSource_ConnectorFailure(t *testing.T) {
	const linkedRepoID int64 = 1

	repos := map[int64]*types.RepositoryCore{
		linkedRepoID: {
			ID:   linkedRepoID,
			Path: "myspace/linked-repo",
			Type: enum.RepoTypeLinked,
		},
	}

	c := newLinkedSourceTestController(
		repos,
		&mockLinkedRepoStore{
			linked: &types.LinkedRepo{
				RepoID:              linkedRepoID,
				ConnectorPath:       "account.myOrg.myProject",
				ConnectorIdentifier: "revokedConnector",
			},
		},
		&mockConnectorService{
			err: fmt.Errorf("connector token expired"),
		},
	)

	_, err := c.GetLinkedSource(context.Background(), &auth.Session{}, fmt.Sprintf("%d", linkedRepoID))
	if err == nil {
		t.Fatal("expected error when connector fails, got nil")
	}

	// Verify the error wraps context about what failed and preserves the original cause.
	if !strings.Contains(err.Error(), "failed to get access info") {
		t.Errorf("expected error to describe the failure context, got: %q", err.Error())
	}
	if !strings.Contains(err.Error(), "connector token expired") {
		t.Errorf("expected original connector error to be preserved, got: %q", err.Error())
	}
}

// TestGetLinkedSource_LinkedRepoRecordMissing verifies the scenario where a repo has
// type=linked in the repositories table but its corresponding row in
// linked_repositories is missing (data inconsistency that could happen after
// a failed cleanup or partial migration).
func TestGetLinkedSource_LinkedRepoRecordMissing(t *testing.T) {
	const linkedRepoID int64 = 1

	repos := map[int64]*types.RepositoryCore{
		linkedRepoID: {
			ID:   linkedRepoID,
			Path: "myspace/linked-repo",
			Type: enum.RepoTypeLinked,
		},
	}

	connSvc := &mockConnectorService{}

	c := newLinkedSourceTestController(
		repos,
		&mockLinkedRepoStore{
			err: fmt.Errorf("sql: no rows in result set"),
		},
		connSvc,
	)

	_, err := c.GetLinkedSource(context.Background(), &auth.Session{}, fmt.Sprintf("%d", linkedRepoID))
	if err == nil {
		t.Fatal("expected error when linked repo record is missing, got nil")
	}

	if !strings.Contains(err.Error(), "failed to find linked repository") {
		t.Errorf("expected error about finding linked repository, got: %q", err.Error())
	}

	// Connector should never be reached if the DB lookup fails.
	if connSvc.called {
		t.Error("connectorService should not be called when linked repo record is missing")
	}
}
