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

package docker

import (
	"context"
	"testing"

	"github.com/harness/gitness/registry/app/driver/filesystem"
	"github.com/harness/gitness/registry/app/pkg"
	registrystorage "github.com/harness/gitness/registry/app/storage"
	registrytypes "github.com/harness/gitness/registry/types"
	"github.com/harness/gitness/types"

	"github.com/stretchr/testify/require"
)

func TestRpgWireProviders(t *testing.T) {
	env := rpgNewEnv(t)
	rpgSetRegistries(t, nil, nil)

	local := LocalRegistryProvider(
		env.app, env.ms, env.blobs, env.regDao, nil, env.manifests, env.regBlobs, nil, env.tags, env.images,
		env.artifacts, nil, nil, env.gc, rpgTx{}, nil, env.repl, env.hook,
	)
	require.NotNil(t, local)
	require.Same(t, env.app, local.App)

	reporter, _ := rpgArtifactReporter(t)
	svc := ManifestServiceProvider(
		env.regDao, env.manifests, env.blobs, nil, env.refs, env.tags, env.images, env.artifacts, env.layers,
		env.gc, rpgTx{}, env.events, rpgSpaceFinder(nil), env.mappings, &reporter, rpgURLProvider{}, &rpgAudit{},
		env.regBlobs,
	)
	require.NotNil(t, svc)
	concrete, ok := svc.(*manifestService)
	require.True(t, ok)
	require.True(t, concrete.untaggedImagesEnabled(context.Background()))

	ctl := &rpgProxyCtl{}
	remote := RemoteRegistryProvider(local, env.app, &rpgUpstreamRepo{}, rpgSpaceFinder(nil), nil, ctl)
	require.NotNil(t, remote)

	controller := ControllerProvider(
		local, remote, pkg.NewCoreController(nil, nil), nil, nil, &DBStore{}, rpgSpaceFinder(nil),
	)
	require.NotNil(t, controller)
	require.Same(t, local, pkg.TypeRegistry[pkg.LocalRegistry])

	dbStore := DBStoreProvider(env.blobs, env.images, env.artifacts, nil, nil, env.manifests, nil)
	require.Same(t, env.blobs, dbStore.BlobRepo)
	require.Same(t, env.manifests, dbStore.ManifestDao)

	require.NotNil(t, ProvideReporter())
	require.NotNil(t, ProvideProxyController(local, svc, nil, rpgSpaceFinder(nil)))
	require.Len(t, getManifestCacheHandler(local, svc), 4)
}

func TestRpgStorageAndBucketProviders(t *testing.T) {
	resolver := registrystorage.NewStaticStorageResolver(
		filesystem.New(filesystem.DriverParameters{RootDirectory: t.TempDir(), MaxThreads: 2}),
	)
	cfg := &types.Config{}
	cfg.Registry.Storage.S3Storage.Delete = true
	cfg.Registry.Storage.S3Storage.Redirect = true
	require.NotNil(t, StorageServiceProvider(cfg, resolver))
	require.NotNil(t, GetStorageService(&types.Config{}, resolver))

	service := StorageServiceProvider(cfg, resolver)
	factory := ProvideOciBlobStore(service)
	store, err := factory(context.Background(), "reg", "root", registrytypes.BlobLocator{})
	require.NoError(t, err)
	require.NotNil(t, store)

	buckets := ProvideBucketService(factory)
	require.Nil(t, buckets.GetBlobStore(context.Background(), "reg", "root", int64(1), "sha", "root"))
}

func TestRpgLocalPullManifestDelegatesToManifestExist(t *testing.T) {
	env := rpgNewEnv(t)
	mt, payload := rpgDockerManifest(t, "pull")
	env.addManifest(t, 1, mt, payload)
	info := rpgInfo(t)
	info.Digest = rpgPayloadDigest(payload).String()

	headers, desc, m, errs := env.local.PullManifest(context.Background(), info, []string{mt}, nil)
	require.Empty(t, errs)
	require.NotNil(t, m)
	require.Equal(t, rpgPayloadDigest(payload), desc.Digest)
	require.Equal(t, mt, headers.Headers["Content-Type"])
}

func TestRpgNewAppStartsGarbageCollection(t *testing.T) {
	env := rpgNewEnv(t)
	cfg := &types.Config{}
	app := NewApp(
		context.Background(), env.blobs, nil, cfg, env.app.storageService, nil, env.gc, &rpgBucket{},
		rpgSpaceFinder(nil),
	)
	require.NotNil(t, app)
	require.NotEmpty(t, cfg.Registry.HTTP.Secret, "a secret is generated when none is configured")
	require.Same(t, env.app.storageService, app.StorageService())
}
