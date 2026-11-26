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
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sort"
	"sync"
	"testing"

	"github.com/harness/gitness/app/sse"
	appstore "github.com/harness/gitness/app/store"
	"github.com/harness/gitness/app/url"
	"github.com/harness/gitness/audit"
	gitnesserrors "github.com/harness/gitness/errors"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/git/sha"
	"github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/require"
)

const (
	svcSHAOne = "0000000000000000000000000000000000000001"
	svcSHATwo = "0000000000000000000000000000000000000002"
)

// svcWebhookStore is an in-memory store.WebhookStore.
type svcWebhookStore struct {
	appstore.WebhookStore
	mu        sync.Mutex
	hooks     map[int64]*types.Webhook
	nextID    int64
	createErr error
	updateErr error
	deleteErr error
	listErr   error
	countErr  error
	deleted   []int64
}

func newSvcWebhookStore() *svcWebhookStore {
	return &svcWebhookStore{hooks: map[int64]*types.Webhook{}}
}

func (s *svcWebhookStore) add(h *types.Webhook) *types.Webhook {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nextID++
	h.ID = s.nextID
	s.hooks[h.ID] = h
	return h
}

func (s *svcWebhookStore) Find(_ context.Context, id int64) (*types.Webhook, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	h, ok := s.hooks[id]
	if !ok {
		return nil, store.ErrResourceNotFound
	}
	cp := *h
	return &cp, nil
}

func (s *svcWebhookStore) FindByIdentifier(
	_ context.Context, parentType enum.WebhookParent, parentID int64, identifier string,
) (*types.Webhook, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, h := range s.hooks {
		if h.ParentType == parentType && h.ParentID == parentID && h.Identifier == identifier {
			cp := *h
			return &cp, nil
		}
	}
	return nil, store.ErrResourceNotFound
}

func (s *svcWebhookStore) Create(ctx context.Context, h *types.Webhook) error {
	if s.createErr != nil {
		return s.createErr
	}
	if _, err := s.FindByIdentifier(ctx, h.ParentType, h.ParentID, h.Identifier); err == nil {
		return store.ErrDuplicate
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nextID++
	h.ID = s.nextID
	cp := *h
	s.hooks[h.ID] = &cp
	return nil
}

func (s *svcWebhookStore) Update(_ context.Context, h *types.Webhook) error {
	if s.updateErr != nil {
		return s.updateErr
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := *h
	s.hooks[h.ID] = &cp
	return nil
}

func (s *svcWebhookStore) UpdateOptLock(
	_ context.Context, h *types.Webhook, fn func(*types.Webhook) error,
) (*types.Webhook, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	cur, ok := s.hooks[h.ID]
	if !ok {
		return nil, store.ErrResourceNotFound
	}
	if err := fn(cur); err != nil {
		return nil, err
	}
	cp := *cur
	return &cp, nil
}

func (s *svcWebhookStore) Delete(_ context.Context, id int64) error {
	if s.deleteErr != nil {
		return s.deleteErr
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.hooks, id)
	s.deleted = append(s.deleted, id)
	return nil
}

func (s *svcWebhookStore) matching(parents []types.WebhookParentInfo) []*types.Webhook {
	s.mu.Lock()
	defer s.mu.Unlock()
	var res []*types.Webhook
	for _, h := range s.hooks {
		for _, p := range parents {
			if h.ParentID == p.ID && h.ParentType == p.Type {
				cp := *h
				res = append(res, &cp)
				break
			}
		}
	}
	sort.Slice(res, func(i, j int) bool { return res[i].ID < res[j].ID })
	return res
}

func (s *svcWebhookStore) List(
	_ context.Context, parents []types.WebhookParentInfo, _ *types.WebhookFilter,
) ([]*types.Webhook, error) {
	if s.listErr != nil {
		return nil, s.listErr
	}
	return s.matching(parents), nil
}

func (s *svcWebhookStore) Count(
	_ context.Context, parents []types.WebhookParentInfo, _ *types.WebhookFilter,
) (int64, error) {
	if s.countErr != nil {
		return 0, s.countErr
	}
	return int64(len(s.matching(parents))), nil
}

// svcExecutionStore is an in-memory store.WebhookExecutionStore.
type svcExecutionStore struct {
	appstore.WebhookExecutionStore
	mu         sync.Mutex
	executions map[int64]*types.WebhookExecution
	nextID     int64
	listErr    error
	countErr   error
}

func newSvcExecutionStore() *svcExecutionStore {
	return &svcExecutionStore{executions: map[int64]*types.WebhookExecution{}}
}

func (s *svcExecutionStore) Find(_ context.Context, id int64) (*types.WebhookExecution, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.executions[id]
	if !ok {
		return nil, store.ErrResourceNotFound
	}
	return e, nil
}

func (s *svcExecutionStore) Create(_ context.Context, e *types.WebhookExecution) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nextID++
	e.ID = s.nextID
	cp := *e
	s.executions[e.ID] = &cp
	return nil
}

func (s *svcExecutionStore) ListForTrigger(_ context.Context, triggerID string) ([]*types.WebhookExecution, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var res []*types.WebhookExecution
	for _, e := range s.executions {
		if e.TriggerID == triggerID {
			res = append(res, e)
		}
	}
	return res, nil
}

func (s *svcExecutionStore) ListForWebhook(
	_ context.Context, webhookID int64, _ *types.WebhookExecutionFilter,
) ([]*types.WebhookExecution, error) {
	if s.listErr != nil {
		return nil, s.listErr
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	var res []*types.WebhookExecution
	for _, e := range s.executions {
		if e.WebhookID == webhookID {
			res = append(res, e)
		}
	}
	return res, nil
}

func (s *svcExecutionStore) CountForWebhook(_ context.Context, webhookID int64) (int64, error) {
	if s.countErr != nil {
		return 0, s.countErr
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	var n int64
	for _, e := range s.executions {
		if e.WebhookID == webhookID {
			n++
		}
	}
	return n, nil
}

type svcRepoStore struct {
	appstore.RepoStore
	repos map[int64]*types.Repository
	err   error
}

func (s *svcRepoStore) Find(_ context.Context, id int64) (*types.Repository, error) {
	if s.err != nil {
		return nil, s.err
	}
	r, ok := s.repos[id]
	if !ok {
		return nil, store.ErrResourceNotFound
	}
	return r, nil
}

type svcPullReqStore struct {
	appstore.PullReqStore
	prs        map[int64]*types.PullReq
	listResult []*types.PullReq
	listErr    error
	lastFilter *types.PullReqFilter
}

func (s *svcPullReqStore) Find(_ context.Context, id int64) (*types.PullReq, error) {
	p, ok := s.prs[id]
	if !ok {
		return nil, store.ErrResourceNotFound
	}
	return p, nil
}

func (s *svcPullReqStore) List(_ context.Context, f *types.PullReqFilter) ([]*types.PullReq, error) {
	s.lastFilter = f
	return s.listResult, s.listErr
}

type svcPrincipals struct {
	appstore.PrincipalStore
	principals map[int64]*types.Principal
}

func (s *svcPrincipals) Find(_ context.Context, id int64) (*types.Principal, error) {
	p, ok := s.principals[id]
	if !ok {
		return nil, store.ErrResourceNotFound
	}
	return p, nil
}

type svcActivityStore struct {
	appstore.PullReqActivityStore
	activities map[int64]*types.PullReqActivity
}

func (s *svcActivityStore) Find(_ context.Context, id int64) (*types.PullReqActivity, error) {
	a, ok := s.activities[id]
	if !ok {
		return nil, store.ErrResourceNotFound
	}
	return a, nil
}

type svcLabelStore struct {
	appstore.LabelStore
	labels map[int64]*types.Label
}

func (s *svcLabelStore) FindByID(_ context.Context, id int64) (*types.Label, error) {
	l, ok := s.labels[id]
	if !ok {
		return nil, store.ErrResourceNotFound
	}
	return l, nil
}

type svcLabelValueStore struct {
	appstore.LabelValueStore
	values map[int64]*types.LabelValue
}

func (s *svcLabelValueStore) FindByID(_ context.Context, id int64) (*types.LabelValue, error) {
	v, ok := s.values[id]
	if !ok {
		return nil, store.ErrResourceNotFound
	}
	return v, nil
}

type svcUserGroupStore struct {
	appstore.UserGroupStore
	groups map[int64]*types.UserGroup
}

func (s *svcUserGroupStore) Find(_ context.Context, id int64) (*types.UserGroup, error) {
	g, ok := s.groups[id]
	if !ok {
		return nil, store.ErrResourceNotFound
	}
	return g, nil
}

type svcSpaceStore struct {
	appstore.SpaceStore
	ancestors map[int64][]int64
	level     int64
	levelErr  error
	ancErr    error
}

func (s *svcSpaceStore) GetAncestorIDs(_ context.Context, id int64) ([]int64, error) {
	if s.ancErr != nil {
		return nil, s.ancErr
	}
	return s.ancestors[id], nil
}

func (s *svcSpaceStore) GetTreeLevel(context.Context, int64) (int64, error) {
	return s.level, s.levelErr
}

// svcGit is a fake git.Interface supporting GetCommit and ListCommits.
type svcGit struct {
	git.Interface
	commits    map[string]git.Commit
	getErr     error
	listErr    error
	listResult []git.Commit
	total      int
}

func (g *svcGit) GetCommit(_ context.Context, p *git.GetCommitParams) (*git.GetCommitOutput, error) {
	if g.getErr != nil {
		return nil, g.getErr
	}
	c, ok := g.commits[p.Revision]
	if !ok {
		return nil, gitnesserrors.NotFound("commit not found")
	}
	return &git.GetCommitOutput{Commit: c}, nil
}

func (g *svcGit) ListCommits(context.Context, *git.ListCommitsParams) (*git.ListCommitsOutput, error) {
	if g.listErr != nil {
		return nil, g.listErr
	}
	return &git.ListCommitsOutput{Commits: g.listResult, TotalCommits: g.total}, nil
}

type svcSSE struct {
	sse.Streamer
	mu     sync.Mutex
	events []svcSSEEvent
}

type svcSSEEvent struct {
	spaceID int64
	typ     enum.SSEType
	data    any
}

func (s *svcSSE) Publish(_ context.Context, spaceID int64, typ enum.SSEType, data any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, svcSSEEvent{spaceID, typ, data})
}

// svcAudit records audit calls.
type svcAudit struct {
	audit.Service
	mu      sync.Mutex
	actions []audit.Action
	err     error
}

func (a *svcAudit) Log(
	_ context.Context, _ types.Principal, _ audit.Resource, action audit.Action, _ string, _ ...audit.Option,
) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.actions = append(a.actions, action)
	return a.err
}

// svcReceived is a webhook request recorded by the receiver.
type svcReceived struct {
	Trigger string
	Body    map[string]any
}

type svcReceiver struct {
	mu       sync.Mutex
	received []svcReceived
	status   int
}

func (r *svcReceiver) last(t *testing.T) svcReceived {
	t.Helper()
	r.mu.Lock()
	defer r.mu.Unlock()
	require.NotEmpty(t, r.received, "no webhook request received")
	return r.received[len(r.received)-1]
}

func (r *svcReceiver) count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.received)
}

// svcHarness wires a webhook Service to in-memory fakes and an HTTP receiver.
type svcHarness struct {
	svc       *Service
	receiver  *svcReceiver
	hooks     *svcWebhookStore
	execs     *svcExecutionStore
	repos     *svcRepoStore
	prs       *svcPullReqStore
	principal *svcPrincipals
	git       *svcGit
	activity  *svcActivityStore
	labels    *svcLabelStore
	values    *svcLabelValueStore
	groups    *svcUserGroupStore
	spaces    *svcSpaceStore
	sse       *svcSSE
	audit     *svcAudit
	hookURL   string
}

func newSvcHarness(t *testing.T) *svcHarness {
	t.Helper()

	rec := &svcReceiver{status: http.StatusOK}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		body := map[string]any{}
		_ = json.Unmarshal(b, &body)
		rec.mu.Lock()
		rec.received = append(rec.received, svcReceived{Trigger: r.Header.Get("X-Svctest-Trigger"), Body: body})
		status := rec.status
		rec.mu.Unlock()
		w.WriteHeader(status)
	}))
	t.Cleanup(srv.Close)

	srcRepoID := int64(1)
	h := &svcHarness{
		receiver: rec,
		hooks:    newSvcWebhookStore(),
		execs:    newSvcExecutionStore(),
		repos: &svcRepoStore{repos: map[int64]*types.Repository{
			1: {ID: 1, ParentID: 10, Identifier: "repo", Path: "root/repo", GitUID: "uid1"},
			2: {ID: 2, ParentID: 10, Identifier: "fork", Path: "root/fork", GitUID: "uid2"},
		}},
		prs: &svcPullReqStore{prs: map[int64]*types.PullReq{
			100: {
				ID: 100, Number: 7, TargetRepoID: 1, SourceRepoID: &srcRepoID,
				TargetBranch: "main", SourceBranch: "feature", MergeBaseSHA: svcSHAOne, Title: "pr title",
				State: enum.PullReqStateOpen, CreatedBy: 5,
			},
		}},
		principal: &svcPrincipals{principals: map[int64]*types.Principal{
			5: {ID: 5, UID: "alice", Email: "alice@example.com", DisplayName: "Alice"},
			6: {ID: 6, UID: "bob", Email: "bob@example.com", DisplayName: "Bob"},
		}},
		git: &svcGit{
			commits: map[string]git.Commit{
				svcSHAOne: {SHA: sha.Must(svcSHAOne), Title: "first", Message: "first commit"},
				svcSHATwo: {SHA: sha.Must(svcSHATwo), Title: "second", Message: "second commit"},
			},
			listResult: []git.Commit{
				{SHA: sha.Must(svcSHATwo), Title: "second", Message: "second commit"},
				{SHA: sha.Must(svcSHAOne), Title: "first", Message: "first commit"},
			},
			total: 2,
		},
		activity: &svcActivityStore{activities: map[int64]*types.PullReqActivity{
			200: {ID: 200, Text: "a comment", Kind: enum.PullReqActivityKindComment, Created: 1, Updated: 2},
		}},
		labels: &svcLabelStore{labels: map[int64]*types.Label{3: {ID: 3, Key: "bug"}}},
		values: &svcLabelValueStore{values: map[int64]*types.LabelValue{4: {ID: 4, Value: "high"}}},
		groups: &svcUserGroupStore{groups: map[int64]*types.UserGroup{
			8: {ID: 8, Identifier: "devs", Name: "Developers"},
		}},
		spaces:  &svcSpaceStore{ancestors: map[int64][]int64{10: {10, 11}}},
		sse:     &svcSSE{},
		audit:   &svcAudit{},
		hookURL: srv.URL,
	}

	cfg := Config{
		UserAgentIdentity: "Svctest",
		HeaderIdentity:    "Svctest",
		AllowLoopback:     true,
		EventReaderName:   "reader",
		Concurrency:       1,
	}
	execStore := &GitnessWebhookExecutorStore{webhookStore: h.hooks, webhookExecutionStore: h.execs}
	executor := NewWebhookExecutor(cfg, &GitnessURLProvider{}, svcEncrypter{}, svcSpacePathStore{},
		svcSecretService{}, h.principal, execStore, RepoTrigger)

	h.svc = &Service{
		WebhookExecutor:       executor,
		webhookStore:          h.hooks,
		webhookExecutionStore: h.execs,
		urlProvider:           &svcURLs{},
		spaceStore:            h.spaces,
		repoStore:             h.repos,
		pullreqStore:          h.prs,
		principalStore:        h.principal,
		git:                   h.git,
		activityStore:         h.activity,
		labelStore:            h.labels,
		labelValueStore:       h.values,
		encrypter:             svcEncrypter{},
		config:                cfg,
		auditService:          h.audit,
		sseStreamer:           h.sse,
		userGroupStore:        h.groups,
	}
	return h
}

// addRepoHook registers an enabled webhook on repo 1 pointing at the receiver.
func (h *svcHarness) addRepoHook(triggers ...enum.WebhookTrigger) *types.Webhook {
	return h.hooks.add(&types.Webhook{
		ParentID: 1, ParentType: enum.WebhookParentRepo, Identifier: "hook", DisplayName: "hook",
		URL: h.hookURL, Enabled: true, Triggers: triggers, Type: enum.WebhookTypeExternal,
	})
}

// svcURLs is a url.Provider fake supporting the methods used for payload building.
type svcURLs struct {
	url.Provider
}

func (*svcURLs) GenerateUIRepoURL(_ context.Context, p string) string { return "https://ui.test/" + p }
func (*svcURLs) GenerateGITCloneURL(_ context.Context, p string) string {
	return "https://git.test/" + p + ".git"
}
func (*svcURLs) GenerateGITCloneSSHURL(_ context.Context, p string) string {
	return "ssh://git.test/" + p + ".git"
}
func (*svcURLs) GenerateUIPRURL(_ context.Context, p string, n int64) string {
	return fmt.Sprintf("https://ui.test/%s/pulls/%d", p, n)
}
func (*svcURLs) GenerateUIRefURL(_ context.Context, p, ref string) string {
	return "https://ui.test/" + p + "/commit/" + ref
}
