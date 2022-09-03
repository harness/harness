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

		// List returns a list of users.
		List(ctx context.Context, params types.UserFilter) ([]*types.User, error)

		// Create saves the user details.
		Create(ctx context.Context, user *types.User) error

		// Update updates the user details.
		Update(ctx context.Context, user *types.User) error

		// Delete deletes the user.
		Delete(ctx context.Context, user *types.User) error

		// Count returns a count of users.
		Count(ctx context.Context) (int64, error)
	}

	// SpaceStore defines the space data storage.
	SpaceStore interface {
		// Finds the space by id.
		Find(ctx context.Context, id int64) (*types.Space, error)

		// Finds the space by the full qualified space name.
		FindFqsn(ctx context.Context, fqsn string) (*types.Space, error)

		// List returns a list of child spaces in a space.
		List(ctx context.Context, id int64, opts types.SpaceFilter) ([]*types.Space, error)

		// Creates a new space
		Create(ctx context.Context, space *types.Space) error

		// Updates the space details.
		Update(ctx context.Context, space *types.Space) error

		// Deletes the space.
		Delete(ctx context.Context, id int64) error

		// Count the child spaces of a space.
		Count(ctx context.Context, id int64) (int64, error)
	}

	// SystemStore defines internal system metadata storage.
	SystemStore interface {
		// Config returns the system configuration.
		Config(ctx context.Context) *types.Config
	}
)
