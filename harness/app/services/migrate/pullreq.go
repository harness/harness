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

package migrate

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/harness/gitness/app/services/refcache"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/app/url"
	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/git/parser"
	"github.com/harness/gitness/lock"
	gitness_store "github.com/harness/gitness/store"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

// PullReq is pull request migrate.
type PullReq struct {
	urlProvider                 url.Provider
	git                         git.Interface
	principalStore              store.PrincipalStore
	spaceStore                  store.SpaceStore
	repoStore                   store.RepoStore
	pullReqStore                store.PullReqStore
	pullReqActStore             store.PullReqActivityStore
	labelStore                  store.LabelStore
	labelValueStore             store.LabelValueStore
	pullReqLabelAssignmentStore store.PullReqLabelAssignmentStore
	pullReqReviewerStore        store.PullReqReviewerStore
	pullReqReviewStore          store.PullReqReviewStore
	repoFinder                  refcache.RepoFinder
	tx                          dbtx.Transactor
	mtxManager                  lock.MutexManager
}

func NewPullReq(
	urlProvider url.Provider,
	git git.Interface,
	principalStore store.PrincipalStore,
	spaceStore store.SpaceStore,
	repoStore store.RepoStore,
	pullReqStore store.PullReqStore,
	pullReqActStore store.PullReqActivityStore,
	labelStore store.LabelStore,
	labelValueStore store.LabelValueStore,
	pullReqLabelAssignmentStore store.PullReqLabelAssignmentStore,
	pullReqReviewerStore store.PullReqReviewerStore,
	pullReqReviewStore store.PullReqReviewStore,
	repoFinder refcache.RepoFinder,
	tx dbtx.Transactor,
	mtxManager lock.MutexManager,
) *PullReq {
	return &PullReq{
		urlProvider:                 urlProvider,
		git:                         git,
		principalStore:              principalStore,
		spaceStore:                  spaceStore,
		repoStore:                   repoStore,
		pullReqStore:                pullReqStore,
		pullReqActStore:             pullReqActStore,
		labelStore:                  labelStore,
		labelValueStore:             labelValueStore,
		pullReqLabelAssignmentStore: pullReqLabelAssignmentStore,
		pullReqReviewerStore:        pullReqReviewerStore,
		pullReqReviewStore:          pullReqReviewStore,
		repoFinder:                  repoFinder,
		tx:                          tx,
		mtxManager:                  mtxManager,
	}
}

type repoImportState struct {
	git                         git.Interface
	readParams                  git.ReadParams
	principalStore              store.PrincipalStore
	spaceStore                  store.SpaceStore
	pullReqStore                store.PullReqStore
	pullReqActivityStore        store.PullReqActivityStore
	labelStore                  store.LabelStore
	labelValueStore             store.LabelValueStore
	pullReqLabelAssignmentStore store.PullReqLabelAssignmentStore
	pullReqReviewerStore        store.PullReqReviewerStore
	pullReqReviewStore          store.PullReqReviewStore
	branchCheck                 map[string]*git.Branch
	principals                  map[string]*types.Principal
	unknownEmails               map[int]map[string]bool
	labels                      map[string]int64            // map for labels {"label.key":label.id,}
	labelValues                 map[int64]map[string]*int64 // map for label values {label.id:{"value-key":value-id,}}
	migrator                    types.Principal
	scope                       int64 // depth of space used for labels
}

// Import load provided pull requests in go-scm format and imports them.
//
//nolint:gocognit
func (migrate PullReq) Import(
	ctx context.Context,
	migrator types.Principal,
	repo *types.RepositoryCore,
	extPullReqs []*ExternalPullRequest,
) ([]*types.PullReq, error) {
	readParams := git.ReadParams{RepoUID: repo.GitUID}

	repoState := repoImportState{
		git:                         migrate.git,
		readParams:                  readParams,
		principalStore:              migrate.principalStore,
		spaceStore:                  migrate.spaceStore,
		pullReqStore:                migrate.pullReqStore,
		pullReqActivityStore:        migrate.pullReqActStore,
		labelStore:                  migrate.labelStore,
		labelValueStore:             migrate.labelValueStore,
		pullReqLabelAssignmentStore: migrate.pullReqLabelAssignmentStore,
		pullReqReviewerStore:        migrate.pullReqReviewerStore,
		pullReqReviewStore:          migrate.pullReqReviewStore,
		branchCheck:                 map[string]*git.Branch{},
		principals:                  map[string]*types.Principal{},
		unknownEmails:               map[int]map[string]bool{},
		labels:                      map[string]int64{},
		labelValues:                 map[int64]map[string]*int64{},
		migrator:                    migrator,
		scope:                       0,
	}

	pullReqUnique := map[int]ExternalPullRequest{}
	pullReqComments := map[*types.PullReq][]ExternalComment{}

	pullReqs := make([]*types.PullReq, 0, len(extPullReqs))
	// create the PR objects, one by one. Each pull request will mutate the repository object (to update the counters).
	for _, extPullReqData := range extPullReqs {
		extPullReq := &extPullReqData.PullRequest

		if _, exists := pullReqUnique[extPullReq.Number]; exists {
			return nil, errors.Conflict("duplicate pull request number %d", extPullReq.Number)
		}
		pullReqUnique[extPullReq.Number] = *extPullReqData

		pr, err := repoState.convertPullReq(ctx, repo, extPullReqData)
		if err != nil {
			return nil, fmt.Errorf("failed to import pull request %d: %w", extPullReq.Number, err)
		}

		pullReqs = append(pullReqs, pr)
		pullReqComments[pr] = extPullReqData.Comments
	}

	if len(pullReqs) == 0 { // nothing to do: exit early to avoid accessing the database
		return nil, nil
	}

	err := migrate.tx.WithTx(ctx, func(ctx context.Context) error {
		var deltaOpen, deltaClosed, deltaMerged int
		var maxNumber int64

		for _, pullReq := range pullReqs {
			if err := migrate.pullReqStore.Create(ctx, pullReq); err != nil {
				return fmt.Errorf("failed to import the pull request %d: %w", pullReq.Number, err)
			}
		}

		for _, pr := range pullReqs {
			extPullReqData := pullReqUnique[int(pr.Number)]

			_, err := repoState.createReviewers(ctx, repo, pr, extPullReqData.Reviewers)
			if err != nil {
				return fmt.Errorf("failed to create reviewers for PR %d: %w", pr.Number, err)
			}

			_, err = repoState.createReviews(ctx, repo, pr, extPullReqData.Reviews)
			if err != nil {
				return fmt.Errorf("failed to create reviews for PR %d: %w", pr.Number, err)
			}
		}

		for _, pullReq := range pullReqs {
			switch pullReq.State {
			case enum.PullReqStateOpen:
				deltaOpen++
			case enum.PullReqStateClosed:
				deltaClosed++
			case enum.PullReqStateMerged:
				deltaMerged++
			}

			if maxNumber < pullReq.Number {
				maxNumber = pullReq.Number
			}

			comments, err := repoState.createComments(ctx, repo, pullReq, pullReqComments[pullReq])
			if err != nil {
				return fmt.Errorf("failed to import pull request comments: %w", err)
			}

			// Add a comment if any principal (PR author or commenter) were replaced by the fallback migrator principal
			if prUnknownEmails, ok := repoState.unknownEmails[int(pullReq.Number)]; ok && len(prUnknownEmails) != 0 {
				infoComment, err := repoState.createInfoComment(ctx, repo, pullReq)
				if err != nil {
					log.Ctx(ctx).Warn().Err(err).Msg("failed to add an informational comment for replacing non-existing users")
				} else {
					comments = append(comments, infoComment)
				}
			}

			prLabels := pullReqUnique[int(pullReq.Number)].PullRequest.Labels
			err = repoState.assignLabels(ctx, repo.ParentID, pullReq, prLabels)
			if err != nil {
				return fmt.Errorf("failed to assign pull request %d labels: %w", pullReq.Number, err)
			}

			// no need to update the pull request object in the DB if there are no comments.
			if len(comments) == 0 && len(prLabels) == 0 {
				continue
			}

			if err := migrate.pullReqStore.Update(ctx, pullReq); err != nil {
				return fmt.Errorf("failed to update pull request after importing of the comments: %w", err)
			}
		}

		// Update the repository

		repoUpdate, err := migrate.repoStore.Find(ctx, repo.ID)
		if err != nil {
			return fmt.Errorf("failed to fetch repo in pull request import: %w", err)
		}

		if repoUpdate.PullReqSeq < maxNumber {
			repoUpdate.PullReqSeq = maxNumber
		}
		repoUpdate.NumPulls += len(pullReqs)
		repoUpdate.NumOpenPulls += deltaOpen
		repoUpdate.NumClosedPulls += deltaClosed
		repoUpdate.NumMergedPulls += deltaMerged

		if err := migrate.repoStore.Update(ctx, repoUpdate); err != nil {
			return fmt.Errorf("failed to update repo in pull request import: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	migrate.repoFinder.MarkChanged(ctx, repo)

	return pullReqs, nil
}

// convertPullReq analyses external pull request object and creates types.PullReq object out of it.
func (r *repoImportState) convertPullReq(
	ctx context.Context,
	repo *types.RepositoryCore,
	extPullReqData *ExternalPullRequest,
) (*types.PullReq, error) {
	extPullReq := extPullReqData.PullRequest

	log := log.Ctx(ctx).With().
		Str("repo.identifier", repo.Identifier).
		Int("pullreq.number", extPullReq.Number).
		Logger()

	author, err := r.getPrincipalByEmail(ctx, extPullReq.Author.Email, extPullReq.Number, false)
	if err != nil {
		return nil, fmt.Errorf("failed to get pull request author: %w", err)
	}

	now := time.Now().UnixMilli()
	createdAt := timestampMillis(extPullReq.Created, now)
	updatedAt := timestampMillis(extPullReq.Updated, now)

	const maxTitleLen = 256
	const maxDescriptionLen = 100000 // This limit is deliberately higher than the limit in our API.
	if len(extPullReq.Title) > maxTitleLen {
		extPullReq.Title = extPullReq.Title[:maxTitleLen]
	}
	if len(extPullReq.Body) > maxDescriptionLen {
		extPullReq.Body = extPullReq.Body[:maxDescriptionLen]
	}

	pr := &types.PullReq{
		ID:              0, // the ID will be populated in the data layer
		Version:         0,
		Number:          int64(extPullReq.Number),
		CreatedBy:       author.ID,
		Created:         createdAt,
		Updated:         updatedAt,
		Edited:          updatedAt,
		Closed:          nil,
		State:           enum.PullReqStateOpen,
		IsDraft:         extPullReq.Draft,
		CommentCount:    0,
		UnresolvedCount: 0,
		Title:           extPullReq.Title,
		Description:     extPullReq.Body,
		SourceRepoID:    repo.ID,
		SourceBranch:    extPullReq.Head.Name,
		SourceSHA:       extPullReq.Head.SHA,
		TargetRepoID:    repo.ID,
		TargetBranch:    extPullReq.Base.Name,
		ActivitySeq:     0,
		// Merge related fields are all left unset and will be set depending on the PR state
	}

	params := git.ReadParams{RepoUID: repo.GitUID}

	// Set the state of the PR
	switch {
	case extPullReq.Merged:
		pr.State = enum.PullReqStateMerged
	case extPullReq.Closed:
		pr.State = enum.PullReqStateClosed
	default:
		pr.State = enum.PullReqStateOpen
	}

	// Update the PR depending on its state
	switch pr.State {
	case enum.PullReqStateMerged:
		// For merged PR's assume the Head.Sha and Base.Sha point to commits at the time of merging.

		pr.Merged = &pr.Updated
		pr.MergedBy = &author.ID             // Don't have real info for this - use the author.
		mergeMethod := enum.MergeMethodMerge // Don't know
		pr.MergeMethod = &mergeMethod

		pr.SourceSHA = extPullReq.Head.SHA
		pr.MergeTargetSHA = &extPullReq.Base.SHA // TODO: Check why target == base. Can it be nil?
		pr.MergeBaseSHA = extPullReq.Base.SHA
		pr.MergeSHA = nil // Don't have this.
		pr.MarkAsMerged()

	case enum.PullReqStateClosed:
		// For closed PR's it's not important to verify existence of branches and commits.
		// If these don't exist the PR will be impossible to open.
		pr.SourceSHA = extPullReq.Head.SHA
		pr.MergeTargetSHA = nil
		pr.MergeBaseSHA = extPullReq.Base.SHA
		pr.MergeSHA = nil
		pr.MergeConflicts = nil
		pr.MarkAsMergeUnchecked()

		pr.Closed = &pr.Updated

	case enum.PullReqStateOpen:
		// For open PR we need to verify existence of branches and find to merge base.

		sourceBranch, err := r.git.GetBranch(ctx, &git.GetBranchParams{
			ReadParams: params,
			BranchName: extPullReq.Head.Name,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to fetch source branch of an open pull request: %w", err)
		}

		// TODO: Cache this in the repoImportState - it's very likely that it will be the same for other PRs
		targetBranch, err := r.git.GetBranch(ctx, &git.GetBranchParams{
			ReadParams: params,
			BranchName: extPullReq.Base.Name,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to fetch target branch of an open pull request: %w", err)
		}

		mergeBase, err := r.git.MergeBase(ctx, git.MergeBaseParams{
			ReadParams: params,
			Ref1:       sourceBranch.Branch.SHA.String(),
			Ref2:       targetBranch.Branch.SHA.String(),
		})
		if err != nil {
			return nil, fmt.Errorf("failed to find merge base an open pull request: %w", err)
		}

		sourceSHA := sourceBranch.Branch.SHA.String()
		targetSHA := targetBranch.Branch.SHA.String()
		pr.SourceSHA = sourceSHA
		pr.MergeTargetSHA = &targetSHA
		pr.MergeBaseSHA = mergeBase.MergeBaseSHA.String()
		pr.MarkAsMergeUnchecked()
	}

	log.Debug().Str("pullreq.state", string(pr.State)).Msg("importing pull request")

	return pr, nil
}

// createComments analyses external pull request comment objects and stores types.PullReqActivity object to the DB.
// It will mutate the pull request object to update counter fields.
func (r *repoImportState) createComments(
	ctx context.Context,
	repo *types.RepositoryCore,
	pullReq *types.PullReq,
	extComments []ExternalComment,
) ([]*types.PullReqActivity, error) {
	log := log.Ctx(ctx).With().
		Str("repo.id", repo.Identifier).
		Int("pullreq.number", int(pullReq.Number)).
		Logger()

	extThreads := generateThreads(extComments)

	comments := make([]*types.PullReqActivity, 0, len(extComments))
	for idxTopLevel, extThread := range extThreads {
		order := int(pullReq.ActivitySeq) + idxTopLevel + 1

		// Create the top level comment with the correct value of Order, SubOrder and ReplySeq.
		commentTopLevel, err := r.createComment(ctx, repo, pullReq, nil,
			order, 0, len(extThread.Replies), &extThread.TopLevel)
		if err != nil {
			return nil, fmt.Errorf("failed to create top level comment: %w", err)
		}

		comments = append(comments, commentTopLevel)

		for idxReply, extReply := range extThread.Replies {
			subOrder := idxReply + 1

			// Create the reply comment with the correct value of Order, SubOrder and ReplySeq.
			//nolint:gosec
			commentReply, err := r.createComment(ctx, repo, pullReq, &commentTopLevel.ID,
				order, subOrder, 0, &extReply)
			if err != nil {
				return nil, fmt.Errorf("failed to create reply comment: %w", err)
			}

			comments = append(comments, commentReply)
		}
	}

	log.Debug().Int("count", len(comments)).Msg("imported pull request comments")

	return comments, nil
}

// createComment analyses an external pull request comment object and creates types.PullReqActivity object out of it.
// It will mutate the pull request object to update counter fields.
func (r *repoImportState) createComment(
	ctx context.Context,
	repo *types.RepositoryCore,
	pullReq *types.PullReq,
	parentID *int64,
	order, subOrder, replySeq int,
	extComment *ExternalComment,
) (*types.PullReqActivity, error) {
	commenter, err := r.getPrincipalByEmail(ctx, extComment.Author.Email, int(pullReq.Number), false)
	if err != nil {
		return nil, fmt.Errorf("failed to get comment ID=%d author: %w", extComment.ID, err)
	}
	commentedAt := extComment.Created.UnixMilli()

	// Mark comments as resolved if the PR is merged, otherwise they are unresolved.
	var resolved, resolvedBy *int64
	if pullReq.State == enum.PullReqStateMerged {
		resolved = &commentedAt
		resolvedBy = &commenter.ID
	}

	const maxLenText = 64 << 10 // This limit is deliberately larger than the limit in our API.
	if len(extComment.Body) > maxLenText {
		extComment.Body = extComment.Body[:maxLenText]
	}

	comment := &types.PullReqActivity{
		CreatedBy:   commenter.ID,
		Created:     commentedAt,
		Updated:     commentedAt,
		Edited:      commentedAt,
		Deleted:     nil,
		ParentID:    parentID,
		RepoID:      repo.ID,
		PullReqID:   pullReq.ID,
		Order:       int64(order),
		SubOrder:    int64(subOrder),
		ReplySeq:    int64(replySeq),
		Type:        enum.PullReqActivityTypeComment,
		Kind:        enum.PullReqActivityKindComment,
		Text:        extComment.Body,
		PayloadRaw:  json.RawMessage("{}"),
		Metadata:    nil,
		ResolvedBy:  resolvedBy,
		Resolved:    resolved,
		CodeComment: nil,
		Mentions:    nil,
	}

	if cc := extComment.CodeComment; cc != nil && cc.HunkHeader != "" && extComment.ParentID == 0 {
		// a code comment must have a valid HunkHeader and must not be a reply
		hunkHeader, ok := parser.ParseDiffHunkHeader(cc.HunkHeader)
		if !ok {
			return nil, errors.InvalidArgument("Invalid hunk header for code comment: %s", cc.HunkHeader)
		}

		comment.Kind = enum.PullReqActivityKindChangeComment
		comment.Type = enum.PullReqActivityTypeCodeComment

		comment.CodeComment = &types.CodeCommentFields{
			Outdated:     cc.SourceSHA != pullReq.SourceSHA,
			MergeBaseSHA: cc.MergeBaseSHA,
			SourceSHA:    cc.SourceSHA,
			Path:         cc.Path,
			LineNew:      hunkHeader.NewLine,
			SpanNew:      hunkHeader.NewSpan,
			LineOld:      hunkHeader.OldLine,
			SpanOld:      hunkHeader.OldSpan,
		}

		sideNew := !strings.EqualFold(cc.Side, "OLD") // cc.Side can be either OLD or NEW
		_ = comment.SetPayload(&types.PullRequestActivityPayloadCodeComment{
			Title:        cc.CodeSnippet.Header,
			Lines:        cc.CodeSnippet.Lines,
			LineStartNew: sideNew,
			LineEndNew:   sideNew,
		})
	}

	// store the comment

	if err := r.pullReqActivityStore.Create(ctx, comment); err != nil {
		return nil, fmt.Errorf("failed to store the external comment ID=%d author: %w", extComment.ID, err)
	}

	// update the pull request's counter fields

	pullReq.CommentCount++
	if comment.IsBlocking() {
		pullReq.UnresolvedCount++
	}

	if pullReq.ActivitySeq < comment.Order {
		pullReq.ActivitySeq = comment.Order
	}

	return comment, nil
}

// createInfoComment creates an informational comment on the PR
// if any of the principals were replaced with the migrator.
func (r *repoImportState) createInfoComment(
	ctx context.Context,
	repo *types.RepositoryCore,
	pullReq *types.PullReq,
) (*types.PullReqActivity, error) {
	var unknownEmails []string
	for email := range r.unknownEmails[int(pullReq.Number)] {
		unknownEmails = append(unknownEmails, email)
	}
	now := time.Now().UnixMilli()
	text := fmt.Sprintf(InfoCommentMessage, r.migrator.UID, strings.Join(unknownEmails, ", "))
	comment := &types.PullReqActivity{
		CreatedBy:   r.migrator.ID,
		Created:     now,
		Updated:     now,
		Deleted:     nil,
		ParentID:    nil,
		RepoID:      repo.ID,
		PullReqID:   pullReq.ID,
		Order:       pullReq.ActivitySeq + 1,
		SubOrder:    0,
		ReplySeq:    0,
		Type:        enum.PullReqActivityTypeComment,
		Kind:        enum.PullReqActivityKindComment,
		Text:        text,
		PayloadRaw:  json.RawMessage("{}"),
		Metadata:    nil,
		ResolvedBy:  &r.migrator.ID,
		Resolved:    &now,
		CodeComment: nil,
		Mentions:    nil,
	}

	if err := r.pullReqActivityStore.Create(ctx, comment); err != nil {
		return nil, fmt.Errorf("failed to store the info comment author: %w", err)
	}

	pullReq.ActivitySeq++
	pullReq.CommentCount++

	return comment, nil
}

//nolint:unparam
func (r *repoImportState) getPrincipalByEmail(
	ctx context.Context,
	emailAddress string,
	prNumber int,
	strict bool,
) (*types.Principal, error) {
	if principal, exists := r.principals[emailAddress]; exists {
		return principal, nil
	}

	principal, err := r.principalStore.FindByEmail(ctx, emailAddress)
	if err != nil && !errors.Is(err, gitness_store.ErrResourceNotFound) {
		return nil, fmt.Errorf("failed to load principal by email: %w", err)
	}

	if err == nil {
		r.principals[emailAddress] = principal
		return principal, nil
	}

	if strict {
		return nil, fmt.Errorf(
			"could not find principal by email %s and automatic replacing unknown prinicapls is disabled: %w",
			emailAddress, err)
	}

	// ignore not found emails if is not strict
	if _, exists := r.unknownEmails[prNumber]; !exists {
		r.unknownEmails[prNumber] = make(map[string]bool, 0)
	}

	if _, ok := r.unknownEmails[prNumber][emailAddress]; !ok && len(r.unknownEmails[prNumber]) < MaxNumberOfUnknownEmails {
		r.unknownEmails[prNumber][emailAddress] = true
	}

	return &r.migrator, nil
}

func (r *repoImportState) assignLabels(
	ctx context.Context,
	spaceID int64,
	pullreq *types.PullReq,
	labels []ExternalLabel,
) error {
	if len(labels) == 0 {
		return nil
	}
	now := time.Now().UnixMilli()
	for _, l := range labels {
		var label *types.Label
		var err error
		labelID, found := r.labels[l.Name]
		if !found {
			label, err = r.labelStore.Find(ctx, &spaceID, nil, l.Name)
			if errors.Is(err, gitness_store.ErrResourceNotFound) {
				label, err = r.defineLabel(ctx, spaceID, l)
				if err != nil {
					return fmt.Errorf("failed to define label: %w", err)
				}
			} else if err != nil {
				return fmt.Errorf("failed to find the label with key %s in space %d: %w", l.Name, spaceID, err)
			}

			r.labels[l.Name], labelID = label.ID, label.ID
		}

		var valueID *int64
		valueID, found = r.labelValues[labelID][l.Value]
		if !found && l.Value != "" {
			var labelValue *types.LabelValue
			labelValue, err = r.labelValueStore.FindByLabelID(ctx, labelID, l.Value)
			if errors.Is(err, gitness_store.ErrResourceNotFound) {
				labelValue, err = r.defineLabelValue(ctx, labelID, l.Value)
				if err != nil {
					return fmt.Errorf("failed to define label values: %w", err)
				}
			} else if err != nil {
				return fmt.Errorf("failed to find the label with value %s and key %s in space %d: %w",
					l.Value, l.Name, spaceID, err)
			}

			valueID = &labelValue.ID
		}

		pullReqLabel := &types.PullReqLabel{
			PullReqID: pullreq.ID,
			LabelID:   labelID,
			ValueID:   valueID,
			Created:   now,
			Updated:   now,
			CreatedBy: r.migrator.ID,
			UpdatedBy: r.migrator.ID,
		}

		err = r.pullReqLabelAssignmentStore.Assign(ctx, pullReqLabel)
		if err != nil {
			return fmt.Errorf("failed to assign label %s to pull request: %w", l.Name, err)
		}
		pullreq.ActivitySeq++
	}

	return nil
}

func (r *repoImportState) defineLabel(
	ctx context.Context,
	spaceID int64,
	extLabel ExternalLabel,
) (*types.Label, error) {
	if r.scope == 0 {
		spaceIDs, err := r.spaceStore.GetAncestorIDs(ctx, spaceID)
		if err != nil {
			return nil, fmt.Errorf("failed to get space ids hierarchy: %w", err)
		}

		r.scope = int64(len(spaceIDs))
	}

	labelIn, err := convertLabelWithSanitization(ctx, r.migrator, spaceID, r.scope, extLabel)
	if err != nil {
		return nil, fmt.Errorf("failed to sanitize and convert external label input: %w", err)
	}

	label, err := r.labelStore.Find(ctx, &spaceID, nil, labelIn.Key)
	if errors.Is(err, gitness_store.ErrResourceNotFound) {
		err = r.labelStore.Define(ctx, labelIn)
		if err != nil {
			return nil, fmt.Errorf("failed to define and find the label: %w", err)
		}
		return labelIn, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to define and find the label: %w", err)
	}

	return label, nil
}

func (r *repoImportState) defineLabelValue(
	ctx context.Context,
	labelID int64,
	value string,
) (*types.LabelValue, error) {
	valueIn := &types.DefineValueInput{
		Value: value,
		Color: defaultLabelValueColor,
	}
	if err := valueIn.Sanitize(); err != nil {
		return nil, fmt.Errorf("failed to sanitize external label value input: %w", err)
	}

	if _, exists := r.labelValues[labelID]; !exists {
		r.labelValues[labelID] = make(map[string]*int64)
	}

	labelValue, err := r.labelValueStore.FindByLabelID(ctx, labelID, valueIn.Value)
	if err == nil {
		r.labelValues[labelID][labelValue.Value] = &labelValue.ID
		return labelValue, nil
	}

	if !errors.Is(err, gitness_store.ErrResourceNotFound) {
		return nil, fmt.Errorf("failed to fine label value: %w", err)
	}

	// define the label value if not exists
	now := time.Now().UnixMilli()
	labelValue = &types.LabelValue{
		LabelID:   labelID,
		Value:     valueIn.Value,
		Color:     defaultLabelValueColor,
		Created:   now,
		Updated:   now,
		CreatedBy: r.migrator.ID,
		UpdatedBy: r.migrator.ID,
	}
	err = r.labelValueStore.Define(ctx, labelValue)
	if err != nil {
		return nil, fmt.Errorf("failed to define label value: %w", err)
	}
	_, err = r.labelStore.IncrementValueCount(ctx, labelID, 1)
	if err != nil {
		return nil, fmt.Errorf("failed to update label value count: %w", err)
	}

	r.labelValues[labelID][labelValue.Value] = &labelValue.ID
	return labelValue, nil
}

func timestampMillis(t time.Time, def int64) int64 {
	if t.IsZero() {
		return def
	}

	return t.UnixMilli()
}

func generateThreads(extComments []ExternalComment) []*externalCommentThread {
	extCommentParents := make(map[int]int, len(extComments))
	extCommentMap := make(map[int]ExternalComment, len(extComments))
	for _, extComment := range extComments {
		extCommentParents[extComment.ID] = extComment.ParentID
		extCommentMap[extComment.ID] = extComment
	}

	// Make flat list of reply comment IDs: create map[topLevelCommentID]->[]commentID
	extCommentIDReplyMap := make(map[int][]int)
	for _, extComment := range extComments {
		topLevelParentID := getTopLevelParentID(extComment.ID, extCommentParents)
		if topLevelParentID < 0 {
			continue
		}
		if topLevelParentID == extComment.ID {
			// Make sure the item with topLevelParentID exist in the map, at least as a nil entry.
			extCommentIDReplyMap[topLevelParentID] = extCommentIDReplyMap[topLevelParentID] //nolint:staticcheck
			continue
		}
		extCommentIDReplyMap[topLevelParentID] = append(extCommentIDReplyMap[topLevelParentID], extComment.ID)
	}

	countTopLevel := len(extCommentIDReplyMap)
	if countTopLevel == 0 {
		return nil
	}

	extCommentThreads := make([]*externalCommentThread, 0, countTopLevel)
	for topLevelID, replyIDs := range extCommentIDReplyMap {
		expReplyComments := make([]ExternalComment, len(replyIDs))
		for i, replyID := range replyIDs {
			expReplyComments[i] = extCommentMap[replyID]
		}
		thread := &externalCommentThread{
			TopLevel: extCommentMap[topLevelID],
			Replies:  expReplyComments,
		}
		extCommentThreads = append(extCommentThreads, thread)
	}

	// order top level comments

	sort.Slice(extCommentThreads, func(i, j int) bool {
		created1 := extCommentThreads[i].TopLevel.Created
		created2 := extCommentThreads[j].TopLevel.Created
		return created1.Before(created2)
	})

	// order reply comments

	for _, thread := range extCommentThreads {
		sort.Slice(thread.Replies, func(i, j int) bool {
			created1 := thread.Replies[i].Created
			created2 := thread.Replies[j].Created
			return created1.Before(created2)
		})
	}

	return extCommentThreads
}

func getTopLevelParentID(id int, tree map[int]int) int {
	const maxDepth = 20
	for currID, depth := id, 0; depth < maxDepth; depth++ {
		parentID := tree[currID]
		if parentID == 0 {
			return currID
		}

		currID = parentID
	}

	return -1
}

// createReviewers processes external reviewer objects.
func (r *repoImportState) createReviewers(
	ctx context.Context,
	repo *types.RepositoryCore,
	pullReq *types.PullReq,
	extReviewers []ExternalReviewer,
) ([]*types.PullReqReviewer, error) {
	log := log.Ctx(ctx).With().
		Str("repo.id", repo.Identifier).
		Int("pullreq.number", int(pullReq.Number)).
		Logger()

	reviewers := make([]*types.PullReqReviewer, 0, len(extReviewers))
	for _, extReviewer := range extReviewers {
		reviewer, err := r.getPrincipalByEmail(ctx, extReviewer.User.Email, int(pullReq.Number), false)
		if err != nil {
			return nil, fmt.Errorf("failed to get reviewer principal: %w", err)
		}

		if reviewer.ID == pullReq.CreatedBy {
			continue
		}

		// Use PR created timestamp for reviewer assignment
		assignedAt := pullReq.Created

		prReviewer := &types.PullReqReviewer{
			PullReqID:      pullReq.ID,
			PrincipalID:    reviewer.ID,
			CreatedBy:      r.migrator.ID,
			Created:        assignedAt,
			Updated:        assignedAt,
			RepoID:         repo.ID,
			Type:           enum.PullReqReviewerTypeRequested,
			LatestReviewID: nil,                               // Will be set when reviews are processed
			ReviewDecision: enum.PullReqReviewDecisionPending, // Will be updated when reviews are processed
			SHA:            pullReq.SourceSHA,
			Reviewer:       *reviewer.ToPrincipalInfo(),
			AddedBy:        *r.migrator.ToPrincipalInfo(),
		}

		if err := r.pullReqReviewerStore.Create(ctx, prReviewer); err != nil {
			return nil, fmt.Errorf("failed to store pull request reviewer: %w", err)
		}

		reviewers = append(reviewers, prReviewer)
	}

	log.Debug().Int("count", len(reviewers)).Msg("imported pull request reviewers")

	if len(reviewers) > 0 {
		reviewerIDs := make([]int64, 0, len(reviewers))
		for _, reviewer := range reviewers {
			reviewerIDs = append(reviewerIDs, reviewer.PrincipalID)
		}
		r.createReviewerActivity(ctx, pullReq, reviewerIDs, enum.PullReqReviewerTypeRequested)
	}

	return reviewers, nil
}

// createReviews processes external review objects.
func (r *repoImportState) createReviews(
	ctx context.Context,
	repo *types.RepositoryCore,
	pullReq *types.PullReq,
	extReviews []ExternalReview,
) ([]*types.PullReqReview, error) {
	log := log.Ctx(ctx).With().
		Str("repo.id", repo.Identifier).
		Int("pullreq.number", int(pullReq.Number)).
		Logger()

	reviews := make([]*types.PullReqReview, 0, len(extReviews))
	for _, extReview := range extReviews {
		reviewer, err := r.getPrincipalByEmail(ctx, extReview.Author.Email, int(pullReq.Number), false)
		if err != nil {
			return nil, fmt.Errorf("failed to get reviewer principal: %w", err)
		}

		if reviewer.ID == pullReq.CreatedBy {
			continue
		}

		decision := enum.PullReqReviewDecision(extReview.Decision)

		submittedAt := time.Now().UnixMilli()
		if !extReview.Updated.IsZero() {
			submittedAt = extReview.Updated.UnixMilli()
		} else if !extReview.Created.IsZero() {
			submittedAt = extReview.Created.UnixMilli()
		}

		prReview := &types.PullReqReview{
			CreatedBy: reviewer.ID,
			Created:   submittedAt,
			Updated:   submittedAt,
			PullReqID: pullReq.ID,
			Decision:  decision,
			SHA:       extReview.SHA,
		}

		if err := r.pullReqReviewStore.Create(ctx, prReview); err != nil {
			return nil, fmt.Errorf("failed to store pull request review: %w", err)
		}

		existingReviewer, err := r.pullReqReviewerStore.Find(ctx, pullReq.ID, reviewer.ID)
		if err != nil && !errors.Is(err, gitness_store.ErrResourceNotFound) {
			return nil, fmt.Errorf("failed to find reviewer from review author: %w", err)
		}
		if errors.Is(err, gitness_store.ErrResourceNotFound) {
			reviewerFromReview := &types.PullReqReviewer{
				PullReqID:      pullReq.ID,
				PrincipalID:    reviewer.ID,
				CreatedBy:      r.migrator.ID,
				Created:        submittedAt, // Use review submission time
				Updated:        submittedAt,
				RepoID:         repo.ID,
				Type:           enum.PullReqReviewerTypeSelfAssigned,
				LatestReviewID: &prReview.ID,
				ReviewDecision: decision,
				SHA:            pullReq.SourceSHA,
				Reviewer:       *reviewer.ToPrincipalInfo(),
				AddedBy:        *reviewer.ToPrincipalInfo(),
			}

			if err := r.pullReqReviewerStore.Create(ctx, reviewerFromReview); err != nil {
				return nil, fmt.Errorf("failed to create reviewer from review author: %w", err)
			}
		}

		if existingReviewer != nil {
			// Update existing reviewer with latest review
			existingReviewer.LatestReviewID = &prReview.ID
			existingReviewer.ReviewDecision = decision
			existingReviewer.Updated = submittedAt
			if err := r.pullReqReviewerStore.Update(ctx, existingReviewer); err != nil {
				log.Warn().Err(err).Msg("failed to update reviewer with latest review")
			}
		}

		reviews = append(reviews, prReview)
	}

	log.Debug().Int("count", len(reviews)).Msg("imported pull request reviews")

	// Create activity entries for review submissions
	for _, review := range reviews {
		r.createReviewSubmitActivity(ctx, pullReq, review)
	}

	return reviews, nil
}

// createReviewerActivity creates an activity entry for reviewer addition.
func (r *repoImportState) createReviewerActivity(
	ctx context.Context,
	pullReq *types.PullReq,
	reviewerIDs []int64,
	reviewerType enum.PullReqReviewerType,
) {
	if len(reviewerIDs) == 0 {
		return
	}

	// Increment ActivitySeq in database and update pullReq object
	updatedPR, err := r.pullReqStore.UpdateActivitySeq(ctx, pullReq)
	if err != nil {
		log.Ctx(ctx).Err(err).Msg("failed to increment pull request activity sequence for reviewer activity")
		return
	}
	*pullReq = *updatedPR // Update the in-memory object

	payload := &types.PullRequestActivityPayloadReviewerAdd{
		ReviewerType: reviewerType,
		PrincipalIDs: reviewerIDs,
	}

	metadata := &types.PullReqActivityMetadata{
		Mentions: &types.PullReqActivityMentionsMetadata{IDs: reviewerIDs},
	}

	if _, err := r.pullReqActivityStore.CreateWithPayload(
		ctx, pullReq, r.migrator.ID, payload, metadata,
	); err != nil {
		log.Ctx(ctx).Err(err).Msgf(
			"failed to write create %s reviewer pull req activity", reviewerType,
		)
	}
}

// createReviewSubmitActivity creates an activity entry for review submission.
func (r *repoImportState) createReviewSubmitActivity(
	ctx context.Context,
	pullReq *types.PullReq,
	review *types.PullReqReview,
) {
	updatedPR, err := r.pullReqStore.UpdateActivitySeq(ctx, pullReq)
	if err != nil {
		log.Ctx(ctx).Err(err).Msg("failed to increment pull request activity sequence for review activity")
		return
	}
	*pullReq = *updatedPR

	payload := &types.PullRequestActivityPayloadReviewSubmit{
		CommitSHA: review.SHA,
		Decision:  review.Decision,
	}

	if _, err := r.pullReqActivityStore.CreateWithPayload(
		ctx, pullReq, review.CreatedBy, payload, nil,
	); err != nil {
		log.Ctx(ctx).Err(err).Msgf(
			"failed to write review submit pull req activity for review ID %d", review.ID,
		)
	}
}
