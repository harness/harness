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
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	pullreqevents "github.com/harness/gitness/app/events/pullreq"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/events"
	gitness_store "github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/require"
)

// The fakes below embed the store interfaces and implement only the methods the
// draft changed handler calls; any other call panics, which flags unexpected use.

type draftFakePrincipalStore struct {
	store.PrincipalStore
	principals map[int64]*types.Principal
}

func (f *draftFakePrincipalStore) Find(_ context.Context, id int64) (*types.Principal, error) {
	if p, ok := f.principals[id]; ok {
		return p, nil
	}
	return nil, gitness_store.ErrResourceNotFound
}

type draftFakePullReqStore struct {
	store.PullReqStore
	prs map[int64]*types.PullReq
}

func (f *draftFakePullReqStore) Find(_ context.Context, id int64) (*types.PullReq, error) {
	if pr, ok := f.prs[id]; ok {
		return pr, nil
	}
	return nil, gitness_store.ErrResourceNotFound
}

type draftFakeRepoStore struct {
	store.RepoStore
	repos map[int64]*types.Repository
}

func (f *draftFakeRepoStore) Find(_ context.Context, id int64) (*types.Repository, error) {
	if r, ok := f.repos[id]; ok {
		return r, nil
	}
	return nil, gitness_store.ErrResourceNotFound
}

type draftFakeSpaceStore struct {
	store.SpaceStore
	ancestors map[int64][]int64
}

func (f *draftFakeSpaceStore) GetAncestorIDs(_ context.Context, spaceID int64) ([]int64, error) {
	return f.ancestors[spaceID], nil
}

// draftFakeExecutorStore serves a fixed set of webhooks and records executions.
type draftFakeExecutorStore struct {
	webhooks      []*types.WebhookCore
	listedParents []types.WebhookParentInfo
	executions    []*types.WebhookExecutionCore
}

func (f *draftFakeExecutorStore) Find(context.Context, int64) (*types.WebhookExecutionCore, error) {
	return nil, gitness_store.ErrResourceNotFound
}

func (f *draftFakeExecutorStore) ListWebhooks(
	_ context.Context, parents []types.WebhookParentInfo,
) ([]*types.WebhookCore, error) {
	f.listedParents = parents
	return f.webhooks, nil
}

func (f *draftFakeExecutorStore) UpdateOptLock(
	_ context.Context, hook *types.WebhookCore, _ *types.WebhookExecutionCore,
) (*types.WebhookCore, error) {
	return hook, nil
}

func (f *draftFakeExecutorStore) FindWebhook(context.Context, int64) (*types.WebhookCore, error) {
	return nil, gitness_store.ErrResourceNotFound
}

func (f *draftFakeExecutorStore) ListForTrigger(context.Context, string) ([]*types.WebhookExecutionCore, error) {
	return nil, nil
}

func (f *draftFakeExecutorStore) CreateWebhookExecution(_ context.Context, e *types.WebhookExecutionCore) error {
	f.executions = append(f.executions, e)
	return nil
}

type draftFakeWebhookURLProvider struct{}

func (draftFakeWebhookURLProvider) GetWebhookURL(_ context.Context, w *types.WebhookCore) (string, error) {
	return w.URL, nil
}

// draftRecordingTransport captures outgoing webhook requests and answers 200 OK.
type draftRecordingTransport struct {
	requests []*http.Request
	bodies   []string
}

func (rt *draftRecordingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	raw, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}
	rt.requests = append(rt.requests, req)
	rt.bodies = append(rt.bodies, string(raw))
	return &http.Response{
		StatusCode: http.StatusOK,
		Status:     "200 OK",
		Header:     http.Header{},
		Body:       io.NopCloser(strings.NewReader("")),
		Request:    req,
	}, nil
}

const (
	draftTestPrincipalID  int64 = 7
	draftTestPRID         int64 = 100
	draftTestTargetRepoID int64 = 1
	draftTestForkRepoID   int64 = 2
	draftTestParentSpace  int64 = 10
	draftTestRootSpace    int64 = 11
)

type draftHandlerFixture struct {
	svc       *Service
	executor  *draftFakeExecutorStore
	transport *draftRecordingTransport
	prs       *draftFakePullReqStore
	repos     *draftFakeRepoStore
}

func newDraftHandlerFixture(t *testing.T, webhooks ...*types.WebhookCore) *draftHandlerFixture {
	t.Helper()

	principalStore := &draftFakePrincipalStore{principals: map[int64]*types.Principal{
		draftTestPrincipalID: {
			ID:          draftTestPrincipalID,
			UID:         "jdoe",
			DisplayName: "J Doe",
			Email:       "jdoe@example.com",
			Type:        enum.PrincipalTypeUser,
		},
	}}

	repos := &draftFakeRepoStore{repos: map[int64]*types.Repository{
		draftTestTargetRepoID: {
			ID:            draftTestTargetRepoID,
			ParentID:      draftTestParentSpace,
			Identifier:    "repo",
			Path:          "space/repo",
			DefaultBranch: "main",
		},
		draftTestForkRepoID: {
			ID:            draftTestForkRepoID,
			ParentID:      draftTestParentSpace,
			Identifier:    "fork",
			Path:          "space/fork",
			DefaultBranch: "main",
		},
	}}

	sourceRepoID := draftTestTargetRepoID
	prs := &draftFakePullReqStore{prs: map[int64]*types.PullReq{
		draftTestPRID: {
			ID:           draftTestPRID,
			Number:       42,
			State:        enum.PullReqStateOpen,
			IsDraft:      true,
			Title:        "Add feature",
			SourceRepoID: &sourceRepoID,
			SourceBranch: "feature",
			TargetRepoID: draftTestTargetRepoID,
			TargetBranch: "main",
			MergeBaseSHA: "abc123",
			Author:       types.PrincipalInfo{ID: draftTestPrincipalID, UID: "jdoe"},
		},
	}}

	executorStore := &draftFakeExecutorStore{webhooks: webhooks}
	transport := &draftRecordingTransport{}
	client := &http.Client{Transport: transport}

	executor := &WebhookExecutor{
		secureHTTPClient:           client,
		insecureHTTPClient:         client,
		secureHTTPClientInternal:   client,
		insecureHTTPClientInternal: client,
		config:                     Config{UserAgentIdentity: "Gitness", HeaderIdentity: "Gitness"},
		webhookURLProvider:         draftFakeWebhookURLProvider{},
		principalStore:             principalStore,
		webhookExecutorStore:       executorStore,
		source:                     "Trigger",
	}

	svc := &Service{
		WebhookExecutor: executor,
		urlProvider:     &mockURLProvider{},
		spaceStore: &draftFakeSpaceStore{ancestors: map[int64][]int64{
			draftTestParentSpace: {draftTestParentSpace, draftTestRootSpace},
		}},
		repoStore:      repos,
		pullreqStore:   prs,
		principalStore: principalStore,
	}

	return &draftHandlerFixture{
		svc:       svc,
		executor:  executorStore,
		transport: transport,
		prs:       prs,
		repos:     repos,
	}
}

func draftEvent(isDraft bool) *events.Event[*pullreqevents.DraftChangedPayload] {
	return &events.Event[*pullreqevents.DraftChangedPayload]{
		ID: "evt-1",
		Payload: &pullreqevents.DraftChangedPayload{
			Base: pullreqevents.Base{
				PullReqID:    draftTestPRID,
				TargetRepoID: draftTestTargetRepoID,
				PrincipalID:  draftTestPrincipalID,
				Number:       42,
			},
			IsDraft: isDraft,
		},
	}
}

func enabledWebhook(id int64, triggers ...enum.WebhookTrigger) *types.WebhookCore {
	return &types.WebhookCore{
		ID:         id,
		ParentID:   draftTestTargetRepoID,
		ParentType: enum.WebhookParentRepo,
		Identifier: "hook",
		URL:        "https://hooks.example.com/draft",
		Enabled:    true,
		Triggers:   triggers,
	}
}

func TestHandleEventPullReqDraftChanged_Payload(t *testing.T) {
	tests := []struct {
		name    string
		isDraft bool
	}{
		{name: "marked as draft", isDraft: true},
		{name: "marked ready for review", isDraft: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newDraftHandlerFixture(t, enabledWebhook(1, enum.WebhookTriggerPullReqDraftChanged))
			f.prs.prs[draftTestPRID].IsDraft = tt.isDraft

			err := f.svc.handleEventPullReqDraftChanged(context.Background(), draftEvent(tt.isDraft))
			require.NoError(t, err)

			require.Len(t, f.transport.requests, 1)
			req := f.transport.requests[0]
			require.Equal(t, "https://hooks.example.com/draft", req.URL.String())
			require.Equal(t, string(enum.WebhookTriggerPullReqDraftChanged), req.Header.Get("X-Gitness-Trigger"))

			// webhooks must be looked up on the target repo and all its ancestor spaces.
			require.Equal(t, []types.WebhookParentInfo{
				{ID: draftTestTargetRepoID, Type: enum.WebhookParentRepo},
				{ID: draftTestParentSpace, Type: enum.WebhookParentSpace},
				{ID: draftTestRootSpace, Type: enum.WebhookParentSpace},
			}, f.executor.listedParents)

			var body PullReqDraftChangedPayload
			require.NoError(t, json.Unmarshal([]byte(f.transport.bodies[0]), &body))

			require.Equal(t, enum.WebhookTriggerPullReqDraftChanged, body.Trigger)
			require.Equal(t, "space/repo", body.Repo.Path)
			require.Equal(t, draftTestPrincipalID, body.Principal.ID)
			require.Equal(t, "jdoe", body.Principal.UID)

			require.Equal(t, int64(42), body.PullReq.Number)
			require.Equal(t, tt.isDraft, body.PullReq.IsDraft)
			require.Equal(t, "Add feature", body.PullReq.Title)

			require.Equal(t, "refs/heads/main", body.TargetRef.Name)
			require.Equal(t, "space/repo", body.TargetRef.Repo.Path)
			require.Equal(t, "refs/heads/feature", body.Ref.Name)
			require.Equal(t, "space/repo", body.Ref.Repo.Path)

			require.Equal(t, tt.isDraft, body.IsDraft)

			// raw JSON keys are part of the public webhook contract.
			var raw map[string]json.RawMessage
			require.NoError(t, json.Unmarshal([]byte(f.transport.bodies[0]), &raw))
			for _, key := range []string{"trigger", "repo", "principal", "pull_req", "target_ref", "ref",
				"is_draft"} {
				require.Contains(t, raw, key)
			}
			for _, key := range []string{"old_draft", "new_draft"} {
				require.NotContains(t, raw, key)
			}

			require.Len(t, f.executor.executions, 1)
			exec := f.executor.executions[0]
			require.Equal(t, "event-evt-1", exec.TriggerID)
			require.Equal(t, enum.WebhookTriggerPullReqDraftChanged, exec.TriggerType)
			require.Equal(t, enum.WebhookExecutionResultSuccess, exec.Result)
		})
	}
}

func TestHandleEventPullReqDraftChanged_ForkSourceRepo(t *testing.T) {
	f := newDraftHandlerFixture(t, enabledWebhook(1))
	forkID := draftTestForkRepoID
	f.prs.prs[draftTestPRID].SourceRepoID = &forkID

	err := f.svc.handleEventPullReqDraftChanged(context.Background(), draftEvent(false))
	require.NoError(t, err)

	require.Len(t, f.transport.bodies, 1)
	var body PullReqDraftChangedPayload
	require.NoError(t, json.Unmarshal([]byte(f.transport.bodies[0]), &body))

	require.Equal(t, "space/repo", body.Repo.Path)
	require.Equal(t, "space/repo", body.TargetRef.Repo.Path)
	require.Equal(t, "space/fork", body.Ref.Repo.Path, "source ref must point at the fork")
}

func TestHandleEventPullReqDraftChanged_WebhookFiltering(t *testing.T) {
	disabled := enabledWebhook(2, enum.WebhookTriggerPullReqDraftChanged)
	disabled.Enabled = false

	f := newDraftHandlerFixture(t,
		enabledWebhook(1, enum.WebhookTriggerPullReqCreated), // subscribed to a different trigger
		disabled,
		enabledWebhook(3), // no trigger filter: receives everything
		enabledWebhook(4, enum.WebhookTriggerPullReqDraftChanged),
	)

	err := f.svc.handleEventPullReqDraftChanged(context.Background(), draftEvent(true))
	require.NoError(t, err)

	executed := make([]int64, 0, len(f.executor.executions))
	for _, e := range f.executor.executions {
		executed = append(executed, e.WebhookID)
	}
	require.ElementsMatch(t, []int64{3, 4}, executed)
}

func TestHandleEventPullReqDraftChanged_Discards(t *testing.T) {
	tests := []struct {
		name  string
		setup func(f *draftHandlerFixture, e *events.Event[*pullreqevents.DraftChangedPayload])
	}{
		{
			name: "pull request deleted",
			setup: func(f *draftHandlerFixture, _ *events.Event[*pullreqevents.DraftChangedPayload]) {
				delete(f.prs.prs, draftTestPRID)
			},
		},
		{
			name: "principal not found",
			setup: func(_ *draftHandlerFixture, e *events.Event[*pullreqevents.DraftChangedPayload]) {
				e.Payload.PrincipalID = 999
			},
		},
		{
			name: "target repo deleted",
			setup: func(f *draftHandlerFixture, _ *events.Event[*pullreqevents.DraftChangedPayload]) {
				delete(f.repos.repos, draftTestTargetRepoID)
			},
		},
		{
			name: "source repo purged",
			setup: func(f *draftHandlerFixture, _ *events.Event[*pullreqevents.DraftChangedPayload]) {
				f.prs.prs[draftTestPRID].SourceRepoID = nil
			},
		},
		{
			name: "fork source repo deleted",
			setup: func(f *draftHandlerFixture, _ *events.Event[*pullreqevents.DraftChangedPayload]) {
				missing := int64(555)
				f.prs.prs[draftTestPRID].SourceRepoID = &missing
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newDraftHandlerFixture(t, enabledWebhook(1))
			e := draftEvent(true)
			tt.setup(f, e)

			err := f.svc.handleEventPullReqDraftChanged(context.Background(), e)
			require.Error(t, err)

			// discard errors match each other via their Is method; the concrete type is unexported.
			require.ErrorIs(t, err, events.NewDiscardEventError(nil), "expected discard error, got %v", err)
			require.Empty(t, f.transport.requests, "no webhook must be sent")
		})
	}
}
