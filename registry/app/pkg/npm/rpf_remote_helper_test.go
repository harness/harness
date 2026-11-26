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

package npm

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/harness/gitness/registry/app/metadata/npm"
	"github.com/harness/gitness/registry/app/remote/registry"

	"github.com/stretchr/testify/require"
)

type rpfFakeNpmAdapter struct {
	registry.NpmRegistry
	metadata *npm.PackageMetadata
	file     io.ReadCloser
	err      error
}

func (f *rpfFakeNpmAdapter) GetPackageMetadata(context.Context, string) (*npm.PackageMetadata, error) {
	return f.metadata, f.err
}

func (f *rpfFakeNpmAdapter) GetPackage(context.Context, string, string) (io.ReadCloser, error) {
	return f.file, f.err
}

func TestRpfNpmRemoteGetPackage(t *testing.T) {
	ctx := context.Background()
	h := &remoteRegistryHelper{adapter: &rpfFakeNpmAdapter{file: io.NopCloser(strings.NewReader("tgz"))}}
	rc, err := h.GetPackage(ctx, "demo", "1.0.0")
	require.NoError(t, err)
	require.NotNil(t, rc)

	h = &remoteRegistryHelper{adapter: &rpfFakeNpmAdapter{err: errors.New("down")}}
	_, err = h.GetPackage(ctx, "demo", "1.0.0")
	require.Error(t, err)
}

func TestRpfNpmRemoteGetPackageMetadata(t *testing.T) {
	ctx := context.Background()
	want := &npm.PackageMetadata{Name: "demo"}
	h := &remoteRegistryHelper{adapter: &rpfFakeNpmAdapter{metadata: want}}
	got, err := h.GetPackageMetadata(ctx, "demo")
	require.NoError(t, err)
	require.Same(t, want, got)

	h = &remoteRegistryHelper{adapter: &rpfFakeNpmAdapter{err: errors.New("down")}}
	got, err = h.GetPackageMetadata(ctx, "demo")
	require.Error(t, err)
	require.Nil(t, got)
}

func TestRpfNpmRemoteGetVersionMetadata(t *testing.T) {
	ctx := context.Background()
	newMetadata := func() *npm.PackageMetadata {
		return &npm.PackageMetadata{
			Name: "demo",
			Versions: map[string]*npm.PackageMetadataVersion{
				"1.0.0": {Version: "1.0.0"},
				"2.0.0": {Version: "2.0.0"},
			},
		}
	}

	h := &remoteRegistryHelper{adapter: &rpfFakeNpmAdapter{metadata: newMetadata()}}
	got, err := h.GetVersionMetadata(ctx, "demo", "2.0.0")
	require.NoError(t, err)
	require.Len(t, got.Versions, 1)
	require.Contains(t, got.Versions, "2.0.0")

	// An unknown version leaves the full version list in place.
	h = &remoteRegistryHelper{adapter: &rpfFakeNpmAdapter{metadata: newMetadata()}}
	got, err = h.GetVersionMetadata(ctx, "demo", "9.9.9")
	require.NoError(t, err)
	require.Len(t, got.Versions, 2)

	h = &remoteRegistryHelper{adapter: &rpfFakeNpmAdapter{err: errors.New("down")}}
	_, err = h.GetVersionMetadata(ctx, "demo", "1.0.0")
	require.Error(t, err)
}
