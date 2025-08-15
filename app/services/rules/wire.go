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

	"github.com/google/wire"
)

// WireSet provides a wire set for this package.
var WireSet = wire.NewSet(
	ProvideService,
)

func ProvideService(
	tx dbtx.Transactor,
	ruleStore store.RuleStore,
	repoStore store.RepoStore,
	spaceStore store.SpaceStore,
	protectionManager *protection.Manager,
	auditService audit.Service,
	instrumentation instrument.Service,
	principalInfoCache store.PrincipalInfoCache,
	userGroupStore store.UserGroupStore,
	userGroupService usergroup.Service,
	eventReporter *ruleevents.Reporter,
	sseStreamer sse.Streamer,
	ruleValidator Validator,
	repoIDCache store.RepoIDCache,
) *Service {
	return NewService(
		tx,
		ruleStore,
		repoStore,
		spaceStore,
		protectionManager,
		auditService,
		instrumentation,
		principalInfoCache,
		userGroupStore,
		userGroupService,
		eventReporter,
		sseStreamer,
		ruleValidator,
		repoIDCache,
	)
}

func ProvideValidator() Validator {
	return validator{}
}
