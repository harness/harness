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
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	userctrl "github.com/harness/gitness/app/api/controller/user"
	"github.com/harness/gitness/app/api/request"
	userevents "github.com/harness/gitness/app/events/user"
	"github.com/harness/gitness/app/store/database"
	"github.com/harness/gitness/events"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/check"
	"github.com/harness/gitness/types/enum"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"
)

type ctlEnv struct {
	*ctlBase
	ctrl *userctrl.Controller
	repo *types.Repository
}

func ctlNewEnv(t *testing.T) *ctlEnv {
	t.Helper()
	b := ctlNewBase(t)

	sys, err := events.ProvideSystem(
		events.Config{Mode: events.ModeInMemory, Namespace: "ctl", MaxStreamLength: 100},
		nil, events.NewNoopCollector())
	require.NoError(t, err)
	reporter, err := userevents.NewReporter(sys)
	require.NoError(t, err)

	ctrl := userctrl.NewController(
		b.tx, check.PrincipalUIDDefault, ctlAuthorizer{mode: ctlAllow}, b.principals, database.NewTokenStore(b.db),
		b.memberships, database.NewPublicKeyStore(b.db), database.NewPublicKeySubKeyStore(b.db),
		database.NewGitSignatureResultStore(b.db), reporter, b.repoFinder, database.NewFavoriteStore(b.db),
	)

	space := b.createSpace(t, "root", 0)
	return &ctlEnv{ctlBase: b, ctrl: ctrl, repo: b.createRepo(t, space, "repo1")}
}

// ctlRequest describes a request sent to a handler.
type ctlRequest struct {
	method string
	params map[string]string
	query  string
	body   string
}

// ctlServe sends the request to the handler with the session of the admin user and returns the response.
func ctlServe(t *testing.T, b *ctlBase, h http.Handler, req ctlRequest) *httptest.ResponseRecorder {
	t.Helper()

	method := req.method
	if method == "" {
		method = http.MethodGet
	}
	target := "/"
	if req.query != "" {
		target += "?" + req.query
	}

	r := httptest.NewRequestWithContext(t.Context(), method, target, strings.NewReader(req.body))
	routeCtx := chi.NewRouteContext()
	for k, v := range req.params {
		routeCtx.URLParams.Add(k, v)
	}
	ctx := context.WithValue(r.Context(), chi.RouteCtxKey, routeCtx)
	ctx = request.WithAuthSession(ctx, b.session())
	r = r.WithContext(ctx)

	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

func ctlSSHKey(t *testing.T) string {
	t.Helper()
	pub, _, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	sshPub, err := ssh.NewPublicKey(pub)
	require.NoError(t, err)
	return string(ssh.MarshalAuthorizedKey(sshPub))
}

func ctlJSON(t *testing.T, v any) string {
	t.Helper()
	data, err := json.Marshal(v)
	require.NoError(t, err)
	return string(data)
}

func TestCtlUserHandlers_Profile(t *testing.T) {
	e := ctlNewEnv(t)

	w := ctlServe(t, e.ctlBase, HandleFind(e.ctrl), ctlRequest{})
	require.Equal(t, http.StatusOK, w.Code)
	var found types.User
	require.NoError(t, json.NewDecoder(w.Body).Decode(&found))
	require.Equal(t, e.admin.UID, found.UID)

	w = ctlServe(t, e.ctlBase, HandleUpdate(e.ctrl), ctlRequest{
		method: http.MethodPatch, body: `{"display_name":"New Name"}`,
	})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	w = ctlServe(t, e.ctlBase, HandleUpdate(e.ctrl), ctlRequest{method: http.MethodPatch, body: "{"})
	require.Equal(t, http.StatusBadRequest, w.Code)
	w = ctlServe(t, e.ctlBase, HandleUpdate(e.ctrl), ctlRequest{method: http.MethodPatch, body: `{"email":""}`})
	require.Equal(t, http.StatusBadRequest, w.Code)

	other := e.createUser(t, "other", false)
	otherParams := map[string]string{request.PathParamUserUID: other.UID}
	w = ctlServe(t, e.ctlBase, HandleUpdateAdmin(e.ctrl), ctlRequest{
		method: http.MethodPatch, params: otherParams, body: `{"admin":true}`,
	})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	w = ctlServe(t, e.ctlBase, HandleUpdateAdmin(e.ctrl), ctlRequest{method: http.MethodPatch, body: `{"admin":true}`})
	require.Equal(t, http.StatusBadRequest, w.Code)
	w = ctlServe(t, e.ctlBase, HandleUpdateAdmin(e.ctrl), ctlRequest{
		method: http.MethodPatch, params: otherParams, body: "{",
	})
	require.Equal(t, http.StatusBadRequest, w.Code)
	w = ctlServe(t, e.ctlBase, HandleUpdateAdmin(e.ctrl), ctlRequest{
		method: http.MethodPatch, params: map[string]string{request.PathParamUserUID: "ghost"}, body: `{}`,
	})
	require.Equal(t, http.StatusNotFound, w.Code)

	w = ctlServe(t, e.ctlBase, HandleMembershipSpaces(e.ctrl), ctlRequest{query: "page=1&limit=10"})
	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "0", w.Header().Get("X-Total"))
}

func TestCtlUserHandlers_Tokens(t *testing.T) {
	e := ctlNewEnv(t)

	w := ctlServe(t, e.ctlBase, HandleCreateAccessToken(e.ctrl), ctlRequest{
		method: http.MethodPost, body: `{"identifier":"my-token"}`,
	})
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	w = ctlServe(t, e.ctlBase, HandleCreateAccessToken(e.ctrl), ctlRequest{method: http.MethodPost, body: "{"})
	require.Equal(t, http.StatusBadRequest, w.Code)
	w = ctlServe(t, e.ctlBase, HandleCreateAccessToken(e.ctrl), ctlRequest{method: http.MethodPost, body: `{}`})
	require.Equal(t, http.StatusBadRequest, w.Code)

	w = ctlServe(t, e.ctlBase, HandleListTokens(e.ctrl, enum.TokenTypePAT), ctlRequest{})
	require.Equal(t, http.StatusOK, w.Code)
	var tokens []types.Token
	require.NoError(t, json.NewDecoder(w.Body).Decode(&tokens))
	require.Len(t, tokens, 1)
	w = ctlServe(t, e.ctlBase, HandleListTokens(e.ctrl, enum.TokenTypeSAT), ctlRequest{})
	require.Equal(t, http.StatusBadRequest, w.Code)

	params := map[string]string{request.PathParamTokenIdentifier: "my-token"}
	w = ctlServe(t, e.ctlBase, HandleDeleteToken(e.ctrl, enum.TokenTypePAT), ctlRequest{
		method: http.MethodDelete, params: map[string]string{},
	})
	require.Equal(t, http.StatusBadRequest, w.Code)
	w = ctlServe(t, e.ctlBase, HandleDeleteToken(e.ctrl, enum.TokenTypeSession), ctlRequest{
		method: http.MethodDelete, params: params,
	})
	require.Equal(t, http.StatusNotFound, w.Code)
	w = ctlServe(t, e.ctlBase, HandleDeleteToken(e.ctrl, enum.TokenTypePAT), ctlRequest{
		method: http.MethodDelete, params: params,
	})
	require.Equal(t, http.StatusNoContent, w.Code)
}

func TestCtlUserHandlers_PublicKeys(t *testing.T) {
	e := ctlNewEnv(t)

	w := ctlServe(t, e.ctlBase, HandleCreatePublicKey(e.ctrl), ctlRequest{method: http.MethodPost, body: "{"})
	require.Equal(t, http.StatusBadRequest, w.Code)
	w = ctlServe(t, e.ctlBase, HandleCreatePublicKey(e.ctrl), ctlRequest{
		method: http.MethodPost, body: ctlJSON(t, map[string]string{"identifier": "k", "content": "junk"}),
	})
	require.Equal(t, http.StatusBadRequest, w.Code)
	w = ctlServe(t, e.ctlBase, HandleCreatePublicKey(e.ctrl), ctlRequest{
		method: http.MethodPost, body: ctlJSON(t, map[string]string{"identifier": "laptop", "content": ctlSSHKey(t)}),
	})
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())

	w = ctlServe(t, e.ctlBase, HandleListPublicKeys(e.ctrl), ctlRequest{query: "page=1&limit=10"})
	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "1", w.Header().Get("X-Total"))
	w = ctlServe(t, e.ctlBase, HandleListPublicKeys(e.ctrl), ctlRequest{query: "sort=bogus"})
	require.Equal(t, http.StatusBadRequest, w.Code)

	params := map[string]string{request.PathParamPublicKeyIdentifier: "laptop"}
	w = ctlServe(t, e.ctlBase, HandleUpdatePublicKey(e.ctrl), ctlRequest{
		method: http.MethodPatch, params: params, body: `{"valid_from":1000,"valid_to":2000}`,
	})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	w = ctlServe(t, e.ctlBase, HandleUpdatePublicKey(e.ctrl), ctlRequest{method: http.MethodPatch, body: `{}`})
	require.Equal(t, http.StatusBadRequest, w.Code)
	w = ctlServe(t, e.ctlBase, HandleUpdatePublicKey(e.ctrl), ctlRequest{
		method: http.MethodPatch, params: params, body: "{",
	})
	require.Equal(t, http.StatusBadRequest, w.Code)

	w = ctlServe(t, e.ctlBase, HandleDeletePublicKey(e.ctrl), ctlRequest{method: http.MethodDelete})
	require.Equal(t, http.StatusBadRequest, w.Code)
	w = ctlServe(t, e.ctlBase, HandleDeletePublicKey(e.ctrl), ctlRequest{method: http.MethodDelete, params: params})
	require.Equal(t, http.StatusNoContent, w.Code)
}

func TestCtlUserHandlers_Favorites(t *testing.T) {
	e := ctlNewEnv(t)

	body := ctlJSON(t, types.FavoriteResource{ID: e.repo.ID, Type: enum.ResourceTypeRepo})
	w := ctlServe(t, e.ctlBase, HandleCreateFavorite(e.ctrl), ctlRequest{method: http.MethodPost, body: "{"})
	require.Equal(t, http.StatusBadRequest, w.Code)
	w = ctlServe(t, e.ctlBase, HandleCreateFavorite(e.ctrl), ctlRequest{method: http.MethodPost, body: body})
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	w = ctlServe(t, e.ctlBase, HandleCreateFavorite(e.ctrl), ctlRequest{method: http.MethodPost, body: body})
	require.GreaterOrEqual(t, w.Code, 400)

	id := jsonInt(e.repo.ID)
	params := map[string]string{request.PathParamResourceID: id}
	w = ctlServe(t, e.ctlBase, HandleDeleteFavorite(e.ctrl), ctlRequest{method: http.MethodDelete})
	require.Equal(t, http.StatusBadRequest, w.Code)
	w = ctlServe(t, e.ctlBase, HandleDeleteFavorite(e.ctrl), ctlRequest{
		method: http.MethodDelete, params: params, query: "resource_type=bogus",
	})
	require.Equal(t, http.StatusBadRequest, w.Code)
	w = ctlServe(t, e.ctlBase, HandleDeleteFavorite(e.ctrl), ctlRequest{
		method: http.MethodDelete, params: params, query: "resource_type=" + string(enum.ResourceTypeRepo),
	})
	require.Equal(t, http.StatusNoContent, w.Code)
}

func jsonInt(v int64) string {
	data, _ := json.Marshal(v)
	return string(data)
}
