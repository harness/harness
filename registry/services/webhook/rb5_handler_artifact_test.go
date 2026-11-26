//  Copyright 2023 Harness, Inc.
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
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/harness/gitness/app/services/refcache"
	gitnesswebhook "github.com/harness/gitness/app/services/webhook"
	"github.com/harness/gitness/app/store"
	storecache "github.com/harness/gitness/app/store/cache"
	"github.com/harness/gitness/app/url"
	"github.com/harness/gitness/events"
	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	registryevents "github.com/harness/gitness/registry/app/events/artifact"
	"github.com/harness/gitness/registry/app/pkg"
	registrystore "github.com/harness/gitness/registry/app/store"
	registrytypes "github.com/harness/gitness/registry/types"
	basestore "github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/require"
)

var errRb5Boom = errors.New("boom")

type rb5Principals struct {
	store.PrincipalStore
	err error
}

func (p rb5Principals) Find(_ context.Context, id int64) (*types.Principal, error) {
	if p.err != nil {
		return nil, p.err
	}
	return &types.Principal{ID: id, UID: "alice", DisplayName: "Alice", Email: "alice@example.com"}, nil
}

type rb5Spaces struct {
	store.SpaceStore
	ids []int64
	err error
}

func (s rb5Spaces) GetAncestorIDs(context.Context, int64) ([]int64, error) { return s.ids, s.err }

type rb5SpaceIDCache struct{ err error }

func (c rb5SpaceIDCache) Get(_ context.Context, id int64) (*types.SpaceCore, error) {
	if c.err != nil {
		return nil, c.err
	}
	return &types.SpaceCore{ID: id, Path: "root/team", Identifier: "team"}, nil
}
func (rb5SpaceIDCache) Stats() (int64, int64)        { return 0, 0 }
func (rb5SpaceIDCache) Evict(context.Context, int64) {}
func (rb5SpaceIDCache) Map(context.Context, []int64) (map[int64]*types.SpaceCore, error) {
	return map[int64]*types.SpaceCore{}, nil
}

type rb5Registries struct {
	registrystore.RegistryRepository
	err error
}

func (r rb5Registries) Get(
	_ context.Context, id int64, _ ...registrytypes.QueryOption,
) (*registrytypes.Registry, error) {
	if r.err != nil {
		return nil, r.err
	}
	return &registrytypes.Registry{ID: id, Name: "reg", Description: "a registry", ParentID: 9}, nil
}

type rb5Hooks struct {
	registrystore.WebhooksRepository
	mu      sync.Mutex
	hooks   []*types.WebhookCore
	parents []types.WebhookParentInfo
	updated int
	listErr error
}

func (h *rb5Hooks) ListAllByRegistry(
	_ context.Context, parents []types.WebhookParentInfo,
) ([]*types.WebhookCore, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.parents = parents
	return h.hooks, h.listErr
}

func (h *rb5Hooks) UpdateOptLock(
	_ context.Context, hook *types.WebhookCore, mutate func(*types.WebhookCore) error,
) (*types.WebhookCore, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.updated++
	return hook, mutate(hook)
}

func (h *rb5Hooks) Find(_ context.Context, id int64) (*types.WebhookCore, error) {
	for _, hook := range h.hooks {
		if hook.ID == id {
			return hook, nil
		}
	}
	return nil, basestore.ErrResourceNotFound
}

type rb5Executions struct {
	registrystore.WebhooksExecutionRepository
	mu      sync.Mutex
	created []*types.WebhookExecutionCore
	stored  map[int64]*types.WebhookExecutionCore
}

func (e *rb5Executions) Create(_ context.Context, ex *types.WebhookExecutionCore) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.created = append(e.created, ex)
	return nil
}

func (e *rb5Executions) ListForTrigger(context.Context, string) ([]*types.WebhookExecutionCore, error) {
	return nil, nil
}

func (e *rb5Executions) Find(_ context.Context, id int64) (*types.WebhookExecutionCore, error) {
	if ex, ok := e.stored[id]; ok {
		return ex, nil
	}
	return nil, basestore.ErrResourceNotFound
}

type rb5URLs struct{ target string }

func (u rb5URLs) GetWebhookURL(context.Context, *types.WebhookCore) (string, error) {
	return u.target, nil
}

type rb5UIURL struct{ url.Provider }

func (rb5UIURL) GenerateUIRegistryURL(_ context.Context, parent, name string) string {
	return "https://ui/" + parent + "/" + name
}

func (rb5UIURL) RegistryURL(_ context.Context, params ...string) string {
	return "https://host/" + strings.Join(params, "/")
}

type rb5Env struct {
	svc        *Service
	hooks      *rb5Hooks
	executions *rb5Executions
	server     *httptest.Server
	mu         sync.Mutex
	bodies     []map[string]any
	headers    []http.Header
}

type rb5Opts struct {
	principalErr error
	registryErr  error
	spaceErr     error
	ancestorsErr error
	ancestors    []int64
}

func rb5NewEnv(t *testing.T, opts rb5Opts) *rb5Env {
	t.Helper()
	env := &rb5Env{}
	env.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		var body map[string]any
		_ = json.Unmarshal(raw, &body)
		env.mu.Lock()
		env.bodies = append(env.bodies, body)
		env.headers = append(env.headers, r.Header.Clone())
		env.mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(env.server.Close)

	env.hooks = &rb5Hooks{hooks: []*types.WebhookCore{{
		ID: 3, ParentID: 1, ParentType: enum.WebhookParentRegistry, Identifier: "hook", Enabled: true,
		Type: enum.WebhookTypeExternal, URL: env.server.URL,
	}}}
	env.executions = &rb5Executions{stored: map[int64]*types.WebhookExecutionCore{}}
	principals := rb5Principals{err: opts.principalErr}
	cfg := gitnesswebhook.Config{
		UserAgentIdentity: "rb5", EventReaderName: "rb5", Concurrency: 1, MaxRetries: 1, AllowLoopback: true,
		AllowPrivateNetwork: true,
	}
	executor := gitnesswebhook.NewWebhookExecutor(cfg, rb5URLs{target: env.server.URL}, nil, nil, nil, principals,
		&RegistryWebhookExecutorStore{webhookStore: env.hooks, webhookExecutionStore: env.executions},
		gitnesswebhook.ArtifactRegistryTrigger)
	env.svc = &Service{
		WebhookExecutor:    executor,
		urlProvider:        rb5UIURL{},
		spaceStore:         rb5Spaces{ids: opts.ancestors, err: opts.ancestorsErr},
		principalStore:     principals,
		config:             cfg,
		registryRepository: rb5Registries{err: opts.registryErr},
		spaceFinder: refcache.NewSpaceFinder(
			rb5SpaceIDCache{err: opts.spaceErr}, nil, nil, storecache.Evictor[*types.SpaceCore]{}),
	}
	return env
}

func (e *rb5Env) lastBody(t *testing.T) map[string]any {
	t.Helper()
	e.mu.Lock()
	defer e.mu.Unlock()
	require.NotEmpty(t, e.bodies)
	return e.bodies[len(e.bodies)-1]
}

func TestRb5HandleEventArtifactCreated(t *testing.T) {
	tests := []struct {
		name     string
		artifact registryevents.Artifact
		wantType string
		wantName string
		wantVer  string
	}{
		{
			name: "docker",
			artifact: &registryevents.DockerArtifact{
				BaseArtifact: registryevents.BaseArtifact{Name: "img"}, Tag: "v1",
			},
			wantType: "DOCKER", wantName: "img", wantVer: "v1",
		},
		{
			name: "helm",
			artifact: &registryevents.HelmArtifact{
				BaseArtifact: registryevents.BaseArtifact{Name: "chart"}, Tag: "1.2.3",
			},
			wantType: "HELM", wantName: "chart", wantVer: "1.2.3",
		},
		{
			name: "common",
			artifact: &registryevents.CommonArtifact{
				BaseArtifact: registryevents.BaseArtifact{Name: "pkg"}, Version: "2.0", Type: artifact.PackageTypeNPM,
			},
			wantType: "NPM", wantName: "pkg", wantVer: "2.0",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := rb5NewEnv(t, rb5Opts{ancestors: []int64{20, 30}})
			err := env.svc.handleEventArtifactCreated(context.Background(),
				&events.Event[*registryevents.ArtifactCreatedPayload]{
					ID:      "evt-1",
					Payload: &registryevents.ArtifactCreatedPayload{PrincipalID: 7, RegistryID: 1, Artifact: tt.artifact},
				})
			require.NoError(t, err)

			body := env.lastBody(t)
			require.Equal(t, string(enum.WebhookTriggerArtifactCreated), body["trigger"])
			require.Equal(t, "https://ui/root/team/reg", rb5Map(t, body["registry"])["url"])
			require.Equal(t, "alice", rb5Map(t, body["principal"])["uid"])
			info := rb5Map(t, body["artifact_info"])
			require.Equal(t, tt.wantType, info["type"])
			require.Equal(t, tt.wantName, info["name"])
			require.Equal(t, tt.wantVer, info["version"])

			require.Equal(t, []types.WebhookParentInfo{
				{ID: 1, Type: enum.WebhookParentRegistry},
				{ID: 20, Type: enum.WebhookParentSpace},
				{ID: 30, Type: enum.WebhookParentSpace},
			}, env.hooks.parents)
			require.Len(t, env.executions.created, 1)
			require.Equal(t, 1, env.hooks.updated)
		})
	}
}

func TestRb5HandleEventArtifactDeleted(t *testing.T) {
	env := rb5NewEnv(t, rb5Opts{})
	err := env.svc.handleEventArtifactDeleted(context.Background(),
		&events.Event[*registryevents.ArtifactDeletedPayload]{
			ID: "evt-2",
			Payload: &registryevents.ArtifactDeletedPayload{
				PrincipalID: 7, RegistryID: 1,
				Artifact: &registryevents.DockerArtifact{BaseArtifact: registryevents.BaseArtifact{Name: "img"}, Tag: "v1"},
			},
		})
	require.NoError(t, err)
	body := env.lastBody(t)
	require.Equal(t, string(enum.WebhookTriggerArtifactDeleted), body["trigger"])
	require.Equal(t, "img", rb5Map(t, body["artifact_info"])["name"])
}

func TestRb5HandleEventErrors(t *testing.T) {
	created := &events.Event[*registryevents.ArtifactCreatedPayload]{
		ID: "e", Payload: &registryevents.ArtifactCreatedPayload{PrincipalID: 7, RegistryID: 1},
	}
	deleted := &events.Event[*registryevents.ArtifactDeletedPayload]{
		ID: "e", Payload: &registryevents.ArtifactDeletedPayload{PrincipalID: 7, RegistryID: 1},
	}
	tests := []struct {
		name    string
		opts    rb5Opts
		wantMsg string
	}{
		{name: "principal missing is discarded", opts: rb5Opts{principalErr: basestore.ErrResourceNotFound},
			wantMsg: "doesn't exist anymore"},
		{name: "principal error", opts: rb5Opts{principalErr: errRb5Boom}, wantMsg: "failed to get principal"},
		{name: "registry error", opts: rb5Opts{registryErr: errRb5Boom}, wantMsg: "boom"},
		{name: "space error", opts: rb5Opts{spaceErr: errRb5Boom}, wantMsg: "body creation function failed"},
		{name: "ancestors error", opts: rb5Opts{ancestorsErr: errRb5Boom},
			wantMsg: "failed to get parent space ids"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := rb5NewEnv(t, tt.opts)
			require.ErrorContains(t, env.svc.handleEventArtifactCreated(context.Background(), created), tt.wantMsg)
			require.ErrorContains(t, env.svc.handleEventArtifactDeleted(context.Background(), deleted), tt.wantMsg)
			require.Empty(t, env.bodies)
		})
	}

	t.Run("listing webhooks fails", func(t *testing.T) {
		env := rb5NewEnv(t, rb5Opts{})
		env.hooks.listErr = errRb5Boom
		require.ErrorIs(t, env.svc.handleEventArtifactCreated(context.Background(), created), errRb5Boom)
	})
}

func TestRb5GetArtifactInfo(t *testing.T) {
	require.Equal(t, artifact.PackageType(""), getArtifactInfo(nil).Type)

	info := getArtifactInfo(&registryevents.CommonArtifact{
		BaseArtifact: registryevents.BaseArtifact{Name: "n"}, Version: "1", Type: artifact.PackageTypeMAVEN,
	})
	require.Equal(t, artifact.PackageTypeMAVEN, info.Type)
	require.Equal(t, "n", info.Name)
	require.Equal(t, "1", info.Version)
	require.NotNil(t, info.Artifact)
}

func TestRb5GetParentInfoRegistryNotInherited(t *testing.T) {
	env := rb5NewEnv(t, rb5Opts{ancestors: []int64{1, 2}})
	parents, err := env.svc.getParentInfoRegistry(context.Background(), 4, false)
	require.NoError(t, err)
	require.Equal(t, []types.WebhookParentInfo{{ID: 4, Type: enum.WebhookParentRegistry}}, parents)

	env = rb5NewEnv(t, rb5Opts{registryErr: errRb5Boom})
	_, err = env.svc.getParentInfoRegistry(context.Background(), 4, true)
	require.ErrorContains(t, err, "failed to get registry")
}

func TestRb5ReTriggerWebhookExecution(t *testing.T) {
	ctx := context.Background()
	env := rb5NewEnv(t, rb5Opts{})
	env.executions.stored[11] = &types.WebhookExecutionCore{
		ID: 11, WebhookID: 3, Retriggerable: true, TriggerID: "trig", TriggerType: enum.WebhookTriggerArtifactCreated,
		Request: types.WebhookExecutionRequest{Body: `{"trigger":"artifact_created"}`},
	}
	env.executions.stored[12] = &types.WebhookExecutionCore{ID: 12, WebhookID: 3}
	env.executions.stored[13] = &types.WebhookExecutionCore{ID: 13, WebhookID: 99, Retriggerable: true}

	res, err := env.svc.ReTriggerWebhookExecution(ctx, 11)
	require.NoError(t, err)
	require.NotNil(t, res.Execution)
	require.Equal(t, "trig", res.TriggerID)
	require.Equal(t, "artifact_created", env.lastBody(t)["trigger"])

	_, err = env.svc.ReTriggerWebhookExecution(ctx, 12)
	require.ErrorIs(t, err, gitnesswebhook.ErrWebhookNotRetriggerable)

	_, err = env.svc.ReTriggerWebhookExecution(ctx, 13)
	require.ErrorContains(t, err, "failed to find webhook")

	_, err = env.svc.ReTriggerWebhookExecution(ctx, 404)
	require.ErrorContains(t, err, "failed to find webhook execution")
}

func rb5Info(pt artifact.PackageType, image string) pkg.RegistryInfo {
	return pkg.RegistryInfo{
		PackageType: pt,
		ArtifactInfo: &pkg.ArtifactInfo{
			BaseInfo: &pkg.BaseInfo{RootIdentifier: "root"}, Image: image,
		},
	}
}

func TestRb5ArtifactPayloadHelpers(t *testing.T) {
	ctx := context.Background()
	urls := rb5UIURL{}

	t.Run("created docker", func(t *testing.T) {
		info := rb5Info(artifact.PackageTypeDOCKER, "img")
		got := GetArtifactCreatedPayload(ctx, info, 7, 1, "reg", "v1", "sha256:abc", urls)
		docker, ok := got.Artifact.(*registryevents.DockerArtifact)
		require.True(t, ok)
		require.Equal(t, "host/root/reg/img:v1", docker.URL)
		require.Equal(t, "img:v1", docker.Ref)
		require.Equal(t, "sha256:abc", docker.Digest)
	})
	t.Run("created helm", func(t *testing.T) {
		info := rb5Info(artifact.PackageTypeHELM, "chart")
		got := GetArtifactCreatedPayload(ctx, info, 7, 1, "reg", "1.0", "sha256:abc", urls)
		helm, ok := got.Artifact.(*registryevents.HelmArtifact)
		require.True(t, ok)
		require.Equal(t, "oci://host/root/reg/chart:1.0", helm.URL)
	})
	t.Run("created other type has no artifact", func(t *testing.T) {
		info := rb5Info(artifact.PackageTypeNPM, "n")
		require.Nil(t, GetArtifactCreatedPayload(ctx, info, 7, 1, "reg", "1", "", urls).Artifact)
	})

	t.Run("deleted docker untagged", func(t *testing.T) {
		got := GetArtifactDeletedPayload(ctx, 7, 1, "reg", "sha256:abc", "sha256:abc", "root",
			artifact.PackageTypeDOCKER, "img", urls, true)
		docker, ok := got.Artifact.(*registryevents.DockerArtifact)
		require.True(t, ok)
		require.Equal(t, "img@sha256:abc", docker.Ref)
		require.Empty(t, docker.Tag)
	})
	t.Run("deleted helm tagged", func(t *testing.T) {
		got := GetArtifactDeletedPayload(ctx, 7, 1, "reg", "1.0", "sha256:abc", "root",
			artifact.PackageTypeHELM, "chart", urls, false)
		helm, ok := got.Artifact.(*registryevents.HelmArtifact)
		require.True(t, ok)
		require.Equal(t, "chart:1.0", helm.Ref)
		require.Equal(t, "1.0", helm.Tag)
	})
	t.Run("deleted other type has no artifact", func(t *testing.T) {
		got := GetArtifactDeletedPayload(ctx, 7, 1, "reg", "1", "", "root", artifact.PackageTypeNPM, "n", urls, false)
		require.Nil(t, got.Artifact)
	})

	t.Run("common created and deleted", func(t *testing.T) {
		created := GetArtifactCreatedPayloadForCommonArtifacts(7, 1, artifact.PackageTypeNPM, "n", "1.0")
		deleted := GetArtifactDeletedPayloadForCommonArtifacts(7, 1, artifact.PackageTypeNPM, "n", "1.0")
		createdArtifact, ok := created.Artifact.(*registryevents.CommonArtifact)
		require.True(t, ok)
		deletedArtifact, ok := deleted.Artifact.(*registryevents.CommonArtifact)
		require.True(t, ok)
		for _, common := range []*registryevents.CommonArtifact{createdArtifact, deletedArtifact} {
			require.Equal(t, "n:1.0", common.Ref)
			require.Equal(t, "1.0", common.Version)
			require.Equal(t, artifact.PackageTypeNPM, common.Type)
		}
	})
}

func TestRb5GetRepoURLWithoutProtocol(t *testing.T) {
	ctx := context.Background()
	require.Equal(t, "host/a/b", GetRepoURLWithoutProtocol(ctx, "https://host/a/b"))
	require.Empty(t, GetRepoURLWithoutProtocol(ctx, "http://bad host/%zz"))
}

func rb5Map(t *testing.T, v any) map[string]any {
	t.Helper()
	m, ok := v.(map[string]any)
	require.True(t, ok)
	return m
}
