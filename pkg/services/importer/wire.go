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
	"github.com/harness/gitness/encrypt"
	"github.com/harness/gitness/gitrpc"
	"github.com/harness/gitness/pkg/services/job"
	"github.com/harness/gitness/pkg/sse"
	"github.com/harness/gitness/pkg/store"
	"github.com/harness/gitness/pkg/url"
	"github.com/harness/gitness/types"

	"github.com/google/wire"
	"github.com/jmoiron/sqlx"
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
