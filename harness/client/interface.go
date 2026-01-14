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

package client

import (
	"context"

	"github.com/harness/gitness/app/api/controller/user"
	"github.com/harness/gitness/types"
)

// Client to access the remote APIs.
type Client interface {
	// Login authenticates the user and returns a JWT token.
	Login(ctx context.Context, input *user.LoginInput) (*types.TokenResponse, error)

	// Register registers a new  user and returns a JWT token.
	Register(ctx context.Context, input *user.RegisterInput) (*types.TokenResponse, error)

	// Self returns the currently authenticated user.
	Self(ctx context.Context) (*types.User, error)

	// User returns a user by ID or email.
	User(ctx context.Context, key string) (*types.User, error)

	// UserList returns a list of all registered users.
	UserList(ctx context.Context, params types.UserFilter) ([]types.User, error)

	// UserCreate creates a new user account.
	UserCreate(ctx context.Context, user *types.User) (*types.User, error)

	// UserUpdate updates a user account by ID or email.
	UserUpdate(ctx context.Context, key string, input *types.UserInput) (*types.User, error)

	// UserDelete deletes a user account by ID or email.
	UserDelete(ctx context.Context, key string) error

	// UserCreatePAT creates a new PAT for the user.
	UserCreatePAT(ctx context.Context, in user.CreateTokenInput) (*types.TokenResponse, error)
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
