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

package triage

import (
	"context"

	"github.com/harness/gitness/app/services/aigateway"
	"github.com/harness/gitness/app/services/insights"
	"github.com/harness/gitness/app/services/publicaccess"
	"github.com/harness/gitness/app/services/refcache"
	"github.com/harness/gitness/app/services/settings"
	"github.com/harness/gitness/app/services/task"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/git"

	"github.com/google/wire"
)

// WireSet provides a wire set for this package.
var WireSet = wire.NewSet(
	ProvideObserver,
)

// ProvideObserver creates the triage observer, registers it on the insights service and starts
// its workers. The decider is the AI gateway; swap GatewayDecider for another Decider to change it.
func ProvideObserver(
	ctx context.Context,
	insightsSvc *insights.Service,
	gateway *aigateway.Gateway,
	issues store.InsightsIssueStore,
	events store.InsightsEventStore,
	repoFinder refcache.RepoFinder,
	settingsSvc *settings.Service,
	publicAccess publicaccess.Service,
	gitInterface git.Interface,
	taskSvc *task.Service,
) *Observer {
	o := New(Config{}, Deps{
		Issues:   issues,
		Events:   events,
		Repos:    repoFinder,
		Modes:    aigateway.NewModeResolver(settingsSvc, publicAccess),
		Suspects: GitSuspectFinder{Git: gitInterface},
		Decider:  GatewayDecider{Gateway: gateway},
		Tasks:    taskSvc,
	})
	o.Start(ctx)
	insightsSvc.SetObserver(o)
	return o
}
