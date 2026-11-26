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

package users

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/adrg/xdg"
	"github.com/stretchr/testify/require"
	"golang.org/x/term"
	"gopkg.in/alecthomas/kingpin.v2"
)

// infAPIRequest records one request received by the fake API.
type infAPIRequest struct {
	method string
	uri    string
	auth   string
	body   map[string]any
}

type infAPI struct {
	mx       sync.Mutex
	requests []infAPIRequest
	status   int
	response string
}

func (a *infAPI) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	raw, _ := io.ReadAll(r.Body)
	var body map[string]any
	_ = json.Unmarshal(raw, &body)
	a.mx.Lock()
	a.requests = append(a.requests, infAPIRequest{
		method: r.Method, uri: r.URL.RequestURI(), auth: r.Header.Get("Authorization"), body: body,
	})
	status, response := a.status, a.response
	a.mx.Unlock()

	w.Header().Set("Content-Type", "application/json")
	if status == 0 {
		status = http.StatusOK
	}
	w.WriteHeader(status)
	_, _ = io.WriteString(w, response)
}

func (a *infAPI) last() infAPIRequest {
	a.mx.Lock()
	defer a.mx.Unlock()
	return a.requests[len(a.requests)-1]
}

// infSetup starts the fake API and points the CLI session at it.
func infSetup(t *testing.T, api *infAPI) {
	t.Helper()
	srv := httptest.NewServer(api)
	t.Cleanup(srv.Close)

	dir := t.TempDir()
	t.Cleanup(xdg.Reload) // registered first so it runs after the environment is restored.
	t.Setenv("XDG_CONFIG_HOME", dir)
	xdg.Reload()

	session, err := json.Marshal(map[string]any{
		"uri": srv.URL, "expires_at": time.Now().Add(time.Hour).Unix(), "access_token": "session-token-0000",
	})
	require.NoError(t, err)
	path := filepath.Join(dir, "app", "config.json")
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o700))
	require.NoError(t, os.WriteFile(path, session, 0o600))
}

func infCaptureStdout(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	require.NoError(t, err)
	old := os.Stdout
	os.Stdout = w
	defer func() { os.Stdout = old }()

	fn()
	require.NoError(t, w.Close())
	out, err := io.ReadAll(r)
	require.NoError(t, err)
	return string(out)
}

func infRun(t *testing.T, args ...string) (string, error) {
	t.Helper()
	app := kingpin.New("test", "")
	app.Terminate(nil)
	Register(app)
	var err error
	out := infCaptureStdout(t, func() {
		_, err = app.Parse(args)
	})
	return out, err
}

const infUserJSON = `{"id": 7, "uid": "alice", "email": "alice@example.invalid", "admin": true}`

func TestInfUsersFind(t *testing.T) {
	api := &infAPI{response: infUserJSON}
	infSetup(t, api)

	out, err := infRun(t, "users", "find", "alice@example.invalid")
	require.NoError(t, err)
	require.Contains(t, out, "id:    7")
	require.Contains(t, out, "email: alice@example.invalid")
	require.Contains(t, out, "admin: true")
	req := api.last()
	require.Equal(t, http.MethodGet, req.method)
	require.Equal(t, "/api/v1/users/alice@example.invalid", req.uri)
	require.Equal(t, "Bearer session-token-0000", req.auth)

	out, err = infRun(t, "users", "find", "alice", "--json")
	require.NoError(t, err)
	var decoded map[string]any
	require.NoError(t, json.Unmarshal([]byte(out), &decoded))
	require.Equal(t, "alice@example.invalid", decoded["email"])
}

func TestInfUsersFind_ServerError(t *testing.T) {
	api := &infAPI{status: http.StatusNotFound, response: `{"message": "resource not found"}`}
	infSetup(t, api)
	_, err := infRun(t, "users", "find", "ghost")
	require.Error(t, err)
}

func TestInfUsersList(t *testing.T) {
	api := &infAPI{response: `[` + infUserJSON + `, {"id": 8, "email": "bob@example.invalid"}]`}
	infSetup(t, api)

	out, err := infRun(t, "users", "ls", "--page", "2", "--per-page", "5")
	require.NoError(t, err)
	require.Equal(t, "/api/v1/users?page=2&limit=5", api.last().uri)
	require.Contains(t, out, "alice@example.invalid")
	require.Contains(t, out, "bob@example.invalid")
	require.Equal(t, 2, strings.Count(out, "email:"))

	out, err = infRun(t, "users", "ls", "--json")
	require.NoError(t, err)
	var decoded []map[string]any
	require.NoError(t, json.Unmarshal([]byte(out), &decoded))
	require.Len(t, decoded, 2)

	api.mx.Lock()
	api.status, api.response = http.StatusInternalServerError, `{"message": "boom"}`
	api.mx.Unlock()
	_, err = infRun(t, "users", "ls")
	require.Error(t, err)
}

func TestInfUsersCreate(t *testing.T) {
	infSkipIfTerminal(t)
	api := &infAPI{response: infUserJSON}
	infSetup(t, api)

	out, err := infRun(t, "users", "create", "alice@example.invalid", "true")
	require.NoError(t, err)
	req := api.last()
	require.Equal(t, http.MethodPost, req.method)
	require.Equal(t, "/api/v1/users", req.uri)
	require.Equal(t, "alice@example.invalid", req.body["email"])
	require.Equal(t, true, req.body["admin"])
	require.Contains(t, out, "email: alice@example.invalid")

	out, err = infRun(t, "users", "create", "bob@example.invalid", "--json")
	require.NoError(t, err)
	require.Contains(t, out, `"email": "alice@example.invalid"`)
	require.Equal(t, "bob@example.invalid", api.last().body["email"])

	api.mx.Lock()
	api.status, api.response = http.StatusConflict, `{"message": "exists"}`
	api.mx.Unlock()
	_, err = infRun(t, "users", "create", "dup@example.invalid")
	require.Error(t, err)
}

func TestInfUsersUpdate(t *testing.T) {
	api := &infAPI{response: infUserJSON}
	infSetup(t, api)

	out, err := infRun(t, "users", "update", "alice", "--email", "new@example.invalid",
		"--password", "new-password-0000", "--promote")
	require.NoError(t, err)
	req := api.last()
	require.Equal(t, http.MethodPatch, req.method)
	require.Equal(t, "/api/v1/users/alice", req.uri)
	require.Equal(t, "new@example.invalid", req.body["email"])
	require.Equal(t, "new-password-0000", req.body["password"])
	require.Equal(t, true, req.body["admin"])
	require.Contains(t, out, "email: alice@example.invalid")

	_, err = infRun(t, "users", "update", "alice", "--demote")
	require.NoError(t, err)
	require.Equal(t, false, api.last().body["admin"])

	out, err = infRun(t, "users", "update", "alice", "--password-gen", "--json")
	require.NoError(t, err)
	require.Contains(t, out, "generated temporary password: ")
	generated, ok := api.last().body["password"].(string)
	require.True(t, ok)
	require.Len(t, generated, 8)
	require.Contains(t, out, generated)

	api.mx.Lock()
	api.status, api.response = http.StatusForbidden, `{"message": "no"}`
	api.mx.Unlock()
	_, err = infRun(t, "users", "update", "alice")
	require.Error(t, err)
}

func TestInfUsersDelete(t *testing.T) {
	api := &infAPI{status: http.StatusNoContent}
	infSetup(t, api)

	_, err := infRun(t, "users", "delete", "alice")
	require.NoError(t, err)
	require.Equal(t, http.MethodDelete, api.last().method)
	require.Equal(t, "/api/v1/users/alice", api.last().uri)

	api.mx.Lock()
	api.status, api.response = http.StatusNotFound, `{"message": "not found"}`
	api.mx.Unlock()
	_, err = infRun(t, "users", "delete", "alice")
	require.Error(t, err)
}

func TestInfUsersRegistration(t *testing.T) {
	app := kingpin.New("test", "")
	Register(app)
	cmd := app.GetCommand("users")
	require.NotNil(t, cmd)
	for _, name := range []string{"find", "ls", "create", "update", "delete"} {
		require.NotNil(t, cmd.GetCommand(name), name)
	}
}

// infSkipIfTerminal skips tests that would block on reading a password from an interactive terminal.
func infSkipIfTerminal(t *testing.T) {
	t.Helper()
	if term.IsTerminal(syscall.Stdin) {
		t.Skip("stdin is an interactive terminal")
	}
}
