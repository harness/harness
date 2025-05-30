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

package githook

import (
	"context"
	"fmt"

	"github.com/harness/gitness/app/api/controller/limiter"
	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/auth/authz"
	gitevents "github.com/harness/gitness/app/events/git"
	repoevents "github.com/harness/gitness/app/events/repo"
	"github.com/harness/gitness/app/services/protection"
	"github.com/harness/gitness/app/services/refcache"
	"github.com/harness/gitness/app/services/settings"
	"github.com/harness/gitness/app/sse"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/app/url"
	"github.com/harness/gitness/audit"
	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/git/api"
	"github.com/harness/gitness/git/hook"
	"github.com/harness/gitness/git/sha"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

type Controller struct {
	authorizer          authz.Authorizer
	principalStore      store.PrincipalStore
	repoStore           store.RepoStore
	repoFinder          refcache.RepoFinder
	gitReporter         *gitevents.Reporter
	repoReporter        *repoevents.Reporter
	git                 git.Interface
	pullreqStore        store.PullReqStore
	urlProvider         url.Provider
	protectionManager   *protection.Manager
	limiter             limiter.ResourceLimiter
	settings            *settings.Service
	preReceiveExtender  PreReceiveExtender
	updateExtender      UpdateExtender
	postReceiveExtender PostReceiveExtender
	sseStreamer         sse.Streamer
	lfsStore            store.LFSObjectStore
	auditService        audit.Service
}

func NewController(
	authorizer authz.Authorizer,
	principalStore store.PrincipalStore,
	repoStore store.RepoStore,
	repoFinder refcache.RepoFinder,
	gitReporter *gitevents.Reporter,
	repoReporter *repoevents.Reporter,
	git git.Interface,
	pullreqStore store.PullReqStore,
	urlProvider url.Provider,
	protectionManager *protection.Manager,
	limiter limiter.ResourceLimiter,
	settings *settings.Service,
	preReceiveExtender PreReceiveExtender,
	updateExtender UpdateExtender,
	postReceiveExtender PostReceiveExtender,
	sseStreamer sse.Streamer,
	lfsStore store.LFSObjectStore,
	auditService audit.Service,
) *Controller {
	return &Controller{
		authorizer:          authorizer,
		principalStore:      principalStore,
		repoStore:           repoStore,
		repoFinder:          repoFinder,
		gitReporter:         gitReporter,
		repoReporter:        repoReporter,
		git:                 git,
		pullreqStore:        pullreqStore,
		urlProvider:         urlProvider,
		protectionManager:   protectionManager,
		limiter:             limiter,
		settings:            settings,
		preReceiveExtender:  preReceiveExtender,
		updateExtender:      updateExtender,
		postReceiveExtender: postReceiveExtender,
		sseStreamer:         sseStreamer,
		lfsStore:            lfsStore,
		auditService:        auditService,
	}
}

func (c *Controller) getRepoCheckAccess(
	ctx context.Context,
	_ *auth.Session,
	repoID int64,
	_ enum.Permission,
) (*types.RepositoryCore, error) {
	if repoID < 1 {
		return nil, usererror.BadRequest("A valid repository reference must be provided.")
	}

	repo, err := c.repoFinder.FindByID(ctx, repoID)
	if err != nil {
		return nil, fmt.Errorf("failed to find repo with id %d: %w", repoID, err)
	}
	// repo state check is done in pre-receive.

	// TODO: execute permission check. block anything but Harness service?

	return repo, nil
}

// GetBaseSHAForScanningChanges returns the commit sha to which the new sha of the reference
// should be compared against when scanning incoming changes.
// NOTE: If no such a sha exists, then (sha.None, false, nil) is returned.
// This will happen in case the default branch doesn't exist yet.
func GetBaseSHAForScanningChanges(
	ctx context.Context,
	rgit RestrictedGIT,
	repo *types.RepositoryCore,
	env hook.Environment,
	refUpdates []hook.ReferenceUpdate,
	findBaseFor hook.ReferenceUpdate,
) (sha.SHA, bool, error) {
	// always return old SHA of ref if possible (even if ref was deleted, that's on the caller)
	if !findBaseFor.Old.IsNil() {
		return findBaseFor.Old, true, nil
	}

	// reference is just being created.
	// For now we use default branch as a fallback (can be optimized to most recent commit on reference that exists)
	dfltBranchFullRef := api.BranchPrefix + repo.DefaultBranch
	for _, refUpdate := range refUpdates {
		if refUpdate.Ref != dfltBranchFullRef {
			continue
		}

		// default branch is being updated as part of push - make sure we use OLD default branch sha for comparison
		if !refUpdate.Old.IsNil() {
			return refUpdate.Old, true, nil
		}

		// default branch is being created - no fallback available
		return sha.None, false, nil
	}

	// read default branch from git
	dfltBranchOut, err := rgit.GetBranch(ctx, &git.GetBranchParams{
		ReadParams: git.ReadParams{
			RepoUID:             repo.GitUID,
			AlternateObjectDirs: env.AlternateObjectDirs,
		},
		BranchName: repo.DefaultBranch,
	})
	if errors.IsNotFound(err) {
		// this happens for empty repo's where the default branch wasn't created yet.
		return sha.None, false, nil
	}
	if err != nil {
		return sha.None, false, fmt.Errorf("failed to get default branch from git: %w", err)
	}

	return dfltBranchOut.Branch.SHA, true, nil
}

func isForcePush(
	ctx context.Context,
	rgit RestrictedGIT,
	gitUID string,
	alternateObjectDirs []string,
	refUpdate hook.ReferenceUpdate,
) (bool, error) {
	if refUpdate.Old.IsNil() || refUpdate.New.IsNil() {
		return false, nil
	}

	if isTag(refUpdate.Ref) {
		return true, nil
	}

	result, err := rgit.IsAncestor(ctx, git.IsAncestorParams{
		ReadParams: git.ReadParams{
			RepoUID:             gitUID,
			AlternateObjectDirs: alternateObjectDirs,
		},
		AncestorCommitSHA:   refUpdate.Old,
		DescendantCommitSHA: refUpdate.New,
	})
	if err != nil {
		return false, err
	}

	return !result.Ancestor, nil
}
