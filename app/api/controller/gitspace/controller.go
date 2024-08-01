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

package gitspace

import (
	"github.com/harness/gitness/app/auth/authz"
	gitspaceevents "github.com/harness/gitness/app/events/gitspace"
	"github.com/harness/gitness/app/gitspace/logutil"
	"github.com/harness/gitness/app/gitspace/orchestrator"
	"github.com/harness/gitness/app/gitspace/scm"
	"github.com/harness/gitness/app/services/gitspace"
	"github.com/harness/gitness/app/services/infraprovider"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/store/database/dbtx"
)

type Controller struct {
	authorizer            authz.Authorizer
	infraProviderSvc      *infraprovider.Service
	gitspaceConfigStore   store.GitspaceConfigStore
	gitspaceInstanceStore store.GitspaceInstanceStore
	spaceStore            store.SpaceStore
	eventReporter         *gitspaceevents.Reporter
	orchestrator          orchestrator.Orchestrator
	gitspaceEventStore    store.GitspaceEventStore
	tx                    dbtx.Transactor
	statefulLogger        *logutil.StatefulLogger
	scm                   scm.SCM
	repoStore             store.RepoStore
	gitspaceSvc           *gitspace.Service
}

func NewController(
	tx dbtx.Transactor,
	authorizer authz.Authorizer,
	infraProviderSvc *infraprovider.Service,
	gitspaceConfigStore store.GitspaceConfigStore,
	gitspaceInstanceStore store.GitspaceInstanceStore,
	spaceStore store.SpaceStore,
	eventReporter *gitspaceevents.Reporter,
	orchestrator orchestrator.Orchestrator,
	gitspaceEventStore store.GitspaceEventStore,
	statefulLogger *logutil.StatefulLogger,
	scm scm.SCM,
	repoStore store.RepoStore,
	gitspaceSvc *gitspace.Service,
) *Controller {
	return &Controller{
		tx:                    tx,
		authorizer:            authorizer,
		infraProviderSvc:      infraProviderSvc,
		gitspaceConfigStore:   gitspaceConfigStore,
		gitspaceInstanceStore: gitspaceInstanceStore,
		spaceStore:            spaceStore,
		eventReporter:         eventReporter,
		orchestrator:          orchestrator,
		gitspaceEventStore:    gitspaceEventStore,
		statefulLogger:        statefulLogger,
		scm:                   scm,
		repoStore:             repoStore,
		gitspaceSvc:           gitspaceSvc,
	}
}
