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

package rpm

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/harness/gitness/registry/app/remote/registry"

	"github.com/stretchr/testify/require"
)

type rpfFakeRpmAdapter struct {
	registry.RpmRegistry
	file io.ReadCloser
	err  error
}

func (f *rpfFakeRpmAdapter) GetMetadataFile(context.Context, string) (io.ReadCloser, error) {
	return f.file, f.err
}

func (f *rpfFakeRpmAdapter) GetPackage(context.Context, string) (io.ReadCloser, error) {
	return f.file, f.err
}

func TestRpfRpmRemoteHelper(t *testing.T) {
	ctx := context.Background()
	ok := &remoteRegistryHelper{adapter: &rpfFakeRpmAdapter{file: io.NopCloser(strings.NewReader("x"))}}
	failing := &remoteRegistryHelper{adapter: &rpfFakeRpmAdapter{err: errors.New("down")}}

	rc, err := ok.GetMetadataFile(ctx, "repodata/repomd.xml")
	require.NoError(t, err)
	require.NotNil(t, rc)
	_, err = failing.GetMetadataFile(ctx, "repodata/repomd.xml")
	require.Error(t, err)

	rc, err = ok.GetPackage(ctx, "Packages/a.rpm")
	require.NoError(t, err)
	require.NotNil(t, rc)
	rc, err = failing.GetPackage(ctx, "Packages/a.rpm")
	require.Error(t, err)
	require.Nil(t, rc)
}
