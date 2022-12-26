// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

// Package store defines the data storage interfaces.
package store

import (
	"context"

	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

type (
	// UserStore defines the user data storage.
	UserStore interface {
		// Find finds the user by id.
		Find(ctx context.Context, id int64) (*types.User, error)

		// FindUID finds the user by uid.
		FindUID(ctx context.Context, uid string) (*types.User, error)

		// FindEmail finds the user by email.
		FindEmail(ctx context.Context, email string) (*types.User, error)

		// Create saves the user details.
		Create(ctx context.Context, user *types.User) error

		// Update updates the user details.
		Update(ctx context.Context, user *types.User) error

		// Delete deletes the user.
		Delete(ctx context.Context, id int64) error

		// List returns a list of users.
		List(ctx context.Context, params *types.UserFilter) ([]*types.User, error)

		// Count returns a count of users.
		Count(ctx context.Context) (int64, error)
	}

	// ServiceAccountStore defines the service account data storage.
	ServiceAccountStore interface {
		// Find finds the service account by id.
		Find(ctx context.Context, id int64) (*types.ServiceAccount, error)

		// FindUID finds the service account by uid.
		FindUID(ctx context.Context, uid string) (*types.ServiceAccount, error)

		// Create saves the service account.
		Create(ctx context.Context, sa *types.ServiceAccount) error

		// Update updates the service account details.
		Update(ctx context.Context, sa *types.ServiceAccount) error

		// Delete deletes the service account.
		Delete(ctx context.Context, id int64) error

		// List returns a list of service accounts for a specific parent.
		List(ctx context.Context, parentType enum.ParentResourceType, parentID int64) ([]*types.ServiceAccount, error)

		// Count returns a count of service accounts for a specific parent.
		Count(ctx context.Context, parentType enum.ParentResourceType, parentID int64) (int64, error)
	}

	// ServiceStore defines the service data storage.
	ServiceStore interface {
		// Find finds the service by id.
		Find(ctx context.Context, id int64) (*types.Service, error)

		// FindUID finds the service by uid.
		FindUID(ctx context.Context, uid string) (*types.Service, error)

		// Create saves the service.
		Create(ctx context.Context, sa *types.Service) error

		// Update updates the service details.
		Update(ctx context.Context, sa *types.Service) error

		// Delete deletes the service.
		Delete(ctx context.Context, id int64) error

		// List returns a list of all services.
		List(ctx context.Context) ([]*types.Service, error)

		// Count returns a count of all services.
		Count(ctx context.Context) (int64, error)
	}

	// SpaceStore defines the space data storage.
	SpaceStore interface {
		// Find the space by id.
		Find(ctx context.Context, id int64) (*types.Space, error)

		// FindByPath the space by its path.
		FindByPath(ctx context.Context, path string) (*types.Space, error)

		// FindSpaceFromRef finds space by path or ref
		FindSpaceFromRef(ctx context.Context, spaceRef string) (*types.Space, error)

		// Create creates a new space
		Create(ctx context.Context, space *types.Space) error

		// Move moves an existing space.
		Move(ctx context.Context, principalID int64, id int64, newParentID int64, newName string,
			keepAsAlias bool) (*types.Space, error)

		// Update updates the space details.
		Update(ctx context.Context, space *types.Space) error

		// Delete deletes the space.
		Delete(ctx context.Context, id int64) error

		// Count the child spaces of a space.
		Count(ctx context.Context, id int64, opts *types.SpaceFilter) (int64, error)

		// List returns a list of child spaces in a space.
		List(ctx context.Context, id int64, opts *types.SpaceFilter) ([]*types.Space, error)

		// CountPaths returns a count of all paths of a space.
		CountPaths(ctx context.Context, id int64, opts *types.PathFilter) (int64, error)

		// ListPaths returns a list of all paths of a space.
		ListPaths(ctx context.Context, id int64, opts *types.PathFilter) ([]*types.Path, error)

		// CreatePath create an alias for a space
		CreatePath(ctx context.Context, id int64, params *types.PathParams) (*types.Path, error)

		// DeletePath delete an alias of a space
		DeletePath(ctx context.Context, id int64, pathID int64) error
	}

	// RepoStore defines the repository data storage.
	RepoStore interface {
		// Find the repo by id.
		Find(ctx context.Context, id int64) (*types.Repository, error)

		// FindByPath the repo by path.
		FindByPath(ctx context.Context, path string) (*types.Repository, error)

		// FindRepoFromRef finds the repo by path or ref.
		FindRepoFromRef(ctx context.Context, repoRef string) (*types.Repository, error)

		// Create a new repo.
		Create(ctx context.Context, repo *types.Repository) error

		// Move moves an existing repo.
		Move(ctx context.Context, principalID int64, repoID int64, newParentID int64, newName string,
			keepAsAlias bool) (*types.Repository, error)

		// Update the repo details.
		Update(ctx context.Context, repo *types.Repository) error

		// Delete the repo.
		Delete(ctx context.Context, id int64) error

		// Count of repos in a space.
		Count(ctx context.Context, parentID int64, opts *types.RepoFilter) (int64, error)

		// List returns a list of repos in a space.
		List(ctx context.Context, parentID int64, opts *types.RepoFilter) ([]*types.Repository, error)

		// CountPaths returns a count of all paths of a repo.
		CountPaths(ctx context.Context, id int64, opts *types.PathFilter) (int64, error)

		// ListPaths returns a list of all paths of a repo.
		ListPaths(ctx context.Context, id int64, opts *types.PathFilter) ([]*types.Path, error)

		// CreatePath an alias for a repo
		CreatePath(ctx context.Context, repoID int64, params *types.PathParams) (*types.Path, error)

		// DeletePath delete an alias of a repo
		DeletePath(ctx context.Context, repoID int64, pathID int64) error
	}

	// TokenStore defines the token data storage.
	TokenStore interface {
		// Find finds the token by id
		Find(ctx context.Context, id int64) (*types.Token, error)

		// Find finds the token by principalId and tokenUID
		FindByUID(ctx context.Context, principalID int64, tokenUID string) (*types.Token, error)

		// Create saves the token details.
		Create(ctx context.Context, token *types.Token) error

		// Delete deletes the token with the given id.
		Delete(ctx context.Context, id int64) error

		// DeleteForPrincipal deletes all tokens for a specific principal
		DeleteForPrincipal(ctx context.Context, principalID int64) error

		// List returns a list of tokens of a specific type for a specific principal.
		List(ctx context.Context, principalID int64, tokenType enum.TokenType) ([]*types.Token, error)

		// Count returns a count of tokens of a specifc type for a specific principal.
		Count(ctx context.Context, principalID int64, tokenType enum.TokenType) (int64, error)
	}

	// PullReqStore defines the pull request data storage.
	PullReqStore interface {
		// Find the pull request by id.
		Find(ctx context.Context, id int64) (*types.PullReq, error)

		// FindByNumber finds the pull request by repo ID and the pull request number.
		FindByNumber(ctx context.Context, repoID, number int64) (*types.PullReq, error)

		// Create a new pull request.
		Create(ctx context.Context, pullreq *types.PullReq) error

		// Update the pull request. It will set new values to the Version and Updated fields.
		Update(ctx context.Context, repo *types.PullReq) error

		// UpdateActivitySeq the pull request's activity sequence number.
		// It will set new values to the ActivitySeq, Version and Updated fields.
		UpdateActivitySeq(ctx context.Context, pr *types.PullReq) (*types.PullReq, error)

		// Delete the pull request.
		Delete(ctx context.Context, id int64) error

		// LastNumber returns the last pull request number for a repo.
		LastNumber(ctx context.Context, repoID int64) (int64, error)

		// Count of pull requests in a space.
		Count(ctx context.Context, repoID int64, opts *types.PullReqFilter) (int64, error)

		// List returns a list of pull requests in a space.
		List(ctx context.Context, repoID int64, opts *types.PullReqFilter) ([]*types.PullReq, error)
	}

	PullReqActivityStore interface {
		// Find the pull request activity by id.
		Find(ctx context.Context, id int64) (*types.PullReqActivity, error)

		// Create a new pull request activity. Value of the Order field should be fetched with UpdateActivitySeq.
		// Value of the SubOrder field (for replies) should be fetched with UpdateReplySeq (non-replies have 0).
		Create(ctx context.Context, act *types.PullReqActivity) error

		// Update the pull request activity. It will set new values to the Version and Updated fields.
		Update(ctx context.Context, act *types.PullReqActivity) error

		// UpdateReplySeq the pull request activity's reply sequence number.
		// It will set new values to the ReplySeq, Version and Updated fields.
		UpdateReplySeq(ctx context.Context, act *types.PullReqActivity) (*types.PullReqActivity, error)

		// Count returns number of pull request activities in a pull request.
		Count(ctx context.Context, prID int64, opts *types.PullReqActivityFilter) (int64, error)

		// List returns a list of pull request activities in a pull request (a timeline).
		List(ctx context.Context, prID int64, opts *types.PullReqActivityFilter) ([]*types.PullReqActivity, error)
	}

	// SystemStore defines internal system metadata storage.
	SystemStore interface {
		// Config returns the system configuration.
		Config(ctx context.Context) *types.Config
	}
)
