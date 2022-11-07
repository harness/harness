// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package check

import (
	"github.com/harness/gitness/types"
)

var (
	ErrRepositoryRequiresParentID = &ValidationError{
		"ParentID required - Standalone repositories are not supported.",
	}
)

// Repo returns true if the Repo is valid.
type Repo func(*types.Repository) error

// RepoDefault is the default Repo validation.
func RepoDefault(repo *types.Repository) error {
	// validate UID
	if err := UID(repo.UID); err != nil {
		return err
	}

	// validate the rest
	return RepoNoUID(repo)
}

// RepoNoUID validates the repo and ignores the UID field.
func RepoNoUID(repo *types.Repository) error {
	// validate repo within a space
	if repo.ParentID <= 0 {
		return ErrRepositoryRequiresParentID
	}

	// TODO: validate defaultBranch, ...

	return nil
}
