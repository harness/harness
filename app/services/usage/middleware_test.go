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

package usage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/harness/gitness/app/api/request"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"
)

func TestMiddleware(t *testing.T) {
	var m Metric
	mock := &mockInterface{
		SendFunc: func(_ context.Context, payload Metric) error {
			m.Out += payload.Out
			m.In += payload.In
			return nil
		},
	}

	r := chi.NewRouter()
	r.Route(fmt.Sprintf("/testing/{%s}", request.PathParamRepoRef), func(r chi.Router) {
		r.Use(Middleware(mock, true))
		r.Post("/", func(w http.ResponseWriter, r *http.Request) {
			// read from body
			_, _ = io.Copy(io.Discard, r.Body)
			// write to response
			_, _ = w.Write([]byte(sampleText))
		})
	})

	ts := httptest.NewServer(r)
	defer ts.Close()

	body := []byte(sampleText)

	_, _ = testRequest(t, ts, http.MethodPost, "/testing/"+spaceRef, bytes.NewReader(body))

	require.Equal(t, int64(sampleLength), m.Out)
	require.Equal(t, int64(sampleLength), m.In)
}

func testRequest(t *testing.T, ts *httptest.Server, method, path string, body io.Reader) (*http.Response, string) {
	t.Helper()

	req, err := http.NewRequest(method, ts.URL+path, body)
	if err != nil {
		t.Fatal(err)
		return nil, ""
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
		return nil, ""
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
		return nil, ""
	}
	defer resp.Body.Close()

	return resp, string(respBody)
}
