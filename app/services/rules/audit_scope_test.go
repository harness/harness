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

package rules

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/harness/gitness/app/services/protection"
	"github.com/harness/gitness/app/sse"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/audit"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/require"
)

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

type fakeRuleStore struct {
	store.RuleStore
	rule *types.Rule

	// updated records the rules handed to Update, so tests can assert whether a change
	// was actually persisted.
	updated []*types.Rule
}

func (f *fakeRuleStore) FindByIdentifier(
	context.Context, enum.RuleParent, int64, string,
) (*types.Rule, error) {
	return f.rule, nil
}

func (f *fakeRuleStore) Update(_ context.Context, rule *types.Rule) error {
	f.updated = append(f.updated, rule)
	return nil
}

func (f *fakeRuleStore) Delete(_ context.Context, _ int64) error { return nil }

type fakePrincipalInfoCache struct {
	store.PrincipalInfoCache
}

func (fakePrincipalInfoCache) Map(
	context.Context, []int64,
) (map[int64]*types.PrincipalInfo, error) {
	return map[int64]*types.PrincipalInfo{}, nil
}

type fakeUserGroupStore struct {
	store.UserGroupStore
}

type fakeRepoIDCache struct {
	store.RepoIDCache
}

func (fakeRepoIDCache) Get(_ context.Context, id int64) (*types.RepositoryCore, error) {
	return &types.RepositoryCore{ID: id}, nil
}

type fakeRepoStore struct {
	store.RepoStore
}

func (fakeRepoStore) Find(_ context.Context, id int64) (*types.Repository, error) {
	return &types.Repository{ID: id, ParentID: 1}, nil
}

type fakeSSEStreamer struct {
	sse.Streamer
}

func (fakeSSEStreamer) Publish(context.Context, int64, enum.SSEType, any) {}

type fakeValidator struct{}

func (fakeValidator) Validate(context.Context, []int64, map[int64]*types.PrincipalInfo) error {
	return nil
}

func newTestService(
	t *testing.T,
	auditSvc audit.Service,
	rule *types.Rule,
) (*Service, *fakeRuleStore) {
	t.Helper()

	ruleStore := &fakeRuleStore{rule: rule}

	protectionManager, err := protection.ProvideManager(ruleStore)
	require.NoError(t, err)

	return &Service{
		ruleStore:          ruleStore,
		repoStore:          fakeRepoStore{},
		protectionManager:  protectionManager,
		auditService:       auditSvc,
		principalInfoCache: fakePrincipalInfoCache{},
		userGroupStore:     fakeUserGroupStore{},
		repoIDCache:        fakeRepoIDCache{},
		sseStreamer:        fakeSSEStreamer{},
		ruleValidator:      fakeValidator{},
	}, ruleStore
}

func newTestRule(parentType enum.RuleParent) *types.Rule {
	parentID := int64(42)

	rule := &types.Rule{
		ID:         7,
		Identifier: "require_PR",
		Type:       protection.TypeBranch,
		State:      enum.RuleStateActive,
		Pattern:    json.RawMessage(`{}`),
		RepoTarget: json.RawMessage(`{}`),
		Definition: json.RawMessage(`{}`),
	}

	switch parentType {
	case enum.RuleParentRepo:
		rule.RepoID = &parentID
	case enum.RuleParentSpace:
		rule.SpaceID = &parentID
	}

	return rule
}

// TestAuditSpacePath asserts the scope-derivation contract shared by rule create, update
// and delete. Create is covered here rather than end-to-end because it additionally
// depends on instrumentation, event reporting and a transactor.
func TestAuditSpacePath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		parentType enum.RuleParent
		path       string
		want       string
	}{
		{"account-level space", enum.RuleParentSpace, "acctId", "acctId"},
		{"org-level space", enum.RuleParentSpace, "acctId/orgId", "acctId/orgId"},
		{"project-level space", enum.RuleParentSpace, "acctId/orgId/projId", "acctId/orgId/projId"},
		{"repo under project", enum.RuleParentRepo, "acctId/orgId/projId/repoId", "acctId/orgId/projId"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, test.want, auditSpacePath(test.parentType, test.path))
		})
	}
}

// TestAuditSpacePath_Update asserts that a rule update is audited against the scope
// that actually owns the rule.
//
// Regression test for CODE-6972: a space-level (account/org/project) rule update was
// audited against paths.Parent(path), which overshoots by one level because, unlike a
// repo path, a space path is already the scope. For an account-level rule the derived
// path was "", producing an audit event with a blank account identifier that never
// surfaced in the Audit Trail.
func TestAuditSpacePath_Update(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		parentType    enum.RuleParent
		path          string
		wantSpacePath string
	}{
		{
			// The exact reproduction from CODE-6972: an account-level branch rule whose
			// description is edited. Created is recorded, Updated was not.
			name:          "account-level space rule is audited at account scope",
			parentType:    enum.RuleParentSpace,
			path:          "acctId",
			wantSpacePath: "acctId",
		},
		{
			name:          "org-level space rule is audited at org scope",
			parentType:    enum.RuleParentSpace,
			path:          "acctId/orgId",
			wantSpacePath: "acctId/orgId",
		},
		{
			name:          "project-level space rule is audited at project scope",
			parentType:    enum.RuleParentSpace,
			path:          "acctId/orgId/projId",
			wantSpacePath: "acctId/orgId/projId",
		},
		{
			// A repo path includes the repo itself, so the owning scope is its parent.
			name:          "repo-level rule is audited at its parent space scope",
			parentType:    enum.RuleParentRepo,
			path:          "acctId/orgId/projId/repoId",
			wantSpacePath: "acctId/orgId/projId",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			auditSvc := &capturingAuditService{}
			svc, _ := newTestService(t, auditSvc, newTestRule(test.parentType))

			newDescription := "aaaaaaa"
			_, err := svc.Update(
				context.Background(),
				&types.Principal{ID: 1, Type: enum.PrincipalTypeUser},
				test.parentType,
				42,
				"scopeIdentifier",
				test.path,
				"require_PR",
				&UpdateInput{Description: &newDescription},
			)
			require.NoError(t, err)

			require.Len(t, auditSvc.calls, 1, "a persisted update must emit exactly one audit event")
			require.Equal(t, audit.ActionUpdated, auditSvc.calls[0].action)
			require.Equal(t, audit.ResourceTypeBranchRule, auditSvc.calls[0].resource.Type)
			require.Equal(t, test.wantSpacePath, auditSvc.calls[0].spacePath)
		})
	}
}

// TestAuditSpacePath_Delete asserts the same scope derivation for deletes, which shared
// the CODE-6972 bug with updates.
func TestAuditSpacePath_Delete(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		parentType    enum.RuleParent
		path          string
		wantSpacePath string
	}{
		{
			name:          "account-level space rule is audited at account scope",
			parentType:    enum.RuleParentSpace,
			path:          "acctId",
			wantSpacePath: "acctId",
		},
		{
			name:          "project-level space rule is audited at project scope",
			parentType:    enum.RuleParentSpace,
			path:          "acctId/orgId/projId",
			wantSpacePath: "acctId/orgId/projId",
		},
		{
			name:          "repo-level rule is audited at its parent space scope",
			parentType:    enum.RuleParentRepo,
			path:          "acctId/orgId/projId/repoId",
			wantSpacePath: "acctId/orgId/projId",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			auditSvc := &capturingAuditService{}
			svc, _ := newTestService(t, auditSvc, newTestRule(test.parentType))

			err := svc.Delete(
				context.Background(),
				&types.Principal{ID: 1, Type: enum.PrincipalTypeUser},
				test.parentType,
				42,
				"scopeIdentifier",
				test.path,
				"require_PR",
			)
			require.NoError(t, err)

			require.Len(t, auditSvc.calls, 1, "a delete must emit exactly one audit event")
			require.Equal(t, audit.ActionDeleted, auditSvc.calls[0].action)
			require.Equal(t, test.wantSpacePath, auditSvc.calls[0].spacePath)
		})
	}
}

// TestUpdate_RepoTargetOnly asserts that an update changing only RepoTarget is persisted
// and audited.
//
// Regression test: UpdateInput.isEmpty() omitted RepoTarget, so a RepoTarget-only request
// short-circuited the whole Update before reaching ruleStore.Update - silently discarding
// the change itself, not merely its audit event.
func TestUpdate_RepoTargetOnly(t *testing.T) {
	t.Parallel()

	auditSvc := &capturingAuditService{}
	rule := newTestRule(enum.RuleParentSpace)
	svc, ruleStore := newTestService(t, auditSvc, rule)

	repoTarget := &protection.RepoTarget{
		Include: protection.RepoTargetFilter{IDs: []int64{101, 102}},
	}

	got, err := svc.Update(
		context.Background(),
		&types.Principal{ID: 1, Type: enum.PrincipalTypeUser},
		enum.RuleParentSpace,
		42,
		"scopeIdentifier",
		"acctId/orgId/projId",
		"require_PR",
		&UpdateInput{RepoTarget: repoTarget},
	)
	require.NoError(t, err)

	require.Len(t, ruleStore.updated, 1, "a RepoTarget-only change must be persisted")
	require.JSONEq(t, string(repoTarget.JSON()), string(got.RepoTarget),
		"the new RepoTarget must be stored on the rule")

	require.Len(t, auditSvc.calls, 1, "a RepoTarget-only change must emit an audit event")
	require.Equal(t, audit.ActionUpdated, auditSvc.calls[0].action)
	require.Equal(t, "acctId/orgId/projId", auditSvc.calls[0].spacePath)
}

// TestUpdate_NoChange asserts that the isEmpty/IsEqual short-circuits still suppress
// writes and audit events when nothing actually changes.
func TestUpdate_NoChange(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input *UpdateInput
	}{
		{
			name:  "no fields supplied",
			input: &UpdateInput{},
		},
		{
			name:  "fields supplied but identical to current values",
			input: &UpdateInput{Description: ptr("original description")},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			auditSvc := &capturingAuditService{}
			rule := newTestRule(enum.RuleParentSpace)
			rule.Description = "original description"
			svc, ruleStore := newTestService(t, auditSvc, rule)

			_, err := svc.Update(
				context.Background(),
				&types.Principal{ID: 1, Type: enum.PrincipalTypeUser},
				enum.RuleParentSpace,
				42,
				"scopeIdentifier",
				"acctId/orgId/projId",
				"require_PR",
				test.input,
			)
			require.NoError(t, err)

			require.Empty(t, ruleStore.updated, "a no-op update must not be persisted")
			require.Empty(t, auditSvc.calls, "a no-op update must not emit an audit event")
		})
	}
}

func ptr[T any](v T) *T { return &v }
