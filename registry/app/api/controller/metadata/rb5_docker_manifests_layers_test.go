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

package metadata

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/auth/authz"
	"github.com/harness/gitness/app/services/refcache"
	corestore "github.com/harness/gitness/app/store"
	"github.com/harness/gitness/registry/app/api/interfaces"
	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/registry/app/driver/filesystem"
	"github.com/harness/gitness/registry/app/manifest/manifestlist"
	"github.com/harness/gitness/registry/app/manifest/schema2"
	"github.com/harness/gitness/registry/app/pkg/docker"
	"github.com/harness/gitness/registry/app/storage"
	"github.com/harness/gitness/registry/app/store"
	"github.com/harness/gitness/registry/gc"
	"github.com/harness/gitness/registry/types"
	basestore "github.com/harness/gitness/store"
	coretypes "github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/opencontainers/go-digest"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/stretchr/testify/require"
)

var errRb5Boom = errors.New("rb5 boom")

const (
	rb5Root  = "Root"
	rb5Reg   = "reg"
	rb5Image = "img"
)

type rb5Helper struct {
	interfaces.RegistryMetadataHelper
	err error
}

func (h rb5Helper) GetRegistryRequestBaseInfo(
	context.Context, string, string,
) (*types.RegistryRequestBaseInfo, error) {
	if h.err != nil {
		return nil, h.err
	}
	return &types.RegistryRequestBaseInfo{
		RootIdentifier: rb5Root, RegistryIdentifier: rb5Reg, RegistryID: 5, ParentID: 2, ParentRef: "root",
	}, nil
}

func (rb5Helper) GetPermissionChecks(*coretypes.SpaceCore, string, enum.Permission) []coretypes.PermissionCheck {
	return nil
}

type rb5Spaces struct {
	interfaces.SpaceFinder
	err error
}

func (s rb5Spaces) FindByRef(context.Context, string) (*coretypes.SpaceCore, error) {
	if s.err != nil {
		return nil, s.err
	}
	return &coretypes.SpaceCore{ID: 2, Path: "root", Identifier: "root"}, nil
}

type rb5Authz struct {
	authz.Authorizer
	allow bool
}

func (a rb5Authz) CheckAll(context.Context, *auth.Session, ...coretypes.PermissionCheck) (bool, error) {
	return a.allow, nil
}

type rb5Registries struct {
	store.RegistryRepository
	registry *types.Registry
	err      error
}

func (r rb5Registries) GetByParentIDAndName(
	context.Context, int64, string, ...types.QueryOption,
) (*types.Registry, error) {
	return r.registry, r.err
}

type rb5Manifests struct {
	store.ManifestRepository
	byDigest map[string]*types.Manifest
	byID     map[int64]*types.Manifest
	findErr  error
	getErr   error
}

func (m rb5Manifests) FindManifestByDigest(
	_ context.Context, _ int64, _ string, dgst types.Digest,
) (*types.Manifest, error) {
	if m.findErr != nil {
		return nil, m.findErr
	}
	found, ok := m.byDigest[string(dgst)]
	if !ok {
		return nil, basestore.ErrResourceNotFound
	}
	return found, nil
}

func (m rb5Manifests) Get(_ context.Context, id int64) (*types.Manifest, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	found, ok := m.byID[id]
	if !ok {
		return nil, basestore.ErrResourceNotFound
	}
	return found, nil
}

type rb5Tags struct {
	store.TagRepository
	tag       *types.Tag
	tagErr    error
	downloads map[string]int64
	countErr  error
}

func (t rb5Tags) FindTag(context.Context, int64, string, string) (*types.Tag, error) {
	return t.tag, t.tagErr
}

func (t rb5Tags) GetDownloadCountByManifests(context.Context, []string, int64) (map[string]int64, error) {
	return t.downloads, t.countErr
}

type rb5Images struct {
	store.ImageRepository
	err error
}

func (i rb5Images) GetByName(_ context.Context, _ int64, name string, _ ...types.QueryOption) (*types.Image, error) {
	if i.err != nil {
		return nil, i.err
	}
	return &types.Image{ID: 11, Name: name}, nil
}

type rb5Quarantine struct {
	store.QuarantineArtifactRepository
	items []*types.QuarantineArtifact
	err   error
}

func (q rb5Quarantine) GetByFilePath(
	context.Context, string, int64, string, string, *artifact.ArtifactType,
) ([]*types.QuarantineArtifact, error) {
	return q.items, q.err
}

type rb5GC struct{ gc.Service }

func (rb5GC) Start(
	context.Context, corestore.SpaceStore, store.BlobRepository, storage.StorageResolver, *coretypes.Config,
) {
}

type rb5Env struct {
	ctrl      *APIController
	manifests rb5Manifests
	tags      rb5Tags
	config    digest.Digest
	layerSize int64
}

type rb5Opts struct {
	helperErr    error
	spaceErr     error
	allow        bool
	registry     *types.Registry
	registryErr  error
	untagged     bool
	configJSON   string
	skipConfig   bool
	quarantine   []*types.QuarantineArtifact
	quarantineEr error
	imageErr     error
}

func rb5DefaultConfig() string {
	return `{"os":"linux","architecture":"amd64","history":[` +
		`{"created_by":"ADD rootfs","empty_layer":false},{"created_by":"CMD sh","empty_layer":true},` +
		`{"created_by":"RUN make","empty_layer":false}]}`
}

func rb5NewEnv(ctx context.Context, t *testing.T, opts rb5Opts) *rb5Env {
	t.Helper()

	fs, err := filesystem.FromParameters(map[string]any{"rootdirectory": t.TempDir(), "maxthreads": uint64(10)})
	require.NoError(t, err)
	resolver := storage.NewStaticStorageResolver(fs)
	storageService, err := storage.NewStorageService(resolver)
	require.NoError(t, err)
	app := docker.NewApp(
		ctx, nil, nil, &coretypes.Config{}, storageService, resolver, rb5GC{}, nil, refcache.SpaceFinder{},
	)

	configJSON := opts.configJSON
	if configJSON == "" {
		configJSON = rb5DefaultConfig()
	}
	configDigest := digest.FromString(configJSON)
	if !opts.skipConfig {
		blobs, blobErr := storageService.OciBlobsStore(ctx, rb5Reg, rb5Root, types.BlobLocator{})
		require.NoError(t, blobErr)
		_, err = blobs.Put(ctx, strings.ToLower(rb5Root), []byte(configJSON))
		require.NoError(t, err)
	}

	registry := opts.registry
	if registry == nil {
		registry = &types.Registry{ID: 5, Name: rb5Reg, ParentID: 2}
	}

	env := &rb5Env{config: configDigest, layerSize: 3 * 1024 * 1024}
	env.manifests = rb5Manifests{byDigest: map[string]*types.Manifest{}, byID: map[int64]*types.Manifest{}}
	env.tags = rb5Tags{tag: &types.Tag{ManifestID: 1}, downloads: map[string]int64{}}
	env.ctrl = &APIController{
		RegistryMetadataHelper:       rb5Helper{err: opts.helperErr},
		SpaceFinder:                  rb5Spaces{err: opts.spaceErr},
		Authorizer:                   rb5Authz{allow: opts.allow},
		RegistryRepository:           rb5Registries{registry: registry, err: opts.registryErr},
		ImageStore:                   rb5Images{err: opts.imageErr},
		QuarantineArtifactRepository: rb5Quarantine{items: opts.quarantine, err: opts.quarantineEr},
		UntaggedImagesEnabled:        func(context.Context) bool { return opts.untagged },
		app:                          app,
	}
	return env
}

// install wires the stores after the manifests were registered.
func (e *rb5Env) install() {
	e.ctrl.ManifestStore = e.manifests
	e.ctrl.TagStore = e.tags
}

func (e *rb5Env) ociManifest(id int64, mediaType string) *types.Manifest {
	payload, err := json.Marshal(map[string]any{
		"schemaVersion": 2,
		"mediaType":     mediaType,
		"config": map[string]any{
			"mediaType": v1.MediaTypeImageConfig, "digest": e.config.String(), "size": len(rb5DefaultConfig()),
		},
		"layers": []map[string]any{
			{"mediaType": v1.MediaTypeImageLayerGzip, "digest": digest.FromString("l1").String(), "size": e.layerSize},
			{"mediaType": v1.MediaTypeImageLayerGzip, "digest": digest.FromString("l2").String(), "size": 2048},
		},
	})
	if err != nil {
		panic(err)
	}
	m := &types.Manifest{
		ID: id, RegistryID: 5, ImageName: rb5Image, SchemaVersion: 2, MediaType: mediaType,
		Digest: digest.FromBytes(payload), Payload: types.Payload(payload), TotalSize: 5 * 1024 * 1024,
		Configuration: &types.Configuration{Digest: e.config}, CreatedAt: time.UnixMilli(1700000000000),
	}
	e.manifests.byID[id] = m
	e.manifests.byDigest[string(rb5Dgst(m))] = m
	return m
}

func rb5Dgst(m *types.Manifest) types.Digest {
	d, err := types.NewDigest(m.Digest)
	if err != nil {
		panic(err)
	}
	return d
}

func (e *rb5Env) indexManifest(children ...*types.Manifest) *types.Manifest {
	entries := make([]map[string]any, 0, len(children))
	for _, child := range children {
		entries = append(entries, map[string]any{
			"mediaType": child.MediaType, "digest": child.Digest.String(), "size": len(child.Payload),
			"platform": map[string]any{"architecture": "amd64", "os": "linux"},
		})
	}
	payload, err := json.Marshal(map[string]any{
		"schemaVersion": 2, "mediaType": v1.MediaTypeImageIndex, "manifests": entries,
	})
	if err != nil {
		panic(err)
	}
	const id = int64(1)
	m := &types.Manifest{
		ID: id, RegistryID: 5, ImageName: rb5Image, SchemaVersion: 2, MediaType: v1.MediaTypeImageIndex,
		Digest: digest.FromBytes(payload), Payload: types.Payload(payload), CreatedAt: time.UnixMilli(1700000000000),
	}
	e.manifests.byID[id] = m
	e.manifests.byDigest[string(rb5Dgst(m))] = m
	return m
}

func rb5ManifestsRequest(version string, byTag bool) artifact.GetDockerArtifactManifestsRequestObject {
	req := artifact.GetDockerArtifactManifestsRequestObject{
		RegistryRef: "root/reg", Artifact: rb5Image, Version: artifact.VersionPathParam(version),
	}
	if byTag {
		versionType := artifact.GetDockerArtifactManifestsParamsVersionTypeTAG
		req.Params.VersionType = &versionType
	}
	return req
}

func TestRb5GetDockerArtifactManifests(t *testing.T) {
	ctx := context.Background()

	t.Run("tagged image manifest", func(t *testing.T) {
		env := rb5NewEnv(ctx, t, rb5Opts{allow: true})
		m := env.ociManifest(1, v1.MediaTypeImageManifest)
		env.tags.downloads[string(rb5Dgst(m))] = 9
		env.install()

		resp, err := env.ctrl.GetDockerArtifactManifests(ctx, rb5ManifestsRequest("v1", true))
		require.NoError(t, err)
		ok, isOK := resp.(artifact.GetDockerArtifactManifests200JSONResponse)
		require.True(t, isOK, "%T", resp)
		require.Equal(t, rb5Image, ok.Data.ImageName)
		details := *ok.Data.Manifests
		require.Len(t, details, 1)
		require.Equal(t, m.Digest.String(), details[0].Digest)
		require.Equal(t, "linux/amd64", details[0].OsArch)
		require.Equal(t, int64(9), *details[0].DownloadsCount)
		require.False(t, *details[0].IsQuarantined)
		require.Equal(t, "5.00MB", *details[0].Size)
	})

	t.Run("docker schema2 manifest by digest", func(t *testing.T) {
		env := rb5NewEnv(ctx, t, rb5Opts{allow: true, untagged: true})
		m := env.ociManifest(1, schema2.MediaTypeManifest)
		env.install()

		resp, err := env.ctrl.GetDockerArtifactManifests(ctx, rb5ManifestsRequest(m.Digest.String(), false))
		require.NoError(t, err)
		ok, isOK := resp.(artifact.GetDockerArtifactManifests200JSONResponse)
		require.True(t, isOK, "%T", resp)
		require.Len(t, *ok.Data.Manifests, 1)
	})

	t.Run("quarantined manifest", func(t *testing.T) {
		env := rb5NewEnv(ctx, t, rb5Opts{
			allow: true, quarantine: []*types.QuarantineArtifact{{Reason: "malware"}},
		})
		env.ociManifest(1, v1.MediaTypeImageManifest)
		env.install()

		resp, err := env.ctrl.GetDockerArtifactManifests(ctx, rb5ManifestsRequest("v1", true))
		require.NoError(t, err)
		ok, isOK := resp.(artifact.GetDockerArtifactManifests200JSONResponse)
		require.True(t, isOK, "%T", resp)
		details := *ok.Data.Manifests
		require.True(t, *details[0].IsQuarantined)
		require.Equal(t, "malware", *details[0].QuarantineReason)
	})

	t.Run("manifest list", func(t *testing.T) {
		env := rb5NewEnv(ctx, t, rb5Opts{allow: true})
		child := env.ociManifest(2, v1.MediaTypeImageManifest)
		env.indexManifest(child)
		env.install()

		resp, err := env.ctrl.GetDockerArtifactManifests(ctx, rb5ManifestsRequest("v1", true))
		require.NoError(t, err)
		ok, isOK := resp.(artifact.GetDockerArtifactManifests200JSONResponse)
		require.True(t, isOK, "%T", resp)
		require.Len(t, *ok.Data.Manifests, 1)
		require.Equal(t, child.Digest.String(), (*ok.Data.Manifests)[0].Digest)
	})

	t.Run("manifest list child missing", func(t *testing.T) {
		env := rb5NewEnv(ctx, t, rb5Opts{allow: true})
		child := env.ociManifest(2, v1.MediaTypeImageManifest)
		env.indexManifest(child)
		delete(env.manifests.byDigest, string(rb5Dgst(child)))
		env.install()

		resp, err := env.ctrl.GetDockerArtifactManifests(ctx, rb5ManifestsRequest("v1", true))
		require.NoError(t, err)
		_, isErr := resp.(artifact.GetDockerArtifactManifests500JSONResponse)
		require.True(t, isErr, "%T", resp)
	})

	t.Run("upstream manifest list skips missing children", func(t *testing.T) {
		upstream := &types.Registry{ID: 5, Name: rb5Reg, ParentID: 2, Type: artifact.RegistryTypeUPSTREAM}
		env := rb5NewEnv(ctx, t, rb5Opts{allow: true, registry: upstream})
		child := env.ociManifest(2, v1.MediaTypeImageManifest)
		env.indexManifest(child)
		delete(env.manifests.byDigest, string(rb5Dgst(child)))
		env.install()

		resp, err := env.ctrl.GetDockerArtifactManifests(ctx, rb5ManifestsRequest("v1", true))
		require.NoError(t, err)
		ok, isOK := resp.(artifact.GetDockerArtifactManifests200JSONResponse)
		require.True(t, isOK, "%T", resp)
		require.Empty(t, *ok.Data.Manifests)
	})

	t.Run("manifest list child lookup fails", func(t *testing.T) {
		env := rb5NewEnv(ctx, t, rb5Opts{allow: true})
		child := env.ociManifest(2, v1.MediaTypeImageManifest)
		env.indexManifest(child)
		env.install()
		failing := env.manifests
		failing.findErr = errRb5Boom
		env.ctrl.ManifestStore = failing

		_, err := env.ctrl.ProcessManifest(ctx, &types.RegistryRequestBaseInfo{
			ParentID: 2, RegistryIdentifier: rb5Reg, RootIdentifier: rb5Root,
		}, rb5Image, "v1", true)
		require.ErrorIs(t, err, errRb5Boom)
	})

	t.Run("request failures", func(t *testing.T) {
		cases := []struct {
			name string
			opts rb5Opts
			want any
		}{
			{"bad registry ref", rb5Opts{helperErr: errRb5Boom}, artifact.GetDockerArtifactManifests400JSONResponse{}},
			{"space missing", rb5Opts{spaceErr: errRb5Boom}, artifact.GetDockerArtifactManifests400JSONResponse{}},
			{"forbidden", rb5Opts{allow: false}, artifact.GetDockerArtifactManifests403JSONResponse{}},
			{
				"registry lookup", rb5Opts{allow: true, registryErr: errRb5Boom},
				artifact.GetDockerArtifactManifests500JSONResponse{},
			},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				env := rb5NewEnv(ctx, t, tc.opts)
				env.install()
				resp, err := env.ctrl.GetDockerArtifactManifests(ctx, rb5ManifestsRequest("v1", true))
				require.NoError(t, err)
				require.IsType(t, tc.want, resp)
			})
		}
	})
}

func TestRb5ProcessManifestErrors(t *testing.T) {
	ctx := context.Background()
	info := &types.RegistryRequestBaseInfo{ParentID: 2, RegistryIdentifier: rb5Reg, RootIdentifier: rb5Root, RegistryID: 5}

	t.Run("invalid digest", func(t *testing.T) {
		env := rb5NewEnv(ctx, t, rb5Opts{untagged: true})
		env.install()
		_, err := env.ctrl.ProcessManifest(ctx, info, rb5Image, "not-a-digest", false)
		require.Error(t, err)
	})

	t.Run("digest not found", func(t *testing.T) {
		env := rb5NewEnv(ctx, t, rb5Opts{untagged: true})
		env.install()
		_, err := env.ctrl.ProcessManifest(ctx, info, rb5Image, digest.FromString("x").String(), false)
		require.ErrorIs(t, err, basestore.ErrResourceNotFound)
	})

	t.Run("tag lookup fails", func(t *testing.T) {
		env := rb5NewEnv(ctx, t, rb5Opts{})
		env.tags.tagErr = errRb5Boom
		env.install()
		_, err := env.ctrl.ProcessManifest(ctx, info, rb5Image, "v1", true)
		require.ErrorIs(t, err, errRb5Boom)
	})

	t.Run("manifest lookup fails", func(t *testing.T) {
		env := rb5NewEnv(ctx, t, rb5Opts{})
		env.manifests.getErr = errRb5Boom
		env.install()
		_, err := env.ctrl.ProcessManifest(ctx, info, rb5Image, "v1", true)
		require.ErrorIs(t, err, errRb5Boom)
	})

	t.Run("manifest cannot be decoded", func(t *testing.T) {
		env := rb5NewEnv(ctx, t, rb5Opts{})
		m := env.ociManifest(1, v1.MediaTypeImageManifest)
		m.SchemaVersion = 1
		env.install()
		_, err := env.ctrl.ProcessManifest(ctx, info, rb5Image, "v1", true)
		require.Error(t, err)
	})

	t.Run("config blob missing", func(t *testing.T) {
		env := rb5NewEnv(ctx, t, rb5Opts{skipConfig: true})
		env.ociManifest(1, v1.MediaTypeImageManifest)
		env.install()
		_, err := env.ctrl.ProcessManifest(ctx, info, rb5Image, "v1", true)
		require.ErrorContains(t, err, "failed to get content for image config")
	})

	t.Run("config is not json", func(t *testing.T) {
		env := rb5NewEnv(ctx, t, rb5Opts{configJSON: "not json"})
		env.ociManifest(1, v1.MediaTypeImageManifest)
		env.install()
		_, err := env.ctrl.ProcessManifest(ctx, info, rb5Image, "v1", true)
		require.ErrorContains(t, err, "failed to unmarshal manifest config")
	})

	t.Run("image lookup fails", func(t *testing.T) {
		env := rb5NewEnv(ctx, t, rb5Opts{imageErr: errRb5Boom})
		env.ociManifest(1, v1.MediaTypeImageManifest)
		env.install()
		_, err := env.ctrl.ProcessManifest(ctx, info, rb5Image, "v1", true)
		require.ErrorIs(t, err, errRb5Boom)
	})

	t.Run("quarantine lookup fails", func(t *testing.T) {
		env := rb5NewEnv(ctx, t, rb5Opts{quarantineEr: errRb5Boom})
		env.ociManifest(1, v1.MediaTypeImageManifest)
		env.install()
		_, err := env.ctrl.ProcessManifest(ctx, info, rb5Image, "v1", true)
		require.ErrorIs(t, err, errRb5Boom)
	})

	t.Run("download counts are optional", func(t *testing.T) {
		env := rb5NewEnv(ctx, t, rb5Opts{})
		env.ociManifest(1, v1.MediaTypeImageManifest)
		env.tags.countErr = errRb5Boom
		env.install()
		details, err := env.ctrl.ProcessManifest(ctx, info, rb5Image, "v1", true)
		require.NoError(t, err)
		require.Nil(t, details[0].DownloadsCount)
	})

	t.Run("not found tag row", func(t *testing.T) {
		env := rb5NewEnv(ctx, t, rb5Opts{})
		env.tags.tagErr = sql.ErrNoRows
		env.tags.tag = &types.Tag{ManifestID: 99}
		env.install()
		_, err := env.ctrl.ProcessManifest(ctx, info, rb5Image, "v1", true)
		require.ErrorIs(t, err, basestore.ErrResourceNotFound)
	})

	t.Run("unknown manifest type yields no details", func(t *testing.T) {
		env := rb5NewEnv(ctx, t, rb5Opts{})
		m := env.ociManifest(1, v1.MediaTypeImageManifest)
		m.MediaType = manifestlist.MediaTypeManifestList
		m.Payload = types.Payload(
			`{"schemaVersion":2,"mediaType":"` + manifestlist.MediaTypeManifestList + `","manifests":[]}`)
		env.install()
		details, err := env.ctrl.ProcessManifest(ctx, info, rb5Image, "v1", true)
		require.NoError(t, err)
		require.Empty(t, details)
	})
}

func rb5LayersRequest(dgst string) artifact.GetDockerArtifactLayersRequestObject {
	req := artifact.GetDockerArtifactLayersRequestObject{RegistryRef: "root/reg", Artifact: rb5Image}
	req.Params.Digest = artifact.DigestParam(dgst)
	return req
}

func TestRb5GetDockerArtifactLayers(t *testing.T) {
	ctx := context.Background()

	t.Run("layers follow the image history", func(t *testing.T) {
		env := rb5NewEnv(ctx, t, rb5Opts{allow: true})
		m := env.ociManifest(1, v1.MediaTypeImageManifest)
		env.install()

		resp, err := env.ctrl.GetDockerArtifactLayers(ctx, rb5LayersRequest(m.Digest.String()))
		require.NoError(t, err)
		ok, isOK := resp.(artifact.GetDockerArtifactLayers200JSONResponse)
		require.True(t, isOK, "%T", resp)
		require.Equal(t, "linux/amd64", *ok.Data.OsArch)
		layers := *ok.Data.Layers
		require.Len(t, layers, 3)
		require.Equal(t, "3.00 MB", *layers[0].Size)
		require.Equal(t, DefaultSize, *layers[1].Size)
		require.Equal(t, "2.00 KB", *layers[2].Size)
		require.Equal(t, "RUN make", layers[2].Command)
	})

	t.Run("docker schema2 layers", func(t *testing.T) {
		env := rb5NewEnv(ctx, t, rb5Opts{allow: true})
		m := env.ociManifest(1, schema2.MediaTypeManifest)
		env.install()
		resp, err := env.ctrl.GetDockerArtifactLayers(ctx, rb5LayersRequest(m.Digest.String()))
		require.NoError(t, err)
		_, isOK := resp.(artifact.GetDockerArtifactLayers200JSONResponse)
		require.True(t, isOK, "%T", resp)
	})

	t.Run("failures", func(t *testing.T) {
		valid := digest.FromString("manifest").String()
		cases := []struct {
			name   string
			opts   rb5Opts
			digest string
			setup  func(env *rb5Env)
			want   any
		}{
			{"space missing", rb5Opts{spaceErr: errRb5Boom}, valid, nil, artifact.GetDockerArtifactLayers400JSONResponse{}},
			{"forbidden", rb5Opts{allow: false}, valid, nil, artifact.GetDockerArtifactLayers403JSONResponse{}},
			{"invalid digest", rb5Opts{allow: true}, "bad", nil, artifact.GetDockerArtifactLayers500JSONResponse{}},
			{
				"registry lookup", rb5Opts{allow: true, registryErr: errRb5Boom}, valid, nil,
				artifact.GetDockerArtifactLayers500JSONResponse{},
			},
			{"nil registry", rb5Opts{allow: true, registry: nil}, valid, func(env *rb5Env) {
				env.ctrl.RegistryRepository = rb5Registries{}
			}, artifact.GetDockerArtifactLayers500JSONResponse{}},
			{"manifest missing", rb5Opts{allow: true}, valid, nil, artifact.GetDockerArtifactLayers500JSONResponse{}},
			{"manifest lookup fails", rb5Opts{allow: true}, valid, func(env *rb5Env) {
				env.manifests.findErr = errRb5Boom
			}, artifact.GetDockerArtifactLayers500JSONResponse{}},
			{"config missing", rb5Opts{allow: true, skipConfig: true}, "", func(env *rb5Env) {
				env.ociManifest(1, v1.MediaTypeImageManifest)
			}, artifact.GetDockerArtifactLayers500JSONResponse{}},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				env := rb5NewEnv(ctx, t, tc.opts)
				dgst := tc.digest
				if tc.setup != nil {
					tc.setup(env)
				}
				if dgst == "" {
					for key := range env.manifests.byDigest {
						dgst = key
					}
				}
				env.install()
				resp, err := env.ctrl.GetDockerArtifactLayers(ctx, rb5LayersRequest(dgst))
				require.NoError(t, err)
				require.IsType(t, tc.want, resp)
			})
		}
	})

	t.Run("manifest cannot be converted", func(t *testing.T) {
		env := rb5NewEnv(ctx, t, rb5Opts{allow: true})
		m := env.ociManifest(1, v1.MediaTypeImageManifest)
		m.SchemaVersion = 1
		env.install()
		resp, err := env.ctrl.GetDockerArtifactLayers(ctx, rb5LayersRequest(m.Digest.String()))
		require.NoError(t, err)
		require.IsType(t, artifact.GetDockerArtifactLayers500JSONResponse{}, resp)
	})

	t.Run("manifest lists have no layers", func(t *testing.T) {
		env := rb5NewEnv(ctx, t, rb5Opts{allow: true})
		child := env.ociManifest(2, v1.MediaTypeImageManifest)
		index := env.indexManifest(child)
		index.Configuration = &types.Configuration{Digest: env.config}
		env.install()
		resp, err := env.ctrl.GetDockerArtifactLayers(ctx, rb5LayersRequest(index.Digest.String()))
		require.NoError(t, err)
		require.IsType(t, artifact.GetDockerArtifactLayers500JSONResponse{}, resp)
	})
}

func TestRb5GetManifestLayersAndSizes(t *testing.T) {
	_, err := getManifestLayers(&manifestlist.DeserializedManifestList{})
	require.Error(t, err)

	require.Equal(t, "0 B", GetSizeString(0))
	require.Equal(t, "1023 B", GetSizeString(1023))
	require.Equal(t, "1.00 KB", GetSizeString(KB))
	require.Equal(t, "1.50 MB", GetSizeString(MB+MB/2))
	require.Equal(t, "2.00 GB", GetSizeString(2*GB))
}
