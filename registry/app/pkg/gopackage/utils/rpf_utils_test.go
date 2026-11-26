//  Copyright 2023 Harness, Inc.
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
	"archive/zip"
	"bytes"
	"io"
	"strings"
	"testing"

	gopackagemetadata "github.com/harness/gitness/registry/app/metadata/gopackage"

	"github.com/stretchr/testify/require"
)

func TestRpfGetModuleNameFromModFile(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		want    string
		wantErr bool
	}{
		{"simple", "module example.com/hello\n\ngo 1.21\n", "example.com/hello", false},
		{"leading whitespace", "// c\n   module   example.com/x  \n", "example.com/x", false},
		{"missing", "go 1.21\nrequire a v1\n", "", true},
		{"empty", "", "", true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := GetModuleNameFromModFile(strings.NewReader(tc.in))
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}

type rpfFailingReader struct{}

func (rpfFailingReader) Read([]byte) (int, error) { return 0, io.ErrClosedPipe }

func TestRpfGetModuleNameFromModFileReadError(t *testing.T) {
	_, err := GetModuleNameFromModFile(rpfFailingReader{})
	require.ErrorContains(t, err, "error scanning mod file")
}

func TestRpfGetArtifactInfoFromURL(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		image    string
		version  string
		filename string
		wantErr  bool
	}{
		{"latest", "example.com/hello/@latest", "example.com/hello", "@latest", "", false},
		{"zip", "example.com/hello/@v/v1.0.2.zip", "example.com/hello", "v1.0.2", "v1.0.2.zip", false},
		{"mod", "example.com/hello/@v/v1.0.2.mod", "example.com/hello", "v1.0.2", "v1.0.2.mod", false},
		{"info", "example.com/hello/@v/v1.0.2.info", "example.com/hello", "v1.0.2", "v1.0.2.info", false},
		{"list", "example.com/hello/@v/list", "example.com/hello", "", "list", false},
		{"no marker", "example.com/hello/v1.0.2.zip", "", "", "", true},
		{"empty version", "example.com/hello/@v/.zip", "", "", "", true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			image, version, filename, err := GetArtifactInfoFromURL(tc.path)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.image, image)
			require.Equal(t, tc.version, version)
			require.Equal(t, tc.filename, filename)
		})
	}
}

func TestRpfGetIndexFilePathAndMainArtifact(t *testing.T) {
	require.Equal(t, "/example.com/hello/index", GetIndexFilePath("example.com/hello"))
	require.True(t, IsMainArtifactFile("v1.0.0.zip"))
	require.False(t, IsMainArtifactFile("v1.0.0.mod"))
	require.False(t, IsMainArtifactFile("list"))
}

func TestRpfGetPackageMetadataFromInfoFile(t *testing.T) {
	body := `{"Version":"v1.2.3","Time":"2024-01-01T00:00:00Z","Origin":{"VCS":"git","URL":"https://x/y"}}`
	md, err := GetPackageMetadataFromInfoFile(bytes.NewBufferString(body))
	require.NoError(t, err)
	require.Equal(t, "v1.2.3", md.Version)
	require.Equal(t, "git", md.Origin.VCS)

	_, err = GetPackageMetadataFromInfoFile(bytes.NewBufferString("{not json"))
	require.ErrorContains(t, err, "error decoding info file")
}

func TestRpfUpdateMetadataFromModFile(t *testing.T) {
	mod := "module example.com/hello\n\ngo 1.21\n\nrequire (\n\tgithub.com/a/b v1.0.0\n" +
		"\tgithub.com/c/d v1.1.0 // indirect\n)\n"
	md := &gopackagemetadata.VersionMetadata{}
	require.NoError(t, UpdateMetadataFromModFile(bytes.NewBufferString(mod), md))
	require.Equal(t, []gopackagemetadata.Dependency{
		{Name: "github.com/a/b", Version: "v1.0.0"},
		{Name: "github.com/c/d", Version: "v1.1.0"},
	}, md.Dependencies)

	err := UpdateMetadataFromModFile(bytes.NewBufferString("this is not a mod file"), md)
	require.ErrorContains(t, err, "error parsing mod file")
}

func rpfBuildZip(t *testing.T, method uint16, files map[string]string, order []string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, name := range order {
		w, err := zw.CreateHeader(&zip.FileHeader{Name: name, Method: method})
		require.NoError(t, err)
		_, err = w.Write([]byte(files[name]))
		require.NoError(t, err)
	}
	require.NoError(t, zw.Close())
	return buf.Bytes()
}

func TestRpfUpdateMetadataFromZipFile(t *testing.T) {
	files := map[string]string{
		"example.com/hello@v1.0.0/go.mod":    "module example.com/hello\n",
		"example.com/hello@v1.0.0/main.go":   "package main\n",
		"example.com/hello@v1.0.0/README.md": "# Hello readme\n",
	}
	order := []string{
		"example.com/hello@v1.0.0/go.mod",
		"example.com/hello@v1.0.0/README.md",
		"example.com/hello@v1.0.0/main.go",
	}

	for name, method := range map[string]uint16{"stored": zip.Store, "deflated": zip.Deflate} {
		t.Run(name, func(t *testing.T) {
			data := rpfBuildZip(t, method, files, order)
			md := &gopackagemetadata.VersionMetadata{}
			err := UpdateMetadataFromZipFile(io.NopCloser(bytes.NewReader(data)), md)
			require.NoError(t, err)
			require.Equal(t, "# Hello readme\n", md.Readme)
		})
	}

	t.Run("no readme", func(t *testing.T) {
		data := rpfBuildZip(t, zip.Store, map[string]string{"a/main.go": "package a\n"}, []string{"a/main.go"})
		md := &gopackagemetadata.VersionMetadata{}
		require.NoError(t, UpdateMetadataFromZipFile(io.NopCloser(bytes.NewReader(data)), md))
		require.Empty(t, md.Readme)
	})

	t.Run("empty input", func(t *testing.T) {
		md := &gopackagemetadata.VersionMetadata{}
		require.NoError(t, UpdateMetadataFromZipFile(io.NopCloser(bytes.NewReader(nil)), md))
		require.Empty(t, md.Readme)
	})
}
