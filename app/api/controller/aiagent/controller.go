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

package aiagent

import (
	"github.com/harness/gitness/app/auth/authz"
	"github.com/harness/gitness/app/services/aiagent"
	"github.com/harness/gitness/app/services/messaging"
	"github.com/harness/gitness/app/services/refcache"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/app/url"
	"github.com/harness/gitness/git"
)

type Controller struct {
	authorizer     authz.Authorizer
	intelligence   aiagent.Intelligence
	repoFinder     refcache.RepoFinder
	pipelineStore  store.PipelineStore
	executionStore store.ExecutionStore
	git            git.Interface
	urlProvider    url.Provider
	slackbot       *messaging.Slack
}

func NewController(
	authorizer authz.Authorizer,
	intelligence aiagent.Intelligence,
	repoFinder refcache.RepoFinder,
	pipelineStore store.PipelineStore,
	executionStore store.ExecutionStore,
	git git.Interface,
	urlProvider url.Provider,
	slackbot *messaging.Slack,
) *Controller {
	return &Controller{
		authorizer:     authorizer,
		intelligence:   intelligence,
		repoFinder:     repoFinder,
		pipelineStore:  pipelineStore,
		executionStore: executionStore,
		git:            git,
		urlProvider:    urlProvider,
		slackbot:       slackbot,
	}
}
