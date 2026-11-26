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

package generic

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/harness/gitness/registry/app/pkg/commons"
	"github.com/harness/gitness/registry/app/remote/registry"

	"github.com/stretchr/testify/require"
)

type rpfFakeGenericAdapter struct {
	registry.GenericRegistry
	headers *commons.ResponseHeaders
	file    io.ReadCloser
	err     error
}

func (f *rpfFakeGenericAdapter) GetFile(context.Context, string) (*commons.ResponseHeaders, io.ReadCloser, error) {
	return f.headers, f.file, f.err
}

func (f *rpfFakeGenericAdapter) HeadFile(context.Context, string) (*commons.ResponseHeaders, error) {
	return f.headers, f.err
}

func TestRpfGenericRemoteGetFile(t *testing.T) {
	ctx := context.Background()
	h := &remoteRegistryHelper{adapter: &rpfFakeGenericAdapter{file: io.NopCloser(strings.NewReader("x"))}}
	rc, err := h.GetFile(ctx, "a/b")
	require.NoError(t, err)
	require.NotNil(t, rc)

	// Failure with and without response headers.
	withHeaders := &commons.ResponseHeaders{Code: http.StatusBadGateway, Headers: map[string]string{"A": "b"}}
	for _, headers := range []*commons.ResponseHeaders{nil, withHeaders} {
		h = &remoteRegistryHelper{adapter: &rpfFakeGenericAdapter{headers: headers, err: errors.New("down")}}
		_, err = h.GetFile(ctx, "a/b")
		require.ErrorContains(t, err, "GetFile Failure")
	}
}

func TestRpfGenericRemoteHeadFile(t *testing.T) {
	ctx := context.Background()
	want := &commons.ResponseHeaders{Code: http.StatusOK}
	h := &remoteRegistryHelper{adapter: &rpfFakeGenericAdapter{headers: want}}
	got, err := h.HeadFile(ctx, "a/b")
	require.NoError(t, err)
	require.Same(t, want, got)

	h = &remoteRegistryHelper{adapter: &rpfFakeGenericAdapter{err: errors.New("down")}}
	got, err = h.HeadFile(ctx, "a/b")
	require.ErrorContains(t, err, "HeadFile Failure")
	require.Nil(t, got)
}
