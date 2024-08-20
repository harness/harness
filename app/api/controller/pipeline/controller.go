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

package pipeline

import (
	"github.com/harness/gitness/app/auth/authz"
	events "github.com/harness/gitness/app/events/pipeline"
	"github.com/harness/gitness/app/store"
)

type Controller struct {
	defaultBranch string
	repoStore     store.RepoStore
	triggerStore  store.TriggerStore
	authorizer    authz.Authorizer
	pipelineStore store.PipelineStore
	reporter      events.Reporter
}

func NewController(
	authorizer authz.Authorizer,
	repoStore store.RepoStore,
	triggerStore store.TriggerStore,
	pipelineStore store.PipelineStore,
	reporter events.Reporter,
) *Controller {
	return &Controller{
		repoStore:     repoStore,
		triggerStore:  triggerStore,
		authorizer:    authorizer,
		pipelineStore: pipelineStore,
		reporter:      reporter,
	}
}
