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
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	apiauth "github.com/harness/gitness/app/api/auth"
	"github.com/harness/gitness/app/api/controller/limiter"
	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/auth/authz"
	repoevents "github.com/harness/gitness/app/events/repo"
	"github.com/harness/gitness/app/services/codeowners"
	"github.com/harness/gitness/app/services/importer"
	"github.com/harness/gitness/app/services/instrument"
	"github.com/harness/gitness/app/services/keywordsearch"
	"github.com/harness/gitness/app/services/label"
	"github.com/harness/gitness/app/services/locker"
	"github.com/harness/gitness/app/services/protection"
	"github.com/harness/gitness/app/services/publicaccess"
	"github.com/harness/gitness/app/services/refcache"
	"github.com/harness/gitness/app/services/rules"
	"github.com/harness/gitness/app/services/settings"
	"github.com/harness/gitness/app/services/usergroup"
	"github.com/harness/gitness/app/sse"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/app/url"
	"github.com/harness/gitness/audit"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/lock"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/check"
	"github.com/harness/gitness/types/enum"
)

var errPublicRepoCreationDisabled = usererror.BadRequest("Public repository creation is disabled.")

type RepositoryOutput struct {
	types.Repository
	IsPublic  bool `json:"is_public" yaml:"is_public"`
	Importing bool `json:"importing" yaml:"-"`
	Archived  bool `json:"archived" yaml:"-"`
}

// TODO [CODE-1363]: remove after identifier migration.
func (r RepositoryOutput) MarshalJSON() ([]byte, error) {
	// alias allows us to embed the original object while avoiding an infinite loop of marshaling.
	type alias RepositoryOutput
	return json.Marshal(&struct {
		alias
		UID string `json:"uid"`
	}{
		alias: (alias)(r),
		UID:   r.Identifier,
	})
}

type Controller struct {
	defaultBranch string

	tx                 dbtx.Transactor
	urlProvider        url.Provider
	authorizer         authz.Authorizer
	repoStore          store.RepoStore
	spaceStore         store.SpaceStore
	pipelineStore      store.PipelineStore
	executionStore     store.ExecutionStore
	principalStore     store.PrincipalStore
	ruleStore          store.RuleStore
	checkStore         store.CheckStore
	pullReqStore       store.PullReqStore
	settings           *settings.Service
	principalInfoCache store.PrincipalInfoCache
	userGroupStore     store.UserGroupStore
	userGroupService   usergroup.SearchService
	protectionManager  *protection.Manager
	git                git.Interface
	spaceCache         refcache.SpaceCache
	repoFinder         refcache.RepoFinder
	importer           *importer.Repository
	codeOwners         *codeowners.Service
	eventReporter      *repoevents.Reporter
	indexer            keywordsearch.Indexer
	resourceLimiter    limiter.ResourceLimiter
	locker             *locker.Locker
	auditService       audit.Service
	mtxManager         lock.MutexManager
	identifierCheck    check.RepoIdentifier
	repoCheck          Check
	publicAccess       publicaccess.Service
	labelSvc           *label.Service
	instrumentation    instrument.Service
	rulesSvc           *rules.Service
	sseStreamer        sse.Streamer
}

func NewController(
	config *types.Config,
	tx dbtx.Transactor,
	urlProvider url.Provider,
	authorizer authz.Authorizer,
	repoStore store.RepoStore,
	spaceStore store.SpaceStore,
	pipelineStore store.PipelineStore,
	executionStore store.ExecutionStore,
	principalStore store.PrincipalStore,
	ruleStore store.RuleStore,
	checkStore store.CheckStore,
	pullReqStore store.PullReqStore,
	settings *settings.Service,
	principalInfoCache store.PrincipalInfoCache,
	protectionManager *protection.Manager,
	git git.Interface,
	spaceCache refcache.SpaceCache,
	repoFinder refcache.RepoFinder,
	importer *importer.Repository,
	codeOwners *codeowners.Service,
	eventReporter *repoevents.Reporter,
	indexer keywordsearch.Indexer,
	limiter limiter.ResourceLimiter,
	locker *locker.Locker,
	auditService audit.Service,
	mtxManager lock.MutexManager,
	identifierCheck check.RepoIdentifier,
	repoCheck Check,
	publicAccess publicaccess.Service,
	labelSvc *label.Service,
	instrumentation instrument.Service,
	userGroupStore store.UserGroupStore,
	userGroupService usergroup.SearchService,
	rulesSvc *rules.Service,
	sseStreamer sse.Streamer,
) *Controller {
	return &Controller{
		defaultBranch:      config.Git.DefaultBranch,
		tx:                 tx,
		urlProvider:        urlProvider,
		authorizer:         authorizer,
		repoStore:          repoStore,
		spaceStore:         spaceStore,
		pipelineStore:      pipelineStore,
		executionStore:     executionStore,
		principalStore:     principalStore,
		ruleStore:          ruleStore,
		checkStore:         checkStore,
		pullReqStore:       pullReqStore,
		settings:           settings,
		principalInfoCache: principalInfoCache,
		protectionManager:  protectionManager,
		git:                git,
		spaceCache:         spaceCache,
		repoFinder:         repoFinder,
		importer:           importer,
		codeOwners:         codeOwners,
		eventReporter:      eventReporter,
		indexer:            indexer,
		resourceLimiter:    limiter,
		locker:             locker,
		auditService:       auditService,
		mtxManager:         mtxManager,
		identifierCheck:    identifierCheck,
		repoCheck:          repoCheck,
		publicAccess:       publicAccess,
		labelSvc:           labelSvc,
		instrumentation:    instrumentation,
		userGroupStore:     userGroupStore,
		userGroupService:   userGroupService,
		rulesSvc:           rulesSvc,
		sseStreamer:        sseStreamer,
	}
}

// getRepoCheckAccess fetches a repo, checks if repo state allows requested permission
// and checks if the current user has permission to access it.
//
//nolint:unparam
func (c *Controller) getRepoCheckAccess(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	reqPermission enum.Permission,
	allowedRepoStates ...enum.RepoState,
) (*types.Repository, error) {
	return GetRepoCheckAccess(
		ctx,
		c.repoFinder,
		c.authorizer,
		session,
		repoRef,
		reqPermission,
		allowedRepoStates...,
	)
}

// getRepoCheckAccessForGit fetches a repo
// and checks if the current user has permission to access it.
func (c *Controller) getRepoCheckAccessForGit(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	reqPermission enum.Permission,
) (*types.Repository, error) {
	return GetRepoCheckAccess(
		ctx,
		c.repoFinder,
		c.authorizer,
		session,
		repoRef,
		reqPermission,
		// importing/migrating states are allowed - we'll block in the pre-receive hook if needed.
		enum.RepoStateGitImport, enum.RepoStateMigrateDataImport, enum.RepoStateMigrateGitPush,
	)
}

func (c *Controller) getSpaceCheckAuthRepoCreation(
	ctx context.Context,
	session *auth.Session,
	parentRef string,
) (*types.Space, error) {
	return GetSpaceCheckAuthRepoCreation(ctx, c.spaceCache, c.authorizer, session, parentRef)
}

func ValidateParentRef(parentRef string) error {
	parentRefAsID, err := strconv.ParseInt(parentRef, 10, 64)
	if (err == nil && parentRefAsID <= 0) || (len(strings.TrimSpace(parentRef)) == 0) {
		return errRepositoryRequiresParent
	}

	return nil
}

func (c *Controller) fetchRules(
	ctx context.Context,
	session *auth.Session,
	repo *types.Repository,
) (protection.Protection, bool, error) {
	isRepoOwner, err := apiauth.IsRepoOwner(ctx, c.authorizer, session, repo)
	if err != nil {
		return nil, false, fmt.Errorf("failed to determine if user is repo owner: %w", err)
	}

	protectionRules, err := c.protectionManager.ForRepository(ctx, repo.ID)
	if err != nil {
		return nil, false, fmt.Errorf("failed to fetch protection rules for the repository: %w", err)
	}

	return protectionRules, isRepoOwner, nil
}
