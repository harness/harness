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
	"testing"

	cache2 "github.com/harness/gitness/app/store/cache"
	"github.com/harness/gitness/pubsub"
	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/registry/app/store"
	"github.com/harness/gitness/registry/types"

	"github.com/stretchr/testify/require"
)

type rpgQuarantineRepo struct {
	store.QuarantineArtifactRepository
	result []*types.QuarantineArtifact
	calls  int
}

func (r *rpgQuarantineRepo) GetByFilePath(
	context.Context, string, int64, string, string, *artifact.ArtifactType,
) ([]*types.QuarantineArtifact, error) {
	r.calls++
	return r.result, nil
}

func TestRpgWireProvidersBuildAWorkingFinder(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	repo := &rpgQuarantineRepo{}
	ps := pubsub.NewInMemory()

	evictor := ProvideEvictorQuarantine(ps)
	service := ProvideService(repo, nil)
	require.NotNil(t, service)
	quarantineCache := ProvideQuarantineCache(ctx, service, evictor)
	require.NotNil(t, quarantineCache)
	finder := ProvideFinder(service, quarantineCache, evictor)
	require.NotNil(t, finder)

	version := "01aaaa"
	require.NoError(t, finder.CheckArtifactQuarantineStatus(ctx, 1, "img", version, nil))
	require.NoError(t, finder.CheckArtifactQuarantineStatus(ctx, 1, "img", version, nil))
	require.Equal(t, 1, repo.calls, "the second lookup is served from the cache")

	// Evicting publishes on the bus without failing.
	require.NotPanics(t, func() { finder.EvictCache(ctx, 1, "img", version, nil) })
}

func TestRpgFinderOCIManifestStatusWithoutDigest(t *testing.T) {
	ctx := context.Background()
	finder := NewFinder(NewService(&rpgQuarantineRepo{}, nil), nil, cache2.Evictor[*CacheKey]{})

	// Nothing to resolve means nothing to check.
	require.NoError(t, finder.CheckOCIManifestQuarantineStatus(ctx, 1, "img", "", ""))
	// A digest that is not valid is rejected.
	require.Error(t, finder.CheckOCIManifestQuarantineStatus(ctx, 1, "img", "", "not-a-digest"))
}
