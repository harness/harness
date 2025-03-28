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

package rules

import (
	ruleevents "github.com/harness/gitness/app/events/rule"
	"github.com/harness/gitness/app/services/instrument"
	"github.com/harness/gitness/app/services/protection"
	"github.com/harness/gitness/app/services/usergroup"
	"github.com/harness/gitness/app/sse"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/audit"
	"github.com/harness/gitness/store/database/dbtx"
)

// Service is responsible for processing rules.
type Service struct {
	tx dbtx.Transactor

	ruleStore          store.RuleStore
	repoStore          store.RepoStore
	spaceStore         store.SpaceStore
	protectionManager  *protection.Manager
	auditService       audit.Service
	instrumentation    instrument.Service
	principalInfoCache store.PrincipalInfoCache
	userGroupStore     store.UserGroupStore
	userGroupService   usergroup.SearchService
	eventReporter      *ruleevents.Reporter

	sseStreamer sse.Streamer
}

func NewService(
	tx dbtx.Transactor,
	ruleStore store.RuleStore,
	repoStore store.RepoStore,
	spaceStore store.SpaceStore,
	protectionManager *protection.Manager,
	auditService audit.Service,
	instrumentation instrument.Service,
	principalInfoCache store.PrincipalInfoCache,
	userGroupStore store.UserGroupStore,
	userGroupService usergroup.SearchService,
	eventReporter *ruleevents.Reporter,
	sseStreamer sse.Streamer,
) *Service {
	return &Service{
		tx:                 tx,
		ruleStore:          ruleStore,
		repoStore:          repoStore,
		spaceStore:         spaceStore,
		protectionManager:  protectionManager,
		auditService:       auditService,
		instrumentation:    instrumentation,
		principalInfoCache: principalInfoCache,
		userGroupStore:     userGroupStore,
		userGroupService:   userGroupService,
		eventReporter:      eventReporter,
		sseStreamer:        sseStreamer,
	}
}
