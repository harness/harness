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
	stderrors "errors"
	"fmt"
	"strings"
	"testing"

	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/auth/authz/authztest"
	"github.com/harness/gitness/app/services/importer"
	"github.com/harness/gitness/app/services/refcache"
	"github.com/harness/gitness/app/store"
	storecache "github.com/harness/gitness/app/store/cache"
	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

// recordingWebhookService captures the input passed to UpsertWebhook so tests
// can assert that the controller forwards the connector + clone URL it pulled
// from the linked-repo row and the connector service.
type recordingWebhookService struct {
	gotInput importer.UpsertWebhookInput
	called   bool
	err      error
}

func (s *recordingWebhookService) UpsertWebhook(_ context.Context, in importer.UpsertWebhookInput) error {
	s.called = true
	s.gotInput = in
	return s.err
}

func newLinkedRegisterTestController(
	repos map[int64]*types.RepositoryCore,
	linkedStore store.LinkedRepoStore,
	connService importer.ConnectorService,
	webhookSvc importer.WebhookService,
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
		authorizer:       authztest.AllowAuthorizer{},
		linkedRepoStore:  linkedStore,
		connectorService: connService,
		webhookService:   webhookSvc,
	}
}

// TestLinkedRegisterWebhook_Success verifies the happy path: a linked repo
// resolves its connector, fetches the clone URL, and forwards (parent space
// path, connector path/identifier, clone URL) to WebhookService.UpsertWebhook.
func TestLinkedRegisterWebhook_Success(t *testing.T) {
	const linkedRepoID int64 = 1
	const parentSpacePath = "myspace"

	repos := map[int64]*types.RepositoryCore{
		linkedRepoID: {
			ID:   linkedRepoID,
			Path: parentSpacePath + "/linked-repo",
			Type: enum.RepoTypeLinked,
		},
	}

	connSvc := &mockConnectorService{
		info: importer.AccessInfo{URL: "https://github.com/myorg/myrepo"},
	}
	webhookSvc := &recordingWebhookService{}

	c := newLinkedRegisterTestController(
		repos,
		&mockLinkedRepoStore{
			linked: &types.LinkedRepo{
				RepoID:              linkedRepoID,
				ConnectorPath:       "account.myOrg.myProject",
				ConnectorIdentifier: "myGithubConnector",
			},
		},
		connSvc,
		webhookSvc,
	)

	err := c.LinkedRegisterWebhook(context.Background(), &auth.Session{}, fmt.Sprintf("%d", linkedRepoID))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !webhookSvc.called {
		t.Fatal("expected WebhookService.UpsertWebhook to be called")
	}
	if webhookSvc.gotInput.SpacePath != parentSpacePath {
		t.Errorf("space path = %q, want %q", webhookSvc.gotInput.SpacePath, parentSpacePath)
	}
	if webhookSvc.gotInput.ConnectorPath != "account.myOrg.myProject" {
		t.Errorf("connector path = %q, want %q",
			webhookSvc.gotInput.ConnectorPath, "account.myOrg.myProject")
	}
	if webhookSvc.gotInput.ConnectorIdentifier != "myGithubConnector" {
		t.Errorf("connector identifier = %q, want %q",
			webhookSvc.gotInput.ConnectorIdentifier, "myGithubConnector")
	}
	if webhookSvc.gotInput.CloneURL != "https://github.com/myorg/myrepo" {
		t.Errorf("clone URL = %q, want %q",
			webhookSvc.gotInput.CloneURL, "https://github.com/myorg/myrepo")
	}
}

// TestLinkedRegisterWebhook_NonLinkedRepoRejected verifies the API rejects
// non-linked repos with InvalidArgument before reaching the connector or the
// webhook service.
func TestLinkedRegisterWebhook_NonLinkedRepoRejected(t *testing.T) {
	const normalRepoID int64 = 2

	repos := map[int64]*types.RepositoryCore{
		normalRepoID: {
			ID:   normalRepoID,
			Path: "myspace/normal-repo",
			Type: enum.RepoTypeNormal,
		},
	}

	connSvc := &mockConnectorService{}
	webhookSvc := &recordingWebhookService{}

	c := newLinkedRegisterTestController(repos, &mockLinkedRepoStore{}, connSvc, webhookSvc)

	err := c.LinkedRegisterWebhook(context.Background(), &auth.Session{}, fmt.Sprintf("%d", normalRepoID))
	if err == nil {
		t.Fatal("expected error for non-linked repo, got nil")
	}
	if !errors.IsInvalidArgument(err) {
		t.Fatalf("expected InvalidArgument error, got status %q: %v", errors.AsStatus(err), err)
	}
	if !strings.Contains(err.Error(), "not a linked repository") {
		t.Errorf("error = %q; want it to mention 'not a linked repository'", err.Error())
	}

	if connSvc.called {
		t.Error("connector service must not be called for non-linked repos")
	}
	if webhookSvc.called {
		t.Error("webhook service must not be called for non-linked repos")
	}
}

// TestLinkedRegisterWebhook_LinkedRepoRecordMissing verifies the controller
// returns an error if linkedRepoStore.Find fails (data inconsistency case)
// and never reaches the connector or webhook service.
func TestLinkedRegisterWebhook_LinkedRepoRecordMissing(t *testing.T) {
	const linkedRepoID int64 = 1

	repos := map[int64]*types.RepositoryCore{
		linkedRepoID: {
			ID:   linkedRepoID,
			Path: "myspace/linked-repo",
			Type: enum.RepoTypeLinked,
		},
	}

	connSvc := &mockConnectorService{}
	webhookSvc := &recordingWebhookService{}

	c := newLinkedRegisterTestController(
		repos,
		&mockLinkedRepoStore{err: fmt.Errorf("sql: no rows in result set")},
		connSvc,
		webhookSvc,
	)

	err := c.LinkedRegisterWebhook(context.Background(), &auth.Session{}, fmt.Sprintf("%d", linkedRepoID))
	if err == nil {
		t.Fatal("expected error when linked repo record is missing, got nil")
	}
	if !strings.Contains(err.Error(), "failed to find linked repository") {
		t.Errorf("error = %q; want it to mention 'failed to find linked repository'", err.Error())
	}

	if connSvc.called {
		t.Error("connector service must not be called when linked repo record is missing")
	}
	if webhookSvc.called {
		t.Error("webhook service must not be called when linked repo record is missing")
	}
}

// TestLinkedRegisterWebhook_ConnectorAccessFailure verifies that connector
// failures (token revoked, connector deleted, etc.) propagate with context
// and the webhook service is not invoked.
func TestLinkedRegisterWebhook_ConnectorAccessFailure(t *testing.T) {
	const linkedRepoID int64 = 1

	repos := map[int64]*types.RepositoryCore{
		linkedRepoID: {
			ID:   linkedRepoID,
			Path: "myspace/linked-repo",
			Type: enum.RepoTypeLinked,
		},
	}

	connSvc := &mockConnectorService{err: fmt.Errorf("connector token expired")}
	webhookSvc := &recordingWebhookService{}

	c := newLinkedRegisterTestController(
		repos,
		&mockLinkedRepoStore{
			linked: &types.LinkedRepo{
				RepoID:              linkedRepoID,
				ConnectorPath:       "account.myOrg.myProject",
				ConnectorIdentifier: "revokedConnector",
			},
		},
		connSvc,
		webhookSvc,
	)

	err := c.LinkedRegisterWebhook(context.Background(), &auth.Session{}, fmt.Sprintf("%d", linkedRepoID))
	if err == nil {
		t.Fatal("expected error when connector fails, got nil")
	}
	if !strings.Contains(err.Error(), "failed to get connector access info") {
		t.Errorf("error = %q; want it to wrap 'failed to get connector access info'", err.Error())
	}
	if !strings.Contains(err.Error(), "connector token expired") {
		t.Errorf("error = %q; want it to preserve underlying connector error", err.Error())
	}

	if webhookSvc.called {
		t.Error("webhook service must not be called when connector access fails")
	}
}

// TestLinkedRegisterWebhook_EmptyCloneURLRejected guards the invariant that an
// empty URL from the connector indicates a misconfigured connector and must
// never be forwarded to the webhook service (an empty URL would either match
// nothing or — worse — collide on the SCM-service side).
func TestLinkedRegisterWebhook_EmptyCloneURLRejected(t *testing.T) {
	const linkedRepoID int64 = 1

	repos := map[int64]*types.RepositoryCore{
		linkedRepoID: {
			ID:   linkedRepoID,
			Path: "myspace/linked-repo",
			Type: enum.RepoTypeLinked,
		},
	}

	connSvc := &mockConnectorService{info: importer.AccessInfo{URL: ""}}
	webhookSvc := &recordingWebhookService{}

	c := newLinkedRegisterTestController(
		repos,
		&mockLinkedRepoStore{
			linked: &types.LinkedRepo{
				RepoID:              linkedRepoID,
				ConnectorPath:       "account.myOrg.myProject",
				ConnectorIdentifier: "myConnector",
			},
		},
		connSvc,
		webhookSvc,
	)

	err := c.LinkedRegisterWebhook(context.Background(), &auth.Session{}, fmt.Sprintf("%d", linkedRepoID))
	if err == nil {
		t.Fatal("expected error for empty clone URL, got nil")
	}
	if !strings.Contains(err.Error(), "empty repository URL") {
		t.Errorf("error = %q; want it to mention 'empty repository URL'", err.Error())
	}

	if webhookSvc.called {
		t.Error("webhook service must not be called when connector returns empty URL")
	}
}

// TestLinkedRegisterWebhook_UserFaultMappedTo400 verifies the user-fault
// classification path: a WebhookService UserFault error is rendered as a 400
// usererror so the user can see and act on the (token / permission) issue.
func TestLinkedRegisterWebhook_UserFaultMappedTo400(t *testing.T) {
	const linkedRepoID int64 = 1

	repos := map[int64]*types.RepositoryCore{
		linkedRepoID: {
			ID:   linkedRepoID,
			Path: "myspace/linked-repo",
			Type: enum.RepoTypeLinked,
		},
	}

	connSvc := &mockConnectorService{
		info: importer.AccessInfo{URL: "https://github.com/myorg/myrepo"},
	}
	webhookSvc := &recordingWebhookService{
		err: fmt.Errorf("%w: provider rejected hook", importer.ErrWebhookUpsertUserFault),
	}

	c := newLinkedRegisterTestController(
		repos,
		&mockLinkedRepoStore{
			linked: &types.LinkedRepo{
				RepoID:              linkedRepoID,
				ConnectorPath:       "account.myOrg.myProject",
				ConnectorIdentifier: "myConnector",
			},
		},
		connSvc,
		webhookSvc,
	)

	err := c.LinkedRegisterWebhook(context.Background(), &auth.Session{}, fmt.Sprintf("%d", linkedRepoID))
	if err == nil {
		t.Fatal("expected error for user-fault webhook failure, got nil")
	}

	var userErr *usererror.Error
	if !stderrors.As(err, &userErr) {
		t.Fatalf("expected *usererror.Error for user-fault failure, got %T: %v", err, err)
	}
	if userErr.Status != 400 {
		t.Errorf("status = %d, want 400 for user-fault failure", userErr.Status)
	}
	if !strings.Contains(strings.ToLower(userErr.Message), "webhook") {
		t.Errorf("message = %q; want it to mention webhook", userErr.Message)
	}
}

// TestLinkedRegisterWebhook_InfraFaultMappedTo500 verifies the infra-fault
// classification path: a WebhookService Infra error is surfaced as an internal
// error so it doesn't masquerade as a user-fixable problem.
func TestLinkedRegisterWebhook_InfraFaultMappedTo500(t *testing.T) {
	const linkedRepoID int64 = 1

	repos := map[int64]*types.RepositoryCore{
		linkedRepoID: {
			ID:   linkedRepoID,
			Path: "myspace/linked-repo",
			Type: enum.RepoTypeLinked,
		},
	}

	connSvc := &mockConnectorService{
		info: importer.AccessInfo{URL: "https://github.com/myorg/myrepo"},
	}
	webhookSvc := &recordingWebhookService{
		err: fmt.Errorf("%w: ng-manager 503", importer.ErrWebhookUpsertInfra),
	}

	c := newLinkedRegisterTestController(
		repos,
		&mockLinkedRepoStore{
			linked: &types.LinkedRepo{
				RepoID:              linkedRepoID,
				ConnectorPath:       "account.myOrg.myProject",
				ConnectorIdentifier: "myConnector",
			},
		},
		connSvc,
		webhookSvc,
	)

	err := c.LinkedRegisterWebhook(context.Background(), &auth.Session{}, fmt.Sprintf("%d", linkedRepoID))
	if err == nil {
		t.Fatal("expected error for infra-fault webhook failure, got nil")
	}

	if errors.AsStatus(err) != errors.StatusInternal {
		t.Errorf("error status = %q; want %q", errors.AsStatus(err), errors.StatusInternal)
	}
	// Infra errors must NOT be classified as user-fixable.
	if stderrors.Is(err, importer.ErrWebhookUpsertUserFault) {
		t.Error("infra fault was classified as user fault")
	}
}
