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

package api

import (
	"archive/tar"
	"bytes"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// colonFilePath is the path of the file inside the directory named ":" in the test repository.
const (
	colonFilePath    = ":/f.txt"
	colonNewFilePath = ":/new.txt"
)

// TestArchive_ColonDirectory asserts that the path filter of an archive takes a path inside a
// directory named ":" literally.
//
// Regression guard for CODE-6237.
func TestArchive_ColonDirectory(t *testing.T) {
	requireGit(t)

	repoPath := newTestRepoWithColonDir(t)

	var buf bytes.Buffer
	err := (&Git{}).Archive(t.Context(), repoPath,
		ArchiveParams{Treeish: "main", Format: ArchiveFormatTar, Paths: []string{colonFilePath}}, &buf)
	if err != nil {
		t.Fatalf("expected to archive %q, got: %v", colonFilePath, err)
	}

	var names []string
	tr := tar.NewReader(&buf)
	for {
		hdr, nextErr := tr.Next()
		if nextErr == io.EOF {
			break
		}
		if nextErr != nil {
			t.Fatalf("failed to read the archive: %v", nextErr)
		}
		names = append(names, hdr.Name)
	}

	if !slices.Contains(names, colonFilePath) {
		t.Errorf("expected %q in the archive, got: %v", colonFilePath, names)
	}
}

// TestAddFiles_ColonDirectory asserts that a file inside a directory named ":" is staged by
// its literal path.
//
// Regression guard for CODE-6237.
func TestAddFiles_ColonDirectory(t *testing.T) {
	requireGit(t)

	repoPath := newTestRepoWithColonDir(t)

	if err := os.WriteFile(filepath.Join(repoPath, ":", "new.txt"), []byte("new\n"), 0o600); err != nil {
		t.Fatalf("failed to write the file: %v", err)
	}

	if err := (&Git{}).AddFiles(t.Context(), repoPath, false, colonNewFilePath); err != nil {
		t.Fatalf("expected to stage %q, got: %v", colonNewFilePath, err)
	}

	cmd := exec.CommandContext(t.Context(), "git", "diff", "--cached", "--name-only")
	cmd.Dir = repoPath
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git diff --cached: %v", err)
	}
	if !strings.Contains(string(out), colonNewFilePath) {
		t.Errorf("expected %q to be staged, got: %q", colonNewFilePath, out)
	}
}
