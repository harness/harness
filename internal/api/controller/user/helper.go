// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package user

import (
	"context"

	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
	"golang.org/x/crypto/bcrypt"
)

var hashPassword = bcrypt.GenerateFromPassword

func findUserFromUID(ctx context.Context,
	userStore store.UserStore, userUID string) (*types.User, error) {
	return userStore.FindUID(ctx, userUID)
}
func findUserFromEmail(ctx context.Context,
	userStore store.UserStore, email string) (*types.User, error) {
	return userStore.FindEmail(ctx, email)
}

func isUserTokenType(tokenType enum.TokenType) bool {
	return tokenType == enum.TokenTypePAT || tokenType == enum.TokenTypeSession
}
