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

// Package store defines the data storage interfaces.
package store

import (
	"context"
	"encoding/json"
	"time"

	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

type (
	// PrincipalStore defines the principal data storage.
	PrincipalStore interface {
		/*
		 * PRINCIPAL RELATED OPERATIONS.
		 */
		// Find finds the principal by id.
		Find(ctx context.Context, id int64) (*types.Principal, error)

		// FindByUID finds the principal by uid.
		FindByUID(ctx context.Context, uid string) (*types.Principal, error)

		// FindManyByUID returns all principals found for the provided UIDs.
		// If a UID isn't found, it's not returned in the list.
		FindManyByUID(ctx context.Context, uids []string) ([]*types.Principal, error)

		// FindByEmail finds the principal by email.
		FindByEmail(ctx context.Context, email string) (*types.Principal, error)

		/*
		 * USER RELATED OPERATIONS.
		 */

		// FindUser finds the user by id.
		FindUser(ctx context.Context, id int64) (*types.User, error)

		// List lists the principals matching the provided filter.
		List(ctx context.Context, fetchQuery *types.PrincipalFilter) ([]*types.Principal, error)

		// FindUserByUID finds the user by uid.
		FindUserByUID(ctx context.Context, uid string) (*types.User, error)

		// FindUserByEmail finds the user by email.
		FindUserByEmail(ctx context.Context, email string) (*types.User, error)

		// CreateUser saves the user details.
		CreateUser(ctx context.Context, user *types.User) error

		// UpdateUser updates an existing user.
		UpdateUser(ctx context.Context, user *types.User) error

		// DeleteUser deletes the user.
		DeleteUser(ctx context.Context, id int64) error

		// ListUsers returns a list of users.
		ListUsers(ctx context.Context, params *types.UserFilter) ([]*types.User, error)

		// CountUsers returns a count of users which match the given filter.
		CountUsers(ctx context.Context, opts *types.UserFilter) (int64, error)

		/*
		 * SERVICE ACCOUNT RELATED OPERATIONS.
		 */

		// FindServiceAccount finds the service account by id.
		FindServiceAccount(ctx context.Context, id int64) (*types.ServiceAccount, error)

		// FindServiceAccountByUID finds the service account by uid.
		FindServiceAccountByUID(ctx context.Context, uid string) (*types.ServiceAccount, error)

		// CreateServiceAccount saves the service account.
		CreateServiceAccount(ctx context.Context, sa *types.ServiceAccount) error

		// UpdateServiceAccount updates the service account details.
		UpdateServiceAccount(ctx context.Context, sa *types.ServiceAccount) error

		// DeleteServiceAccount deletes the service account.
		DeleteServiceAccount(ctx context.Context, id int64) error

		// ListServiceAccounts returns a list of service accounts for a specific parent.
		ListServiceAccounts(ctx context.Context,
			parentType enum.ParentResourceType, parentID int64) ([]*types.ServiceAccount, error)

		// CountServiceAccounts returns a count of service accounts for a specific parent.
		CountServiceAccounts(ctx context.Context,
			parentType enum.ParentResourceType, parentID int64) (int64, error)

		/*
		 * SERVICE RELATED OPERATIONS.
		 */

		// FindService finds the service by id.
		FindService(ctx context.Context, id int64) (*types.Service, error)

		// FindServiceByUID finds the service by uid.
		FindServiceByUID(ctx context.Context, uid string) (*types.Service, error)

		// CreateService saves the service.
		CreateService(ctx context.Context, sa *types.Service) error

		// UpdateService updates the service.
		UpdateService(ctx context.Context, sa *types.Service) error

		// DeleteService deletes the service.
		DeleteService(ctx context.Context, id int64) error

		// ListServices returns a list of service for a specific parent.
		ListServices(ctx context.Context) ([]*types.Service, error)

		// CountServices returns a count of service for a specific parent.
		CountServices(ctx context.Context) (int64, error)
	}

	// PrincipalInfoView defines helper utility for fetching types.PrincipalInfo objects.
	// It uses the same underlying data storage as PrincipalStore.
	PrincipalInfoView interface {
		Find(ctx context.Context, id int64) (*types.PrincipalInfo, error)
		FindMany(ctx context.Context, ids []int64) ([]*types.PrincipalInfo, error)
	}

	// SpacePathStore defines the path data storage for spaces.
	SpacePathStore interface {
		// InsertSegment inserts a space path segment to the table.
		InsertSegment(ctx context.Context, segment *types.SpacePathSegment) error

		// FindPrimaryBySpaceID finds the primary path of a space given its ID.
		FindPrimaryBySpaceID(ctx context.Context, spaceID int64) (*types.SpacePath, error)

		// FindByPath returns the space path for a given raw path.
		FindByPath(ctx context.Context, path string) (*types.SpacePath, error)

		// DeletePrimarySegment deletes the primary segment of a space.
		DeletePrimarySegment(ctx context.Context, spaceID int64) error

		// DeletePathsAndDescendandPaths deletes all space paths reachable from spaceID including itself.
		DeletePathsAndDescendandPaths(ctx context.Context, spaceID int64) error
	}

	// SpaceStore defines the space data storage.
	SpaceStore interface {
		// Find the space by id.
		Find(ctx context.Context, id int64) (*types.Space, error)

		// FindByRef finds the space using the spaceRef as either the id or the space path.
		FindByRef(ctx context.Context, spaceRef string) (*types.Space, error)

		// FindByRefAndDeletedAt finds the space using the spaceRef and deleted timestamp.
		FindByRefAndDeletedAt(ctx context.Context, spaceRef string, deletedAt int64) (*types.Space, error)

		// GetRootSpace returns a space where space_parent_id is NULL.
		GetRootSpace(ctx context.Context, spaceID int64) (*types.Space, error)

		// GetAncestorIDs returns a list of all space IDs along the recursive path to the root space.
		GetAncestorIDs(ctx context.Context, spaceID int64) ([]int64, error)

		GetHierarchy(
			ctx context.Context,
			spaceID int64,
		) ([]*types.Space, error)

		// Create creates a new space
		Create(ctx context.Context, space *types.Space) error

		// Update updates the space details.
		Update(ctx context.Context, space *types.Space) error

		// UpdateOptLock updates the space using the optimistic locking mechanism.
		UpdateOptLock(ctx context.Context, space *types.Space,
			mutateFn func(space *types.Space) error) (*types.Space, error)

		// FindForUpdate finds the space and locks it for an update.
		FindForUpdate(ctx context.Context, id int64) (*types.Space, error)

		// SoftDelete deletes the space.
		SoftDelete(ctx context.Context, space *types.Space, deletedAt int64) error

		// Purge deletes a space permanently.
		Purge(ctx context.Context, id int64, deletedAt *int64) error

		// Restore restores a soft deleted space.
		Restore(ctx context.Context, space *types.Space,
			newIdentifier *string, newParentID *int64) (*types.Space, error)

		// Count the child spaces of a space.
		Count(ctx context.Context, id int64, opts *types.SpaceFilter) (int64, error)

		// List returns a list of child spaces in a space.
		List(ctx context.Context, id int64, opts *types.SpaceFilter) ([]*types.Space, error)
	}

	// RepoStore defines the repository data storage.
	RepoStore interface {
		// Find the repo by id.
		Find(ctx context.Context, id int64) (*types.Repository, error)

		// FindByRefAndDeletedAt finds the repo using the repoRef and deleted timestamp.
		FindByRefAndDeletedAt(ctx context.Context, repoRef string, deletedAt int64) (*types.Repository, error)

		// FindByRef finds the repo using the repoRef as either the id or the repo path.
		FindByRef(ctx context.Context, repoRef string) (*types.Repository, error)

		// Create a new repo.
		Create(ctx context.Context, repo *types.Repository) error

		// Update the repo details.
		Update(ctx context.Context, repo *types.Repository) error

		// UpdateSize updates the size of a specific repository in the database (size is in KiB).
		UpdateSize(ctx context.Context, id int64, sizeInKiB int64) error

		// Get the repo size.
		GetSize(ctx context.Context, id int64) (int64, error)

		// UpdateOptLock the repo details using the optimistic locking mechanism.
		UpdateOptLock(ctx context.Context, repo *types.Repository,
			mutateFn func(repository *types.Repository) error) (*types.Repository, error)

		// SoftDelete a repo.
		SoftDelete(ctx context.Context, repo *types.Repository, deletedAt int64) error

		// Purge the soft deleted repo permanently.
		Purge(ctx context.Context, id int64, deletedAt *int64) error

		// Restore a deleted repo using the optimistic locking mechanism.
		Restore(ctx context.Context, repo *types.Repository,
			newIdentifier *string, newParentID *int64) (*types.Repository, error)

		// Count of active repos in a space. With "DeletedBeforeOrAt" filter, counts deleted repos.
		Count(ctx context.Context, parentID int64, opts *types.RepoFilter) (int64, error)

		// List returns a list of repos in a space. With "DeletedBeforeOrAt" filter, lists deleted repos.
		List(ctx context.Context, parentID int64, opts *types.RepoFilter) ([]*types.Repository, error)

		// ListSizeInfos returns a list of all active repo sizes.
		ListSizeInfos(ctx context.Context) ([]*types.RepositorySizeInfo, error)
	}

	// SettingsStore defines the settings storage.
	SettingsStore interface {
		// Find returns the value of the setting with the given key for the provided scope.
		Find(
			ctx context.Context,
			scope enum.SettingsScope,
			scopeID int64,
			key string,
		) (json.RawMessage, error)

		// FindMany returns the values of the settings with the given keys for the provided scope.
		// NOTE: if a setting key doesn't exist the map just won't contain an entry for it (no error returned).
		FindMany(
			ctx context.Context,
			scope enum.SettingsScope,
			scopeID int64,
			keys ...string,
		) (map[string]json.RawMessage, error)

		// Upsert upserts the value of the setting with the given key for the provided scope.
		Upsert(
			ctx context.Context,
			scope enum.SettingsScope,
			scopeID int64,
			key string,
			value json.RawMessage,
		) error
	}

	// RepoGitInfoView defines the repository GitUID view.
	RepoGitInfoView interface {
		Find(ctx context.Context, id int64) (*types.RepositoryGitInfo, error)
	}

	// MembershipStore defines the membership data storage.
	MembershipStore interface {
		Find(ctx context.Context, key types.MembershipKey) (*types.Membership, error)
		FindUser(ctx context.Context, key types.MembershipKey) (*types.MembershipUser, error)
		Create(ctx context.Context, membership *types.Membership) error
		Update(ctx context.Context, membership *types.Membership) error
		Delete(ctx context.Context, key types.MembershipKey) error
		CountUsers(ctx context.Context, spaceID int64, filter types.MembershipUserFilter) (int64, error)
		ListUsers(ctx context.Context, spaceID int64, filter types.MembershipUserFilter) ([]types.MembershipUser, error)
		CountSpaces(ctx context.Context, userID int64, filter types.MembershipSpaceFilter) (int64, error)
		ListSpaces(ctx context.Context, userID int64, filter types.MembershipSpaceFilter) ([]types.MembershipSpace, error)
	}

	// PublicAccessStore defines the publicly accessible resources data storage.
	PublicAccessStore interface {
		Find(ctx context.Context, typ enum.PublicResourceType, id int64) (bool, error)
		Create(ctx context.Context, typ enum.PublicResourceType, id int64) error
		Delete(ctx context.Context, typ enum.PublicResourceType, id int64) error
	}

	// TokenStore defines the token data storage.
	TokenStore interface {
		// Find finds the token by id
		Find(ctx context.Context, id int64) (*types.Token, error)

		// FindByIdentifier finds the token by principalId and token identifier.
		FindByIdentifier(ctx context.Context, principalID int64, identifier string) (*types.Token, error)

		// Create saves the token details.
		Create(ctx context.Context, token *types.Token) error

		// Delete deletes the token with the given id.
		Delete(ctx context.Context, id int64) error

		// DeleteExpiredBefore deletes all tokens that expired before the provided time.
		// If tokenTypes are provided, then only tokens of that type are deleted.
		DeleteExpiredBefore(ctx context.Context, before time.Time, tknTypes []enum.TokenType) (int64, error)

		// List returns a list of tokens of a specific type for a specific principal.
		List(ctx context.Context, principalID int64, tokenType enum.TokenType) ([]*types.Token, error)

		// Count returns a count of tokens of a specifc type for a specific principal.
		Count(ctx context.Context, principalID int64, tokenType enum.TokenType) (int64, error)
	}

	// PullReqStore defines the pull request data storage.
	PullReqStore interface {
		// Find the pull request by id.
		Find(ctx context.Context, id int64) (*types.PullReq, error)

		// FindByNumberWithLock finds the pull request by repo ID and the pull request number
		// and acquires an exclusive lock of the pull request database row for the duration of the transaction.
		FindByNumberWithLock(ctx context.Context, repoID, number int64) (*types.PullReq, error)

		// FindByNumber finds the pull request by repo ID and the pull request number.
		FindByNumber(ctx context.Context, repoID, number int64) (*types.PullReq, error)

		// Create a new pull request.
		Create(ctx context.Context, pullreq *types.PullReq) error

		// Update the pull request. It will set new values to the Version and Updated fields.
		Update(ctx context.Context, pr *types.PullReq) error

		// UpdateOptLock the pull request details using the optimistic locking mechanism.
		UpdateOptLock(ctx context.Context, pr *types.PullReq,
			mutateFn func(pr *types.PullReq) error) (*types.PullReq, error)

		// UpdateActivitySeq the pull request's activity sequence number.
		// It will set new values to the ActivitySeq, Version and Updated fields.
		UpdateActivitySeq(ctx context.Context, pr *types.PullReq) (*types.PullReq, error)

		// ResetMergeCheckStatus resets the pull request's mergeability status to unchecked
		// for all prs with target branch pointing to targetBranch.
		ResetMergeCheckStatus(ctx context.Context, targetRepo int64, targetBranch string) error

		// Delete the pull request.
		Delete(ctx context.Context, id int64) error

		// Count of pull requests in a space.
		Count(ctx context.Context, opts *types.PullReqFilter) (int64, error)

		// List returns a list of pull requests in a space.
		List(ctx context.Context, opts *types.PullReqFilter) ([]*types.PullReq, error)
	}

	PullReqActivityStore interface {
		// Find the pull request activity by id.
		Find(ctx context.Context, id int64) (*types.PullReqActivity, error)

		// Create a new pull request activity. Value of the Order field should be fetched with UpdateActivitySeq.
		// Value of the SubOrder field (for replies) should be the incremented ReplySeq field (non-replies have 0).
		Create(ctx context.Context, act *types.PullReqActivity) error

		// CreateWithPayload create a new system activity from the provided payload.
		CreateWithPayload(ctx context.Context,
			pr *types.PullReq,
			principalID int64,
			payload types.PullReqActivityPayload,
			metadata *types.PullReqActivityMetadata,
		) (*types.PullReqActivity, error)

		// Update the pull request activity. It will set new values to the Version and Updated fields.
		Update(ctx context.Context, act *types.PullReqActivity) error

		// UpdateOptLock updates the pull request activity using the optimistic locking mechanism.
		UpdateOptLock(ctx context.Context,
			act *types.PullReqActivity,
			mutateFn func(act *types.PullReqActivity) error,
		) (*types.PullReqActivity, error)

		// Count returns number of pull request activities in a pull request.
		Count(ctx context.Context, prID int64, opts *types.PullReqActivityFilter) (int64, error)

		// CountUnresolved returns number of unresolved comments.
		CountUnresolved(ctx context.Context, prID int64) (int, error)

		// List returns a list of pull request activities in a pull request (a timeline).
		List(ctx context.Context, prID int64, opts *types.PullReqActivityFilter) ([]*types.PullReqActivity, error)

		// ListAuthorIDs returns a list of pull request activity author ids in a thread (order).
		ListAuthorIDs(ctx context.Context, prID int64, order int64) ([]int64, error)
	}

	// CodeCommentView is to manipulate only code-comment subset of PullReqActivity.
	// It's used by internal service that migrates code comment line numbers after new commits.
	CodeCommentView interface {
		// ListNotAtSourceSHA loads code comments that need to be updated after a new commit.
		// Resulting list is ordered by the file name and the relevant line number.
		ListNotAtSourceSHA(ctx context.Context, prID int64, sourceSHA string) ([]*types.CodeComment, error)

		// ListNotAtMergeBaseSHA loads code comments that need to be updated after merge base update.
		// Resulting list is ordered by the file name and the relevant line number.
		ListNotAtMergeBaseSHA(ctx context.Context, prID int64, targetSHA string) ([]*types.CodeComment, error)

		// UpdateAll updates code comments (pull request activity of types code-comment).
		// entities coming from the input channel.
		UpdateAll(ctx context.Context, codeComments []*types.CodeComment) error
	}

	// PullReqReviewStore defines the pull request review storage.
	PullReqReviewStore interface {
		// Find returns the pull request review entity or an error if it doesn't exist.
		Find(ctx context.Context, id int64) (*types.PullReqReview, error)

		// Create creates a new pull request review.
		Create(ctx context.Context, v *types.PullReqReview) error
	}

	// PullReqReviewerStore defines the pull request reviewer storage.
	PullReqReviewerStore interface {
		// Find returns the pull request reviewer or an error if it doesn't exist.
		Find(ctx context.Context, prID, principalID int64) (*types.PullReqReviewer, error)

		// Create creates the new pull request reviewer.
		Create(ctx context.Context, v *types.PullReqReviewer) error

		// Update updates the pull request reviewer.
		Update(ctx context.Context, v *types.PullReqReviewer) error

		// Delete the Pull request reviewer
		Delete(ctx context.Context, prID, principalID int64) error

		// List returns all pull request reviewers for the pull request.
		List(ctx context.Context, prID int64) ([]*types.PullReqReviewer, error)
	}

	// PullReqFileViewStore stores information about what file a user viewed.
	PullReqFileViewStore interface {
		// Upsert inserts or updates the latest viewed sha for a file in a PR.
		Upsert(ctx context.Context, fileView *types.PullReqFileView) error

		// DeleteByFileForPrincipal deletes the entry for the specified PR, principal, and file.
		DeleteByFileForPrincipal(ctx context.Context, prID int64, principalID int64, filePath string) error

		// MarkObsolete updates all entries of the files as obsolete for the PR.
		MarkObsolete(ctx context.Context, prID int64, filePaths []string) error

		// List lists all files marked as viewed by the user for the specified PR.
		List(ctx context.Context, prID int64, principalID int64) ([]*types.PullReqFileView, error)
	}

	// RuleStore defines database interface for protection rules.
	RuleStore interface {
		// Find finds a protection rule by ID.
		Find(ctx context.Context, id int64) (*types.Rule, error)

		// FindByIdentifier finds a protection rule by parent ID and identifier.
		FindByIdentifier(ctx context.Context, spaceID, repoID *int64, identifier string) (*types.Rule, error)

		// Create inserts a new protection rule.
		Create(ctx context.Context, rule *types.Rule) error

		// Update updates an existing protection rule.
		Update(ctx context.Context, rule *types.Rule) error

		// Delete removes a protection rule by its ID.
		Delete(ctx context.Context, id int64) error

		// DeleteByIdentifier removes a protection rule by its identifier.
		DeleteByIdentifier(ctx context.Context, spaceID, repoID *int64, identifier string) error

		// Count returns count of protection rules matching the provided criteria.
		Count(ctx context.Context, spaceID, repoID *int64, filter *types.RuleFilter) (int64, error)

		// List returns a list of protection rules of a repository or a space that matches the provided criteria.
		List(ctx context.Context, spaceID, repoID *int64, filter *types.RuleFilter) ([]types.Rule, error)

		// ListAllRepoRules returns a list of all protection rules that can be applied on a repository.
		ListAllRepoRules(ctx context.Context, repoID int64) ([]types.RuleInfoInternal, error)
	}

	// WebhookStore defines the webhook data storage.
	WebhookStore interface {
		// Find finds the webhook by id.
		Find(ctx context.Context, id int64) (*types.Webhook, error)

		// FindByIdentifier finds the webhook with the given Identifier for the given parent.
		FindByIdentifier(
			ctx context.Context,
			parentType enum.WebhookParent,
			parentID int64,
			identifier string,
		) (*types.Webhook, error)

		// Create creates a new webhook.
		Create(ctx context.Context, hook *types.Webhook) error

		// Update updates an existing webhook.
		Update(ctx context.Context, hook *types.Webhook) error

		// UpdateOptLock updates the webhook using the optimistic locking mechanism.
		UpdateOptLock(ctx context.Context, hook *types.Webhook,
			mutateFn func(hook *types.Webhook) error) (*types.Webhook, error)

		// Delete deletes the webhook for the given id.
		Delete(ctx context.Context, id int64) error

		// DeleteByIdentifier deletes the webhook with the given identifier for the given parent.
		DeleteByIdentifier(ctx context.Context, parentType enum.WebhookParent, parentID int64, identifier string) error

		// Count counts the webhooks for a given parent type and id.
		Count(ctx context.Context, parentType enum.WebhookParent, parentID int64,
			opts *types.WebhookFilter) (int64, error)

		// List lists the webhooks for a given parent type and id.
		List(ctx context.Context, parentType enum.WebhookParent, parentID int64,
			opts *types.WebhookFilter) ([]*types.Webhook, error)
	}

	// WebhookExecutionStore defines the webhook execution data storage.
	WebhookExecutionStore interface {
		// Find finds the webhook execution by id.
		Find(ctx context.Context, id int64) (*types.WebhookExecution, error)

		// Create creates a new webhook execution entry.
		Create(ctx context.Context, hook *types.WebhookExecution) error

		// DeleteOld removes all executions that are older than the provided time.
		DeleteOld(ctx context.Context, olderThan time.Time) (int64, error)

		// ListForWebhook lists the webhook executions for a given webhook id.
		ListForWebhook(ctx context.Context, webhookID int64,
			opts *types.WebhookExecutionFilter) ([]*types.WebhookExecution, error)

		// ListForTrigger lists the webhook executions for a given trigger id.
		ListForTrigger(ctx context.Context, triggerID string) ([]*types.WebhookExecution, error)
	}

	CheckStore interface {
		// FindByIdentifier returns status check result for given unique key.
		FindByIdentifier(ctx context.Context, repoID int64, commitSHA string, identifier string) (types.Check, error)

		// Upsert creates new or updates an existing status check result.
		Upsert(ctx context.Context, check *types.Check) error

		// Count counts status check results for a specific commit in a repo.
		Count(ctx context.Context, repoID int64, commitSHA string, opts types.CheckListOptions) (int, error)

		// List returns a list of status check results for a specific commit in a repo.
		List(ctx context.Context, repoID int64, commitSHA string, opts types.CheckListOptions) ([]types.Check, error)

		// ListRecent returns a list of recently executed status checks in a repository.
		ListRecent(ctx context.Context, repoID int64, opts types.CheckRecentOptions) ([]string, error)

		// ListResults returns a list of status check results for a specific commit in a repo.
		ListResults(ctx context.Context, repoID int64, commitSHA string) ([]types.CheckResult, error)
	}

	GitspaceConfigStore interface {
		// Find returns a gitspace config given a ID from the datastore.
		Find(ctx context.Context, id int64) (*types.GitspaceConfig, error)

		// FindByIdentifier returns a gitspace config with a given UID in a space
		FindByIdentifier(ctx context.Context, spaceID int64, identifier string) (*types.GitspaceConfig, error)

		// Create creates a new gitspace config in the datastore.
		Create(ctx context.Context, gitspaceConfig *types.GitspaceConfig) error

		// Update tries to update a gitspace config in the datastore with optimistic locking.
		Update(ctx context.Context, gitspaceConfig *types.GitspaceConfig) error

		// List lists the gitspace configs present in a parent space ID in the datastore.
		List(ctx context.Context, filter *types.GitspaceFilter) ([]*types.GitspaceConfig, error)

		// Count the number of gitspace configs in a space matching the given filter.
		Count(ctx context.Context, filter *types.GitspaceFilter) (int64, error)

		// ListAll lists all the gitspace configs present for a user in the given spaces in the datastore.
		ListAll(ctx context.Context, userUID string) ([]*types.GitspaceConfig, error)
	}

	GitspaceInstanceStore interface {
		// Find returns a gitspace instance given a gitspace instance ID from the datastore.
		Find(ctx context.Context, id int64) (*types.GitspaceInstance, error)

		// FindLatestByGitspaceConfigID returns the latest gitspace instance given a gitspace config ID from the datastore.
		FindLatestByGitspaceConfigID(
			ctx context.Context,
			gitspaceConfigID int64,
			spaceID int64,
		) (*types.GitspaceInstance, error)

		// Create creates a new gitspace instance in the datastore.
		Create(ctx context.Context, gitspaceInstance *types.GitspaceInstance) error

		// Update tries to update a gitspace instance in the datastore with optimistic locking.
		Update(ctx context.Context, gitspaceInstance *types.GitspaceInstance) error

		// List lists the gitspace instance present in a parent space ID in the datastore.
		List(ctx context.Context, filter *types.GitspaceFilter) ([]*types.GitspaceInstance, error)

		// List lists the latest gitspace instance present for the gitspace configs in the datastore.
		FindAllLatestByGitspaceConfigID(ctx context.Context, gitspaceConfigIDs []int64) ([]*types.GitspaceInstance, error)
	}

	InfraProviderConfigStore interface {
		// Find returns a infra provider config given a ID from the datastore.
		Find(ctx context.Context, id int64) (*types.InfraProviderConfig, error)

		// FindByIdentifier returns a infra provider config with a given UID in a space
		FindByIdentifier(ctx context.Context, spaceID int64, identifier string) (*types.InfraProviderConfig, error)

		// Create creates a new infra provider config in the datastore.
		Create(ctx context.Context, infraProviderConfig *types.InfraProviderConfig) error
	}

	InfraProviderResourceStore interface {
		// Find returns a Infra provider resource given a ID from the datastore.
		Find(ctx context.Context, id int64) (*types.InfraProviderResource, error)

		// FindByIdentifier returns a infra provider resource with a given UID in a space
		FindByIdentifier(ctx context.Context, spaceID int64, identifier string) (*types.InfraProviderResource, error)

		// Create creates a new infra provider resource in the datastore.
		Create(ctx context.Context, infraProviderResource *types.InfraProviderResource) error

		// List lists the infra provider resource present for the gitspace config in a parent space ID in the datastore.
		List(ctx context.Context,
			infraProviderConfigID int64,
			filter types.ListQueryFilter,
		) ([]*types.InfraProviderResource, error)

		// DeleteByIdentifier deletes the Infra provider resource with the given identifier for the given space.
		DeleteByIdentifier(ctx context.Context, spaceID int64, identifier string) error
	}

	PipelineStore interface {
		// Find returns a pipeline given a pipeline ID from the datastore.
		Find(ctx context.Context, id int64) (*types.Pipeline, error)

		// FindByIdentifier returns a pipeline with a given Identifier in a space
		FindByIdentifier(ctx context.Context, id int64, identifier string) (*types.Pipeline, error)

		// Create creates a new pipeline in the datastore.
		Create(ctx context.Context, pipeline *types.Pipeline) error

		// Update tries to update a pipeline in the datastore
		Update(ctx context.Context, pipeline *types.Pipeline) error

		// List lists the pipelines present in a repository in the datastore.
		List(ctx context.Context, repoID int64, pagination types.ListQueryFilter) ([]*types.Pipeline, error)

		// ListLatest lists the pipelines present in a repository in the datastore.
		// It also returns latest build information for all the returned entries.
		ListLatest(ctx context.Context, repoID int64, pagination types.ListQueryFilter) ([]*types.Pipeline, error)

		// UpdateOptLock updates the pipeline using the optimistic locking mechanism.
		UpdateOptLock(ctx context.Context, pipeline *types.Pipeline,
			mutateFn func(pipeline *types.Pipeline) error) (*types.Pipeline, error)

		// Delete deletes a pipeline ID from the datastore.
		Delete(ctx context.Context, id int64) error

		// Count the number of pipelines in a repository matching the given filter.
		Count(ctx context.Context, repoID int64, filter types.ListQueryFilter) (int64, error)

		// DeleteByIdentifier deletes a pipeline with a given identifier under a repo.
		DeleteByIdentifier(ctx context.Context, repoID int64, identifier string) error

		// IncrementSeqNum increments the sequence number of the pipeline
		IncrementSeqNum(ctx context.Context, pipeline *types.Pipeline) (*types.Pipeline, error)
	}

	SecretStore interface {
		// Find returns a secret given an ID
		Find(ctx context.Context, id int64) (*types.Secret, error)

		// FindByIdentifier returns a secret given a space ID and a identifier
		FindByIdentifier(ctx context.Context, spaceID int64, identifier string) (*types.Secret, error)

		// Create creates a new secret
		Create(ctx context.Context, secret *types.Secret) error

		// Count the number of secrets in a space matching the given filter.
		Count(ctx context.Context, spaceID int64, pagination types.ListQueryFilter) (int64, error)

		// UpdateOptLock updates the secret using the optimistic locking mechanism.
		UpdateOptLock(ctx context.Context, secret *types.Secret,
			mutateFn func(secret *types.Secret) error) (*types.Secret, error)

		// Update tries to update a secret.
		Update(ctx context.Context, secret *types.Secret) error

		// Delete deletes a secret given an ID.
		Delete(ctx context.Context, id int64) error

		// DeleteByIdentifier deletes a secret given a space ID and a identifier.
		DeleteByIdentifier(ctx context.Context, spaceID int64, identifier string) error

		// List lists the secrets in a given space.
		List(ctx context.Context, spaceID int64, filter types.ListQueryFilter) ([]*types.Secret, error)

		// ListAll lists all the secrets in a given space.
		ListAll(ctx context.Context, parentID int64) ([]*types.Secret, error)
	}

	ExecutionStore interface {
		// Find returns a execution given an execution ID.
		Find(ctx context.Context, id int64) (*types.Execution, error)

		// FindByNumber returns a execution given a pipeline and an execution number
		FindByNumber(ctx context.Context, pipelineID int64, num int64) (*types.Execution, error)

		// Create creates a new execution in the datastore.
		Create(ctx context.Context, execution *types.Execution) error

		// Update tries to update an execution.
		Update(ctx context.Context, execution *types.Execution) error

		// List lists the executions for a given pipeline ID
		List(ctx context.Context, pipelineID int64, pagination types.Pagination) ([]*types.Execution, error)

		// Delete deletes an execution given a pipeline ID and an execution number
		Delete(ctx context.Context, pipelineID int64, num int64) error

		// Count the number of executions in a space
		Count(ctx context.Context, parentID int64) (int64, error)
	}

	StageStore interface {
		// List returns a build stage list from the datastore
		// where the stage is incomplete (pending or running).
		ListIncomplete(ctx context.Context) ([]*types.Stage, error)

		// List returns a list of stages corresponding to an execution ID.
		List(ctx context.Context, executionID int64) ([]*types.Stage, error)

		// ListWithSteps returns a stage list from the datastore corresponding to an execution,
		// with the individual steps included.
		ListWithSteps(ctx context.Context, executionID int64) ([]*types.Stage, error)

		// Find returns a build stage from the datastore by ID.
		Find(ctx context.Context, stageID int64) (*types.Stage, error)

		// FindByNumber returns a stage from the datastore by number.
		FindByNumber(ctx context.Context, executionID int64, stageNum int) (*types.Stage, error)

		// Update tries to update a stage and returns an optimistic locking error if it was
		// unable to do so.
		Update(ctx context.Context, stage *types.Stage) error

		// Create creates a new stage.
		Create(ctx context.Context, stage *types.Stage) error
	}

	StepStore interface {
		// FindByNumber returns a step from the datastore by number.
		FindByNumber(ctx context.Context, stageID int64, stepNum int) (*types.Step, error)

		// Create creates a new step.
		Create(ctx context.Context, step *types.Step) error

		// Update tries to update a step and returns an optimistic locking error if it was
		// unable to do so.
		Update(ctx context.Context, e *types.Step) error
	}

	ConnectorStore interface {
		// Find returns a connector given an ID.
		Find(ctx context.Context, id int64) (*types.Connector, error)

		// FindByIdentifier returns a connector given a space ID and a identifier.
		FindByIdentifier(ctx context.Context, spaceID int64, identifier string) (*types.Connector, error)

		// Create creates a new connector.
		Create(ctx context.Context, connector *types.Connector) error

		// Count the number of connectors in a space matching the given filter.
		Count(ctx context.Context, spaceID int64, pagination types.ListQueryFilter) (int64, error)

		// UpdateOptLock updates the connector using the optimistic locking mechanism.
		UpdateOptLock(ctx context.Context, connector *types.Connector,
			mutateFn func(connector *types.Connector) error) (*types.Connector, error)

		// Update tries to update a connector.
		Update(ctx context.Context, connector *types.Connector) error

		// Delete deletes a connector given an ID.
		Delete(ctx context.Context, id int64) error

		// DeleteByIdentifier deletes a connector given a space ID and an identifier.
		DeleteByIdentifier(ctx context.Context, spaceID int64, identifier string) error

		// List lists the connectors in a given space.
		List(ctx context.Context, spaceID int64, filter types.ListQueryFilter) ([]*types.Connector, error)
	}

	TemplateStore interface {
		// Find returns a template given an ID.
		Find(ctx context.Context, id int64) (*types.Template, error)

		// FindByIdentifierAndType returns a template given a space ID, identifier and a type
		FindByIdentifierAndType(ctx context.Context, spaceID int64,
			identifier string, resolverType enum.ResolverType) (*types.Template, error)

		// Create creates a new template.
		Create(ctx context.Context, template *types.Template) error

		// Count the number of templates in a space matching the given filter.
		Count(ctx context.Context, spaceID int64, pagination types.ListQueryFilter) (int64, error)

		// UpdateOptLock updates the template using the optimistic locking mechanism.
		UpdateOptLock(ctx context.Context, template *types.Template,
			mutateFn func(template *types.Template) error) (*types.Template, error)

		// Update tries to update a template.
		Update(ctx context.Context, template *types.Template) error

		// Delete deletes a template given an ID.
		Delete(ctx context.Context, id int64) error

		// DeleteByIdentifierAndType deletes a template given a space ID, identifier and a type.
		DeleteByIdentifierAndType(ctx context.Context, spaceID int64, identifier string, resolverType enum.ResolverType) error

		// List lists the templates in a given space.
		List(ctx context.Context, spaceID int64, filter types.ListQueryFilter) ([]*types.Template, error)
	}

	TriggerStore interface {
		// FindByIdentifier returns a trigger given a pipeline and a trigger identifier.
		FindByIdentifier(ctx context.Context, pipelineID int64, identifier string) (*types.Trigger, error)

		// Create creates a new trigger in the datastore.
		Create(ctx context.Context, trigger *types.Trigger) error

		// Update tries to update an trigger.
		Update(ctx context.Context, trigger *types.Trigger) error

		// UpdateOptLock updates the trigger using the optimistic locking mechanism.
		UpdateOptLock(ctx context.Context, trigger *types.Trigger,
			mutateFn func(trigger *types.Trigger) error) (*types.Trigger, error)

		// List lists the triggers for a given pipeline ID.
		List(ctx context.Context, pipelineID int64, filter types.ListQueryFilter) ([]*types.Trigger, error)

		// Delete deletes an trigger given a pipeline ID and a trigger identifier.
		DeleteByIdentifier(ctx context.Context, pipelineID int64, identifier string) error

		// Count the number of triggers in a pipeline.
		Count(ctx context.Context, pipelineID int64, filter types.ListQueryFilter) (int64, error)

		// ListAllEnabled lists all enabled triggers for a given repo without pagination.
		// It's used only internally to trigger builds.
		ListAllEnabled(ctx context.Context, repoID int64) ([]*types.Trigger, error)
	}

	PluginStore interface {
		// List returns back the list of plugins matching the given filter
		// along with their associated schemas.
		List(ctx context.Context, filter types.ListQueryFilter) ([]*types.Plugin, error)

		// ListAll returns back the full list of plugins.
		ListAll(ctx context.Context) ([]*types.Plugin, error)

		// Create creates a new entry in the plugin datastore.
		Create(ctx context.Context, plugin *types.Plugin) error

		// Update tries to update an trigger.
		Update(ctx context.Context, plugin *types.Plugin) error

		// Count counts the number of plugins matching the given filter.
		Count(ctx context.Context, filter types.ListQueryFilter) (int64, error)

		// Find returns a plugin given a name and a version.
		Find(ctx context.Context, name, version string) (*types.Plugin, error)
	}

	UserGroupStore interface {
		// FindByIdentifier returns a types.UserGroup given a space ID and identifier.
		FindByIdentifier(ctx context.Context, spaceID int64, identifier string) (*types.UserGroup, error)
	}

	PublicKeyStore interface {
		// Find returns a public key given an ID.
		Find(ctx context.Context, id int64) (*types.PublicKey, error)

		// FindByIdentifier returns a public key given a principal ID and an identifier.
		FindByIdentifier(ctx context.Context, principalID int64, identifier string) (*types.PublicKey, error)

		// Create creates a new public key.
		Create(ctx context.Context, publicKey *types.PublicKey) error

		// DeleteByIdentifier deletes a public key.
		DeleteByIdentifier(ctx context.Context, principalID int64, identifier string) error

		// MarkAsVerified updates the public key to mark it as verified.
		MarkAsVerified(ctx context.Context, id int64, verified int64) error

		// Count returns the number of public keys for the principal that match provided the filter.
		Count(ctx context.Context, principalID int64, filter *types.PublicKeyFilter) (int, error)

		// List returns the public keys for the principal that match provided the filter.
		List(ctx context.Context, principalID int64, filter *types.PublicKeyFilter) ([]types.PublicKey, error)

		// ListByFingerprint returns public keys given a fingerprint and key usage.
		ListByFingerprint(ctx context.Context, fingerprint string) ([]types.PublicKey, error)
	}

	GitspaceEventStore interface {
		// Create creates a new record for the given gitspace event.
		Create(ctx context.Context, gitspaceEvent *types.GitspaceEvent) error

		// List returns all events and count for the given query filter.
		List(ctx context.Context, filter *types.GitspaceEventFilter) ([]*types.GitspaceEvent, int, error)

		// FindLatestByTypeAndGitspaceConfigID returns the latest gitspace event for the given config ID and event type
		// where the entity type is gitspace config.
		FindLatestByTypeAndGitspaceConfigID(
			ctx context.Context,
			eventType enum.GitspaceEventType,
			gitspaceConfigID int64,
		) (*types.GitspaceEvent, error)
	}

	LabelStore interface {
		// Define defines a label.
		Define(ctx context.Context, lbl *types.Label) error

		// Update updates a label.
		Update(ctx context.Context, lbl *types.Label) error

		// Find finds a label defined in a specified space/repo with a specified key.
		Find(
			ctx context.Context,
			spaceID, repoID *int64,
			key string,
		) (*types.Label, error)

		// Delete deletes a label defined in a specified space/repo with a specified key.
		Delete(ctx context.Context, spaceID, repoID *int64, key string) error

		// List list labels defined in a specified space/repo.
		List(
			ctx context.Context,
			spaceID, repoID *int64,
			filter *types.LabelFilter,
		) ([]*types.Label, error)

		// FindByID finds label with a specified id.
		FindByID(ctx context.Context, id int64) (*types.Label, error)

		// ListInScopes lists labels defined in specified repo/spaces.
		ListInScopes(
			ctx context.Context,
			repoID int64,
			spaceIDs []int64,
			filter *types.LabelFilter,
		) ([]*types.Label, error)

		// ListInfosInScopes lists label infos defined in specified repo/spaces.
		ListInfosInScopes(
			ctx context.Context,
			repoID int64,
			spaceIDs []int64,
			filter *types.AssignableLabelFilter,
		) ([]*types.LabelInfo, error)

		// IncrementValueCount increments count of values defined for a specified label.
		IncrementValueCount(ctx context.Context, labelID int64, increment int) (int64, error)

		// CountInSpace counts the number of labels defined in a specified space.
		CountInSpace(ctx context.Context, spaceID int64, filter *types.LabelFilter) (int64, error)

		// CountInRepo counts the number of labels defined in a specified repository.
		CountInRepo(ctx context.Context, repoID int64, filter *types.LabelFilter) (int64, error)

		// CountInScopes counts the number of labels defined in specified repo/spaces.
		CountInScopes(
			ctx context.Context,
			repoID int64,
			spaceIDs []int64,
			filter *types.LabelFilter,
		) (int64, error)
	}

	LabelValueStore interface {
		// Define defines a label value.
		Define(ctx context.Context, lbl *types.LabelValue) error

		// Update updates a label value.
		Update(ctx context.Context, lblVal *types.LabelValue) error

		// Delete deletes a label value associated with a specified label.
		Delete(ctx context.Context, labelID int64, value string) error

		// Delete deletes specified label values associated with a specified label.
		DeleteMany(ctx context.Context, labelID int64, values []string) error

		// FindByLabelID finds a label value defined for a specified label.
		FindByLabelID(
			ctx context.Context,
			labelID int64,
			value string,
		) (*types.LabelValue, error)

		// List lists label values defined for a specified label.
		List(
			ctx context.Context,
			labelID int64,
			opts *types.ListQueryFilter,
		) ([]*types.LabelValue, error)

		// FindByID finds label value with a specified id.
		FindByID(ctx context.Context, id int64) (*types.LabelValue, error)

		// ListInfosByLabelIDs list label infos by a specified label id.
		ListInfosByLabelIDs(
			ctx context.Context,
			labelIDs []int64,
		) (map[int64][]*types.LabelValueInfo, error)
	}

	PullReqLabelAssignmentStore interface {
		// Assign assigns a label to a pullreq.
		Assign(ctx context.Context, label *types.PullReqLabel) error

		// Unassign removes a label from a pullreq with a specified id.
		Unassign(ctx context.Context, pullreqID int64, labelID int64) error

		// ListAssigned list labels assigned to a specified pullreq.
		ListAssigned(
			ctx context.Context,
			pullreqID int64,
		) (map[int64]*types.LabelAssignment, error)

		// Find finds a label assigned to a pullreq with a specified id.
		FindByLabelID(
			ctx context.Context,
			pullreqID int64,
			labelID int64,
		) (*types.PullReqLabel, error)

		// FindValueByLabelID finds a value assigned to a pullreq label.
		FindValueByLabelID(ctx context.Context, labelID int64) (*types.LabelValue, error)

		// ListAssignedByPullreqIDs list labels assigned to specified pullreqs.
		ListAssignedByPullreqIDs(
			ctx context.Context,
			pullreqIDs []int64,
		) (map[int64][]*types.LabelPullReqAssignmentInfo, error)
	}

	InfraProviderTemplateStore interface {
		FindByIdentifier(ctx context.Context, spaceID int64, identifier string) (*types.InfraProviderTemplate, error)
		Find(ctx context.Context, id int64) (*types.InfraProviderTemplate, error)
		Create(ctx context.Context, infraProviderTemplate *types.InfraProviderTemplate) error
		Delete(ctx context.Context, id int64) error
	}

	InfraProvisionedStore interface {
		Find(ctx context.Context, id int64) (*types.InfraProvisioned, error)
		FindAllLatestByGateway(ctx context.Context, gatewayHost string) ([]*types.InfraProvisionedGatewayView, error)
		FindLatestByGitspaceInstanceID(
			ctx context.Context,
			spaceID int64,
			gitspaceInstanceID int64,
		) (*types.InfraProvisioned, error)
		FindLatestByGitspaceInstanceIdentifier(ctx context.Context,
			spaceID int64,
			gitspaceInstanceIdentifier string,
		) (*types.InfraProvisioned, error)
		Create(ctx context.Context, infraProvisioned *types.InfraProvisioned) error
		Delete(ctx context.Context, id int64) error
		Update(ctx context.Context, infraProvisioned *types.InfraProvisioned) error
	}
)
