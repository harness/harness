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

package refcache

import (
	"context"
	"fmt"
	"strconv"

	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/app/store/cache"
	"github.com/harness/gitness/types"
)

type SpaceFinder struct {
	spaceIDCache   store.SpaceIDCache
	spacePathCache store.SpacePathCache
	evictor        cache.Evictor[*types.SpaceCore]
}

func NewSpaceFinder(
	spaceIDCache store.SpaceIDCache,
	spacePathCache store.SpacePathCache,
	evictor cache.Evictor[*types.SpaceCore],
) SpaceFinder {
	s := SpaceFinder{
		spaceIDCache:   spaceIDCache,
		spacePathCache: spacePathCache,
		evictor:        evictor,
	}

	return s
}

func (s SpaceFinder) MarkChanged(ctx context.Context, spaceCore *types.SpaceCore) {
	s.evictor.Evict(ctx, spaceCore)
}

func (s SpaceFinder) FindByID(ctx context.Context, spaceID int64) (*types.SpaceCore, error) {
	spaceCore, err := s.spaceIDCache.Get(ctx, spaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get space by ID from cache: %w", err)
	}

	return spaceCore, nil
}

func (s SpaceFinder) FindByRef(ctx context.Context, spaceRef string) (*types.SpaceCore, error) {
	spaceID, err := strconv.ParseInt(spaceRef, 10, 64)
	if err != nil || spaceID <= 0 {
		spacePath, err := s.spacePathCache.Get(ctx, spaceRef)
		if err != nil {
			return nil, fmt.Errorf("failed to get space ID by space path from cache: %w", err)
		}

		spaceID = spacePath.SpaceID
	}

	spaceCore, err := s.spaceIDCache.Get(ctx, spaceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get space by ID from cache: %w", err)
	}

	return spaceCore, nil
}
