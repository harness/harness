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
	"fmt"
	"unicode/utf8"

	apiauth "github.com/harness/gitness/app/api/auth"
	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/auth/authz"
	pullreqevents "github.com/harness/gitness/app/events/pullreq"
	"github.com/harness/gitness/app/services/codecomments"
	"github.com/harness/gitness/app/services/codeowners"
	"github.com/harness/gitness/app/services/label"
	locker "github.com/harness/gitness/app/services/locker"
	"github.com/harness/gitness/app/services/migrate"
	"github.com/harness/gitness/app/services/protection"
	"github.com/harness/gitness/app/services/pullreq"
	"github.com/harness/gitness/app/sse"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/app/url"
	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/git"
	gitenum "github.com/harness/gitness/git/enum"
	"github.com/harness/gitness/git/sha"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

type Controller struct {
	tx                  dbtx.Transactor
	urlProvider         url.Provider
	authorizer          authz.Authorizer
	pullreqStore        store.PullReqStore
	activityStore       store.PullReqActivityStore
	codeCommentView     store.CodeCommentView
	reviewStore         store.PullReqReviewStore
	reviewerStore       store.PullReqReviewerStore
	repoStore           store.RepoStore
	principalStore      store.PrincipalStore
	principalInfoCache  store.PrincipalInfoCache
	fileViewStore       store.PullReqFileViewStore
	membershipStore     store.MembershipStore
	checkStore          store.CheckStore
	git                 git.Interface
	eventReporter       *pullreqevents.Reporter
	codeCommentMigrator *codecomments.Migrator
	pullreqService      *pullreq.Service
	protectionManager   *protection.Manager
	sseStreamer         sse.Streamer
	codeOwners          *codeowners.Service
	locker              *locker.Locker
	importer            *migrate.PullReq
	labelSvc            *label.Service
}

func NewController(
	tx dbtx.Transactor,
	urlProvider url.Provider,
	authorizer authz.Authorizer,
	pullreqStore store.PullReqStore,
	pullreqActivityStore store.PullReqActivityStore,
	codeCommentView store.CodeCommentView,
	pullreqReviewStore store.PullReqReviewStore,
	pullreqReviewerStore store.PullReqReviewerStore,
	repoStore store.RepoStore,
	principalStore store.PrincipalStore,
	principalInfoCache store.PrincipalInfoCache,
	fileViewStore store.PullReqFileViewStore,
	membershipStore store.MembershipStore,
	checkStore store.CheckStore,
	git git.Interface,
	eventReporter *pullreqevents.Reporter,
	codeCommentMigrator *codecomments.Migrator,
	pullreqService *pullreq.Service,
	protectionManager *protection.Manager,
	sseStreamer sse.Streamer,
	codeowners *codeowners.Service,
	locker *locker.Locker,
	importer *migrate.PullReq,
	labelSvc *label.Service,
) *Controller {
	return &Controller{
		tx:                  tx,
		urlProvider:         urlProvider,
		authorizer:          authorizer,
		pullreqStore:        pullreqStore,
		activityStore:       pullreqActivityStore,
		codeCommentView:     codeCommentView,
		reviewStore:         pullreqReviewStore,
		reviewerStore:       pullreqReviewerStore,
		repoStore:           repoStore,
		principalStore:      principalStore,
		principalInfoCache:  principalInfoCache,
		fileViewStore:       fileViewStore,
		membershipStore:     membershipStore,
		checkStore:          checkStore,
		git:                 git,
		codeCommentMigrator: codeCommentMigrator,
		eventReporter:       eventReporter,
		pullreqService:      pullreqService,
		protectionManager:   protectionManager,
		sseStreamer:         sseStreamer,
		codeOwners:          codeowners,
		locker:              locker,
		importer:            importer,
		labelSvc:            labelSvc,
	}
}

func (c *Controller) verifyBranchExistence(ctx context.Context,
	repo *types.Repository, branch string,
) (sha.SHA, error) {
	if branch == "" {
		return sha.SHA{}, usererror.BadRequest("branch name can't be empty")
	}

	ref, err := c.git.GetRef(ctx,
		git.GetRefParams{
			ReadParams: git.ReadParams{RepoUID: repo.GitUID},
			Name:       branch,
			Type:       gitenum.RefTypeBranch,
		})
	if errors.AsStatus(err) == errors.StatusNotFound {
		return sha.SHA{}, usererror.BadRequest(
			fmt.Sprintf("branch %q does not exist in the repository %q", branch, repo.Identifier))
	}
	if err != nil {
		return sha.SHA{}, fmt.Errorf(
			"failed to check existence of the branch %q in the repository %q: %w",
			branch, repo.Identifier, err)
	}

	return ref.SHA, nil
}

func (c *Controller) getRepo(ctx context.Context, repoRef string) (*types.Repository, error) {
	if repoRef == "" {
		return nil, usererror.BadRequest("A valid repository reference must be provided.")
	}

	repo, err := c.repoStore.FindByRef(ctx, repoRef)
	if err != nil {
		return nil, fmt.Errorf("failed to find repository: %w", err)
	}

	if repo.State != enum.RepoStateActive {
		return nil, usererror.BadRequest("Repository is not ready to use.")
	}

	return repo, nil
}

func (c *Controller) getRepoCheckAccess(ctx context.Context,
	session *auth.Session, repoRef string, reqPermission enum.Permission,
) (*types.Repository, error) {
	repo, err := c.getRepo(ctx, repoRef)
	if err != nil {
		return nil, err
	}

	if err = apiauth.CheckRepo(ctx, c.authorizer, session, repo, reqPermission); err != nil {
		return nil, fmt.Errorf("access check failed: %w", err)
	}

	return repo, nil
}

func (c *Controller) getCommentForPR(
	ctx context.Context,
	pr *types.PullReq,
	commentID int64,
) (*types.PullReqActivity, error) {
	if commentID <= 0 {
		return nil, usererror.BadRequest("A valid comment ID must be provided.")
	}

	comment, err := c.activityStore.Find(ctx, commentID)
	if err != nil {
		return nil, fmt.Errorf("failed to find comment by ID: %w", err)
	}

	if comment == nil {
		return nil, usererror.ErrNotFound
	}

	if comment.Deleted != nil || comment.RepoID != pr.TargetRepoID || comment.PullReqID != pr.ID {
		return nil, usererror.ErrNotFound
	}

	if comment.Kind == enum.PullReqActivityKindSystem {
		return nil, usererror.BadRequest("Can't update a comment created by the system.")
	}

	if comment.Type != enum.PullReqActivityTypeComment && comment.Type != enum.PullReqActivityTypeCodeComment {
		return nil, usererror.BadRequest("Only comments and code comments can be edited.")
	}

	return comment, nil
}

func (c *Controller) getCommentCheckEditAccess(ctx context.Context,
	session *auth.Session, pr *types.PullReq, commentID int64,
) (*types.PullReqActivity, error) {
	comment, err := c.getCommentForPR(ctx, pr, commentID)
	if err != nil {
		return nil, err
	}

	if comment.CreatedBy != session.Principal.ID {
		return nil, usererror.BadRequest("Only own comments may be updated.")
	}

	return comment, nil
}

func (c *Controller) getCommentCheckChangeStatusAccess(ctx context.Context,
	pr *types.PullReq, commentID int64,
) (*types.PullReqActivity, error) {
	comment, err := c.getCommentForPR(ctx, pr, commentID)
	if err != nil {
		return nil, err
	}

	if comment.SubOrder != 0 {
		return nil, usererror.BadRequest("Can't change status of replies.")
	}

	return comment, nil
}

func (c *Controller) checkIfAlreadyExists(ctx context.Context,
	targetRepoID, sourceRepoID int64, targetBranch, sourceBranch string,
) error {
	existing, err := c.pullreqStore.List(ctx, &types.PullReqFilter{
		SourceRepoID: sourceRepoID,
		SourceBranch: sourceBranch,
		TargetRepoID: targetRepoID,
		TargetBranch: targetBranch,
		States:       []enum.PullReqState{enum.PullReqStateOpen},
		Size:         1,
		Sort:         enum.PullReqSortNumber,
		Order:        enum.OrderAsc,
	})
	if err != nil {
		return fmt.Errorf("failed to get existing pull requests: %w", err)
	}
	if len(existing) > 0 {
		return usererror.ConflictWithPayload(
			"a pull request for this target and source branch already exists",
			map[string]any{
				"type":   "pr already exists",
				"number": existing[0].Number,
				"title":  existing[0].Title,
			},
		)
	}

	return nil
}

func eventBase(pr *types.PullReq, principal *types.Principal) pullreqevents.Base {
	return pullreqevents.Base{
		PullReqID:    pr.ID,
		SourceRepoID: pr.SourceRepoID,
		TargetRepoID: pr.TargetRepoID,
		Number:       pr.Number,
		PrincipalID:  principal.ID,
	}
}

func validateTitle(title string) error {
	if title == "" {
		return usererror.BadRequest("pull request title can't be empty")
	}

	const maxLen = 256
	if utf8.RuneCountInString(title) > maxLen {
		return usererror.BadRequestf("pull request title is too long (maximum is %d characters)", maxLen)
	}

	return nil
}

func validateDescription(desc string) error {
	const maxLen = 64 << 10 // 64K
	if len(desc) > maxLen {
		return usererror.BadRequest("pull request description is too long")
	}

	return nil
}

func validateComment(desc string) error {
	const maxLen = 16 << 10 // 16K
	if len(desc) > maxLen {
		return usererror.BadRequest("pull request comment is too long")
	}

	return nil
}
