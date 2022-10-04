// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package user

import (
	"github.com/harness/gitness/internal/auth/authz"
	"github.com/harness/gitness/internal/store"
)

type Controller struct {
	authorizer authz.Authorizer
	userStore  store.UserStore
	tokenStore store.TokenStore
}

func NewController(authorizer authz.Authorizer, userStore store.UserStore,
	tokenStore store.TokenStore) *Controller {
	return &Controller{
		authorizer: authorizer,
		userStore:  userStore,
		tokenStore: tokenStore,
	}
}
