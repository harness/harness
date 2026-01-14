// Copyright 2023 Harness, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package importer

import (
	repoevents "github.com/harness/gitness/app/events/repo"
	"github.com/harness/gitness/app/services/keywordsearch"
	"github.com/harness/gitness/app/services/publicaccess"
	"github.com/harness/gitness/app/services/refcache"
	"github.com/harness/gitness/app/services/settings"
	"github.com/harness/gitness/app/sse"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/app/url"
	"github.com/harness/gitness/audit"
	"github.com/harness/gitness/encrypt"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/job"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types"

	"github.com/google/wire"
)

var WireSet = wire.NewSet(
	ProvideRepoImporter,
	ProvideReferenceSync,
)

func ProvideRepoImporter(
	config *types.Config,
	urlProvider url.Provider,
	git git.Interface,
	tx dbtx.Transactor,
	repoStore store.RepoStore,
	pipelineStore store.PipelineStore,
	triggerStore store.TriggerStore,
	repoFinder refcache.RepoFinder,
	encrypter encrypt.Encrypter,
	scheduler *job.Scheduler,
	executor *job.Executor,
	sseStreamer sse.Streamer,
	indexer keywordsearch.Indexer,
	publicAccess publicaccess.Service,
	eventReporter *repoevents.Reporter,
	auditService audit.Service,
	settings *settings.Service,
) (*Repository, error) {
	importer := &Repository{
		defaultBranch: config.Git.DefaultBranch,
		urlProvider:   urlProvider,
		git:           git,
		tx:            tx,
		repoStore:     repoStore,
		pipelineStore: pipelineStore,
		triggerStore:  triggerStore,
		repoFinder:    repoFinder,
		encrypter:     encrypter,
		scheduler:     scheduler,
		sseStreamer:   sseStreamer,
		indexer:       indexer,
		publicAccess:  publicAccess,
		eventReporter: eventReporter,
		auditService:  auditService,
		settings:      settings,
	}

	err := executor.Register(jobType, importer)
	if err != nil {
		return nil, err
	}

	return importer, nil
}

func ProvideReferenceSync(
	config *types.Config,
	urlProvider url.Provider,
	git git.Interface,
	repoStore store.RepoStore,
	repoFinder refcache.RepoFinder,
	scheduler *job.Scheduler,
	executor *job.Executor,
	indexer keywordsearch.Indexer,
	eventReporter *repoevents.Reporter,
) (*ReferenceSync, error) {
	importer := &ReferenceSync{
		defaultBranch: config.Git.DefaultBranch,
		urlProvider:   urlProvider,
		git:           git,
		repoStore:     repoStore,
		repoFinder:    repoFinder,
		scheduler:     scheduler,
		indexer:       indexer,
		eventReporter: eventReporter,
	}

	err := executor.Register(refSyncJobType, importer)
	if err != nil {
		return nil, err
	}

	return importer, nil
}
