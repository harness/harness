// Copyright 2026 Harness, Inc.
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
	"testing"

	"github.com/harness/gitness/app/services/publicaccess"
	"github.com/harness/gitness/app/services/refcache"

	"github.com/stretchr/testify/require"
)

func TestCt5ProvideAuthorizerAndCache(t *testing.T) {
	cache := ProvidePermissionCache(refcache.SpaceFinder{}, nil)
	require.NotNil(t, cache)

	var public publicaccess.Service
	require.NotNil(t, ProvideAuthorizer(cache, refcache.SpaceFinder{}, public))
}
