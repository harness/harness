// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package system

import (
	"context"

	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/types"
)

type Controller struct {
	principalStore store.PrincipalStore
	config         *types.Config
}

func NewController(principalStore store.PrincipalStore, config *types.Config) *Controller {
	return &Controller{
		principalStore: principalStore,
		config:         config,
	}
}

func (c *Controller) IsUserRegistrationAllowed(ctx context.Context) (bool, error) {
	usrCount, err := c.principalStore.CountUsers(ctx, &types.UserFilter{})
	if err != nil {
		return false, err
	}

	return usrCount == 0 || c.config.AllowSignUp, nil
}
