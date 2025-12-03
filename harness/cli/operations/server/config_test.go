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

func TestBackfillURLsHTTPEmptyPort(t *testing.T) {
	config := &types.Config{}

	err := backfillURLs(config)
	require.NoError(t, err)

	require.Equal(t, "http://localhost", config.URL.Internal)
	require.Equal(t, "http://host.docker.internal", config.URL.Container)

	require.Equal(t, "http://localhost/api", config.URL.API)
	require.Equal(t, "http://localhost/git", config.URL.Git)
	require.Equal(t, "http://localhost", config.URL.UI)
}

func TestBackfillURLsSSHEmptyPort(t *testing.T) {
	config := &types.Config{}

	err := backfillURLs(config)
	require.NoError(t, err)

	require.Equal(t, "ssh://localhost", config.URL.GitSSH)
}

func TestBackfillURLsHTTPHostPort(t *testing.T) {
	config := &types.Config{}
	config.HTTP.Host = "myhost"
	config.HTTP.Port = 1234

	err := backfillURLs(config)
	require.NoError(t, err)

	require.Equal(t, "http://localhost:1234", config.URL.Internal)
	require.Equal(t, "http://host.docker.internal:1234", config.URL.Container)

	require.Equal(t, "http://myhost:1234/api", config.URL.API)
	require.Equal(t, "http://myhost:1234/git", config.URL.Git)
	require.Equal(t, "http://myhost:1234", config.URL.UI)
}

func TestBackfillURLsSSHHostPort(t *testing.T) {
	config := &types.Config{}
	config.SSH.Host = "myhost"
	config.SSH.Port = 1234

	err := backfillURLs(config)
	require.NoError(t, err)

	require.Equal(t, "ssh://myhost:1234", config.URL.GitSSH)
}

func TestBackfillURLsHTTPPortStripsDefaultHTTP(t *testing.T) {
	config := &types.Config{}
	config.HTTP.Port = 80

	err := backfillURLs(config)
	require.NoError(t, err)

	require.Equal(t, "http://localhost", config.URL.Internal)
	require.Equal(t, "http://host.docker.internal", config.URL.Container)

	require.Equal(t, "http://localhost/api", config.URL.API)
	require.Equal(t, "http://localhost/git", config.URL.Git)
	require.Equal(t, "http://localhost", config.URL.UI)
}

// TODO: Update once we add proper https support - as of now nothing is stripped!
func TestBackfillURLsHTTPPortStripsDefaultHTTPS(t *testing.T) {
	config := &types.Config{}
	config.HTTP.Port = 443

	err := backfillURLs(config)
	require.NoError(t, err)

	require.Equal(t, "http://localhost:443", config.URL.Internal)
	require.Equal(t, "http://host.docker.internal:443", config.URL.Container)

	require.Equal(t, "http://localhost:443/api", config.URL.API)
	require.Equal(t, "http://localhost:443/git", config.URL.Git)
	require.Equal(t, "http://localhost:443", config.URL.UI)
}

func TestBackfillURLsSSHPortStripsDefault(t *testing.T) {
	config := &types.Config{}
	config.SSH.Port = 22

	err := backfillURLs(config)
	require.NoError(t, err)

	require.Equal(t, "ssh://localhost", config.URL.GitSSH)
}

func TestBackfillURLsBaseInvalidProtocol(t *testing.T) {
	config := &types.Config{}
	config.URL.Base = "abc://xyz:4321/test"

	err := backfillURLs(config)
	require.ErrorContains(t, err, "base url scheme 'abc' is not supported")
}

func TestBackfillURLsBaseNoHost(t *testing.T) {
	config := &types.Config{}
	config.URL.Base = "http:///test"

	err := backfillURLs(config)
	require.ErrorContains(t, err, "a non-empty base url host has to be provided")
}

func TestBackfillURLsBaseNoHostWithPort(t *testing.T) {
	config := &types.Config{}
	config.URL.Base = "http://:4321/test"

	err := backfillURLs(config)
	require.ErrorContains(t, err, "a non-empty base url host has to be provided")
}

func TestBackfillURLsBaseInvalidPort(t *testing.T) {
	config := &types.Config{}
	config.URL.Base = "http://localhost:abc/test"

	err := backfillURLs(config)
	require.ErrorContains(t, err, "invalid port \":abc\" after host")
}

func TestBackfillURLsBase(t *testing.T) {
	config := &types.Config{}
	config.HTTP.Host = "xyz"
	config.HTTP.Port = 1234
	config.SSH.Host = "kmno"
	config.SSH.Port = 421
	config.URL.Base = "https://xyz:4321/test"

	err := backfillURLs(config)
	require.NoError(t, err)

	require.Equal(t, "http://localhost:1234", config.URL.Internal)
	require.Equal(t, "http://host.docker.internal:1234", config.URL.Container)

	require.Equal(t, "https://xyz:4321/test/api", config.URL.API)
	require.Equal(t, "https://xyz:4321/test/git", config.URL.Git)
	require.Equal(t, "https://xyz:4321/test", config.URL.UI)

	require.Equal(t, "ssh://xyz:421", config.URL.GitSSH)
}

func TestBackfillURLsBaseDefaultPortHTTP(t *testing.T) {
	config := &types.Config{}
	config.HTTP.Port = 1234
	config.URL.Base = "http://xyz/test"

	err := backfillURLs(config)
	require.NoError(t, err)

	require.Equal(t, "http://localhost:1234", config.URL.Internal)
	require.Equal(t, "http://host.docker.internal:1234", config.URL.Container)

	require.Equal(t, "http://xyz/test/api", config.URL.API)
	require.Equal(t, "http://xyz/test/git", config.URL.Git)
	require.Equal(t, "http://xyz/test", config.URL.UI)
}

func TestBackfillURLsBaseDefaultPortHTTPExplicit(t *testing.T) {
	config := &types.Config{}
	config.HTTP.Port = 1234
	config.URL.Base = "http://xyz:80/test"

	err := backfillURLs(config)
	require.NoError(t, err)

	require.Equal(t, "http://localhost:1234", config.URL.Internal)
	require.Equal(t, "http://host.docker.internal:1234", config.URL.Container)

	require.Equal(t, "http://xyz:80/test/api", config.URL.API)
	require.Equal(t, "http://xyz:80/test/git", config.URL.Git)
	require.Equal(t, "http://xyz:80/test", config.URL.UI)
}

func TestBackfillURLsBaseDefaultPortHTTPS(t *testing.T) {
	config := &types.Config{}
	config.HTTP.Port = 1234
	config.URL.Base = "https://xyz/test"

	err := backfillURLs(config)
	require.NoError(t, err)

	require.Equal(t, "http://localhost:1234", config.URL.Internal)
	require.Equal(t, "http://host.docker.internal:1234", config.URL.Container)

	require.Equal(t, "https://xyz/test/api", config.URL.API)
	require.Equal(t, "https://xyz/test/git", config.URL.Git)
	require.Equal(t, "https://xyz/test", config.URL.UI)
}

func TestBackfillURLsBaseDefaultPortHTTPSExplicit(t *testing.T) {
	config := &types.Config{}
	config.HTTP.Port = 1234
	config.URL.Base = "https://xyz:443/test"

	err := backfillURLs(config)
	require.NoError(t, err)

	require.Equal(t, "http://localhost:1234", config.URL.Internal)
	require.Equal(t, "http://host.docker.internal:1234", config.URL.Container)

	require.Equal(t, "https://xyz:443/test/api", config.URL.API)
	require.Equal(t, "https://xyz:443/test/git", config.URL.Git)
	require.Equal(t, "https://xyz:443/test", config.URL.UI)
}

func TestBackfillURLsBaseRootPathStripped(t *testing.T) {
	config := &types.Config{}
	config.HTTP.Port = 1234
	config.URL.Base = "https://xyz:4321/"

	err := backfillURLs(config)
	require.NoError(t, err)

	require.Equal(t, "http://localhost:1234", config.URL.Internal)
	require.Equal(t, "http://host.docker.internal:1234", config.URL.Container)

	require.Equal(t, "https://xyz:4321/api", config.URL.API)
	require.Equal(t, "https://xyz:4321/git", config.URL.Git)
	require.Equal(t, "https://xyz:4321", config.URL.UI)
}

func TestBackfillURLsSSHBasePathIgnored(t *testing.T) {
	config := &types.Config{}
	config.SSH.Port = 1234
	config.URL.Base = "https://xyz:4321/abc"

	err := backfillURLs(config)
	require.NoError(t, err)

	require.Equal(t, "ssh://xyz:1234", config.URL.GitSSH)
}

func TestBackfillURLsCustom(t *testing.T) {
	config := &types.Config{}
	config.HTTP.Host = "abc"
	config.HTTP.Port = 1234
	config.SSH.Host = "abc"
	config.SSH.Port = 1234
	config.URL.Internal = "http://APIInternal/APIInternal/p"
	config.URL.Container = "https://GitContainer/GitContainer/p"
	config.URL.Base = "https://xyz:4321/test"
	config.URL.API = "http://API:1111/API/p"
	config.URL.Git = "https://GIT:443/GIT/p"
	config.URL.UI = "http://UI:80/UI/p"
	config.URL.GitSSH = "ssh://GITSSH:21/GITSSH/p"

	err := backfillURLs(config)
	require.NoError(t, err)

	require.Equal(t, "http://APIInternal/APIInternal/p", config.URL.Internal)
	require.Equal(t, "https://GitContainer/GitContainer/p", config.URL.Container)

	require.Equal(t, "http://API:1111/API/p", config.URL.API)
	require.Equal(t, "https://GIT:443/GIT/p", config.URL.Git)
	require.Equal(t, "http://UI:80/UI/p", config.URL.UI)

	require.Equal(t, "ssh://GITSSH:21/GITSSH/p", config.URL.GitSSH)
}
