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

// Package github hosts the linked-PR handlers that consume GitHub webhook
// events. Handlers are webhook-only — no GitHub API calls per event.
package github

import (
	"context"
	"errors"
	"fmt"
	"time"

	pullreqevents "github.com/harness/gitness/app/events/pullreq"
	"github.com/harness/gitness/app/services/importer"
	"github.com/harness/gitness/app/services/linkedpr"
	"github.com/harness/gitness/app/services/linkedpr/handlers"
	"github.com/harness/gitness/app/services/refcache"
	"github.com/harness/gitness/app/store"
	gitnessurl "github.com/harness/gitness/app/url"
	"github.com/harness/gitness/git"
	gitness_store "github.com/harness/gitness/store"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/gotidy/ptr"
	"github.com/rs/zerolog/log"
)

// PullRequestHandler mirrors GitHub PR state from the webhook payload into
// pullreqs + linked_pullreqs and publishes the matching Reporter event.
type PullRequestHandler struct {
	pullReqStore       store.PullReqStore
	linkedPullReqStore store.LinkedPullReqStore
	activityStore      store.PullReqActivityStore
	authorResolver     linkedpr.AuthorResolver
	reporter           *pullreqevents.Reporter
	gitClient          git.Interface
	repoFinder         refcache.RepoFinder
	urlProvider        gitnessurl.Provider
	connectorService   importer.ConnectorService
	tx                 dbtx.Transactor
}

func NewPullRequestHandler(
	pullReqStore store.PullReqStore,
	linkedPullReqStore store.LinkedPullReqStore,
	activityStore store.PullReqActivityStore,
	authorResolver linkedpr.AuthorResolver,
	reporter *pullreqevents.Reporter,
	gitClient git.Interface,
	repoFinder refcache.RepoFinder,
	urlProvider gitnessurl.Provider,
	connectorService importer.ConnectorService,
	tx dbtx.Transactor,
) *PullRequestHandler {
	return &PullRequestHandler{
		pullReqStore:       pullReqStore,
		linkedPullReqStore: linkedPullReqStore,
		activityStore:      activityStore,
		authorResolver:     authorResolver,
		reporter:           reporter,
		gitClient:          gitClient,
		repoFinder:         repoFinder,
		urlProvider:        urlProvider,
		connectorService:   connectorService,
		tx:                 tx,
	}
}

func (h *PullRequestHandler) Handle(
	ctx context.Context,
	ev *linkedpr.Event,
	prPayload linkedpr.PullRequestPayload,
	linkedRepo *types.LinkedRepo,
) error {
	// Drop malformed payloads — otherwise refsToFetch would emit "refs/heads/"
	// and trigger a poison-pill retry loop on the consumer.
	if prPayload.HeadRef == "" || prPayload.HeadSHA == "" ||
		prPayload.BaseRef == "" || prPayload.BaseSHA == "" {
		log.Ctx(ctx).Warn().
			Str("provider_id", linkedRepo.ProviderRepoID).
			Int("provider_pr_number", prPayload.Number).
			Str("head_ref", prPayload.HeadRef).Str("head_sha", prPayload.HeadSHA).
			Str("base_ref", prPayload.BaseRef).Str("base_sha", prPayload.BaseSHA).
			Msg("linkedpr: payload missing required ref/sha fields; ack-skipping")
		return nil
	}

	existing, err := h.linkedPullReqStore.FindByLinkedRepoAndProviderPR(
		ctx, linkedRepo.RepoID, string(ev.Provider), linkedRepo.ProviderRepoID, prPayload.Number)
	if err != nil && !errors.Is(err, gitness_store.ErrResourceNotFound) {
		return fmt.Errorf("find existing linked PR: %w", err)
	}

	if existing == nil {
		if err := handlers.RunSyncRefs(
			ctx, h.gitClient, h.repoFinder, h.urlProvider, h.connectorService, linkedRepo, refsToFetch(prPayload),
		); err != nil {
			return fmt.Errorf("sync refs: %w", err)
		}
		mergeBaseSHA, err := handlers.RunMergeBase(
			ctx, h.gitClient, h.repoFinder, linkedRepo, prPayload.HeadSHA, prPayload.BaseSHA,
		)
		if err != nil {
			return fmt.Errorf("compute merge base: %w", err)
		}
		// AuthorResolver only on create: pullreq_created_by is immutable.
		authorPrincipalID, err := h.authorResolver.Resolve(ctx, ev)
		if err != nil {
			return fmt.Errorf("resolve author: %w", err)
		}
		return h.create(ctx, linkedRepo, ev, prPayload, authorPrincipalID, mergeBaseSHA)
	}

	parent, err := h.pullReqStore.Find(ctx, existing.PullReqID)
	if err != nil {
		return fmt.Errorf("find parent pullreq: %w", err)
	}

	// Out-of-order guard: drop deliveries with a provider-clock UpdatedAt
	// older than or equal to the last we applied. <= also dedups identical
	// webhook retries that GitHub re-sends on transient errors
	if prPayload.UpdatedAt > 0 && existing.ProviderUpdatedAt > 0 &&
		prPayload.UpdatedAt <= existing.ProviderUpdatedAt {
		log.Ctx(ctx).Debug().
			Int64("payload_updated_at", prPayload.UpdatedAt).
			Int64("stored_provider_updated_at", existing.ProviderUpdatedAt).
			Str("provider_id", linkedRepo.ProviderRepoID).
			Int("provider_pr_number", prPayload.Number).
			Msg("linkedpr: stale webhook (older than last applied event); dropping")
		return nil
	}

	// Re-sync refs + recompute merge base only when SHAs have moved;
	// pure-metadata events reuse the stored merge base.
	headChanged := prPayload.HeadSHA != parent.SourceSHA
	baseChanged := prPayload.BaseSHA != ptr.ToString(parent.MergeTargetSHA) ||
		prPayload.BaseRef != parent.TargetBranch
	shaChanged := headChanged || baseChanged

	mergeBaseSHA := parent.MergeBaseSHA
	if shaChanged {
		if err := handlers.RunSyncRefs(
			ctx, h.gitClient, h.repoFinder, h.urlProvider, h.connectorService, linkedRepo, refsToFetch(prPayload),
		); err != nil {
			return fmt.Errorf("sync refs: %w", err)
		}
		mergeBaseSHA, err = handlers.RunMergeBase(
			ctx, h.gitClient, h.repoFinder, linkedRepo, prPayload.HeadSHA, prPayload.BaseSHA,
		)
		if err != nil {
			return fmt.Errorf("compute merge base: %w", err)
		}
	}
	return h.update(ctx, prPayload, existing, parent, mergeBaseSHA, shaChanged)
}

func refsToFetch(p linkedpr.PullRequestPayload) []string {
	return []string{
		fmt.Sprintf("refs/pull/%d/head", p.Number),
		fmt.Sprintf("refs/heads/%s", p.HeadRef),
		fmt.Sprintf("refs/heads/%s", p.BaseRef),
	}
}

// Sentinels distinguish the two create() duplicate cases: a native PR
// already owning (repo_id, number) is dropped; a racing linked-PR insert
// falls through to update.
var (
	errParentDuplicate = errors.New("parent pullreq number conflict")
	errLinkedDuplicate = errors.New("linked pullreq provider_id conflict")
)

func (h *PullRequestHandler) create(
	ctx context.Context,
	linkedRepo *types.LinkedRepo,
	ev *linkedpr.Event,
	prPayload linkedpr.PullRequestPayload,
	authorPrincipalID int64,
	mergeBaseSHA string,
) error {
	now := time.Now().UnixMilli()
	// Linked PRs mirror both head and base into the same local repo, so
	// SourceRepoID intentionally equals TargetRepoID.
	srcRepoID := linkedRepo.RepoID

	prType := enum.PullReqTypeLinked
	parent := &types.PullReq{
		Number:           int64(prPayload.Number),
		CreatedBy:        authorPrincipalID,
		Created:          firstNonZero(prPayload.CreatedAt, now),
		Updated:          firstNonZero(prPayload.UpdatedAt, now),
		Edited:           firstNonZero(prPayload.UpdatedAt, now),
		State:            prPayload.State,
		IsDraft:          prPayload.Draft,
		Title:            prPayload.Title,
		Description:      prPayload.Description,
		SourceRepoID:     &srcRepoID,
		SourceBranch:     prPayload.HeadRef,
		SourceSHA:        prPayload.HeadSHA,
		TargetRepoID:     linkedRepo.RepoID,
		TargetBranch:     prPayload.BaseRef,
		MergeTargetSHA:   ptr.String(prPayload.BaseSHA),
		MergeBaseSHA:     mergeBaseSHA,
		MergeCheckStatus: enum.MergeCheckStatusUnchecked,
		Type:             &prType,
	}
	if prPayload.State == enum.PullReqStateClosed || prPayload.State == enum.PullReqStateMerged {
		// UpdatedAt stands in for closed_at/merged_at — proto exposes neither.
		ts := firstNonZero(prPayload.UpdatedAt, now)
		parent.Closed = &ts
		if prPayload.State == enum.PullReqStateMerged {
			parent.Merged = &ts
		}
	}

	txErr := h.tx.WithTx(ctx, func(ctx context.Context) error {
		if err := h.pullReqStore.Create(ctx, parent); err != nil {
			if errors.Is(err, gitness_store.ErrDuplicate) {
				return errParentDuplicate
			}
			return fmt.Errorf("create parent pullreq: %w", err)
		}
		linked := &types.LinkedPullReq{
			PullReqID:               parent.ID,
			ProviderType:            string(ev.Provider),
			ProviderRepoID:          linkedRepo.ProviderRepoID,
			ProviderURL:             prPayload.HTMLURL,
			ProviderAuthorLogin:     prPayload.Author.Login,
			ProviderAuthorAvatarURL: prPayload.Author.Avatar,
			ProviderAuthorURL:       prPayload.Author.HTMLURL,
			ProviderUpdatedAt:       prPayload.UpdatedAt,
		}
		if prPayload.State == enum.PullReqStateMerged {
			linked.MergerLogin = prPayload.Sender.Login
		}
		if err := h.linkedPullReqStore.Create(ctx, linked); err != nil {
			if errors.Is(err, gitness_store.ErrDuplicate) {
				return errLinkedDuplicate
			}
			return fmt.Errorf("create linked pullreq: %w", err)
		}
		return nil
	})
	switch {
	case txErr == nil:
		h.publishCreated(ctx, parent)
		return nil
	case errors.Is(txErr, errLinkedDuplicate):
		return h.handleLinkedDuplicate(ctx, ev, prPayload, linkedRepo, mergeBaseSHA)
	case errors.Is(txErr, errParentDuplicate):
		log.Ctx(ctx).Warn().
			Int64("repo_id", linkedRepo.RepoID).
			Int("pr_number", prPayload.Number).
			Str("provider", string(ev.Provider)).
			Str("provider_id", linkedRepo.ProviderRepoID).
			Msg("linkedpr: pullreq number collision with native PR; dropping event")
		return nil
	default:
		return txErr
	}
}

// handleLinkedDuplicate re-loads the row a racing delivery just committed
// and falls through to update with shaChanged=true to force a merge re-check.
func (h *PullRequestHandler) handleLinkedDuplicate(
	ctx context.Context,
	ev *linkedpr.Event,
	prPayload linkedpr.PullRequestPayload,
	linkedRepo *types.LinkedRepo,
	mergeBaseSHA string,
) error {
	existing, err := h.linkedPullReqStore.FindByLinkedRepoAndProviderPR(
		ctx, linkedRepo.RepoID, string(ev.Provider), linkedRepo.ProviderRepoID, prPayload.Number)
	if err != nil {
		return fmt.Errorf("re-find after duplicate: %w", err)
	}
	// Race-recovery: drop ourselves if the winner already stored an equal or
	// fresher ProviderUpdatedAt.
	if prPayload.UpdatedAt > 0 && prPayload.UpdatedAt <= existing.ProviderUpdatedAt {
		log.Ctx(ctx).Debug().
			Int64("payload_updated_at", prPayload.UpdatedAt).
			Int64("stored_provider_updated_at", existing.ProviderUpdatedAt).
			Str("provider_id", linkedRepo.ProviderRepoID).
			Int("provider_pr_number", prPayload.Number).
			Msg("linkedpr: stale duplicate (race-recovery loser); dropping")
		return nil
	}
	parent, err := h.pullReqStore.Find(ctx, existing.PullReqID)
	if err != nil {
		return fmt.Errorf("find parent after duplicate: %w", err)
	}
	return h.update(ctx, prPayload, existing, parent, mergeBaseSHA, true)
}

func (h *PullRequestHandler) update(
	ctx context.Context,
	prPayload linkedpr.PullRequestPayload,
	existing *types.LinkedPullReq,
	parent *types.PullReq,
	mergeBaseSHA string,
	shaChanged bool,
) error {
	prevState := parent.State
	prevDraft := parent.IsDraft
	prevTitle := parent.Title
	prevSourceSHA := parent.SourceSHA
	prevMergeBaseSHA := parent.MergeBaseSHA
	now := time.Now().UnixMilli()

	parent.State = prPayload.State
	parent.IsDraft = prPayload.Draft
	parent.Title = prPayload.Title
	parent.Description = prPayload.Description
	parent.SourceBranch = prPayload.HeadRef
	parent.SourceSHA = prPayload.HeadSHA
	parent.TargetBranch = prPayload.BaseRef
	parent.MergeTargetSHA = ptr.String(prPayload.BaseSHA)
	parent.MergeBaseSHA = mergeBaseSHA
	if shaChanged {
		// Reset stats so BranchUpdated subscribers recompute (mirrors handlers_branch.go).
		parent.MergeCheckStatus = enum.MergeCheckStatusUnchecked
		parent.MergeSHA = nil
		parent.Stats.DiffStats.Commits = nil
		parent.Stats.DiffStats.FilesChanged = nil
		parent.Stats.DiffStats.Additions = nil
		parent.Stats.DiffStats.Deletions = nil
	}
	// Mirror native pr_state.go on close/merge/reopen. UpdatedAt stands in
	// for closed_at/merged_at — proto exposes neither.
	switch parent.State {
	case enum.PullReqStateMerged:
		ts := firstNonZero(prPayload.UpdatedAt, now)
		parent.Closed = &ts
		parent.Merged = &ts
	case enum.PullReqStateClosed:
		ts := firstNonZero(prPayload.UpdatedAt, now)
		parent.Closed = &ts
		parent.Merged = nil
	case enum.PullReqStateOpen:
		parent.Closed = nil
		parent.Merged = nil
	}

	// Pre-bump ActivitySeq atomically with the PR update so concurrent
	// deliveries can't reuse our reserved Order values.
	activityPayloads := h.activityPayloadsFor(
		parent, prevState, prevDraft, prevTitle, prevSourceSHA,
	)
	activitySeqBase := parent.ActivitySeq
	parent.ActivitySeq += int64(len(activityPayloads))

	if err := h.tx.WithTx(ctx, func(ctx context.Context) error {
		if err := h.pullReqStore.Update(ctx, parent); err != nil {
			return fmt.Errorf("update parent pullreq: %w", err)
		}
		existing.ProviderURL = prPayload.HTMLURL
		existing.ProviderAuthorLogin = prPayload.Author.Login
		existing.ProviderAuthorAvatarURL = prPayload.Author.Avatar
		existing.ProviderAuthorURL = prPayload.Author.HTMLURL
		existing.ProviderUpdatedAt = prPayload.UpdatedAt
		// Only stamp MergerLogin on the merge itself — later events (push,
		// label) carry a different sender that would otherwise clobber it.
		if parent.State == enum.PullReqStateMerged && existing.MergerLogin == "" {
			existing.MergerLogin = prPayload.Sender.Login
		}
		if err := h.linkedPullReqStore.Update(ctx, existing); err != nil {
			return fmt.Errorf("update linked pullreq: %w", err)
		}
		return nil
	}); err != nil {
		return err
	}

	if shaChanged {
		// GitHub "synchronize" doesn't expose force-push; Forced=false is fine
		// since the downstream merge-checker re-runs unconditionally.
		h.reporter.BranchUpdated(ctx, &pullreqevents.BranchUpdatedPayload{
			Base:            pullreqBase(parent),
			OldSHA:          prevSourceSHA,
			NewSHA:          parent.SourceSHA,
			OldMergeBaseSHA: prevMergeBaseSHA,
			NewMergeBaseSHA: parent.MergeBaseSHA,
			Forced:          false,
		})
	}

	h.writeActivities(ctx, parent, activitySeqBase, activityPayloads)
	h.publishStateTransition(ctx, parent, prevState)
	return nil
}

// activityPayloadsFor builds the timeline rows for one update, in display order.
func (h *PullRequestHandler) activityPayloadsFor(
	parent *types.PullReq,
	prevState enum.PullReqState,
	prevDraft bool,
	prevTitle string,
	prevSourceSHA string,
) []types.PullReqActivityPayload {
	out := make([]types.PullReqActivityPayload, 0, 3)
	// Gate on head SHA only; base-only moves shouldn't write "pushed commit" rows.
	if prevSourceSHA != parent.SourceSHA {
		out = append(out, &types.PullRequestActivityPayloadBranchUpdate{
			Old:    prevSourceSHA,
			New:    parent.SourceSHA,
			Forced: false,
		})
	}
	if parent.State != prevState || parent.IsDraft != prevDraft {
		out = append(out, &types.PullRequestActivityPayloadStateChange{
			Old: prevState, New: parent.State,
			OldDraft: prevDraft, NewDraft: parent.IsDraft,
		})
	}
	if parent.Title != prevTitle {
		out = append(out, &types.PullRequestActivityPayloadTitleChange{
			Old: prevTitle, New: parent.Title,
		})
	}
	return out
}

// writeActivities emits timeline rows best-effort; failures are logged but
// don't roll back the pullreq update. Attribution is parent.CreatedBy since
// the webhook sender is a provider login, not a Harness principal id.
func (h *PullRequestHandler) writeActivities(
	ctx context.Context,
	parent *types.PullReq,
	baseSeq int64,
	payloads []types.PullReqActivityPayload,
) {
	for i, payload := range payloads {
		snapshot := *parent
		snapshot.ActivitySeq = baseSeq + int64(i+1)
		if _, err := h.activityStore.CreateWithPayload(
			ctx, &snapshot, parent.CreatedBy, payload, nil,
		); err != nil {
			log.Ctx(ctx).Err(err).
				Str("activity_type", string(payload.ActivityType())).
				Int64("pullreq_id", parent.ID).
				Msg("linkedpr: failed to write pull request activity")
		}
	}
}

// firstNonZero returns the first strictly-positive value, or 0 if all
// inputs are zero. Used to fall back across (provider timestamp →
// local-clock now) when the provider timestamp is missing.
func firstNonZero(vs ...int64) int64 {
	for _, v := range vs {
		if v > 0 {
			return v
		}
	}
	return 0
}

func (h *PullRequestHandler) publishCreated(ctx context.Context, pr *types.PullReq) {
	h.reporter.Created(ctx, &pullreqevents.CreatedPayload{
		Base:         pullreqBase(pr),
		SourceBranch: pr.SourceBranch,
		TargetBranch: pr.TargetBranch,
		SourceSHA:    pr.SourceSHA,
	})
}

func (h *PullRequestHandler) publishStateTransition(
	ctx context.Context,
	pr *types.PullReq,
	prevState enum.PullReqState,
) {
	switch {
	case pr.State == enum.PullReqStateMerged && prevState != enum.PullReqStateMerged:
		h.reporter.Merged(ctx, &pullreqevents.MergedPayload{
			Base:      pullreqBase(pr),
			SourceSHA: pr.SourceSHA,
		})
	case pr.State == enum.PullReqStateClosed && prevState != enum.PullReqStateClosed:
		h.reporter.Closed(ctx, &pullreqevents.ClosedPayload{
			Base:         pullreqBase(pr),
			SourceBranch: pr.SourceBranch,
			SourceSHA:    pr.SourceSHA,
		})
	case pr.State == enum.PullReqStateOpen &&
		(prevState == enum.PullReqStateClosed || prevState == enum.PullReqStateMerged):
		// Treat Merged→Open the same as Closed→Open so a stale "open"
		// re-delivery after a merge still fires Reopened subscribers.
		h.reporter.Reopened(ctx, &pullreqevents.ReopenedPayload{
			Base:         pullreqBase(pr),
			SourceBranch: pr.SourceBranch,
			SourceSHA:    pr.SourceSHA,
		})
	default:
		h.reporter.Updated(ctx, &pullreqevents.UpdatedPayload{
			Base: pullreqBase(pr),
		})
	}
}

func pullreqBase(pr *types.PullReq) pullreqevents.Base {
	return pullreqevents.Base{
		PullReqID:    pr.ID,
		SourceRepoID: pr.SourceRepoID,
		TargetRepoID: pr.TargetRepoID,
		PrincipalID:  pr.CreatedBy,
		Number:       pr.Number,
	}
}
