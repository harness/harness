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
	"errors"
	"testing"

	"github.com/harness/gitness/app/services/refcache"
	"github.com/harness/gitness/app/store/cache"
	"github.com/harness/gitness/registry/app/dist_temp/errcode"
	"github.com/harness/gitness/registry/app/pkg"
	"github.com/harness/gitness/registry/app/storage"
	registrytypes "github.com/harness/gitness/registry/types"
	"github.com/harness/gitness/types"

	"github.com/opencontainers/go-digest"
	"github.com/stretchr/testify/require"
)

type rpfSpaceCache struct{}

func (rpfSpaceCache) Stats() (int64, int64) { return 0, 0 }

func (rpfSpaceCache) Get(_ context.Context, key int64) (*types.SpaceCore, error) {
	if key != 7 {
		return nil, errors.New("space not found")
	}
	return &types.SpaceCore{ID: 7, Identifier: "acme"}, nil
}

func (rpfSpaceCache) Evict(context.Context, int64) {}

type rpfBucketService struct {
	result  *BlobStore
	calls   int
	gotID   any
	gotRoot string
}

func (b *rpfBucketService) GetBlobStore(
	_ context.Context, _ string, _ string, blobID any, _ string, rootSpaceIdentifier string,
) *BlobStore {
	b.calls++
	b.gotID = blobID
	b.gotRoot = rootSpaceIdentifier
	return b.result
}

type rpfOciStore struct {
	storage.OciBlobStore
}

func rpfRegistryInfo() pkg.RegistryInfo {
	return pkg.RegistryInfo{
		ArtifactInfo: &pkg.ArtifactInfo{RegIdentifier: "reg", BaseInfo: &pkg.BaseInfo{RootIdentifier: "root"}},
		Reference:    "upload-id",
		Digest:       string(digest.FromString("blob")),
	}
}

func TestRpfAppGetBlobsContextUsesTheBucketService(t *testing.T) {
	ctx := context.Background()
	finder := refcache.NewSpaceFinder(rpfSpaceCache{}, nil, nil, cache.Evictor[*types.SpaceCore]{})
	ociStore := &rpfOciStore{}
	buckets := &rpfBucketService{result: &BlobStore{OciStore: ociStore}}
	app := &App{Context: ctx, bucketService: buckets, spaceFinder: finder}

	got, err := app.GetBlobsContext(ctx, rpfRegistryInfo(), registrytypes.BlobLocator{BlobID: 5, RootParentID: 7})
	require.NoError(t, err)
	require.Same(t, ociStore, got.OciBlobStore)
	require.Equal(t, "upload-id", got.UUID)
	require.Equal(t, digest.FromString("blob"), got.Digest)
	require.Equal(t, 1, buckets.calls)
	require.Equal(t, int64(5), buckets.gotID)
	require.Equal(t, "acme", buckets.gotRoot)

	_, err = app.GetBlobsContext(ctx, rpfRegistryInfo(), registrytypes.BlobLocator{BlobID: 5, RootParentID: 99})
	require.ErrorContains(t, err, "could not find blob store root")
	require.Equal(t, 1, buckets.calls)
}

func TestRpfAppConfigureSecret(t *testing.T) {
	app := &App{Context: context.Background()}

	cfg := &types.Config{}
	app.configureSecret(cfg)
	require.Len(t, cfg.Registry.HTTP.Secret, randomSecretSize)

	generated := cfg.Registry.HTTP.Secret
	app.configureSecret(cfg)
	require.Equal(t, generated, cfg.Registry.HTTP.Secret, "an existing secret is kept")

	cfg = &types.Config{}
	cfg.Registry.HTTP.Secret = "shared-secret"
	app.configureSecret(cfg)
	require.Equal(t, "shared-secret", cfg.Registry.HTTP.Secret)
	require.Nil(t, app.StorageService())
}

func TestRpfLogError(t *testing.T) {
	require.NotPanics(t, func() {
		LogError(errcode.Errors{errors.New("first"), errors.New("second")})
		LogError(nil)
	})
}
