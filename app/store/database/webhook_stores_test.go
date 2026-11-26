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

package database_test

import (
	"testing"
	"time"

	"github.com/harness/gitness/app/store/database"
	gitnessstore "github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/require"
)

func newWebhook(parentType enum.WebhookParent, parentID int64, uid string) *types.Webhook {
	return &types.Webhook{
		ParentType: parentType, ParentID: parentID, CreatedBy: userID, Created: 1, Updated: 1,
		Identifier: uid, DisplayName: "Hook " + uid, Description: "d", URL: "https://example.com/" + uid,
		Secret: "s3cret", Enabled: true, Insecure: false, Scope: 0,
		Triggers:     []enum.WebhookTrigger{enum.WebhookTriggerBranchCreated, enum.WebhookTriggerTagCreated},
		ExtraHeaders: []types.ExtraHeader{{Key: "X-Token", Value: "v", Masked: true}},
	}
}

func TestWebhookStore(t *testing.T) {
	ctx, f := newScopedFixture(t)
	ws := database.NewWebhookStore(f.db)

	repoHook := newWebhook(enum.WebhookParentRepo, 1, "repo-hook")
	spaceHook := newWebhook(enum.WebhookParentSpace, 1, "space-hook")
	otherHook := newWebhook(enum.WebhookParentSpace, 2, "other-hook")
	for _, h := range []*types.Webhook{repoHook, spaceHook, otherHook} {
		require.NoError(t, ws.Create(ctx, h))
		require.NotZero(t, h.ID)
	}

	t.Run("duplicate identifier per parent", func(t *testing.T) {
		require.ErrorIs(t, ws.Create(ctx, newWebhook(enum.WebhookParentRepo, 1, "REPO-HOOK")), gitnessstore.ErrDuplicate)
		// The same identifier is fine below a different parent.
		require.NoError(t, ws.Create(ctx, newWebhook(enum.WebhookParentSpace, 2, "repo-hook")))
	})

	t.Run("find", func(t *testing.T) {
		got, err := ws.Find(ctx, repoHook.ID)
		require.NoError(t, err)
		require.Equal(t, "repo-hook", got.Identifier)
		require.Equal(t, enum.WebhookParentRepo, got.ParentType)
		require.Equal(t, int64(1), got.ParentID)
		require.Equal(t, "s3cret", got.Secret)
		require.ElementsMatch(t, repoHook.Triggers, got.Triggers)
		require.Equal(t, repoHook.ExtraHeaders, got.ExtraHeaders)

		byIdentifier, err := ws.FindByIdentifier(ctx, enum.WebhookParentSpace, 1, "SPACE-HOOK")
		require.NoError(t, err)
		require.Equal(t, spaceHook.ID, byIdentifier.ID)

		byRepo, err := ws.FindByIdentifier(ctx, enum.WebhookParentRepo, 1, "repo-hook")
		require.NoError(t, err)
		require.Equal(t, repoHook.ID, byRepo.ID)

		_, err = ws.Find(ctx, 9999)
		require.ErrorIs(t, err, gitnessstore.ErrResourceNotFound)
		_, err = ws.FindByIdentifier(ctx, enum.WebhookParentRepo, 1, "space-hook")
		require.ErrorIs(t, err, gitnessstore.ErrResourceNotFound)
		_, err = ws.FindByIdentifier(ctx, enum.WebhookParent("bogus"), 1, "x")
		require.Error(t, err)
	})

	t.Run("update and optimistic locking", func(t *testing.T) {
		stale := *repoHook

		repoHook.Enabled = false
		repoHook.URL = "https://example.com/changed"
		require.NoError(t, ws.Update(ctx, repoHook))
		require.Equal(t, int64(1), repoHook.Version)

		got, err := ws.Find(ctx, repoHook.ID)
		require.NoError(t, err)
		require.False(t, got.Enabled)
		require.Equal(t, "https://example.com/changed", got.URL)

		require.ErrorIs(t, ws.Update(ctx, &stale), gitnessstore.ErrVersionConflict)

		calls := 0
		updated, err := ws.UpdateOptLock(ctx, &stale, func(h *types.Webhook) error {
			calls++
			h.Description = "locked"
			return nil
		})
		require.NoError(t, err)
		require.Equal(t, 2, calls)
		require.Equal(t, "locked", updated.Description)
		require.Equal(t, "https://example.com/changed", updated.URL, "the retry starts from the stored webhook")

		_, err = ws.UpdateOptLock(ctx, updated, func(*types.Webhook) error { return errBoom })
		require.ErrorIs(t, err, errBoom)
	})

	t.Run("list and count by parent", func(t *testing.T) {
		spaceParent := []types.WebhookParentInfo{{Type: enum.WebhookParentSpace, ID: 1}}
		both := []types.WebhookParentInfo{
			{Type: enum.WebhookParentSpace, ID: 1},
			{Type: enum.WebhookParentRepo, ID: 1},
		}
		filter := &types.WebhookFilter{Page: 1, Size: 10}

		list, err := ws.List(ctx, spaceParent, filter)
		require.NoError(t, err)
		require.Len(t, list, 1)

		list, err = ws.List(ctx, both, filter)
		require.NoError(t, err)
		require.Len(t, list, 2)

		count, err := ws.Count(ctx, both, filter)
		require.NoError(t, err)
		require.Equal(t, int64(2), count)

		searched, err := ws.List(ctx, both, &types.WebhookFilter{Page: 1, Size: 10, Query: "space"})
		require.NoError(t, err)
		require.Len(t, searched, 1)
		count, err = ws.Count(ctx, both, &types.WebhookFilter{Query: "space"})
		require.NoError(t, err)
		require.Equal(t, int64(1), count)

		_, err = ws.List(ctx, []types.WebhookParentInfo{{Type: "bogus", ID: 1}}, filter)
		require.Error(t, err)
		_, err = ws.Count(ctx, []types.WebhookParentInfo{{Type: "bogus", ID: 1}}, filter)
		require.Error(t, err)
	})

	t.Run("list sorting", func(t *testing.T) {
		parents := []types.WebhookParentInfo{{Type: enum.WebhookParentSpace, ID: 2}}
		a := newWebhook(enum.WebhookParentSpace, 2, "aaa")
		a.Created, a.Updated = 50, 50
		require.NoError(t, ws.Create(ctx, a))

		for _, sortAttr := range []enum.WebhookAttr{
			enum.WebhookAttrID, enum.WebhookAttrIdentifier, enum.WebhookAttrDisplayName,
			enum.WebhookAttrCreated, enum.WebhookAttrUpdated, enum.WebhookAttrNone,
		} {
			list, err := ws.List(ctx, parents, &types.WebhookFilter{Page: 1, Size: 10, Sort: sortAttr, Order: enum.OrderDesc})
			require.NoError(t, err, sortAttr.String())
			require.NotEmpty(t, list, sortAttr.String())
		}

		byIdentifier, err := ws.List(ctx, parents, &types.WebhookFilter{
			Page: 1, Size: 10, Sort: enum.WebhookAttrIdentifier, Order: enum.OrderAsc,
		})
		require.NoError(t, err)
		require.Equal(t, "aaa", byIdentifier[0].Identifier)
	})

	t.Run("move to another parent space", func(t *testing.T) {
		moved, err := ws.UpdateParentSpace(ctx, 1, 2)
		require.NoError(t, err)
		require.Equal(t, int64(1), moved)
	})

	t.Run("delete", func(t *testing.T) {
		require.NoError(t, ws.Delete(ctx, repoHook.ID))
		_, err := ws.Find(ctx, repoHook.ID)
		require.ErrorIs(t, err, gitnessstore.ErrResourceNotFound)

		require.NoError(t, ws.DeleteByIdentifier(ctx, enum.WebhookParentSpace, 2, "OTHER-HOOK"))
		_, err = ws.Find(ctx, otherHook.ID)
		require.ErrorIs(t, err, gitnessstore.ErrResourceNotFound)

		require.Error(t, ws.DeleteByIdentifier(ctx, enum.WebhookParent("bogus"), 1, "x"))
	})
}

func TestWebhookExecutionStore(t *testing.T) {
	ctx, f := newScopedFixture(t)
	ws := database.NewWebhookStore(f.db)
	es := database.NewWebhookExecutionStore(f.db)

	hook := newWebhook(enum.WebhookParentRepo, 1, "hook")
	require.NoError(t, ws.Create(ctx, hook))

	now := time.Now()
	newExecution := func(triggerID string, created int64, result enum.WebhookExecutionResult) *types.WebhookExecution {
		return &types.WebhookExecution{
			WebhookID: hook.ID, TriggerType: enum.WebhookTriggerBranchCreated, TriggerID: triggerID,
			Created: created, Result: result, Duration: 12, Error: "",
			Retriggerable: true,
			Request:       types.WebhookExecutionRequest{URL: hook.URL, Headers: "h", Body: "{}"},
			Response:      types.WebhookExecutionResponse{StatusCode: 200, Status: "200 OK", Headers: "rh", Body: "ok"},
		}
	}

	old := newExecution("trigger-1", now.Add(-48*time.Hour).UnixMilli(), enum.WebhookExecutionResultSuccess)
	recent := newExecution("trigger-1", now.UnixMilli(), enum.WebhookExecutionResultRetriableError)
	other := newExecution("trigger-2", now.UnixMilli(), enum.WebhookExecutionResultFatalError)
	for _, e := range []*types.WebhookExecution{old, recent, other} {
		require.NoError(t, es.Create(ctx, e))
		require.NotZero(t, e.ID)
	}

	got, err := es.Find(ctx, recent.ID)
	require.NoError(t, err)
	require.Equal(t, enum.WebhookExecutionResultRetriableError, got.Result)
	require.Equal(t, 200, got.Response.StatusCode)
	require.Equal(t, "{}", got.Request.Body)
	require.True(t, got.Retriggerable)
	require.Nil(t, got.RetriggerOf)

	_, err = es.Find(ctx, 9999)
	require.ErrorIs(t, err, gitnessstore.ErrResourceNotFound)

	retrigger := newExecution("trigger-3", now.UnixMilli(), enum.WebhookExecutionResultSuccess)
	retrigger.RetriggerOf = &old.ID
	require.NoError(t, es.Create(ctx, retrigger))
	got, err = es.Find(ctx, retrigger.ID)
	require.NoError(t, err)
	require.Equal(t, old.ID, *got.RetriggerOf)

	list, err := es.ListForWebhook(ctx, hook.ID, &types.WebhookExecutionFilter{Page: 1, Size: 2})
	require.NoError(t, err)
	require.Len(t, list, 2)
	count, err := es.CountForWebhook(ctx, hook.ID)
	require.NoError(t, err)
	require.Equal(t, int64(4), count)

	byTrigger, err := es.ListForTrigger(ctx, "trigger-1")
	require.NoError(t, err)
	require.Len(t, byTrigger, 2)
	none, err := es.ListForTrigger(ctx, "missing")
	require.NoError(t, err)
	require.Empty(t, none)

	deleted, err := es.DeleteOld(ctx, now.Add(-24*time.Hour))
	require.NoError(t, err)
	require.Equal(t, int64(1), deleted)
	count, err = es.CountForWebhook(ctx, hook.ID)
	require.NoError(t, err)
	require.Equal(t, int64(3), count)
}
