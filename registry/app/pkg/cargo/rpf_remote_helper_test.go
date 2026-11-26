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
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/harness/gitness/registry/app/metadata/cargo"
	"github.com/harness/gitness/registry/app/remote/registry"

	"github.com/stretchr/testify/require"
)

type rpfFakeCargoAdapter struct {
	registry.CargoRegistry
	config    *cargo.RegistryConfig
	file      io.ReadCloser
	err       error
	lastPath  string
	callCount int
}

func (f *rpfFakeCargoAdapter) GetRegistryConfig(context.Context) (*cargo.RegistryConfig, error) {
	return f.config, f.err
}

func (f *rpfFakeCargoAdapter) GetPackageFile(_ context.Context, filepath string) (io.ReadCloser, error) {
	f.callCount++
	f.lastPath = filepath
	return f.file, f.err
}

func TestRpfCargoRemoteGetRegistryConfig(t *testing.T) {
	ctx := context.Background()
	want := &cargo.RegistryConfig{}
	h := &remoteRegistryHelper{adapter: &rpfFakeCargoAdapter{config: want}}
	got, err := h.GetRegistryConfig(ctx)
	require.NoError(t, err)
	require.Same(t, want, got)

	h = &remoteRegistryHelper{adapter: &rpfFakeCargoAdapter{err: errors.New("down")}}
	got, err = h.GetRegistryConfig(ctx)
	require.ErrorContains(t, err, "failed to get registry config")
	require.Nil(t, got)
}

func TestRpfCargoRemoteGetPackageIndex(t *testing.T) {
	ctx := context.Background()
	fake := &rpfFakeCargoAdapter{file: io.NopCloser(strings.NewReader("idx"))}
	h := &remoteRegistryHelper{adapter: fake}
	rc, err := h.GetPackageIndex(ctx, "demo", "de/mo/demo")
	require.NoError(t, err)
	data, err := io.ReadAll(rc)
	require.NoError(t, err)
	require.Equal(t, "idx", string(data))
	require.Equal(t, "de/mo/demo", fake.lastPath)

	h = &remoteRegistryHelper{adapter: &rpfFakeCargoAdapter{}}
	_, err = h.GetPackageIndex(ctx, "demo", "p")
	require.ErrorContains(t, err, "index metadata not found")

	h = &remoteRegistryHelper{adapter: &rpfFakeCargoAdapter{err: errors.New("down")}}
	_, err = h.GetPackageIndex(ctx, "demo", "p")
	require.ErrorContains(t, err, "failed to get package index demo")
}

func TestRpfCargoRemoteGetPackageFile(t *testing.T) {
	ctx := context.Background()
	fake := &rpfFakeCargoAdapter{file: io.NopCloser(strings.NewReader("crate"))}
	h := &remoteRegistryHelper{adapter: fake}
	rc, err := h.GetPackageFile(ctx, "demo", "1.0.0")
	require.NoError(t, err)
	require.NotNil(t, rc)
	require.Equal(t, "demo/1.0.0/download", fake.lastPath)

	h = &remoteRegistryHelper{adapter: &rpfFakeCargoAdapter{}}
	_, err = h.GetPackageFile(ctx, "demo", "1.0.0")
	require.ErrorContains(t, err, "file not found for package")

	h = &remoteRegistryHelper{adapter: &rpfFakeCargoAdapter{err: errors.New("down")}}
	_, err = h.GetPackageFile(ctx, "demo", "1.0.0")
	require.ErrorContains(t, err, "failed to get package file")
}
