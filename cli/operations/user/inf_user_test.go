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

package user

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/adrg/xdg"
	"github.com/stretchr/testify/require"
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

const infUserJSON = `{"id": 7, "uid": "alice", "display_name": "Alice", ` +
	`"email": "alice@example.invalid", "admin": true}`

func TestInfUserSelf(t *testing.T) {
	api := &infAPI{response: infUserJSON}
	infSetup(t, api)

	out, err := infRun(t, "user", "self")
	require.NoError(t, err)
	require.Contains(t, out, "uid:   alice")
	require.Contains(t, out, "name:  Alice")
	require.Contains(t, out, "email: alice@example.invalid")
	require.Equal(t, "/api/v1/user", api.last().uri)
	require.Equal(t, "Bearer session-token-0000", api.last().auth)

	out, err = infRun(t, "user", "self", "--json")
	require.NoError(t, err)
	require.Contains(t, out, `"uid": "alice"`)

	api.mx.Lock()
	api.status, api.response = http.StatusUnauthorized, `{"message": "unauthorized"}`
	api.mx.Unlock()
	_, err = infRun(t, "user", "self")
	require.Error(t, err)
}

func TestInfUserCreatePAT(t *testing.T) {
	api := &infAPI{response: `{"access_token": "pat-secret-0000", "token": {"principal_id": 7, "identifier": "my-pat"}}`}
	infSetup(t, api)

	out, err := infRun(t, "user", "pat", "my-pat", "3600")
	require.NoError(t, err)
	req := api.last()
	require.Equal(t, http.MethodPost, req.method)
	require.Equal(t, "/api/v1/user/tokens", req.uri)
	require.Equal(t, "my-pat", req.body["identifier"])
	require.EqualValues(t, 3600*time.Second, req.body["lifetime"])
	require.Contains(t, out, "token:       pat-secret-0000")
	require.Contains(t, out, "principalID: 7")

	// no lifetime means the token never expires.
	out, err = infRun(t, "user", "pat", "forever", "--json")
	require.NoError(t, err)
	require.Nil(t, api.last().body["lifetime"])
	require.Contains(t, out, `"access_token": "pat-secret-0000"`)

	api.mx.Lock()
	api.status, api.response = http.StatusBadRequest, `{"message": "bad"}`
	api.mx.Unlock()
	_, err = infRun(t, "user", "pat", "x")
	require.Error(t, err)
}

func TestInfUserRegistration(t *testing.T) {
	app := kingpin.New("test", "")
	Register(app)
	cmd := app.GetCommand("user")
	require.NotNil(t, cmd)
	require.NotNil(t, cmd.GetCommand("self"))
	require.NotNil(t, cmd.GetCommand("pat"))
}
