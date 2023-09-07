// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package user

import (
	"context"
	"fmt"

	"github.com/harness/gitness/internal/api/controller/system"
	"github.com/harness/gitness/internal/api/usererror"
	"github.com/harness/gitness/internal/token"
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

	return &types.TokenResponse{Token: *token, AccessToken: jwtToken}, nil
}
