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

package authz

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/harness/gitness/app/paths"
	"github.com/harness/gitness/app/services/refcache"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/cache"
	gitness_store "github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"golang.org/x/exp/slices"
)

type PermissionCacheKey struct {
	PrincipalID int64
	SpaceRef    string
	Permission  enum.Permission
}
type PermissionCache cache.Cache[PermissionCacheKey, bool]

func NewPermissionCache(
	spaceFinder refcache.SpaceFinder,
	membershipStore store.MembershipStore,
	cacheDuration time.Duration,
) PermissionCache {
	return cache.New[PermissionCacheKey, bool](permissionCacheGetter{
		spaceFinder:     spaceFinder,
		membershipStore: membershipStore,
	}, cacheDuration)
}

type permissionCacheGetter struct {
	spaceFinder     refcache.SpaceFinder
	membershipStore store.MembershipStore
}

func (g permissionCacheGetter) Find(ctx context.Context, key PermissionCacheKey) (bool, error) {
	spaceRef := key.SpaceRef
	principalID := key.PrincipalID

	// Find the first existing space.
	space, err := g.findFirstExistingSpace(ctx, spaceRef)
	// authz fails if no active space is found on the path; admins can still operate on deleted top-level spaces.
	if errors.Is(err, gitness_store.ErrResourceNotFound) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to find an existing space on path '%s': %w", spaceRef, err)
	}

	// limit the depth to be safe (e.g. root/space1/space2 => maxDepth of 3)
	maxDepth := len(paths.Segments(spaceRef))

	for range maxDepth {
		// Find the membership in the current space.
		membership, err := g.membershipStore.Find(ctx, types.MembershipKey{
			SpaceID:     space.ID,
			PrincipalID: principalID,
		})
		if err != nil && !errors.Is(err, gitness_store.ErrResourceNotFound) {
			return false, fmt.Errorf("failed to find membership: %w", err)
		}

		// If the membership is defined in the current space, check if the user has the required permission.
		if membership != nil &&
			roleHasPermission(membership.Role, key.Permission) {
			return true, nil
		}

		// If membership with the requested permission has not been found in the current space,
		// move to the parent space, if any.

		if space.ParentID == 0 {
			return false, nil
		}

		space, err = g.spaceFinder.FindByID(ctx, space.ParentID)
		if err != nil {
			return false, fmt.Errorf("failed to find parent space with id %d: %w", space.ParentID, err)
		}
	}

	return false, nil
}

func roleHasPermission(role enum.MembershipRole, permission enum.Permission) bool {
	_, hasRole := slices.BinarySearch(role.Permissions(), permission)
	return hasRole
}

// findFirstExistingSpace returns the initial or first existing ancestor space (permissions are inherited).
func (g permissionCacheGetter) findFirstExistingSpace(ctx context.Context, spaceRef string) (*types.SpaceCore, error) {
	for {
		space, err := g.spaceFinder.FindByRef(ctx, spaceRef)
		if err == nil {
			return space, nil
		}

		if !errors.Is(err, gitness_store.ErrResourceNotFound) {
			return nil, fmt.Errorf("failed to find space '%s': %w", spaceRef, err)
		}

		// check whether parent space exists as permissions are inherited.
		spaceRef, _, err = paths.DisectLeaf(spaceRef)
		if err != nil {
			return nil, fmt.Errorf("failed to disect path '%s': %w", spaceRef, err)
		}

		if spaceRef == "" {
			return nil, gitness_store.ErrResourceNotFound
		}
	}
}
