// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package importer

import (
	"github.com/harness/gitness/encrypt"
	"github.com/harness/gitness/gitrpc"
	"github.com/harness/gitness/internal/services/job"
	"github.com/harness/gitness/internal/sse"
	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/internal/url"
	"github.com/harness/gitness/types"
	"github.com/jmoiron/sqlx"

	"github.com/google/wire"
)

var WireSet = wire.NewSet(
	ProvideRepoImporter,
)

func ProvideRepoImporter(
	config *types.Config,
	urlProvider *url.Provider,
	git gitrpc.Interface,
	db *sqlx.DB,
	repoStore store.RepoStore,
	pipelineStore store.PipelineStore,
	triggerStore store.TriggerStore,
	encrypter encrypt.Encrypter,
	scheduler *job.Scheduler,
	executor *job.Executor,
	sseStreamer sse.Streamer,
) (*Repository, error) {
	importer := &Repository{
		defaultBranch: config.Git.DefaultBranch,
		urlProvider:   urlProvider,
		git:           git,
		db:            db,
		repoStore:     repoStore,
		pipelineStore: pipelineStore,
		triggerStore:  triggerStore,
		encrypter:     encrypter,
		scheduler:     scheduler,
		sseStreamer:   sseStreamer,
	}

	err := executor.Register(jobType, importer)
	if err != nil {
		return nil, err
	}

	return importer, nil
}
