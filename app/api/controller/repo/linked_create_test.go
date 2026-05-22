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

	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/services/importer"
	"github.com/harness/gitness/app/services/refcache"
	storecache "github.com/harness/gitness/app/store/cache"
	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/types"

	_ "unsafe" // for go:linkname
)

// bootstrapSystemServicePrincipal aliases the unexported package-level principal
// used by bootstrap.NewSystemServiceSession(). Tests that drive a controller path
// reaching verifyConnectorAccess seed this so the session lookup doesn't deref nil.
//
//go:linkname bootstrapSystemServicePrincipal github.com/harness/gitness/app/bootstrap.systemServicePrincipal
var bootstrapSystemServicePrincipal *types.Principal

func init() {
	bootstrapSystemServicePrincipal = &types.Principal{
		ID:    1,
		UID:   "harness-test",
		Email: "harness-test@local",
	}
}

func TestLinkedCreateInput_Sanitize_EmptyParentRef(t *testing.T) {
	in := &LinkedCreateInput{
		ParentRef:    "",
		Identifier:   "my-linked-repo",
		ConnectorRef: "account.githubConn",
	}

	err := in.sanitize()
	if err == nil {
		t.Fatal("expected error for empty parent_ref, got nil")
	}

	if !strings.Contains(err.Error(), "Parent space required") {
		t.Errorf("expected parent-space error from ValidateParentRef, got: %q", err.Error())
	}
}

func TestLinkedCreateInput_Sanitize_EmptyConnectorRef(t *testing.T) {
	in := &LinkedCreateInput{
		ParentRef:    "my-org/my-project",
		Identifier:   "my-linked-repo",
		ConnectorRef: "",
	}

	err := in.sanitize()
	if err == nil {
		t.Fatal("expected error for empty connector_ref, got nil")
	}

	if !errors.IsInvalidArgument(err) {
		t.Fatalf("expected InvalidArgument error, got status %q: %v", errors.AsStatus(err), err)
	}

	if !strings.Contains(err.Error(), "connector_ref") {
		t.Errorf("expected error to mention connector_ref, got: %q", err.Error())
	}
}

func TestLinkedCreateInput_Sanitize_ValidRefs(t *testing.T) {
	cases := []struct {
		name         string
		connectorRef string
	}{
		{"account-scoped", "account.githubConn"},
		{"org-scoped", "org.githubConn"},
		{"project-scoped bare identifier", "githubConn"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			in := &LinkedCreateInput{
				ParentRef:    "my-org/my-project",
				Identifier:   "my-linked-repo",
				ConnectorRef: tc.connectorRef,
			}

			if err := in.sanitize(); err != nil {
				t.Fatalf("sanitize() failed for valid input: %v", err)
			}
		})
	}
}

func TestLinkedCreateInput_Sanitize_TrimsRepoIdentifier(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{"leading and trailing spaces", "  owner/repo  ", "owner/repo"},
		{"leading tab", "\towner/repo", "owner/repo"},
		{"trailing newline", "owner/repo\n", "owner/repo"},
		{"already clean", "owner/repo", "owner/repo"},
		{"empty stays empty", "", ""},
		{"only whitespace becomes empty", "   ", ""},
		{"nested path preserved", "  group/subgroup/project  ", "group/subgroup/project"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			in := &LinkedCreateInput{
				ParentRef:      "my-org/my-project",
				Identifier:     "my-linked-repo",
				ConnectorRef:   "account.githubConn",
				RepoIdentifier: tc.input,
			}

			if err := in.sanitize(); err != nil {
				t.Fatalf("sanitize() failed: %v", err)
			}
			if in.RepoIdentifier != tc.want {
				t.Errorf("RepoIdentifier = %q; want %q", in.RepoIdentifier, tc.want)
			}
		})
	}
}

type staticSpaceIDCache struct {
	spaces map[int64]*types.SpaceCore
}

func (c *staticSpaceIDCache) Stats() (int64, int64)            { return 0, 0 }
func (c *staticSpaceIDCache) Evict(_ context.Context, _ int64) {}
func (c *staticSpaceIDCache) Get(_ context.Context, id int64) (*types.SpaceCore, error) {
	if s, ok := c.spaces[id]; ok {
		return s, nil
	}
	return nil, fmt.Errorf("space %d not found", id)
}

// TestLinkedCreate_DelegatesConnectorRefDecode drives LinkedCreate up to
// verifyConnectorAccess and asserts the controller routes the API-facing ref
// + parent space path through ConnectorService.ResolveConnectorRef, then
// forwards the resolved (path, identifier) pair plus the caller-supplied
// repo_identifier to GetAccessInfo. The connector service is stubbed to fail
// so the call returns before the transactional repo-creation block, which
// would otherwise need a full store/git/tx setup.
func TestLinkedCreate_DelegatesConnectorRefDecode(t *testing.T) {
	const parentSpaceID int64 = 1
	const parentSpacePath = "acme/platform/code"
	const inputRef = "account.githubConn"
	const inputRepoIdentifier = "harness/gitness"
	const resolvedPath = "acme"
	const resolvedIdentifier = "githubConn"

	connSvc := &mockConnectorService{
		err:                     stderrors.New("stubbed connector failure"),
		resolveReturnPath:       resolvedPath,
		resolveReturnIdentifier: resolvedIdentifier,
	}
	c := newLinkedCreateTestController(parentSpaceID, parentSpacePath, connSvc)

	_, err := c.LinkedCreate(context.Background(), &auth.Session{}, &LinkedCreateInput{
		ParentRef:      fmt.Sprintf("%d", parentSpaceID),
		Identifier:     "my-linked-repo",
		ConnectorRef:   inputRef,
		RepoIdentifier: inputRepoIdentifier,
	})
	if err == nil {
		t.Fatal("expected connector-access error, got nil")
	}
	if !strings.Contains(err.Error(), "Failed to use connector") {
		t.Errorf("expected connector-access error, got: %q", err.Error())
	}

	if !connSvc.resolveCalled {
		t.Fatal("connectorService.ResolveConnectorRef was not invoked")
	}
	if connSvc.resolveGotParentSpacePath != parentSpacePath {
		t.Errorf("resolver got parent path %q; want %q", connSvc.resolveGotParentSpacePath, parentSpacePath)
	}
	if connSvc.resolveGotRef != inputRef {
		t.Errorf("resolver got ref %q; want %q", connSvc.resolveGotRef, inputRef)
	}

	if !connSvc.called {
		t.Fatal("connectorService.GetAccessInfo was not invoked")
	}
	if connSvc.receivedDef.Path != resolvedPath {
		t.Errorf("connector path = %q; want %q", connSvc.receivedDef.Path, resolvedPath)
	}
	if connSvc.receivedDef.Identifier != resolvedIdentifier {
		t.Errorf("connector identifier = %q; want %q",
			connSvc.receivedDef.Identifier, resolvedIdentifier)
	}
	// RepoIdentifier is the user-supplied "which repo under this account" value;
	// the controller must thread it through unchanged so resolveCloneURL on the
	// service side sees it. If this regresses, account-level linked repos will
	// silently fall back to the "missing repo_identifier" error path.
	if connSvc.receivedDef.RepoIdentifier != inputRepoIdentifier {
		t.Errorf("connector repo_identifier = %q; want %q",
			connSvc.receivedDef.RepoIdentifier, inputRepoIdentifier)
	}
}

func newLinkedCreateTestController(
	spaceID int64,
	spacePath string,
	connSvc importer.ConnectorService,
) *Controller {
	spaceFinder := refcache.NewSpaceFinder(
		&staticSpaceIDCache{
			spaces: map[int64]*types.SpaceCore{
				spaceID: {ID: spaceID, Path: spacePath},
			},
		},
		nil,
		nil,
		storecache.Evictor[*types.SpaceCore]{},
	)
	return &Controller{
		spaceFinder:      spaceFinder,
		authorizer:       alwaysAllowAuthorizer{},
		publicAccess:     errPublicAccess{},
		identifierCheck:  func(_ string, _ *auth.Session) error { return nil },
		connectorService: connSvc,
	}
}
