// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package user

import (
	"github.com/harness/gitness/internal/auth/authz"
	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/types/check"
)

type Controller struct {
	userCheck  check.User
	authorizer authz.Authorizer
	userStore  store.UserStore
	tokenStore store.TokenStore
}

func NewController(userCheck check.User, authorizer authz.Authorizer, userStore store.UserStore,
	tokenStore store.TokenStore) *Controller {
	return &Controller{
		userCheck:  userCheck,
		authorizer: authorizer,
		userStore:  userStore,
		tokenStore: tokenStore,
	}
}
