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
	ruleevents "github.com/harness/gitness/app/events/rule"
	userevents "github.com/harness/gitness/app/events/user"
	"github.com/harness/gitness/app/services/publicaccess"
	"github.com/harness/gitness/app/services/refcache"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/events"
	"github.com/harness/gitness/stream"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

func registerEventListeners(
	ctx context.Context,
	config *types.Config,
	principalInfoCache store.PrincipalInfoCache,
	pullReqStore store.PullReqStore,
	ruleStore store.RuleStore,
	userEvReaderFactory *events.ReaderFactory[*userevents.Reader],
	repoEvReaderFactory *events.ReaderFactory[*repoevents.Reader],
	pullreqEvReaderFactory *events.ReaderFactory[*pullreqevents.Reader],
	ruleEvReaderFactory *events.ReaderFactory[*ruleevents.Reader],
	spaceFinder refcache.SpaceFinder,
	repoFinder refcache.RepoFinder,
	publicAccess publicaccess.Service,
	submitter Submitter,
) error {
	if submitter == nil {
		return nil
	}

	var err error

	const groupMetricsUser = "gitness:metrics:user"
	_, err = userEvReaderFactory.Launch(ctx, groupMetricsUser, config.InstanceID,
		func(r *userevents.Reader) error {
			const idleTimeout = 10 * time.Second
			r.Configure(
				stream.WithConcurrency(1),
				stream.WithHandlerOptions(
					stream.WithIdleTimeout(idleTimeout),
					stream.WithMaxRetries(2),
				))

			h := handlersUser{
				principalInfoCache: principalInfoCache,
				submitter:          submitter,
			}

			_ = r.RegisterCreated(h.Create)
			_ = r.RegisterRegistered(h.Register)
			_ = r.RegisterLoggedIn(h.Login)

			return nil
		})
	if err != nil {
		return err
	}

	const groupMetricsRepo = "gitness:metrics:repo"
	_, err = repoEvReaderFactory.Launch(ctx, groupMetricsRepo, config.InstanceID,
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
				publicAccess:       publicAccess,
				submitter:          submitter,
			}

			_ = r.RegisterCreated(h.Create)
			_ = r.RegisterPushed(h.Push)
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
				publicAccess:       publicAccess,
				submitter:          submitter,
			}

			_ = r.RegisterCreated(h.Create)
			_ = r.RegisterReopened(h.Reopen)
			_ = r.RegisterClosed(h.Close)
			_ = r.RegisterMerged(h.Merge)
			_ = r.RegisterCommentCreated(h.CommentCreate)

			return nil
		})
	if err != nil {
		return err
	}

	const groupMetricsRule = "gitness:metrics:rule"
	_, err = ruleEvReaderFactory.Launch(ctx, groupMetricsRule, config.InstanceID,
		func(r *ruleevents.Reader) error {
			const idleTimeout = 10 * time.Second
			r.Configure(
				stream.WithConcurrency(1),
				stream.WithHandlerOptions(
					stream.WithIdleTimeout(idleTimeout),
					stream.WithMaxRetries(2),
				))

			h := handlersRule{
				principalInfoCache: principalInfoCache,
				spaceFinder:        spaceFinder,
				repoFinder:         repoFinder,
				ruleStore:          ruleStore,
				publicAccess:       publicAccess,
				submitter:          submitter,
			}

			_ = r.RegisterCreated(h.Create)

			return nil
		})
	if err != nil {
		return err
	}

	return nil
}

func prepareProps(m map[string]any) map[string]any {
	if m != nil {
		return m
	}
	return make(map[string]any, 8)
}

// User fields.
const (
	userID             = "user_id"
	userName           = "user_name"
	userEmail          = "user_email"
	userCreatedByID    = "user_created_by_id"
	userCreatedByName  = "user_created_by_name"
	userCreatedByEmail = "user_created_by_email"
)

type handlersUser struct {
	principalInfoCache store.PrincipalInfoCache
	submitter          Submitter
}

func (h handlersUser) Register(ctx context.Context, e *events.Event[*userevents.RegisteredPayload]) error {
	return h.submit(ctx, e.Payload.PrincipalID, VerbUserCreate, nil)
}

func (h handlersUser) Create(ctx context.Context, e *events.Event[*userevents.CreatedPayload]) error {
	principal, err := h.principalInfoCache.Get(ctx, e.Payload.PrincipalID)
	if err != nil {
		return fmt.Errorf("failed to find principal who created a user: %w", err)
	}

	props := prepareProps(nil)
	props[userCreatedByID] = principal.ID
	props[userCreatedByName] = principal.UID
	props[userCreatedByEmail] = principal.Email

	return h.submit(ctx, e.Payload.CreatedPrincipalID, VerbUserCreate, props)
}

func (h handlersUser) Login(ctx context.Context, e *events.Event[*userevents.LoggedInPayload]) error {
	return h.submit(ctx, e.Payload.PrincipalID, VerbUserLogin, nil)
}

func (h handlersUser) submit(
	ctx context.Context,
	principalID int64,
	verb Verb,
	props map[string]any,
) error {
	principal, err := h.principalInfoCache.Get(ctx, principalID)
	if err != nil {
		return fmt.Errorf("failed to find principal info")
	}

	props = prepareProps(props)
	props[userID] = principal.ID
	props[userName] = principal.UID
	props[userEmail] = principal.Email

	err = h.submitter.Submit(ctx, principal, ObjectUser, verb, props)
	if err != nil {
		return fmt.Errorf("failed to submit metric data for user: %w", err)
	}

	return nil
}

// Space fields.
const (
	spaceID       = "space_id"
	spaceName     = "space_name"
	spacePath     = "space_path"
	spaceParentID = "space_parent_id"
	spacePrivate  = "space_private"
)

// Repository fields.
const (
	repoID           = "repo_id"
	repoName         = "repo_name"
	repoPath         = "repo_path"
	repoParentID     = "repo_parent_id"
	repoPrivate      = "repo_private"
	repoMigrated     = "repo_migrated"
	repoImported     = "repo_imported"
	repoImportedFrom = "repo_imported_from"
)

type handlersRepo struct {
	principalInfoCache store.PrincipalInfoCache
	repoFinder         refcache.RepoFinder
	publicAccess       publicaccess.Service
	submitter          Submitter
}

func (h handlersRepo) Create(ctx context.Context, e *events.Event[*repoevents.CreatedPayload]) error {
	props := prepareProps(nil)
	props[repoPrivate] = !e.Payload.IsPublic
	if e.Payload.IsMigrated {
		props[repoMigrated] = true
	}
	if e.Payload.ImportedFrom != "" {
		props[repoImported] = true
		props[repoImportedFrom] = e.Payload.ImportedFrom
	}
	return h.submitForActive(ctx, e.Payload.RepoID, e.Payload.PrincipalID, VerbRepoCreate, props)
}

func (h handlersRepo) Push(
	ctx context.Context,
	e *events.Event[*repoevents.PushedPayload],
) error {
	return h.submitForActive(ctx, e.Payload.RepoID, e.Payload.PrincipalID, VerbRepoPush, nil)
}

func (h handlersRepo) SoftDelete(
	ctx context.Context,
	e *events.Event[*repoevents.SoftDeletedPayload],
) error {
	return h.submitForDeleted(ctx, e.Payload.RepoPath, e.Payload.Deleted, e.Payload.PrincipalID, VerbRepoDelete, nil)
}

func (h handlersRepo) submitForActive(
	ctx context.Context,
	id int64,
	principalID int64,
	verb Verb,
	props map[string]any,
) error {
	repo, err := h.repoFinder.FindByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to find repository")
	}

	props, err = fillRepoData(ctx, props, repo, h.publicAccess)
	if err != nil {
		return fmt.Errorf("failed to fill repo data: %w", err)
	}

	err = h.submit(ctx, principalID, verb, props)
	if err != nil {
		return fmt.Errorf("failed to submit event: %w", err)
	}

	return nil
}

func (h handlersRepo) submitForDeleted(
	ctx context.Context,
	repoRef string,
	deletedAt int64,
	principalID int64,
	verb Verb,
	props map[string]any,
) error {
	repo, err := h.repoFinder.FindDeletedByRef(ctx, repoRef, deletedAt)
	if err != nil {
		return fmt.Errorf("failed to find deleted repo: %w", err)
	}

	props, err = fillRepoData(ctx, props, repo.Core(), nil)
	if err != nil {
		return fmt.Errorf("failed to fill deleted repo data: %w", err)
	}

	err = h.submit(ctx, principalID, verb, props)
	if err != nil {
		return fmt.Errorf("failed to submit metric data for deleted repository: %w", err)
	}

	return nil
}

func (h handlersRepo) submit(
	ctx context.Context,
	principalID int64,
	verb Verb,
	props map[string]any,
) error {
	principal, err := h.principalInfoCache.Get(ctx, principalID)
	if err != nil {
		return fmt.Errorf("failed to get principal info: %w", err)
	}

	err = h.submitter.Submit(ctx, principal, ObjectRepository, verb, props)
	if err != nil {
		return fmt.Errorf("failed to submit metric data for repositoy: %w", err)
	}

	return nil
}

// Pull request fields.
const (
	prNumber         = "pr_number"
	prTargetBranch   = "pr_target_branch"
	prSourceBranch   = "pr_source_branch"
	prSourceRepoID   = "pr_source_repo_id"
	prSourceRepoName = "pr_source_repo_name"
	prSourceRepoPath = "pr_source_repo_path"
	prMergeMethod    = "pr_merge_method"
	prCommentReply   = "pr_comment_reply"
)

type handlersPullReq struct {
	principalInfoCache store.PrincipalInfoCache
	repoFinder         refcache.RepoFinder
	pullReqStore       store.PullReqStore
	publicAccess       publicaccess.Service
	submitter          Submitter
}

func (h handlersPullReq) Create(ctx context.Context, e *events.Event[*pullreqevents.CreatedPayload]) error {
	return h.submit(ctx, e.Payload.PullReqID, e.Payload.PrincipalID, VerbPullReqCreate, nil)
}

func (h handlersPullReq) Close(ctx context.Context, e *events.Event[*pullreqevents.ClosedPayload]) error {
	return h.submit(ctx, e.Payload.PullReqID, e.Payload.PrincipalID, VerbPullReqClose, nil)
}

func (h handlersPullReq) Reopen(ctx context.Context, e *events.Event[*pullreqevents.ReopenedPayload]) error {
	return h.submit(ctx, e.Payload.PullReqID, e.Payload.PrincipalID, VerbPullReqReopen, nil)
}

func (h handlersPullReq) Merge(ctx context.Context, e *events.Event[*pullreqevents.MergedPayload]) error {
	return h.submit(ctx, e.Payload.PullReqID, e.Payload.PrincipalID, VerbPullReqMerge, nil)
}

func (h handlersPullReq) CommentCreate(
	ctx context.Context,
	e *events.Event[*pullreqevents.CommentCreatedPayload],
) error {
	props := prepareProps(nil)
	props[prCommentReply] = e.Payload.IsReply
	return h.submit(ctx, e.Payload.PullReqID, e.Payload.PrincipalID, VerbPullReqComment, props)
}

func (h handlersPullReq) submit(
	ctx context.Context,
	pullReqID, principalID int64,
	verb Verb,
	props map[string]any,
) error {
	pr, err := h.pullReqStore.Find(ctx, pullReqID)
	if err != nil {
		return fmt.Errorf("failed to find pull request: %w", err)
	}

	props, err = fillPullReqProps(ctx, props, pr, h.repoFinder, h.publicAccess)
	if err != nil {
		return fmt.Errorf("failed to fill pull request props: %w", err)
	}

	principal, err := h.principalInfoCache.Get(ctx, principalID)
	if err != nil {
		return fmt.Errorf("failed to get principal info: %w", err)
	}

	err = h.submitter.Submit(ctx, principal, ObjectPullRequest, verb, props)
	if err != nil {
		return fmt.Errorf("failed to submit metric data for pull request: %w", err)
	}

	return nil
}

// Rule fields.
const (
	ruleID   = "rule_id"
	ruleName = "rule_name"
	ruleType = "rule_type"
)

func (h handlersRule) Create(ctx context.Context, e *events.Event[*ruleevents.CreatedPayload]) error {
	return h.submit(ctx, e.Payload.RuleID, e.Payload.PrincipalID, VerbRuleCreate, nil)
}

func (h handlersRule) submit(
	ctx context.Context,
	ruleID, principalID int64,
	verb Verb,
	props map[string]any,
) error {
	rule, err := h.ruleStore.Find(ctx, ruleID)
	if err != nil {
		return fmt.Errorf("failed to find pull request: %w", err)
	}

	props, err = fillRuleProps(ctx, props, rule, h.spaceFinder, h.repoFinder, h.publicAccess)
	if err != nil {
		return fmt.Errorf("failed to fill pull request props: %w", err)
	}

	principal, err := h.principalInfoCache.Get(ctx, principalID)
	if err != nil {
		return fmt.Errorf("failed to get principal info: %w", err)
	}

	err = h.submitter.Submit(ctx, principal, ObjectRule, verb, props)
	if err != nil {
		return fmt.Errorf("failed to submit metric data for rule: %w", err)
	}

	return nil
}

type handlersRule struct {
	principalInfoCache store.PrincipalInfoCache
	spaceFinder        refcache.SpaceFinder
	repoFinder         refcache.RepoFinder
	ruleStore          store.RuleStore
	publicAccess       publicaccess.Service
	submitter          Submitter
}

func fillSpaceData(
	ctx context.Context,
	props map[string]any,
	space *types.SpaceCore,
	publicAccess publicaccess.Service,
) (map[string]any, error) {
	props = prepareProps(props)
	props[spaceID] = space.ID
	props[spaceName] = space.Identifier
	props[spacePath] = space.Path
	props[spaceParentID] = space.ParentID

	if _, ok := props[spacePrivate]; !ok && publicAccess != nil {
		isRepoPublic, err := publicAccess.Get(ctx, enum.PublicResourceTypeSpace, space.Path)
		if err != nil {
			return nil, fmt.Errorf("failed to check public access for space: %w", err)
		}
		props[spacePrivate] = !isRepoPublic
	}

	return props, nil
}

func fillRepoData(
	ctx context.Context,
	props map[string]any,
	repo *types.RepositoryCore,
	publicAccess publicaccess.Service,
) (map[string]any, error) {
	props = prepareProps(props)
	props[repoID] = repo.ID
	props[repoName] = repo.Identifier
	props[repoPath] = repo.Path
	props[repoParentID] = repo.ParentID

	if _, ok := props[repoPrivate]; !ok && publicAccess != nil {
		isRepoPublic, err := publicAccess.Get(ctx, enum.PublicResourceTypeRepo, repo.Path)
		if err != nil {
			return nil, fmt.Errorf("failed to check public access for repo: %w", err)
		}
		props[repoPrivate] = !isRepoPublic
	}

	return props, nil
}

func fillPullReqProps(
	ctx context.Context,
	props map[string]any,
	pr *types.PullReq,
	repoFinder refcache.RepoFinder,
	publicAccess publicaccess.Service,
) (map[string]any, error) {
	props = prepareProps(props)
	props[prNumber] = pr.Number
	props[prSourceBranch] = pr.SourceBranch
	props[prTargetBranch] = pr.TargetBranch
	if pr.MergeMethod != nil {
		props[prMergeMethod] = string(*pr.MergeMethod)
	}

	targetRepo, err := repoFinder.FindByID(ctx, pr.TargetRepoID)
	if err != nil {
		return nil, fmt.Errorf("failed to find target repo: %w", err)
	}

	props, err = fillRepoData(ctx, props, targetRepo, publicAccess)
	if err != nil {
		return nil, fmt.Errorf("failed to fill repo data for target repo: %w", err)
	}

	if pr.SourceRepoID != pr.TargetRepoID {
		sourceRepo, err := repoFinder.FindByID(ctx, pr.SourceRepoID)
		if err != nil {
			return nil, fmt.Errorf("failed to find source repo: %w", err)
		}

		props[prSourceRepoID] = pr.SourceRepoID
		props[prSourceRepoName] = sourceRepo.Identifier
		props[prSourceRepoPath] = sourceRepo.Path
	}

	return props, nil
}

func fillRuleProps(
	ctx context.Context,
	props map[string]any,
	rule *types.Rule,
	spaceFinder refcache.SpaceFinder,
	repoFinder refcache.RepoFinder,
	publicAccess publicaccess.Service,
) (map[string]any, error) {
	props = prepareProps(props)
	props[ruleID] = rule.RepoID
	props[ruleName] = rule.Identifier
	props[ruleType] = string(rule.Type)

	//nolint:nestif
	if rule.SpaceID != nil {
		space, err := spaceFinder.FindByID(ctx, *rule.SpaceID)
		if err != nil {
			return nil, fmt.Errorf("failed to find space: %w", err)
		}

		props, err = fillSpaceData(ctx, props, space, publicAccess)
		if err != nil {
			return nil, fmt.Errorf("failed to fill space data for rule: %w", err)
		}
	} else if rule.RepoID != nil {
		repo, err := repoFinder.FindByID(ctx, *rule.RepoID)
		if err != nil {
			return nil, fmt.Errorf("failed to find repo: %w", err)
		}

		props, err = fillRepoData(ctx, props, repo, publicAccess)
		if err != nil {
			return nil, fmt.Errorf("failed to fill repo data for rule: %w", err)
		}
	}

	return props, nil
}
