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

package repo

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	repoctrl "github.com/harness/gitness/app/api/controller/repo"
	"github.com/harness/gitness/app/api/request"
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

// ctlHandlers returns every handler of this package that takes the repository controller.
func ctlHandlers(c *repoctrl.Controller) map[string]http.HandlerFunc {
	return map[string]http.HandlerFunc{
		"HandleArchive":                   HandleArchive(c),
		"HandleAutolinkCreate":            HandleAutolinkCreate(c),
		"HandleAutolinkDelete":            HandleAutolinkDelete(c),
		"HandleAutolinkFind":              HandleAutolinkFind(c),
		"HandleAutolinkList":              HandleAutolinkList(c),
		"HandleAutolinkUpdate":            HandleAutolinkUpdate(c),
		"HandleBlame":                     HandleBlame(c),
		"HandleCalculateCommitDivergence": HandleCalculateCommitDivergence(c),
		"HandleCodeOwnersRaw":             HandleCodeOwnersRaw(c),
		"HandleCodeOwnersValidate":        HandleCodeOwnersValidate(c),
		"HandleCommitFiles":               HandleCommitFiles(c),
		"HandleGetContent":                HandleGetContent(c),
		"HandlePathsDetails":              HandlePathsDetails(c),
		"HandleCreate":                    HandleCreate(c),
		"HandleCreateBranch":              HandleCreateBranch(c),
		"HandleCreateCommitTag":           HandleCreateCommitTag(c),
		"HandleCreateFork":                HandleCreateFork(c),
		"HandleUpdateDefaultBranch":       HandleUpdateDefaultBranch(c),
		"HandleDeleteBranch":              HandleDeleteBranch(c),
		"HandleDeleteCommitTag":           HandleDeleteCommitTag(c),
		"HandleDiff":                      HandleDiff(c),
		"HandleCommitDiff":                HandleCommitDiff(c),
		"HandleDiffStats":                 HandleDiffStats(c),
		"HandleFind":                      HandleFind(c),
		"HandleForkSync":                  HandleForkSync(c),
		"HandleGetBranch":                 HandleGetBranch(c),
		"HandleGetCommit":                 HandleGetCommit(c),
		"HandleGetLinkedSource":           HandleGetLinkedSource(c),
		"HandleGitInfoRefs":               HandleGitInfoRefs(c, ctlURLProvider{}),
		"HandleImport":                    HandleImport(c),
		"HandleImportProgress":            HandleImportProgress(c),
		"HandleDefineLabel":               HandleDefineLabel(c),
		"HandleDeleteLabel":               HandleDeleteLabel(c),
		"HandleFindLabel":                 HandleFindLabel(c),
		"HandleListLabels":                HandleListLabels(c),
		"HandleSaveLabel":                 HandleSaveLabel(c),
		"HandleUpdateLabel":               HandleUpdateLabel(c),
		"HandleDefineLabelValue":          HandleDefineLabelValue(c),
		"HandleDeleteLabelValue":          HandleDeleteLabelValue(c),
		"HandleListLabelValues":           HandleListLabelValues(c),
		"HandleUpdateLabelValue":          HandleUpdateLabelValue(c),
		"HandleGetLanguages":              HandleGetLanguages(c),
		"HandleLinkedCreate":              HandleLinkedCreate(c),
		"HandleLinkedRegisterWebhook":     HandleLinkedRegisterWebhook(c),
		"HandleLinkedSync":                HandleLinkedSync(c),
		"HandleListActivities":            HandleListActivities(c),
		"HandleListBranches":              HandleListBranches(c),
		"HandleListCommitTags":            HandleListCommitTags(c),
		"HandleListCommits":               HandleListCommits(c),
		"HandleListPaths":                 HandleListPaths(c),
		"HandleListPipelines":             HandleListPipelines(c),
		"HandleListServiceAccounts":       HandleListServiceAccounts(c),
		"HandleMergeCheck":                HandleMergeCheck(c),
		"HandleMergeCommit":               HandleMergeCommit(c),
		"HandleMergeQueueClear":           HandleMergeQueueClear(c),
		"HandleMergeQueueListEntries":     HandleMergeQueueListEntries(c),
		"HandleMove":                      HandleMove(c),
		"HandlePipelineGenerate":          HandlePipelineGenerate(c),
		"HandlePurge":                     HandlePurge(c),
		"HandleRaw":                       HandleRaw(c),
		"HandleRebase":                    HandleRebase(c),
		"HandleRestore":                   HandleRestore(c),
		"HandleRuleCreate":                HandleRuleCreate(c),
		"HandleRuleDelete":                HandleRuleDelete(c),
		"HandleRuleFind":                  HandleRuleFind(c),
		"HandleRuleList":                  HandleRuleList(c),
		"HandleRuleUpdate":                HandleRuleUpdate(c),
		"HandleSoftDelete":                HandleSoftDelete(c),
		"HandleSquash":                    HandleSquash(c),
		"HandleSummary":                   HandleSummary(c),
		"HandleUpdate":                    HandleUpdate(c),
		"HandleUpdatePublicAccess":        HandleUpdatePublicAccess(c),
		"HandleGitServicePack":            HandleGitServicePack(enum.GitServiceTypeUploadPack, c, ctlURLProvider{}),
	}
}

// ctlAllPathParams provides a value for every path parameter any of the handlers reads.
func ctlAllPathParams(repoRef string) map[string]string {
	return map[string]string{
		request.PathParamRepoRef:             repoRef,
		request.PathParamCommitSHA:           "0123456789abcdef0123456789abcdef01234567",
		request.PathParamRemainder:           "main/README.md",
		request.PathParamAutolinkID:          "1",
		request.PathParamLabelKey:            "key",
		request.PathParamLabelValue:          "value",
		request.PathParamRuleIdentifier:      "rule",
		request.PathParamResourceID:          "1",
		request.PathParamWebhookIdentifier:   "hook",
		request.PathParamPullReqNumber:       "1",
		request.PathParamSourceBranch:        "src",
		request.PathParamTargetBranch:        "dst",
		request.PathParamPublicKeyIdentifier: "key",
	}
}

// ctlBodyOverrides provides the valid empty body for handlers that don't decode into an object.
var ctlBodyOverrides = map[string]string{"HandleDiff": "[]"}

// ctlRendersDecodeErrorAsServerError lists handlers that translate a request body decoding error into a
// 500 instead of a 400 (see the PR description); the generic malformed body check skips them.
var ctlRendersDecodeErrorAsServerError = map[string]bool{
	"HandleDiff":                true,
	"HandleUpdateDefaultBranch": true,
}

func ctlBody(name string) string {
	if body, ok := ctlBodyOverrides[name]; ok {
		return body
	}
	return "{}"
}

func TestCtlRepoHandlers_RequestValidation(t *testing.T) {
	e := ctlNewEnv(t, ctlAllow)

	for name, h := range ctlHandlers(e.ctrl) {
		t.Run(name, func(t *testing.T) {
			// No path parameters at all.
			w := ctlServe(t, e.ctlBase, h, ctlRequest{method: http.MethodPost, body: ctlBody(name)})
			require.GreaterOrEqual(t, w.Code, 400, "no params")
			require.Less(t, w.Code, 500, "no params")

			// Malformed body with all path parameters present.
			if !ctlRendersDecodeErrorAsServerError[name] {
				w = ctlServe(t, e.ctlBase, h, ctlRequest{
					method: http.MethodPost, params: ctlAllPathParams("root/unknown"), body: "{",
				})
				require.GreaterOrEqual(t, w.Code, 400, "bad body")
				require.Less(t, w.Code, 500, "bad body")
			}

			// Unknown repository.
			w = ctlServe(t, e.ctlBase, h, ctlRequest{
				method: http.MethodPost, params: ctlAllPathParams("root/unknown"), body: ctlBody(name),
			})
			require.GreaterOrEqual(t, w.Code, 400, "unknown repo")
			require.Less(t, w.Code, 500, "unknown repo")
		})
	}
}

func TestCtlRepoHandlers_AccessDenied(t *testing.T) {
	e := ctlNewEnv(t, ctlDeny)

	for name, h := range ctlHandlers(e.ctrl) {
		t.Run(name, func(t *testing.T) {
			w := ctlServe(t, e.ctlBase, h, ctlRequest{
				method: http.MethodPost, params: ctlAllPathParams("root/repo1"), body: ctlBody(name),
			})
			require.GreaterOrEqual(t, w.Code, 400)
			require.Less(t, w.Code, 500)
		})
	}
}
