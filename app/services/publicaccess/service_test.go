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

package publicaccess

import (
	"testing"

	"github.com/harness/gitness/app/services/refcache"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/require"
)

func TestService_IsPublicAccessSupported(t *testing.T) {
	for _, enabled := range []bool{true, false} {
		svc := NewService(enabled, nil, refcacheSpaceFinderZero, refcacheRepoFinderZero, nil)
		supported, err := svc.IsPublicAccessSupported(t.Context(), enum.PublicResourceTypeRepo, "space")
		require.NoError(t, err)
		require.Equal(t, enabled, supported)
	}
}

func TestService_SetNotAllowed(t *testing.T) {
	svc := NewService(false, nil, refcacheSpaceFinderZero, refcacheRepoFinderZero, nil)

	// enabling is rejected before any lookup happens
	err := svc.Set(t.Context(), enum.PublicResourceTypeRepo, "space/repo", true)
	require.ErrorIs(t, err, ErrPublicAccessNotAllowed)
}

func TestService_InvalidResourceType(t *testing.T) {
	svc := NewService(true, nil, refcacheSpaceFinderZero, refcacheRepoFinderZero, nil)

	_, err := svc.Get(t.Context(), "bogus", "path")
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid public resource type")

	require.Error(t, svc.Set(t.Context(), "bogus", "path", false))
	require.Error(t, svc.Delete(t.Context(), "bogus", "path"))
}

var (
	refcacheSpaceFinderZero = refcache.SpaceFinder{}
	refcacheRepoFinderZero  = refcache.RepoFinder{}
)
