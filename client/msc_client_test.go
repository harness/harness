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

package client

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/harness/gitness/app/api/controller/user"
	"github.com/harness/gitness/types"

	"github.com/stretchr/testify/require"
)

type mscAPIRequest struct {
	method string
	path   string
	auth   string
	body   map[string]any
}

type mscAPIServer struct {
	mx       sync.Mutex
	requests []mscAPIRequest
	status   int
	body     string
}

func (s *mscAPIServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	raw, _ := io.ReadAll(r.Body)
	var body map[string]any
	_ = json.Unmarshal(raw, &body)

	s.mx.Lock()
	s.requests = append(s.requests, mscAPIRequest{
		method: r.Method, path: r.URL.Path, auth: r.Header.Get("Authorization"), body: body,
	})
	status, response := s.status, s.body
	s.mx.Unlock()

	w.Header().Set("Content-Type", "application/json")
	if status == 0 {
		status = http.StatusOK
	}
	w.WriteHeader(status)
	_, _ = io.WriteString(w, response)
}

func (s *mscAPIServer) last() mscAPIRequest {
	s.mx.Lock()
	defer s.mx.Unlock()
	return s.requests[len(s.requests)-1]
}

func mscNewTestClient(t *testing.T, srv *mscAPIServer, token string) *HTTPClient {
	t.Helper()
	server := httptest.NewServer(srv)
	t.Cleanup(server.Close)

	c := NewToken(server.URL, token)
	c.SetClient(server.Client())
	return c
}

const (
	mscTokenJSON  = `{"access_token":"pat-1111","token_type":"bearer"}`
	mscUserJSON   = `{"id":1,"uid":"alice","email":"alice@test.invalid","admin":false}`
	mscUsersJSON  = `[{"id":1,"uid":"alice","email":"alice@test.invalid","admin":false}]`
	mscErrMessage = `{"message":"not found"}`
)

func TestMscHTTPClient_UserList(t *testing.T) {
	api := &mscAPIServer{body: mscUsersJSON}
	c := mscNewTestClient(t, api, "pat-0000")

	users, err := c.UserList(context.Background(), types.UserFilter{Page: 1, Size: 10})
	require.NoError(t, err)
	require.Len(t, users, 1)
	require.Equal(t, http.MethodGet, api.last().method)
}

func TestMscHTTPClient_Self(t *testing.T) {
	api := &mscAPIServer{body: mscUserJSON}
	c := mscNewTestClient(t, api, "pat-0000")

	out, err := c.Self(context.Background())
	require.NoError(t, err)
	require.Equal(t, "alice", out.UID)
	require.Equal(t, "Bearer pat-0000", api.last().auth)
}

func TestMscHTTPClient_UserCRUD(t *testing.T) {
	api := &mscAPIServer{body: mscUserJSON}
	c := mscNewTestClient(t, api, "pat-0000")

	_, err := c.User(context.Background(), "alice")
	require.NoError(t, err)
	require.Equal(t, http.MethodGet, api.last().method)

	_, err = c.UserCreate(context.Background(), &types.User{UID: "bob", Email: "bob@test.invalid"})
	require.NoError(t, err)
	require.Equal(t, http.MethodPost, api.last().method)

	admin := true
	_, err = c.UserUpdate(context.Background(), "bob", &types.UserInput{Admin: &admin})
	require.NoError(t, err)
	require.Equal(t, http.MethodPatch, api.last().method)

	require.NoError(t, c.UserDelete(context.Background(), "bob"))
	require.Equal(t, http.MethodDelete, api.last().method)
}

func TestMscHTTPClient_AuthEndpoints(t *testing.T) {
	api := &mscAPIServer{body: mscTokenJSON}
	c := mscNewTestClient(t, api, "should-not-send")

	_, err := c.Login(context.Background(), &user.LoginInput{LoginIdentifier: "alice", Password: "secret"})
	require.NoError(t, err)
	require.Empty(t, api.last().auth)

	_, err = c.Register(context.Background(), &user.RegisterInput{Email: "bob@test.invalid", Password: "secret"})
	require.NoError(t, err)
	require.Empty(t, api.last().auth)

	c.token = "pat-0000"
	_, err = c.UserCreatePAT(context.Background(), user.CreateTokenInput{Identifier: "cli"})
	require.NoError(t, err)
	require.Equal(t, "Bearer pat-0000", api.last().auth)
}

func TestMscHTTPClient_RemoteError(t *testing.T) {
	api := &mscAPIServer{status: http.StatusNotFound, body: mscErrMessage}
	c := mscNewTestClient(t, api, "pat-0000")

	_, err := c.Self(context.Background())
	require.Error(t, err)
	var remote *remoteError
	require.ErrorAs(t, err, &remote)
	require.Equal(t, "not found", remote.Error())
}

func TestMscHTTPClient_Configuration(t *testing.T) {
	c := New("http://example.invalid")
	require.Equal(t, "http://example.invalid", c.base)

	c.SetDebug(true)
	require.True(t, c.debug)

	custom := &http.Client{}
	c.SetClient(custom)
	require.Same(t, custom, c.client)
}

func TestMscRemoteError_String(t *testing.T) {
	err := &remoteError{Message: "bad request"}
	require.Equal(t, "bad request", err.Error())
	require.Contains(t, strings.ToLower(err.Error()), "bad")
}
