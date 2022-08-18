// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package client

import "github.com/harness/scm/types"

// Client to access the remote APIs.
type Client interface {
	// Login authenticates the user and returns a JWT token.
	Login(username, password string) (*types.Token, error)

	// Register registers a new  user and returns a JWT token.
	Register(username, password string) (*types.Token, error)

	// Self returns the currently authenticated user.
	Self() (*types.User, error)

	// Token returns an oauth2 bearer token for the currently
	// authenticated user.
	Token() (*types.Token, error)

	// User returns a user by ID or email.
	User(key string) (*types.User, error)

	// UserList returns a list of all registered users.
	UserList(params types.Params) ([]*types.User, error)

	// UserCreate creates a new user account.
	UserCreate(user *types.User) (*types.User, error)

	// UserUpdate updates a user account by ID or email.
	UserUpdate(key string, input *types.UserInput) (*types.User, error)

	// UserDelete deletes a user account by ID or email.
	UserDelete(key string) error

	// Pipeline returns a pipeline by slug.
	Pipeline(slug string) (*types.Pipeline, error)

	// PipelineList returns a list of all pipelines.
	PipelineList(params types.Params) ([]*types.Pipeline, error)

	// PipelineCreate creates a new pipeline.
	PipelineCreate(user *types.Pipeline) (*types.Pipeline, error)

	// PipelineUpdate updates a pipeline.
	PipelineUpdate(slug string, input *types.PipelineInput) (*types.Pipeline, error)

	// PipelineDelete deletes a pipeline.
	PipelineDelete(slug string) error

	// Execution returns a execution by pipeline and slug.
	Execution(pipeline, slug string) (*types.Execution, error)

	// ExecutionList returns a list of all executions by pipeline slug.
	ExecutionList(pipeline string, params types.Params) ([]*types.Execution, error)

	// ExecutionCreate creates a new execution.
	ExecutionCreate(pipeline string, execution *types.Execution) (*types.Execution, error)

	// ExecutionUpdate updates a execution.
	ExecutionUpdate(pipeline, slug string, input *types.ExecutionInput) (*types.Execution, error)

	// ExecutionDelete deletes a execution.
	ExecutionDelete(pipeline, slug string) error
}

// remoteError store the error payload returned
// fro the remote API.
type remoteError struct {
	Message string `json:"message"`
}

// Error returns the error message.
func (e *remoteError) Error() string {
	return e.Message
}
