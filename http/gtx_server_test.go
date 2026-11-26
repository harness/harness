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

package http

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func gtxFreePort(t *testing.T) int {
	t.Helper()

	l, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer func() { _ = l.Close() }()

	addr, ok := l.Addr().(*net.TCPAddr)
	require.True(t, ok)

	return addr.Port
}

func gtxGet(ctx context.Context, t *testing.T, url string) (int, string) {
	t.Helper()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	require.NoError(t, err)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err.Error()
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	return resp.StatusCode, string(body)
}

func TestGtxNewServer_DefaultTimeout(t *testing.T) {
	assert.Equal(t, DefaultReadHeaderTimeout, NewServer(Config{}, nil).config.ReadHeaderTimeout)
	assert.Equal(t, time.Minute, NewServer(Config{ReadHeaderTimeout: time.Minute}, nil).config.ReadHeaderTimeout)
}

func TestGtxServer_ListenAndServe(t *testing.T) {
	ctx := context.Background()
	port := gtxFreePort(t)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprintf(w, "hello %s", r.URL.Path)
	})

	server := NewServer(Config{Host: "127.0.0.1", Port: port}, handler)
	g, shutdown := server.ListenAndServe()

	url := fmt.Sprintf("http://127.0.0.1:%d/gtx", port)

	require.Eventually(t, func() bool {
		status, _ := gtxGet(ctx, t, url)
		return status == http.StatusOK
	}, 5*time.Second, 20*time.Millisecond)

	status, body := gtxGet(ctx, t, url)
	assert.Equal(t, http.StatusOK, status)
	assert.Equal(t, "hello /gtx", body)

	shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	require.NoError(t, shutdown(shutdownCtx))

	require.ErrorIs(t, g.Wait(), http.ErrServerClosed)

	status, _ = gtxGet(ctx, t, url)
	assert.Zero(t, status, "the server doesn't accept connections after the shutdown")
}

func TestGtxServer_ListenAndServeFailsIfPortIsTaken(t *testing.T) {
	l, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer func() { _ = l.Close() }()

	addr, ok := l.Addr().(*net.TCPAddr)
	require.True(t, ok)

	g, shutdown := NewServer(Config{Host: "127.0.0.1", Port: addr.Port}, http.NotFoundHandler()).ListenAndServe()

	err = g.Wait()
	require.Error(t, err)
	assert.False(t, errors.Is(err, http.ErrServerClosed))

	require.NoError(t, shutdown(context.Background()))
}

func TestGtxRedirect(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want string
	}{
		{name: "root", url: "http://example.com/", want: "https://example.com/"},
		{name: "path", url: "http://example.com/a/b", want: "https://example.com/a/b"},
		{name: "empty path", url: "http://example.com", want: "https://example.com/"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			redirect(rec, httptest.NewRequest(http.MethodGet, test.url, nil))

			assert.Equal(t, http.StatusTemporaryRedirect, rec.Code)
			assert.Equal(t, test.want, rec.Header().Get("Location"))
		})
	}
}

func TestGtxNoOpListenAndServeServer(t *testing.T) {
	var s ListenAndServeServer = NoOpListenAndServeServer{}

	g, shutdown := s.ListenAndServe()
	require.NotNil(t, g)
	require.NoError(t, g.Wait())
	require.NoError(t, shutdown(context.Background()))
}
