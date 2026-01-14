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

package user

import (
	"context"
	"fmt"

	"github.com/harness/gitness/app/api/controller/system"
	"github.com/harness/gitness/app/api/usererror"
	userevents "github.com/harness/gitness/app/events/user"
	"github.com/harness/gitness/app/token"
	"github.com/harness/gitness/types"
)

type RegisterInput struct {
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
	UID         string `json:"uid"`
	Password    string `json:"password"`
}

// Register creates a new user and returns a new session token on success.
// This doesn't require auth, but has limited functionalities (unable to create admin user for example).
func (c *Controller) Register(ctx context.Context, sysCtrl *system.Controller,
	in *RegisterInput) (*types.TokenResponse, error) {
	signUpAllowed, err := sysCtrl.IsUserSignupAllowed(ctx)
	if err != nil {
		return nil, err
	}

	if !signUpAllowed {
		return nil, usererror.Forbidden("User sign-up is disabled")
	}

	user, err := c.CreateNoAuth(ctx, &CreateInput{
		UID:         in.UID,
		Email:       in.Email,
		DisplayName: in.DisplayName,
		Password:    in.Password,
	}, false)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// TODO: how should we name session tokens?
	token, jwtToken, err := token.CreateUserSession(ctx, c.tokenStore, user, "register")
	if err != nil {
		return nil, fmt.Errorf("failed to create token after successful user creation: %w", err)
	}

	c.eventReporter.Registered(ctx, &userevents.RegisteredPayload{
		Base: userevents.Base{PrincipalID: user.ID},
	})

	return &types.TokenResponse{Token: *token, AccessToken: jwtToken}, nil
}
