// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package check

import (
	"fmt"

	"github.com/harness/gitness/types"
)

var (
	RepositoryRequiresSpaceIdError = fmt.Errorf("SpaceId required - Repositories don't exist outside of a space.")
)

// Repo checks the provided repository and returns an error in it isn't valid.
func Repo(repo *types.Repository) error {
	// validate name
	if err := Name(repo.Name); err != nil {
		return err
	}

	// validate display name
	if err := DisplayName(repo.DisplayName); err != nil {
		return err
	}

	// validate repo within a space
	if repo.SpaceId <= 0 {
		return RepositoryRequiresSpaceIdError
	}

	return nil
}
