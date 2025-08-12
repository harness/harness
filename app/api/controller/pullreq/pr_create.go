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
	"strconv"
	"strings"
	"time"

	apiauth "github.com/harness/gitness/app/api/auth"
	"github.com/harness/gitness/app/api/controller"
	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/app/auth"
	pullreqevents "github.com/harness/gitness/app/events/pullreq"
	"github.com/harness/gitness/app/services/codeowners"
	"github.com/harness/gitness/app/services/instrument"
	labelsvc "github.com/harness/gitness/app/services/label"
	"github.com/harness/gitness/app/services/protection"
	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/git"
	gitenum "github.com/harness/gitness/git/enum"
	"github.com/harness/gitness/git/sha"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
	"golang.org/x/exp/maps"
)

type CreateInput struct {
	IsDraft bool `json:"is_draft"`

	Title       string `json:"title"`
	Description string `json:"description"`

	SourceRepoRef string `json:"source_repo_ref"`
	SourceBranch  string `json:"source_branch"`
	TargetBranch  string `json:"target_branch"`

	ReviewerIDs []int64 `json:"reviewer_ids"`

	Labels []*types.PullReqLabelAssignInput `json:"labels"`

	BypassRules bool `json:"bypass_rules"`
}

func (in *CreateInput) Sanitize() error {
	in.Title = strings.TrimSpace(in.Title)
	in.Description = strings.TrimSpace(in.Description)

	if err := validateTitle(in.Title); err != nil {
		return err
	}

	if err := validateDescription(in.Description); err != nil {
		return err
	}

	return nil
}

// Create creates a new pull request.
func (c *Controller) Create(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	in *CreateInput,
) (*types.PullReq, error) {
	if err := in.Sanitize(); err != nil {
		return nil, err
	}

	targetRepo, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoPush)
	if err != nil {
		return nil, fmt.Errorf("failed to acquire access to target repo: %w", err)
	}

	sourceRepo := targetRepo
	if in.SourceRepoRef != "" {
		sourceRepo, err = c.getRepoCheckAccess(ctx, session, in.SourceRepoRef, enum.PermissionRepoPush)
		if err != nil {
			return nil, fmt.Errorf("failed to acquire access to source repo: %w", err)
		}
	}

	if sourceRepo.ID == targetRepo.ID && in.TargetBranch == in.SourceBranch {
		return nil, usererror.BadRequest("Target and source branch can't be the same")
	}

	var sourceSHA sha.SHA

	if sourceSHA, err = c.verifyBranchExistence(ctx, sourceRepo, in.SourceBranch); err != nil {
		return nil, err
	}

	if _, err = c.verifyBranchExistence(ctx, targetRepo, in.TargetBranch); err != nil {
		return nil, err
	}

	if err = c.checkIfAlreadyExists(ctx, targetRepo.ID, sourceRepo.ID, in.TargetBranch, in.SourceBranch); err != nil {
		return nil, err
	}

	targetWriteParams, err := controller.CreateRPCSystemReferencesWriteParams(
		ctx, c.urlProvider, session, targetRepo,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create RPC write params: %w", err)
	}

	mergeBaseResult, err := c.git.MergeBase(ctx, git.MergeBaseParams{
		ReadParams: git.ReadParams{RepoUID: sourceRepo.GitUID},
		Ref1:       in.SourceBranch,
		Ref2:       in.TargetBranch,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to find merge base: %w", err)
	}

	mergeBaseSHA := mergeBaseResult.MergeBaseSHA

	if mergeBaseSHA == sourceSHA {
		return nil, usererror.BadRequest("The source branch doesn't contain any new commits")
	}

	prStats, err := c.git.DiffStats(ctx, &git.DiffParams{
		ReadParams: git.ReadParams{RepoUID: targetRepo.GitUID},
		BaseRef:    mergeBaseSHA.String(),
		HeadRef:    sourceSHA.String(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch PR diff stats: %w", err)
	}

	var pr *types.PullReq

	targetRepoID := targetRepo.ID

	var activitySeq int64

	// Payload based reviewers

	reviewerInputMap, err := c.preparePayloadReviewers(ctx, session, in.ReviewerIDs, targetRepo)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare reviewers: %w", err)
	}

	payloadReviewerIDs := maps.Keys(reviewerInputMap)

	if len(reviewerInputMap) > 0 {
		activitySeq++
	}

	// Rules based reviewers

	codeownerReviewers, defaultReviewers, err := c.prepareRuleReviewers(
		ctx, session, targetRepo, in, mergeBaseSHA.String(), sourceSHA.String(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare codeowners: %w", err)
	}

	if len(codeownerReviewers) > 0 {
		activitySeq++
	}
	if len(defaultReviewers) > 0 {
		activitySeq++
	}

	for _, codeownerReviewer := range codeownerReviewers {
		if _, ok := reviewerInputMap[codeownerReviewer.ID]; !ok {
			reviewerInputMap[codeownerReviewer.ID] = codeownerReviewer
		}
	}

	for _, defaultReviewer := range defaultReviewers {
		if _, ok := reviewerInputMap[defaultReviewer.ID]; !ok {
			reviewerInputMap[defaultReviewer.ID] = defaultReviewer
		}
	}

	// Prepare label assign input

	var labelAssignOuts []*labelsvc.AssignToPullReqOut

	labelAssignInputMap, err := c.prepareLabels(
		ctx, in.Labels, session.Principal.ID, targetRepo.ID, targetRepo.ParentID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare labels: %w", err)
	}

	if len(labelAssignInputMap) > 0 {
		activitySeq++
	}

	err = controller.TxOptLock(ctx, c.tx, func(ctx context.Context) error {
		// Always re-fetch at the start of the transaction because the repo we have is from a cache.

		targetRepoFull, err := c.repoStore.Find(ctx, targetRepoID)
		if err != nil {
			return fmt.Errorf("failed to find repository: %w", err)
		}

		// Update the repository's pull request sequence number

		targetRepoFull.PullReqSeq++
		err = c.repoStore.Update(ctx, targetRepoFull)
		if err != nil {
			return fmt.Errorf("failed to update pullreq sequence number: %w", err)
		}

		// Create pull request in the DB

		pr = newPullReq(session, targetRepoFull.PullReqSeq, sourceRepo.ID, targetRepo.ID, in, sourceSHA, mergeBaseSHA)
		pr.Stats = types.PullReqStats{
			DiffStats:       types.NewDiffStats(prStats.Commits, prStats.FilesChanged, prStats.Additions, prStats.Deletions),
			Conversations:   0,
			UnresolvedCount: 0,
		}

		targetRepo = targetRepoFull.Core()

		pr.ActivitySeq = activitySeq

		err = c.pullreqStore.Create(ctx, pr)
		if err != nil {
			return fmt.Errorf("pullreq creation failed: %w", err)
		}

		// reset pr activity seq: we increment pr.ActivitySeq on activity creation
		pr.ActivitySeq = 0

		// Create reviewers and assign labels

		if err = c.createReviewers(ctx, session, reviewerInputMap, targetRepo, pr); err != nil {
			return fmt.Errorf("failed to create reviewers: %w", err)
		}

		if labelAssignOuts, err = c.assignLabels(ctx, pr, session.Principal.ID, labelAssignInputMap); err != nil {
			return fmt.Errorf("failed to assign labels: %w", err)
		}

		// Create PR head reference in the git repository

		err = c.git.UpdateRef(ctx, git.UpdateRefParams{
			WriteParams: targetWriteParams,
			Name:        strconv.FormatInt(targetRepoFull.PullReqSeq, 10),
			Type:        gitenum.RefTypePullReqHead,
			NewValue:    sourceSHA,
			OldValue:    sha.None, // we don't care about the old value
		})
		if err != nil {
			return fmt.Errorf("failed to create PR head ref: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create pullreq: %w", err)
	}

	c.storeCreateReviewerActivity(
		ctx, pr, session.Principal.ID, payloadReviewerIDs, enum.PullReqReviewerTypeRequested,
	)
	c.storeCreateReviewerActivity(
		ctx, pr, session.Principal.ID, maps.Keys(codeownerReviewers), enum.PullReqReviewerTypeCodeOwners,
	)
	c.storeCreateReviewerActivity(
		ctx, pr, session.Principal.ID, maps.Keys(defaultReviewers), enum.PullReqReviewerTypeDefault,
	)

	backfillWithLabelAssignInfo(pr, labelAssignOuts)
	c.storeLabelAssignActivity(ctx, pr, session.Principal.ID, labelAssignOuts)

	c.eventReporter.Created(ctx, &pullreqevents.CreatedPayload{
		Base:         eventBase(pr, &session.Principal),
		SourceBranch: in.SourceBranch,
		TargetBranch: in.TargetBranch,
		SourceSHA:    sourceSHA.String(),
		ReviewerIDs:  maps.Keys(reviewerInputMap),
	})

	c.sseStreamer.Publish(ctx, targetRepo.ParentID, enum.SSETypePullReqUpdated, pr)

	err = c.instrumentation.Track(ctx, instrument.Event{
		Type:      instrument.EventTypeCreatePullRequest,
		Principal: session.Principal.ToPrincipalInfo(),
		Path:      sourceRepo.Path,
		Properties: map[instrument.Property]any{
			instrument.PropertyRepositoryID:   sourceRepo.ID,
			instrument.PropertyRepositoryName: sourceRepo.Identifier,
			instrument.PropertyPullRequestID:  pr.Number,
		},
	})
	if err != nil {
		log.Ctx(ctx).Warn().Msgf("failed to insert instrumentation record for create pull request operation: %s", err)
	}

	return pr, nil
}

// preparePayloadReviewers fetches principal data and checks principal repo access and permissions.
// The data recency is not critical: principals might change and the op will either be valid or fail.
// Because it makes db calls, we use it before, i.e. outside of the PR creation tx.
func (c *Controller) preparePayloadReviewers(
	ctx context.Context,
	session *auth.Session,
	reviewers []int64,
	repo *types.RepositoryCore,
) (map[int64]*types.PrincipalInfo, error) {
	if len(reviewers) == 0 {
		return map[int64]*types.PrincipalInfo{}, nil
	}

	principalEmailMap := make(map[int64]*types.PrincipalInfo, len(reviewers))

	for _, id := range reviewers {
		if id == session.Principal.ID {
			return nil, usererror.BadRequest("PR creator cannot be added as a reviewer.")
		}

		reviewerPrincipal, err := c.principalStore.Find(ctx, id)
		if err != nil {
			return nil, usererror.BadRequest("Failed to find principal reviewer.")
		}

		// TODO: To check the reviewer's access to the repo we create a dummy session object. Fix it.
		if err = apiauth.CheckRepo(
			ctx,
			c.authorizer,
			&auth.Session{
				Principal: *reviewerPrincipal,
				Metadata:  nil,
			},
			repo,
			enum.PermissionRepoReview,
		); err != nil {
			if !errors.Is(err, apiauth.ErrForbidden) {
				return nil, usererror.BadRequest(
					"The reviewer doesn't have enough permissions for the repository.",
				)
			}
			return nil, fmt.Errorf("reviewer principal %s check repo access error: %w", reviewerPrincipal.UID, err)
		}

		principalEmailMap[reviewerPrincipal.ID] = reviewerPrincipal.ToPrincipalInfo()
	}

	return principalEmailMap, nil
}

func (c *Controller) prepareRuleReviewers(
	ctx context.Context,
	session *auth.Session,
	targetRepo *types.RepositoryCore,
	in *CreateInput,
	mergeBaseSHA string,
	sourceSHA string,
) (map[int64]*types.PrincipalInfo, map[int64]*types.PrincipalInfo, error) {
	rules, isRepoOwner, err := c.fetchRules(ctx, session, targetRepo)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to fetch protection rules: %w", err)
	}

	out, _, err := rules.CreatePullReqVerify(ctx, protection.CreatePullReqVerifyInput{
		ResolveUserGroupID: c.userGroupService.ListUserIDsByGroupIDs,
		Actor:              &session.Principal,
		AllowBypass:        in.BypassRules,
		IsRepoOwner:        isRepoOwner,
		DefaultBranch:      targetRepo.DefaultBranch,
		TargetBranch:       in.TargetBranch,
		RepoID:             targetRepo.ID,
		RepoIdentifier:     targetRepo.Identifier,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to verify protection rules: %w", err)
	}

	if !out.RequestCodeOwners && len(out.DefaultReviewerIDs) == 0 {
		return map[int64]*types.PrincipalInfo{}, map[int64]*types.PrincipalInfo{}, nil
	}

	codeownerReviewers := make(map[int64]*types.PrincipalInfo)
	if out.RequestCodeOwners {
		codeownerReviewers, err = c.getApplicableCodeOwners(
			ctx, targetRepo, in.TargetBranch, mergeBaseSHA, sourceSHA,
		)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to prepare code owner reviewers: %w", err)
		}

		// ensure we remove author from list
		delete(codeownerReviewers, session.Principal.ID)
	}

	defaultReviewers := make(map[int64]*types.PrincipalInfo, len(out.DefaultReviewerIDs))
	if len(out.DefaultReviewerIDs) > 0 {
		defaultReviewers, err = c.getDefaultReviewers(ctx, out.DefaultReviewerIDs)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to prepare default reviewers: %w", err)
		}

		// ensure we remove author from list
		delete(defaultReviewers, session.Principal.ID)
	}

	return codeownerReviewers, defaultReviewers, nil
}

func (c *Controller) getApplicableCodeOwners(
	ctx context.Context,
	targetRepo *types.RepositoryCore,
	targetBranch string,
	mergeBaseSHA string,
	sourceSHA string,
) (map[int64]*types.PrincipalInfo, error) {
	applicableCodeOwners, err := c.codeOwners.GetApplicableCodeOwners(
		ctx, targetRepo, targetBranch, mergeBaseSHA, sourceSHA,
	)
	if errors.Is(err, codeowners.ErrNotFound) {
		return map[int64]*types.PrincipalInfo{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get applicable code owners: %w", err)
	}

	var emails []string
	for _, entry := range applicableCodeOwners.Entries {
		emails = append(emails, entry.Owners...)
	}

	principals, err := c.principalStore.FindManyByEmail(ctx, emails)
	if err != nil {
		return nil, fmt.Errorf("failed to find many principals by email: %w", err)
	}

	principalInfoMap := make(map[int64]*types.PrincipalInfo, len(principals))
	for _, principal := range principals {
		principalInfoMap[principal.ID] = principal.ToPrincipalInfo()
	}

	return principalInfoMap, nil
}

func (c *Controller) getDefaultReviewers(
	ctx context.Context,
	reviewerIDs []int64,
) (map[int64]*types.PrincipalInfo, error) {
	principals, err := c.principalInfoCache.Map(ctx, reviewerIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to find principal infos by ids: %w", err)
	}

	return principals, nil
}

func (c *Controller) createReviewers(
	ctx context.Context,
	session *auth.Session,
	principalInfos map[int64]*types.PrincipalInfo,
	repo *types.RepositoryCore,
	pr *types.PullReq,
) error {
	if len(principalInfos) == 0 {
		return nil
	}

	reviewers := make([]*types.PullReqReviewer, len(principalInfos))

	var i int
	for _, principalInfo := range principalInfos {
		reviewer := newPullReqReviewer(
			session, pr, repo,
			principalInfo,
			session.Principal.ToPrincipalInfo(),
			enum.PullReqReviewerTypeRequested,
			&ReviewerAddInput{
				ReviewerID: principalInfo.ID,
			},
		)

		if err := c.reviewerStore.Create(ctx, reviewer); err != nil {
			return fmt.Errorf("failed to create pull request reviewer: %w", err)
		}

		reviewers[i] = reviewer
		i++
	}

	return nil
}

func (c *Controller) storeCreateReviewerActivity(
	ctx context.Context,
	pr *types.PullReq,
	authorID int64,
	reviewerIDs []int64,
	reviewerType enum.PullReqReviewerType,
) {
	if len(reviewerIDs) == 0 {
		return
	}

	pr.ActivitySeq++

	payload := &types.PullRequestActivityPayloadReviewerAdd{
		ReviewerType: reviewerType,
		PrinciaplIDs: reviewerIDs,
	}

	metadata := &types.PullReqActivityMetadata{
		Mentions: &types.PullReqActivityMentionsMetadata{IDs: reviewerIDs},
	}

	if _, err := c.activityStore.CreateWithPayload(
		ctx, pr, authorID, payload, metadata,
	); err != nil {
		log.Ctx(ctx).Err(err).Msgf(
			"failed to write create %s reviewer pull req activity", reviewerType,
		)
	}
}

// prepareLabels fetches data (labels and label values) necessary for the pr label assignment.
// The data recency is not critical: labels/values might change and the op will either be valid or fail.
// Because it makes db calls, we use it before, i.e. outside of the PR creation tx.
func (c *Controller) prepareLabels(
	ctx context.Context,
	labelAssignInputs []*types.PullReqLabelAssignInput,
	principalID int64,
	repoID int64,
	repoParentID int64,
) (map[*types.PullReqLabelAssignInput]*labelsvc.WithValue, error) {
	labelAssignInputMap := make(
		map[*types.PullReqLabelAssignInput]*labelsvc.WithValue,
		len(labelAssignInputs),
	)

	for _, labelAssignInput := range labelAssignInputs {
		labelWithValue, err := c.labelSvc.PreparePullReqLabel(
			ctx,
			principalID, repoID, repoParentID,
			labelAssignInput,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to prepare label assignment data: %w", err)
		}

		labelAssignInputMap[labelAssignInput] = &labelWithValue
	}

	return labelAssignInputMap, nil
}

// assignLabels is a critical op for PR creation, so we use it in the PR creation tx.
func (c *Controller) assignLabels(
	ctx context.Context,
	pr *types.PullReq,
	principalID int64,
	labelAssignInputMap map[*types.PullReqLabelAssignInput]*labelsvc.WithValue,
) ([]*labelsvc.AssignToPullReqOut, error) {
	assignOuts := make([]*labelsvc.AssignToPullReqOut, len(labelAssignInputMap))

	var err error
	var i int
	for labelAssignInput, labelWithValue := range labelAssignInputMap {
		assignOuts[i], err = c.labelSvc.AssignToPullReqOnCreation(
			ctx,
			pr.ID,
			principalID,
			labelWithValue,
			labelAssignInput,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to assign label to pullreq: %w", err)
		}

		i++
	}

	return assignOuts, nil
}

func backfillWithLabelAssignInfo(
	pr *types.PullReq,
	labelAssignOuts []*labelsvc.AssignToPullReqOut,
) {
	pr.Labels = make([]*types.LabelPullReqAssignmentInfo, len(labelAssignOuts))
	for i, assignOut := range labelAssignOuts {
		pr.Labels[i] = assignOut.ToLabelPullReqAssignmentInfo()
	}
}

func (c *Controller) storeLabelAssignActivity(
	ctx context.Context,
	pr *types.PullReq,
	principalID int64,
	labelAssignOuts []*labelsvc.AssignToPullReqOut,
) {
	if len(labelAssignOuts) == 0 {
		return
	}

	pr.ActivitySeq++

	payload := &types.PullRequestActivityLabels{
		Labels: make([]*types.PullRequestActivityLabelBase, len(labelAssignOuts)),
		Type:   enum.LabelActivityAssign,
	}

	for i, out := range labelAssignOuts {
		var value *string
		var valueColor *enum.LabelColor
		if out.NewLabelValue != nil {
			value = &out.NewLabelValue.Value
			valueColor = &out.NewLabelValue.Color
		}
		payload.Labels[i] = &types.PullRequestActivityLabelBase{
			Label:      out.Label.Key,
			LabelColor: out.Label.Color,
			LabelScope: out.Label.Scope,
			Value:      value,
			ValueColor: valueColor,
		}
	}

	if _, err := c.activityStore.CreateWithPayload(
		ctx, pr, principalID, payload, nil,
	); err != nil {
		log.Ctx(ctx).Err(err).Msg("failed to write label assign pull req activity")
	}
}

// newPullReq creates new pull request object.
func newPullReq(
	session *auth.Session,
	number int64,
	sourceRepoID int64,
	targetRepoID int64,
	in *CreateInput,
	sourceSHA, mergeBaseSHA sha.SHA,
) *types.PullReq {
	now := time.Now().UnixMilli()
	return &types.PullReq{
		ID:                0, // the ID will be populated in the data layer
		Version:           0,
		Number:            number,
		CreatedBy:         session.Principal.ID,
		Created:           now,
		Updated:           now,
		Edited:            now,
		State:             enum.PullReqStateOpen,
		IsDraft:           in.IsDraft,
		Title:             in.Title,
		Description:       in.Description,
		SourceRepoID:      sourceRepoID,
		SourceBranch:      in.SourceBranch,
		SourceSHA:         sourceSHA.String(),
		TargetRepoID:      targetRepoID,
		TargetBranch:      in.TargetBranch,
		ActivitySeq:       0,
		MergedBy:          nil,
		Merged:            nil,
		MergeMethod:       nil,
		MergeBaseSHA:      mergeBaseSHA.String(),
		MergeCheckStatus:  enum.MergeCheckStatusUnchecked,
		RebaseCheckStatus: enum.MergeCheckStatusUnchecked,
		Author:            *session.Principal.ToPrincipalInfo(),
		Merger:            nil,
	}
}
