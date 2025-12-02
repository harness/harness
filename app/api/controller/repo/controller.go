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
	"github.com/harness/gitness/app/api/controller"
	"github.com/harness/gitness/app/api/controller/lfs"
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
	"github.com/harness/gitness/app/services/publickey"
	"github.com/harness/gitness/app/services/refcache"
	"github.com/harness/gitness/app/services/rules"
	"github.com/harness/gitness/app/services/settings"
	"github.com/harness/gitness/app/services/usergroup"
	"github.com/harness/gitness/app/sse"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/app/url"
	"github.com/harness/gitness/audit"
	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/git"
	gitenum "github.com/harness/gitness/git/enum"
	"github.com/harness/gitness/git/sha"
	"github.com/harness/gitness/lock"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/check"
	"github.com/harness/gitness/types/enum"
)

var errPublicRepoCreationDisabled = usererror.BadRequest("Public repository creation is disabled.")

type RepositoryOutput struct {
	types.Repository
	IsPublic   bool                  `json:"is_public" yaml:"is_public"`
	Importing  bool                  `json:"importing" yaml:"-"`
	Archived   bool                  `json:"archived" yaml:"-"`
	IsFavorite bool                  `json:"is_favorite" yaml:"is_favorite"`
	Upstream   *types.RepositoryCore `json:"upstream,omitempty" yaml:"-,omitempty"`
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

	tx                     dbtx.Transactor
	urlProvider            url.Provider
	authorizer             authz.Authorizer
	repoStore              store.RepoStore
	spaceStore             store.SpaceStore
	pipelineStore          store.PipelineStore
	executionStore         store.ExecutionStore
	principalStore         store.PrincipalStore
	ruleStore              store.RuleStore
	checkStore             store.CheckStore
	pullReqStore           store.PullReqStore
	settings               *settings.Service
	principalInfoCache     store.PrincipalInfoCache
	userGroupStore         store.UserGroupStore
	userGroupService       usergroup.Service
	protectionManager      *protection.Manager
	git                    git.Interface
	spaceFinder            refcache.SpaceFinder
	repoFinder             refcache.RepoFinder
	importer               *importer.JobRepository
	referenceSync          *importer.JobReferenceSync
	codeOwners             *codeowners.Service
	eventReporter          *repoevents.Reporter
	indexer                keywordsearch.Indexer
	resourceLimiter        limiter.ResourceLimiter
	locker                 *locker.Locker
	auditService           audit.Service
	mtxManager             lock.MutexManager
	identifierCheck        check.RepoIdentifier
	repoCheck              Check
	publicAccess           publicaccess.Service
	labelSvc               *label.Service
	instrumentation        instrument.Service
	rulesSvc               *rules.Service
	sseStreamer            sse.Streamer
	lfsCtrl                *lfs.Controller
	favoriteStore          store.FavoriteStore
	signatureVerifyService publickey.SignatureVerifyService
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
	spaceFinder refcache.SpaceFinder,
	repoFinder refcache.RepoFinder,
	importer *importer.JobRepository,
	referenceSync *importer.JobReferenceSync,
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
	userGroupService usergroup.Service,
	rulesSvc *rules.Service,
	sseStreamer sse.Streamer,
	lfsCtrl *lfs.Controller,
	favoriteStore store.FavoriteStore,
	signatureVerifyService publickey.SignatureVerifyService,
) *Controller {
	return &Controller{
		defaultBranch:          config.Git.DefaultBranch,
		tx:                     tx,
		urlProvider:            urlProvider,
		authorizer:             authorizer,
		repoStore:              repoStore,
		spaceStore:             spaceStore,
		pipelineStore:          pipelineStore,
		executionStore:         executionStore,
		principalStore:         principalStore,
		ruleStore:              ruleStore,
		checkStore:             checkStore,
		pullReqStore:           pullReqStore,
		settings:               settings,
		principalInfoCache:     principalInfoCache,
		protectionManager:      protectionManager,
		git:                    git,
		spaceFinder:            spaceFinder,
		repoFinder:             repoFinder,
		importer:               importer,
		referenceSync:          referenceSync,
		codeOwners:             codeOwners,
		eventReporter:          eventReporter,
		indexer:                indexer,
		resourceLimiter:        limiter,
		locker:                 locker,
		auditService:           auditService,
		mtxManager:             mtxManager,
		identifierCheck:        identifierCheck,
		repoCheck:              repoCheck,
		publicAccess:           publicAccess,
		labelSvc:               labelSvc,
		instrumentation:        instrumentation,
		userGroupStore:         userGroupStore,
		userGroupService:       userGroupService,
		rulesSvc:               rulesSvc,
		sseStreamer:            sseStreamer,
		lfsCtrl:                lfsCtrl,
		favoriteStore:          favoriteStore,
		signatureVerifyService: signatureVerifyService,
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
) (*types.RepositoryCore, error) {
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
) (*types.RepositoryCore, error) {
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
) (*types.SpaceCore, error) {
	return GetSpaceCheckAuthRepoCreation(ctx, c.spaceFinder, c.authorizer, session, parentRef)
}

func ValidateParentRef(parentRef string) error {
	parentRefAsID, err := strconv.ParseInt(parentRef, 10, 64)
	if (err == nil && parentRefAsID <= 0) || (len(strings.TrimSpace(parentRef)) == 0) {
		return errRepositoryRequiresParent
	}

	return nil
}

func eventBase(repo *types.RepositoryCore, principal *types.Principal) repoevents.Base {
	return repoevents.Base{
		RepoID:      repo.ID,
		PrincipalID: principal.ID,
	}
}

func (c *Controller) fetchBranchRules(
	ctx context.Context,
	session *auth.Session,
	repo *types.RepositoryCore,
) (protection.BranchProtection, bool, error) {
	isRepoOwner, err := apiauth.IsRepoOwner(ctx, c.authorizer, session, repo)
	if err != nil {
		return nil, false, fmt.Errorf("failed to determine if user is repo owner: %w", err)
	}

	protectionRules, err := c.protectionManager.ListRepoBranchRules(ctx, repo.ID)
	if err != nil {
		return nil, false, fmt.Errorf("failed to fetch protection rules for the repository: %w", err)
	}

	return protectionRules, isRepoOwner, nil
}

func (c *Controller) fetchTagRules(
	ctx context.Context,
	session *auth.Session,
	repo *types.RepositoryCore,
) (protection.TagProtection, bool, error) {
	isRepoOwner, err := apiauth.IsRepoOwner(ctx, c.authorizer, session, repo)
	if err != nil {
		return nil, false, fmt.Errorf("failed to determine if user is repo owner: %w", err)
	}

	protectionRules, err := c.protectionManager.ListRepoTagRules(ctx, repo.ID)
	if err != nil {
		return nil, false, fmt.Errorf("failed to fetch protection rules for the repository: %w", err)
	}

	return protectionRules, isRepoOwner, nil
}

func (c *Controller) fetchUpstreamBranch(
	ctx context.Context,
	session *auth.Session,
	repoForkCore *types.RepositoryCore,
	branchName string,
) (sha.SHA, *types.RepositoryCore, error) {
	return c.fetchUpstreamObjects(
		ctx,
		session,
		repoForkCore,
		func(readParams git.ReadParams) (sha.SHA, error) {
			result, err := c.git.GetRef(ctx, git.GetRefParams{
				ReadParams: readParams,
				Name:       branchName,
				Type:       gitenum.RefTypeBranch,
			})
			if err != nil {
				return sha.SHA{}, fmt.Errorf("failed to fetch branch %s: %w", branchName, err)
			}

			return result.SHA, nil
		})
}

func (c *Controller) fetchUpstreamRevision(
	ctx context.Context,
	session *auth.Session,
	repoForkCore *types.RepositoryCore,
	revision string,
) (sha.SHA, *types.RepositoryCore, error) {
	return c.fetchUpstreamObjects(
		ctx,
		session,
		repoForkCore,
		func(readParams git.ReadParams) (sha.SHA, error) {
			result, err := c.git.ResolveRevision(ctx, git.ResolveRevisionParams{
				ReadParams: readParams,
				Revision:   revision,
			})
			if err != nil {
				return sha.SHA{}, fmt.Errorf("failed to resolve revision %s: %w", revision, err)
			}

			return result.SHA, nil
		})
}

func (c *Controller) fetchUpstreamObjects(
	ctx context.Context,
	session *auth.Session,
	repoForkCore *types.RepositoryCore,
	getSHA func(params git.ReadParams) (sha.SHA, error),
) (sha.SHA, *types.RepositoryCore, error) {
	repoFork, err := c.repoStore.Find(ctx, repoForkCore.ID)
	if err != nil {
		return sha.None, nil, fmt.Errorf("failed to find fork repo: %w", err)
	}

	if repoFork.ForkID == 0 {
		return sha.None, nil, errors.InvalidArgument("Repository is not a fork.")
	}

	repoUpstreamCore, err := c.repoFinder.FindByID(ctx, repoFork.ForkID)
	if err != nil {
		return sha.None, nil, fmt.Errorf("failed to find upstream repo: %w", err)
	}

	if err = apiauth.CheckRepo(
		ctx,
		c.authorizer,
		session,
		repoUpstreamCore,
		enum.PermissionRepoView,
	); errors.Is(err, apiauth.ErrForbidden) {
		return sha.None, nil, usererror.BadRequest(
			"Not enough permissions to view the upstream repository.",
		)
	} else if err != nil {
		return sha.None, nil, fmt.Errorf("failed to check access to upstream repo: %w", err)
	}

	upstreamSHA, err := getSHA(git.CreateReadParams(repoUpstreamCore))
	if err != nil {
		return sha.None, nil, fmt.Errorf("failed to get upstream repo SHA: %w", err)
	}

	writeParams, err := controller.CreateRPCSystemReferencesWriteParams(ctx, c.urlProvider, session, repoForkCore)
	if err != nil {
		return sha.None, nil, fmt.Errorf("failed to create RPC write params: %w", err)
	}

	_, err = c.git.FetchObjects(ctx, &git.FetchObjectsParams{
		WriteParams: writeParams,
		Source:      repoUpstreamCore.GitUID,
		ObjectSHAs:  []sha.SHA{upstreamSHA},
	})
	if err != nil {
		return sha.None, nil, fmt.Errorf("failed to fetch commit from upstream repo: %w", err)
	}

	return upstreamSHA, repoUpstreamCore, nil
}

const dotRangeUpstreamMarker = "upstream:"

type DotRange struct {
	BaseRef      string
	BaseUpstream bool
	HeadRef      string
	HeadUpstream bool
	MergeBase    bool
}

func (r DotRange) String() string {
	sb := strings.Builder{}

	if r.BaseUpstream {
		sb.WriteString(dotRangeUpstreamMarker)
	}
	sb.WriteString(r.BaseRef)

	sb.WriteString("..")
	if r.MergeBase {
		sb.WriteByte('.')
	}

	if r.HeadUpstream {
		sb.WriteString(dotRangeUpstreamMarker)
	}
	sb.WriteString(r.HeadRef)

	return sb.String()
}

func parseDotRangePath(path string) (DotRange, error) {
	mergeBase := true
	parts := strings.SplitN(path, "...", 2)
	if len(parts) != 2 {
		mergeBase = false
		parts = strings.SplitN(path, "..", 2)
		if len(parts) != 2 {
			return DotRange{}, usererror.BadRequestf("Invalid format %q", path)
		}
	}

	dotRange, err := makeDotRange(parts[0], parts[1], mergeBase)
	if err != nil {
		return DotRange{}, err
	}

	return dotRange, nil
}

func makeDotRange(base, head string, mergeBase bool) (DotRange, error) {
	dotRange := DotRange{
		BaseRef:   base,
		HeadRef:   head,
		MergeBase: mergeBase,
	}

	dotRange.BaseRef, dotRange.BaseUpstream = strings.CutPrefix(dotRange.BaseRef, dotRangeUpstreamMarker)
	dotRange.HeadRef, dotRange.HeadUpstream = strings.CutPrefix(dotRange.HeadRef, dotRangeUpstreamMarker)

	if dotRange.BaseUpstream && dotRange.HeadUpstream {
		return DotRange{}, usererror.BadRequestf("Only one upstream reference is allowed: %q", dotRange.String())
	}

	return dotRange, nil
}
