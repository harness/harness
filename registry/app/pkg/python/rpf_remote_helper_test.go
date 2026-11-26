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

package python

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/harness/gitness/registry/app/metadata/python"
	pypi2 "github.com/harness/gitness/registry/app/remote/adapter/commons/pypi"
	"github.com/harness/gitness/registry/app/remote/registry"

	"github.com/stretchr/testify/require"
)

type rpfFakePythonAdapter struct {
	registry.PythonRegistry
	simple *pypi2.SimpleMetadata
	meta   *python.Metadata
	file   io.ReadCloser
	err    error
}

func (f *rpfFakePythonAdapter) GetMetadata(context.Context, string) (*pypi2.SimpleMetadata, error) {
	return f.simple, f.err
}

func (f *rpfFakePythonAdapter) GetPackage(context.Context, string, string) (io.ReadCloser, error) {
	return f.file, f.err
}

func (f *rpfFakePythonAdapter) GetJSON(context.Context, string, string) (*python.Metadata, error) {
	return f.meta, f.err
}

func TestRpfPythonRemoteHelper(t *testing.T) {
	ctx := context.Background()
	failing := &remoteRegistryHelper{adapter: &rpfFakePythonAdapter{err: errors.New("down")}}

	simple := &pypi2.SimpleMetadata{}
	meta := &python.Metadata{}
	ok := &remoteRegistryHelper{adapter: &rpfFakePythonAdapter{
		simple: simple, meta: meta, file: io.NopCloser(strings.NewReader("x")),
	}}

	rc, err := ok.GetFile(ctx, "demo", "demo-1.0.whl")
	require.NoError(t, err)
	require.NotNil(t, rc)
	_, err = failing.GetFile(ctx, "demo", "demo-1.0.whl")
	require.Error(t, err)

	gotSimple, err := ok.GetMetadata(ctx, "demo")
	require.NoError(t, err)
	require.Same(t, simple, gotSimple)
	gotSimple, err = failing.GetMetadata(ctx, "demo")
	require.Error(t, err)
	require.Nil(t, gotSimple)

	gotMeta, err := ok.GetJSON(ctx, "demo", "1.0")
	require.NoError(t, err)
	require.Same(t, meta, gotMeta)
	gotMeta, err = failing.GetJSON(ctx, "demo", "1.0")
	require.Error(t, err)
	require.Nil(t, gotMeta)
}
