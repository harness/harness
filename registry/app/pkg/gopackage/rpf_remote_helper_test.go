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

package gopackage

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/harness/gitness/registry/app/remote/registry"

	"github.com/stretchr/testify/require"
)

type rpfFakeGoAdapter struct {
	registry.GoPackageRegistry
	file     io.ReadCloser
	err      error
	lastPath string
}

func (f *rpfFakeGoAdapter) GetPackageFile(_ context.Context, filepath string) (io.ReadCloser, error) {
	f.lastPath = filepath
	return f.file, f.err
}

func TestRpfGoRemoteGetPackageFile(t *testing.T) {
	ctx := context.Background()

	fake := &rpfFakeGoAdapter{file: io.NopCloser(strings.NewReader("zip"))}
	h := &remoteRegistryHelper{adapter: fake}
	rc, err := h.GetPackageFile(ctx, "example.com/hello", "/example.com/hello/@v/v1.0.0.zip")
	require.NoError(t, err)
	require.NotNil(t, rc)
	require.Equal(t, "example.com/hello/@v/v1.0.0.zip", fake.lastPath)

	h = &remoteRegistryHelper{adapter: &rpfFakeGoAdapter{}}
	_, err = h.GetPackageFile(ctx, "example.com/hello", "x")
	require.ErrorContains(t, err, "file not found for package")

	h = &remoteRegistryHelper{adapter: &rpfFakeGoAdapter{err: errors.New("down")}}
	_, err = h.GetPackageFile(ctx, "example.com/hello", "x")
	require.ErrorContains(t, err, "failed to get package file")
}
