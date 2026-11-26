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

package filesystem

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRsg_InnerDriverBatchCopyObjects(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	d := &driver{rootDirectory: root}

	srcDir := filepath.Join(root, "batch")
	require.NoError(t, os.MkdirAll(srcDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "f1"), []byte("1"), 0o600))
	require.NoError(t, d.PutContent(ctx, "/batch/f1", []byte("1")))
	require.NoError(t, d.PutContent(ctx, "/batch/f2", []byte("2")))

	require.NoError(t, d.BatchCopyObjects(ctx, "", []string{"/batch/f1", "/batch/f2"}, 2))
}

func TestRsg_InnerDriverCopyFileContents(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	src := filepath.Join(root, "src")
	dst := filepath.Join(root, "dst")
	require.NoError(t, os.WriteFile(src, []byte("copy"), 0o600))
	require.NoError(t, copyFileContents(ctx, src, dst, 0o644))
	got, err := os.ReadFile(dst)
	require.NoError(t, err)
	require.Equal(t, []byte("copy"), got)
}

func TestRsg_FilesystemFactoryCreate(t *testing.T) {
	ctx := context.Background()
	f := &filesystemDriverFactory{}
	d, err := f.Create(ctx, map[string]any{"rootdirectory": t.TempDir()})
	require.NoError(t, err)
	require.NotNil(t, d)
}
