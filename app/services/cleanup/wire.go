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

package cleanup

import (
	"github.com/harness/gitness/app/api/controller/repo"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/job"

	"github.com/google/wire"
)

var WireSet = wire.NewSet(
	ProvideService,
)

func ProvideService(
	config Config,
	scheduler *job.Scheduler,
	executor *job.Executor,
	webhookExecutionStore store.WebhookExecutionStore,
	tokenStore store.TokenStore,
	repoStore store.RepoStore,
	repoCtrl *repo.Controller,
) (*Service, error) {
	return NewService(
		config,
		scheduler,
		executor,
		webhookExecutionStore,
		tokenStore,
		repoStore,
		repoCtrl,
	)
}
