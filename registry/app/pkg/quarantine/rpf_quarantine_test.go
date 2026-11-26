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

package quarantine

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/harness/gitness/app/api/usererror"
	cache2 "github.com/harness/gitness/app/store/cache"
	"github.com/harness/gitness/cache"
	"github.com/harness/gitness/pubsub"
	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/registry/app/store"
	"github.com/harness/gitness/registry/types"
	store2 "github.com/harness/gitness/store"

	"github.com/opencontainers/go-digest"
	"github.com/stretchr/testify/require"
)

type rpfFakeQuarantineRepo struct {
	store.QuarantineArtifactRepository
	result []*types.QuarantineArtifact
	err    error
	gotReg int64
	gotImg string
	gotVer string
}

func (r *rpfFakeQuarantineRepo) GetByFilePath(
	_ context.Context, _ string, registryID int64, image string, version string, _ *artifact.ArtifactType,
) ([]*types.QuarantineArtifact, error) {
	r.gotReg, r.gotImg, r.gotVer = registryID, image, version
	return r.result, r.err
}

type rpfFakeManifestRepo struct {
	store.ManifestRepository
	digest types.Digest
	err    error
	calls  int
}

func (r *rpfFakeManifestRepo) FindManifestDigestByTagName(
	context.Context, int64, string, string,
) (types.Digest, error) {
	r.calls++
	return r.digest, r.err
}

func rpfSHA256() digest.Digest {
	return digest.Digest("sha256:" + strings.Repeat("a", 64))
}

func rpfDBDigest(t *testing.T) types.Digest {
	t.Helper()
	d, err := types.NewDigest(rpfSHA256())
	require.NoError(t, err)
	return d
}

func TestRpfServiceCheckArtifactQuarantineStatus(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name    string
		result  []*types.QuarantineArtifact
		err     error
		want    bool
		wantErr bool
	}{
		{"quarantined", []*types.QuarantineArtifact{{ID: "1"}}, nil, true, false},
		{"not quarantined", nil, nil, false, false},
		{"repository error", nil, errors.New("db down"), false, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := &rpfFakeQuarantineRepo{result: tc.result, err: tc.err}
			svc := NewService(repo, nil)
			got, err := svc.CheckArtifactQuarantineStatus(ctx, 4, "img", "1.0", nil)
			if tc.wantErr {
				require.ErrorContains(t, err, "failed to check quarantine status")
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
			require.Equal(t, int64(4), repo.gotReg)
			require.Equal(t, "img", repo.gotImg)
			require.Equal(t, "1.0", repo.gotVer)
		})
	}
}

func TestRpfServiceResolveDigest(t *testing.T) {
	ctx := context.Background()

	t.Run("explicit digest wins over tag", func(t *testing.T) {
		manifests := &rpfFakeManifestRepo{}
		svc := NewService(nil, manifests)
		got, err := svc.ResolveDigest(ctx, 1, "img", "latest", string(rpfSHA256()))
		require.NoError(t, err)
		require.Equal(t, rpfSHA256(), got)
		require.Zero(t, manifests.calls)
	})

	t.Run("invalid digest", func(t *testing.T) {
		svc := NewService(nil, &rpfFakeManifestRepo{})
		_, err := svc.ResolveDigest(ctx, 1, "img", "", "not-a-digest")
		require.ErrorContains(t, err, "failed to parse digest")
	})

	t.Run("tag is resolved through the manifest repository", func(t *testing.T) {
		svc := NewService(nil, &rpfFakeManifestRepo{digest: rpfDBDigest(t)})
		got, err := svc.ResolveDigest(ctx, 1, "img", "latest", "")
		require.NoError(t, err)
		require.Equal(t, rpfSHA256(), got)
	})

	t.Run("unknown tag resolves to nothing", func(t *testing.T) {
		svc := NewService(nil, &rpfFakeManifestRepo{err: store2.ErrResourceNotFound})
		got, err := svc.ResolveDigest(ctx, 1, "img", "latest", "")
		require.NoError(t, err)
		require.Empty(t, got)
	})

	t.Run("manifest repository failure", func(t *testing.T) {
		svc := NewService(nil, &rpfFakeManifestRepo{err: errors.New("db down")})
		_, err := svc.ResolveDigest(ctx, 1, "img", "latest", "")
		require.ErrorContains(t, err, "failed to find manifest digest")
	})

	t.Run("stored digest cannot be parsed", func(t *testing.T) {
		svc := NewService(nil, &rpfFakeManifestRepo{digest: types.Digest("zz")})
		_, err := svc.ResolveDigest(ctx, 1, "img", "latest", "")
		require.ErrorContains(t, err, "failed to parse digest")
	})

	t.Run("neither digest nor tag", func(t *testing.T) {
		svc := NewService(nil, &rpfFakeManifestRepo{})
		got, err := svc.ResolveDigest(ctx, 1, "img", "", "")
		require.NoError(t, err)
		require.Empty(t, got)
	})
}

type rpfFakeQuarantineCache struct {
	quarantined bool
	err         error
	gotKey      CacheKey
}

func (c *rpfFakeQuarantineCache) Stats() (int64, int64) { return 0, 0 }

func (c *rpfFakeQuarantineCache) Get(_ context.Context, key CacheKey) (bool, error) {
	c.gotKey = key
	return c.quarantined, c.err
}

func (c *rpfFakeQuarantineCache) Evict(context.Context, CacheKey) {}

var _ cache.Cache[CacheKey, bool] = (*rpfFakeQuarantineCache)(nil)

type rpfFakePubSub struct {
	pubsub.PubSub
	topic   string
	payload []byte
}

func (p *rpfFakePubSub) Publish(_ context.Context, topic string, payload []byte, _ ...pubsub.PublishOption) error {
	p.topic = topic
	p.payload = payload
	return nil
}

func TestRpfFinderCheckArtifactQuarantineStatus(t *testing.T) {
	ctx := context.Background()

	c := &rpfFakeQuarantineCache{quarantined: true}
	f := NewFinder(nil, c, cache2.Evictor[*CacheKey]{})
	err := f.CheckArtifactQuarantineStatus(ctx, 9, "img", "1.0", nil)
	require.ErrorIs(t, err, usererror.ErrQuarantinedArtifact)
	require.Equal(t, CacheKey{RegistryID: 9, Image: "img", Version: "1.0"}, c.gotKey)

	f = NewFinder(nil, &rpfFakeQuarantineCache{}, cache2.Evictor[*CacheKey]{})
	require.NoError(t, f.CheckArtifactQuarantineStatus(ctx, 9, "img", "1.0", nil))

	f = NewFinder(nil, &rpfFakeQuarantineCache{err: errors.New("cache down")}, cache2.Evictor[*CacheKey]{})
	err = f.CheckArtifactQuarantineStatus(ctx, 9, "img", "1.0", nil)
	require.ErrorContains(t, err, "failed to check quarantine status")
}

func TestRpfFinderCheckOCIManifestQuarantineStatus(t *testing.T) {
	ctx := context.Background()
	wantKey := CacheKey{RegistryID: 3, Image: "img", Version: rpfDBDigest(t).String()}

	t.Run("quarantined digest", func(t *testing.T) {
		c := &rpfFakeQuarantineCache{quarantined: true}
		f := NewFinder(NewService(nil, &rpfFakeManifestRepo{}), c, cache2.Evictor[*CacheKey]{})
		err := f.CheckOCIManifestQuarantineStatus(ctx, 3, "img", "", string(rpfSHA256()))
		require.ErrorIs(t, err, usererror.ErrQuarantinedArtifact)
		require.Equal(t, wantKey, c.gotKey)
	})

	t.Run("tag resolved to a clean digest", func(t *testing.T) {
		manifests := &rpfFakeManifestRepo{digest: rpfDBDigest(t)}
		f := NewFinder(NewService(nil, manifests), &rpfFakeQuarantineCache{}, cache2.Evictor[*CacheKey]{})
		require.NoError(t, f.CheckOCIManifestQuarantineStatus(ctx, 3, "img", "latest", ""))
		require.Equal(t, 1, manifests.calls)
	})

	t.Run("nothing to resolve skips the cache", func(t *testing.T) {
		c := &rpfFakeQuarantineCache{quarantined: true}
		f := NewFinder(NewService(nil, &rpfFakeManifestRepo{}), c, cache2.Evictor[*CacheKey]{})
		require.NoError(t, f.CheckOCIManifestQuarantineStatus(ctx, 3, "img", "", ""))
		require.Equal(t, CacheKey{}, c.gotKey)
	})

	t.Run("resolution error is wrapped", func(t *testing.T) {
		f := NewFinder(NewService(nil, &rpfFakeManifestRepo{}), &rpfFakeQuarantineCache{}, cache2.Evictor[*CacheKey]{})
		err := f.CheckOCIManifestQuarantineStatus(ctx, 3, "img", "", "bad")
		require.ErrorContains(t, err, "failed to parse digest")
	})
}

func TestRpfFinderEvictCache(t *testing.T) {
	ctx := context.Background()
	bus := &rpfFakePubSub{}
	f := NewFinder(nil, &rpfFakeQuarantineCache{}, cache2.NewEvictor[*CacheKey]("ns", "quarantine-topic", bus))
	f.EvictCache(ctx, 1, "img", "1.0", nil)
	require.Contains(t, bus.topic, "quarantine-topic")
	require.NotEmpty(t, bus.payload)

	// Without a message bus eviction is a no-op.
	NewFinder(nil, &rpfFakeQuarantineCache{}, cache2.Evictor[*CacheKey]{}).EvictCache(ctx, 1, "img", "1.0", nil)
}

func TestRpfQuarantineCacheGetter(t *testing.T) {
	ctx := context.Background()
	repo := &rpfFakeQuarantineRepo{result: []*types.QuarantineArtifact{{ID: "1"}}}
	g := quarantineCacheGetter{service: NewService(repo, nil)}
	got, err := g.Find(ctx, CacheKey{RegistryID: 2, Image: "img", Version: "1.0"})
	require.NoError(t, err)
	require.True(t, got)
	require.Equal(t, int64(2), repo.gotReg)

	// The getter plugs into a cache implementation.
	c := cache.NewNoCache[CacheKey, bool](g)
	got, err = c.Get(ctx, CacheKey{RegistryID: 2, Image: "img", Version: "1.0"})
	require.NoError(t, err)
	require.True(t, got)
}
