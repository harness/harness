// Copyright 2023 Harness, Inc.
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

package render

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/harness/gitness/app/url"
	"github.com/harness/gitness/types"

	"github.com/stretchr/testify/require"
)

type hostnameProvider struct {
	url.Provider
}

func (hostnameProvider) GetAPIHostname(context.Context) string { return "api.example.com" }

func TestPagination(t *testing.T) {
	tests := []struct {
		name        string
		target      string
		page, size  int
		total       int
		wantHeaders map[string]string
		wantLinks   []string
		noLinks     []string
	}{
		{
			name:   "first page of many",
			target: "/repos?page=1&limit=10",
			page:   1,
			size:   10,
			total:  35,
			wantHeaders: map[string]string{
				"x-page": "1", "x-per-page": "10", "x-total": "35", "x-total-pages": "4", "x-next-page": "2",
			},
			wantLinks: []string{`rel="next"`, `rel="last"`},
			noLinks:   []string{`rel="prev"`},
		},
		{
			name:        "middle page",
			target:      "/repos?page=2&limit=10",
			page:        2,
			size:        10,
			total:       35,
			wantHeaders: map[string]string{"x-next-page": "3", "x-prev-page": "1"},
			wantLinks:   []string{`rel="next"`, `rel="prev"`, `rel="last"`},
		},
		{
			name:        "last page",
			target:      "/repos?page=4&limit=10",
			page:        4,
			size:        10,
			total:       35,
			wantHeaders: map[string]string{"x-prev-page": "3", "x-total-pages": "4"},
			wantLinks:   []string{`rel="prev"`, `rel="last"`},
			noLinks:     []string{`rel="next"`},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			Pagination(httptest.NewRequest(http.MethodGet, tt.target, nil), rec, tt.page, tt.size, tt.total)

			for k, v := range tt.wantHeaders {
				require.Equal(t, v, rec.Header().Get(k), k)
			}
			links := strings.Join(rec.Header().Values("Link"), ",")
			for _, l := range tt.wantLinks {
				require.Contains(t, links, l)
			}
			for _, l := range tt.noLinks {
				require.NotContains(t, links, l)
			}
			if tt.wantHeaders["x-next-page"] == "" {
				require.Empty(t, rec.Header().Get("x-next-page"))
			}
		})
	}
}

func TestPaginationNoTotal(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/repos?page=2&limit=5", nil)

	rec := httptest.NewRecorder()
	PaginationNoTotal(req, rec, 2, 5, false)
	require.Equal(t, "2", rec.Header().Get("x-page"))
	require.Equal(t, "5", rec.Header().Get("x-per-page"))
	require.Equal(t, "3", rec.Header().Get("x-next-page"))
	require.Equal(t, "1", rec.Header().Get("x-prev-page"))
	require.Empty(t, rec.Header().Get("x-total"))

	rec = httptest.NewRecorder()
	PaginationNoTotal(req, rec, 1, 5, true)
	require.Empty(t, rec.Header().Get("x-next-page"))
	require.Empty(t, rec.Header().Get("x-prev-page"))
	require.Empty(t, rec.Header().Values("Link"))
}

func TestPaginationLimit(t *testing.T) {
	rec := httptest.NewRecorder()
	PaginationLimit(nil, rec, 17)
	require.Equal(t, "17", rec.Header().Get("x-total"))
}

func TestNoCache(t *testing.T) {
	rec := httptest.NewRecorder()
	NoCache(rec)
	require.Equal(t, "no-cache", rec.Header().Get("Pragma"))
	require.Equal(t, "no-cache, max-age=0, must-revalidate", rec.Header().Get("Cache-Control"))
	require.NotEmpty(t, rec.Header().Get("Expires"))
}

func TestDeleteSuccessful(t *testing.T) {
	rec := httptest.NewRecorder()
	DeleteSuccessful(rec)
	require.Equal(t, http.StatusNoContent, rec.Code)
	require.Empty(t, rec.Body.String())
}

type rndFailingReader struct{}

func (rndFailingReader) Read([]byte) (int, error) { return 0, context.DeadlineExceeded }

func TestReader(t *testing.T) {
	rec := httptest.NewRecorder()
	Reader(context.Background(), rec, http.StatusAccepted, strings.NewReader("raw body"))
	require.Equal(t, http.StatusAccepted, rec.Code)
	require.Equal(t, "raw body", rec.Body.String())

	// A failing reader is logged, the status is still written.
	rec = httptest.NewRecorder()
	Reader(context.Background(), rec, http.StatusOK, rndFailingReader{})
	require.Equal(t, http.StatusOK, rec.Code)
	require.Empty(t, rec.Body.String())
}

func TestUnprocessableAndViolations(t *testing.T) {
	rec := httptest.NewRecorder()
	Unprocessable(rec, map[string]string{"k": "v"})
	require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
	require.JSONEq(t, `{"k":"v"}`, rec.Body.String())

	rec = httptest.NewRecorder()
	Violations(rec, nil)
	require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
	require.Contains(t, rec.Body.String(), "violations")

	rec = httptest.NewRecorder()
	Violations(rec, []types.RuleViolations{})
	require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
}

func TestGitBasicAuth(t *testing.T) {
	rec := httptest.NewRecorder()
	GitBasicAuth(context.Background(), rec, hostnameProvider{})
	require.Equal(t, http.StatusUnauthorized, rec.Code)
	require.Equal(t, `Basic realm="api.example.com"`, rec.Header().Get("WWW-Authenticate"))
}

func TestFormattedErrorHelpers(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name   string
		render func(http.ResponseWriter)
		status int
		msg    string
	}{
		{"forbiddenf", func(w http.ResponseWriter) { Forbiddenf(ctx, w, "no %s", "way") }, http.StatusForbidden, "no way"},
		{"badRequestf", func(w http.ResponseWriter) { BadRequestf(ctx, w, "bad %d", 1) }, http.StatusBadRequest, "bad 1"},
		{"internalError", func(w http.ResponseWriter) { InternalError(ctx, w) }, http.StatusInternalServerError, ""},
		{
			"internalErrorf", func(w http.ResponseWriter) { InternalErrorf(ctx, w, "boom %s", "x") },
			http.StatusInternalServerError, "boom x",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			tt.render(rec)
			require.Equal(t, tt.status, rec.Code)
			if tt.msg != "" {
				require.Contains(t, rec.Body.String(), tt.msg)
			}
		})
	}
}
