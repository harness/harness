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

package maintenance

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func mscTouchFile(t *testing.T, path string, age time.Duration) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o700))
	require.NoError(t, os.WriteFile(path, []byte("stale"), 0o600))
	mtime := time.Now().Add(-age)
	require.NoError(t, os.Chtimes(path, mtime, mtime))
}

func TestMscFindTempObjectsRecentIgnored(t *testing.T) {
	repo := t.TempDir()
	objects := filepath.Join(repo, "objects")
	stale := filepath.Join(objects, "aa", "tmp_old")
	recent := filepath.Join(objects, "bb", "tmp_new")

	mscTouchFile(t, stale, 48*time.Hour)
	mscTouchFile(t, recent, time.Hour)

	got, err := FindTempObjects(repo)
	require.NoError(t, err)
	require.Equal(t, []string{stale}, got)
}
