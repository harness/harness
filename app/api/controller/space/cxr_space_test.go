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
	"testing"
	"time"

	"github.com/harness/gitness/app/api/controller/limiter"
	"github.com/harness/gitness/app/api/controller/service"
	"github.com/harness/gitness/app/bootstrap"
	"github.com/harness/gitness/app/services/exporter"
	"github.com/harness/gitness/app/services/importer"
	"github.com/harness/gitness/app/services/instrument"
	"github.com/harness/gitness/app/services/label"
	"github.com/harness/gitness/app/services/publicaccess"
	"github.com/harness/gitness/app/services/pullreq"
	"github.com/harness/gitness/app/sse"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/app/store/cache"
	"github.com/harness/gitness/app/store/database"
	"github.com/harness/gitness/audit"
	"github.com/harness/gitness/job"
	"github.com/harness/gitness/lock"
	"github.com/harness/gitness/netpolicy"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/check"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/require"
)

// cxrJobStore is an in-memory job.Store.
type cxrJobStore struct {
	job.Store
	jobs      []*job.Job
	createErr error
	listErr   error
	progress  []*job.Job
}

func (s *cxrJobStore) Create(_ context.Context, j *job.Job) error {
	if s.createErr != nil {
		return s.createErr
	}
	s.jobs = append(s.jobs, j)
	return nil
}

func (s *cxrJobStore) ListByGroupID(context.Context, string) ([]*job.Job, error) {
	return s.progress, s.listErr
}

func (s *cxrJobStore) DeleteByGroupID(context.Context, string) (int64, error) {
	n := int64(len(s.progress))
	s.progress = nil
	return n, nil
}

// cxrCrypt is an encrypt.Encrypter that doesn't encrypt.
type cxrCrypt struct{}

func (cxrCrypt) Encrypt(plaintext string) ([]byte, error)  { return []byte(plaintext), nil }
func (cxrCrypt) Decrypt(ciphertext []byte) (string, error) { return string(ciphertext), nil }

type cxrPublicAccess struct {
	publicaccess.Service
}

func (cxrPublicAccess) Get(context.Context, enum.PublicResourceType, string) (bool, error) {
	return false, nil
}

func (cxrPublicAccess) Set(context.Context, enum.PublicResourceType, string, bool) error { return nil }

func (cxrPublicAccess) IsPublicAccessSupported(context.Context, enum.PublicResourceType, string) (bool, error) {
	return true, nil
}

type cxrSSE struct{ sse.Streamer }

func (cxrSSE) Publish(context.Context, int64, enum.SSEType, any) {}

// cxrPullReqStore serves empty pull request lists and counts.
type cxrPullReqStore struct {
	store.PullReqStore
	count int64
}

func (s cxrPullReqStore) Count(context.Context, *types.PullReqFilter) (int64, error) {
	return s.count, nil
}

func (cxrPullReqStore) ListOpenByBranchName(
	context.Context, int64, []string,
) (map[string][]*types.PullReq, error) {
	return map[string][]*types.PullReq{}, nil
}

func (cxrPullReqStore) Stream(context.Context, *types.PullReqFilter) (<-chan *types.PullReq, <-chan error) {
	prs := make(chan *types.PullReq)
	errs := make(chan error, 1)
	close(prs)
	close(errs)
	return prs, errs
}

type cxrSpaceEnv struct {
	*cxrBase
	ctrl     *Controller
	root     *types.Space
	jobs     *cxrJobStore
	prStore  cxrPullReqStore
	limiter  limiter.ResourceLimiter
	hosts    *httptest.Server
	repoList []map[string]any
}

func cxrNewSpaceEnv(ctx context.Context, t *testing.T) *cxrSpaceEnv {
	t.Helper()
	b := cxrNewBase(ctx, t)
	cxrBootstrapSystem(ctx, t, b)

	e := &cxrSpaceEnv{
		cxrBase: b, jobs: &cxrJobStore{}, limiter: limiter.NewResourceLimiter(),
		prStore: cxrPullReqStore{},
	}
	e.root = b.createSpace(ctx, t, "root")
	e.hosts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(e.repoList)
	}))
	t.Cleanup(e.hosts.Close)
	e.repoList = []map[string]any{
		{"name": "one", "private": false, "default_branch": "main", "clone_url": "https://x/owner/one.git",
			"owner": map[string]any{"login": "owner"}, "full_name": "owner/one"},
		{"name": "two", "private": true, "default_branch": "dev", "clone_url": "https://x/owner/two.git",
			"owner": map[string]any{"login": "owner"}, "full_name": "owner/two"},
	}
	e.build(t, cxrAllow)
	return e
}

func (e *cxrSpaceEnv) build(t *testing.T, mode cxrAuthMode) {
	t.Helper()
	tx := dbtx.New(e.db)
	executor := job.NewExecutor(e.jobs, nil)
	scheduler, err := job.NewScheduler(e.jobs, executor, lock.NewInMemory(lock.Config{
		App: "test", Namespace: "test", Expiry: time.Minute, Tries: 3, RetryDelay: 10 * time.Millisecond,
	}), nil, "test", 1, time.Hour)
	require.NoError(t, err)

	imp := importer.NewImporter(
		"main", nil, nil, tx, e.repos, nil, nil, e.repoFinder, nil, nil, nil, nil, audit.New(), nil,
		netpolicy.Policy{AllowLoopback: true}, false)
	jobRepo, err := importer.ProvideJobRepositoryImport(cxrCrypt{}, scheduler, executor, imp)
	require.NoError(t, err)

	exp, err := exporter.ProvideSpaceExporter(
		&types.Config{}, nil, nil, e.repos, scheduler, executor, cxrCrypt{}, cxrSSE{})
	require.NoError(t, err)

	authz := cxrAuthorizer{mode: mode}
	pCache := cache.ProvidePrincipalInfoCache(database.NewPrincipalInfoView(e.db))
	labelSvc := label.New(
		tx, e.spaces, database.NewLabelStore(e.db), database.NewLabelValueStore(e.db),
		database.NewPullReqLabelStore(e.db), database.NewPullReqLabelSuggestionStore(e.db), pCache, e.spaceFinder)
	prList := pullreq.NewListService(tx, nil, authz, e.spaces, e.prStore, nil, e.repoFinder, labelSvc, nil)

	e.ctrl = NewController(
		&types.Config{NestedSpacesEnabled: true}, tx, nil, cxrSSE{}, check.SpaceIdentifierDefault, authz,
		e.spacePaths, nil, nil, nil, nil, e.spaces, e.repos, e.principals, nil, e.memberships, prList, e.spaceFinder,
		e.repoFinder, jobRepo, exp, e.limiter, cxrPublicAccess{}, audit.New(), nil, nil, instrument.Noop{}, nil,
		nil, database.NewUsageMetricsStore(e.db), check.RepoIdentifierDefault, nil, nil, nil, nil,
	)
}

func cxrBootstrapSystem(ctx context.Context, t *testing.T, b *cxrBase) {
	t.Helper()
	svcCtrl := service.NewController(check.PrincipalUIDDefault, cxrAuthorizer{mode: cxrAllow}, b.principals)
	cfg := &types.Config{}
	cfg.Principal.System.UID = "harness"
	cfg.Principal.System.Email = "system@example.com"
	cfg.Principal.System.DisplayName = "Harness"
	require.NoError(t, bootstrap.SystemService(ctx, cfg, svcCtrl))
}

func (e *cxrSpaceEnv) provider() importer.Provider {
	return importer.Provider{Type: importer.ProviderTypeGitea, Host: e.hosts.URL, Username: "u", Password: "p"}
}

func TestCxrSpaceImport(t *testing.T) {
	ctx := t.Context()
	e := cxrNewSpaceEnv(ctx, t)

	out, err := e.ctrl.Import(ctx, e.session(), &ImportInput{
		CreateInput:   CreateInput{Identifier: "imported"},
		ProviderInput: ProviderInput{Provider: e.provider(), ProviderSpace: "owner"},
	})
	require.NoError(t, err)
	require.Equal(t, "imported", out.Identifier)
	require.Len(t, e.jobs.jobs, 2, "one import job per repository")

	repos, err := e.repos.List(ctx, out.ID, &types.RepoFilter{Page: 1, Size: 10})
	require.NoError(t, err)
	require.Len(t, repos, 2)
	for _, repo := range repos {
		require.Equal(t, enum.RepoStateGitImport, repo.State)
	}

	// The identifier defaults to the provider space.
	e.jobs.jobs = nil
	out, err = e.ctrl.Import(ctx, e.session(), &ImportInput{
		ProviderInput: ProviderInput{Provider: e.provider(), ProviderSpace: "owner"},
	})
	require.NoError(t, err)
	require.Equal(t, "owner", out.Identifier)
}

func TestCxrSpaceImportErrors(t *testing.T) {
	ctx := t.Context()
	in := func(e *cxrSpaceEnv) *ImportInput {
		return &ImportInput{
			CreateInput:   CreateInput{Identifier: "imported"},
			ProviderInput: ProviderInput{Provider: e.provider(), ProviderSpace: "owner"},
		}
	}

	t.Run("forbidden", func(t *testing.T) {
		e := cxrNewSpaceEnv(ctx, t)
		e.build(t, cxrDeny)
		forbidden := in(e)
		forbidden.ParentRef = "root"
		_, err := e.ctrl.Import(ctx, e.session(), forbidden)
		require.Error(t, err)
	})

	t.Run("invalid input", func(t *testing.T) {
		e := cxrNewSpaceEnv(ctx, t)
		bad := in(e)
		bad.Identifier = "!!"
		_, err := e.ctrl.Import(ctx, e.session(), bad)
		require.ErrorContains(t, err, "failed to sanitize input")
	})

	t.Run("provider failure", func(t *testing.T) {
		e := cxrNewSpaceEnv(ctx, t)
		bad := in(e)
		bad.ProviderSpace = ""
		bad.Identifier = "imported"
		_, err := e.ctrl.Import(ctx, e.session(), bad)
		require.Error(t, err)
	})

	t.Run("no repositories", func(t *testing.T) {
		e := cxrNewSpaceEnv(ctx, t)
		e.repoList = []map[string]any{}
		_, err := e.ctrl.Import(ctx, e.session(), in(e))
		require.ErrorContains(t, err, "Found no repositories")
	})

	t.Run("repository count limit", func(t *testing.T) {
		e := cxrNewSpaceEnv(ctx, t)
		e.limiter = cxrLimiter{countErr: errCxrAuthz}
		e.build(t, cxrAllow)
		_, err := e.ctrl.Import(ctx, e.session(), in(e))
		require.ErrorIs(t, err, limiter.ErrMaxNumReposReached)
	})

	t.Run("storage over limit", func(t *testing.T) {
		e := cxrNewSpaceEnv(ctx, t)
		e.limiter = cxrLimiter{storageErr: &limiter.TotalQuotaStorageError{Level: limiter.TotalQuotaLevelOverLimit}}
		e.build(t, cxrAllow)
		_, err := e.ctrl.Import(ctx, e.session(), in(e))
		require.Error(t, err)
	})

	t.Run("job scheduling failure", func(t *testing.T) {
		e := cxrNewSpaceEnv(ctx, t)
		e.jobs.createErr = errCxrAuthz
		_, err := e.ctrl.Import(ctx, e.session(), in(e))
		require.ErrorIs(t, err, errCxrAuthz)
	})
}

func TestCxrSpaceImportRepositories(t *testing.T) {
	ctx := t.Context()
	e := cxrNewSpaceEnv(ctx, t)
	in := &ImportRepositoriesInput{ProviderInput{Provider: e.provider(), ProviderSpace: "owner"}}

	out, err := e.ctrl.ImportRepositories(ctx, e.session(), "root", in)
	require.NoError(t, err)
	require.Len(t, out.ImportingRepos, 2)
	require.Len(t, e.jobs.jobs, 2)

	// Importing the same repositories again conflicts.
	_, err = e.ctrl.ImportRepositories(ctx, e.session(), "root", in)
	require.ErrorContains(t, err, "already exists")

	_, err = e.ctrl.ImportRepositories(ctx, e.session(), "nope", in)
	require.Error(t, err)

	e.repoList = []map[string]any{}
	_, err = e.ctrl.ImportRepositories(ctx, e.session(), "root", in)
	require.ErrorContains(t, err, "Found no repositories")
}

func TestCxrSpaceImportRepositoriesErrors(t *testing.T) {
	ctx := t.Context()
	e := cxrNewSpaceEnv(ctx, t)
	in := &ImportRepositoriesInput{ProviderInput{Provider: e.provider(), ProviderSpace: "owner"}}

	e.build(t, cxrDeny)
	_, err := e.ctrl.ImportRepositories(ctx, e.session(), "root", in)
	require.Error(t, err)

	e.limiter = cxrLimiter{countErr: errCxrAuthz}
	e.build(t, cxrAllow)
	_, err = e.ctrl.ImportRepositories(ctx, e.session(), "root", in)
	require.ErrorIs(t, err, limiter.ErrMaxNumReposReached)

	e.limiter = cxrLimiter{storageErr: &limiter.TotalQuotaStorageError{Level: limiter.TotalQuotaLevelOverLimit}}
	e.build(t, cxrAllow)
	_, err = e.ctrl.ImportRepositories(ctx, e.session(), "root", in)
	require.Error(t, err)

	e.limiter = limiter.NewResourceLimiter()
	e.build(t, cxrAllow)
	e.jobs.createErr = errCxrAuthz
	_, err = e.ctrl.ImportRepositories(ctx, e.session(), "root", in)
	require.ErrorIs(t, err, errCxrAuthz)

	// A repository identifier that is not acceptable fails the whole import.
	e.jobs.createErr = nil
	e.repoList[0]["name"] = "bad name!"
	_, err = e.ctrl.ImportRepositories(ctx, e.session(), "root", in)
	require.ErrorContains(t, err, "failed to sanitize the repo")
}

type cxrLimiter struct {
	countErr   error
	storageErr error
}

func (l cxrLimiter) RepoCount(context.Context, int64, int) error   { return l.countErr }
func (cxrLimiter) RepoSize(context.Context, int64) error           { return nil }
func (l cxrLimiter) RootSpaceStorage(context.Context, int64) error { return l.storageErr }

func TestCxrSpaceExport(t *testing.T) {
	ctx := t.Context()
	e := cxrNewSpaceEnv(ctx, t)
	// NOTE: the space has no repositories, the exporter has no public access service wired.
	in := func() *ExportInput {
		return &ExportInput{AccountID: "acc", OrgIdentifier: "org", ProjectIdentifier: "proj", Token: "token"}
	}

	require.NoError(t, e.ctrl.Export(ctx, e.session(), "root", in()))

	// Previously finished jobs are replaced.
	e.jobs.progress = []*job.Job{{State: job.JobStateFinished}}
	require.NoError(t, e.ctrl.Export(ctx, e.session(), "root", in()))

	// An export that is still running conflicts.
	e.jobs.progress = []*job.Job{{State: job.JobStateRunning}}
	require.ErrorContains(t, e.ctrl.Export(ctx, e.session(), "root", in()), "already in progress")
	e.jobs.progress = nil

	for name, mutate := range map[string]func(*ExportInput){
		"account": func(i *ExportInput) { i.AccountID = "" },
		"org":     func(i *ExportInput) { i.OrgIdentifier = "" },
		"project": func(i *ExportInput) { i.ProjectIdentifier = "" },
		"token":   func(i *ExportInput) { i.Token = "" },
	} {
		bad := in()
		mutate(bad)
		require.ErrorContains(t, e.ctrl.Export(ctx, e.session(), "root", bad), "failed to sanitize input", name)
	}

	require.ErrorContains(t, e.ctrl.Export(ctx, e.session(), "nope", in()), "failed to acquire access to space")

	e.jobs.listErr = errCxrAuthz
	require.ErrorContains(t, e.ctrl.Export(ctx, e.session(), "root", in()), "failed to start export repository job")
}

func TestCxrSpaceExportProgress(t *testing.T) {
	ctx := t.Context()
	e := cxrNewSpaceEnv(ctx, t)

	_, err := e.ctrl.ExportProgress(ctx, e.session(), "root")
	require.ErrorContains(t, err, "No recent or ongoing export")

	e.jobs.progress = []*job.Job{{State: job.JobStateRunning, RunProgress: 40}}
	out, err := e.ctrl.ExportProgress(ctx, e.session(), "root")
	require.NoError(t, err)
	require.Len(t, out.Repos, 1)
	require.Equal(t, 40, out.Repos[0].Progress)

	_, err = e.ctrl.ExportProgress(ctx, e.session(), "nope")
	require.ErrorContains(t, err, "failed to acquire access to space")
}

func TestCxrSpacePullReqs(t *testing.T) {
	ctx := t.Context()
	e := cxrNewSpaceEnv(ctx, t)
	e.prStore = cxrPullReqStore{count: 3}
	e.build(t, cxrAllow)

	count, err := e.ctrl.CountPullReqs(ctx, e.session(), "root", false, &types.PullReqFilter{})
	require.NoError(t, err)
	require.EqualValues(t, 3, count)

	count, err = e.ctrl.CountPullReqs(ctx, e.session(), "root", true, &types.PullReqFilter{})
	require.NoError(t, err)
	require.EqualValues(t, 3, count)

	_, err = e.ctrl.CountPullReqs(ctx, e.session(), "nope", false, &types.PullReqFilter{})
	require.ErrorContains(t, err, "space not found")

	list, err := e.ctrl.ListPullReqs(ctx, e.session(), "root", false, &types.PullReqFilter{})
	require.NoError(t, err)
	require.Empty(t, list)

	_, err = e.ctrl.ListPullReqs(ctx, e.session(), "nope", false, &types.PullReqFilter{})
	require.ErrorContains(t, err, "space not found")

	// Listing does not check the space permission (each repository is checked instead), counting does.
	e.build(t, cxrDeny)
	_, err = e.ctrl.CountPullReqs(ctx, e.session(), "root", false, &types.PullReqFilter{})
	require.Error(t, err)
}
