// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package service

import (
	"github.com/harness/gitness/internal/auth/authz"
	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/types/check"
)

type Controller struct {
	serviceCheck check.Service
	authorizer   authz.Authorizer
	serviceStore store.ServiceStore
}

func NewController(serviceCheck check.Service, authorizer authz.Authorizer,
	serviceStore store.ServiceStore) *Controller {
	return &Controller{
		serviceCheck: serviceCheck,
		authorizer:   authorizer,
		serviceStore: serviceStore,
	}
}
