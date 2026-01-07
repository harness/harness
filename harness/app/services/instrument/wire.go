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

package instrument

import (
	"context"

	gitevents "github.com/harness/gitness/app/events/git"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/events"
	"github.com/harness/gitness/job"
	"github.com/harness/gitness/types"

	"github.com/google/wire"
)

var WireSet = wire.NewSet(
	ProvideService,
	ProvideRepositoryCount,
	ProvideGitConsumer,
)

func ProvideGitConsumer(
	ctx context.Context,
	config *types.Config,
	gitReaderFactory *events.ReaderFactory[*gitevents.Reader],
	repoStore store.RepoStore,
	principalInfoCache store.PrincipalInfoCache,
	instrumentation Service,
) (Consumer, error) {
	return NewConsumer(
		ctx,
		config,
		gitReaderFactory,
		repoStore,
		principalInfoCache,
		instrumentation,
	)
}

func ProvideRepositoryCount(
	ctx context.Context,
	config *types.Config,
	svc Service,
	repoStore store.RepoStore,
	scheduler *job.Scheduler,
	executor *job.Executor,
) (*RepositoryCount, error) {
	return NewRepositoryCount(
		ctx,
		config,
		svc,
		repoStore,
		scheduler,
		executor,
	)
}

func ProvideService() Service {
	return Noop{}
}
