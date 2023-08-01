// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package authz

import (
	"time"

	"github.com/harness/gitness/internal/store"

	"github.com/google/wire"
)

// WireSet provides a wire set for this package.
var WireSet = wire.NewSet(
	ProvideAuthorizer,
	ProvidePermissionCache,
)

func ProvideAuthorizer(pCache PermissionCache) Authorizer {
	return NewMembershipAuthorizer(pCache)
}

func ProvidePermissionCache(
	spaceStore store.SpaceStore,
	membershipStore store.MembershipStore,
) PermissionCache {
	const permissionCacheTimeout = time.Second * 15
	return NewPermissionCache(spaceStore, membershipStore, permissionCacheTimeout)
}
