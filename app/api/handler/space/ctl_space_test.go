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

package space

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/harness/gitness/app/api/request"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"
)

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

type ctlStep struct {
	name    string
	handler http.HandlerFunc
	req     ctlRequest
	want    int
}

func ctlRunSteps(t *testing.T, e *ctlEnv, steps []ctlStep) {
	t.Helper()
	for _, step := range steps {
		w := ctlServe(t, e.ctlBase, step.handler, step.req)
		require.Equal(t, step.want, w.Code, "%s: %s", step.name, w.Body.String())
	}
}

func ctlSpaceParams(extra ...string) map[string]string {
	params := map[string]string{request.PathParamSpaceRef: "root"}
	for i := 0; i+1 < len(extra); i += 2 {
		params[extra[i]] = extra[i+1]
	}
	return params
}

func ctlJSONBody(t *testing.T, v any) string {
	t.Helper()
	data, err := json.Marshal(v)
	require.NoError(t, err)
	return string(data)
}

func TestCtlSpaceHandlers_Basics(t *testing.T) {
	e := ctlNewEnv(t)
	c := e.ctrl
	e.createRepo(t, e.root, "repo1")
	e.createUser(t, "member", false)

	steps := []ctlStep{
		{"create bad body", HandleCreate(c), ctlRequest{method: http.MethodPost, body: "{"}, http.StatusBadRequest},
		{"create invalid", HandleCreate(c), ctlRequest{method: http.MethodPost, body: `{"identifier":"api"}`},
			http.StatusBadRequest},
		{"create", HandleCreate(c), ctlRequest{method: http.MethodPost, body: `{"identifier":"top"}`},
			http.StatusCreated},
		{"create child", HandleCreate(c), ctlRequest{
			method: http.MethodPost, body: `{"parent_ref":"root","identifier":"child"}`,
		}, http.StatusCreated},

		{"find", HandleFind(c), ctlRequest{params: ctlSpaceParams()}, http.StatusOK},
		{"find no ref", HandleFind(c), ctlRequest{}, http.StatusBadRequest},
		{"find unknown", HandleFind(c), ctlRequest{params: map[string]string{request.PathParamSpaceRef: "nope"}},
			http.StatusNotFound},

		{"update", HandleUpdate(c), ctlRequest{
			method: http.MethodPatch, params: ctlSpaceParams(), body: `{"description":"d"}`,
		}, http.StatusOK},
		{"update bad body", HandleUpdate(c), ctlRequest{method: http.MethodPatch, params: ctlSpaceParams(), body: "{"},
			http.StatusBadRequest},
		{"update no ref", HandleUpdate(c), ctlRequest{method: http.MethodPatch, body: "{}"}, http.StatusBadRequest},

		{"public access", HandleUpdatePublicAccess(c), ctlRequest{
			method: http.MethodPost, params: ctlSpaceParams(), body: `{"is_public":true}`,
		}, http.StatusOK},
		{"public access bad body", HandleUpdatePublicAccess(c), ctlRequest{
			method: http.MethodPost, params: ctlSpaceParams(), body: "{",
		}, http.StatusBadRequest},
		{"public access no ref", HandleUpdatePublicAccess(c), ctlRequest{method: http.MethodPost, body: "{}"},
			http.StatusBadRequest},

		{"move", HandleMove(c), ctlRequest{
			method: http.MethodPost, params: map[string]string{request.PathParamSpaceRef: "root/child"}, body: `{}`,
		}, http.StatusOK},
		{"move bad body", HandleMove(c), ctlRequest{method: http.MethodPost, params: ctlSpaceParams(), body: "{"},
			http.StatusBadRequest},
		{"move no ref", HandleMove(c), ctlRequest{method: http.MethodPost, body: "{}"}, http.StatusBadRequest},

		{"list spaces", HandleListSpaces(c), ctlRequest{params: ctlSpaceParams(), query: "page=1&limit=10"},
			http.StatusOK},
		{"list spaces no ref", HandleListSpaces(c), ctlRequest{}, http.StatusBadRequest},
		{"list repos", HandleListRepos(c), ctlRequest{params: ctlSpaceParams(), query: "page=1&limit=10"},
			http.StatusOK},
		{"list repos no ref", HandleListRepos(c), ctlRequest{}, http.StatusBadRequest},
		{"list pipelines", HandleListPipelines(c), ctlRequest{params: ctlSpaceParams()}, http.StatusOK},
		{"list pipelines no ref", HandleListPipelines(c), ctlRequest{}, http.StatusBadRequest},
		{"list executions", HandleListExecutions(c), ctlRequest{params: ctlSpaceParams()}, http.StatusOK},
		{"list executions no ref", HandleListExecutions(c), ctlRequest{}, http.StatusBadRequest},
		{"list secrets", HandleListSecrets(c), ctlRequest{params: ctlSpaceParams()}, http.StatusOK},
		{"list secrets no ref", HandleListSecrets(c), ctlRequest{}, http.StatusBadRequest},
		{"list templates", HandleListTemplates(c), ctlRequest{params: ctlSpaceParams()}, http.StatusOK},
		{"list templates no ref", HandleListTemplates(c), ctlRequest{}, http.StatusBadRequest},
		{"list connectors", HandleListConnectors(c), ctlRequest{params: ctlSpaceParams()}, http.StatusOK},
		{"list connectors no ref", HandleListConnectors(c), ctlRequest{}, http.StatusBadRequest},
		{"list service accounts", HandleListServiceAccounts(c), ctlRequest{params: ctlSpaceParams()}, http.StatusOK},
		{"list service accounts bad inherited", HandleListServiceAccounts(c), ctlRequest{
			params: ctlSpaceParams(), query: "inherited=maybe",
		}, http.StatusBadRequest},
		{"list service accounts no ref", HandleListServiceAccounts(c), ctlRequest{}, http.StatusBadRequest},

		{"usage", HandleUsageMetric(c), ctlRequest{params: ctlSpaceParams()}, http.StatusOK},
		{"usage no ref", HandleUsageMetric(c), ctlRequest{}, http.StatusBadRequest},
		{"usage unknown", HandleUsageMetric(c), ctlRequest{
			params: map[string]string{request.PathParamSpaceRef: "nope"},
		}, http.StatusNotFound},

		{"export progress no ref", HandleExportProgress(c), ctlRequest{}, http.StatusBadRequest},
		{"export no ref", HandleExport(c), ctlRequest{method: http.MethodPost, body: "{}"}, http.StatusBadRequest},
		{"export bad body", HandleExport(c), ctlRequest{method: http.MethodPost, params: ctlSpaceParams(), body: "{"},
			http.StatusBadRequest},
		{"export invalid", HandleExport(c), ctlRequest{method: http.MethodPost, params: ctlSpaceParams(), body: "{}"},
			http.StatusBadRequest},
		{"import bad body", HandleImport(c), ctlRequest{method: http.MethodPost, body: "{"}, http.StatusBadRequest},
		{"import repositories no ref", HandleImportRepositories(c), ctlRequest{method: http.MethodPost, body: "{}"},
			http.StatusBadRequest},
		{"import repositories bad body", HandleImportRepositories(c), ctlRequest{
			method: http.MethodPost, params: ctlSpaceParams(), body: "{",
		}, http.StatusBadRequest},

		{"pr count no ref", HandleCountPullReqs(c), ctlRequest{}, http.StatusBadRequest},
		{"pr count bad flag", HandleCountPullReqs(c), ctlRequest{
			params: ctlSpaceParams(), query: "include_subspaces=maybe",
		}, http.StatusBadRequest},
		{"pr list no ref", HandleListPullReqs(c), ctlRequest{}, http.StatusBadRequest},
		{"pr list bad flag", HandleListPullReqs(c), ctlRequest{
			params: ctlSpaceParams(), query: "include_subspaces=maybe",
		}, http.StatusBadRequest},
	}
	ctlRunSteps(t, e, steps)

	// Soft delete, restore and purge of a space.
	ctlRunSteps(t, e, []ctlStep{
		{"soft delete no ref", HandleSoftDelete(c), ctlRequest{method: http.MethodDelete}, http.StatusBadRequest},
		{"soft delete unknown", HandleSoftDelete(c), ctlRequest{
			method: http.MethodDelete, params: map[string]string{request.PathParamSpaceRef: "nope"},
		}, http.StatusNotFound},
	})
	doomed := e.createSpace(t, "doomed", 0)
	w := ctlServe(t, e.ctlBase, HandleSoftDelete(c), ctlRequest{
		method: http.MethodDelete, params: map[string]string{request.PathParamSpaceRef: doomed.Identifier},
	})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var del struct {
		DeletedAt int64 `json:"deleted_at"`
	}
	require.NoError(t, json.NewDecoder(w.Body).Decode(&del))
	deletedAt := jsonInt(del.DeletedAt)
	doomedParams := map[string]string{request.PathParamSpaceRef: doomed.Identifier}

	ctlRunSteps(t, e, []ctlStep{
		{"restore no ref", HandleRestore(c), ctlRequest{method: http.MethodPost, body: "{}"}, http.StatusBadRequest},
		{"restore no deleted_at", HandleRestore(c), ctlRequest{
			method: http.MethodPost, params: doomedParams, body: "{}",
		}, http.StatusBadRequest},
		{"restore bad body", HandleRestore(c), ctlRequest{
			method: http.MethodPost, params: doomedParams, query: "deleted_at=" + deletedAt, body: "{",
		}, http.StatusBadRequest},
		{"restore", HandleRestore(c), ctlRequest{
			method: http.MethodPost, params: doomedParams, query: "deleted_at=" + deletedAt, body: "{}",
		}, http.StatusOK},
		{"purge no ref", HandlePurge(c), ctlRequest{method: http.MethodDelete}, http.StatusBadRequest},
		{"purge no deleted_at", HandlePurge(c), ctlRequest{method: http.MethodDelete, params: doomedParams},
			http.StatusBadRequest},
		{"purge not deleted", HandlePurge(c), ctlRequest{
			method: http.MethodDelete, params: doomedParams, query: "deleted_at=" + deletedAt,
		}, http.StatusNotFound},
	})
}

func jsonInt(v int64) string {
	data, _ := json.Marshal(v)
	return string(data)
}

func TestCtlSpaceHandlers_Memberships(t *testing.T) {
	e := ctlNewEnv(t)
	c := e.ctrl
	e.createUser(t, "member", false)
	userParams := ctlSpaceParams(request.PathParamUserUID, "member")

	ctlRunSteps(t, e, []ctlStep{
		{"add no ref", HandleMembershipAdd(c), ctlRequest{method: http.MethodPost, body: "{}"}, http.StatusBadRequest},
		{"add bad body", HandleMembershipAdd(c), ctlRequest{method: http.MethodPost, params: ctlSpaceParams(), body: "{"},
			http.StatusBadRequest},
		{"add invalid", HandleMembershipAdd(c), ctlRequest{method: http.MethodPost, params: ctlSpaceParams(), body: "{}"},
			http.StatusBadRequest},
		{"add", HandleMembershipAdd(c), ctlRequest{
			method: http.MethodPost, params: ctlSpaceParams(), body: `{"user_uid":"member","role":"reader"}`,
		}, http.StatusCreated},
		{"list", HandleMembershipList(c), ctlRequest{params: ctlSpaceParams(), query: "page=1&limit=10"}, http.StatusOK},
		{"list no ref", HandleMembershipList(c), ctlRequest{}, http.StatusBadRequest},
		{"update no ref", HandleMembershipUpdate(c), ctlRequest{method: http.MethodPatch, body: "{}"},
			http.StatusBadRequest},
		{"update no user", HandleMembershipUpdate(c), ctlRequest{
			method: http.MethodPatch, params: ctlSpaceParams(), body: "{}",
		}, http.StatusBadRequest},
		{"update bad body", HandleMembershipUpdate(c), ctlRequest{method: http.MethodPatch, params: userParams, body: "{"},
			http.StatusBadRequest},
		{"update", HandleMembershipUpdate(c), ctlRequest{
			method: http.MethodPatch, params: userParams, body: `{"role":"contributor"}`,
		}, http.StatusOK},
		{"delete no ref", HandleMembershipDelete(c), ctlRequest{method: http.MethodDelete}, http.StatusBadRequest},
		{"delete no user", HandleMembershipDelete(c), ctlRequest{method: http.MethodDelete, params: ctlSpaceParams()},
			http.StatusBadRequest},
		{"delete", HandleMembershipDelete(c), ctlRequest{method: http.MethodDelete, params: userParams},
			http.StatusNoContent},
	})
}

func TestCtlSpaceHandlers_Labels(t *testing.T) {
	e := ctlNewEnv(t)
	c := e.ctrl
	keyParams := ctlSpaceParams(request.PathParamLabelKey, "priority")
	valueParams := ctlSpaceParams(request.PathParamLabelKey, "priority", request.PathParamLabelValue, "high")
	label := ctlJSONBody(t, types.DefineLabelInput{Key: "priority", Type: enum.LabelTypeStatic, Color: enum.LabelColorRed})

	ctlRunSteps(t, e, []ctlStep{
		{"define no ref", HandleDefineLabel(c), ctlRequest{method: http.MethodPost, body: label}, http.StatusBadRequest},
		{"define bad body", HandleDefineLabel(c), ctlRequest{method: http.MethodPost, params: ctlSpaceParams(), body: "{"},
			http.StatusBadRequest},
		{"define", HandleDefineLabel(c), ctlRequest{method: http.MethodPost, params: ctlSpaceParams(), body: label},
			http.StatusCreated},
		{"list", HandleListLabels(c), ctlRequest{params: ctlSpaceParams(), query: "page=1&limit=10"}, http.StatusOK},
		{"list no ref", HandleListLabels(c), ctlRequest{}, http.StatusBadRequest},
		{"find", HandleFindLabel(c), ctlRequest{params: keyParams}, http.StatusOK},
		{"find with values", HandleFindLabel(c), ctlRequest{params: keyParams, query: "include_values=true"}, http.StatusOK},
		{"find bad flag", HandleFindLabel(c), ctlRequest{params: keyParams, query: "include_values=maybe"},
			http.StatusBadRequest},
		{"find no key", HandleFindLabel(c), ctlRequest{params: ctlSpaceParams()}, http.StatusBadRequest},
		{"find no ref", HandleFindLabel(c), ctlRequest{}, http.StatusBadRequest},
		{"update", HandleUpdateLabel(c), ctlRequest{
			method: http.MethodPatch, params: keyParams, body: `{"description":"d"}`,
		}, http.StatusOK},
		{"update bad body", HandleUpdateLabel(c), ctlRequest{method: http.MethodPatch, params: keyParams, body: "{"},
			http.StatusBadRequest},
		{"update no key", HandleUpdateLabel(c), ctlRequest{method: http.MethodPatch, params: ctlSpaceParams(), body: "{}"},
			http.StatusBadRequest},
		{"update no ref", HandleUpdateLabel(c), ctlRequest{method: http.MethodPatch, body: "{}"}, http.StatusBadRequest},
		{"save", HandleSaveLabel(c), ctlRequest{
			method: http.MethodPut, params: ctlSpaceParams(),
			body: `{"label":{"key":"area","type":"dynamic","color":"yellow"}}`,
		}, http.StatusOK},
		{"save bad body", HandleSaveLabel(c), ctlRequest{method: http.MethodPut, params: ctlSpaceParams(), body: "{"},
			http.StatusBadRequest},
		{"save no ref", HandleSaveLabel(c), ctlRequest{method: http.MethodPut, body: "{}"}, http.StatusBadRequest},

		{"define value", HandleDefineLabelValue(c), ctlRequest{
			method: http.MethodPost, params: keyParams, body: `{"value":"high","color":"green"}`,
		}, http.StatusCreated},
		{"define value bad body", HandleDefineLabelValue(c), ctlRequest{
			method: http.MethodPost, params: keyParams, body: "{",
		}, http.StatusBadRequest},
		{"define value no ref", HandleDefineLabelValue(c), ctlRequest{method: http.MethodPost, body: "{}"},
			http.StatusBadRequest},
		{"define value no key", HandleDefineLabelValue(c), ctlRequest{
			method: http.MethodPost, params: ctlSpaceParams(), body: "{}",
		}, http.StatusBadRequest},
		{"list values", HandleListLabelValues(c), ctlRequest{params: keyParams}, http.StatusOK},
		{"list values no key", HandleListLabelValues(c), ctlRequest{params: ctlSpaceParams()}, http.StatusBadRequest},
		{"list values no ref", HandleListLabelValues(c), ctlRequest{}, http.StatusBadRequest},
		{"update value", HandleUpdateLabelValue(c), ctlRequest{
			method: http.MethodPatch, params: valueParams, body: `{"color":"red"}`,
		}, http.StatusOK},
		{"update value bad body", HandleUpdateLabelValue(c), ctlRequest{
			method: http.MethodPatch, params: valueParams, body: "{",
		}, http.StatusBadRequest},
		{"update value no value", HandleUpdateLabelValue(c), ctlRequest{
			method: http.MethodPatch, params: keyParams, body: "{}",
		}, http.StatusBadRequest},
		{"update value no key", HandleUpdateLabelValue(c), ctlRequest{
			method: http.MethodPatch, params: ctlSpaceParams(), body: "{}",
		}, http.StatusBadRequest},
		{"update value no ref", HandleUpdateLabelValue(c), ctlRequest{method: http.MethodPatch, body: "{}"},
			http.StatusBadRequest},
		{"delete value no ref", HandleDeleteLabelValue(c), ctlRequest{method: http.MethodDelete}, http.StatusBadRequest},
		{"delete value no key", HandleDeleteLabelValue(c), ctlRequest{method: http.MethodDelete, params: ctlSpaceParams()},
			http.StatusBadRequest},
		{"delete value no value", HandleDeleteLabelValue(c), ctlRequest{method: http.MethodDelete, params: keyParams},
			http.StatusBadRequest},
		{"delete value", HandleDeleteLabelValue(c), ctlRequest{method: http.MethodDelete, params: valueParams},
			http.StatusNoContent},
		{"delete no ref", HandleDeleteLabel(c), ctlRequest{method: http.MethodDelete}, http.StatusBadRequest},
		{"delete no key", HandleDeleteLabel(c), ctlRequest{method: http.MethodDelete, params: ctlSpaceParams()},
			http.StatusBadRequest},
		{"delete", HandleDeleteLabel(c), ctlRequest{method: http.MethodDelete, params: keyParams}, http.StatusNoContent},
	})
}

func TestCtlSpaceHandlers_Rules(t *testing.T) {
	e := ctlNewEnv(t)
	c := e.ctrl
	ruleParams := ctlSpaceParams(request.PathParamRuleIdentifier, "rule1")
	rule := `{"type":"branch","state":"active","identifier":"rule1","pattern":{"default":true},` +
		`"definition":{"lifecycle":{"delete_forbidden":true}}}`

	ctlRunSteps(t, e, []ctlStep{
		{"create no ref", HandleRuleCreate(c), ctlRequest{method: http.MethodPost, body: rule}, http.StatusBadRequest},
		{"create bad body", HandleRuleCreate(c), ctlRequest{method: http.MethodPost, params: ctlSpaceParams(), body: "{"},
			http.StatusBadRequest},
		{"create invalid", HandleRuleCreate(c), ctlRequest{method: http.MethodPost, params: ctlSpaceParams(), body: "{}"},
			http.StatusBadRequest},
		{"create", HandleRuleCreate(c), ctlRequest{method: http.MethodPost, params: ctlSpaceParams(), body: rule},
			http.StatusCreated},
		{"find", HandleRuleFind(c), ctlRequest{params: ruleParams}, http.StatusOK},
		{"find no ref", HandleRuleFind(c), ctlRequest{}, http.StatusBadRequest},
		{"find no identifier", HandleRuleFind(c), ctlRequest{params: ctlSpaceParams()}, http.StatusBadRequest},
		{"list", HandleRuleList(c), ctlRequest{params: ctlSpaceParams(), query: "page=1&limit=10"}, http.StatusOK},
		{"list bad inherited", HandleRuleList(c), ctlRequest{params: ctlSpaceParams(), query: "inherited=maybe"},
			http.StatusBadRequest},
		{"list no ref", HandleRuleList(c), ctlRequest{}, http.StatusBadRequest},
		{"update", HandleRuleUpdate(c), ctlRequest{
			method: http.MethodPatch, params: ruleParams, body: `{"description":"d"}`,
		}, http.StatusOK},
		{"update bad body", HandleRuleUpdate(c), ctlRequest{method: http.MethodPatch, params: ruleParams, body: "{"},
			http.StatusBadRequest},
		{"update no identifier", HandleRuleUpdate(c), ctlRequest{
			method: http.MethodPatch, params: ctlSpaceParams(), body: "{}",
		}, http.StatusBadRequest},
		{"update no ref", HandleRuleUpdate(c), ctlRequest{method: http.MethodPatch, body: "{}"}, http.StatusBadRequest},
		{"delete no ref", HandleRuleDelete(c), ctlRequest{method: http.MethodDelete}, http.StatusBadRequest},
		{"delete no identifier", HandleRuleDelete(c), ctlRequest{method: http.MethodDelete, params: ctlSpaceParams()},
			http.StatusBadRequest},
		{"delete", HandleRuleDelete(c), ctlRequest{method: http.MethodDelete, params: ruleParams}, http.StatusNoContent},
	})
}

func TestCtlSpaceHandlers_Autolinks(t *testing.T) {
	e := ctlNewEnv(t)
	c := e.ctrl
	body := `{"type":"regex","pattern":"JIRA-(\\d+)","target_url":"https://jira.example.com/<num>"}`

	ctlRunSteps(t, e, []ctlStep{
		{"create no ref", HandleAutolinkCreate(c), ctlRequest{method: http.MethodPost, body: body}, http.StatusBadRequest},
		{"create bad body", HandleAutolinkCreate(c), ctlRequest{method: http.MethodPost, params: ctlSpaceParams(), body: "{"},
			http.StatusBadRequest},
		{"create invalid", HandleAutolinkCreate(c), ctlRequest{
			method: http.MethodPost, params: ctlSpaceParams(), body: "{}",
		}, http.StatusBadRequest},
		{"create", HandleAutolinkCreate(c), ctlRequest{method: http.MethodPost, params: ctlSpaceParams(), body: body},
			http.StatusCreated},
		{"list", HandleAutolinkList(c), ctlRequest{params: ctlSpaceParams(), query: "page=1&limit=10"}, http.StatusOK},
		{"list no ref", HandleAutolinkList(c), ctlRequest{}, http.StatusBadRequest},
	})

	idParams := ctlSpaceParams(request.PathParamAutolinkID, "1")
	ctlRunSteps(t, e, []ctlStep{
		{"find", HandleAutolinkFind(c), ctlRequest{params: idParams}, http.StatusOK},
		{"find no ref", HandleAutolinkFind(c), ctlRequest{}, http.StatusBadRequest},
		{"find no id", HandleAutolinkFind(c), ctlRequest{params: ctlSpaceParams()}, http.StatusBadRequest},
		{"update", HandleAutolinkUpdate(c), ctlRequest{
			method: http.MethodPatch, params: idParams, body: `{"pattern":"BUG-(\\d+)"}`,
		}, http.StatusOK},
		{"update bad body", HandleAutolinkUpdate(c), ctlRequest{method: http.MethodPatch, params: idParams, body: "{"},
			http.StatusBadRequest},
		{"update no id", HandleAutolinkUpdate(c), ctlRequest{
			method: http.MethodPatch, params: ctlSpaceParams(), body: "{}",
		}, http.StatusBadRequest},
		{"update no ref", HandleAutolinkUpdate(c), ctlRequest{method: http.MethodPatch, body: "{}"}, http.StatusBadRequest},
		{"delete no ref", HandleAutolinkDelete(c), ctlRequest{method: http.MethodDelete}, http.StatusBadRequest},
		{"delete no id", HandleAutolinkDelete(c), ctlRequest{method: http.MethodDelete, params: ctlSpaceParams()},
			http.StatusBadRequest},
		{"delete", HandleAutolinkDelete(c), ctlRequest{method: http.MethodDelete, params: idParams}, http.StatusNoContent},
	})
}
