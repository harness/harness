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

package reposettings

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/auth/authz"
	"github.com/harness/gitness/app/services/refcache"
	"github.com/harness/gitness/app/services/settings"
	appstore "github.com/harness/gitness/app/store"
	storecache "github.com/harness/gitness/app/store/cache"
	"github.com/harness/gitness/audit"
	basestore "github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/require"
)

// TestGeneralUpdateSpace_AuditScope asserts that updating general space settings is audited
// against the scope that actually owns the settings.
//
// Regression test for CODE-6972: the space settings update was audited against
// paths.Parent(space.Path), which overshoots by one level because, unlike a repo path, a
// space path is already the scope. For an account-level space the derived path was "",
// producing an audit event with a blank account identifier that never surfaced in the
// Audit Trail; at project level the event was misfiled at org scope.
func TestGeneralUpdateSpace_AuditScope(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		path          string
		wantSpacePath string
	}{
		{
			// The account-level reproduction: paths.Parent("acctId") is "", so the event
			// carried a blank account identifier.
			name:          "account-level space is audited at account scope",
			path:          "acctId",
			wantSpacePath: "acctId",
		},
		{
			name:          "org-level space is audited at org scope",
			path:          "acctId/orgId",
			wantSpacePath: "acctId/orgId",
		},
		{
			name:          "project-level space is audited at project scope",
			path:          "acctId/orgId/projId",
			wantSpacePath: "acctId/orgId/projId",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			auditSvc := &capturingAuditService{}
			space := &types.SpaceCore{ID: 42, Identifier: "projId", Path: test.path}
			ctrl := newTestController(auditSvc, space)

			defaultBranch := "develop"
			_, err := ctrl.GeneralUpdateSpace(
				context.Background(),
				&auth.Session{Principal: types.Principal{ID: 1, Type: enum.PrincipalTypeUser}},
				test.path,
				&settings.GeneralSettingsSpace{DefaultBranch: &defaultBranch},
			)
			require.NoError(t, err)

			require.Len(t, auditSvc.calls, 1, "an update must emit exactly one audit event")
			require.Equal(t, audit.ActionUpdated, auditSvc.calls[0].action)
			require.Equal(t, audit.ResourceTypeSpaceSettings, auditSvc.calls[0].resource.Type)
			require.Equal(t, test.wantSpacePath, auditSvc.calls[0].spacePath)
		})
	}
}

func newTestController(auditSvc audit.Service, space *types.SpaceCore) *Controller {
	spaceFinder := refcache.NewSpaceFinder(
		&fakeSpaceIDCache{space: space},
		&fakeSpacePathCache{space: space},
		nil,
		storecache.Evictor[*types.SpaceCore]{},
	)

	return NewController(
		allowAllAuthorizer{},
		refcache.RepoFinder{},
		spaceFinder,
		settings.NewService(newInMemorySettingsStore(), spaceFinder),
		auditSvc,
	)
}

// auditCall records a single audit.Service.Log invocation.
type auditCall struct {
	action    audit.Action
	resource  audit.Resource
	spacePath string
}

// capturingAuditService records the spacePath handed to the audit service so tests can
// assert the scope an audit event would be filed under.
type capturingAuditService struct {
	calls []auditCall
}

func (c *capturingAuditService) Log(
	_ context.Context,
	_ types.Principal,
	resource audit.Resource,
	action audit.Action,
	spacePath string,
	_ ...audit.Option,
) error {
	c.calls = append(c.calls, auditCall{action: action, resource: resource, spacePath: spacePath})
	return nil
}

type allowAllAuthorizer struct {
	authz.Authorizer
}

func (allowAllAuthorizer) Check(
	context.Context, *auth.Session, *types.Scope, *types.Resource, enum.Permission,
) (bool, error) {
	return true, nil
}

// fakeSpaceIDCache serves a single space so SpaceFinder can resolve it without a database.
type fakeSpaceIDCache struct {
	space *types.SpaceCore
}

func (c *fakeSpaceIDCache) Get(_ context.Context, id int64) (*types.SpaceCore, error) {
	if c.space == nil || c.space.ID != id {
		return nil, basestore.ErrResourceNotFound
	}

	return c.space, nil
}

func (c *fakeSpaceIDCache) Stats() (int64, int64)            { return 0, 0 }
func (c *fakeSpaceIDCache) Evict(_ context.Context, _ int64) {}

// fakeSpacePathCache resolves the single space's path to its ID.
type fakeSpacePathCache struct {
	space *types.SpaceCore
}

func (c *fakeSpacePathCache) Get(_ context.Context, path string) (*types.SpacePath, error) {
	if c.space == nil || c.space.Path != path {
		return nil, basestore.ErrResourceNotFound
	}

	return &types.SpacePath{Value: path, IsPrimary: true, SpaceID: c.space.ID}, nil
}

func (c *fakeSpacePathCache) Stats() (int64, int64)             { return 0, 0 }
func (c *fakeSpacePathCache) Evict(_ context.Context, _ string) {}

type inMemorySettingsStore struct {
	values map[string]json.RawMessage
}

func newInMemorySettingsStore() *inMemorySettingsStore {
	return &inMemorySettingsStore{values: map[string]json.RawMessage{}}
}

func (s *inMemorySettingsStore) Find(
	_ context.Context,
	scope enum.SettingsScope,
	scopeID int64,
	key string,
) (json.RawMessage, error) {
	value, ok := s.values[s.makeKey(scope, scopeID, key)]
	if !ok {
		return nil, basestore.ErrResourceNotFound
	}

	return value, nil
}

func (s *inMemorySettingsStore) FindMany(
	_ context.Context,
	scope enum.SettingsScope,
	scopeID int64,
	keys ...string,
) (map[string]json.RawMessage, error) {
	out := make(map[string]json.RawMessage, len(keys))
	for _, key := range keys {
		if value, ok := s.values[s.makeKey(scope, scopeID, key)]; ok {
			out[key] = value
		}
	}

	return out, nil
}

func (s *inMemorySettingsStore) Upsert(
	_ context.Context,
	scope enum.SettingsScope,
	scopeID int64,
	key string,
	value json.RawMessage,
) error {
	s.values[s.makeKey(scope, scopeID, key)] = value
	return nil
}

func (s *inMemorySettingsStore) Delete(
	_ context.Context,
	scope enum.SettingsScope,
	scopeID int64,
	key string,
) error {
	delete(s.values, s.makeKey(scope, scopeID, key))
	return nil
}

func (s *inMemorySettingsStore) DeleteMany(
	_ context.Context,
	scope enum.SettingsScope,
	scopeID int64,
	keys ...string,
) error {
	for _, key := range keys {
		delete(s.values, s.makeKey(scope, scopeID, key))
	}

	return nil
}

func (s *inMemorySettingsStore) makeKey(scope enum.SettingsScope, scopeID int64, key string) string {
	return fmt.Sprintf("%s:%d:%s", scope, scopeID, key)
}

var _ appstore.SettingsStore = (*inMemorySettingsStore)(nil)
