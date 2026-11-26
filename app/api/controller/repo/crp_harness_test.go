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

package repo

import (
	"context"
	"errors"
	"testing"

	"github.com/harness/gitness/app/api/controller/limiter"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/auth/authz"
	"github.com/harness/gitness/app/auth/authz/authztest"
	repoevents "github.com/harness/gitness/app/events/repo"
	"github.com/harness/gitness/app/services/dotrange"
	"github.com/harness/gitness/app/services/instrument"
	"github.com/harness/gitness/app/services/protection"
	"github.com/harness/gitness/app/services/publickey"
	"github.com/harness/gitness/app/services/refcache"
	"github.com/harness/gitness/app/services/usergroup"
	"github.com/harness/gitness/app/store"
	storecache "github.com/harness/gitness/app/store/cache"
	"github.com/harness/gitness/app/url"
	"github.com/harness/gitness/audit"
	"github.com/harness/gitness/events"
	"github.com/harness/gitness/git/sha"
	mockgit "github.com/harness/gitness/mocks/git"
	gitnessstore "github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/require"
)

const (
	crpRepoID         int64 = 7
	crpRepoRef              = "7"
	crpRepoPath             = "space/repo"
	crpDefaultBranch        = "main"
	crpUnknownRepoRef       = "9999"
	crpGitUID               = "gituid"
	crpRepoIdentifier       = "repo"
	crpAliceEmail           = "alice@example.com"
	crpFeatureBranch        = "feature"
	crpHeadBranch           = "head"
	crpBaseRef              = "base"
	crpReleaseBranch        = "release"
	crpOtherSpacePath       = "other"
)

var (
	errCrpBoom      = errors.New("boom")
	crpTestSHA      = sha.Must("1111111111111111111111111111111111111111")
	crpTestSHAOther = sha.Must("2222222222222222222222222222222222222222")
)

// crpSession returns an authenticated session of a regular user.
func crpSession() *auth.Session {
	return &auth.Session{
		Principal: types.Principal{
			ID:          42,
			UID:         "alice",
			Email:       crpAliceEmail,
			DisplayName: "Alice",
			Type:        enum.PrincipalTypeUser,
		},
	}
}

// crpErrAuthorizer is an authorizer that fails every check with the provided error.
type crpErrAuthorizer struct{ err error }

func (a crpErrAuthorizer) Check(
	context.Context, *auth.Session, *types.Scope, *types.Resource, enum.Permission,
) (bool, error) {
	return false, a.err
}

func (a crpErrAuthorizer) CheckAll(context.Context, *auth.Session, ...types.PermissionCheck) (bool, error) {
	return false, a.err
}

func (a crpErrAuthorizer) CheckMany(
	context.Context, *auth.Session, ...types.PermissionCheck,
) ([]bool, error) {
	return nil, a.err
}

// crpDenyPermAuthorizer allows everything except the listed permissions.
type crpDenyPermAuthorizer struct {
	authztest.AllowAuthorizer
	denied map[enum.Permission]bool
}

func (a crpDenyPermAuthorizer) Check(
	_ context.Context, _ *auth.Session, _ *types.Scope, _ *types.Resource, perm enum.Permission,
) (bool, error) {
	return !a.denied[perm], nil
}

// crpIDCache is an in-memory store.RepoIDCache.
type crpIDCache struct {
	repos map[int64]*types.RepositoryCore
}

func (c *crpIDCache) Stats() (int64, int64) { return 0, 0 }

func (c *crpIDCache) Get(_ context.Context, id int64) (*types.RepositoryCore, error) {
	repo, ok := c.repos[id]
	if !ok {
		return nil, gitnessstore.ErrResourceNotFound
	}

	clone := *repo

	return &clone, nil
}

func (c *crpIDCache) Evict(context.Context, int64) {}

var _ store.RepoIDCache = (*crpIDCache)(nil)

// crpRuleStore is a store.RuleStore returning preconfigured rules.
type crpRuleStore struct {
	store.RuleStore
	rules []types.RuleInfoInternal
	err   error

	// fields used by the rules service.
	rule      *types.Rule
	findErr   error
	deleteErr error
	listRules []types.Rule
	listErr   error
	countVal  int64
	countErr  error
}

func (s *crpRuleStore) FindByIdentifier(
	context.Context, enum.RuleParent, int64, string,
) (*types.Rule, error) {
	if s.findErr != nil {
		return nil, s.findErr
	}

	rule := *s.rule

	return &rule, nil
}

func (s *crpRuleStore) Delete(context.Context, int64) error { return s.deleteErr }

func (s *crpRuleStore) List(context.Context, []types.RuleParentInfo, *types.RuleFilter) ([]types.Rule, error) {
	return s.listRules, s.listErr
}

func (s *crpRuleStore) Count(context.Context, []types.RuleParentInfo, *types.RuleFilter) (int64, error) {
	return s.countVal, s.countErr
}

func (s *crpRuleStore) ListAllRepoRules(
	context.Context, int64, ...enum.RuleType,
) ([]types.RuleInfoInternal, error) {
	return s.rules, s.err
}

// crpURLProvider is a url.Provider with static URLs.
type crpURLProvider struct {
	url.Provider
}

func (crpURLProvider) GetInternalAPIURL(context.Context) string { return "http://internal.example.com" }

func (crpURLProvider) GenerateGITCloneURL(_ context.Context, repoPath string) string {
	return "https://git.example.com/" + repoPath + ".git"
}

func (crpURLProvider) GenerateGITCloneSSHURL(_ context.Context, repoPath string) string {
	return "ssh://git.example.com/" + repoPath + ".git"
}

func (s *crpRuleStore) ListOnlyRepoRules(
	context.Context, *types.RepositoryCore, ...enum.RuleType,
) ([]types.RuleInfoInternal, error) {
	return s.rules, s.err
}

// crpBranchRule builds an active repo-level branch rule matching all branches with the given definition JSON.
func crpBranchRule(definition string) types.RuleInfoInternal {
	return crpRule(protection.TypeBranch, definition)
}

// crpTagRule builds an active repo-level tag rule matching all tags with the given definition JSON.
func crpTagRule(definition string) types.RuleInfoInternal {
	return crpRule(protection.TypeTag, definition)
}

func crpRule(ruleType enum.RuleType, definition string) types.RuleInfoInternal {
	return types.RuleInfoInternal{
		RuleInfo: types.RuleInfo{
			RepoPath:   crpRepoPath,
			ID:         1,
			Identifier: "rule1",
			Type:       ruleType,
			State:      enum.RuleStateActive,
		},
		RepoTarget: []byte(`{}`),
		Pattern:    []byte(`{"include":["**"]}`),
		Definition: []byte(definition),
	}
}

// crpSigStore is a store.GitSignatureResultStore that knows no signatures and accepts all writes.
type crpSigStore struct {
	store.GitSignatureResultStore
}

func (crpSigStore) Map(context.Context, int64, []sha.SHA) (map[sha.SHA]types.GitSignatureResult, error) {
	return map[sha.SHA]types.GitSignatureResult{}, nil
}

func (crpSigStore) TryCreateAll(context.Context, []*types.GitSignatureResult) error { return nil }

// crpUserGroupService is a usergroup.Service with no groups.
type crpUserGroupService struct {
	usergroup.Service
}

func (crpUserGroupService) ListUserIDsByGroupIDs(context.Context, []int64) ([]int64, error) {
	return nil, nil
}

// crpEnv bundles a controller with the fakes backing it.
type crpEnv struct {
	c         *Controller
	git       *mockgit.Interface
	ruleStore *crpRuleStore
	repo      *types.RepositoryCore
	idCache   *crpIDCache
	repoStore *crpRepoStore
	spaces    *crpSpaceStore
	spaceIDs  *crpSpaceIDCache
	public    *crpPublicAccess
}

// crpNewEnv builds a controller with an active repository crpRepoID and an allow-all authorizer.
func crpNewEnv(t *testing.T) *crpEnv {
	t.Helper()

	eventSystem, err := events.ProvideSystem(events.Config{
		Mode:            events.ModeInMemory,
		MaxStreamLength: 100,
	}, nil, events.NewNoopCollector())
	require.NoError(t, err)

	eventReporter, err := repoevents.NewReporter(eventSystem)
	require.NoError(t, err)

	repo := &types.RepositoryCore{
		ID:            crpRepoID,
		ParentID:      3,
		Identifier:    crpRepoIdentifier,
		Path:          crpRepoPath,
		GitUID:        crpGitUID,
		DefaultBranch: crpDefaultBranch,
		State:         enum.RepoStateActive,
	}

	idCache := &crpIDCache{repos: map[int64]*types.RepositoryCore{crpRepoID: repo}}
	repoStore := newCrpRepoStore(&types.Repository{
		ID:                  crpRepoID,
		ParentID:            3,
		RootSpaceID:         3,
		RootSpaceIdentifier: "space",
		Identifier:          crpRepoIdentifier,
		Path:                crpRepoPath,
		GitUID:              crpGitUID,
		DefaultBranch:       crpDefaultBranch,
		State:               enum.RepoStateActive,
	})
	repoFinder := refcache.NewRepoFinder(
		repoStore, nil, idCache, nil, storecache.Evictor[*types.RepositoryCore]{},
	)

	spaceCores := map[int64]*types.SpaceCore{3: {ID: 3, Path: "space", Identifier: "space"}}
	spaceIDs := &crpSpaceIDCache{spaces: spaceCores}
	spaceFinder := refcache.NewSpaceFinder(spaceIDs, nil, nil, storecache.Evictor[*types.SpaceCore]{})
	spaceStore := &crpSpaceStore{spaces: map[int64]*types.Space{
		3: {ID: 3, Path: "space", Identifier: "space", RootSpaceID: 3, RootSpaceIdentifier: "space"},
	}}
	publicAccess := newCrpPublicAccess()

	gitMock := &mockgit.Interface{}
	ruleStore := &crpRuleStore{}

	protectionManager, err := protection.ProvideManager(ruleStore)
	require.NoError(t, err)

	return &crpEnv{
		c: &Controller{
			defaultBranch:          crpDefaultBranch,
			authorizer:             authztest.AllowAuthorizer{},
			repoFinder:             repoFinder,
			repoStore:              repoStore,
			spaceFinder:            spaceFinder,
			spaceStore:             spaceStore,
			publicAccess:           publicAccess,
			favoriteStore:          crpFavoriteStore{favorites: map[int64]bool{}},
			repoCheck:              NewNoOpRepoChecks(),
			resourceLimiter:        limiter.Unlimited{},
			identifierCheck:        func(string, *auth.Session) error { return nil },
			indexer:                &crpIndexer{},
			locker:                 crpNewLocker(),
			tx:                     crpTx{},
			git:                    gitMock,
			ruleStore:              ruleStore,
			protectionManager:      protectionManager,
			userGroupService:       crpUserGroupService{},
			urlProvider:            crpURLProvider{},
			dotRangeService:        dotrange.ProvideService(gitMock, repoFinder, crpURLProvider{}, authztest.AllowAuthorizer{}),
			auditService:           audit.New(),
			signatureVerifyService: publickey.NewSignatureVerifyService(nil, nil, crpSigStore{}),
			instrumentation:        instrument.Noop{},
			eventReporter:          eventReporter,
		},
		git:       gitMock,
		ruleStore: ruleStore,
		repo:      repo,
		idCache:   idCache,
		repoStore: repoStore,
		spaces:    spaceStore,
		spaceIDs:  spaceIDs,
		public:    publicAccess,
	}
}

// addOtherSpace makes a second root space (id 4, path crpOtherSpacePath) resolvable by the controller.
func (e *crpEnv) addOtherSpace() {
	const (
		id   int64 = 4
		path       = crpOtherSpacePath
	)

	e.spaceIDs.spaces[id] = &types.SpaceCore{ID: id, Path: path, Identifier: path}
	e.spaces.spaces[id] = &types.Space{
		ID: id, Path: path, Identifier: path, RootSpaceID: id, RootSpaceIdentifier: path,
	}
}

// addRepo makes another repository resolvable by the controller's repo finder (for example an upstream).
func (e *crpEnv) addRepo(repo *types.RepositoryCore) {
	e.idCache.repos[repo.ID] = repo
}

// setAuthorizer replaces the authorizer of the controller.
func (e *crpEnv) setAuthorizer(a authz.Authorizer) { e.c.authorizer = a }

// setRepo mutates the stored repository (also visible through the controller's finder).
func (e *crpEnv) setRepo(mutate func(r *types.RepositoryCore)) {
	mutate(e.repo)
}

// crpIndexerOf returns the recording indexer the controller was built with.
func crpIndexerOf(t *testing.T, e *crpEnv) *crpIndexer {
	t.Helper()

	indexer, ok := e.c.indexer.(*crpIndexer)
	require.True(t, ok)

	return indexer
}

// crpLinkedStoreOf returns the recording linked repo store the controller was built with.
func crpLinkedStoreOf(t *testing.T, e *crpEnv) *crpLinkedRepoCreateStore {
	t.Helper()

	linked, ok := e.c.linkedRepoStore.(*crpLinkedRepoCreateStore)
	require.True(t, ok)

	return linked
}
