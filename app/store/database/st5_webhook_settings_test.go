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

package database_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/harness/gitness/app/store/database"
	gitness_store "github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/require"
)

const st5UnsupportedScope = enum.SettingsScope("st5-unsupported")

func st5NewWebhook(identifier string, parent enum.WebhookParent, parentID int64) *types.Webhook {
	return &types.Webhook{
		ParentType: parent, ParentID: parentID, CreatedBy: userID, Created: 1, Updated: 1,
		Identifier: identifier, DisplayName: identifier, URL: "http://localhost/hook",
		Enabled: true, Triggers: []enum.WebhookTrigger{enum.WebhookTriggerBranchCreated, enum.WebhookTriggerBranchUpdated},
	}
}

//nolint:funlen
func TestSt5WebhookStore(t *testing.T) {
	ctx := context.Background()
	env := st5NewEnv(ctx, t)
	s := database.NewWebhookStore(env.db)
	repoID := env.repoIDs[0]

	repoHook := st5NewWebhook("Repo-Hook", enum.WebhookParentRepo, repoID)
	repoHook.ExtraHeaders = []types.ExtraHeader{{Key: "X-Token", Value: "value", Masked: true}}
	require.NoError(t, s.Create(ctx, repoHook))
	spaceHook := st5NewWebhook("space-hook", enum.WebhookParentSpace, 1)
	spaceHook.Type = enum.WebhookTypeInternal
	spaceHook.Triggers = nil
	require.NoError(t, s.Create(ctx, spaceHook))

	t.Run("find maps parents, triggers and headers", func(t *testing.T) {
		got, err := s.Find(ctx, repoHook.ID)
		require.NoError(t, err)
		require.Equal(t, enum.WebhookParentRepo, got.ParentType)
		require.Equal(t, repoID, got.ParentID)
		require.Equal(t, repoHook.Triggers, got.Triggers)
		require.Equal(t, repoHook.ExtraHeaders, got.ExtraHeaders)

		got, err = s.Find(ctx, spaceHook.ID)
		require.NoError(t, err)
		require.Equal(t, enum.WebhookParentSpace, got.ParentType)
		require.Empty(t, got.Triggers)
		require.Empty(t, got.ExtraHeaders)
	})

	t.Run("find by identifier is case insensitive", func(t *testing.T) {
		got, err := s.FindByIdentifier(ctx, enum.WebhookParentRepo, repoID, "REPO-hook")
		require.NoError(t, err)
		require.Equal(t, repoHook.ID, got.ID)

		got, err = s.FindByIdentifier(ctx, enum.WebhookParentSpace, 1, "space-hook")
		require.NoError(t, err)
		require.Equal(t, spaceHook.ID, got.ID)

		_, err = s.FindByIdentifier(ctx, enum.WebhookParentRepo, repoID, "missing")
		require.ErrorIs(t, err, gitness_store.ErrResourceNotFound)

		_, err = s.FindByIdentifier(ctx, enum.WebhookParent("st5"), 1, "x")
		require.ErrorContains(t, err, "not supported")
	})

	t.Run("registry parent is stored without repo or space", func(t *testing.T) {
		registryHook := st5NewWebhook("registry-hook", enum.WebhookParentRegistry, 0)
		require.NoError(t, s.Create(ctx, registryHook))

		// the row carries neither a repo nor a space and cannot be mapped back.
		_, err := s.FindByIdentifier(ctx, enum.WebhookParentRegistry, 0, "registry-hook")
		require.ErrorContains(t, err, "neither repoID nor spaceID")
		_, err = s.Find(ctx, registryHook.ID)
		require.ErrorContains(t, err, "neither repoID nor spaceID")

		require.NoError(t, s.DeleteByIdentifier(ctx, enum.WebhookParentRegistry, 0, "registry-hook"))
	})

	t.Run("row with repo and space cannot be mapped", func(t *testing.T) {
		_, execErr := env.db.ExecContext(ctx, `INSERT INTO webhooks (webhook_created_by, webhook_created, webhook_updated,
			webhook_space_id, webhook_repo_id, webhook_display_name, webhook_description, webhook_url, webhook_secret,
			webhook_enabled, webhook_insecure, webhook_triggers, webhook_uid)
			VALUES (1, 1, 1, 1, ?, 'both', '', 'http://x', '', 1, 0, '', 'both-parents')`, repoID)
		require.NoError(t, execErr)
		_, err := s.FindByIdentifier(ctx, enum.WebhookParentRepo, repoID, "both-parents")
		require.ErrorContains(t, err, "both repoID and spaceID")

		_, err = s.List(ctx, []types.WebhookParentInfo{{Type: enum.WebhookParentRepo, ID: repoID}},
			&types.WebhookFilter{Order: enum.OrderAsc})
		require.ErrorContains(t, err, "both repoID and spaceID")

		_, execErr = env.db.ExecContext(ctx, `DELETE FROM webhooks WHERE webhook_uid = 'both-parents'`)
		require.NoError(t, execErr)
	})

	t.Run("create with unsupported parent fails", func(t *testing.T) {
		err := s.Create(ctx, st5NewWebhook("x", enum.WebhookParent("st5"), 1))
		require.ErrorContains(t, err, "not supported")
		err = s.Update(ctx, st5NewWebhook("x", enum.WebhookParent("st5"), 1))
		require.ErrorContains(t, err, "not supported")
	})

	t.Run("update and opt lock", func(t *testing.T) {
		got, err := s.Find(ctx, repoHook.ID)
		require.NoError(t, err)
		stale := *got

		got.Description = "updated"
		require.NoError(t, s.Update(ctx, got))
		require.EqualValues(t, 1, got.Version)
		require.ErrorIs(t, s.Update(ctx, &stale), gitness_store.ErrVersionConflict)

		calls := 0
		res, err := s.UpdateOptLock(ctx, &stale, func(h *types.Webhook) error {
			calls++
			h.Enabled = false

			return nil
		})
		require.NoError(t, err)
		require.Equal(t, 2, calls)
		require.False(t, res.Enabled)
		require.Equal(t, "updated", res.Description)

		_, err = s.UpdateOptLock(ctx, got, func(*types.Webhook) error { return errSt5Cache })
		require.ErrorIs(t, err, errSt5Cache)

		_, err = s.UpdateOptLock(ctx, st5NewWebhook("x", enum.WebhookParent("st5"), 1),
			func(*types.Webhook) error { return nil })
		require.ErrorContains(t, err, "failed to update the webhook")

		_, err = s.UpdateOptLock(ctx, &types.Webhook{ID: 9999, Version: 3, ParentType: enum.WebhookParentRepo, ParentID: 1},
			func(*types.Webhook) error { return nil })
		require.ErrorContains(t, err, "failed to find the latst version")
	})

	t.Run("count and list", func(t *testing.T) {
		parents := []types.WebhookParentInfo{
			{Type: enum.WebhookParentRepo, ID: repoID},
			{Type: enum.WebhookParentSpace, ID: 1},
		}

		n, err := s.Count(ctx, parents, &types.WebhookFilter{})
		require.NoError(t, err)
		require.EqualValues(t, 2, n)

		n, err = s.Count(ctx, parents, &types.WebhookFilter{SkipInternal: true})
		require.NoError(t, err)
		require.EqualValues(t, 1, n)

		n, err = s.Count(ctx, parents, &types.WebhookFilter{Query: "SPACE"})
		require.NoError(t, err)
		require.EqualValues(t, 1, n)

		_, err = s.Count(ctx, []types.WebhookParentInfo{{Type: enum.WebhookParent("st5")}}, &types.WebhookFilter{})
		require.ErrorContains(t, err, "failed to select webhook parents")
		_, err = s.List(ctx, []types.WebhookParentInfo{{Type: enum.WebhookParent("st5")}}, &types.WebhookFilter{})
		require.ErrorContains(t, err, "failed to select webhook parents")

		for _, sort := range []enum.WebhookAttr{
			enum.WebhookAttrNone, enum.WebhookAttrID, enum.WebhookAttrUID, enum.WebhookAttrIdentifier,
			enum.WebhookAttrDisplayName, enum.WebhookAttrCreated, enum.WebhookAttrUpdated,
		} {
			list, listErr := s.List(ctx, []types.WebhookParentInfo{{Type: enum.WebhookParentSpace, ID: 1}},
				&types.WebhookFilter{Sort: sort, Order: enum.OrderDesc, Page: 1, Size: 10})
			require.NoError(t, listErr, sort)
			require.Len(t, list, 1)
		}
	})

	t.Run("update parent space", func(t *testing.T) {
		n, err := s.UpdateParentSpace(ctx, 999, 1)
		require.NoError(t, err)
		require.EqualValues(t, 0, n)
	})

	t.Run("delete", func(t *testing.T) {
		require.ErrorContains(t,
			s.DeleteByIdentifier(ctx, enum.WebhookParent("st5"), 1, "x"), "not supported")
		require.NoError(t, s.DeleteByIdentifier(ctx, enum.WebhookParentSpace, 1, "space-hook"))
		_, err := s.Find(ctx, spaceHook.ID)
		require.ErrorIs(t, err, gitness_store.ErrResourceNotFound)

		require.NoError(t, s.Delete(ctx, repoHook.ID))
		_, err = s.Find(ctx, repoHook.ID)
		require.ErrorIs(t, err, gitness_store.ErrResourceNotFound)
	})
}

//nolint:funlen
func TestSt5SettingsStore(t *testing.T) {
	ctx := context.Background()
	env := st5NewEnv(ctx, t)
	s := database.NewSettingsStore(env.db)

	scopes := []struct {
		scope enum.SettingsScope
		id    int64
	}{
		{enum.SettingsScopeRepo, env.repoIDs[0]},
		{enum.SettingsScopeSpace, 1},
		{enum.SettingsScopeSystem, 0},
	}

	for _, sc := range scopes {
		t.Run(string(sc.scope), func(t *testing.T) {
			require.NoError(t, s.Upsert(ctx, sc.scope, sc.id, "Feature.Flag", json.RawMessage(`true`)))
			require.NoError(t, s.Upsert(ctx, sc.scope, sc.id, "other", json.RawMessage(`{"a":1}`)))

			got, err := s.Find(ctx, sc.scope, sc.id, "feature.flag")
			require.NoError(t, err)
			require.JSONEq(t, `true`, string(got))

			// upserting the same key replaces the value, also with a different casing.
			require.NoError(t, s.Upsert(ctx, sc.scope, sc.id, "feature.FLAG", json.RawMessage(`false`)))
			got, err = s.Find(ctx, sc.scope, sc.id, "Feature.Flag")
			require.NoError(t, err)
			require.JSONEq(t, `false`, string(got))

			many, err := s.FindMany(ctx, sc.scope, sc.id, "FEATURE.flag", "other", "unknown")
			require.NoError(t, err)
			require.Len(t, many, 2)

			empty, err := s.FindMany(ctx, sc.scope, sc.id)
			require.NoError(t, err)
			require.Empty(t, empty)

			_, err = s.Find(ctx, sc.scope, sc.id, "unknown")
			require.ErrorIs(t, err, gitness_store.ErrResourceNotFound)

			require.NoError(t, s.Delete(ctx, sc.scope, sc.id, "FEATURE.flag"))
			_, err = s.Find(ctx, sc.scope, sc.id, "feature.flag")
			require.ErrorIs(t, err, gitness_store.ErrResourceNotFound)

			require.NoError(t, s.DeleteMany(ctx, sc.scope, sc.id))
			require.NoError(t, s.DeleteMany(ctx, sc.scope, sc.id, "other", "unknown"))
			_, err = s.Find(ctx, sc.scope, sc.id, "other")
			require.ErrorIs(t, err, gitness_store.ErrResourceNotFound)
		})
	}

	t.Run("unsupported scope", func(t *testing.T) {
		_, err := s.Find(ctx, st5UnsupportedScope, 1, "k")
		require.ErrorContains(t, err, "not supported")
		_, err = s.FindMany(ctx, st5UnsupportedScope, 1, "k")
		require.ErrorContains(t, err, "not supported")
		require.ErrorContains(t, s.Upsert(ctx, st5UnsupportedScope, 1, "k", json.RawMessage(`1`)), "not supported")
		require.ErrorContains(t, s.Delete(ctx, st5UnsupportedScope, 1, "k"), "not supported")
	})
}
