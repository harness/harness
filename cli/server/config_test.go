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

package server

import (
	"testing"

	"github.com/harness/gitness/types"
	"github.com/stretchr/testify/require"
)

func TestBackfilURLsPortBind(t *testing.T) {
	config := &types.Config{}
	config.Server.HTTP.Bind = ":1234"

	err := backfillURLs(config)
	require.NoError(t, err)

	require.Equal(t, "http://localhost:1234/api", config.URL.API)
	require.Equal(t, "http://localhost:1234/git", config.URL.Git)
	require.Equal(t, "http://localhost:1234", config.URL.UI)
	require.Equal(t, "http://localhost:1234/api", config.URL.APIInternal)
	require.Equal(t, "http://host.docker.internal:1234/git", config.URL.GitContainer)
}

func TestBackfilURLsHostBind(t *testing.T) {
	config := &types.Config{}
	config.Server.HTTP.Bind = "abc:1234"

	err := backfillURLs(config)
	require.NoError(t, err)

	require.Equal(t, "http://abc:1234/api", config.URL.API)
	require.Equal(t, "http://abc:1234/git", config.URL.Git)
	require.Equal(t, "http://abc:1234", config.URL.UI)
	require.Equal(t, "http://localhost:1234/api", config.URL.APIInternal)
	require.Equal(t, "http://host.docker.internal:1234/git", config.URL.GitContainer)
}

func TestBackfilURLsBase(t *testing.T) {
	config := &types.Config{}
	config.Server.HTTP.Bind = "abc:1234"
	config.URL.Base = "https://xyz:4321/test"

	err := backfillURLs(config)
	require.NoError(t, err)

	require.Equal(t, "https://xyz:4321/test/api", config.URL.API)
	require.Equal(t, "https://xyz:4321/test/git", config.URL.Git)
	require.Equal(t, "https://xyz:4321/test", config.URL.UI)
	require.Equal(t, "http://localhost:1234/api", config.URL.APIInternal)
	require.Equal(t, "http://host.docker.internal:1234/git", config.URL.GitContainer)
}

func TestBackfilURLsCustom(t *testing.T) {
	config := &types.Config{}
	config.Server.HTTP.Bind = "abc:1234"
	config.URL.Base = "https://xyz:4321/test"
	config.URL.API = "http://API:1111/API/p"
	config.URL.APIInternal = "http://APIInternal:1111/APIInternal/p"
	config.URL.Git = "http://Git:1111/Git/p"
	config.URL.GitContainer = "http://GitContainer:1111/GitContainer/p"
	config.URL.UI = "http://UI:1111/UI/p"

	err := backfillURLs(config)
	require.NoError(t, err)

	require.Equal(t, "http://API:1111/API/p", config.URL.API)
	require.Equal(t, "http://Git:1111/Git/p", config.URL.Git)
	require.Equal(t, "http://UI:1111/UI/p", config.URL.UI)
	require.Equal(t, "http://APIInternal:1111/APIInternal/p", config.URL.APIInternal)
	require.Equal(t, "http://GitContainer:1111/GitContainer/p", config.URL.GitContainer)
}
