// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package store

import (
	"github.com/harness/gitness/cache"
	"github.com/harness/gitness/types"
)

type (
	// PrincipalInfoCache caches principal IDs to principal info.
	PrincipalInfoCache cache.ExtendedCache[int64, *types.PrincipalInfo]

	// SpacePathCache caches a raw path to a space path.
	SpacePathCache cache.Cache[string, *types.SpacePath]

	// RepoGitInfoCache caches repository IDs to values GitUID.
	RepoGitInfoCache cache.Cache[int64, *types.RepositoryGitInfo]
)
