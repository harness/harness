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

package metric

import (
	"context"
	"fmt"
	"time"

	pullreqevents "github.com/harness/gitness/app/events/pullreq"
	repoevents "github.com/harness/gitness/app/events/repo"
	"github.com/harness/gitness/app/services/refcache"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/events"
	"github.com/harness/gitness/stream"
	"github.com/harness/gitness/types"
)

func registerEventListeners(
	ctx context.Context,
	config *types.Config,
	principalInfoCache store.PrincipalInfoCache,
	pullReqStore store.PullReqStore,
	repoReaderFactory *events.ReaderFactory[*repoevents.Reader],
	pullreqEvReaderFactory *events.ReaderFactory[*pullreqevents.Reader],
	repoFinder refcache.RepoFinder,
	submitter Submitter,
) error {
	if submitter == nil {
		return nil
	}

	var err error

	const groupMetricsRepo = "gitness:metrics:repo"
	_, err = repoReaderFactory.Launch(ctx, groupMetricsRepo, config.InstanceID,
		func(r *repoevents.Reader) error {
			const idleTimeout = 10 * time.Second
			r.Configure(
				stream.WithConcurrency(1),
				stream.WithHandlerOptions(
					stream.WithIdleTimeout(idleTimeout),
					stream.WithMaxRetries(2),
				))

			h := handlersRepo{
				principalInfoCache: principalInfoCache,
				repoFinder:         repoFinder,
				submitter:          submitter,
			}

			_ = r.RegisterCreated(h.Create)
			_ = r.RegisterDefaultBranchUpdated(h.DefaultBranchUpdate)
			_ = r.RegisterStateChanged(h.StateChange)
			_ = r.RegisterPublicAccessChanged(h.PublicAccessChange)
			_ = r.RegisterSoftDeleted(h.SoftDelete)

			return nil
		})
	if err != nil {
		return err
	}

	const groupMetricsPullReq = "gitness:metrics:pullreq"
	_, err = pullreqEvReaderFactory.Launch(ctx, groupMetricsPullReq, config.InstanceID,
		func(r *pullreqevents.Reader) error {
			const idleTimeout = 10 * time.Second
			r.Configure(
				stream.WithConcurrency(1),
				stream.WithHandlerOptions(
					stream.WithIdleTimeout(idleTimeout),
					stream.WithMaxRetries(2),
				))

			h := handlersPullReq{
				principalInfoCache: principalInfoCache,
				repoFinder:         repoFinder,
				pullReqStore:       pullReqStore,
				submitter:          submitter,
			}

			_ = r.RegisterCreated(h.Create)
			_ = r.RegisterReopened(h.Reopen)
			_ = r.RegisterClosed(h.Close)
			_ = r.RegisterMerged(h.Merge)

			return nil
		})
	if err != nil {
		return err
	}

	return nil
}

type handlersRepo struct {
	principalInfoCache store.PrincipalInfoCache
	repoFinder         refcache.RepoFinder
	submitter          Submitter
}

func (h handlersRepo) Create(ctx context.Context, e *events.Event[*repoevents.CreatedPayload]) error {
	props := map[string]any{
		"type": e.Payload.Type,
	}
	return h.submit(ctx, e.Payload.RepoID, e.Payload.PrincipalID, VerbRepoCreate, props)
}

func (h handlersRepo) DefaultBranchUpdate(
	ctx context.Context,
	e *events.Event[*repoevents.DefaultBranchUpdatedPayload],
) error {
	props := map[string]any{
		"change":                  "default_branch",
		"repo_old_default_branch": e.Payload.OldName,
		"repo_new_default_branch": e.Payload.NewName,
	}
	return h.submit(ctx, e.Payload.RepoID, e.Payload.PrincipalID, VerbRepoUpdate, props)
}

func (h handlersRepo) StateChange(
	ctx context.Context,
	e *events.Event[*repoevents.StateChangedPayload],
) error {
	props := map[string]any{
		"change":         "state",
		"repo_old_state": e.Payload.OldState,
		"repo_new_state": e.Payload.NewState,
	}
	return h.submit(ctx, e.Payload.RepoID, e.Payload.PrincipalID, VerbRepoUpdate, props)
}

func (h handlersRepo) PublicAccessChange(
	ctx context.Context,
	e *events.Event[*repoevents.PublicAccessChangedPayload],
) error {
	props := map[string]any{
		"change":             "public_access",
		"repo_old_is_public": e.Payload.OldIsPublic,
		"repo_new_is_public": e.Payload.NewIsPublic,
	}
	return h.submit(ctx, e.Payload.RepoID, e.Payload.PrincipalID, VerbRepoUpdate, props)
}

func (h handlersRepo) SoftDelete(
	ctx context.Context,
	e *events.Event[*repoevents.SoftDeletedPayload],
) error {
	return h.submitDeleted(ctx, e.Payload.RepoPath, e.Payload.Deleted, e.Payload.PrincipalID, VerbRepoDelete, nil)
}

func (h handlersRepo) submit(
	ctx context.Context,
	repoID int64,
	principalID int64,
	verb VerbRepo,
	props map[string]any,
) error {
	repo, err := h.repoFinder.FindByID(ctx, repoID)
	if err != nil {
		return fmt.Errorf("failed to find repository")
	}

	err = h.submitMetric(ctx, repo, principalID, verb, props)
	if err != nil {
		return fmt.Errorf("failed to submit event: %w", err)
	}

	return nil
}

func (h handlersRepo) submitDeleted(
	ctx context.Context,
	repoRef string,
	deletedAt int64,
	principalID int64,
	verb VerbRepo,
	props map[string]any,
) error {
	repo, err := h.repoFinder.FindDeletedByRef(ctx, repoRef, deletedAt)
	if err != nil {
		return fmt.Errorf("failed to find delete repo: %w", err)
	}

	err = h.submitMetric(ctx, repo.Core(), principalID, verb, props)
	if err != nil {
		return fmt.Errorf("failed to submit event: %w", err)
	}

	return nil
}

func (h handlersRepo) submitMetric(
	ctx context.Context,
	repo *types.RepositoryCore,
	principalID int64,
	verb VerbRepo,
	props map[string]any,
) error {
	principal, err := h.principalInfoCache.Get(ctx, principalID)
	if err != nil {
		return fmt.Errorf("failed to get principal info: %w", err)
	}

	if props == nil {
		props = make(map[string]any)
	}

	props["repo_id"] = repo.ID
	props["repo_path"] = repo.Path
	props["repo_parent_id"] = repo.ParentID

	err = h.submitter.SubmitForRepo(ctx, principal, verb, props)
	if err != nil {
		return fmt.Errorf("failed to submit metric data for repositoy: %w", err)
	}

	return nil
}

type handlersPullReq struct {
	principalInfoCache store.PrincipalInfoCache
	repoFinder         refcache.RepoFinder
	pullReqStore       store.PullReqStore
	submitter          Submitter
}

func (h handlersPullReq) Create(ctx context.Context, e *events.Event[*pullreqevents.CreatedPayload]) error {
	return h.submit(ctx, e.Payload.PullReqID, e.Payload.PrincipalID, VerbPullReqCreate)
}

func (h handlersPullReq) Close(ctx context.Context, e *events.Event[*pullreqevents.ClosedPayload]) error {
	return h.submit(ctx, e.Payload.PullReqID, e.Payload.PrincipalID, VerbPullReqClose)
}

func (h handlersPullReq) Reopen(ctx context.Context, e *events.Event[*pullreqevents.ReopenedPayload]) error {
	return h.submit(ctx, e.Payload.PullReqID, e.Payload.PrincipalID, VerbPullReqReopen)
}

func (h handlersPullReq) Merge(ctx context.Context, e *events.Event[*pullreqevents.MergedPayload]) error {
	return h.submit(ctx, e.Payload.PullReqID, e.Payload.PrincipalID, VerbPullReqMerge)
}

func (h handlersPullReq) submit(ctx context.Context, pullReqID, principalID int64, verb VerbPullReq) error {
	pr, err := h.pullReqStore.Find(ctx, pullReqID)
	if err != nil {
		return fmt.Errorf("failed to find pull request: %w", err)
	}

	repo, err := h.repoFinder.FindByID(ctx, pr.TargetRepoID)
	if err != nil {
		return fmt.Errorf("failed to find pull request: %w", err)
	}

	principal, err := h.principalInfoCache.Get(ctx, principalID)
	if err != nil {
		return fmt.Errorf("failed to get principal info: %w", err)
	}

	author, err := h.principalInfoCache.Get(ctx, pr.CreatedBy)
	if err != nil {
		return fmt.Errorf("failed to get author principal info: %w", err)
	}

	props := map[string]any{
		"repo_id":                repo.ID,
		"repo_path":              repo.Path,
		"repo_parent_id":         repo.ParentID,
		"pullreq_author_email":   author.Email,
		"pullreq_number":         pr.Number,
		"pullreq_target_repo_id": pr.TargetRepoID,
		"pullreq_source_repo_id": pr.SourceRepoID,
		"pullreq_target_branch":  pr.TargetBranch,
		"pullreq_source_branch":  pr.SourceBranch,
	}

	if pr.MergeMethod != nil {
		props["pullreq_merge_method"] = string(*pr.MergeMethod)
	}

	err = h.submitter.SubmitForPullReq(ctx, principal, verb, props)
	if err != nil {
		return fmt.Errorf("failed to submit metric data for pull request: %w", err)
	}

	return nil
}
