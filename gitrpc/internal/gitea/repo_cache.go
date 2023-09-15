// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package gitea

import (
	"context"

	"github.com/harness/gitness/cache"

	gogit "github.com/go-git/go-git/v5"
)

func NewRepoCache() cache.Cache[string, *RepoEntryValue] {
	return cache.NewNoCache[string, *RepoEntryValue](repoGetter{})
}

type repoGetter struct{}

type RepoEntryValue gogit.Repository

func (repo *RepoEntryValue) Repo() *gogit.Repository {
	return (*gogit.Repository)(repo)
}

func (r repoGetter) Find(_ context.Context, path string) (*RepoEntryValue, error) {
	repo, err := gogit.PlainOpen(path)
	if err != nil {
		return nil, err
	}

	return (*RepoEntryValue)(repo), nil
}
