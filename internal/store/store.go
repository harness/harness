// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

// Package store defines the data storage interfaces.
package store

import (
	"context"

	"github.com/harness/gitness/types"
)

type (
	// UserStore defines the user data storage.
	UserStore interface {
		// Find finds the user by id.
		Find(ctx context.Context, id int64) (*types.User, error)

		// FindEmail finds the user by email.
		FindEmail(ctx context.Context, email string) (*types.User, error)

		// FindKey finds the user by unique key (email or id).
		FindKey(ctx context.Context, key string) (*types.User, error)

		// Create saves the user details.
		Create(ctx context.Context, user *types.User) error

		// Update updates the user details.
		Update(ctx context.Context, user *types.User) error

		// Delete deletes the user.
		Delete(ctx context.Context, user *types.User) error

		// List returns a list of users.
		List(ctx context.Context, params *types.UserFilter) ([]*types.User, error)

		// Count returns a count of users.
		Count(ctx context.Context) (int64, error)
	}

	// SpaceStore defines the space data storage.
	SpaceStore interface {
		// Find the space by id.
		Find(ctx context.Context, id int64) (*types.Space, error)

		// FindByPath the space by its path.
		FindByPath(ctx context.Context, path string) (*types.Space, error)

		// Create creates a new space
		Create(ctx context.Context, space *types.Space) error

		// Move moves an existing space.
		Move(ctx context.Context, userID int64, spaceID int64, newParentID int64, newName string,
			keepAsAlias bool) (*types.Space, error)

		// Update updates the space details.
		Update(ctx context.Context, space *types.Space) error

		// Delete deletes the space.
		Delete(ctx context.Context, id int64) error

		// Count the child spaces of a space.
		Count(ctx context.Context, id int64) (int64, error)

		// List returns a list of child spaces in a space.
		List(ctx context.Context, id int64, opts *types.SpaceFilter) ([]*types.Space, error)

		// ListAllPaths returns a list of all paths of a space.
		ListAllPaths(ctx context.Context, id int64, opts *types.PathFilter) ([]*types.Path, error)

		// CreatePath create an alias for a space
		CreatePath(ctx context.Context, spaceID int64, params *types.PathParams) (*types.Path, error)

		// DeletePath delete an alias of a space
		DeletePath(ctx context.Context, spaceID int64, pathID int64) error
	}

	// RepoStore defines the repository data storage.
	RepoStore interface {
		// Find the repo by id.
		Find(ctx context.Context, id int64) (*types.Repository, error)

		// FindByPath the repo by path.
		FindByPath(ctx context.Context, path string) (*types.Repository, error)

		// Create a new repo
		Create(ctx context.Context, repo *types.Repository) error

		// Move moves an existing repo.
		Move(ctx context.Context, userID int64, repoID int64, newSpaceID int64, newName string,
			keepAsAlias bool) (*types.Repository, error)

		// Update the repo details.
		Update(ctx context.Context, repo *types.Repository) error

		// Delete the repo.
		Delete(ctx context.Context, id int64) error

		// Count of repos in a space.
		Count(ctx context.Context, spaceID int64) (int64, error)

		// List returns a list of repos in a space.
		List(ctx context.Context, spaceID int64, opts *types.RepoFilter) ([]*types.Repository, error)

		// ListAllPaths returns a list of all alias paths of a repo.
		ListAllPaths(ctx context.Context, id int64, opts *types.PathFilter) ([]*types.Path, error)

		// CreatePath an alias for a repo
		CreatePath(ctx context.Context, repoID int64, params *types.PathParams) (*types.Path, error)

		// DeletePath delete an alias of a repo
		DeletePath(ctx context.Context, repoID int64, pathID int64) error
	}

	// SystemStore defines internal system metadata storage.
	SystemStore interface {
		// Config returns the system configuration.
		Config(ctx context.Context) *types.Config
	}
)
