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

package commons

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/harness/gitness/registry/app/dist_temp/errcode"

	"github.com/stretchr/testify/require"
)

func TestRpfIsEmpty(t *testing.T) {
	var nilSlice []string
	var nilPtr *[]string
	emptyPtr := &[]string{}
	nonEmptyPtr := &[]string{"a"}

	tests := []struct {
		name string
		in   any
		want bool
	}{
		{"nil", nil, true},
		{"nil slice", nilSlice, true},
		{"empty slice", []string{}, true},
		{"slice", []string{"a"}, false},
		{"empty string", "", true},
		{"string", "x", false},
		{"empty map", map[string]int{}, true},
		{"map", map[string]int{"a": 1}, false},
		{"array", [2]int{1, 2}, false},
		{"nil pointer", nilPtr, true},
		{"empty pointer", emptyPtr, true},
		{"pointer", nonEmptyPtr, false},
		{"int", 5, false},
		{"zero int", 0, false},
		{"struct", struct{}{}, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, IsEmpty(tc.in))
		})
	}
}

func TestRpfIsEmptyError(t *testing.T) {
	require.True(t, IsEmptyError(errcode.Error{}))
	require.False(t, IsEmptyError(errcode.Error{Code: errcode.CodeError(1)}))
}

func TestRpfResponseHeadersWrite(t *testing.T) {
	h := &ResponseHeaders{Headers: map[string]string{"X-Test": "1"}, Code: http.StatusCreated}

	rec := httptest.NewRecorder()
	h.WriteToResponse(rec)
	require.Equal(t, http.StatusCreated, rec.Code)
	require.Equal(t, "1", rec.Header().Get("X-Test"))

	rec = httptest.NewRecorder()
	h.WriteHeadersToResponse(rec)
	require.Equal(t, "1", rec.Header().Get("X-Test"))
	require.False(t, rec.Flushed)

	// Zero code and nil headers do not write anything.
	rec = httptest.NewRecorder()
	(&ResponseHeaders{}).WriteToResponse(rec)
	require.Empty(t, rec.Header())

	// Nil receivers and writers are ignored.
	var nilHeaders *ResponseHeaders
	require.NotPanics(t, func() {
		nilHeaders.WriteToResponse(httptest.NewRecorder())
		nilHeaders.WriteHeadersToResponse(httptest.NewRecorder())
		h.WriteToResponse(nil)
		h.WriteHeadersToResponse(nil)
	})
}

type rpfErrReader struct{}

func (rpfErrReader) Read([]byte) (int, error) { return 0, errors.New("boom") }

func TestRpfServeContent(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/x", nil)

	t.Run("read closer is copied", func(t *testing.T) {
		rec := httptest.NewRecorder()
		err := ServeContent(rec, req, nil, "f.txt", io.NopCloser(bytes.NewBufferString("data")))
		require.NoError(t, err)
		require.Equal(t, "data", rec.Body.String())
	})

	t.Run("copy failure is wrapped", func(t *testing.T) {
		rec := httptest.NewRecorder()
		err := ServeContent(rec, req, nil, "f.txt", io.NopCloser(rpfErrReader{}))
		require.ErrorContains(t, err, "failed to copy content")
	})

	t.Run("no content", func(t *testing.T) {
		err := ServeContent(httptest.NewRecorder(), req, nil, "f.txt", nil)
		require.ErrorContains(t, err, "no content to serve")
	})
}

func TestRpfUserErrors(t *testing.T) {
	e := New(http.StatusTeapot, "short and stout", "detail")
	require.Equal(t, "short and stout", e.Error())
	require.Equal(t, http.StatusTeapot, e.Status)
	require.Equal(t, "detail", e.Detail)

	nf := NotFoundError("missing", nil)
	require.Equal(t, http.StatusNotFound, nf.Status)
	require.Equal(t, "missing", nf.Error())

	require.Equal(t, http.StatusMethodNotAllowed, ErrNotSupported.Status)
	require.NotEmpty(t, ErrBadRequest.Error())
}
