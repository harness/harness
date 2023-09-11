// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package importer

import (
	"github.com/harness/gitness/gitrpc"
	"github.com/harness/gitness/internal/services/job"
	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/internal/url"
	"github.com/harness/gitness/types"

	"github.com/google/wire"
)

var WireSet = wire.NewSet(
	ProvideRepoImporter,
)

func ProvideRepoImporter(
	config *types.Config,
	urlProvider *url.Provider,
	git gitrpc.Interface,
	repoStore store.RepoStore,
	scheduler *job.Scheduler,
	executor *job.Executor,
) (*Repository, error) {
	importer := &Repository{
		defaultBranch: config.Git.DefaultBranch,
		urlProvider:   urlProvider,
		git:           git,
		repoStore:     repoStore,
		scheduler:     scheduler,
	}

	err := executor.Register(jobType, importer)
	if err != nil {
		return nil, err
	}

	return importer, nil
}
