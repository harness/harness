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

package githook

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/harness/gitness/app/api/controller/limiter"
	"github.com/harness/gitness/app/api/controller/service"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/bootstrap"
	checkevents "github.com/harness/gitness/app/events/check"
	gitevents "github.com/harness/gitness/app/events/git"
	mergequeueevents "github.com/harness/gitness/app/events/mergequeue"
	repoevents "github.com/harness/gitness/app/events/repo"
	"github.com/harness/gitness/app/services/locker"
	"github.com/harness/gitness/app/services/mergequeue"
	"github.com/harness/gitness/app/services/protection"
	"github.com/harness/gitness/app/services/settings"
	"github.com/harness/gitness/app/services/usergroup"
	"github.com/harness/gitness/app/sse"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/app/store/database"
	"github.com/harness/gitness/app/url"
	"github.com/harness/gitness/audit"
	"github.com/harness/gitness/events"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/git/api"
	"github.com/harness/gitness/git/hook"
	"github.com/harness/gitness/git/sha"
	"github.com/harness/gitness/job"
	"github.com/harness/gitness/lock"
	gitnessstore "github.com/harness/gitness/store"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/check"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/require"
)

var (
	errCxrBoom = errors.New("boom")

	cxrSHA1 = sha.Must("1111111111111111111111111111111111111111")
	cxrSHA2 = sha.Must("2222222222222222222222222222222222222222")
	cxrSHA3 = sha.Must("3333333333333333333333333333333333333333")
)

const (
	cxrRefMain    = "refs/heads/main"
	cxrRefFeature = "refs/heads/feature"
	cxrRefTag     = "refs/tags/v1"
)

// cxrGit is a configurable RestrictedGIT.
type cxrGit struct {
	ancestor      bool
	ancestorErr   error
	scanFindings  []git.ScanSecretsFinding
	scanErr       error
	scanCalls     []*git.ScanSecretsParams
	branch        *git.GetBranchOutput
	branchErr     error
	processOut    git.ProcessPreReceiveObjectsOutput
	processErr    error
	processParams []git.ProcessPreReceiveObjectsParams
	mergeBaseErr  error
}

func (g *cxrGit) IsAncestor(context.Context, git.IsAncestorParams) (git.IsAncestorOutput, error) {
	return git.IsAncestorOutput{Ancestor: g.ancestor}, g.ancestorErr
}

func (g *cxrGit) ScanSecrets(_ context.Context, p *git.ScanSecretsParams) (*git.ScanSecretsOutput, error) {
	g.scanCalls = append(g.scanCalls, p)
	if g.scanErr != nil {
		return nil, g.scanErr
	}
	return &git.ScanSecretsOutput{Findings: g.scanFindings}, nil
}

func (g *cxrGit) GetBranch(context.Context, *git.GetBranchParams) (*git.GetBranchOutput, error) {
	return g.branch, g.branchErr
}

func (g *cxrGit) Diff(
	context.Context, *git.DiffParams, ...api.FileDiffRequest,
) (<-chan *git.FileDiff, <-chan error) {
	return nil, nil
}

func (g *cxrGit) GetBlob(context.Context, *git.GetBlobParams) (*git.GetBlobOutput, error) {
	return nil, errCxrBoom
}

func (g *cxrGit) GetTreeNode(context.Context, *git.GetTreeNodeParams) (*git.GetTreeNodeOutput, error) {
	return nil, errCxrBoom
}

func (g *cxrGit) ProcessPreReceiveObjects(
	_ context.Context, p git.ProcessPreReceiveObjectsParams,
) (git.ProcessPreReceiveObjectsOutput, error) {
	g.processParams = append(g.processParams, p)
	// like the real implementation, only produce output for the requested checks.
	out := g.processOut
	if p.FindOversizeFilesParams == nil {
		out.FindOversizeFilesOutput = nil
	}
	if p.FindCommitterMismatchParams == nil {
		out.FindCommitterMismatchOutput = nil
	}
	if p.FindLFSPointersParams == nil {
		out.FindLFSPointersOutput = nil
	}
	return out, g.processErr
}

func (g *cxrGit) MergeBase(context.Context, git.MergeBaseParams) (git.MergeBaseOutput, error) {
	return git.MergeBaseOutput{}, g.mergeBaseErr
}

// cxrLimiter never limits.
type cxrLimiter struct{}

func (cxrLimiter) RepoCount(context.Context, int64, int) error   { return nil }
func (cxrLimiter) RepoSize(context.Context, int64) error         { return nil }
func (cxrLimiter) RootSpaceStorage(context.Context, int64) error { return nil }

var _ limiter.ResourceLimiter = cxrLimiter{}

// cxrRuleStore serves a fixed set of rules.
type cxrRuleStore struct {
	store.RuleStore
	rules   []types.RuleInfoInternal
	err     error
	onlyErr error
}

func (s *cxrRuleStore) ListAllRepoRules(
	_ context.Context, _ int64, ruleTypes ...enum.RuleType,
) ([]types.RuleInfoInternal, error) {
	return cxrFilterRules(s.rules, s.err, ruleTypes), s.err
}

func (s *cxrRuleStore) ListOnlyRepoRules(
	_ context.Context, _ *types.RepositoryCore, ruleTypes ...enum.RuleType,
) ([]types.RuleInfoInternal, error) {
	return cxrFilterRules(s.rules, s.onlyErr, ruleTypes), s.onlyErr
}

func cxrFilterRules(
	rules []types.RuleInfoInternal, err error, ruleTypes []enum.RuleType,
) []types.RuleInfoInternal {
	if err != nil {
		return nil
	}
	var out []types.RuleInfoInternal
	for _, r := range rules {
		for _, rt := range ruleTypes {
			if r.Type == rt {
				out = append(out, r)
				break
			}
		}
	}
	return out
}

// cxrRule builds a rule of the given type with the provided JSON definition that matches everything.
func cxrRule(ruleType enum.RuleType, definition string) types.RuleInfoInternal {
	return types.RuleInfoInternal{
		RuleInfo: types.RuleInfo{
			ID: 1, Identifier: "rule", Type: ruleType, State: enum.RuleStateActive,
		},
		Pattern:    json.RawMessage(`{}`),
		Definition: json.RawMessage(definition),
		RepoTarget: json.RawMessage(`{"include":{},"exclude":{}}`),
	}
}

// cxrUserGroups is a usergroup.Service without any groups.
type cxrUserGroups struct {
	usergroup.Service
}

func (cxrUserGroups) ListUserIDsByGroupIDs(context.Context, []int64) ([]int64, error) {
	return nil, nil
}

// cxrURLs is a url.Provider generating predictable URLs.
type cxrURLs struct {
	url.Provider
}

func (cxrURLs) GenerateUICompareURL(_ context.Context, repoPath, ref1, ref2 string) string {
	return "https://ui.example.com/" + repoPath + "/compare/" + ref1 + "..." + ref2
}

func (cxrURLs) GenerateUIPRURL(_ context.Context, repoPath string, prID int64) string {
	return "https://ui.example.com/" + repoPath + "/pulls/" + string(rune('0'+prID))
}

// cxrPullReqStore lists a fixed set of pull requests.
type cxrPullReqStore struct {
	store.PullReqStore
	prs []*types.PullReq
	err error
}

func (s *cxrPullReqStore) List(context.Context, *types.PullReqFilter) ([]*types.PullReq, error) {
	return s.prs, s.err
}

// cxrSSE is an sse.Streamer that discards events.
type cxrSSE struct {
	sse.Streamer
}

func (cxrSSE) Publish(context.Context, int64, enum.SSEType, any) {}

// cxrAudit records audit events.
type cxrAudit struct {
	actions []audit.Action
	err     error
}

func (a *cxrAudit) Log(
	_ context.Context, _ types.Principal, _ audit.Resource, action audit.Action, _ string, _ ...audit.Option,
) error {
	a.actions = append(a.actions, action)
	return a.err
}

// cxrExtender implements all three hook extenders.
type cxrExtender struct {
	err error
	// messages are appended to the hook output.
	messages []string
	errMsg   *string
	calls    int
}

func (e *cxrExtender) apply(out *hook.Output) error {
	e.calls++
	out.Messages = append(out.Messages, e.messages...)
	if e.errMsg != nil {
		out.Error = e.errMsg
	}
	return e.err
}

func (e *cxrExtender) Extend(
	_ context.Context, _ RestrictedGIT, _ *auth.Session, _ *types.RepositoryCore,
	_ types.GithookPreReceiveInput, out *hook.Output,
) error {
	return e.apply(out)
}

type cxrUpdateExtender struct{ *cxrExtender }

func (e cxrUpdateExtender) Extend(
	_ context.Context, _ RestrictedGIT, _ *auth.Session, _ *types.RepositoryCore,
	_ types.GithookUpdateInput, out *hook.Output,
) error {
	return e.apply(out)
}

type cxrPostExtender struct{ *cxrExtender }

func (e cxrPostExtender) Extend(
	_ context.Context, _ RestrictedGIT, _ *auth.Session, _ *types.RepositoryCore,
	_ types.GithookPostReceiveInput, out *hook.Output,
) error {
	return e.apply(out)
}

// cxrMQEntries reports a fixed set of branches as queued.
type cxrMQEntries struct {
	store.MergeQueueEntryStore
	queued map[string]struct{}
	err    error
}

func (s *cxrMQEntries) BranchesWithPullReqInQueue(
	_ context.Context, _ int64, branches []string,
) (map[string]struct{}, error) {
	if s.err != nil {
		return nil, s.err
	}
	out := map[string]struct{}{}
	for _, b := range branches {
		if _, ok := s.queued[b]; ok {
			out[b] = struct{}{}
		}
	}
	return out, nil
}

type cxrMQStore struct {
	store.MergeQueueStore
}

func (cxrMQStore) FindByRepoAndBranch(context.Context, int64, string) (*types.MergeQueue, error) {
	return nil, gitnessstore.ErrResourceNotFound
}

type cxrJobStore struct {
	job.Store
}

func (cxrJobStore) Create(context.Context, *job.Job) error { return nil }
func (cxrJobStore) Upsert(context.Context, *job.Job) error { return nil }

type cxrTx struct{}

func (cxrTx) WithTx(ctx context.Context, fn func(ctx context.Context) error, _ ...any) error {
	return fn(ctx)
}

var _ dbtx.Transactor = cxrTx{}

// cxrEnv is a githook controller wired on a real SQLite database with fakes for everything else.
type cxrEnv struct {
	*cxrBase
	ctrl     *Controller
	repo     *types.Repository
	git      *cxrGit
	rules    *cxrRuleStore
	prs      *cxrPullReqStore
	audit    *cxrAudit
	mqQueued *cxrMQEntries
	pre      *cxrExtender
	update   *cxrExtender
	post     *cxrExtender
	settings *settings.Service
	repoLFS  *cxrLFSStore
}

type cxrLFSStore struct {
	store.LFSObjectStore
	found []*types.LFSObject
	err   error
}

func (s *cxrLFSStore) FindMany(context.Context, int64, []string) ([]*types.LFSObject, error) {
	return s.found, s.err
}

func cxrNewEnv(ctx context.Context, t *testing.T) *cxrEnv {
	t.Helper()
	b := cxrNewBase(ctx, t)
	space := b.createSpace(ctx, t, "root")
	repo := b.createRepo(ctx, t, space, "repo")

	sys, err := events.ProvideSystem(
		events.Config{Mode: events.ModeInMemory, Namespace: "cxr", MaxStreamLength: 100},
		nil, events.NewNoopCollector())
	require.NoError(t, err)
	gitReporter, err := gitevents.NewReporter(sys)
	require.NoError(t, err)
	repoReporter, err := repoevents.NewReporter(sys)
	require.NoError(t, err)

	svc := settings.NewService(database.NewSettingsStore(b.db), b.spaceFinder)

	e := &cxrEnv{
		cxrBase:  b,
		repo:     repo,
		git:      &cxrGit{ancestor: true},
		rules:    &cxrRuleStore{},
		prs:      &cxrPullReqStore{},
		audit:    &cxrAudit{},
		mqQueued: &cxrMQEntries{queued: map[string]struct{}{}},
		pre:      &cxrExtender{},
		update:   &cxrExtender{},
		post:     &cxrExtender{},
		settings: svc,
		repoLFS:  &cxrLFSStore{},
	}

	manager := protection.NewManager(e.rules)
	require.NoError(t, manager.Register(
		protection.TypeBranch, func() protection.Definition { return &protection.Branch{} }))
	require.NoError(t, manager.Register(
		protection.TypeTag, func() protection.Definition { return &protection.Tag{} }))
	require.NoError(t, manager.Register(
		protection.TypePush, func() protection.Definition { return &protection.Push{} }))

	mq := cxrNewMergeQueue(ctx, t, sys, gitReporter, b, manager, e.mqQueued)

	e.ctrl = NewController(
		cxrAuthorizer{mode: cxrAllow}, b.principals, b.repos, b.repoFinder, gitReporter, repoReporter,
		e.prs, cxrURLs{}, manager, cxrLimiter{}, svc,
		e.pre, cxrUpdateExtender{e.update}, cxrPostExtender{e.post},
		cxrSSE{}, e.repoLFS, e.audit, cxrUserGroups{}, mq, nil)
	return e
}

func cxrNewMergeQueue(
	ctx context.Context,
	t *testing.T,
	sys *events.System,
	gitReporter *gitevents.Reporter,
	b *cxrBase,
	manager *protection.Manager,
	entries *cxrMQEntries,
) *mergequeue.Service {
	t.Helper()
	ctx, cancel := context.WithCancel(ctx)
	t.Cleanup(cancel)

	mqReporter, err := mergequeueevents.NewReporter(sys)
	require.NoError(t, err)
	checkFactory, err := checkevents.NewReaderFactory(sys)
	require.NoError(t, err)
	mqFactory, err := mergequeueevents.NewReaderFactory(sys)
	require.NoError(t, err)

	executor := job.NewExecutor(cxrJobStore{}, nil)
	scheduler, err := job.NewScheduler(cxrJobStore{}, executor, nil, nil, "test", 1, time.Hour)
	require.NoError(t, err)

	cfg := &types.Config{}
	cfg.MergeQueue.CheckEventsConcurrency = 1
	cfg.MergeQueue.QueueEventsConcurrency = 1

	svc, err := mergequeue.NewService(
		ctx, cfg, nil, cxrTx{}, gitReporter, mqReporter, checkFactory, mqFactory,
		b.repoFinder, nil, nil, nil, nil, cxrMQStore{}, entries, manager, nil,
		cxrURLs{}, locker.NewLocker(lock.NewInMemory(lock.Config{})), scheduler, executor,
	)
	require.NoError(t, err)
	return svc
}

// cxrBootstrap initializes the system service principal used when the default branch changes.
func cxrBootstrap(ctx context.Context, t *testing.T, b *cxrBase) {
	t.Helper()
	svcCtrl := service.NewController(check.PrincipalUIDDefault, cxrAuthorizer{mode: cxrAllow}, b.principals)
	cfg := &types.Config{}
	cfg.Principal.System.UID = "harness"
	cfg.Principal.System.Email = "system@example.com"
	cfg.Principal.System.DisplayName = "Harness"
	require.NoError(t, bootstrap.SystemService(ctx, cfg, svcCtrl))
}

// setRepoColumn updates a column of the test repository and evicts it from the finder cache.
func (e *cxrEnv) setRepoColumn(ctx context.Context, t *testing.T, column string, value any) {
	t.Helper()
	_, err := e.db.ExecContext(ctx, "UPDATE repositories SET "+column+" = $1 WHERE repo_id = $2",
		value, e.repo.ID)
	require.NoError(t, err)
}
