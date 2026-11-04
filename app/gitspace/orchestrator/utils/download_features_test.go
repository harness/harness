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

package utils

import (
	"archive/tar"
	"os"
	"path/filepath"
	"testing"
)

type tarEntry struct {
	name     string
	typeflag byte
	body     string
	linkname string
}

func writeTarball(t *testing.T, path string, entries []tarEntry) {
	t.Helper()

	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	tw := tar.NewWriter(f)
	for _, e := range entries {
		hdr := &tar.Header{
			Name:     e.name,
			Typeflag: e.typeflag,
			Linkname: e.linkname,
			Mode:     0o644,
			Size:     int64(len(e.body)),
		}
		if e.typeflag == tar.TypeDir {
			hdr.Mode = 0o755
		}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write([]byte(e.body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestUnpackTarball(t *testing.T) {
	tests := []struct {
		name      string
		entries   []tarEntry
		wantErr   bool
		wantFiles []string
	}{
		{
			name: "local entries",
			entries: []tarEntry{
				{name: "./", typeflag: tar.TypeDir},
				{name: "./devcontainer-feature.json", typeflag: tar.TypeReg, body: "{}"},
				{name: "nested/", typeflag: tar.TypeDir},
				{name: "nested/install.sh", typeflag: tar.TypeReg, body: "echo"},
			},
			wantFiles: []string{"devcontainer-feature.json", "nested/install.sh"},
		},
		{
			name:    "parent directory",
			entries: []tarEntry{{name: "../outside.txt", typeflag: tar.TypeReg, body: "x"}},
			wantErr: true,
		},
		{
			name:    "nested parent directory",
			entries: []tarEntry{{name: "nested/../../outside.txt", typeflag: tar.TypeReg, body: "x"}},
			wantErr: true,
		},
		{
			name:    "directory outside output",
			entries: []tarEntry{{name: "../outside/", typeflag: tar.TypeDir}},
			wantErr: true,
		},
		{
			name: "symlink outside output",
			entries: []tarEntry{
				{name: "outside", typeflag: tar.TypeSymlink, linkname: "../outside"},
				{name: "outside/outside.txt", typeflag: tar.TypeReg, body: "x"},
			},
			wantErr: true,
		},
		{
			name:    "absolute path",
			entries: []tarEntry{{name: "/outside.txt", typeflag: tar.TypeReg, body: "x"}},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			base := t.TempDir()
			outDir := filepath.Join(base, "feature")
			tarball := filepath.Join(base, "feature.tar")
			writeTarball(t, tarball, tt.entries)

			err := unpackTarball(tarball, outDir)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected an error, got nil")
				}
				if _, err := os.Stat(filepath.Join(base, "outside.txt")); !os.IsNotExist(err) {
					t.Error("file was written outside the output directory")
				}
				if _, err := os.Stat(filepath.Join(base, "outside")); !os.IsNotExist(err) {
					t.Error("directory was created outside the output directory")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			for _, name := range tt.wantFiles {
				if _, err := os.Stat(filepath.Join(outDir, name)); err != nil {
					t.Errorf("expected %s to be extracted: %v", name, err)
				}
			}
		})
	}
}
