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

	"github.com/harness/gitness/app/auth/authz"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/check"
	"github.com/harness/gitness/types/enum"

	"golang.org/x/crypto/bcrypt"
)

type Controller struct {
	tx                dbtx.Transactor
	principalUIDCheck check.PrincipalUID
	authorizer        authz.Authorizer
	principalStore    store.PrincipalStore
	tokenStore        store.TokenStore
	membershipStore   store.MembershipStore
	publicKeyStore    store.PublicKeyStore
}

func NewController(
	tx dbtx.Transactor,
	principalUIDCheck check.PrincipalUID,
	authorizer authz.Authorizer,
	principalStore store.PrincipalStore,
	tokenStore store.TokenStore,
	membershipStore store.MembershipStore,
	publicKeyStore store.PublicKeyStore,
) *Controller {
	return &Controller{
		tx:                tx,
		principalUIDCheck: principalUIDCheck,
		authorizer:        authorizer,
		principalStore:    principalStore,
		tokenStore:        tokenStore,
		membershipStore:   membershipStore,
		publicKeyStore:    publicKeyStore,
	}
}

var hashPassword = bcrypt.GenerateFromPassword

func findUserFromUID(ctx context.Context,
	principalStore store.PrincipalStore, userUID string,
) (*types.User, error) {
	return principalStore.FindUserByUID(ctx, userUID)
}

func findUserFromEmail(ctx context.Context,
	principalStore store.PrincipalStore, email string,
) (*types.User, error) {
	return principalStore.FindUserByEmail(ctx, email)
}

func isUserTokenType(tokenType enum.TokenType) bool {
	return tokenType == enum.TokenTypePAT || tokenType == enum.TokenTypeSession
}
