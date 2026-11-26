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

package provide

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/harness/gitness/cli/session"

	"github.com/adrg/xdg"
	"github.com/stretchr/testify/require"
)

// infXDG points the xdg config directory to a temporary directory and returns the session file path.
func infXDG(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Cleanup(xdg.Reload) // registered first so it runs after the environment is restored.
	t.Setenv("XDG_CONFIG_HOME", dir)
	xdg.Reload()
	return filepath.Join(dir, "app", "config.json")
}

func TestInfSessionPath(t *testing.T) {
	want := infXDG(t)
	got, err := sessionPath()
	require.NoError(t, err)
	require.Equal(t, want, got)
	require.DirExists(t, filepath.Dir(got), "the config directory is created on demand")
}

func TestInfNewSession(t *testing.T) {
	path := infXDG(t)
	ss := NewSession()
	require.Equal(t, path, ss.Path())
	require.Equal(t, DefaultServerURI, ss.URI)
	require.Empty(t, ss.AccessToken)
}

func TestInfLoadSession(t *testing.T) {
	path := infXDG(t)

	t.Run("no stored session", func(t *testing.T) {
		_, err := loadSession()
		require.ErrorIs(t, err, os.ErrNotExist)
	})
	t.Run("stored session", func(t *testing.T) {
		stored := session.New(path).SetURI("http://example.invalid").
			SetExpiresAt(time.Now().Add(time.Hour).Unix()).SetAccessToken("tok-0000")
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o700))
		require.NoError(t, stored.Store())

		ss, err := loadSession()
		require.NoError(t, err)
		require.Equal(t, "http://example.invalid", ss.URI)
		require.Equal(t, "tok-0000", ss.AccessToken)
		require.Equal(t, "tok-0000", Session().AccessToken)
	})
	t.Run("missing uri falls back to the default server", func(t *testing.T) {
		stored := session.New(path).SetExpiresAt(time.Now().Add(time.Hour).Unix())
		require.NoError(t, stored.Store())
		ss, err := loadSession()
		require.NoError(t, err)
		require.Equal(t, DefaultServerURI, ss.URI)
	})
	t.Run("expired session", func(t *testing.T) {
		require.NoError(t, session.New(path).SetURI("http://example.invalid").
			SetExpiresAt(time.Now().Add(-time.Hour).Unix()).Store())
		_, err := loadSession()
		require.ErrorIs(t, err, session.ErrTokenExpired)
	})
}

func TestInfClients(t *testing.T) {
	path := infXDG(t)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o700))
	require.NoError(t, session.New(path).SetURI("http://example.invalid").
		SetExpiresAt(time.Now().Add(time.Hour).Unix()).SetAccessToken("tok-0000").Store())

	require.NotNil(t, Client())
	require.NotNil(t, OpenClient("http://other.invalid"))

	t.Setenv("DEBUG", "true")
	require.NotNil(t, newClient(session.Session{URI: "http://example.invalid"}))
}
