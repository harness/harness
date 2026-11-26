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

package session

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestInfSession_BuildersAreImmutable(t *testing.T) {
	base := New("/tmp/some/session.json")
	require.Equal(t, "/tmp/some/session.json", base.Path())

	updated := base.SetURI("http://localhost:3000").SetExpiresAt(42).SetAccessToken("tok-0000")
	require.Equal(t, "http://localhost:3000", updated.URI)
	require.EqualValues(t, 42, updated.ExpiresAt)
	require.Equal(t, "tok-0000", updated.AccessToken)
	require.Equal(t, updated.Path(), base.Path())

	require.Empty(t, base.URI, "value receivers must not modify the original")
	require.Zero(t, base.ExpiresAt)
	require.Empty(t, base.AccessToken)
}

func TestInfSession_StoreAndLoad(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	expires := time.Now().Add(time.Hour).Unix()

	require.NoError(t, New(path).SetURI("http://host").SetExpiresAt(expires).SetAccessToken("tok-0000").Store())

	info, err := os.Stat(path)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o600), info.Mode().Perm())

	loaded, err := LoadFromPath(path)
	require.NoError(t, err)
	require.Equal(t, "http://host", loaded.URI)
	require.Equal(t, expires, loaded.ExpiresAt)
	require.Equal(t, "tok-0000", loaded.AccessToken)
	require.Equal(t, path, loaded.Path())

	// the path itself is not part of the stored document.
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	var raw map[string]any
	require.NoError(t, json.Unmarshal(data, &raw))
	require.ElementsMatch(t, []string{"uri", "expires_at", "access_token"}, infKeys(raw))
}

func infKeys(m map[string]any) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

func TestInfSession_LoadErrors(t *testing.T) {
	dir := t.TempDir()

	t.Run("missing file", func(t *testing.T) {
		_, err := LoadFromPath(filepath.Join(dir, "missing.json"))
		require.ErrorContains(t, err, "failed to read session from file")
		require.ErrorIs(t, err, os.ErrNotExist)
	})
	t.Run("invalid json", func(t *testing.T) {
		path := filepath.Join(dir, "bad.json")
		require.NoError(t, os.WriteFile(path, []byte("{not json"), 0o600))
		_, err := LoadFromPath(path)
		require.ErrorContains(t, err, "failed to deserialize session")
	})
	t.Run("expired token", func(t *testing.T) {
		path := filepath.Join(dir, "expired.json")
		require.NoError(t, New(path).SetExpiresAt(time.Now().Add(-time.Minute).Unix()).Store())
		loaded, err := LoadFromPath(path)
		require.True(t, errors.Is(err, ErrTokenExpired))
		require.Equal(t, path, loaded.Path(), "the session is still returned so it can be replaced")
	})
}

func TestInfSession_StoreErrors(t *testing.T) {
	err := New(filepath.Join(t.TempDir(), "no-such-dir", "config.json")).Store()
	require.ErrorContains(t, err, "failed to write session to file")
}
