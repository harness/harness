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

package webhook

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/audit"
	gitness_store "github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/check"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/require"
)

func TestCheckURL_Rules(t *testing.T) {
	tests := []struct {
		name                string
		url                 string
		loopback, private   bool
		linkLocal, internal bool
		wantErr             bool
		wantContains        string
	}{
		{name: "public host", url: "https://example.com/hook"},
		{name: "internal skips validation", url: "not even a url", internal: true},
		{name: "too long", url: "https://example.com/" + strings.Repeat("a", webhookMaxURLLength), wantErr: true,
			wantContains: "at most"},
		{name: "unparseable", url: "https://exa mple.com/%zz", wantErr: true, wantContains: "invalid"},
		{name: "empty host", url: "https:///hook", wantErr: true, wantContains: "non-empty host"},
		{name: "localhost", url: "http://localhost:8080/x", wantErr: true, wantContains: "localhost"},
		{name: "loopback blocked", url: "http://127.0.0.1/x", wantErr: true, wantContains: "Loopback"},
		{name: "loopback allowed", url: "http://127.0.0.1/x", loopback: true},
		{name: "private blocked", url: "http://10.1.2.3/x", wantErr: true, wantContains: "Private"},
		{name: "private allowed", url: "http://10.1.2.3/x", private: true},
		{name: "link local blocked", url: "http://169.254.1.1/x", wantErr: true, wantContains: "Link-local"},
		{name: "link local allowed", url: "http://169.254.1.1/x", linkLocal: true},
		{name: "public ip", url: "http://8.8.8.8/x"},
		{name: "reserved ip", url: "http://0.0.0.0/x", wantErr: true, wantContains: "not allowed"},
		{name: "bad scheme", url: "ftp://example.com/x", wantErr: true, wantContains: "http or https"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CheckURL(tt.url, tt.loopback, tt.private, tt.linkLocal, tt.internal)
			if !tt.wantErr {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			require.ErrorIs(t, err, check.ErrAny)
			require.Contains(t, err.Error(), tt.wantContains)
		})
	}
}

func TestCheckSecretTriggersAndHeaders(t *testing.T) {
	require.NoError(t, CheckSecret(""))
	require.NoError(t, CheckSecret(strings.Repeat("s", webhookMaxSecretLength)))
	require.Error(t, CheckSecret(strings.Repeat("s", webhookMaxSecretLength+1)))

	require.NoError(t, CheckTriggers(nil))
	require.NoError(t, CheckTriggers([]enum.WebhookTrigger{
		enum.WebhookTriggerBranchCreated, enum.WebhookTriggerBranchCreated,
	}))
	require.Error(t, CheckTriggers([]enum.WebhookTrigger{enum.WebhookTriggerBranchCreated, "bogus"}))

	require.NoError(t, CheckExtraHeaders(nil))
	require.NoError(t, CheckExtraHeaders([]types.ExtraHeader{{Key: "X-A"}, {Key: "X-B"}}))
	require.Error(t, CheckExtraHeaders([]types.ExtraHeader{{Key: ""}}))
	tooMany := make([]types.ExtraHeader, webhookMaxExtraHeaders+1)
	for i := range tooMany {
		tooMany[i].Key = "X-" + strings.Repeat("a", i+1)
	}
	require.Error(t, CheckExtraHeaders(tooMany))
	require.NoError(t, CheckExtraHeaders(tooMany[:webhookMaxExtraHeaders]))
}

func TestTriggerHelpers(t *testing.T) {
	require.Empty(t, DeduplicateTriggers(nil))
	require.NotNil(t, DeduplicateTriggers(nil))
	require.Equal(t,
		[]enum.WebhookTrigger{enum.WebhookTriggerTagCreated, enum.WebhookTriggerBranchCreated},
		DeduplicateTriggers([]enum.WebhookTrigger{
			enum.WebhookTriggerTagCreated, enum.WebhookTriggerBranchCreated, enum.WebhookTriggerTagCreated,
		}))

	require.Equal(t, []enum.WebhookTrigger{"a", "b"}, ConvertTriggers([]string{"a", "b"}))
	require.Empty(t, ConvertTriggers(nil))

	require.True(t, shouldAuditWebhook(enum.WebhookTypeExternal))
	require.False(t, shouldAuditWebhook(enum.WebhookTypeInternal))
	require.False(t, shouldAuditWebhook(enum.WebhookTypeJira))

	typ, name := getWebhookAuditInfo(enum.WebhookParentSpace)
	require.Equal(t, audit.ResourceTypeCodeWebhook, typ)
	require.Equal(t, audit.SpaceName, name)
	typ, name = getWebhookAuditInfo(enum.WebhookParentRegistry)
	require.Equal(t, audit.ResourceTypeRegistryWebhook, typ)
	require.Equal(t, audit.RegistryName, name)
	typ, name = getWebhookAuditInfo(enum.WebhookParentRepo)
	require.Equal(t, audit.ResourceTypeCodeWebhook, typ)
	require.Equal(t, audit.RepoName, name)
	typ, name = getWebhookAuditInfo("unknown")
	require.Empty(t, typ)
	require.Empty(t, name)
}

func TestSanitizeCreateInput(t *testing.T) {
	svc := &Service{config: Config{}}

	t.Run("identifier backfills the display name", func(t *testing.T) {
		in := &types.WebhookCreateInput{Identifier: "my-hook", URL: "https://example.com/hook"}
		require.NoError(t, svc.sanitizeCreateInput(in, false))
		require.Equal(t, "my-hook", in.DisplayName)
	})

	t.Run("deprecated uid is used as identifier", func(t *testing.T) {
		in := &types.WebhookCreateInput{UID: "legacy", URL: "https://example.com/hook"}
		require.NoError(t, svc.sanitizeCreateInput(in, false))
		require.Equal(t, "legacy", in.Identifier)
		require.Equal(t, "legacy", in.DisplayName)
	})

	t.Run("display name only derives an identifier", func(t *testing.T) {
		in := &types.WebhookCreateInput{DisplayName: "My Hook", URL: "https://example.com/hook"}
		require.NoError(t, svc.sanitizeCreateInput(in, false))
		require.NotEmpty(t, in.Identifier)
		require.NoError(t, check.Identifier(in.Identifier))
	})

	t.Run("internal webhooks skip the url check", func(t *testing.T) {
		in := &types.WebhookCreateInput{Identifier: "internal", URL: "garbage"}
		require.NoError(t, svc.sanitizeCreateInput(in, true))
	})

	for name, mutate := range map[string]func(*types.WebhookCreateInput){
		"missing identifier and name": func(in *types.WebhookCreateInput) { in.Identifier, in.DisplayName = "", "" },
		"bad identifier":              func(in *types.WebhookCreateInput) { in.Identifier = "bad id" },
		"bad display name":            func(in *types.WebhookCreateInput) { in.DisplayName = "tab\there" },
		"bad description":             func(in *types.WebhookCreateInput) { in.Description = "line\nbreak" },
		"bad url":                     func(in *types.WebhookCreateInput) { in.URL = "ftp://example.com" },
		"loopback url":                func(in *types.WebhookCreateInput) { in.URL = "http://127.0.0.1/x" },
		"bad secret": func(in *types.WebhookCreateInput) {
			in.Secret = strings.Repeat("s", webhookMaxSecretLength+1)
		},
		"bad trigger": func(in *types.WebhookCreateInput) { in.Triggers = []enum.WebhookTrigger{"bogus"} },
		"bad headers": func(in *types.WebhookCreateInput) { in.ExtraHeaders = []types.ExtraHeader{{}} },
	} {
		t.Run(name, func(t *testing.T) {
			in := &types.WebhookCreateInput{Identifier: "hook", URL: "https://example.com/hook"}
			mutate(in)
			require.Error(t, svc.sanitizeCreateInput(in, false))
		})
	}

	t.Run("config allows loopback", func(t *testing.T) {
		allowing := &Service{config: Config{AllowLoopback: true}}
		in := &types.WebhookCreateInput{Identifier: "hook", URL: "http://127.0.0.1/x"}
		require.NoError(t, allowing.sanitizeCreateInput(in, false))
	})
}

func TestSanitizeUpdateInput(t *testing.T) {
	svc := &Service{config: Config{}}
	str := func(s string) *string { return &s }

	t.Run("nothing to validate", func(t *testing.T) {
		require.NoError(t, svc.sanitizeUpdateInput(&types.WebhookUpdateInput{}))
	})

	t.Run("deprecated uid is promoted to identifier", func(t *testing.T) {
		in := &types.WebhookUpdateInput{UID: str("legacy")}
		require.NoError(t, svc.sanitizeUpdateInput(in))
		require.Equal(t, "legacy", *in.Identifier)
	})

	t.Run("valid values", func(t *testing.T) {
		in := &types.WebhookUpdateInput{
			Identifier: str("hook"), DisplayName: str("Hook"), Description: str("desc"),
			URL: str("https://example.com/x"), Secret: str("s"),
			Triggers:     []enum.WebhookTrigger{enum.WebhookTriggerTagCreated},
			ExtraHeaders: []types.ExtraHeader{{Key: "X-A"}},
		}
		require.NoError(t, svc.sanitizeUpdateInput(in))
	})

	for name, in := range map[string]*types.WebhookUpdateInput{
		"identifier":   {Identifier: str("bad id")},
		"display name": {DisplayName: str("tab\there")},
		"description":  {Description: str("line\nbreak")},
		"url":          {URL: str("ftp://example.com")},
		"secret":       {Secret: str(strings.Repeat("s", webhookMaxSecretLength+1))},
		"triggers":     {Triggers: []enum.WebhookTrigger{"bogus"}},
	} {
		t.Run("invalid "+name, func(t *testing.T) {
			require.Error(t, svc.sanitizeUpdateInput(in))
		})
	}
}

func TestConversions(t *testing.T) {
	result := enum.WebhookExecutionResultSuccess
	retrigger := int64(9)
	webhook := &types.Webhook{
		ID: 1, Version: 2, ParentID: 3, ParentType: enum.WebhookParentRepo, CreatedBy: 4, Created: 5, Updated: 6,
		Type: enum.WebhookTypeExternal, Scope: 7, Identifier: "id", DisplayName: "name", Description: "d",
		URL: "https://example.com", Secret: "s", Enabled: true, Insecure: true,
		Triggers: []enum.WebhookTrigger{enum.WebhookTriggerBranchCreated}, LatestExecutionResult: &result,
		ExtraHeaders: []types.ExtraHeader{{Key: "k", Value: "v"}},
	}

	core := GitnessWebhookToWebhookCore(webhook)
	require.Equal(t, webhook.URL, core.URL)
	require.Equal(t, webhook.ExtraHeaders, core.ExtraHeaders)
	require.Equal(t, webhook, CoreWebhookToGitnessWebhook(core), "the round trip keeps every field")

	cores := GitnessWebhooksToWebhooksCore([]*types.Webhook{webhook, webhook})
	require.Len(t, cores, 2)
	require.Empty(t, GitnessWebhooksToWebhooksCore(nil))

	execution := &types.WebhookExecution{
		ID: 1, WebhookID: 2, TriggerID: "t", TriggerType: enum.WebhookTriggerTagCreated, Result: result,
		Error: "e", Request: types.WebhookExecutionRequest{URL: "u"}, Response: types.WebhookExecutionResponse{Status: "ok"},
		RetriggerOf: &retrigger, Retriggerable: true, Duration: 8, Created: 9,
	}
	executionCore := GitnessWebhookExecutionToWebhookExecutionCore(execution)
	require.Equal(t, execution.TriggerID, executionCore.TriggerID)
	require.Equal(t, execution, CoreWebhookExecutionToGitnessWebhookExecution(executionCore))
}

func TestTriggerResult_Skipped(t *testing.T) {
	require.True(t, (&TriggerResult{}).Skipped())
	require.False(t, (&TriggerResult{Execution: &types.WebhookExecutionCore{}}).Skipped())
}

func TestExecutor_toXHeader(t *testing.T) {
	w := &WebhookExecutor{config: Config{HeaderIdentity: "Gitness"}}
	require.Equal(t, "X-Gitness-Trigger", w.toXHeader("Trigger"))
}

func TestHandleWebhookResponse(t *testing.T) {
	respond := func(code int, body string) *http.Response {
		return &http.Response{
			StatusCode: code,
			Status:     http.StatusText(code),
			Header:     http.Header{"X-Test": []string{"yes"}},
			Body:       io.NopCloser(strings.NewReader(body)),
		}
	}

	tests := []struct {
		code       int
		wantResult enum.WebhookExecutionResult
		wantErr    bool
	}{
		{100, enum.WebhookExecutionResultFatalError, true},
		{200, enum.WebhookExecutionResultSuccess, false},
		{204, enum.WebhookExecutionResultSuccess, false},
		{301, enum.WebhookExecutionResultFatalError, true},
		{400, enum.WebhookExecutionResultFatalError, true},
		{404, enum.WebhookExecutionResultFatalError, true},
		{408, enum.WebhookExecutionResultRetriableError, true},
		{429, enum.WebhookExecutionResultRetriableError, true},
		{501, enum.WebhookExecutionResultFatalError, true},
		{500, enum.WebhookExecutionResultRetriableError, true},
		{503, enum.WebhookExecutionResultRetriableError, true},
		{600, enum.WebhookExecutionResultFatalError, true},
	}
	for _, tt := range tests {
		t.Run(http.StatusText(tt.code)+"_"+string(rune('0'+tt.code/100)), func(t *testing.T) {
			execution := &types.WebhookExecutionCore{}
			resp := respond(tt.code, "response body")
			defer resp.Body.Close()
			err := handleWebhookResponse(execution, resp)
			require.Equal(t, tt.wantErr, err != nil)
			require.Equal(t, tt.wantResult, execution.Result)
			require.Equal(t, tt.code, execution.Response.StatusCode)
			require.Equal(t, "response body", execution.Response.Body)
			require.Contains(t, execution.Response.Headers, "X-Test: yes")
			if !tt.wantErr {
				require.Empty(t, execution.Error)
			} else {
				require.NotEmpty(t, execution.Error)
			}
		})
	}

	t.Run("body and headers are truncated", func(t *testing.T) {
		resp := respond(200, strings.Repeat("b", responseBodyBytesLimit*3))
		defer resp.Body.Close()
		resp.Header.Set("X-Long", strings.Repeat("h", responseHeadersBytesLimit*3))
		execution := &types.WebhookExecutionCore{}
		require.NoError(t, handleWebhookResponse(execution, resp))
		require.Len(t, execution.Response.Body, responseBodyBytesLimit)
		require.Len(t, execution.Response.Headers, responseHeadersBytesLimit)
	})

	t.Run("nil body", func(t *testing.T) {
		resp := respond(200, "")
		require.NoError(t, resp.Body.Close())
		resp.Body = nil
		execution := &types.WebhookExecutionCore{}
		require.NoError(t, handleWebhookResponse(execution, resp))
		require.Empty(t, execution.Response.Body)
	})

	t.Run("unreadable body is retriable", func(t *testing.T) {
		resp := respond(200, "")
		require.NoError(t, resp.Body.Close())
		resp.Body = io.NopCloser(errReader{})
		defer resp.Body.Close()
		execution := &types.WebhookExecutionCore{}
		require.Error(t, handleWebhookResponse(execution, resp))
		require.Equal(t, enum.WebhookExecutionResultRetriableError, execution.Result)
		require.Contains(t, execution.Error, "reading the response body")
	})
}

type errReader struct{}

func (errReader) Read([]byte) (int, error) { return 0, errors.New("read failed") }

type fakeRepoStore struct {
	store.RepoStore

	repos map[int64]*types.Repository
	err   error
}

func (f fakeRepoStore) Find(_ context.Context, id int64) (*types.Repository, error) {
	if f.err != nil {
		return nil, f.err
	}
	r, ok := f.repos[id]
	if !ok {
		return nil, gitness_store.ErrResourceNotFound
	}
	return r, nil
}

type fakePullReqStore struct {
	store.PullReqStore

	prs map[int64]*types.PullReq
	err error
}

func (f fakePullReqStore) Find(_ context.Context, id int64) (*types.PullReq, error) {
	if f.err != nil {
		return nil, f.err
	}
	p, ok := f.prs[id]
	if !ok {
		return nil, gitness_store.ErrResourceNotFound
	}
	return p, nil
}

type fakePrincipalStore struct {
	store.PrincipalStore

	principals map[int64]*types.Principal
	err        error
}

func (f fakePrincipalStore) Find(_ context.Context, id int64) (*types.Principal, error) {
	if f.err != nil {
		return nil, f.err
	}
	p, ok := f.principals[id]
	if !ok {
		return nil, gitness_store.ErrResourceNotFound
	}
	return p, nil
}

type fakeSpaceStore struct {
	store.SpaceStore

	ancestors []int64
	err       error
}

func (f fakeSpaceStore) GetAncestorIDs(context.Context, int64) ([]int64, error) {
	return f.ancestors, f.err
}

type fakeWebhookStore struct {
	store.WebhookStore

	parentsSeen [][]types.WebhookParentInfo
	count       int64
	hooks       []*types.Webhook
	countErr    error
	listErr     error
}

func (f *fakeWebhookStore) Count(
	_ context.Context, parents []types.WebhookParentInfo, _ *types.WebhookFilter,
) (int64, error) {
	f.parentsSeen = append(f.parentsSeen, parents)
	return f.count, f.countErr
}

func (f *fakeWebhookStore) List(
	context.Context, []types.WebhookParentInfo, *types.WebhookFilter,
) ([]*types.Webhook, error) {
	return f.hooks, f.listErr
}

func isDiscarded(err error) bool {
	return err != nil && strings.Contains(err.Error(), "discarding requested due to")
}

func TestFindForEvent(t *testing.T) {
	boom := errors.New("boom")

	t.Run("generate trigger id", func(t *testing.T) {
		require.Equal(t, "event-abc", generateTriggerIDFromEventID("abc"))
	})

	t.Run("repository", func(t *testing.T) {
		svc := &Service{repoStore: fakeRepoStore{repos: map[int64]*types.Repository{1: {ID: 1}}}}
		repo, err := svc.findRepositoryForEvent(t.Context(), 1)
		require.NoError(t, err)
		require.Equal(t, int64(1), repo.ID)

		_, err = svc.findRepositoryForEvent(t.Context(), 2)
		require.True(t, isDiscarded(err), "a missing repo discards the event: %v", err)

		svc.repoStore = fakeRepoStore{err: boom}
		_, err = svc.findRepositoryForEvent(t.Context(), 1)
		require.ErrorIs(t, err, boom)
		require.False(t, isDiscarded(err), "other errors are retried")
	})

	t.Run("pull request", func(t *testing.T) {
		svc := &Service{pullreqStore: fakePullReqStore{prs: map[int64]*types.PullReq{1: {ID: 1}}}}
		pr, err := svc.findPullReqForEvent(t.Context(), 1)
		require.NoError(t, err)
		require.Equal(t, int64(1), pr.ID)

		_, err = svc.findPullReqForEvent(t.Context(), 2)
		require.True(t, isDiscarded(err))

		svc.pullreqStore = fakePullReqStore{err: boom}
		_, err = svc.findPullReqForEvent(t.Context(), 1)
		require.ErrorIs(t, err, boom)
		require.False(t, isDiscarded(err))
	})

	t.Run("principal", func(t *testing.T) {
		w := &WebhookExecutor{principalStore: fakePrincipalStore{principals: map[int64]*types.Principal{1: {ID: 1}}}}
		principal, err := w.FindPrincipalForEvent(t.Context(), 1)
		require.NoError(t, err)
		require.Equal(t, int64(1), principal.ID)

		_, err = w.FindPrincipalForEvent(t.Context(), 2)
		require.True(t, isDiscarded(err))

		w.principalStore = fakePrincipalStore{err: boom}
		_, err = w.FindPrincipalForEvent(t.Context(), 1)
		require.ErrorIs(t, err, boom)
		require.False(t, isDiscarded(err))
	})
}

func TestGetParentInfo(t *testing.T) {
	boom := errors.New("boom")
	repos := fakeRepoStore{repos: map[int64]*types.Repository{5: {ID: 5, ParentID: 50}}}
	spaces := fakeSpaceStore{ancestors: []int64{50, 10}}
	svc := &Service{repoStore: repos, spaceStore: spaces}

	t.Run("repo without inheritance", func(t *testing.T) {
		got, err := svc.getParentInfoRepo(t.Context(), 5, false)
		require.NoError(t, err)
		require.Equal(t, []types.WebhookParentInfo{{ID: 5, Type: enum.WebhookParentRepo}}, got)
	})

	t.Run("repo with inheritance", func(t *testing.T) {
		got, err := svc.getParentInfoRepo(t.Context(), 5, true)
		require.NoError(t, err)
		require.Equal(t, []types.WebhookParentInfo{
			{ID: 5, Type: enum.WebhookParentRepo},
			{ID: 50, Type: enum.WebhookParentSpace},
			{ID: 10, Type: enum.WebhookParentSpace},
		}, got)
	})

	t.Run("repo errors", func(t *testing.T) {
		_, err := svc.getParentInfoRepo(t.Context(), 404, true)
		require.Error(t, err)

		failing := &Service{repoStore: repos, spaceStore: fakeSpaceStore{err: boom}}
		_, err = failing.getParentInfoRepo(t.Context(), 5, true)
		require.ErrorIs(t, err, boom)
	})

	t.Run("space", func(t *testing.T) {
		got, err := svc.getParentInfoSpace(t.Context(), 50, false)
		require.NoError(t, err)
		require.Equal(t, []types.WebhookParentInfo{{ID: 50, Type: enum.WebhookParentSpace}}, got)

		got, err = svc.getParentInfoSpace(t.Context(), 50, true)
		require.NoError(t, err)
		require.Equal(t, []types.WebhookParentInfo{
			{ID: 50, Type: enum.WebhookParentSpace}, {ID: 10, Type: enum.WebhookParentSpace},
		}, got)

		failing := &Service{spaceStore: fakeSpaceStore{err: boom}}
		_, err = failing.getParentInfoSpace(t.Context(), 50, true)
		require.ErrorIs(t, err, boom)
	})
}

func TestService_List(t *testing.T) {
	newService := func(ws *fakeWebhookStore) *Service {
		return &Service{
			webhookStore: ws,
			repoStore:    fakeRepoStore{repos: map[int64]*types.Repository{5: {ID: 5, ParentID: 50}}},
			spaceStore:   fakeSpaceStore{ancestors: []int64{50}},
		}
	}

	t.Run("repo", func(t *testing.T) {
		ws := &fakeWebhookStore{count: 2, hooks: []*types.Webhook{{ID: 1}, {ID: 2}}}
		hooks, count, err := newService(ws).List(t.Context(), 5, enum.WebhookParentRepo, true, &types.WebhookFilter{})
		require.NoError(t, err)
		require.Equal(t, int64(2), count)
		require.Len(t, hooks, 2)
		require.Len(t, ws.parentsSeen[0], 2)
	})

	t.Run("space", func(t *testing.T) {
		ws := &fakeWebhookStore{}
		_, _, err := newService(ws).List(t.Context(), 50, enum.WebhookParentSpace, false, &types.WebhookFilter{})
		require.NoError(t, err)
		require.Equal(t, []types.WebhookParentInfo{{ID: 50, Type: enum.WebhookParentSpace}}, ws.parentsSeen[0])
	})

	t.Run("registry lists without parents", func(t *testing.T) {
		ws := &fakeWebhookStore{}
		_, _, err := newService(ws).List(t.Context(), 1, enum.WebhookParentRegistry, false, &types.WebhookFilter{})
		require.NoError(t, err)
		require.Empty(t, ws.parentsSeen[0])
	})

	t.Run("unsupported parent type", func(t *testing.T) {
		_, _, err := newService(&fakeWebhookStore{}).List(t.Context(), 1, "bogus", false, &types.WebhookFilter{})
		require.Error(t, err)
	})

	t.Run("errors", func(t *testing.T) {
		boom := errors.New("boom")
		filter := &types.WebhookFilter{}
		_, _, err := newService(&fakeWebhookStore{}).List(t.Context(), 404, enum.WebhookParentRepo, true, filter)
		require.Error(t, err)
		_, _, err = newService(&fakeWebhookStore{countErr: boom}).List(t.Context(), 5, enum.WebhookParentRepo, false, filter)
		require.ErrorIs(t, err, boom)
		_, _, err = newService(&fakeWebhookStore{listErr: boom}).List(t.Context(), 5, enum.WebhookParentRepo, false, filter)
		require.ErrorIs(t, err, boom)

		failing := &Service{webhookStore: &fakeWebhookStore{}, spaceStore: fakeSpaceStore{err: boom}}
		_, _, err = failing.List(t.Context(), 50, enum.WebhookParentSpace, true, &types.WebhookFilter{})
		require.ErrorIs(t, err, boom)
	})
}

type fakeExecutorStore struct {
	WebhookExecutorStore

	webhooks      []*types.WebhookCore
	executions    []*types.WebhookExecutionCore
	listErr       error
	forTriggerErr error
}

func (f *fakeExecutorStore) ListWebhooks(context.Context, []types.WebhookParentInfo) ([]*types.WebhookCore, error) {
	return f.webhooks, f.listErr
}

func (f *fakeExecutorStore) ListForTrigger(context.Context, string) ([]*types.WebhookExecutionCore, error) {
	return f.executions, f.forTriggerErr
}

func TestExecutor_TriggerForEvent_Skipping(t *testing.T) {
	ctx := t.Context()
	parents := []types.WebhookParentInfo{{ID: 1, Type: enum.WebhookParentRepo}}
	trigger := enum.WebhookTriggerBranchCreated

	t.Run("no webhooks", func(t *testing.T) {
		w := &WebhookExecutor{webhookExecutorStore: &fakeExecutorStore{}}
		require.NoError(t, w.TriggerForEvent(ctx, "e1", parents, trigger, nil))
	})

	t.Run("all webhooks are skipped", func(t *testing.T) {
		w := &WebhookExecutor{webhookExecutorStore: &fakeExecutorStore{
			webhooks: []*types.WebhookCore{
				{ID: 1, Enabled: false},
				{ID: 2, Enabled: true, Triggers: []enum.WebhookTrigger{enum.WebhookTriggerTagCreated}},
				{ID: 3, Enabled: true},
				{ID: 4, Enabled: true},
			},
			executions: []*types.WebhookExecutionCore{
				{WebhookID: 3, Result: enum.WebhookExecutionResultSuccess},
				{WebhookID: 4, Result: enum.WebhookExecutionResultFatalError},
			},
		}}
		results, err := w.triggerWebhooksFor(ctx, parents, "event-e2", trigger, nil)
		require.NoError(t, err)
		require.Len(t, results, 4)
		for _, r := range results {
			require.True(t, r.Skipped(), "webhook %d", r.Webhook.ID)
			require.Equal(t, "event-e2", r.TriggerID)
			require.Equal(t, trigger, r.TriggerType)
		}
		require.NoError(t, w.TriggerForEvent(ctx, "e2", parents, trigger, nil))
	})

	t.Run("a missing execution history is fine", func(t *testing.T) {
		w := &WebhookExecutor{webhookExecutorStore: &fakeExecutorStore{
			webhooks:      []*types.WebhookCore{{ID: 1, Enabled: false}},
			forTriggerErr: gitness_store.ErrResourceNotFound,
		}}
		require.NoError(t, w.TriggerForEvent(ctx, "e3", parents, trigger, nil))
	})

	t.Run("listing failures are returned", func(t *testing.T) {
		boom := errors.New("boom")
		w := &WebhookExecutor{webhookExecutorStore: &fakeExecutorStore{listErr: boom}}
		require.ErrorIs(t, w.TriggerForEvent(ctx, "e4", parents, trigger, nil), boom)

		w = &WebhookExecutor{webhookExecutorStore: &fakeExecutorStore{
			webhooks:      []*types.WebhookCore{{ID: 1}},
			forTriggerErr: boom,
		}}
		err := w.TriggerForEvent(ctx, "e5", parents, trigger, nil)
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to get executions for trigger")
	})
}
