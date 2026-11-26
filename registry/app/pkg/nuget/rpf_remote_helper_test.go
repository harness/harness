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

package nuget

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/harness/gitness/registry/app/remote/registry"

	"github.com/stretchr/testify/require"
)

type rpfFakeNugetAdapter struct {
	registry.NugetRegistry
	body  string
	count int64
	err   error
}

func (f *rpfFakeNugetAdapter) reader() (io.ReadCloser, error) {
	if f.err != nil {
		return nil, f.err
	}
	return io.NopCloser(strings.NewReader(f.body)), nil
}

func (f *rpfFakeNugetAdapter) GetPackage(context.Context, string, string, string, string) (io.ReadCloser, error) {
	return f.reader()
}

func (f *rpfFakeNugetAdapter) GetPackageMetadata(context.Context, string, string) (io.ReadCloser, error) {
	return f.reader()
}

func (f *rpfFakeNugetAdapter) GetPackageVersionMetadataV2(context.Context, string, string) (io.ReadCloser, error) {
	return f.reader()
}

func (f *rpfFakeNugetAdapter) ListPackageVersion(context.Context, string) (io.ReadCloser, error) {
	return f.reader()
}

func (f *rpfFakeNugetAdapter) ListPackageVersionV2(context.Context, string) (io.ReadCloser, error) {
	return f.reader()
}

func (f *rpfFakeNugetAdapter) SearchPackageV2(context.Context, string, int, int) (io.ReadCloser, error) {
	return f.reader()
}

func (f *rpfFakeNugetAdapter) SearchPackage(context.Context, string, int, int) (io.ReadCloser, error) {
	return f.reader()
}

func (f *rpfFakeNugetAdapter) CountPackageV2(context.Context, string) (int64, error) {
	return f.count, f.err
}

func (f *rpfFakeNugetAdapter) CountPackageVersionV2(context.Context, string) (int64, error) {
	return f.count, f.err
}

func TestRpfNugetRemoteHelperReaders(t *testing.T) {
	ctx := context.Background()
	calls := map[string]func(h *remoteRegistryHelper) (io.ReadCloser, error){
		"GetFile": func(h *remoteRegistryHelper) (io.ReadCloser, error) {
			return h.GetFile(ctx, "demo", "1.0.0", "ep", "demo.nupkg")
		},
		"GetPackageMetadata": func(h *remoteRegistryHelper) (io.ReadCloser, error) {
			return h.GetPackageMetadata(ctx, "demo", "ep")
		},
		"GetPackageVersionMetadataV2": func(h *remoteRegistryHelper) (io.ReadCloser, error) {
			return h.GetPackageVersionMetadataV2(ctx, "demo", "1.0.0")
		},
		"ListPackageVersion": func(h *remoteRegistryHelper) (io.ReadCloser, error) {
			return h.ListPackageVersion(ctx, "demo")
		},
		"ListPackageVersionV2": func(h *remoteRegistryHelper) (io.ReadCloser, error) {
			return h.ListPackageVersionV2(ctx, "demo")
		},
		"SearchPackageV2": func(h *remoteRegistryHelper) (io.ReadCloser, error) {
			return h.SearchPackageV2(ctx, "term", 10, 0)
		},
		"SearchPackage": func(h *remoteRegistryHelper) (io.ReadCloser, error) {
			return h.SearchPackage(ctx, "term", 10, 0)
		},
	}
	for name, call := range calls {
		t.Run(name, func(t *testing.T) {
			rc, err := call(&remoteRegistryHelper{adapter: &rpfFakeNugetAdapter{body: "payload"}})
			require.NoError(t, err)
			data, err := io.ReadAll(rc)
			require.NoError(t, err)
			require.Equal(t, "payload", string(data))

			_, err = call(&remoteRegistryHelper{adapter: &rpfFakeNugetAdapter{err: errors.New("down")}})
			require.Error(t, err)
		})
	}
}

func TestRpfNugetRemoteHelperCounts(t *testing.T) {
	ctx := context.Background()
	h := &remoteRegistryHelper{adapter: &rpfFakeNugetAdapter{count: 7}}
	n, err := h.CountPackageV2(ctx, "term")
	require.NoError(t, err)
	require.Equal(t, int64(7), n)
	n, err = h.CountPackageVersionV2(ctx, "demo")
	require.NoError(t, err)
	require.Equal(t, int64(7), n)

	h = &remoteRegistryHelper{adapter: &rpfFakeNugetAdapter{count: 7, err: errors.New("down")}}
	n, err = h.CountPackageV2(ctx, "term")
	require.Error(t, err)
	require.Zero(t, n)
	n, err = h.CountPackageVersionV2(ctx, "demo")
	require.Error(t, err)
	require.Zero(t, n)
}
