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

package webhook

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	webhookctrl "github.com/harness/gitness/app/api/controller/webhook"
	"github.com/harness/gitness/app/api/request"
	checkevents "github.com/harness/gitness/app/events/check"
	gitevents "github.com/harness/gitness/app/events/git"
	mergequeueevents "github.com/harness/gitness/app/events/mergequeue"
	pullreqevents "github.com/harness/gitness/app/events/pullreq"
	"github.com/harness/gitness/app/services/webhook"
	"github.com/harness/gitness/app/sse"
	"github.com/harness/gitness/app/store/database"
	"github.com/harness/gitness/audit"
	"github.com/harness/gitness/encrypt"
	"github.com/harness/gitness/events"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"
)

type ctlSSE struct{}

func (ctlSSE) Publish(context.Context, int64, enum.SSEType, any) {}

func (ctlSSE) Stream(context.Context, int64) (<-chan *sse.Event, <-chan error, func(context.Context) error) {
	return nil, nil, nil
}

type ctlURLProvider struct{}

func (ctlURLProvider) GetWebhookURL(context.Context, *types.WebhookCore) (string, error) {
	return "http://localhost/internal", nil
}

type ctlEnv struct {
	*ctlBase
	ctrl       *webhookctrl.Controller
	svc        *webhook.Service
	encrypter  encrypt.Encrypter
	executions *database.WebhookExecutionStore
	space      *types.Space
	repo       *types.Repository
}

func ctlNewEnv(t *testing.T) *ctlEnv {
	t.Helper()
	b := ctlNewBase(t)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	system, err := events.ProvideSystem(
		events.Config{Mode: events.ModeInMemory, Namespace: "ctl", MaxStreamLength: 100},
		nil, events.NewNoopCollector())
	require.NoError(t, err)
	gitFactory, err := gitevents.ProvideReaderFactory(system)
	require.NoError(t, err)
	prFactory, err := pullreqevents.ProvideReaderFactory(system)
	require.NoError(t, err)
	mqFactory, err := mergequeueevents.ProvideReaderFactory(system)
	require.NoError(t, err)
	checkFactory, err := checkevents.ProvideReaderFactory(system)
	require.NoError(t, err)

	encrypter, err := encrypt.New(strings.Repeat("k", 32), false)
	require.NoError(t, err)

	webhookStore := database.NewWebhookStore(b.db)
	executionStore := database.NewWebhookExecutionStore(b.db)

	svc, err := webhook.NewService(
		ctx,
		webhook.Config{
			UserAgentIdentity: "ctl-test", EventReaderName: "ctl-reader", Concurrency: 1, AllowLoopback: true,
		},
		b.tx, gitFactory, prFactory, mqFactory, checkFactory,
		webhookStore, executionStore, b.spaces, b.repos,
		nil, nil, nil, b.principals, nil, encrypter, nil, ctlURLProvider{}, nil,
		audit.New(), ctlSSE{}, nil, b.spacePaths, nil,
	)
	require.NoError(t, err)

	space := b.createSpace(t, "root", 0)
	repo := b.createRepo(t, space, "repo1")

	return &ctlEnv{
		ctlBase: b,
		ctrl: webhookctrl.NewController(
			ctlAuthorizer{mode: ctlAllow}, b.spaceFinder, b.repoFinder, svc, encrypter, webhookctrl.NoopPreprocessor{}),
		svc:        svc,
		encrypter:  encrypter,
		executions: executionStore,
		space:      space,
		repo:       repo,
	}
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

func ctlHookBody(identifier, url string) string {
	return `{"identifier":"` + identifier + `","url":"` + url + `","enabled":true,"triggers":["branch_created"]}`
}

func TestCtlWebhookHandlers_RepoAndSpace(t *testing.T) {
	e := ctlNewEnv(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	scopes := []struct {
		name      string
		refParam  string
		ref       string
		create    http.HandlerFunc
		find      http.HandlerFunc
		list      http.HandlerFunc
		update    http.HandlerFunc
		del       http.HandlerFunc
		listExec  http.HandlerFunc
		findExec  http.HandlerFunc
		retrigger http.HandlerFunc
		parent    enum.WebhookParent
	}{
		{
			name: "repo", refParam: request.PathParamRepoRef, ref: "root/repo1",
			create: HandleCreateRepo(e.ctrl), find: HandleFindRepo(e.ctrl), list: HandleListRepo(e.ctrl),
			update: HandleUpdateRepo(e.ctrl), del: HandleDeleteRepo(e.ctrl),
			listExec: HandleListExecutionsRepo(e.ctrl), findExec: HandleFindExecutionRepo(e.ctrl),
			retrigger: HandleRetriggerExecutionRepo(e.ctrl),
		},
		{
			name: "space", refParam: request.PathParamSpaceRef, ref: "root",
			create: HandleCreateSpace(e.ctrl), find: HandleFindSpace(e.ctrl), list: HandleListSpace(e.ctrl),
			update: HandleUpdateSpace(e.ctrl), del: HandleDeleteSpace(e.ctrl),
			listExec: HandleListExecutionsSpace(e.ctrl), findExec: HandleFindExecutionSpace(e.ctrl),
			retrigger: HandleRetriggerExecutionSpace(e.ctrl),
		},
	}

	for _, sc := range scopes {
		t.Run(sc.name, func(t *testing.T) {
			ref := map[string]string{sc.refParam: sc.ref}
			withHook := map[string]string{sc.refParam: sc.ref, request.PathParamWebhookIdentifier: "hook"}

			// Create: bad bodies and a valid one.
			w := ctlServe(t, e.ctlBase, sc.create, ctlRequest{method: http.MethodPost, params: ref, body: "{"})
			require.Equal(t, http.StatusBadRequest, w.Code)
			w = ctlServe(t, e.ctlBase, sc.create, ctlRequest{
				method: http.MethodPost, params: ref, body: ctlHookBody("bad", "not a url"),
			})
			require.Equal(t, http.StatusBadRequest, w.Code)
			w = ctlServe(t, e.ctlBase, sc.create, ctlRequest{
				method: http.MethodPost, params: ref, body: ctlHookBody("hook", srv.URL),
			})
			require.Equal(t, http.StatusCreated, w.Code, w.Body.String())

			var created types.Webhook
			require.NoError(t, json.NewDecoder(w.Body).Decode(&created))
			require.Equal(t, "hook", created.Identifier)

			// Missing path parameters.
			w = ctlServe(t, e.ctlBase, sc.create, ctlRequest{method: http.MethodPost, body: ctlHookBody("x", srv.URL)})
			require.Equal(t, http.StatusBadRequest, w.Code)

			// Find, list, update.
			w = ctlServe(t, e.ctlBase, sc.find, ctlRequest{params: withHook})
			require.Equal(t, http.StatusOK, w.Code)
			w = ctlServe(t, e.ctlBase, sc.find, ctlRequest{params: ref})
			require.Equal(t, http.StatusBadRequest, w.Code)
			w = ctlServe(t, e.ctlBase, sc.find, ctlRequest{
				params: map[string]string{sc.refParam: sc.ref, request.PathParamWebhookIdentifier: "missing"},
			})
			require.GreaterOrEqual(t, w.Code, 400)

			w = ctlServe(t, e.ctlBase, sc.list, ctlRequest{params: ref, query: "page=1&limit=10"})
			require.Equal(t, http.StatusOK, w.Code)
			require.Equal(t, "1", w.Header().Get("X-Total"))
			w = ctlServe(t, e.ctlBase, sc.list, ctlRequest{})
			require.Equal(t, http.StatusBadRequest, w.Code)

			w = ctlServe(t, e.ctlBase, sc.update, ctlRequest{
				method: http.MethodPatch, params: withHook, body: `{"description":"changed"}`,
			})
			require.Equal(t, http.StatusOK, w.Code, w.Body.String())
			w = ctlServe(t, e.ctlBase, sc.update, ctlRequest{method: http.MethodPatch, params: withHook, body: "{"})
			require.Equal(t, http.StatusBadRequest, w.Code)
			w = ctlServe(t, e.ctlBase, sc.update, ctlRequest{method: http.MethodPatch, params: ref, body: "{}"})
			require.Equal(t, http.StatusBadRequest, w.Code)

			// Executions.
			w = ctlServe(t, e.ctlBase, sc.listExec, ctlRequest{params: withHook})
			require.Equal(t, http.StatusOK, w.Code)
			w = ctlServe(t, e.ctlBase, sc.listExec, ctlRequest{params: ref})
			require.Equal(t, http.StatusBadRequest, w.Code)

			exec := &types.WebhookExecution{
				WebhookID: created.ID, TriggerType: enum.WebhookTriggerBranchCreated, TriggerID: "t-" + sc.name,
				Result: enum.WebhookExecutionResultSuccess, Retriggerable: true,
				Request: types.WebhookExecutionRequest{URL: srv.URL, Body: "{}"},
			}
			require.NoError(t, e.executions.Create(t.Context(), exec))
			execParams := map[string]string{
				sc.refParam: sc.ref, request.PathParamWebhookIdentifier: "hook",
				request.PathParamWebhookExecutionID: "1",
			}
			execParams[request.PathParamWebhookExecutionID] = jsonInt(exec.ID)

			w = ctlServe(t, e.ctlBase, sc.findExec, ctlRequest{params: execParams})
			require.Equal(t, http.StatusOK, w.Code)
			w = ctlServe(t, e.ctlBase, sc.findExec, ctlRequest{params: withHook})
			require.Equal(t, http.StatusBadRequest, w.Code)
			w = ctlServe(t, e.ctlBase, sc.findExec, ctlRequest{params: ref})
			require.Equal(t, http.StatusBadRequest, w.Code)

			w = ctlServe(t, e.ctlBase, sc.retrigger, ctlRequest{method: http.MethodPost, params: execParams})
			require.Equal(t, http.StatusOK, w.Code, w.Body.String())
			w = ctlServe(t, e.ctlBase, sc.retrigger, ctlRequest{method: http.MethodPost, params: withHook})
			require.Equal(t, http.StatusBadRequest, w.Code)
			w = ctlServe(t, e.ctlBase, sc.retrigger, ctlRequest{method: http.MethodPost, params: ref})
			require.Equal(t, http.StatusBadRequest, w.Code)

			// Delete.
			w = ctlServe(t, e.ctlBase, sc.del, ctlRequest{method: http.MethodDelete, params: ref})
			require.Equal(t, http.StatusBadRequest, w.Code)
			w = ctlServe(t, e.ctlBase, sc.del, ctlRequest{method: http.MethodDelete, params: withHook})
			require.Equal(t, http.StatusNoContent, w.Code)
			w = ctlServe(t, e.ctlBase, sc.del, ctlRequest{method: http.MethodDelete, params: withHook})
			require.GreaterOrEqual(t, w.Code, 400)
		})
	}
}

func jsonInt(v int64) string {
	data, _ := json.Marshal(v)
	return string(data)
}

func TestCtlWebhookHandlers_Forbidden(t *testing.T) {
	e := ctlNewEnv(t)
	deny := webhookctrl.NewController(
		ctlAuthorizer{mode: ctlDeny}, e.spaceFinder, e.repoFinder, e.svc, e.encrypter, webhookctrl.NoopPreprocessor{})

	repoParams := map[string]string{
		request.PathParamRepoRef: "root/repo1", request.PathParamWebhookIdentifier: "hook",
	}
	spaceParams := map[string]string{
		request.PathParamSpaceRef: "root", request.PathParamWebhookIdentifier: "hook",
	}

	for name, tc := range map[string]struct {
		h      http.HandlerFunc
		method string
		params map[string]string
	}{
		"find repo":   {HandleFindRepo(deny), http.MethodGet, repoParams},
		"list repo":   {HandleListRepo(deny), http.MethodGet, repoParams},
		"delete repo": {HandleDeleteRepo(deny), http.MethodDelete, repoParams},
		"find space":  {HandleFindSpace(deny), http.MethodGet, spaceParams},
		"list space":  {HandleListSpace(deny), http.MethodGet, spaceParams},
	} {
		w := ctlServe(t, e.ctlBase, tc.h, ctlRequest{method: tc.method, params: tc.params})
		require.Equal(t, http.StatusForbidden, w.Code, name)
	}
}
