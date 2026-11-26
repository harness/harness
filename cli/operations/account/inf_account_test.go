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

package account

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
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

const infTokenJSON = `{"access_token": "login-token-0000", "token": {"expires_at": 4102444800000}}`

func infSessionFile(t *testing.T) string {
	t.Helper()
	path, err := xdg.ConfigFile(filepath.Join("app", "config.json"))
	require.NoError(t, err)
	return path
}

func infRunAccount(t *testing.T, args ...string) error {
	t.Helper()
	app := kingpin.New("test", "")
	app.Terminate(nil)
	RegisterLogin(app)
	RegisterLogout(app)
	RegisterRegister(app)
	var err error
	infCaptureStdout(t, func() {
		_, err = app.Parse(args)
	})
	return err
}

func TestInfLogin(t *testing.T) {
	infSkipIfTerminal(t)
	api := &infAPI{response: infTokenJSON}
	infSetup(t, api)
	srv := httptest.NewServer(api)
	defer srv.Close()

	err := infRunAccount(t, "login", srv.URL)
	require.NoError(t, err)
	require.Equal(t, http.MethodPost, api.last().method)
	require.Equal(t, "/api/v1/login", api.last().uri)
	require.Empty(t, api.last().auth, "login does not send a token")

	data, err := os.ReadFile(infSessionFile(t))
	require.NoError(t, err)
	var stored map[string]any
	require.NoError(t, json.Unmarshal(data, &stored))
	require.Equal(t, srv.URL, stored["uri"])
	require.Equal(t, "login-token-0000", stored["access_token"])
	require.EqualValues(t, 4102444800000, stored["expires_at"])
}

func TestInfLogin_ServerRejects(t *testing.T) {
	infSkipIfTerminal(t)
	api := &infAPI{status: http.StatusUnauthorized, response: `{"message": "invalid credentials"}`}
	infSetup(t, api)
	srv := httptest.NewServer(api)
	defer srv.Close()

	err := infRunAccount(t, "login", srv.URL)
	require.Error(t, err)
}

func TestInfRegister(t *testing.T) {
	infSkipIfTerminal(t)
	api := &infAPI{response: infTokenJSON}
	infSetup(t, api)
	srv := httptest.NewServer(api)
	defer srv.Close()

	err := infRunAccount(t, "register", srv.URL)
	require.NoError(t, err)
	require.Equal(t, "/api/v1/register", api.last().uri)

	data, err := os.ReadFile(infSessionFile(t))
	require.NoError(t, err)
	require.Contains(t, string(data), "login-token-0000")

	api.mx.Lock()
	api.status, api.response = http.StatusConflict, `{"message": "exists"}`
	api.mx.Unlock()
	err = infRunAccount(t, "register", srv.URL)
	require.Error(t, err)
}

func TestInfLogout(t *testing.T) {
	infSetup(t, &infAPI{})
	path := infSessionFile(t)
	require.FileExists(t, path)

	err := infRunAccount(t, "logout")
	require.NoError(t, err)
	require.NoFileExists(t, path)
}

// infSkipIfTerminal skips tests that would block on reading a password from an interactive terminal.
func infSkipIfTerminal(t *testing.T) {
	t.Helper()
	if term.IsTerminal(syscall.Stdin) {
		t.Skip("stdin is an interactive terminal")
	}
}
