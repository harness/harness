// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package service

import (
	"github.com/harness/gitness/internal/auth/authz"
	"github.com/harness/gitness/internal/store"
)

type Controller struct {
	authorizer   authz.Authorizer
	serviceStore store.ServiceStore
}

func NewController(authorizer authz.Authorizer, serviceStore store.ServiceStore) *Controller {
	return &Controller{
		authorizer:   authorizer,
		serviceStore: serviceStore,
	}
}
