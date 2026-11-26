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
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/harness/gitness/app/store"
	appcache "github.com/harness/gitness/app/store/cache"
	"github.com/harness/gitness/app/store/database"
	"github.com/harness/gitness/cache"
	gitness_store "github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/require"
)

// rstCDEFixture adds the cloud development environment stores on top of rstFixture.
type rstCDEFixture struct {
	*rstFixture

	spaceID      int64
	spacePath    string
	configs      store.InfraProviderConfigStore
	resources    store.InfraProviderResourceStore
	templates    store.InfraProviderTemplateStore
	provisioned  store.InfraProvisionedStore
	gitspaces    store.GitspaceConfigStore
	instances    store.GitspaceInstanceStore
	events       store.GitspaceEventStore
	settings     store.GitspaceSettingsStore
	gateways     *database.CDEGatewayStore
	nextInstance int
}

func rstNewCDEFixture(t *testing.T) (context.Context, *rstCDEFixture) {
	t.Helper()

	ctx, base := rstNewFixture(t)
	space := base.rstMkSpace(ctx, t, 0, "cde")

	evictor := appcache.NewEvictor[*types.SpaceCore]("rst-ns", "rst-topic", nil)
	spaceIDCache := appcache.NewSpaceIDCache(ctx, base.spaces, evictor, time.Minute)
	principalCache := cache.NewExtended[int64, *types.PrincipalInfo](
		database.NewPrincipalInfoView(base.db), time.Minute)
	resourceCache := cache.NewExtended[int64, *types.InfraProviderResource](
		database.NewInfraProviderResourceView(base.db, base.spaces), time.Minute)

	return ctx, &rstCDEFixture{
		rstFixture:  base,
		spaceID:     space.ID,
		spacePath:   "cde",
		configs:     database.NewInfraProviderConfigStore(base.db, spaceIDCache),
		resources:   database.NewInfraProviderResourceStore(base.db, spaceIDCache),
		templates:   database.NewInfraProviderTemplateStore(base.db),
		provisioned: database.NewInfraProvisionedStore(base.db),
		gitspaces:   database.NewGitspaceConfigStore(base.db, principalCache, resourceCache, spaceIDCache),
		instances:   database.NewGitspaceInstanceStore(base.db, spaceIDCache),
		events:      database.NewGitspaceEventStore(base.db),
		settings:    database.NewGitspaceSettingsStore(base.db),
		gateways:    database.NewCDEGatewayStore(base.db),
	}
}

func rstStrPtr(s string) *string { return &s }

func (f *rstCDEFixture) rstMkInfraConfig(
	ctx context.Context, t *testing.T, ident string, infraType enum.InfraProviderType,
) *types.InfraProviderConfig {
	t.Helper()

	cfg := &types.InfraProviderConfig{
		Identifier: ident, Name: "name " + ident, Type: infraType, SpaceID: f.spaceID,
		Metadata: map[string]any{"region": "eu"}, Created: 10, Updated: 10,
	}
	require.NoError(t, f.configs.Create(ctx, cfg))

	return cfg
}

func (f *rstCDEFixture) rstMkInfraResource(
	ctx context.Context, t *testing.T, cfgID int64, ident string, created int64,
) *types.InfraProviderResource {
	t.Helper()

	res := &types.InfraProviderResource{
		UID: ident, Name: "name " + ident, InfraProviderConfigID: cfgID, InfraProviderType: enum.InfraProviderTypeDocker,
		SpaceID: f.spaceID, CPU: rstStrPtr("2"), Memory: rstStrPtr("4GB"), Disk: rstStrPtr("10GB"),
		Region: "eu", Metadata: map[string]string{"k": "v"}, Created: created, Updated: created,
	}
	require.NoError(t, f.resources.Create(ctx, res))

	return res
}

func (f *rstCDEFixture) rstMkGitspace(
	ctx context.Context, t *testing.T, ident string, resourceID int64,
) *types.GitspaceConfig {
	t.Helper()

	uid := userID
	cfg := &types.GitspaceConfig{
		Identifier: ident, Name: "name " + ident, IDE: enum.IDETypeVSCode, SpaceID: f.spaceID,
		Created: 20, Updated: 20, SSHTokenIdentifier: "tok", AIAgents: []enum.AIAgent{enum.AIAgentClaudeCode},
		InfraProviderResource: types.InfraProviderResource{ID: resourceID},
		CodeRepo: types.CodeRepo{
			URL: "https://example.com/" + ident, Type: enum.CodeRepoTypeGithub, Branch: "main",
			AuthType: "anonymous", AuthID: "auth", DevcontainerPath: rstStrPtr(".devcontainer"),
		},
		GitspaceUser: types.GitspaceUser{ID: &uid, Identifier: "user_1"},
	}
	require.NoError(t, f.gitspaces.Create(ctx, cfg))

	return cfg
}

func (f *rstCDEFixture) rstMkInstance(
	ctx context.Context, t *testing.T, cfgID int64, state enum.GitspaceInstanceStateType, created int64,
) *types.GitspaceInstance {
	t.Helper()

	f.nextInstance++
	inst := &types.GitspaceInstance{
		GitSpaceConfigID: cfgID, Identifier: fmt.Sprintf("inst-%d", f.nextInstance), State: state, UserID: "user_1",
		SpaceID: f.spaceID, Created: created, Updated: created, AccessType: enum.GitspaceAccessTypeSSHKey,
	}
	require.NoError(t, f.instances.Create(ctx, inst))

	return inst
}

func TestRstInfraProviderConfigStore(t *testing.T) {
	ctx, f := rstNewCDEFixture(t)

	cfg := f.rstMkInfraConfig(ctx, t, "docker-cfg", enum.InfraProviderTypeDocker)
	other := f.rstMkInfraConfig(ctx, t, "gcp-cfg", enum.InfraProviderTypeHarnessGCP)

	t.Run("duplicate identifier", func(t *testing.T) {
		err := f.configs.Create(ctx, &types.InfraProviderConfig{
			Identifier: "docker-cfg", Name: "dup", Type: enum.InfraProviderTypeDocker, SpaceID: f.spaceID,
			Created: 10, Updated: 10,
		})
		require.ErrorIs(t, err, gitness_store.ErrDuplicate)
	})

	t.Run("find", func(t *testing.T) {
		got, err := f.configs.Find(ctx, cfg.ID, false)
		require.NoError(t, err)
		require.Equal(t, "docker-cfg", got.Identifier)
		require.Equal(t, f.spacePath, got.SpacePath)
		require.Equal(t, map[string]any{"region": "eu"}, got.Metadata)

		_, err = f.configs.Find(ctx, 9999, false)
		require.ErrorIs(t, err, gitness_store.ErrResourceNotFound)
	})

	t.Run("find by identifier and type", func(t *testing.T) {
		got, err := f.configs.FindByIdentifier(ctx, f.spaceID, "gcp-cfg")
		require.NoError(t, err)
		require.Equal(t, other.ID, got.ID)

		_, err = f.configs.FindByIdentifier(ctx, f.spaceID, "missing")
		require.ErrorIs(t, err, gitness_store.ErrResourceNotFound)

		got, err = f.configs.FindByType(ctx, f.spaceID, enum.InfraProviderTypeHarnessGCP, false)
		require.NoError(t, err)
		require.Equal(t, other.ID, got.ID)
	})

	t.Run("list with filters", func(t *testing.T) {
		all, err := f.configs.List(ctx, nil)
		require.NoError(t, err)
		require.Len(t, all, 2)

		byType, err := f.configs.List(ctx, &types.InfraProviderConfigFilter{Type: enum.InfraProviderTypeDocker})
		require.NoError(t, err)
		require.Len(t, byType, 1)
		require.Equal(t, cfg.ID, byType[0].ID)

		bySpace, err := f.configs.List(ctx, &types.InfraProviderConfigFilter{SpaceIDs: []int64{9999}})
		require.NoError(t, err)
		require.Empty(t, bySpace)
	})

	t.Run("update", func(t *testing.T) {
		cfg.Name = rstValRenamed
		cfg.Updated = 99
		cfg.Metadata = map[string]any{rstValChanged: true}
		require.NoError(t, f.configs.Update(ctx, cfg))

		got, err := f.configs.Find(ctx, cfg.ID, false)
		require.NoError(t, err)
		require.Equal(t, rstValRenamed, got.Name)
		require.Equal(t, int64(99), got.Updated)
		require.Equal(t, map[string]any{rstValChanged: true}, got.Metadata)
	})

	t.Run("soft delete", func(t *testing.T) {
		require.NoError(t, f.configs.Delete(ctx, other.ID))

		_, err := f.configs.Find(ctx, other.ID, false)
		require.ErrorIs(t, err, gitness_store.ErrResourceNotFound)

		got, err := f.configs.Find(ctx, other.ID, true)
		require.NoError(t, err)
		require.True(t, got.IsDeleted)
		require.NotNil(t, got.Deleted)

		active, err := f.configs.List(ctx, nil)
		require.NoError(t, err)
		require.Len(t, active, 1)
	})
}

func TestRstInfraProviderResourceStore(t *testing.T) {
	ctx, f := rstNewCDEFixture(t)
	cfg := f.rstMkInfraConfig(ctx, t, "docker-cfg", enum.InfraProviderTypeDocker)

	res := f.rstMkInfraResource(ctx, t, cfg.ID, "small", 100)
	f.rstMkInfraResource(ctx, t, cfg.ID, "large", 200)

	t.Run("find", func(t *testing.T) {
		got, err := f.resources.Find(ctx, res.ID)
		require.NoError(t, err)
		require.Equal(t, "small", got.UID)
		require.Equal(t, map[string]string{"k": "v"}, got.Metadata)
		require.Equal(t, "2", *got.CPU)
		require.Equal(t, f.spacePath, got.SpacePath)

		_, err = f.resources.Find(ctx, 9999)
		require.ErrorIs(t, err, gitness_store.ErrResourceNotFound)
	})

	t.Run("find by config and identifier", func(t *testing.T) {
		got, err := f.resources.FindByConfigAndIdentifier(ctx, f.spaceID, cfg.ID, "large")
		require.NoError(t, err)
		require.Equal(t, "large", got.UID)

		_, err = f.resources.FindByConfigAndIdentifier(ctx, f.spaceID, cfg.ID, "none")
		require.ErrorIs(t, err, gitness_store.ErrResourceNotFound)
	})

	t.Run("list", func(t *testing.T) {
		got, err := f.resources.List(ctx, cfg.ID, types.ListQueryFilter{})
		require.NoError(t, err)
		require.Len(t, got, 2)
		// ordered by identifier
		require.Equal(t, "large", got[0].UID)
		require.Equal(t, "small", got[1].UID)
	})

	t.Run("update", func(t *testing.T) {
		cpu := "8"
		res.Name = rstValRenamed
		res.CPU = &cpu
		res.Region = "us"
		res.Metadata = map[string]string{rstValChanged: "yes"}
		require.NoError(t, f.resources.Update(ctx, res))
		require.NotZero(t, res.Updated)

		got, err := f.resources.Find(ctx, res.ID)
		require.NoError(t, err)
		require.Equal(t, rstValRenamed, got.Name)
		require.Equal(t, "8", *got.CPU)
		require.Equal(t, "us", got.Region)
		require.Equal(t, map[string]string{rstValChanged: "yes"}, got.Metadata)
	})

	t.Run("view fills config and space details", func(t *testing.T) {
		view := database.NewInfraProviderResourceView(f.db, f.spaces)
		got, err := view.Find(ctx, res.ID)
		require.NoError(t, err)
		require.Equal(t, "docker-cfg", got.InfraProviderConfigIdentifier)
		require.Equal(t, "name docker-cfg", got.InfraProviderConfigName)
		require.Equal(t, f.spacePath, got.SpacePath)

		_, err = view.Find(ctx, 9999)
		require.ErrorIs(t, err, gitness_store.ErrResourceNotFound)
	})

	t.Run("soft delete", func(t *testing.T) {
		require.NoError(t, f.resources.Delete(ctx, res.ID))

		_, err := f.resources.Find(ctx, res.ID)
		require.ErrorIs(t, err, gitness_store.ErrResourceNotFound)

		_, err = f.resources.FindByConfigAndIdentifier(ctx, f.spaceID, cfg.ID, "small")
		require.ErrorIs(t, err, gitness_store.ErrResourceNotFound)

		listed, err := f.resources.List(ctx, cfg.ID, types.ListQueryFilter{})
		require.NoError(t, err)
		require.Len(t, listed, 1)
		require.Equal(t, "large", listed[0].UID)
	})
}

func TestRstInfraProviderTemplateStore(t *testing.T) {
	ctx, f := rstNewCDEFixture(t)
	cfg := f.rstMkInfraConfig(ctx, t, "docker-cfg", enum.InfraProviderTypeDocker)

	tmpl := &types.InfraProviderTemplate{
		Identifier: "tmpl", InfraProviderConfigID: cfg.ID, Description: "d", SpaceID: f.spaceID,
		Data: "data", Created: 1, Updated: 1, Version: 1,
	}
	require.NoError(t, f.templates.Create(ctx, tmpl))
	require.NotZero(t, tmpl.ID)

	err := f.templates.Create(ctx, &types.InfraProviderTemplate{
		Identifier: "tmpl", InfraProviderConfigID: cfg.ID, SpaceID: f.spaceID,
	})
	require.ErrorIs(t, err, gitness_store.ErrDuplicate)

	got, err := f.templates.Find(ctx, tmpl.ID)
	require.NoError(t, err)
	require.Equal(t, "data", got.Data)
	require.Equal(t, cfg.ID, got.InfraProviderConfigID)

	got, err = f.templates.FindByIdentifier(ctx, f.spaceID, "tmpl")
	require.NoError(t, err)
	require.Equal(t, tmpl.ID, got.ID)

	_, err = f.templates.FindByIdentifier(ctx, f.spaceID, "other")
	require.ErrorIs(t, err, gitness_store.ErrResourceNotFound)

	tmpl.Description = rstValChanged
	tmpl.Data = "new data"
	tmpl.Updated = 5
	require.NoError(t, f.templates.Update(ctx, tmpl))

	got, err = f.templates.Find(ctx, tmpl.ID)
	require.NoError(t, err)
	require.Equal(t, rstValChanged, got.Description)
	require.Equal(t, "new data", got.Data)
	require.Equal(t, int64(5), got.Updated)
	require.Equal(t, int64(2), got.Version)

	require.NoError(t, f.templates.Delete(ctx, tmpl.ID))
	_, err = f.templates.Find(ctx, tmpl.ID)
	require.ErrorIs(t, err, gitness_store.ErrResourceNotFound)
}

func TestRstGitspaceConfigStore(t *testing.T) {
	ctx, f := rstNewCDEFixture(t)
	infra := f.rstMkInfraConfig(ctx, t, "docker-cfg", enum.InfraProviderTypeDocker)
	resource := f.rstMkInfraResource(ctx, t, infra.ID, "small", 100)

	alpha := f.rstMkGitspace(ctx, t, "Alpha", resource.ID)
	beta := f.rstMkGitspace(ctx, t, "beta", resource.ID)
	gamma := f.rstMkGitspace(ctx, t, "gamma", resource.ID)

	t.Run("duplicate identifier", func(t *testing.T) {
		err := f.gitspaces.Create(ctx, &types.GitspaceConfig{
			Identifier: "beta", SpaceID: f.spaceID,
			InfraProviderResource: types.InfraProviderResource{ID: resource.ID},
		})
		require.ErrorIs(t, err, gitness_store.ErrDuplicate)
	})

	t.Run("find", func(t *testing.T) {
		got, err := f.gitspaces.Find(ctx, alpha.ID, false)
		require.NoError(t, err)
		require.Equal(t, "Alpha", got.Identifier)
		require.Equal(t, enum.IDETypeVSCode, got.IDE)
		require.Equal(t, []enum.AIAgent{enum.AIAgentClaudeCode}, got.AIAgents)
		require.Equal(t, f.spacePath, got.SpacePath)
		require.Equal(t, resource.ID, got.InfraProviderResource.ID)
		require.Equal(t, ".devcontainer", *got.DevcontainerPath)
		require.NotNil(t, got.GitspaceUser.ID)
		require.Equal(t, userID, *got.GitspaceUser.ID)

		_, err = f.gitspaces.Find(ctx, 9999, false)
		require.ErrorIs(t, err, gitness_store.ErrResourceNotFound)
	})

	t.Run("find by identifier is case insensitive", func(t *testing.T) {
		got, err := f.gitspaces.FindByIdentifier(ctx, f.spaceID, "ALPHA")
		require.NoError(t, err)
		require.Equal(t, alpha.ID, got.ID)

		_, err = f.gitspaces.FindByIdentifier(ctx, f.spaceID, "nope")
		require.ErrorIs(t, err, gitness_store.ErrResourceNotFound)
	})

	t.Run("find all and find all by identifier", func(t *testing.T) {
		got, err := f.gitspaces.FindAll(ctx, []int64{alpha.ID, gamma.ID})
		require.NoError(t, err)
		require.Len(t, got, 2)

		ordered, err := f.gitspaces.FindAllByIdentifier(ctx, f.spaceID, []string{"gamma", "Alpha", "beta"})
		require.NoError(t, err)
		require.Len(t, ordered, 3)
		require.Equal(t, "gamma", ordered[0].Identifier)
		require.Equal(t, "Alpha", ordered[1].Identifier)
		require.Equal(t, "beta", ordered[2].Identifier)
	})

	t.Run("update and soft delete", func(t *testing.T) {
		beta.Name = rstValRenamed
		beta.IsMarkedForReset = true
		beta.IsMarkedForInfraReset = true
		beta.AIAgents = nil
		beta.Updated = 50
		require.NoError(t, f.gitspaces.Update(ctx, beta))

		got, err := f.gitspaces.Find(ctx, beta.ID, false)
		require.NoError(t, err)
		require.Equal(t, rstValRenamed, got.Name)
		require.True(t, got.IsMarkedForReset)
		require.True(t, got.IsMarkedForInfraReset)
		require.Empty(t, got.AIAgents)

		beta.IsDeleted = true
		require.NoError(t, f.gitspaces.Update(ctx, beta))
		_, err = f.gitspaces.Find(ctx, beta.ID, false)
		require.ErrorIs(t, err, gitness_store.ErrResourceNotFound)

		got, err = f.gitspaces.Find(ctx, beta.ID, true)
		require.NoError(t, err)
		require.True(t, got.IsDeleted)
	})

	t.Run("active configs for resource", func(t *testing.T) {
		got, err := f.gitspaces.ListActiveConfigsForInfraProviderResource(ctx, resource.ID)
		require.NoError(t, err)
		require.Len(t, got, 2)
		for _, c := range got {
			// no instance exists yet
			require.Equal(t, enum.GitspaceStateUninitialized, c.State)
			require.Nil(t, c.GitspaceInstance)
		}
	})

	t.Run("list with latest instance", func(t *testing.T) {
		// alpha: an old running instance and a newer stopped one; gamma: one error instance.
		f.rstMkInstance(ctx, t, alpha.ID, enum.GitspaceInstanceStateRunning, 100)
		latest := f.rstMkInstance(ctx, t, alpha.ID, enum.GitspaceInstanceStateStopped, 200)
		f.rstMkInstance(ctx, t, gamma.ID, enum.GitspaceInstanceStateError, 150)

		base := types.GitspaceFilter{
			Order: enum.OrderAsc,
			GitspaceInstanceFilter: types.GitspaceInstanceFilter{
				SpaceIDs: []int64{f.spaceID},
			},
		}

		identifiers := func(got []*types.GitspaceConfig) []string {
			out := make([]string, len(got))
			for i, c := range got {
				out[i] = c.Identifier
			}

			return out
		}

		got, err := f.gitspaces.ListWithLatestInstance(ctx, &base)
		require.NoError(t, err)
		require.Equal(t, []string{"gamma", "Alpha"}, identifiers(got))
		require.Equal(t, latest.Identifier, got[1].GitspaceInstance.Identifier)
		require.Equal(t, enum.GitspaceStateStopped, got[1].State)
		require.Equal(t, enum.GitspaceStateError, got[0].State)

		count, err := f.gitspaces.Count(ctx, &base)
		require.NoError(t, err)
		require.Equal(t, int64(2), count)

		t.Run("sorting", func(t *testing.T) {
			for _, sort := range []enum.GitspaceSort{
				enum.GitspaceSortCreated, enum.GitspaceSortLastUsed, enum.GitspaceSortLastActivated,
			} {
				filter := base
				filter.Sort = sort
				sorted, err := f.gitspaces.ListWithLatestInstance(ctx, &filter)
				require.NoError(t, err)
				require.Len(t, sorted, 2)
			}
		})

		t.Run("filters", func(t *testing.T) {
			deleted := false
			marked := false
			tests := []struct {
				name   string
				mutate func(*types.GitspaceFilter)
				want   []string
			}{
				{"states via filter states", func(f *types.GitspaceFilter) {
					f.GitspaceFilterStates = []enum.GitspaceFilterState{enum.GitspaceFilterStateError}
				}, []string{"gamma"}},
				{"stopped state group", func(f *types.GitspaceFilter) {
					f.GitspaceFilterStates = []enum.GitspaceFilterState{enum.GitspaceFilterStateStopped}
				}, []string{"Alpha"}},
				{"running state group", func(f *types.GitspaceFilter) {
					f.GitspaceFilterStates = []enum.GitspaceFilterState{enum.GitspaceFilterStateRunning}
				}, []string{}},
				{"instance states", func(f *types.GitspaceFilter) {
					f.States = []enum.GitspaceInstanceStateType{enum.GitspaceInstanceStateStopped}
				}, []string{"Alpha"}},
				{"owner self", func(f *types.GitspaceFilter) {
					f.Owner = enum.GitspaceOwnerSelf
					f.UserIdentifier = "someone-else"
				}, []string{}},
				{"owner self matches", func(f *types.GitspaceFilter) {
					f.Owner = enum.GitspaceOwnerSelf
					f.UserIdentifier = "user_1"
				}, []string{"gamma", "Alpha"}},
				{"deleted and marked", func(f *types.GitspaceFilter) {
					f.Deleted = &deleted
					f.MarkedForDeletion = &marked
				}, []string{"gamma", "Alpha"}},
				{"code repo type", func(f *types.GitspaceFilter) {
					f.CodeRepoTypes = []enum.GitspaceCodeRepoType{enum.CodeRepoTypeGitlab}
				}, []string{}},
				{"query", func(f *types.GitspaceFilter) {
					f.QueryFilter = types.ListQueryFilter{Query: "alp"}
				}, []string{"Alpha"}},
				{"time bounds", func(f *types.GitspaceFilter) {
					f.LastHeartBeatBefore = 1
					f.LastUsedBefore = 1
					f.LastUpdatedBefore = 1
				}, []string{}},
				{"all spaces", func(f *types.GitspaceFilter) {
					f.AllowAllSpaces = true
					f.SpaceIDs = nil
				}, []string{"gamma", "Alpha"}},
			}
			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					filter := base
					tt.mutate(&filter)
					filtered, err := f.gitspaces.ListWithLatestInstance(ctx, &filter)
					require.NoError(t, err)
					require.Equal(t, tt.want, identifiers(filtered))

					filteredCount, err := f.gitspaces.Count(ctx, &filter)
					require.NoError(t, err)
					require.Equal(t, int64(len(tt.want)), filteredCount)
				})
			}
		})
	})
}

func TestRstGitspaceInstanceStore(t *testing.T) {
	ctx, f := rstNewCDEFixture(t)
	infra := f.rstMkInfraConfig(ctx, t, "docker-cfg", enum.InfraProviderTypeDocker)
	resource := f.rstMkInfraResource(ctx, t, infra.ID, "small", 100)
	cfgA := f.rstMkGitspace(ctx, t, "a", resource.ID)
	cfgB := f.rstMkGitspace(ctx, t, "b", resource.ID)

	first := f.rstMkInstance(ctx, t, cfgA.ID, enum.GitspaceInstanceStateStopped, 100)
	second := f.rstMkInstance(ctx, t, cfgA.ID, enum.GitspaceInstanceStateRunning, 200)
	third := f.rstMkInstance(ctx, t, cfgB.ID, enum.GitspaceInstanceStateError, 300)

	t.Run("duplicate identifier", func(t *testing.T) {
		err := f.instances.Create(ctx, &types.GitspaceInstance{
			GitSpaceConfigID: cfgB.ID, Identifier: first.Identifier, State: enum.GitspaceInstanceStateStopped,
			SpaceID: f.spaceID, UserID: "user_1", AccessType: enum.GitspaceAccessTypeSSHKey,
		})
		require.ErrorIs(t, err, gitness_store.ErrDuplicate)
	})

	t.Run("find", func(t *testing.T) {
		got, err := f.instances.Find(ctx, second.ID)
		require.NoError(t, err)
		require.Equal(t, second.Identifier, got.Identifier)
		require.Equal(t, f.spacePath, got.SpacePath)

		got, err = f.instances.FindByIdentifier(ctx, third.Identifier)
		require.NoError(t, err)
		require.Equal(t, third.ID, got.ID)

		_, err = f.instances.Find(ctx, 9999)
		require.ErrorIs(t, err, gitness_store.ErrResourceNotFound)
		_, err = f.instances.FindByIdentifier(ctx, "missing")
		require.ErrorIs(t, err, gitness_store.ErrResourceNotFound)
	})

	t.Run("latest by config", func(t *testing.T) {
		got, err := f.instances.FindLatestByGitspaceConfigID(ctx, cfgA.ID)
		require.NoError(t, err)
		require.Equal(t, second.ID, got.ID)

		_, err = f.instances.FindLatestByGitspaceConfigID(ctx, 9999)
		require.ErrorIs(t, err, gitness_store.ErrResourceNotFound)

		all, err := f.instances.FindAllLatestByGitspaceConfigID(ctx, []int64{cfgA.ID, cfgB.ID})
		require.NoError(t, err)
		ids := []int64{}
		for _, i := range all {
			ids = append(ids, i.ID)
		}
		require.ElementsMatch(t, []int64{second.ID, third.ID}, ids)

		none, err := f.instances.FindAllLatestByGitspaceConfigID(ctx, nil)
		require.NoError(t, err)
		require.Empty(t, none)
	})

	t.Run("list and count", func(t *testing.T) {
		spaceFilter := types.GitspaceInstanceFilter{SpaceIDs: []int64{f.spaceID}}

		got, err := f.instances.List(ctx, &spaceFilter)
		require.NoError(t, err)
		require.Equal(t, []int64{first.ID, second.ID, third.ID}, []int64{got[0].ID, got[1].ID, got[2].ID})

		tests := []struct {
			name   string
			mutate func(*types.GitspaceInstanceFilter)
			want   int64
		}{
			{"no filter", func(*types.GitspaceInstanceFilter) {}, 3},
			{"state", func(f *types.GitspaceInstanceFilter) {
				f.States = []enum.GitspaceInstanceStateType{enum.GitspaceInstanceStateRunning}
			}, 1},
			{"limit", func(f *types.GitspaceInstanceFilter) { f.Limit = 2 }, 3},
			{"all spaces", func(f *types.GitspaceInstanceFilter) {
				f.AllowAllSpaces = true
				f.SpaceIDs = nil
			}, 3},
			{"other space", func(f *types.GitspaceInstanceFilter) { f.SpaceIDs = []int64{9999} }, 0},
			{"updated before", func(f *types.GitspaceInstanceFilter) { f.LastUpdatedBefore = 250 }, 2},
			{"heartbeat and used before", func(f *types.GitspaceInstanceFilter) {
				f.LastHeartBeatBefore = 1
				f.LastUsedBefore = 1
			}, 0},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				filter := spaceFilter
				tt.mutate(&filter)

				count, err := f.instances.Count(ctx, &filter)
				require.NoError(t, err)
				require.Equal(t, tt.want, count)

				listed, err := f.instances.List(ctx, &filter)
				require.NoError(t, err)
				if filter.Limit > 0 {
					require.Len(t, listed, filter.Limit)
				} else {
					require.Equal(t, tt.want, int64(len(listed)))
				}
			})
		}
	})

	t.Run("update", func(t *testing.T) {
		now := int64(1_000)
		url := "https://gitspace.example.com"
		ssh := "ssh me"
		msg := "boom"
		changes := true
		second.State = enum.GitspaceInstanceStateStopped
		second.TotalTimeUsed = 500
		second.Updated = 777
		second.LastUsed = &now
		second.LastHeartbeat = &now
		second.URL = &url
		second.ActiveTimeStarted = &now
		second.ActiveTimeEnded = &now
		second.HasGitChanges = &changes
		second.ErrorMessage = &msg
		second.SSHCommand = &ssh
		require.NoError(t, f.instances.Update(ctx, second))

		got, err := f.instances.Find(ctx, second.ID)
		require.NoError(t, err)
		require.Equal(t, enum.GitspaceInstanceStateStopped, got.State)
		require.Equal(t, int64(500), got.TotalTimeUsed)
		require.Equal(t, url, *got.URL)
		require.Equal(t, ssh, *got.SSHCommand)
		require.Equal(t, msg, *got.ErrorMessage)
		require.True(t, *got.HasGitChanges)
		require.Equal(t, now, *got.LastUsed)
		require.Equal(t, now, *got.LastHeartbeat)
		require.Equal(t, now, *got.ActiveTimeStarted)
		require.Equal(t, now, *got.ActiveTimeEnded)
	})

	t.Run("update with unusual active times still succeeds", func(t *testing.T) {
		// the store only warns about inconsistent active time details; the update must go through.
		started := int64(1)
		for _, state := range []enum.GitspaceInstanceStateType{
			enum.GitspaceInstanceStateStarting,
			enum.GitspaceInstanceStateRunning,
			enum.GitspaceInstanceStateStopping,
			enum.GitspaceInstanceStateDeleted,
			enum.GitspaceInstanceStateError,
		} {
			inst := *third
			inst.State = state
			inst.ActiveTimeStarted = &started
			inst.ActiveTimeEnded = nil
			inst.TotalTimeUsed = 0
			require.NoError(t, f.instances.Update(ctx, &inst), state)
		}
	})

	t.Run("total usage", func(t *testing.T) {
		start := int64(10_000)
		end := int64(20_000)
		spaceIDs := []int64{f.spaceID}

		// instances created by the previous subtests may still be counted as active.
		baseline, err := f.instances.FindTotalUsage(ctx, start, end, spaceIDs)
		require.NoError(t, err)

		// 5s fully inside the window, and one still running that started 4s before the window ends.
		inst1 := f.rstMkInstance(ctx, t, cfgB.ID, enum.GitspaceInstanceStateStopped, 1)
		inst2 := f.rstMkInstance(ctx, t, cfgB.ID, enum.GitspaceInstanceStateRunning, 2)
		s1, e1, s2 := int64(11_000), int64(16_000), int64(16_000)
		inst1.ActiveTimeStarted, inst1.ActiveTimeEnded = &s1, &e1
		inst2.ActiveTimeStarted = &s2
		require.NoError(t, f.instances.Update(ctx, inst1))
		require.NoError(t, f.instances.Update(ctx, inst2))

		usage, err := f.instances.FindTotalUsage(ctx, start, end, spaceIDs)
		require.NoError(t, err)
		require.Equal(t, baseline+5_000+4_000, usage)

		usage, err = f.instances.FindTotalUsage(ctx, start, end, []int64{9999})
		require.NoError(t, err)
		require.Zero(t, usage)
	})
}

func TestRstGitspaceEventStore(t *testing.T) {
	ctx, f := rstNewCDEFixture(t)

	mk := func(event enum.GitspaceEventType, entityID int64, key string, ts int64) {
		e := &types.GitspaceEvent{
			Event: event, EntityType: enum.GitspaceEntityTypeGitspaceConfig, EntityID: entityID,
			QueryKey: key, Timestamp: ts, Created: ts,
		}
		require.NoError(t, f.events.Create(ctx, e))
		require.NotZero(t, e.ID)
	}

	mk(enum.GitspaceEventTypeGitspaceActionStart, 1, "k1", 100)
	mk(enum.GitspaceEventTypeGitspaceActionStart, 1, "k1", 300)
	mk(enum.GitspaceEventTypeGitspaceActionStop, 1, "k1", 200)
	mk(enum.GitspaceEventTypeGitspaceActionStart, 2, "k2", 400)

	t.Run("latest by type", func(t *testing.T) {
		got, err := f.events.FindLatestByTypeAndGitspaceConfigID(ctx, enum.GitspaceEventTypeGitspaceActionStart, 1)
		require.NoError(t, err)
		require.Equal(t, int64(300), got.Timestamp)
		require.Equal(t, "k1", got.QueryKey)

		_, err = f.events.FindLatestByTypeAndGitspaceConfigID(ctx, enum.GitspaceEventTypeGitspaceActionReset, 1)
		require.ErrorIs(t, err, gitness_store.ErrResourceNotFound)
	})

	t.Run("list", func(t *testing.T) {
		got, count, err := f.events.List(ctx, &types.GitspaceEventFilter{
			Pagination: types.Pagination{Page: 1, Size: 10},
		})
		require.NoError(t, err)
		require.Equal(t, 4, count)
		require.Equal(t, int64(400), got[0].Timestamp)
		require.Equal(t, int64(100), got[3].Timestamp)

		got, count, err = f.events.List(ctx, &types.GitspaceEventFilter{
			Pagination: types.Pagination{Page: 1, Size: 10}, QueryKey: "k1", EntityID: 1,
			EntityType: enum.GitspaceEntityTypeGitspaceConfig,
			SkipEvents: []enum.GitspaceEventType{enum.GitspaceEventTypeGitspaceActionStop},
		})
		require.NoError(t, err)
		require.Equal(t, 2, count)
		require.Len(t, got, 2)

		// paging only limits the rows, the count covers every match.
		got, count, err = f.events.List(ctx, &types.GitspaceEventFilter{Pagination: types.Pagination{Page: 2, Size: 3}})
		require.NoError(t, err)
		require.Equal(t, 4, count)
		require.Len(t, got, 1)
	})
}

func TestRstGitspaceSettingsStore(t *testing.T) {
	ctx, f := rstNewCDEFixture(t)

	criteria := types.GitspaceSettingsCriteria{"org": map[string]any{"id": "o1"}}
	key, err := criteria.ToKey()
	require.NoError(t, err)

	always := &types.GitspaceSettings{
		SpaceID: f.spaceID, SettingsType: enum.SettingsTypeGitspaceConfig,
		Settings: types.SettingsData{Data: map[string]any{"limit": float64(1)}}, Created: 1, Updated: 1,
	}
	scoped := &types.GitspaceSettings{
		SpaceID: f.spaceID, SettingsType: enum.SettingsTypeGitspaceConfig, CriteriaKey: key,
		Settings: types.SettingsData{Data: map[string]any{"limit": float64(2)}, Criteria: criteria},
		Created:  2, Updated: 2,
	}
	require.NoError(t, f.settings.Upsert(ctx, always))
	require.NoError(t, f.settings.Upsert(ctx, scoped))

	t.Run("find by type", func(t *testing.T) {
		got, err := f.settings.FindByType(
			ctx, f.spaceID, enum.SettingsTypeGitspaceConfig, &types.ApplyAlwaysToSpaceCriteria)
		require.NoError(t, err)
		require.Equal(t, float64(1), got.Settings.Data["limit"])

		got, err = f.settings.FindByType(ctx, f.spaceID, enum.SettingsTypeGitspaceConfig, &criteria)
		require.NoError(t, err)
		require.Equal(t, float64(2), got.Settings.Data["limit"])
		require.Equal(t, key, got.CriteriaKey)

		_, err = f.settings.FindByType(
			ctx, f.spaceID, enum.SettingsTypeInfraProvider, &types.ApplyAlwaysToSpaceCriteria)
		require.ErrorIs(t, err, gitness_store.ErrResourceNotFound)
	})

	t.Run("upsert updates existing row", func(t *testing.T) {
		always.Settings.Data = map[string]any{"limit": float64(9)}
		always.Updated = 50
		require.NoError(t, f.settings.Upsert(ctx, always))

		got, err := f.settings.FindByType(
			ctx, f.spaceID, enum.SettingsTypeGitspaceConfig, &types.ApplyAlwaysToSpaceCriteria)
		require.NoError(t, err)
		require.Equal(t, float64(9), got.Settings.Data["limit"])
		require.Equal(t, int64(50), got.Updated)
	})

	t.Run("list", func(t *testing.T) {
		got, err := f.settings.List(ctx, f.spaceID, &types.GitspaceSettingsFilter{})
		require.NoError(t, err)
		require.Len(t, got, 2)
		// most recently updated first
		require.Equal(t, int64(50), got[0].Updated)

		got, err = f.settings.List(ctx, f.spaceID, &types.GitspaceSettingsFilter{
			ListQueryFilter: types.ListQueryFilter{Pagination: types.Pagination{Page: 2, Size: 1}},
		})
		require.NoError(t, err)
		require.Len(t, got, 1)
		require.Equal(t, key, got[0].CriteriaKey)
	})
}

func TestRstInfraProvisionedStore(t *testing.T) {
	ctx, f := rstNewCDEFixture(t)
	infra := f.rstMkInfraConfig(ctx, t, "docker-cfg", enum.InfraProviderTypeDocker)
	resource := f.rstMkInfraResource(ctx, t, infra.ID, "small", 100)
	cfg := f.rstMkGitspace(ctx, t, "cfg", resource.ID)
	stoppedInst := f.rstMkInstance(ctx, t, cfg.ID, enum.GitspaceInstanceStateStopped, 100)
	runningInst := f.rstMkInstance(ctx, t, cfg.ID, enum.GitspaceInstanceStateRunning, 200)

	meta := `{"ip":"1.2.3.4"}`
	mk := func(instID int64, status enum.InfraStatus, created int64, gateway string) *types.InfraProvisioned {
		p := &types.InfraProvisioned{
			GitspaceInstanceID: instID, InfraProviderType: enum.InfraProviderTypeDocker,
			InfraProviderResourceID: resource.ID, SpaceID: f.spaceID, Created: created, Updated: created,
			ResponseMetadata: &meta, InputParams: "params", InfraStatus: status, ServerHostIP: "10.0.0.1",
			ServerHostPort: "22", ProxyHost: "proxy", ProxyPort: 8080, GatewayHost: gateway,
		}
		require.NoError(t, f.provisioned.Create(ctx, p))
		require.NotZero(t, p.ID)

		return p
	}

	oldStopped := mk(stoppedInst.ID, enum.InfraStatusStopped, 100, "gw1")
	newer := mk(runningInst.ID, enum.InfraStatusProvisioned, 300, "gw1")
	mk(runningInst.ID, enum.InfraStatusProvisioned, 200, "gw2")

	t.Run("find", func(t *testing.T) {
		got, err := f.provisioned.Find(ctx, newer.ID)
		require.NoError(t, err)
		require.Equal(t, newer.GatewayHost, got.GatewayHost)
		require.Equal(t, meta, *got.ResponseMetadata)
		require.Equal(t, int32(8080), got.ProxyPort)

		_, err = f.provisioned.Find(ctx, 9999)
		require.ErrorIs(t, err, gitness_store.ErrResourceNotFound)
	})

	t.Run("latest by instance", func(t *testing.T) {
		got, err := f.provisioned.FindLatestByGitspaceInstanceID(ctx, runningInst.ID)
		require.NoError(t, err)
		require.Equal(t, newer.ID, got.ID)

		got, err = f.provisioned.FindLatestByGitspaceInstanceIdentifier(ctx, f.spaceID, runningInst.Identifier)
		require.NoError(t, err)
		require.Equal(t, newer.ID, got.ID)

		_, err = f.provisioned.FindLatestByGitspaceInstanceIdentifier(ctx, f.spaceID, "missing")
		require.ErrorIs(t, err, gitness_store.ErrResourceNotFound)
		_, err = f.provisioned.FindLatestByGitspaceInstanceID(ctx, 9999)
		require.ErrorIs(t, err, gitness_store.ErrResourceNotFound)
	})

	t.Run("all latest by gateway", func(t *testing.T) {
		got, err := f.provisioned.FindAllLatestByGateway(ctx, "gw1")
		require.NoError(t, err)
		// only the provisioned entry of gw1 is returned; the stopped one is excluded.
		require.Len(t, got, 1)
		require.Equal(t, runningInst.Identifier, got[0].GitspaceInstanceIdentifier)
		require.Equal(t, "10.0.0.1", got[0].ServerHostIP)

		got, err = f.provisioned.FindAllLatestByGateway(ctx, "unknown-gw")
		require.NoError(t, err)
		require.Empty(t, got)
	})

	t.Run("stopped infra by config identifier", func(t *testing.T) {
		got, err := f.provisioned.FindStoppedInfraForGitspaceConfigIdentifierByState(
			ctx, "cfg", enum.GitspaceInstanceStateStopped)
		require.NoError(t, err)
		require.Equal(t, oldStopped.ID, got.ID)

		_, err = f.provisioned.FindStoppedInfraForGitspaceConfigIdentifierByState(
			ctx, "cfg", enum.GitspaceInstanceStateRunning)
		require.ErrorIs(t, err, gitness_store.ErrResourceNotFound)
	})

	t.Run("update and delete", func(t *testing.T) {
		newMeta := `{"ip":"5.6.7.8"}`
		newer.ResponseMetadata = &newMeta
		newer.InfraStatus = enum.InfraStatusStopped
		newer.ServerHostIP = "10.0.0.2"
		newer.ProxyPort = 9090
		newer.Updated = 999
		require.NoError(t, f.provisioned.Update(ctx, newer))

		got, err := f.provisioned.Find(ctx, newer.ID)
		require.NoError(t, err)
		require.Equal(t, newMeta, *got.ResponseMetadata)
		require.Equal(t, enum.InfraStatusStopped, got.InfraStatus)
		require.Equal(t, "10.0.0.2", got.ServerHostIP)
		require.Equal(t, int32(9090), got.ProxyPort)

		require.NoError(t, f.provisioned.Delete(ctx, newer.ID))
		_, err = f.provisioned.Find(ctx, newer.ID)
		require.ErrorIs(t, err, gitness_store.ErrResourceNotFound)
	})
}

func TestRstCDEGatewayStore(t *testing.T) {
	ctx, f := rstNewCDEFixture(t)
	cfgA := f.rstMkInfraConfig(ctx, t, "cfg-a", enum.InfraProviderTypeDocker)
	cfgB := f.rstMkInfraConfig(ctx, t, "cfg-b", enum.InfraProviderTypeHarnessGCP)

	now := time.Now().UnixMilli()
	stale := time.Now().Add(-time.Hour).UnixMilli()

	mk := func(name string, cfgID int64, region, health, envoy string, updated int64) {
		gw := &types.CDEGateway{
			CDEGatewayStats: types.CDEGatewayStats{
				Name: name, GroupName: "grp", Region: region, Zone: "z1", Version: "1.0", Health: health,
				EnvoyHealth: envoy,
			},
			SpaceID: f.spaceID, InfraProviderConfigID: cfgID, Created: updated, Updated: updated,
		}
		require.NoError(t, f.gateways.Upsert(ctx, gw))
	}

	mk("gw-healthy", cfgA.ID, "eu", types.GatewayHealthHealthy, types.GatewayHealthHealthy, now)
	mk("gw-stale", cfgA.ID, "us", types.GatewayHealthHealthy, types.GatewayHealthHealthy, stale)
	mk("gw-bad-envoy", cfgB.ID, "eu", types.GatewayHealthHealthy, types.GatewayHealthUnhealthy, now)

	names := func(got []*types.CDEGateway) []string {
		out := make([]string, len(got))
		for i, g := range got {
			out[i] = g.Name
		}

		return out
	}

	t.Run("upsert updates mutable columns", func(t *testing.T) {
		mk("gw-healthy", cfgA.ID, "eu", types.GatewayHealthUnhealthy, types.GatewayHealthUnhealthy, now)

		got, err := f.gateways.List(ctx, &types.CDEGatewayFilter{InfraProviderConfigIDs: []int64{cfgA.ID}})
		require.NoError(t, err)
		require.Len(t, got, 2)
		for _, g := range got {
			if g.Name == "gw-healthy" {
				require.Equal(t, types.GatewayHealthUnhealthy, g.Health)
			}
		}

		mk("gw-healthy", cfgA.ID, "eu", types.GatewayHealthHealthy, types.GatewayHealthHealthy, now)
	})

	t.Run("list without filter", func(t *testing.T) {
		got, err := f.gateways.List(ctx, nil)
		require.NoError(t, err)
		require.Len(t, got, 3)
	})

	t.Run("healthy only", func(t *testing.T) {
		got, err := f.gateways.List(ctx, &types.CDEGatewayFilter{
			Health: types.GatewayHealthHealthy, HealthReportValidityInMins: 5,
		})
		require.NoError(t, err)
		require.Equal(t, []string{"gw-healthy"}, names(got))
	})

	t.Run("unhealthy only", func(t *testing.T) {
		got, err := f.gateways.List(ctx, &types.CDEGatewayFilter{
			Health: types.GatewayHealthUnhealthy, HealthReportValidityInMins: 5,
		})
		require.NoError(t, err)
		require.ElementsMatch(t, []string{"gw-stale", "gw-bad-envoy"}, names(got))
	})

	t.Run("latest per region", func(t *testing.T) {
		got, err := f.gateways.List(ctx, &types.CDEGatewayFilter{IsLatest: true})
		require.NoError(t, err)
		require.ElementsMatch(t, []string{"gw-healthy", "gw-stale", "gw-bad-envoy"}, names(got))
	})
}
