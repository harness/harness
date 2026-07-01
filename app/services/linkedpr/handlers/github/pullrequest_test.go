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

package github_test

import (
	"context"
	"errors"
	"testing"

	pullreqevents "github.com/harness/gitness/app/events/pullreq"
	"github.com/harness/gitness/app/services/importer"
	"github.com/harness/gitness/app/services/linkedpr"
	"github.com/harness/gitness/app/services/linkedpr/handlers/github"
	"github.com/harness/gitness/app/services/refcache"
	"github.com/harness/gitness/app/store"
	storecache "github.com/harness/gitness/app/store/cache"
	gitnessurl "github.com/harness/gitness/app/url"
	gitpkg "github.com/harness/gitness/git"
	"github.com/harness/gitness/git/sha"
	mockgit "github.com/harness/gitness/mocks/git"
	mockpullreq "github.com/harness/gitness/mocks/pullreq"
	mockstore "github.com/harness/gitness/mocks/store"
	gitness_store "github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/mock"

	_ "unsafe" // for go:linkname
)

// bootstrapSystemServicePrincipal aliases the unexported package-level principal
// used by bootstrap.NewSystemServiceSession() inside handlers.RunSyncRefs.
//
//go:linkname bootstrapSystemServicePrincipal github.com/harness/gitness/app/bootstrap.systemServicePrincipal
var bootstrapSystemServicePrincipal *types.Principal

func init() {
	bootstrapSystemServicePrincipal = &types.Principal{
		ID:          1,
		UID:         "harness-test",
		DisplayName: "Harness Test",
		Email:       "harness-test@local",
	}
}

// Dedup key is (provider, repoProviderID, prNumber); repoProviderID is
// the matched LinkedRepo's upstream SCM id, not a PR field.
const (
	testRepoProviderID = "280125018"
	testPRNumber       = 7
	testRepoID         = int64(100)

	testPRTitle  = "feat: x"
	testHeadSHA  = "aaa"
	testBaseRef  = "main"
	testBaseSHA  = "bbb"
	testOldSHA   = "old-sha"
	testActorLog = "octocat"
)

// callHandle invokes PullRequestHandler.Handle with the typed payload
// extracted from ev.Payload, mirroring what SafeHandler does in production.
func callHandle(t *testing.T, h *github.PullRequestHandler, ev *linkedpr.Event, repo *types.LinkedRepo) error {
	t.Helper()
	pl, ok := ev.Payload.(linkedpr.PullRequestPayload)
	if !ok {
		t.Fatalf("test event payload is not PullRequestPayload: %T", ev.Payload)
	}
	return h.Handle(context.Background(), ev, pl, repo)
}

// ─── mock harnesses ───────────────────────────────────────────────────────

type pullReqStoreHarness struct {
	*mockstore.PullReqStore
	CreatedRows []*types.PullReq
	UpdatedRows []*types.PullReq
}

type pullReqStoreConfig struct {
	findFn    func(ctx context.Context, id int64) (*types.PullReq, error)
	createErr error
}

func mockArgPullReq(args mock.Arguments) *types.PullReq {
	pr, ok := args.Get(0).(*types.PullReq)
	if !ok {
		panic("mock: arg 0 is not *types.PullReq")
	}
	return pr
}

func mockArgLinkedPullReq(args mock.Arguments) *types.LinkedPullReq {
	lpr, ok := args.Get(0).(*types.LinkedPullReq)
	if !ok {
		panic("mock: arg 0 is not *types.LinkedPullReq")
	}
	return lpr
}

func newPullReqStoreHarness(cfg pullReqStoreConfig) *pullReqStoreHarness {
	h := &pullReqStoreHarness{PullReqStore: &mockstore.PullReqStore{}}
	h.On("Create", mock.Anything).Run(func(args mock.Arguments) {
		if cfg.createErr != nil {
			return
		}
		pr := mockArgPullReq(args)
		pr.ID = int64(len(h.CreatedRows) + 1)
		h.CreatedRows = append(h.CreatedRows, pr)
	}).Return(cfg.createErr)
	if cfg.findFn != nil {
		h.On("Find", mock.Anything).Return(func(id int64) (*types.PullReq, error) {
			return cfg.findFn(context.Background(), id)
		})
		h.On("Update", mock.Anything).Run(func(args mock.Arguments) {
			h.UpdatedRows = append(h.UpdatedRows, mockArgPullReq(args))
		}).Return(nil)
	}
	return h
}

type linkedPullReqStoreHarness struct {
	*mockstore.LinkedPullReqStore
	CreatedRows []*types.LinkedPullReq
	UpdatedRows []*types.LinkedPullReq
}

type linkedPullReqStoreConfig struct {
	findByFn func(
		ctx context.Context,
		linkedRepoID int64,
		provider, providerID string,
		prNumber int,
	) (*types.LinkedPullReq, error)
	createErr error
}

func newLinkedPullReqStoreHarness(cfg linkedPullReqStoreConfig) *linkedPullReqStoreHarness {
	h := &linkedPullReqStoreHarness{LinkedPullReqStore: &mockstore.LinkedPullReqStore{}}
	h.On("FindByLinkedRepoAndProviderPR", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(func(linkedRepoID int64, provider, providerID string, prNumber int) (*types.LinkedPullReq, error) {
			if cfg.findByFn == nil {
				return nil, gitness_store.ErrResourceNotFound
			}
			return cfg.findByFn(context.Background(), linkedRepoID, provider, providerID, prNumber)
		})
	h.On("Create", mock.Anything).Run(func(args mock.Arguments) {
		if cfg.createErr != nil {
			return
		}
		h.CreatedRows = append(h.CreatedRows, mockArgLinkedPullReq(args))
	}).Return(cfg.createErr)
	h.On("Update", mock.Anything).Run(func(args mock.Arguments) {
		h.UpdatedRows = append(h.UpdatedRows, mockArgLinkedPullReq(args))
	}).Return(nil)
	return h
}

func newActivityStoreMock(t *testing.T) (*mockstore.PullReqActivityStore, *[]*types.PullReqActivity) {
	t.Helper()
	rows := &[]*types.PullReqActivity{}
	m := &mockstore.PullReqActivityStore{}
	m.On("CreateWithPayload", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			pr := mockArgPullReq(args)
			principalID, ok := args.Get(1).(int64)
			if !ok {
				panic("mock: arg 1 is not int64")
			}
			payload, ok := args.Get(2).(types.PullReqActivityPayload)
			if !ok {
				panic("mock: arg 2 is not PullReqActivityPayload")
			}
			act := &types.PullReqActivity{
				CreatedBy: principalID, PullReqID: pr.ID, RepoID: pr.TargetRepoID,
				Order: pr.ActivitySeq, Type: payload.ActivityType(),
			}
			_ = act.SetPayload(payload)
			*rows = append(*rows, act)
		}).
		Return(&types.PullReqActivity{}, nil)
	return m, rows
}

func testReporter(t *testing.T) *pullreqevents.Reporter {
	t.Helper()
	return mockpullreq.NewStubReporter(t)
}

func testActivityStore(t *testing.T) store.PullReqActivityStore {
	t.Helper()
	m, _ := newActivityStoreMock(t)
	return m
}

// stubMergeBaseSHA is a deterministic merge-base returned by the git mock.
const stubMergeBaseSHA = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

type testURLProvider struct{}

func (testURLProvider) GetInternalAPIURL(context.Context) string {
	return "http://localhost:3000/api"
}
func (testURLProvider) GenerateContainerGITCloneURL(context.Context, string) string { return "" }
func (testURLProvider) GenerateGITCloneURL(context.Context, string) string          { return "" }
func (testURLProvider) GenerateGITCloneSSHURL(context.Context, string) string       { return "" }
func (testURLProvider) GenerateUIRepoURL(context.Context, string) string            { return "" }
func (testURLProvider) GenerateUIPRURL(context.Context, string, int64) string       { return "" }
func (testURLProvider) GenerateUICompareURL(context.Context, string, string, string) string {
	return ""
}
func (testURLProvider) GenerateUIRefURL(context.Context, string, string) string { return "" }
func (testURLProvider) GetAPIHostname(context.Context) string                   { return "localhost" }
func (testURLProvider) GenerateUIBuildURL(context.Context, string, string, int64) string {
	return ""
}
func (testURLProvider) GetGITHostname(context.Context) string { return "localhost" }
func (testURLProvider) GetAPIProto(context.Context) string    { return "http" }
func (testURLProvider) RegistryURL(context.Context, ...string) string {
	return ""
}
func (testURLProvider) PackageURL(context.Context, string, string, ...string) string { return "" }
func (testURLProvider) GetUIBaseURL(context.Context, ...string) string               { return "" }
func (testURLProvider) PackagePathFor(context.Context, gitnessurl.PackagePathSpec) (string, error) {
	return "", nil
}
func (testURLProvider) GenerateUIRegistryURL(context.Context, string, string) string { return "" }

type testConnectorService struct{}

func (testConnectorService) GetAccessInfo(context.Context, importer.ConnectorDef) (importer.AccessInfo, error) {
	return importer.AccessInfo{URL: "https://github.com/acme/widget.git"}, nil
}
func (testConnectorService) FetchProviderRepoInfo(
	_ context.Context, _ importer.ConnectorDef,
) (importer.ProviderRepoInfo, error) {
	return importer.ProviderRepoInfo{}, nil
}
func (testConnectorService) ResolveConnectorRef(_, ref string) (string, string) { return "", ref }
func (testConnectorService) EncodeConnectorRef(_, _, identifier string) string  { return identifier }

type repoIDCacheStub struct {
	repo *types.RepositoryCore
}

func (s *repoIDCacheStub) Stats() (int64, int64)            { return 0, 0 }
func (s *repoIDCacheStub) Evict(_ context.Context, _ int64) {}
func (s *repoIDCacheStub) Get(_ context.Context, _ int64) (*types.RepositoryCore, error) {
	if s.repo == nil {
		return nil, errors.New("repo not found")
	}
	return s.repo, nil
}

type spacePathCacheStub struct{}

func (s *spacePathCacheStub) Stats() (int64, int64)             { return 0, 0 }
func (s *spacePathCacheStub) Evict(_ context.Context, _ string) {}
func (s *spacePathCacheStub) Get(_ context.Context, _ string) (*types.SpacePath, error) {
	return nil, errors.New("not used in this test")
}

type repoRefCacheStub struct{}

func (s *repoRefCacheStub) Stats() (int64, int64)                         { return 0, 0 }
func (s *repoRefCacheStub) Evict(_ context.Context, _ types.RepoCacheKey) {}
func (s *repoRefCacheStub) Get(_ context.Context, _ types.RepoCacheKey) (int64, error) {
	return 0, errors.New("not used in this test")
}

func testRepoFinder(repo *types.RepositoryCore) refcache.RepoFinder {
	return refcache.NewRepoFinder(
		nil,
		&spacePathCacheStub{},
		&repoIDCacheStub{repo: repo},
		&repoRefCacheStub{},
		storecache.Evictor[*types.RepositoryCore]{},
	)
}

func configureGitMock(m *mockgit.Interface, opts ...func(*mockgit.Interface)) {
	m.On("SyncRefs", mock.Anything, mock.Anything).Return(&gitpkg.SyncRefsOutput{}, nil)
	m.On("MergeBase", mock.Anything, mock.Anything).Return(
		gitpkg.MergeBaseOutput{MergeBaseSHA: sha.Must(stubMergeBaseSHA)}, nil)
	// GetRef + UpdateRefs are called by RunSyncRefs to rename provider refs
	// (refs/pull/<N>/head → refs/pullreq/<N>/head) after the initial fetch.
	m.On("GetRef", mock.Anything, mock.Anything).Return(
		gitpkg.GetRefResponse{SHA: sha.Must(stubMergeBaseSHA)}, nil)
	m.On("UpdateRefs", mock.Anything, mock.Anything).Return(nil)
	for _, opt := range opts {
		opt(m)
	}
}

func newPullRequestHandler(
	t *testing.T,
	prStore store.PullReqStore,
	linkedStore store.LinkedPullReqStore,
	activity store.PullReqActivityStore,
	author linkedpr.AuthorResolver,
	reporter *pullreqevents.Reporter,
	tx noopTx,
	gitOpts ...func(*mockgit.Interface),
) *github.PullRequestHandler {
	t.Helper()
	gitMock := &mockgit.Interface{}
	configureGitMock(gitMock, gitOpts...)
	repo := &types.RepositoryCore{ID: testRepoID, GitUID: "git-uid-test"}
	return github.NewPullRequestHandler(
		prStore, linkedStore, activity, author, reporter,
		gitMock, testRepoFinder(repo), testURLProvider{}, testConnectorService{}, tx,
	)
}

// noopTx executes txFn directly without any transaction semantics. The store
// mocks don't care about the in-context tx so this works for unit tests.
type noopTx struct{}

func (noopTx) WithTx(ctx context.Context, txFn func(context.Context) error, _ ...any) error {
	return txFn(ctx)
}

// ─── fixtures ─────────────────────────────────────────────────────────────

// basePayload returns a typical "open PR" payload; tests mutate per case.
func basePayload(mutators ...func(*linkedpr.PullRequestPayload)) linkedpr.PullRequestPayload {
	p := linkedpr.PullRequestPayload{
		Number:      testPRNumber,
		Title:       testPRTitle,
		Description: "",
		HeadRef:     "feat/x",
		HeadSHA:     testHeadSHA,
		BaseRef:     testBaseRef,
		BaseSHA:     testBaseSHA,
		State:       enum.PullReqStateOpen,
		Draft:       false,
		CreatedAt:   1_700_000_000_000,
		UpdatedAt:   1_700_000_100_000,
		HTMLURL:     "https://github.com/acme/widget/pull/7",
		Author: linkedpr.User{
			Login:   testActorLog,
			Avatar:  "https://avatars/x.png",
			HTMLURL: "https://github.com/octocat",
		},
		Repository: linkedpr.Repository{
			ProviderID: testRepoProviderID,
		},
	}
	for _, m := range mutators {
		m(&p)
	}
	return p
}

func eventWith(p linkedpr.PullRequestPayload) *linkedpr.Event {
	return &linkedpr.Event{
		Provider:   linkedpr.ProviderGitHub,
		AccountID:  "acct-1",
		DeliveryID: "deliv-1",
		Payload:    p,
	}
}

func testLinkedRepo() *types.LinkedRepo {
	return &types.LinkedRepo{
		RepoID:         testRepoID,
		ProviderType:   string(linkedpr.ProviderGitHub),
		ProviderRepoID: testRepoProviderID,
	}
}

// findNotFound stubs "no existing row" and asserts the dedup key.
func findNotFound(t *testing.T) func(context.Context, int64, string, string, int) (*types.LinkedPullReq, error) {
	t.Helper()
	return func(
		_ context.Context, linkedRepoID int64, provider, providerID string, prNumber int,
	) (*types.LinkedPullReq, error) {
		assertDedupKey(t, linkedRepoID, provider, providerID, prNumber)
		return nil, nil
	}
}

func findExisting(
	t *testing.T,
	row *types.LinkedPullReq,
) func(context.Context, int64, string, string, int) (*types.LinkedPullReq, error) {
	t.Helper()
	return func(
		_ context.Context, linkedRepoID int64, provider, providerID string, prNumber int,
	) (*types.LinkedPullReq, error) {
		assertDedupKey(t, linkedRepoID, provider, providerID, prNumber)
		return row, nil
	}
}

func assertDedupKey(t *testing.T, linkedRepoID int64, provider, providerID string, prNumber int) {
	t.Helper()
	if linkedRepoID != testRepoID {
		t.Errorf("FindByLinkedRepoAndProviderPR linkedRepoID: got %d want %d", linkedRepoID, testRepoID)
	}
	if provider != string(linkedpr.ProviderGitHub) {
		t.Errorf("FindByLinkedRepoAndProviderPR provider: got %q want %q",
			provider, string(linkedpr.ProviderGitHub))
	}
	if providerID != testRepoProviderID {
		t.Errorf("FindByLinkedRepoAndProviderPR providerID: got %q want repo's provider id %q",
			providerID, testRepoProviderID)
	}
	if prNumber != testPRNumber {
		t.Errorf("FindByLinkedRepoAndProviderPR prNumber: got %d want %d", prNumber, testPRNumber)
	}
}

// ─── tests ────────────────────────────────────────────────────────────────

// TestHandle_Create_Opened: unseen PR → parent + linked rows; linked
// row's ProviderID mirrors LinkedRepo.ProviderID.
func TestHandle_Create_Opened(t *testing.T) {
	prStore := newPullReqStoreHarness(pullReqStoreConfig{})
	linkedStore := newLinkedPullReqStoreHarness(linkedPullReqStoreConfig{findByFn: findNotFound(t)})

	h := newPullRequestHandler(t, prStore, linkedStore, testActivityStore(t),
		&linkedpr.SystemPrincipalResolver{PrincipalID: 42}, testReporter(t), noopTx{})

	if err := callHandle(t, h, eventWith(basePayload()), testLinkedRepo()); err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if len(prStore.CreatedRows) != 1 {
		t.Fatalf("expected 1 parent row, got %d", len(prStore.CreatedRows))
	}
	parent := prStore.CreatedRows[0]
	if parent.Number != testPRNumber || parent.CreatedBy != 42 || parent.TargetRepoID != testRepoID {
		t.Errorf("parent fields wrong: %+v", parent)
	}
	if parent.State != enum.PullReqStateOpen {
		t.Errorf("parent State: got %v want Open", parent.State)
	}
	if parent.Type == nil || *parent.Type != enum.PullReqTypeLinked {
		t.Errorf("parent Type: got %v want Linked", parent.Type)
	}

	if len(linkedStore.CreatedRows) != 1 {
		t.Fatalf("expected 1 linked row, got %d", len(linkedStore.CreatedRows))
	}
	linked := linkedStore.CreatedRows[0]
	if linked.ProviderRepoID != testRepoProviderID {
		t.Errorf("linked.ProviderRepoID: got %q want repo's provider id %q",
			linked.ProviderRepoID, testRepoProviderID)
	}
	if linked.ProviderAuthorLogin != testActorLog {
		t.Errorf("linked.ProviderAuthorLogin: got %q want %q", linked.ProviderAuthorLogin, testActorLog)
	}
	if linked.ProviderType != string(linkedpr.ProviderGitHub) {
		t.Errorf("linked ProviderType: got %q want github", linked.ProviderType)
	}
}

func runUpdateTest(
	t *testing.T,
	payloadFn func() linkedpr.PullRequestPayload,
	prevState enum.PullReqState,
	want func(*types.PullReq),
) {
	t.Helper()
	prType := enum.PullReqTypeLinked
	parentRow := &types.PullReq{
		ID:           50,
		Number:       testPRNumber,
		State:        prevState,
		Type:         &prType,
		TargetRepoID: testRepoID,
		SourceSHA:    testOldSHA,
		Updated:      1_700_000_050_000, // older than fixture's UpdatedAt
	}
	prStore := newPullReqStoreHarness(pullReqStoreConfig{
		findFn: func(_ context.Context, _ int64) (*types.PullReq, error) { return parentRow, nil },
	})
	linkedStore := newLinkedPullReqStoreHarness(linkedPullReqStoreConfig{
		findByFn: findExisting(t, &types.LinkedPullReq{
			PullReqID:      50,
			ProviderType:   string(linkedpr.ProviderGitHub),
			ProviderRepoID: testRepoProviderID,
		}),
	})

	h := newPullRequestHandler(t, prStore, linkedStore, testActivityStore(t),
		&linkedpr.SystemPrincipalResolver{PrincipalID: 42}, testReporter(t), noopTx{})

	if err := callHandle(t, h, eventWith(payloadFn()), testLinkedRepo()); err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if len(prStore.UpdatedRows) != 1 {
		t.Fatalf("expected 1 parent update, got %d", len(prStore.UpdatedRows))
	}
	want(prStore.UpdatedRows[0])
}

func TestHandle_Update_Synchronize(t *testing.T) {
	runUpdateTest(t, func() linkedpr.PullRequestPayload {
		return basePayload(func(p *linkedpr.PullRequestPayload) { p.HeadSHA = "new-sha" })
	}, enum.PullReqStateOpen, func(_ *types.PullReq) {})
}

// Parsed proto has no closed_at/merged_at → parent.Closed falls back to UpdatedAt.
func TestHandle_Create_MergedFallsBackToUpdatedAt(t *testing.T) {
	prStore := newPullReqStoreHarness(pullReqStoreConfig{})
	linkedStore := newLinkedPullReqStoreHarness(linkedPullReqStoreConfig{findByFn: findNotFound(t)})
	h := newPullRequestHandler(t, prStore, linkedStore, testActivityStore(t),
		&linkedpr.SystemPrincipalResolver{PrincipalID: 42}, testReporter(t), noopTx{})

	pl := basePayload(func(p *linkedpr.PullRequestPayload) {
		p.State = enum.PullReqStateMerged
		p.UpdatedAt = 1_700_000_200_000
		p.Sender = linkedpr.User{Login: testActorLog}
	})
	if err := callHandle(t, h, eventWith(pl), testLinkedRepo()); err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if len(prStore.CreatedRows) != 1 {
		t.Fatalf("expected 1 parent row, got %d", len(prStore.CreatedRows))
	}
	parent := prStore.CreatedRows[0]
	if parent.State != enum.PullReqStateMerged {
		t.Errorf("parent State: got %v want Merged", parent.State)
	}
	if parent.Closed == nil {
		t.Fatal("parent.Closed must not be nil for a merged PR")
	}
	if *parent.Closed != 1_700_000_200_000 {
		t.Errorf("parent.Closed: got %d, want UpdatedAt fallback 1700000200000", *parent.Closed)
	}
	if parent.Merged == nil || *parent.Merged != 1_700_000_200_000 {
		t.Errorf("parent.Merged: got %v, want 1700000200000", parent.Merged)
	}
	if len(linkedStore.CreatedRows) != 1 {
		t.Fatalf("expected 1 linked row, got %d", len(linkedStore.CreatedRows))
	}
	linked := linkedStore.CreatedRows[0]
	if linked.MergerLogin != testActorLog {
		t.Errorf("linked.MergerLogin: got %q, want %q", linked.MergerLogin, testActorLog)
	}
}

// open → merged transition must populate parent.Closed (using UpdatedAt).
func TestHandle_Update_MergeTransitionFillsClosed(t *testing.T) {
	prType := enum.PullReqTypeLinked
	parentRow := &types.PullReq{
		ID: 50, Number: testPRNumber, State: enum.PullReqStateOpen, Type: &prType,
		TargetRepoID: testRepoID, SourceSHA: testHeadSHA, Title: testPRTitle,
		Updated: 1_700_000_050_000,
	}
	prStore := newPullReqStoreHarness(pullReqStoreConfig{
		findFn: func(_ context.Context, _ int64) (*types.PullReq, error) { return parentRow, nil },
	})
	linkedStore := newLinkedPullReqStoreHarness(linkedPullReqStoreConfig{
		findByFn: findExisting(t, &types.LinkedPullReq{
			PullReqID: 50, ProviderType: string(linkedpr.ProviderGitHub),
			ProviderRepoID: testRepoProviderID,
		}),
	})
	h := newPullRequestHandler(t, prStore, linkedStore, testActivityStore(t),
		&linkedpr.SystemPrincipalResolver{PrincipalID: 42}, testReporter(t), noopTx{})

	pl := basePayload(func(p *linkedpr.PullRequestPayload) {
		p.State = enum.PullReqStateMerged
		p.UpdatedAt = 1_700_000_200_000
		p.Sender = linkedpr.User{Login: "merger-bot"}
	})
	if err := callHandle(t, h, eventWith(pl), testLinkedRepo()); err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if len(prStore.UpdatedRows) != 1 {
		t.Fatalf("expected 1 parent update, got %d", len(prStore.UpdatedRows))
	}
	got := prStore.UpdatedRows[0]
	if got.State != enum.PullReqStateMerged {
		t.Errorf("parent State: got %v want Merged", got.State)
	}
	if got.Closed == nil {
		t.Fatal("parent.Closed must not be nil after merge transition")
	}
	if *got.Closed != 1_700_000_200_000 {
		t.Errorf("parent.Closed: got %d, want UpdatedAt fallback 1700000200000", *got.Closed)
	}
	if got.Merged == nil || *got.Merged != 1_700_000_200_000 {
		t.Errorf("parent.Merged: got %v, want 1700000200000", got.Merged)
	}
	if len(linkedStore.UpdatedRows) != 1 {
		t.Fatalf("expected 1 linked update, got %d", len(linkedStore.UpdatedRows))
	}
	linked := linkedStore.UpdatedRows[0]
	if linked.MergerLogin != "merger-bot" {
		t.Errorf("linked.MergerLogin: got %q, want merger-bot", linked.MergerLogin)
	}
}

func TestHandle_Update_Closed(t *testing.T) {
	runUpdateTest(t, func() linkedpr.PullRequestPayload {
		return basePayload(func(p *linkedpr.PullRequestPayload) {
			p.State = enum.PullReqStateClosed
			p.UpdatedAt = 1_700_000_200_000
		})
	}, enum.PullReqStateOpen, func(parent *types.PullReq) {
		if parent.State != enum.PullReqStateClosed {
			t.Errorf("parent State: got %v want Closed", parent.State)
		}
	})
}

func TestHandle_Update_Merged(t *testing.T) {
	runUpdateTest(t, func() linkedpr.PullRequestPayload {
		return basePayload(func(p *linkedpr.PullRequestPayload) {
			p.State = enum.PullReqStateMerged
			p.UpdatedAt = 1_700_000_200_000
		})
	}, enum.PullReqStateOpen, func(parent *types.PullReq) {
		if parent.State != enum.PullReqStateMerged {
			t.Errorf("parent State: got %v want Merged", parent.State)
		}
	})
}

func TestHandle_Update_Reopened(t *testing.T) {
	runUpdateTest(t, func() linkedpr.PullRequestPayload {
		return basePayload() // open again
	}, enum.PullReqStateClosed, func(parent *types.PullReq) {
		if parent.State != enum.PullReqStateOpen {
			t.Errorf("parent State: got %v want Open", parent.State)
		}
	})
}

// Reopening a previously merged PR must clear both parent.Closed and
// parent.Merged so the UI no longer shows the stale "closed/merged N ago".
func TestHandle_Update_ReopenClearsClosedAndMerged(t *testing.T) {
	prType := enum.PullReqTypeLinked
	staleClosed := int64(1_700_000_000_000)
	staleMerged := int64(1_700_000_000_000)
	parentRow := &types.PullReq{
		ID: 50, Number: testPRNumber, State: enum.PullReqStateMerged, Type: &prType,
		TargetRepoID: testRepoID, SourceSHA: testHeadSHA, Title: testPRTitle,
		Closed: &staleClosed, Merged: &staleMerged,
		Updated: 1_700_000_050_000,
	}
	prStore := newPullReqStoreHarness(pullReqStoreConfig{
		findFn: func(_ context.Context, _ int64) (*types.PullReq, error) { return parentRow, nil },
	})
	linkedStore := newLinkedPullReqStoreHarness(linkedPullReqStoreConfig{
		findByFn: findExisting(t, &types.LinkedPullReq{
			PullReqID: 50, ProviderType: string(linkedpr.ProviderGitHub),
			ProviderRepoID: testRepoProviderID,
		}),
	})
	h := newPullRequestHandler(t, prStore, linkedStore, testActivityStore(t),
		&linkedpr.SystemPrincipalResolver{PrincipalID: 42}, testReporter(t), noopTx{})

	pl := basePayload(func(p *linkedpr.PullRequestPayload) {
		p.State = enum.PullReqStateOpen
		p.UpdatedAt = 1_700_000_300_000
	})
	if err := callHandle(t, h, eventWith(pl), testLinkedRepo()); err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if len(prStore.UpdatedRows) != 1 {
		t.Fatalf("expected 1 parent update, got %d", len(prStore.UpdatedRows))
	}
	got := prStore.UpdatedRows[0]
	if got.State != enum.PullReqStateOpen {
		t.Errorf("parent State: got %v want Open", got.State)
	}
	if got.Closed != nil {
		t.Errorf("parent.Closed must be cleared on reopen, got %v", got.Closed)
	}
	if got.Merged != nil {
		t.Errorf("parent.Merged must be cleared on reopen, got %v", got.Merged)
	}
}

func TestHandle_AuthorResolverError(t *testing.T) {
	prStore := newPullReqStoreHarness(pullReqStoreConfig{})
	linkedStore := newLinkedPullReqStoreHarness(linkedPullReqStoreConfig{findByFn: findNotFound(t)})
	failingResolver := &failResolver{}

	h := newPullRequestHandler(t, prStore, linkedStore, testActivityStore(t), failingResolver, testReporter(t), noopTx{})
	if err := callHandle(t, h, eventWith(basePayload()), testLinkedRepo()); err == nil {
		t.Fatal("expected error when author resolver fails")
	}
}

type failResolver struct{}

func (*failResolver) Resolve(context.Context, *linkedpr.Event) (int64, error) {
	return 0, errors.New("resolver down")
}

func TestHandle_SyncRefsError_PropagatesForRetry(t *testing.T) {
	h := newPullRequestHandler(t,
		newPullReqStoreHarness(pullReqStoreConfig{}),
		newLinkedPullReqStoreHarness(linkedPullReqStoreConfig{findByFn: findNotFound(t)}),
		testActivityStore(t),
		&linkedpr.SystemPrincipalResolver{PrincipalID: 42},
		testReporter(t), noopTx{},
		func(m *mockgit.Interface) {
			m.ExpectedCalls = nil
			m.On("SyncRefs", mock.Anything, mock.Anything).
				Return((*gitpkg.SyncRefsOutput)(nil), errors.New("git: temporary failure"))
		},
	)
	if err := callHandle(t, h, eventWith(basePayload()), testLinkedRepo()); err == nil {
		t.Errorf("expected error so consumer doesn't ack")
	}
}

// TestHandle_Update_SkipsSyncRefsWhenSourceSHAUnchanged: pure-metadata updates
// (title / labels / state) must not trigger a refs fetch or a merge-base
// recompute.
func TestHandle_Update_SkipsSyncRefsWhenSourceSHAUnchanged(t *testing.T) {
	prType := enum.PullReqTypeLinked
	baseSHA := testBaseSHA
	parentRow := &types.PullReq{
		ID: 50, Number: testPRNumber, State: enum.PullReqStateOpen, Type: &prType,
		TargetRepoID:   testRepoID,
		SourceSHA:      testHeadSHA,
		TargetBranch:   testBaseRef, // matches basePayload's BaseRef
		MergeTargetSHA: &baseSHA,    // matches basePayload's BaseSHA
		MergeBaseSHA:   "prev-mb",   // reused as-is when refs are skipped
		Updated:        1_700_000_050_000,
	}
	prStore := newPullReqStoreHarness(pullReqStoreConfig{
		findFn: func(_ context.Context, _ int64) (*types.PullReq, error) { return parentRow, nil },
	})
	linkedStore := newLinkedPullReqStoreHarness(linkedPullReqStoreConfig{
		findByFn: findExisting(t, &types.LinkedPullReq{
			PullReqID: 50, ProviderType: string(linkedpr.ProviderGitHub),
			ProviderRepoID: testRepoProviderID,
		}),
	})
	syncRefsCalls := 0
	h := newPullRequestHandler(t, prStore, linkedStore, testActivityStore(t),
		&linkedpr.SystemPrincipalResolver{PrincipalID: 42}, testReporter(t), noopTx{},
		func(m *mockgit.Interface) {
			m.ExpectedCalls = nil
			m.On("SyncRefs", mock.Anything, mock.Anything).Run(func(mock.Arguments) {
				syncRefsCalls++
			}).Return(&gitpkg.SyncRefsOutput{}, nil)
			m.On("MergeBase", mock.Anything, mock.Anything).Return(
				gitpkg.MergeBaseOutput{MergeBaseSHA: sha.Must(stubMergeBaseSHA)}, nil)
		},
	)
	if err := callHandle(t, h, eventWith(basePayload()), testLinkedRepo()); err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if syncRefsCalls != 0 {
		t.Errorf("syncRefs should be skipped when source SHA is unchanged, got %d calls", syncRefsCalls)
	}
}

// TestHandle_Update_SyncsRefsWhenSourceSHAChanged: a synchronize event MUST
// trigger a sync so the new SHA is locally present before persistence.
func TestHandle_Update_SyncsRefsWhenSourceSHAChanged(t *testing.T) {
	prType := enum.PullReqTypeLinked
	parentRow := &types.PullReq{
		ID: 50, Number: testPRNumber, State: enum.PullReqStateOpen, Type: &prType,
		TargetRepoID: testRepoID, SourceSHA: testOldSHA, Updated: 1_700_000_050_000,
	}
	prStore := newPullReqStoreHarness(pullReqStoreConfig{
		findFn: func(_ context.Context, _ int64) (*types.PullReq, error) { return parentRow, nil },
	})
	linkedStore := newLinkedPullReqStoreHarness(linkedPullReqStoreConfig{
		findByFn: findExisting(t, &types.LinkedPullReq{
			PullReqID: 50, ProviderType: string(linkedpr.ProviderGitHub),
			ProviderRepoID: testRepoProviderID,
		}),
	})
	syncRefsCalls := 0
	h := newPullRequestHandler(t, prStore, linkedStore, testActivityStore(t),
		&linkedpr.SystemPrincipalResolver{PrincipalID: 42}, testReporter(t), noopTx{},
		func(m *mockgit.Interface) {
			m.ExpectedCalls = nil
			m.On("SyncRefs", mock.Anything, mock.Anything).Run(func(mock.Arguments) {
				syncRefsCalls++
			}).Return(&gitpkg.SyncRefsOutput{}, nil)
			m.On("MergeBase", mock.Anything, mock.Anything).Return(
				gitpkg.MergeBaseOutput{MergeBaseSHA: sha.Must(stubMergeBaseSHA)}, nil)
			// RunSyncRefs renames refs/pull/<N>/head → refs/pullreq/<N>/head after fetching.
			m.On("GetRef", mock.Anything, mock.Anything).Return(
				gitpkg.GetRefResponse{SHA: sha.Must(stubMergeBaseSHA)}, nil)
			m.On("UpdateRefs", mock.Anything, mock.Anything).Return(nil)
		},
	)
	// payload has HeadSHA testHeadSHA by default; parent has testOldSHA → SHA moved.
	if err := callHandle(t, h, eventWith(basePayload()), testLinkedRepo()); err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if syncRefsCalls != 1 {
		t.Errorf("syncRefs should be called once when source SHA moved, got %d calls", syncRefsCalls)
	}
}

// TestHandle_Update_SHAChangeResetsMergeCheckStatus: when the head SHA
// moves the merge-check status is stale and must be reset to Unchecked so
// the merge-checker re-evaluates.
func TestHandle_Update_SHAChangeResetsMergeCheckStatus(t *testing.T) {
	prType := enum.PullReqTypeLinked
	parentRow := &types.PullReq{
		ID: 50, Number: testPRNumber, State: enum.PullReqStateOpen, Type: &prType,
		TargetRepoID:     testRepoID,
		SourceSHA:        testOldSHA, // != basePayload().HeadSHA (testHeadSHA) -> shaChanged
		MergeCheckStatus: enum.MergeCheckStatusMergeable,
		Updated:          1_700_000_050_000,
	}
	prStore := newPullReqStoreHarness(pullReqStoreConfig{
		findFn: func(_ context.Context, _ int64) (*types.PullReq, error) { return parentRow, nil },
	})
	linkedStore := newLinkedPullReqStoreHarness(linkedPullReqStoreConfig{
		findByFn: findExisting(t, &types.LinkedPullReq{
			PullReqID: 50, ProviderType: string(linkedpr.ProviderGitHub),
			ProviderRepoID: testRepoProviderID,
		}),
	})
	h := newPullRequestHandler(t, prStore, linkedStore, testActivityStore(t),
		&linkedpr.SystemPrincipalResolver{PrincipalID: 42}, testReporter(t), noopTx{})
	if err := callHandle(t, h, eventWith(basePayload()), testLinkedRepo()); err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if len(prStore.UpdatedRows) != 1 {
		t.Fatalf("expected 1 parent update, got %d", len(prStore.UpdatedRows))
	}
	if got := prStore.UpdatedRows[0].MergeCheckStatus; got != enum.MergeCheckStatusUnchecked {
		t.Errorf("MergeCheckStatus after SHA change: got %q, want Unchecked", got)
	}
}

// A base-only change (target branch advanced upstream) must NOT produce a
// "pushed a new commit" timeline row when the HEAD SHA didn't actually move.
// Stats reset and the downstream merge re-check still fire (gated on the
// broader shaChanged), but the activity row would be a confusing duplicate
// with Old == New.
func TestHandle_Update_BaseOnlyChangeDoesNotEmitBranchUpdateActivity(t *testing.T) {
	prType := enum.PullReqTypeLinked
	staleBaseSHA := "stale-base-sha-1234567890"
	parentRow := &types.PullReq{
		ID: 50, Number: testPRNumber, State: enum.PullReqStateOpen, Type: &prType,
		TargetRepoID: testRepoID,
		Title:        testPRTitle,
		// Head matches basePayload().HeadSHA -> headChanged = false.
		SourceSHA: testHeadSHA,
		// Base SHA is stale relative to basePayload().BaseSHA -> baseChanged = true.
		TargetBranch:   testBaseRef,
		MergeTargetSHA: &staleBaseSHA,
		Updated:        1_700_000_050_000,
	}
	prStore := newPullReqStoreHarness(pullReqStoreConfig{
		findFn: func(_ context.Context, _ int64) (*types.PullReq, error) { return parentRow, nil },
	})
	activityStore, activityRows := newActivityStoreMock(t)
	linkedStore := newLinkedPullReqStoreHarness(linkedPullReqStoreConfig{
		findByFn: findExisting(t, &types.LinkedPullReq{
			PullReqID: 50, ProviderType: string(linkedpr.ProviderGitHub),
			ProviderRepoID: testRepoProviderID,
		}),
	})
	h := newPullRequestHandler(t, prStore, linkedStore, activityStore,
		&linkedpr.SystemPrincipalResolver{PrincipalID: 42}, testReporter(t), noopTx{})
	if err := callHandle(t, h, eventWith(basePayload()), testLinkedRepo()); err != nil {
		t.Fatalf("Handle: %v", err)
	}
	for _, row := range *activityRows {
		if row.Type == enum.PullReqActivityTypeBranchUpdate {
			t.Errorf("base-only change must not emit a branch-update activity, got row: %+v", row)
		}
	}
}

// TestHandle_Update_PreservesMergeCheckStatusWithoutSHAChange: pure-metadata
// updates (e.g. label change) must NOT reset MergeCheckStatus.
func TestHandle_Update_PreservesMergeCheckStatusWithoutSHAChange(t *testing.T) {
	prType := enum.PullReqTypeLinked
	baseSHA := testBaseSHA
	parentRow := &types.PullReq{
		ID: 50, Number: testPRNumber, State: enum.PullReqStateOpen, Type: &prType,
		TargetRepoID:     testRepoID,
		Title:            testPRTitle,
		SourceSHA:        testHeadSHA, // matches basePayload().HeadSHA
		TargetBranch:     testBaseRef, // matches basePayload().BaseRef
		MergeTargetSHA:   &baseSHA,    // matches basePayload().BaseSHA
		MergeCheckStatus: enum.MergeCheckStatusMergeable,
		Updated:          1_700_000_050_000,
	}
	prStore := newPullReqStoreHarness(pullReqStoreConfig{
		findFn: func(_ context.Context, _ int64) (*types.PullReq, error) { return parentRow, nil },
	})
	linkedStore := newLinkedPullReqStoreHarness(linkedPullReqStoreConfig{
		findByFn: findExisting(t, &types.LinkedPullReq{
			PullReqID: 50, ProviderType: string(linkedpr.ProviderGitHub),
			ProviderRepoID: testRepoProviderID,
		}),
	})
	h := newPullRequestHandler(t, prStore, linkedStore, testActivityStore(t),
		&linkedpr.SystemPrincipalResolver{PrincipalID: 42}, testReporter(t), noopTx{})
	pl := basePayload(func(p *linkedpr.PullRequestPayload) { p.Draft = true })
	if err := callHandle(t, h, eventWith(pl), testLinkedRepo()); err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if len(prStore.UpdatedRows) != 1 {
		t.Fatalf("expected 1 parent update, got %d", len(prStore.UpdatedRows))
	}
	if got := prStore.UpdatedRows[0].MergeCheckStatus; got != enum.MergeCheckStatusMergeable {
		t.Errorf("MergeCheckStatus without SHA change: got %q, want Mergeable (unchanged)", got)
	}
}

// TestHandle_Update_DropsStaleWebhook: a payload with UpdatedAt older than
// the linked row's stored ProviderUpdatedAt must be dropped as out-of-order.
func TestHandle_Update_DropsStaleWebhook(t *testing.T) {
	prType := enum.PullReqTypeLinked
	parentRow := &types.PullReq{
		ID: 50, Number: testPRNumber, State: enum.PullReqStateOpen, Type: &prType,
		TargetRepoID: testRepoID, SourceSHA: testHeadSHA,
	}
	prStore := newPullReqStoreHarness(pullReqStoreConfig{
		findFn: func(_ context.Context, _ int64) (*types.PullReq, error) { return parentRow, nil },
	})
	linkedStore := newLinkedPullReqStoreHarness(linkedPullReqStoreConfig{
		findByFn: findExisting(t, &types.LinkedPullReq{
			PullReqID:      50,
			ProviderType:   string(linkedpr.ProviderGitHub),
			ProviderRepoID: testRepoProviderID,
			// Newer than fixture payload's UpdatedAt (1_700_000_100_000):
			// the incoming event predates the last applied one.
			ProviderUpdatedAt: 1_700_000_200_000,
		}),
	})
	h := newPullRequestHandler(t, prStore, linkedStore, testActivityStore(t),
		&linkedpr.SystemPrincipalResolver{PrincipalID: 42}, testReporter(t), noopTx{})
	if err := callHandle(t, h, eventWith(basePayload()), testLinkedRepo()); err != nil {
		t.Fatalf("Handle should swallow stale webhook, got: %v", err)
	}
	if len(prStore.UpdatedRows) != 0 {
		t.Errorf("stale webhook should not produce updates, got %d", len(prStore.UpdatedRows))
	}
}

func TestHandle_Create_ParentDuplicateAcksAndDrops(t *testing.T) {
	prStore := newPullReqStoreHarness(pullReqStoreConfig{createErr: gitness_store.ErrDuplicate})
	linkedStore := newLinkedPullReqStoreHarness(linkedPullReqStoreConfig{findByFn: findNotFound(t)})
	h := newPullRequestHandler(t, prStore, linkedStore, testActivityStore(t),
		&linkedpr.SystemPrincipalResolver{PrincipalID: 42}, testReporter(t), noopTx{})

	err := callHandle(t, h, eventWith(basePayload()), testLinkedRepo())
	if err != nil {
		t.Fatalf("Handle should swallow parent-duplicate, got: %v", err)
	}
	if len(linkedStore.CreatedRows) != 0 {
		t.Errorf("linked_pullreqs Create should not have run, got %d rows", len(linkedStore.CreatedRows))
	}
}

// TestHandle_Create_LinkedDuplicateDropsStaleLoser: when the create-race
// loser carries an older provider UpdatedAt than the winner just stored,
// handleLinkedDuplicate must drop the loser to avoid rolling back the
// stored high-water mark and re-applying out-of-order state.
func TestHandle_Create_LinkedDuplicateDropsStaleLoser(t *testing.T) {
	prType := enum.PullReqTypeLinked
	winner := &types.PullReq{
		ID: 99, Number: testPRNumber, State: enum.PullReqStateOpen, Type: &prType,
		TargetRepoID: testRepoID, SourceSHA: "newer-sha", Updated: 1_700_000_200_000,
	}
	prStore := newPullReqStoreHarness(pullReqStoreConfig{
		findFn: func(_ context.Context, _ int64) (*types.PullReq, error) {
			t.Fatal("pullReqStore.Find must NOT be called: loser should be dropped before parent lookup")
			return winner, nil
		},
	})
	findCalls := 0
	linkedStore := newLinkedPullReqStoreHarness(linkedPullReqStoreConfig{
		createErr: gitness_store.ErrDuplicate,
		findByFn: func(
			_ context.Context, linkedRepoID int64, provider, providerID string, prNumber int,
		) (*types.LinkedPullReq, error) {
			assertDedupKey(t, linkedRepoID, provider, providerID, prNumber)
			findCalls++
			if findCalls == 1 {
				return nil, gitness_store.ErrResourceNotFound
			}
			// Winner stored a fresher ProviderUpdatedAt than our loser carries.
			return &types.LinkedPullReq{
				PullReqID:         winner.ID,
				ProviderType:      string(linkedpr.ProviderGitHub),
				ProviderRepoID:    testRepoProviderID,
				ProviderUpdatedAt: 1_700_000_200_000,
			}, nil
		},
	})
	h := newPullRequestHandler(t, prStore, linkedStore, testActivityStore(t),
		&linkedpr.SystemPrincipalResolver{PrincipalID: 42}, testReporter(t), noopTx{})

	// Loser payload's UpdatedAt is 1_700_000_100_000 (basePayload default) —
	// older than the winner's 1_700_000_200_000.
	if err := callHandle(t, h, eventWith(basePayload()), testLinkedRepo()); err != nil {
		t.Fatalf("Handle should swallow stale duplicate, got: %v", err)
	}
	if len(prStore.UpdatedRows) != 0 {
		t.Errorf("stale loser should not update parent, got %d updates", len(prStore.UpdatedRows))
	}
	if len(linkedStore.UpdatedRows) != 0 {
		t.Errorf("stale loser should not update linked row, got %d updates", len(linkedStore.UpdatedRows))
	}
}

func TestHandle_Create_LinkedDuplicateFallsThroughToUpdate(t *testing.T) {
	prType := enum.PullReqTypeLinked
	winner := &types.PullReq{
		ID: 99, Number: testPRNumber, State: enum.PullReqStateOpen, Type: &prType,
		TargetRepoID: testRepoID, SourceSHA: testHeadSHA, Updated: 1_700_000_050_000,
	}
	findCalls := 0
	prStore := newPullReqStoreHarness(pullReqStoreConfig{
		findFn: func(_ context.Context, id int64) (*types.PullReq, error) {
			if id != winner.ID {
				t.Fatalf("Find called with unexpected id %d", id)
			}
			return winner, nil
		},
	})
	linkedStore := newLinkedPullReqStoreHarness(linkedPullReqStoreConfig{
		createErr: gitness_store.ErrDuplicate,
		findByFn: func(
			_ context.Context, linkedRepoID int64, provider, providerID string, prNumber int,
		) (*types.LinkedPullReq, error) {
			assertDedupKey(t, linkedRepoID, provider, providerID, prNumber)
			findCalls++
			if findCalls == 1 {
				return nil, gitness_store.ErrResourceNotFound
			}
			return &types.LinkedPullReq{
				PullReqID: winner.ID, ProviderType: string(linkedpr.ProviderGitHub),
				ProviderRepoID: testRepoProviderID,
			}, nil
		},
	})
	h := newPullRequestHandler(t, prStore, linkedStore, testActivityStore(t),
		&linkedpr.SystemPrincipalResolver{PrincipalID: 42}, testReporter(t), noopTx{})

	if err := callHandle(t, h, eventWith(basePayload()), testLinkedRepo()); err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if len(prStore.UpdatedRows) != 1 {
		t.Errorf("expected fall-through update on parent, got %d updates", len(prStore.UpdatedRows))
	}
	if len(linkedStore.UpdatedRows) != 1 {
		t.Errorf("expected fall-through update on linked, got %d updates", len(linkedStore.UpdatedRows))
	}
}
