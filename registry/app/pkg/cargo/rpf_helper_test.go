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

package cargo

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"io"
	"testing"

	storagedriver "github.com/harness/gitness/registry/app/driver"
	cargometadata "github.com/harness/gitness/registry/app/metadata/cargo"
	"github.com/harness/gitness/registry/app/pkg"
	cargotype "github.com/harness/gitness/registry/app/pkg/types/cargo"
	"github.com/harness/gitness/registry/app/storage"

	"github.com/stretchr/testify/require"
)

type rpfFakeDriver struct {
	storagedriver.StorageDriver
	content []byte
}

func (d *rpfFakeDriver) Reader(_ context.Context, _ string, _ int64) (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewReader(d.content)), nil
}

func rpfNewFileReader(ctx context.Context, t *testing.T, content []byte) *storage.FileReader {
	t.Helper()
	fr, err := storage.NewFileReader(ctx, &rpfFakeDriver{content: content}, "p", int64(len(content)))
	require.NoError(t, err)
	return fr
}

func rpfBuildCrate(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for name, body := range files {
		require.NoError(t, tw.WriteHeader(&tar.Header{
			Name: name, Mode: 0o600, Size: int64(len(body)), Typeflag: tar.TypeReg,
		}))
		_, err := tw.Write([]byte(body))
		require.NoError(t, err)
	}
	require.NoError(t, tw.WriteHeader(&tar.Header{Name: "demo-1.0.0/src/", Mode: 0o700, Typeflag: tar.TypeDir}))
	require.NoError(t, tw.Close())
	require.NoError(t, gz.Close())
	return buf.Bytes()
}

const rpfCargoToml = `
[package]
name = "demo"
version = "1.0.0"
description = "demo crate"
license = "MIT"
keywords = ["a", "b"]
repository = "https://example.test/repo"
documentation = "https://example.test/docs"
readme = "README.md"

[dependencies]
serde = "1.0"
tokio = { version = "1.2", features = ["rt", "macros"], optional = true, default-features = false, registry = "r" }

[dev-dependencies]
mockall = "0.11"

[build-dependencies]
cc = { version = "1.0" }
`

func rpfInfo() *cargotype.ArtifactInfo {
	return &cargotype.ArtifactInfo{ArtifactInfo: pkg.ArtifactInfo{Image: "demo"}, Version: "1.0.0"}
}

func TestRpfCargoPathBuilders(t *testing.T) {
	require.Equal(t, "demo-1.0.0.crate", getCrateFileName("demo", "1.0.0"))
	require.Equal(t, "/crates/demo/1.0.0/demo-1.0.0.crate", getCrateFilePath("demo", "1.0.0"))
	require.Equal(t, "/index/de/mo/demo", getPackageIndexFilePath("de/mo/demo"))
	require.Equal(t, "demo/1.0.0/download", downloadPackageFilePath("demo", "1.0.0"))
}

func TestRpfGetReadmePath(t *testing.T) {
	require.Equal(t, "docs/R.md", getReadmePath("docs/R.md"))
	require.Equal(t, "README.md", getReadmePath(true))
	require.Empty(t, getReadmePath(false))
	require.Empty(t, getReadmePath(nil))
	require.Empty(t, getReadmePath(42))
}

func TestRpfParseCargoToml(t *testing.T) {
	m, err := parseCargoToml([]byte(rpfCargoToml))
	require.NoError(t, err)
	require.Equal(t, "demo", m.Package.Name)
	require.Equal(t, "README.md", m.Package.Readme)
	require.Len(t, m.Dependencies, 2)

	_, err = parseCargoToml([]byte("[package\nname="))
	require.Error(t, err)
}

func TestRpfParseDependency(t *testing.T) {
	simple := parseDependency("serde", "1.0", cargometadata.DependencyKindTypeNormal, "")
	require.Equal(t, "1.0", simple.VersionRequired)
	require.True(t, simple.DefaultFeatures)
	require.Equal(t, "serde", simple.ExplicitNameInToml)

	full := parseDependency("tokio", map[string]any{
		"version":          "1.2",
		"features":         []any{"rt", "macros"},
		"optional":         true,
		"default-features": false,
		"registry":         "r",
	}, cargometadata.DependencyKindTypeDev, "cfg(unix)")
	require.Equal(t, "1.2", full.VersionRequired)
	require.Equal(t, []string{"rt", "macros"}, full.Features)
	require.True(t, full.IsOptional)
	require.False(t, full.DefaultFeatures)
	require.Equal(t, "r", full.Registry)
	require.Equal(t, cargometadata.DependencyKindTypeDev, full.Kind)
	require.Equal(t, "cfg(unix)", full.Target)

	other := parseDependency("x", 5, cargometadata.DependencyKindTypeBuild, "")
	require.Empty(t, other.VersionRequired)
}

func TestRpfParseDependenciesKinds(t *testing.T) {
	m, err := parseCargoToml([]byte(rpfCargoToml))
	require.NoError(t, err)
	deps := *parseDependencies(m)
	require.Len(t, deps, 4)
	kinds := map[string]cargometadata.DependencyKindType{}
	for _, d := range deps {
		kinds[d.Name] = d.Kind
	}
	require.Equal(t, cargometadata.DependencyKindTypeNormal, kinds["serde"])
	require.Equal(t, cargometadata.DependencyKindTypeNormal, kinds["tokio"])
	require.Equal(t, cargometadata.DependencyKindTypeDev, kinds["mockall"])
	require.Equal(t, cargometadata.DependencyKindTypeBuild, kinds["cc"])
}

func TestRpfExtractCrate(t *testing.T) {
	crate := rpfBuildCrate(t, map[string]string{"demo-1.0.0/Cargo.toml": "x", "demo-1.0.0/a.txt": "hello"})
	files, err := extractCrate(io.NopCloser(bytes.NewReader(crate)))
	require.NoError(t, err)
	require.Len(t, files, 2)
	require.Equal(t, []byte("hello"), files["demo-1.0.0/a.txt"])

	_, err = extractCrate(io.NopCloser(bytes.NewReader([]byte("not gzip"))))
	require.Error(t, err)

	// A valid gzip stream that is not a tar archive.
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	_, err = gz.Write(bytes.Repeat([]byte("z"), 600))
	require.NoError(t, err)
	require.NoError(t, gz.Close())
	_, err = extractCrate(io.NopCloser(&buf))
	require.Error(t, err)
}

func TestRpfGenerateMetadataFromFile(t *testing.T) {
	ctx := context.Background()

	t.Run("success with readme", func(t *testing.T) {
		crate := rpfBuildCrate(t, map[string]string{
			"demo-1.0.0/Cargo.toml": rpfCargoToml,
			"demo-1.0.0/README.md":  "# readme",
		})
		md, err := generateMetadataFromFile(rpfInfo(), rpfNewFileReader(ctx, t, crate))
		require.NoError(t, err)
		require.Equal(t, "demo", md.Name)
		require.Equal(t, "1.0.0", md.Version)
		require.Equal(t, "MIT", md.License)
		require.Equal(t, "# readme", md.Readme)
		require.Equal(t, []string{"a", "b"}, md.Keywords)
		require.Equal(t, "https://example.test/repo", md.RepositoryURL)
		require.False(t, md.Yanked)
		require.Len(t, md.Dependencies, 4)
	})

	t.Run("success without readme", func(t *testing.T) {
		toml := "[package]\nname = \"demo\"\nversion = \"1.0.0\"\n"
		crate := rpfBuildCrate(t, map[string]string{"demo-1.0.0/Cargo.toml": toml})
		md, err := generateMetadataFromFile(rpfInfo(), rpfNewFileReader(ctx, t, crate))
		require.NoError(t, err)
		require.Empty(t, md.Readme)
		require.Empty(t, md.Dependencies)
	})

	t.Run("missing cargo toml", func(t *testing.T) {
		crate := rpfBuildCrate(t, map[string]string{"demo-1.0.0/other": "x"})
		_, err := generateMetadataFromFile(rpfInfo(), rpfNewFileReader(ctx, t, crate))
		require.ErrorContains(t, err, "cargo.toml not found")
	})

	t.Run("invalid cargo toml", func(t *testing.T) {
		crate := rpfBuildCrate(t, map[string]string{"demo-1.0.0/Cargo.toml": "[package\n"})
		_, err := generateMetadataFromFile(rpfInfo(), rpfNewFileReader(ctx, t, crate))
		require.ErrorContains(t, err, "failed to parse cargo.toml")
	})

	t.Run("declared readme missing", func(t *testing.T) {
		crate := rpfBuildCrate(t, map[string]string{"demo-1.0.0/Cargo.toml": rpfCargoToml})
		_, err := generateMetadataFromFile(rpfInfo(), rpfNewFileReader(ctx, t, crate))
		require.ErrorContains(t, err, "readme not found")
	})

	t.Run("not a crate", func(t *testing.T) {
		_, err := generateMetadataFromFile(rpfInfo(), rpfNewFileReader(ctx, t, []byte("garbage")))
		require.Error(t, err)
	})
}
