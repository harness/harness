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

package utils_test

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/harness/gitness/registry/app/api/handler/utils"
	"github.com/harness/gitness/registry/app/common/lib/errors"

	"github.com/lib/pq"
	"github.com/stretchr/testify/require"
)

func TestRchMapLibErrorCodeToHTTPStatus(t *testing.T) {
	tests := map[string]int{
		errors.NotFoundCode:         http.StatusNotFound,
		errors.UnAuthorizedCode:     http.StatusUnauthorized,
		errors.ForbiddenCode:        http.StatusForbidden,
		errors.RateLimitCode:        http.StatusTooManyRequests,
		errors.BadRequestCode:       http.StatusBadRequest,
		errors.ConflictCode:         http.StatusConflict,
		errors.MethodNotAllowedCode: http.StatusMethodNotAllowed,
		errors.GeneralCode:          http.StatusInternalServerError,
		"SOMETHING_ELSE":            http.StatusInternalServerError,
	}
	for code, want := range tests {
		t.Run(code, func(t *testing.T) {
			require.Equal(t, want, utils.MapLibErrorCodeToHTTPStatus(code))
		})
	}
}

func TestRchPatternAllowed(t *testing.T) {
	tests := []struct {
		name     string
		allowed  pq.StringArray
		blocked  pq.StringArray
		artifact string
		wantErr  bool
	}{
		{name: "no patterns allows everything", artifact: "pkg:1.0"},
		{name: "allowed pattern matches", allowed: pq.StringArray{"^pkg:"}, artifact: "pkg:1.0"},
		{name: "allowed pattern misses", allowed: pq.StringArray{"^other:"}, artifact: "pkg:1.0", wantErr: true},
		{name: "blocked pattern matches", blocked: pq.StringArray{"1\\.0$"}, artifact: "pkg:1.0", wantErr: true},
		{name: "blocked pattern misses", blocked: pq.StringArray{"2\\.0$"}, artifact: "pkg:1.0"},
		{name: "blocked wins over allowed", allowed: pq.StringArray{"pkg"}, blocked: pq.StringArray{"pkg"},
			artifact: "pkg:1.0", wantErr: true},
		{name: "second allowed pattern matches", allowed: pq.StringArray{"^a", "^pkg"}, artifact: "pkg:1.0"},
		{name: "invalid blocked regex", blocked: pq.StringArray{"("}, artifact: "pkg", wantErr: true},
		{name: "invalid allowed regex", allowed: pq.StringArray{"("}, artifact: "pkg", wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := utils.PatternAllowed(tc.allowed, tc.blocked, tc.artifact)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

type rchForm struct {
	Name    string            `json:"name"`
	Tags    []string          `json:"tags,omitempty"`
	Extra   map[string]string `json:"extra"`
	Count   int               `json:"count"`
	NoTag   string
	Ignored bool
}

func rchFormRequest(t *testing.T, values url.Values) *http.Request {
	t.Helper()
	buf := &bytes.Buffer{}
	mw := multipart.NewWriter(buf)
	for key, list := range values {
		for _, v := range list {
			require.NoError(t, mw.WriteField(key, v))
		}
	}
	require.NoError(t, mw.Close())
	req := httptest.NewRequest(http.MethodPost, "/x", buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	return req
}

func TestRchFillFromForm(t *testing.T) {
	t.Run("fills strings slices and maps", func(t *testing.T) {
		req := rchFormRequest(t, url.Values{
			"name":  {"pkg"},
			"tags":  {"a", "b"},
			"extra": {`{"k":"v"}`},
		})
		var out rchForm
		require.NoError(t, utils.FillFromForm(req, &out))
		require.Equal(t, "pkg", out.Name)
		require.Equal(t, []string{"a", "b"}, out.Tags)
		require.Equal(t, map[string]string{"k": "v"}, out.Extra)
		require.Empty(t, out.NoTag)
	})
	t.Run("empty map value yields empty map", func(t *testing.T) {
		var out rchForm
		require.NoError(t, utils.FillFromForm(rchFormRequest(t, url.Values{"name": {"x"}}), &out))
		require.NotNil(t, out.Extra)
		require.Empty(t, out.Extra)
	})
	t.Run("map falls back to comma separated pair", func(t *testing.T) {
		var out rchForm
		req := rchFormRequest(t, url.Values{"extra": {"key, value"}})
		require.NoError(t, utils.FillFromForm(req, &out))
		require.Equal(t, map[string]string{"key": "value"}, out.Extra)
	})
	t.Run("invalid map value is rejected", func(t *testing.T) {
		var out rchForm
		req := rchFormRequest(t, url.Values{"extra": {"not-json"}})
		require.Error(t, utils.FillFromForm(req, &out))
	})
	t.Run("non multipart request is rejected", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/x", strings.NewReader("name=pkg"))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		var out rchForm
		require.Error(t, utils.FillFromForm(req, &out))
	})
}

func TestRchGetFileReader(t *testing.T) {
	newRequest := func(t *testing.T, field string) *http.Request {
		t.Helper()
		buf := &bytes.Buffer{}
		mw := multipart.NewWriter(buf)
		require.NoError(t, mw.WriteField("other", "x"))
		part, err := mw.CreateFormFile(field, "pkg.bin")
		require.NoError(t, err)
		_, err = part.Write([]byte("content"))
		require.NoError(t, err)
		require.NoError(t, mw.Close())
		req := httptest.NewRequest(http.MethodPost, "/x", buf)
		req.Header.Set("Content-Type", mw.FormDataContentType())
		return req
	}

	t.Run("finds the file part", func(t *testing.T) {
		part, name, err := utils.GetFileReader(newRequest(t, "package"), "package")
		require.NoError(t, err)
		require.Equal(t, "pkg.bin", name)
		data, err := io.ReadAll(part)
		require.NoError(t, err)
		require.Equal(t, "content", string(data))
	})
	t.Run("missing form key", func(t *testing.T) {
		_, _, err := utils.GetFileReader(newRequest(t, "package"), "absent")
		require.Error(t, err)
	})
	t.Run("not multipart", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/x", strings.NewReader("raw"))
		_, _, err := utils.GetFileReader(req, "package")
		require.Error(t, err)
	})
	t.Run("truncated multipart body", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/x", strings.NewReader("--b\r\ngarbage"))
		req.Header.Set("Content-Type", "multipart/form-data; boundary=b")
		_, _, err := utils.GetFileReader(req, "package")
		require.Error(t, err)
	})
}
