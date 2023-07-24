// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package system

import (
	"context"

	"github.com/harness/gitness/types"
)

// RegisterCheck checks the DB and env config flag to return boolean
// which represents if a user sign-up is allowed or not.
func (c *Controller) RegisterCheck(ctx context.Context, config *types.Config) (bool, error) {
	check, err := IsUserRegistrationAllowed(ctx, c.principalStore, config.AllowSignUp)
	if err != nil {
		return check, err
	}

	return check, nil
}
