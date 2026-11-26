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

package commons

import (
	"context"
	"testing"

	"github.com/harness/gitness/app/services/refcache"
	api "github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/registry/types"

	"github.com/stretchr/testify/require"
)

func TestRrmGetCredentialsAnonymous(t *testing.T) {
	var finder refcache.SpaceFinder
	access, secret, anon, err := GetCredentials(context.Background(), finder, nil, types.UpstreamProxy{
		RepoAuthType: string(api.AuthTypeAnonymous),
	})
	require.NoError(t, err)
	require.True(t, anon)
	require.Empty(t, access)
	require.Empty(t, secret)
}

func TestRrmGetCredentialsUserPassword(t *testing.T) {
	var finder refcache.SpaceFinder
	access, secret, anon, err := GetCredentials(context.Background(), finder, nil, types.UpstreamProxy{
		RepoAuthType: string(api.AuthTypeUserPassword),
		UserName:     "user",
	})
	require.NoError(t, err)
	require.False(t, anon)
	require.Equal(t, "user", access)
	require.Empty(t, secret)
}

func TestRrmGetCredentialsUnsupported(t *testing.T) {
	var finder refcache.SpaceFinder
	_, _, _, err := GetCredentials(context.Background(), finder, nil, types.UpstreamProxy{
		RepoAuthType: "UnsupportedAuth",
	})
	require.Error(t, err)
}
