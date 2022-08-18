// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

// Package store defines the data storage interfaces.
package store

import (
	"context"

	"github.com/harness/scm/types"
)

type (
	// ExecutionStore defines execution data storage.
	ExecutionStore interface {
		// Find finds the execution by id.
		Find(ctx context.Context, id int64) (*types.Execution, error)

		// FindSlug finds the execution by pipeline id and slug.
		FindSlug(ctx context.Context, id int64, slug string) (*types.Execution, error)

		// List returns a list of executions by pipeline id.
		List(ctx context.Context, id int64, params types.Params) ([]*types.Execution, error)

		// Create saves the execution details.
		Create(ctx context.Context, execution *types.Execution) error

		// Update updates the execution details.
		Update(ctx context.Context, execution *types.Execution) error

		// Delete deletes the execution.
		Delete(ctx context.Context, execution *types.Execution) error
	}

	// PipelineStore defines pipeline data storage.
	PipelineStore interface {
		// Find finds the pipeline by id.
		Find(ctx context.Context, id int64) (*types.Pipeline, error)

		// FindToken finds the pipeline by token.
		FindToken(ctx context.Context, token string) (*types.Pipeline, error)

		// FindSlug finds the user unique name.
		FindSlug(ctx context.Context, key string) (*types.Pipeline, error)

		// List returns a list of pipelines by user.
		List(ctx context.Context, user int64, params types.Params) ([]*types.Pipeline, error)

		// Create saves the pipeline details.
		Create(ctx context.Context, pipeline *types.Pipeline) error

		// Update updates the pipeline details.
		Update(ctx context.Context, pipeline *types.Pipeline) error

		// Delete deletes the pipeline.
		Delete(ctx context.Context, pipeline *types.Pipeline) error
	}

	// UserStore defines user data storage.
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

	// SystemStore defines insternal system metadata storage.
	SystemStore interface {
		// Config returns the system configuration.
		Config(ctx context.Context) *types.Config
	}
)
