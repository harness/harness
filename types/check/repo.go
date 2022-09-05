// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package check

import (
	"errors"
	"fmt"
	"regexp"

	"github.com/harness/gitness/types"
)

const (
	minRepoNameLength = 1
	maxRepoNameLength = 64
	repoNameRegex     = "^[a-z][a-z0-9\\-\\_]*$"

	minRepoDisplayNameLength = 1
	maxRepoDisplayNameLength = 256
	repoDisplayNameRegex     = "^[a-zA-Z][a-zA-Z0-9\\-\\_ ]*$"
)

var (
	ErrRepoNameLength = errors.New(fmt.Sprintf("Repository name has to be between %d and %d in length.", minRepoNameLength, maxRepoNameLength))
	ErrRepoNameRegex  = errors.New("Repository name has start with a letter and only contain the following [a-z0-9-_].")

	ErrRepoDisplayNameLength = errors.New(fmt.Sprintf("Repository name has to be between %d and %d in length.", minRepoDisplayNameLength, maxRepoDisplayNameLength))
	ErrRepoDisplayNameRegex  = errors.New("Repository display name has start with a letter and only contain the following [a-zA-Z0-9-_ ].")
)

// Repo returns true if the Repo if valid.
func Repo(repo *types.Repository) (bool, error) {
	l := len(repo.Name)
	if l < minRepoNameLength || l > maxRepoNameLength {
		return false, ErrRepoNameLength
	}

	if ok, _ := regexp.Match(repoNameRegex, []byte(repo.Name)); !ok {
		return false, ErrRepoNameRegex
	}

	l = len(repo.DisplayName)
	if l < minRepoDisplayNameLength || l > maxRepoDisplayNameLength {
		return false, ErrRepoDisplayNameLength
	}

	if ok, _ := regexp.Match(repoDisplayNameRegex, []byte(repo.DisplayName)); !ok {
		return false, ErrRepoDisplayNameRegex
	}

	return true, nil
}
