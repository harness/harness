// Copyright 2026 Harness, Inc.
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

package webhook

import (
	"context"
	"errors"
	"testing"

	"github.com/harness/gitness/audit"
	gitnesserrors "github.com/harness/gitness/errors"
	"github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/check"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var svcPrincipal = &types.Principal{ID: 5, UID: "alice"}

func svcRepoParent() ParentResource {
	return ParentResource{ID: 1, Identifier: "repo", Type: enum.WebhookParentRepo, Path: "root/repo"}
}

func svcSpaceParent() ParentResource {
	return ParentResource{ID: 10, Identifier: "space", Type: enum.WebhookParentSpace, Path: "root"}
}

func TestSvcCreate(t *testing.T) {
	ctx := context.Background()

	newInput := func() *types.WebhookCreateInput {
		return &types.WebhookCreateInput{
			Identifier: "my-hook",
			URL:        "https://example.com/hook",
			Secret:     "topsecret",
			Enabled:    true,
			Triggers:   []enum.WebhookTrigger{enum.WebhookTriggerTagCreated, enum.WebhookTriggerTagCreated},
		}
	}

	t.Run("repo webhook", func(t *testing.T) {
		h := newSvcHarness(t)
		hook, err := h.svc.Create(ctx, svcPrincipal, enum.WebhookTypeExternal, svcRepoParent(), newInput())
		require.NoError(t, err)
		assert.Equal(t, "my-hook", hook.Identifier)
		assert.Equal(t, "my-hook", hook.DisplayName, "display name is backfilled")
		assert.Equal(t, webhookScopeRepo, hook.Scope)
		assert.Equal(t, int64(5), hook.CreatedBy)
		assert.Equal(t, "topsecret", hook.Secret, "secret goes through the encrypter")
		assert.Equal(t, []enum.WebhookTrigger{enum.WebhookTriggerTagCreated}, hook.Triggers, "triggers deduplicated")
		assert.Equal(t, []audit.Action{audit.ActionCreated}, h.audit.actions)
		require.Len(t, h.sse.events, 1)
		assert.Equal(t, enum.SSETypeWebhookCreated, h.sse.events[0].typ)
		assert.Equal(t, int64(10), h.sse.events[0].spaceID, "repo events go to the parent space")
	})
	t.Run("space webhook uses tree level as scope", func(t *testing.T) {
		h := newSvcHarness(t)
		h.spaces.level = 3
		hook, err := h.svc.Create(ctx, svcPrincipal, enum.WebhookTypeExternal, svcSpaceParent(), newInput())
		require.NoError(t, err)
		assert.Equal(t, int64(3), hook.Scope)
		assert.Equal(t, int64(10), h.sse.events[0].spaceID)
	})
	t.Run("space tree level failure", func(t *testing.T) {
		h := newSvcHarness(t)
		h.spaces.levelErr = errors.New("boom")
		_, err := h.svc.Create(ctx, svcPrincipal, enum.WebhookTypeExternal, svcSpaceParent(), newInput())
		require.Error(t, err)
	})
	t.Run("identifier backfilled from uid and display name", func(t *testing.T) {
		h := newSvcHarness(t)
		in := newInput()
		in.Identifier = ""
		in.UID = "legacy-uid"
		hook, err := h.svc.Create(ctx, svcPrincipal, enum.WebhookTypeExternal, svcRepoParent(), in)
		require.NoError(t, err)
		assert.Equal(t, "legacy-uid", hook.Identifier)

		in = newInput()
		in.Identifier = ""
		in.DisplayName = "My Fancy Hook"
		hook, err = h.svc.Create(ctx, svcPrincipal, enum.WebhookTypeExternal, svcRepoParent(), in)
		require.NoError(t, err)
		assert.NotEmpty(t, hook.Identifier)
		assert.Equal(t, "My Fancy Hook", hook.DisplayName)
	})
	t.Run("internal webhooks are not audited and skip url validation", func(t *testing.T) {
		h := newSvcHarness(t)
		in := newInput()
		in.URL = ""
		_, err := h.svc.Create(ctx, svcPrincipal, enum.WebhookTypeInternal, svcRepoParent(), in)
		require.NoError(t, err)
		assert.Empty(t, h.audit.actions)
	})
	t.Run("audit failure is not fatal", func(t *testing.T) {
		h := newSvcHarness(t)
		h.audit.err = errors.New("audit down")
		_, err := h.svc.Create(ctx, svcPrincipal, enum.WebhookTypeExternal, svcRepoParent(), newInput())
		require.NoError(t, err)
	})
	t.Run("invalid input", func(t *testing.T) {
		tests := []struct {
			name   string
			mutate func(in *types.WebhookCreateInput)
		}{
			{"bad identifier", func(in *types.WebhookCreateInput) { in.Identifier = "bad identifier!" }},
			{"bad url", func(in *types.WebhookCreateInput) { in.URL = "ftp://example.com" }},
			{"localhost url", func(in *types.WebhookCreateInput) { in.URL = "http://localhost/x" }},
			{"long secret", func(in *types.WebhookCreateInput) { in.Secret = string(make([]byte, webhookMaxSecretLength+1)) }},
			{"bad trigger", func(in *types.WebhookCreateInput) { in.Triggers = []enum.WebhookTrigger{"nope"} }},
			{"empty header key", func(in *types.WebhookCreateInput) {
				in.ExtraHeaders = []types.ExtraHeader{{Key: "", Value: "v"}}
			}},
			{"too many headers", func(in *types.WebhookCreateInput) {
				in.ExtraHeaders = make([]types.ExtraHeader, webhookMaxExtraHeaders+1)
			}},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				h := newSvcHarness(t)
				in := newInput()
				tt.mutate(in)
				_, err := h.svc.Create(ctx, svcPrincipal, enum.WebhookTypeExternal, svcRepoParent(), in)
				require.Error(t, err)
				assert.Empty(t, h.hooks.hooks)
			})
		}
	})
	t.Run("store failure", func(t *testing.T) {
		h := newSvcHarness(t)
		h.hooks.createErr = errors.New("db down")
		_, err := h.svc.Create(ctx, svcPrincipal, enum.WebhookTypeExternal, svcRepoParent(), newInput())
		require.Error(t, err)
		assert.Empty(t, h.sse.events)
	})
	t.Run("duplicate of external webhook is a plain error", func(t *testing.T) {
		h := newSvcHarness(t)
		_, err := h.svc.Create(ctx, svcPrincipal, enum.WebhookTypeExternal, svcRepoParent(), newInput())
		require.NoError(t, err)
		_, err = h.svc.Create(ctx, svcPrincipal, enum.WebhookTypeExternal, svcRepoParent(), newInput())
		require.ErrorIs(t, err, store.ErrDuplicate)
	})
	t.Run("duplicate of internal webhook is reported as reserved", func(t *testing.T) {
		h := newSvcHarness(t)
		_, err := h.svc.Create(ctx, svcPrincipal, enum.WebhookTypeInternal, svcRepoParent(), newInput())
		require.NoError(t, err)
		_, err = h.svc.Create(ctx, svcPrincipal, enum.WebhookTypeExternal, svcRepoParent(), newInput())
		require.Error(t, err)
		assert.Equal(t, gitnesserrors.StatusConflict, gitnesserrors.AsStatus(err))
	})
}

func TestSvcUpdate(t *testing.T) {
	ctx := context.Background()
	strp := func(s string) *string { return &s }
	boolp := func(b bool) *bool { return &b }

	setup := func(t *testing.T) *svcHarness {
		h := newSvcHarness(t)
		h.hooks.add(&types.Webhook{
			ParentID: 1, ParentType: enum.WebhookParentRepo, Identifier: "hook", DisplayName: "hook",
			URL: "https://example.com", Enabled: true, Type: enum.WebhookTypeExternal,
		})
		return h
	}

	t.Run("updates only provided fields", func(t *testing.T) {
		h := setup(t)
		in := &types.WebhookUpdateInput{
			DisplayName:  strp("New Name"),
			Description:  strp("desc"),
			URL:          strp("https://example.com/new"),
			Secret:       strp("newsecret"),
			Enabled:      boolp(false),
			Insecure:     boolp(true),
			Triggers:     []enum.WebhookTrigger{enum.WebhookTriggerTagCreated, enum.WebhookTriggerTagCreated},
			ExtraHeaders: []types.ExtraHeader{{Key: "X-A", Value: "b"}},
		}
		hook, err := h.svc.Update(ctx, svcPrincipal, "hook", enum.WebhookTypeExternal, svcRepoParent(), in)
		require.NoError(t, err)
		assert.Equal(t, "New Name", hook.DisplayName)
		assert.Equal(t, "hook", hook.Identifier, "identifier is untouched")
		assert.Equal(t, "desc", hook.Description)
		assert.Equal(t, "https://example.com/new", hook.URL)
		assert.Equal(t, "newsecret", hook.Secret)
		assert.False(t, hook.Enabled)
		assert.True(t, hook.Insecure)
		assert.Equal(t, []enum.WebhookTrigger{enum.WebhookTriggerTagCreated}, hook.Triggers)
		assert.Len(t, hook.ExtraHeaders, 1)
		assert.Equal(t, []audit.Action{audit.ActionUpdated}, h.audit.actions)
		require.Len(t, h.sse.events, 1)
		assert.Equal(t, enum.SSETypeWebhookUpdated, h.sse.events[0].typ)
	})
	t.Run("identifier via legacy uid", func(t *testing.T) {
		h := setup(t)
		hook, err := h.svc.Update(ctx, svcPrincipal, "hook", enum.WebhookTypeExternal, svcRepoParent(),
			&types.WebhookUpdateInput{UID: strp("renamed")})
		require.NoError(t, err)
		assert.Equal(t, "renamed", hook.Identifier)
	})
	t.Run("not found", func(t *testing.T) {
		h := setup(t)
		_, err := h.svc.Update(ctx, svcPrincipal, "unknown", enum.WebhookTypeExternal, svcRepoParent(),
			&types.WebhookUpdateInput{})
		require.Error(t, err)
	})
	t.Run("type change is rejected", func(t *testing.T) {
		h := setup(t)
		_, err := h.svc.Update(ctx, svcPrincipal, "hook", enum.WebhookTypeInternal, svcRepoParent(),
			&types.WebhookUpdateInput{})
		require.Error(t, err)
	})
	t.Run("invalid input", func(t *testing.T) {
		tests := []struct {
			name string
			in   *types.WebhookUpdateInput
		}{
			{"identifier", &types.WebhookUpdateInput{Identifier: strp("bad id!")}},
			{"display name", &types.WebhookUpdateInput{DisplayName: strp("")}},
			{"url", &types.WebhookUpdateInput{URL: strp("ftp://x")}},
			{"secret", &types.WebhookUpdateInput{Secret: strp(string(make([]byte, webhookMaxSecretLength+1)))}},
			{"triggers", &types.WebhookUpdateInput{Triggers: []enum.WebhookTrigger{"bad"}}},
			{"headers", &types.WebhookUpdateInput{ExtraHeaders: []types.ExtraHeader{{Key: ""}}}},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				h := setup(t)
				_, err := h.svc.Update(ctx, svcPrincipal, "hook", enum.WebhookTypeExternal, svcRepoParent(), tt.in)
				require.Error(t, err)
				assert.Empty(t, h.sse.events)
			})
		}
	})
	t.Run("store failure", func(t *testing.T) {
		h := setup(t)
		h.hooks.updateErr = errors.New("db down")
		_, err := h.svc.Update(ctx, svcPrincipal, "hook", enum.WebhookTypeExternal, svcRepoParent(),
			&types.WebhookUpdateInput{Enabled: boolp(false)})
		require.Error(t, err)
	})
	t.Run("internal webhook is not audited", func(t *testing.T) {
		h := newSvcHarness(t)
		h.hooks.add(&types.Webhook{
			ParentID: 1, ParentType: enum.WebhookParentRepo, Identifier: "int", Type: enum.WebhookTypeInternal,
		})
		_, err := h.svc.Update(ctx, svcPrincipal, "int", enum.WebhookTypeInternal, svcRepoParent(),
			&types.WebhookUpdateInput{Enabled: boolp(true)})
		require.NoError(t, err)
		assert.Empty(t, h.audit.actions)
	})
}

func TestSvcDelete(t *testing.T) {
	ctx := context.Background()

	t.Run("deletes by identifier", func(t *testing.T) {
		h := newSvcHarness(t)
		hook := h.addRepoHook()
		require.NoError(t, h.svc.Delete(ctx, svcPrincipal, "hook", svcRepoParent(), false))
		assert.Equal(t, []int64{hook.ID}, h.hooks.deleted)
		assert.Equal(t, []audit.Action{audit.ActionDeleted}, h.audit.actions)
		require.Len(t, h.sse.events, 1)
		assert.Equal(t, enum.SSETypeWebhookDeleted, h.sse.events[0].typ)
	})
	t.Run("deletes by numeric id", func(t *testing.T) {
		h := newSvcHarness(t)
		hook := h.addRepoHook()
		require.NoError(t, h.svc.Delete(ctx, svcPrincipal, "1", svcRepoParent(), false))
		assert.Equal(t, []int64{hook.ID}, h.hooks.deleted)
	})
	t.Run("internal webhook needs explicit permission", func(t *testing.T) {
		h := newSvcHarness(t)
		h.hooks.add(&types.Webhook{
			ParentID: 1, ParentType: enum.WebhookParentRepo, Identifier: "int", Type: enum.WebhookTypeInternal,
		})
		err := h.svc.Delete(ctx, svcPrincipal, "int", svcRepoParent(), false)
		require.ErrorIs(t, err, ErrInternalWebhookOperationNotAllowed)
		assert.Empty(t, h.hooks.deleted)

		require.NoError(t, h.svc.Delete(ctx, svcPrincipal, "int", svcRepoParent(), true))
		assert.Len(t, h.hooks.deleted, 1)
		assert.Empty(t, h.audit.actions, "internal hooks are not audited")
	})
	t.Run("unknown webhook", func(t *testing.T) {
		h := newSvcHarness(t)
		require.Error(t, h.svc.Delete(ctx, svcPrincipal, "nope", svcRepoParent(), false))
	})
	t.Run("store failure", func(t *testing.T) {
		h := newSvcHarness(t)
		h.addRepoHook()
		h.hooks.deleteErr = errors.New("db down")
		require.Error(t, h.svc.Delete(ctx, svcPrincipal, "hook", svcRepoParent(), false))
	})
}

func TestSvcGetWebhookVerifyOwnership(t *testing.T) {
	ctx := context.Background()
	h := newSvcHarness(t)
	hook := h.addRepoHook()

	tests := []struct {
		name       string
		identifier string
		parentID   int64
		parentType enum.WebhookParent
		wantStatus gitnesserrors.Status
		wantErr    bool
	}{
		{"by identifier", "hook", 1, enum.WebhookParentRepo, "", false},
		{"by id", "1", 1, enum.WebhookParentRepo, "", false},
		{"empty identifier", "  ", 1, enum.WebhookParentRepo, gitnesserrors.StatusInvalidArgument, true},
		{"zero id", "0", 1, enum.WebhookParentRepo, gitnesserrors.StatusInvalidArgument, true},
		{"negative id", "-3", 1, enum.WebhookParentRepo, gitnesserrors.StatusInvalidArgument, true},
		{"wrong parent id", "1", 2, enum.WebhookParentRepo, gitnesserrors.StatusNotFound, true},
		{"wrong parent type", "1", 1, enum.WebhookParentSpace, gitnesserrors.StatusNotFound, true},
		{"unknown identifier", "other", 1, enum.WebhookParentRepo, "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := h.svc.GetWebhookVerifyOwnership(ctx, tt.parentID, tt.parentType, tt.identifier)
			if tt.wantErr {
				require.Error(t, err)
				if tt.wantStatus != "" {
					assert.Equal(t, tt.wantStatus, gitnesserrors.AsStatus(err))
				}
				return
			}
			require.NoError(t, err)
			assert.Equal(t, hook.ID, got.ID)
		})
	}

	t.Run("Find wraps failures as not found", func(t *testing.T) {
		_, err := h.svc.Find(ctx, 1, enum.WebhookParentRepo, "missing")
		require.Error(t, err)
		assert.Equal(t, gitnesserrors.StatusNotFound, gitnesserrors.AsStatus(err))

		got, err := h.svc.Find(ctx, 1, enum.WebhookParentRepo, "hook")
		require.NoError(t, err)
		assert.Equal(t, hook.ID, got.ID)
	})
}

func TestSvcGetWebhookExecutionVerifyOwnership(t *testing.T) {
	ctx := context.Background()
	h := newSvcHarness(t)
	require.NoError(t, h.execs.Create(ctx, &types.WebhookExecution{WebhookID: 1}))

	_, err := h.svc.GetWebhookExecutionVerifyOwnership(ctx, 1, 0)
	assert.Equal(t, gitnesserrors.StatusInvalidArgument, gitnesserrors.AsStatus(err))

	_, err = h.svc.GetWebhookExecutionVerifyOwnership(ctx, 1, 99)
	require.Error(t, err)

	_, err = h.svc.GetWebhookExecutionVerifyOwnership(ctx, 2, 1)
	assert.Equal(t, gitnesserrors.StatusNotFound, gitnesserrors.AsStatus(err))

	e, err := h.svc.GetWebhookExecutionVerifyOwnership(ctx, 1, 1)
	require.NoError(t, err)
	assert.Equal(t, int64(1), e.ID)
}

func TestSvcList(t *testing.T) {
	ctx := context.Background()
	h := newSvcHarness(t)
	h.addRepoHook()
	h.hooks.add(&types.Webhook{ParentID: 10, ParentType: enum.WebhookParentSpace, Identifier: "space-hook"})
	h.hooks.add(&types.Webhook{ParentID: 11, ParentType: enum.WebhookParentSpace, Identifier: "root-hook"})
	h.hooks.add(&types.Webhook{ParentID: 50, ParentType: enum.WebhookParentSpace, Identifier: "unrelated"})

	tests := []struct {
		name       string
		parentID   int64
		parentType enum.WebhookParent
		inherited  bool
		wantCount  int64
		wantErr    bool
	}{
		{"repo only", 1, enum.WebhookParentRepo, false, 1, false},
		{"repo inherited", 1, enum.WebhookParentRepo, true, 3, false},
		{"space only", 10, enum.WebhookParentSpace, false, 1, false},
		{"space inherited", 10, enum.WebhookParentSpace, true, 2, false},
		{"registry parent has no scope filter", 1, enum.WebhookParentRegistry, false, 0, false},
		{"unsupported parent", 1, enum.WebhookParent("bogus"), false, 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hooks, count, err := h.svc.List(ctx, tt.parentID, tt.parentType, tt.inherited, &types.WebhookFilter{})
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantCount, count)
			assert.Len(t, hooks, int(tt.wantCount))
		})
	}

	t.Run("failures", func(t *testing.T) {
		fh := newSvcHarness(t)
		fh.repos.err = errors.New("db down")
		_, _, err := fh.svc.List(ctx, 1, enum.WebhookParentRepo, true, nil)
		require.Error(t, err)

		fh = newSvcHarness(t)
		fh.spaces.ancErr = errors.New("db down")
		_, _, err = fh.svc.List(ctx, 1, enum.WebhookParentRepo, true, nil)
		require.Error(t, err)
		_, _, err = fh.svc.List(ctx, 10, enum.WebhookParentSpace, true, nil)
		require.Error(t, err)

		fh = newSvcHarness(t)
		fh.hooks.countErr = errors.New("db down")
		_, _, err = fh.svc.List(ctx, 1, enum.WebhookParentRepo, false, nil)
		require.Error(t, err)

		fh = newSvcHarness(t)
		fh.hooks.listErr = errors.New("db down")
		_, _, err = fh.svc.List(ctx, 1, enum.WebhookParentRepo, false, nil)
		require.Error(t, err)
	})
}

func TestSvcExecutions(t *testing.T) {
	ctx := context.Background()
	h := newSvcHarness(t)
	hook := h.addRepoHook()
	h.hooks.add(&types.Webhook{
		ParentID: 1, ParentType: enum.WebhookParentRepo, Identifier: "other", Enabled: true, URL: h.hookURL,
	})
	require.NoError(t, h.execs.Create(ctx, &types.WebhookExecution{
		WebhookID: hook.ID, TriggerID: "t", TriggerType: enum.WebhookTriggerTagCreated, Retriggerable: true,
		Request: types.WebhookExecutionRequest{Body: `{"a":1}`},
	}))

	t.Run("find", func(t *testing.T) {
		e, err := h.svc.FindExecution(ctx, 1, enum.WebhookParentRepo, "hook", 1)
		require.NoError(t, err)
		assert.Equal(t, hook.ID, e.WebhookID)

		_, err = h.svc.FindExecution(ctx, 1, enum.WebhookParentRepo, "nope", 1)
		require.Error(t, err)
		_, err = h.svc.FindExecution(ctx, 1, enum.WebhookParentRepo, "other", 1)
		require.Error(t, err)
	})
	t.Run("list", func(t *testing.T) {
		list, total, err := h.svc.ListExecutions(ctx, 1, enum.WebhookParentRepo, "hook", &types.WebhookExecutionFilter{})
		require.NoError(t, err)
		assert.Equal(t, int64(1), total)
		assert.Len(t, list, 1)

		h.execs.countErr = errors.New("db down")
		_, _, err = h.svc.ListExecutions(ctx, 1, enum.WebhookParentRepo, "hook", nil)
		require.Error(t, err)
		h.execs.countErr = nil
		h.execs.listErr = errors.New("db down")
		_, _, err = h.svc.ListExecutions(ctx, 1, enum.WebhookParentRepo, "hook", nil)
		require.Error(t, err)
		h.execs.listErr = nil
	})
	t.Run("retrigger", func(t *testing.T) {
		exec, err := h.svc.RetriggerExecution(ctx, 1, enum.WebhookParentRepo, "hook", 1)
		require.NoError(t, err)
		assert.Equal(t, enum.WebhookExecutionResultSuccess, exec.Result)
		require.NotNil(t, exec.RetriggerOf)
		assert.Equal(t, int64(1), *exec.RetriggerOf)
		assert.EqualValues(t, 1, h.receiver.last(t).Body["a"])
	})
}

func TestSvcSendSSE(t *testing.T) {
	ctx := context.Background()
	h := newSvcHarness(t)
	hook := &types.Webhook{ID: 1}

	h.svc.sendSSE(ctx, svcSpaceParent(), enum.SSETypeWebhookCreated, hook)
	h.svc.sendSSE(ctx, svcRepoParent(), enum.SSETypeWebhookUpdated, hook)
	h.svc.sendSSE(ctx, ParentResource{ID: 99, Type: enum.WebhookParentRepo}, enum.SSETypeWebhookDeleted, hook)

	require.Len(t, h.sse.events, 2, "events for unknown repos are dropped")
	assert.Equal(t, int64(10), h.sse.events[0].spaceID)
	assert.Equal(t, int64(10), h.sse.events[1].spaceID)
}

func TestSvcShouldAuditAndAuditInfo(t *testing.T) {
	assert.True(t, shouldAuditWebhook(enum.WebhookTypeExternal))
	assert.False(t, shouldAuditWebhook(enum.WebhookTypeInternal))

	tests := []struct {
		parent  enum.WebhookParent
		wantRes audit.ResourceType
		wantKey string
	}{
		{enum.WebhookParentRepo, audit.ResourceTypeCodeWebhook, audit.RepoName},
		{enum.WebhookParentSpace, audit.ResourceTypeCodeWebhook, audit.SpaceName},
		{enum.WebhookParentRegistry, audit.ResourceTypeRegistryWebhook, audit.RegistryName},
		{enum.WebhookParent("other"), "", ""},
	}
	for _, tt := range tests {
		res, key := getWebhookAuditInfo(tt.parent)
		assert.Equal(t, tt.wantRes, res)
		assert.Equal(t, tt.wantKey, key)
	}
}

func TestSvcCommonHelpers(t *testing.T) {
	t.Run("CheckSecret", func(t *testing.T) {
		require.NoError(t, CheckSecret(""))
		require.Error(t, CheckSecret(string(make([]byte, webhookMaxSecretLength+1))))
	})
	t.Run("CheckTriggers", func(t *testing.T) {
		require.NoError(t, CheckTriggers(nil))
		require.NoError(t, CheckTriggers([]enum.WebhookTrigger{enum.WebhookTriggerBranchCreated}))
		err := CheckTriggers([]enum.WebhookTrigger{enum.WebhookTriggerBranchCreated, "bogus"})
		require.Error(t, err)
		var verr *check.ValidationError
		assert.True(t, errors.As(err, &verr))
	})
	t.Run("CheckExtraHeaders", func(t *testing.T) {
		require.NoError(t, CheckExtraHeaders(nil))
		require.NoError(t, CheckExtraHeaders([]types.ExtraHeader{{Key: "A"}, {Key: "B"}}))
		require.Error(t, CheckExtraHeaders([]types.ExtraHeader{{Key: ""}}))
		require.Error(t, CheckExtraHeaders(make([]types.ExtraHeader, webhookMaxExtraHeaders+1)))
	})
	t.Run("DeduplicateTriggers", func(t *testing.T) {
		assert.Equal(t, []enum.WebhookTrigger{}, DeduplicateTriggers(nil))
		in := []enum.WebhookTrigger{
			enum.WebhookTriggerTagCreated, enum.WebhookTriggerBranchCreated, enum.WebhookTriggerTagCreated,
		}
		assert.Equal(t,
			[]enum.WebhookTrigger{enum.WebhookTriggerTagCreated, enum.WebhookTriggerBranchCreated},
			DeduplicateTriggers(in))
	})
	t.Run("ConvertTriggers", func(t *testing.T) {
		assert.Equal(t,
			[]enum.WebhookTrigger{enum.WebhookTriggerTagCreated, "custom"},
			ConvertTriggers([]string{"tag_created", "custom"}))
		assert.Empty(t, ConvertTriggers(nil))
	})
}
