// Copyright 2026 Harness, Inc.
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

package request

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/harness/gitness/git/api"
	"github.com/harness/gitness/types/enum"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"
)

func ct5WithURLParams(r *http.Request, params map[string]string) *http.Request {
	rctx := chi.NewRouteContext()
	for k, v := range params {
		rctx.URLParams.Add(k, v)
	}
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

func TestCt5WebhookPathParams(t *testing.T) {
	req := ct5WithURLParams(httptest.NewRequest(http.MethodGet, "/", nil), map[string]string{
		PathParamWebhookIdentifier:  "hook",
		PathParamWebhookExecutionID: "12",
	})

	identifier, err := GetWebhookIdentifierFromPath(req)
	require.NoError(t, err)
	require.Equal(t, "hook", identifier)

	id, err := GetWebhookExecutionIDFromPath(req)
	require.NoError(t, err)
	require.Equal(t, int64(12), id)

	empty := httptest.NewRequest(http.MethodGet, "/", nil)
	_, err = GetWebhookIdentifierFromPath(empty)
	require.Error(t, err)
	_, err = GetWebhookExecutionIDFromPath(empty)
	require.Error(t, err)
}

func TestCt5ParseWebhookFilters(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/?query=abc&page=2&limit=5&sort=created&order=desc", nil)

	filter := ParseWebhookFilter(req)
	require.Equal(t, "abc", filter.Query)
	require.Equal(t, 2, filter.Page)
	require.Equal(t, 5, filter.Size)
	require.Equal(t, enum.WebhookAttrCreated, filter.Sort)
	require.Equal(t, enum.OrderDesc, filter.Order)

	execFilter := ParseWebhookExecutionFilter(req)
	require.Equal(t, 2, execFilter.Page)
	require.Equal(t, 5, execFilter.Size)

	require.Equal(t, enum.WebhookAttrNone, ParseSortWebhook(httptest.NewRequest(http.MethodGet, "/?sort=bogus", nil)))
}

func TestCt5ParseArchiveParams(t *testing.T) {
	t.Run("all parameters", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet,
			"/?prefix=pre&attributes=export-ignore&time=2024-01-02%2003:04:05&compression=5&path=a&path=b", nil)
		req = ct5WithURLParams(req, map[string]string{PathParamArchiveGitRef: "refs/heads/main.tar.gz"})

		params, filename, err := ParseArchiveParams(req)
		require.NoError(t, err)
		require.Equal(t, "main.tar.gz", filename)
		require.Equal(t, api.ArchiveFormatTarGz, params.Format)
		require.Equal(t, "refs/heads/main", params.Treeish)
		require.Equal(t, "pre", params.Prefix)
		require.Equal(t, api.ArchiveAttribute("export-ignore"), params.Attributes)
		require.Equal(t, []string{"a", "b"}, params.Paths)
		require.NotNil(t, params.Time)
		require.Equal(t, time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC), *params.Time)
		require.NotNil(t, params.Compression)
		require.Equal(t, 5, *params.Compression)
	})

	t.Run("invalid optional values are ignored", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/?time=nope&compression=abc", nil)
		req = ct5WithURLParams(req, map[string]string{PathParamArchiveGitRef: "main.zip"})

		params, _, err := ParseArchiveParams(req)
		require.NoError(t, err)
		require.Equal(t, api.ArchiveFormatZip, params.Format)
		require.Equal(t, "main", params.Treeish)
		require.Nil(t, params.Time)
		require.Nil(t, params.Compression)
	})

	t.Run("unknown format", func(t *testing.T) {
		req := ct5WithURLParams(httptest.NewRequest(http.MethodGet, "/", nil),
			map[string]string{PathParamArchiveGitRef: "main.rar"})
		_, _, err := ParseArchiveParams(req)
		require.Error(t, err)
	})
}
