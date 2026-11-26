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

package pullreq

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"testing"
	"time"

	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/auth/authz"
	"github.com/harness/gitness/app/auth/authz/authztest"
	checkevents "github.com/harness/gitness/app/events/check"
	gitevents "github.com/harness/gitness/app/events/git"
	mergequeueevents "github.com/harness/gitness/app/events/mergequeue"
	pullreqevents "github.com/harness/gitness/app/events/pullreq"
	"github.com/harness/gitness/app/services/automerge"
	"github.com/harness/gitness/app/services/checkreq"
	"github.com/harness/gitness/app/services/codeowners"
	"github.com/harness/gitness/app/services/instrument"
	"github.com/harness/gitness/app/services/label"
	locker "github.com/harness/gitness/app/services/locker"
	"github.com/harness/gitness/app/services/merge"
	mergequeuesvc "github.com/harness/gitness/app/services/mergequeue"
	"github.com/harness/gitness/app/services/protection"
	"github.com/harness/gitness/app/services/publickey"
	pullreqsvc "github.com/harness/gitness/app/services/pullreq"
	"github.com/harness/gitness/app/services/refcache"
	"github.com/harness/gitness/app/services/settings"
	"github.com/harness/gitness/app/services/usergroup"
	"github.com/harness/gitness/app/sse"
	"github.com/harness/gitness/app/store"
	storecache "github.com/harness/gitness/app/store/cache"
	"github.com/harness/gitness/app/store/database"
	"github.com/harness/gitness/app/store/database/migrate"
	"github.com/harness/gitness/app/url"
	"github.com/harness/gitness/audit"
	"github.com/harness/gitness/cache"
	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/events"
	"github.com/harness/gitness/job"
	"github.com/harness/gitness/lock"
	mockgit "github.com/harness/gitness/mocks/git"
	mockpullreq "github.com/harness/gitness/mocks/pullreq"
	"github.com/harness/gitness/pubsub"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"

	_ "github.com/mattn/go-sqlite3" // sqlite driver.
)

const (
	cprUserAuthor   = "cpr_author"
	cprUserReviewer = "cpr_reviewer"
	cprUserOther    = "cpr_other"
)

// cprStreamer is an SSE streamer which records published events.
type cprStreamer struct {
	published []enum.SSEType
}

func (s *cprStreamer) Publish(_ context.Context, _ int64, eventType enum.SSEType, _ any) {
	s.published = append(s.published, eventType)
}

func (s *cprStreamer) Stream(context.Context, int64) (<-chan *sse.Event, <-chan error, func(context.Context) error) {
	return nil, nil, nil
}

// cprRepoIDCache is a repo ID cache that reads repositories from the repo store.
type cprRepoIDCache struct {
	repoStore store.RepoStore
	// repoType overrides the type of returned repositories when set.
	repoType enum.RepoType
}

func (c *cprRepoIDCache) Stats() (int64, int64)            { return 0, 0 }
func (c *cprRepoIDCache) Evict(_ context.Context, _ int64) {}
func (c *cprRepoIDCache) Get(ctx context.Context, id int64) (*types.RepositoryCore, error) {
	repo, err := c.repoStore.Find(ctx, id)
	if err != nil {
		return nil, err
	}
	core := repo.Core()
	if c.repoType != "" {
		core.Type = c.repoType
	}
	return core, nil
}

type cprUnusedSpacePathCache struct{}

func (cprUnusedSpacePathCache) Stats() (int64, int64)             { return 0, 0 }
func (cprUnusedSpacePathCache) Evict(_ context.Context, _ string) {}
func (cprUnusedSpacePathCache) Get(_ context.Context, _ string) (*types.SpacePath, error) {
	return nil, errors.New("space path cache is not available")
}

type cprUnusedRepoRefCache struct{}

func (cprUnusedRepoRefCache) Stats() (int64, int64)                         { return 0, 0 }
func (cprUnusedRepoRefCache) Evict(_ context.Context, _ types.RepoCacheKey) {}
func (cprUnusedRepoRefCache) Get(_ context.Context, _ types.RepoCacheKey) (int64, error) {
	return 0, errors.New("repo ref cache is not available")
}

// cprErrAuthorizer fails every authorization check with an error.
type cprErrAuthorizer struct{ err error }

func (a cprErrAuthorizer) Check(
	context.Context, *auth.Session, *types.Scope, *types.Resource, enum.Permission,
) (bool, error) {
	return false, a.err
}

func (a cprErrAuthorizer) CheckAll(context.Context, *auth.Session, ...types.PermissionCheck) (bool, error) {
	return false, a.err
}

func (a cprErrAuthorizer) CheckMany(context.Context, *auth.Session, ...types.PermissionCheck) ([]bool, error) {
	return nil, a.err
}

func cprDenyAuthorizer() authz.Authorizer {
	return authztest.DenyAuthorizer{}
}

// cprSelectiveAuthorizer denies every check for the listed principals, fails it with an error for others
// and allows all remaining principals.
type cprSelectiveAuthorizer struct {
	denied  map[int64]bool
	failing map[int64]error
}

func (a cprSelectiveAuthorizer) Check(
	_ context.Context, session *auth.Session, _ *types.Scope, _ *types.Resource, _ enum.Permission,
) (bool, error) {
	if err := a.failing[session.Principal.ID]; err != nil {
		return false, err
	}
	return !a.denied[session.Principal.ID], nil
}

func (a cprSelectiveAuthorizer) CheckAll(
	_ context.Context, session *auth.Session, _ ...types.PermissionCheck,
) (bool, error) {
	if err := a.failing[session.Principal.ID]; err != nil {
		return false, err
	}
	return !a.denied[session.Principal.ID], nil
}

func (a cprSelectiveAuthorizer) CheckMany(
	_ context.Context, session *auth.Session, checks ...types.PermissionCheck,
) ([]bool, error) {
	if err := a.failing[session.Principal.ID]; err != nil {
		return nil, err
	}
	res := make([]bool, len(checks))
	for i := range res {
		res[i] = !a.denied[session.Principal.ID]
	}
	return res, nil
}

// cprCheckReqs is a configurable provider of externally required checks.
type cprCheckReqs struct {
	reqs checkreq.Requirements
	err  error
}

func (p *cprCheckReqs) RequiredChecks(
	context.Context, *types.RepositoryCore, *types.PullReq,
) (checkreq.Requirements, error) {
	return p.reqs, p.err
}

// cprEnv bundles a controller backed by a real SQLite database together with seeded fixtures.
type cprEnv struct {
	t    *testing.T
	ctx  context.Context
	db   *sqlx.DB
	ctrl *Controller

	repoIDCache *cprRepoIDCache
	checkReqs   *cprCheckReqs
	git         *mockgit.Interface
	streamer    *cprStreamer

	principalStore *database.PrincipalStore
	repoStore      *database.RepoStore
	spaceStore     *database.SpaceStore
	pullreqStore   store.PullReqStore
	activityStore  store.PullReqActivityStore
	reviewerStore  store.PullReqReviewerStore
	reviewStore    store.PullReqReviewStore
	watcherStore   store.PullReqWatcherStore
	fileViewStore  store.PullReqFileViewStore

	author   *types.User
	reviewer *types.User
	other    *types.User

	space *types.Space
	repo  *types.Repository
}

func cprSetup(t *testing.T) *cprEnv {
	t.Helper()

	ctx := context.Background()

	db, err := sqlx.Connect("sqlite3", "file:"+filepath.Join(t.TempDir(), "cpr.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	_, err = db.Exec("PRAGMA foreign_keys = ON;")
	require.NoError(t, err)
	_, _ = db.Exec("PRAGMA busy_timeout = 5000;")
	require.NoError(t, migrate.Migrate(ctx, db))

	tx := dbtx.ProvideTransactor(dbtx.New(db))

	principalStore := database.NewPrincipalStore(db, store.ToLowerPrincipalUIDTransformation)
	principalInfoCache := cache.NewExtended[int64, *types.PrincipalInfo](
		database.NewPrincipalInfoView(db), time.Minute)

	spacePathStore := database.NewSpacePathStore(db, store.ToLowerSpacePathTransformation)
	spaceEvictor := storecache.NewEvictor[*types.SpaceCore]("cpr", "cpr-space", nil)
	spacePathCache := storecache.New(
		ctx, spacePathStore, store.ToLowerSpacePathTransformation, spaceEvictor, time.Minute)
	spaceStore := database.NewSpaceStore(db, spacePathCache, spacePathStore)
	repoStore := database.NewRepoStore(db, spacePathCache, spacePathStore, spaceStore)

	e := &cprEnv{
		checkReqs:      &cprCheckReqs{},
		t:              t,
		ctx:            ctx,
		db:             db,
		git:            &mockgit.Interface{},
		streamer:       &cprStreamer{},
		principalStore: principalStore,
		repoStore:      repoStore,
		spaceStore:     spaceStore,
		pullreqStore:   database.NewPullReqStore(db, principalInfoCache),
		activityStore:  database.NewPullReqActivityStore(db, principalInfoCache),
		reviewerStore:  database.NewPullReqReviewerStore(db, principalInfoCache),
		reviewStore:    database.NewPullReqReviewStore(db),
		watcherStore:   database.NewPullReqWatcherStore(db, principalInfoCache),
		fileViewStore:  database.NewPullReqFileViewStore(db),
	}

	e.author = e.createUser(1, cprUserAuthor)
	e.reviewer = e.createUser(2, cprUserReviewer)
	e.other = e.createUser(3, cprUserOther)

	e.space = &types.Space{ID: 1, Identifier: "cpr_space", CreatedBy: e.author.ID}
	require.NoError(t, spaceStore.Create(ctx, e.space))
	require.NoError(t, spacePathStore.InsertSegment(ctx, &types.SpacePathSegment{
		ID: e.space.ID, Identifier: e.space.Identifier, CreatedBy: e.author.ID, SpaceID: e.space.ID, IsPrimary: true,
	}))

	e.repo = &types.Repository{
		ParentID:            e.space.ID,
		Identifier:          "cpr_repo",
		GitUID:              "cpr_repo_git_uid",
		RootSpaceID:         e.space.ID,
		RootSpaceIdentifier: e.space.Identifier,
		CreatedBy:           e.author.ID,
		DefaultBranch:       "main",
		State:               enum.RepoStateActive,
		Type:                enum.RepoTypeNormal,
		Tags:                []byte("{}"),
	}
	require.NoError(t, repoStore.Create(ctx, e.repo))

	repoIDCache := &cprRepoIDCache{repoStore: repoStore}
	e.repoIDCache = repoIDCache
	repoFinder := refcache.NewRepoFinder(
		repoStore,
		cprUnusedSpacePathCache{},
		repoIDCache,
		cprUnusedRepoRefCache{},
		storecache.Evictor[*types.RepositoryCore]{},
	)

	spaceFinder := refcache.NewSpaceFinder(
		storecache.NewSpaceIDCache(ctx, spaceStore, spaceEvictor, time.Minute),
		spacePathCache,
		storecache.NewSpaceCaseInsensitiveCache(ctx, spaceStore, spaceEvictor, time.Minute),
		spaceEvictor,
	)

	labelStore := database.NewLabelStore(db)
	labelValueStore := database.NewLabelValueStore(db)
	labelAssignStore := database.NewPullReqLabelStore(db)
	labelSuggestionStore := database.NewPullReqLabelSuggestionStore(db)
	labelSvc := label.New(
		tx, spaceStore, labelStore, labelValueStore, labelAssignStore, labelSuggestionStore,
		principalInfoCache, spaceFinder)

	ruleStore := database.NewRuleStore(db, principalInfoCache)
	protectionManager, err := protection.ProvideManager(ruleStore)
	require.NoError(t, err)

	checkStore := database.NewCheckStore(db, principalInfoCache)
	userGroupStore := database.NewUserGroupStore(db)
	authorizer := authz.Authorizer(authztest.AllowAuthorizer{})

	urlProvider, err := url.NewProvider(
		"http://localhost:3000", "http://host.docker.internal:3000", "http://localhost:3000/api",
		"http://localhost:3000/git", "ssh://localhost", "git", false, "http://localhost:3000",
		"http://localhost:3000/registry")
	require.NoError(t, err)

	e.ctrl = &Controller{
		tx:                      tx,
		authorizer:              authorizer,
		pullreqStore:            e.pullreqStore,
		activityStore:           e.activityStore,
		reviewStore:             e.reviewStore,
		reviewerStore:           e.reviewerStore,
		reviewerSuggestionStore: database.NewPullReqReviewerSuggestionStore(db),
		watcherStore:            e.watcherStore,
		repoStore:               repoStore,
		principalStore:          principalStore,
		principalInfoCache:      principalInfoCache,
		fileViewStore:           e.fileViewStore,
		fileGroupStore:          database.NewPullReqFileGroupStore(db),
		checkStore:              checkStore,
		autoMergeStore:          database.NewAutoMergeStore(db),
		mergeQueueStore:         database.NewMergeQueueStore(db),
		mergeQueueEntryStore:    database.NewMergeQueueEntryStore(db),
		userGroupStore:          userGroupStore,
		userGroupReviewerStore:  database.ProvideUserGroupReviewerStore(db, principalInfoCache, userGroupStore),
		git:                     e.git,
		repoFinder:              repoFinder,
		eventReporter:           mockpullreq.NewStubReporter(t),
		sseStreamer:             e.streamer,
		protectionManager:       protectionManager,
		labelSvc:                labelSvc,
		labelStore:              labelStore,
		labelValueStore:         labelValueStore,
		labelSuggestionStore:    labelSuggestionStore,
		instrumentation:         instrument.Noop{},
		urlProvider:             urlProvider,
		settings:                settings.NewService(database.NewSettingsStore(db), refcache.SpaceFinder{}),
		signatureVerifyService: publickey.NewSignatureVerifyService(
			principalStore, nil, database.NewGitSignatureResultStore(db)),
		mergeQueueService: &mergequeuesvc.Service{},
		branchStore:       database.NewBranchStore(db),
		userGroupService:  usergroup.NewService(),
		userGroupResolver: usergroup.NewGitnessResolver(),
		pullreqListService: pullreqsvc.NewListService(
			tx, e.git, authorizer, spaceStore, e.pullreqStore, checkStore, repoFinder, labelSvc, protectionManager),
	}

	e.ctrl.codeCommentView = database.NewCodeCommentView(db)
	e.wireServices(urlProvider)

	return e
}

// cprReaders holds event reader factories attached to a private in-memory event system. Nothing reports events
// to that system, so no event handler ever runs while a test is executing.
type cprReaders struct {
	git       *events.ReaderFactory[*gitevents.Reader]
	pullreq   *events.ReaderFactory[*pullreqevents.Reader]
	check     *events.ReaderFactory[*checkevents.Reader]
	mergeQ    *events.ReaderFactory[*mergequeueevents.Reader]
	sinkGit   *gitevents.Reporter
	sinkMerge *mergequeueevents.Reporter
}

func (e *cprEnv) newReaders() *cprReaders {
	e.t.Helper()

	newSystem := func() *events.System {
		system, err := events.ProvideSystem(events.Config{
			Mode:            events.ModeInMemory,
			MaxStreamLength: 100,
		}, nil, events.NewNoopCollector())
		require.NoError(e.t, err)
		return system
	}

	readerSystem := newSystem()
	sinkSystem := newSystem()

	var err error
	out := &cprReaders{}

	out.git, err = gitevents.ProvideReaderFactory(readerSystem)
	require.NoError(e.t, err)
	out.pullreq, err = pullreqevents.ProvideReaderFactory(readerSystem)
	require.NoError(e.t, err)
	out.check, err = checkevents.ProvideReaderFactory(readerSystem)
	require.NoError(e.t, err)
	out.mergeQ, err = mergequeueevents.ProvideReaderFactory(readerSystem)
	require.NoError(e.t, err)
	out.sinkGit, err = gitevents.ProvideReporter(sinkSystem)
	require.NoError(e.t, err)
	out.sinkMerge, err = mergequeueevents.ProvideReporter(sinkSystem)
	require.NoError(e.t, err)

	return out
}

// wireServices builds the real pull request, merge, merge queue and auto merge services.
func (e *cprEnv) wireServices(urlProvider url.Provider) {
	e.t.Helper()

	ctx, cancel := context.WithCancel(context.Background())
	e.t.Cleanup(cancel)

	ctrl := e.ctrl
	cfg := cprServiceConfig()
	readers := e.newReaders()
	bus := pubsub.NewInMemory()

	lockMgr := lock.NewInMemory(lock.Config{
		App:        "cpr",
		Namespace:  "cpr",
		Expiry:     10 * time.Second,
		Tries:      3,
		RetryDelay: 10 * time.Millisecond,
		Provider:   lock.MemoryProvider,
	})
	ctrl.locker = locker.NewLocker(lockMgr)

	ctrl.codeOwners = codeowners.New(
		ctrl.repoStore, e.git, codeowners.Config{}, ctrl.principalStore, ctrl.userGroupResolver)

	ctrl.mergeService = merge.NewService(
		e.git, ctrl.tx, ctrl.eventReporter, ctrl.repoFinder, ctrl.repoStore, ctrl.pullreqStore,
		ctrl.activityStore, ctrl.checkStore, ctrl.reviewerStore, ctrl.autoMergeStore, ctrl.codeOwners,
		ctrl.userGroupService, urlProvider, e.streamer, instrument.Noop{}, e.checkReqs)

	jobStore := database.NewJobStore(e.db)
	executor := job.NewExecutor(jobStore, bus)
	scheduler, err := job.NewScheduler(jobStore, executor, lockMgr, bus, "cpr", 1, time.Hour)
	require.NoError(e.t, err)

	mqService, err := mergequeuesvc.NewService(
		ctx, cfg, e.git, ctrl.tx, readers.sinkGit, readers.sinkMerge, readers.check, readers.mergeQ,
		ctrl.repoFinder, ctrl.repoStore, ctrl.pullreqStore, ctrl.activityStore, ctrl.checkStore,
		ctrl.mergeQueueStore, ctrl.mergeQueueEntryStore, ctrl.protectionManager, ctrl.mergeService,
		urlProvider, ctrl.locker, scheduler, executor)
	require.NoError(e.t, err)
	ctrl.mergeQueueService = mqService

	ctrl.autoMergeService, err = automerge.NewService(
		ctx, cfg, e.git, ctrl.tx, ctrl.mergeService, mqService, readers.check, readers.pullreq,
		ctrl.repoFinder, ctrl.pullreqStore, ctrl.activityStore, ctrl.principalStore, ctrl.autoMergeStore,
		ctrl.protectionManager, e.streamer, ctrl.locker)
	require.NoError(e.t, err)

	ctrl.pullreqService, err = pullreqsvc.New(
		ctx, cfg, readers.git, readers.pullreq, ctrl.eventReporter, ctrl.tx, ctrl.authorizer, e.git,
		ctrl.repoFinder, ctrl.repoStore, ctrl.pullreqStore, ctrl.activityStore, ctrl.autoMergeStore,
		ctrl.principalStore, ctrl.reviewerStore, ctrl.reviewerSuggestionStore, ctrl.userGroupReviewerStore,
		ctrl.codeCommentView, nil, ctrl.fileViewStore, ctrl.principalInfoCache, bus, urlProvider, e.streamer)
	require.NoError(e.t, err)

	ctrl.auditService = audit.New()
}

// newPullReqServiceFor builds a pull request service from the dependencies of the given controller.
func (e *cprEnv) newPullReqServiceFor(ctrl *Controller) *pullreqsvc.Service {
	e.t.Helper()

	ctx, cancel := context.WithCancel(context.Background())
	e.t.Cleanup(cancel)

	readers := e.newReaders()
	service, err := pullreqsvc.New(
		ctx, cprServiceConfig(), readers.git, readers.pullreq, ctrl.eventReporter, ctrl.tx, ctrl.authorizer, e.git,
		ctrl.repoFinder, ctrl.repoStore, ctrl.pullreqStore, ctrl.activityStore, ctrl.autoMergeStore,
		ctrl.principalStore, ctrl.reviewerStore, ctrl.reviewerSuggestionStore, ctrl.userGroupReviewerStore,
		ctrl.codeCommentView, nil, ctrl.fileViewStore, ctrl.principalInfoCache, pubsub.NewInMemory(),
		ctrl.urlProvider, e.streamer)
	require.NoError(e.t, err)

	return service
}

func (e *cprEnv) createUser(id int64, uid string) *types.User {
	e.t.Helper()

	user := &types.User{
		ID:          id,
		UID:         uid,
		Email:       uid + "@example.com",
		DisplayName: uid,
		Created:     time.Now().UnixMilli(),
		Updated:     time.Now().UnixMilli(),
	}
	require.NoError(e.t, e.principalStore.CreateUser(e.ctx, user))

	return user
}

// cprStatus returns the HTTP status code carried by a user error or a status error,
// or 0 if err carries neither.
func cprStatus(err error) int {
	var userErr *usererror.Error
	if errors.As(err, &userErr) {
		return userErr.Status
	}

	var statusErr *errors.Error
	if errors.As(err, &statusErr) {
		//nolint:exhaustive // other statuses carry no HTTP mapping.
		switch statusErr.Status {
		case errors.StatusForbidden:
			return http.StatusForbidden
		case errors.StatusNotFound:
			return http.StatusNotFound
		case errors.StatusInvalidArgument:
			return http.StatusBadRequest
		case errors.StatusConflict:
			return http.StatusConflict
		case errors.StatusPreconditionFailed:
			return http.StatusPreconditionFailed
		case errors.StatusUnauthorized:
			return http.StatusUnauthorized
		default:
			return 0
		}
	}

	return 0
}

func (e *cprEnv) session(user *types.User) *auth.Session {
	return &auth.Session{Principal: *user.ToPrincipal()}
}

func (e *cprEnv) repoRef() string {
	return "1"
}

// createPR inserts an open pull request authored by e.author.
func (e *cprEnv) createPR(mutate ...func(pr *types.PullReq)) *types.PullReq {
	e.t.Helper()

	repo, err := e.repoStore.Find(e.ctx, e.repo.ID)
	require.NoError(e.t, err)
	repo.PullReqSeq++
	require.NoError(e.t, e.repoStore.Update(e.ctx, repo))

	now := time.Now().UnixMilli()
	srcSHA := "1111111111111111111111111111111111111111"
	targetSHA := "2222222222222222222222222222222222222222"
	baseSHA := "3333333333333333333333333333333333333333"
	pr := &types.PullReq{
		Number:              repo.PullReqSeq,
		CreatedBy:           e.author.ID,
		Created:             now,
		Updated:             now,
		Edited:              now,
		State:               enum.PullReqStateOpen,
		SubState:            enum.PullReqSubStateNone,
		Title:               "cpr title",
		Description:         "cpr description",
		SourceRepoID:        &e.repo.ID,
		SourceBranch:        fmt.Sprintf("feature-%d", repo.PullReqSeq),
		SourceSHA:           srcSHA,
		TargetRepoID:        e.repo.ID,
		TargetBranch:        "main",
		RootSpaceID:         e.space.ID,
		RootSpaceIdentifier: e.space.Identifier,
		MergeTargetSHA:      &targetSHA,
		MergeBaseSHA:        baseSHA,
		MergeCheckStatus:    enum.MergeCheckStatusUnchecked,
		RebaseCheckStatus:   enum.MergeCheckStatusUnchecked,
		Author:              *e.author.ToPrincipalInfo(),
	}
	for _, m := range mutate {
		m(pr)
	}
	require.NoError(e.t, e.pullreqStore.Create(e.ctx, pr))

	return pr
}

// createComment inserts a plain comment activity on the pull request, authored by the given user.
func (e *cprEnv) createComment(
	pr *types.PullReq,
	user *types.User,
	text string,
	mutate ...func(act *types.PullReqActivity),
) *types.PullReqActivity {
	e.t.Helper()

	pr, err := e.pullreqStore.Find(e.ctx, pr.ID)
	require.NoError(e.t, err)
	pr.ActivitySeq++
	require.NoError(e.t, e.pullreqStore.Update(e.ctx, pr))

	now := time.Now().UnixMilli()
	act := &types.PullReqActivity{
		CreatedBy: user.ID,
		Created:   now,
		Updated:   now,
		Edited:    now,
		RepoID:    pr.TargetRepoID,
		PullReqID: pr.ID,
		Order:     pr.ActivitySeq,
		Kind:      enum.PullReqActivityKindComment,
		Type:      enum.PullReqActivityTypeComment,
		Text:      text,
		Author:    *user.ToPrincipalInfo(),

		PayloadRaw: json.RawMessage("{}"),
	}
	for _, m := range mutate {
		m(act)
	}
	require.NoError(e.t, e.activityStore.Create(e.ctx, act))

	return act
}

// createFork inserts a second repository to act as a fork source.
func (e *cprEnv) createFork() *types.Repository {
	e.t.Helper()

	fork := &types.Repository{
		ParentID:            e.space.ID,
		Identifier:          "cpr_fork",
		GitUID:              "cpr_fork_git_uid",
		RootSpaceID:         e.space.ID,
		RootSpaceIdentifier: e.space.Identifier,
		CreatedBy:           e.author.ID,
		DefaultBranch:       "main",
		ForkID:              e.repo.ID,
		State:               enum.RepoStateActive,
		Type:                enum.RepoTypeNormal,
		Tags:                []byte("{}"),
	}
	require.NoError(e.t, e.repoStore.Create(e.ctx, fork))

	return fork
}

func cprServiceConfig() *types.Config {
	cfg := &types.Config{InstanceID: "cpr"}
	cfg.PullReq.GitEventsConcurrency = 1
	cfg.PullReq.FileViewedConcurrency = 1
	cfg.PullReq.MergeabilityConcurrency = 1
	cfg.PullReq.CodeCommentsConcurrency = 1
	cfg.AutoMerge.CheckEventsConcurrency = 1
	cfg.AutoMerge.PullReqEventsConcurrency = 1
	cfg.MergeQueue.CheckEventsConcurrency = 1
	cfg.MergeQueue.QueueEventsConcurrency = 1

	return cfg
}

// createBranchRule inserts an active repository branch protection rule matching all branches.
func (e *cprEnv) createBranchRule(identifier string, definition string) {
	e.t.Helper()

	now := time.Now().UnixMilli()
	rule := &types.Rule{
		CreatedBy:  e.author.ID,
		Created:    now,
		Updated:    now,
		RepoID:     &e.repo.ID,
		Identifier: identifier,
		Type:       protection.TypeBranch,
		State:      enum.RuleStateActive,
		Pattern:    json.RawMessage(`{"include":["**"]}`),
		RepoTarget: json.RawMessage(`{}`),
		Definition: json.RawMessage(definition),
	}
	require.NoError(e.t, database.NewRuleStore(e.db, e.ctrl.principalInfoCache).Create(e.ctx, rule))
}
