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

package agentlauncher

import (
	"github.com/harness/gitness/app/bootstrap"
	aitaskevents "github.com/harness/gitness/app/events/aitask"
	"github.com/harness/gitness/app/gitspace/agentmodel"
	"github.com/harness/gitness/app/services/agentguard"
	"github.com/harness/gitness/app/services/agentpolicy"
	"github.com/harness/gitness/app/services/aidecide"
	"github.com/harness/gitness/app/services/aigateway"
	"github.com/harness/gitness/app/services/aitaskevent"
	"github.com/harness/gitness/app/services/insights/triage"
	"github.com/harness/gitness/app/services/refcache"
	"github.com/harness/gitness/app/services/task"
	"github.com/harness/gitness/app/services/taskassign"
	"github.com/harness/gitness/app/store"

	"github.com/google/wire"
)

// WireSet provides a wire set for this package.
var WireSet = wire.NewSet(
	ProvideLauncher,
)

// ProvideLauncher creates the launcher and hooks it into the Insights triage (autofix candidates) and into
// the AI task event service (finished AI tasks).
func ProvideLauncher(
	policy *agentpolicy.Evaluator,
	gateway *aigateway.Gateway,
	guard *agentguard.Guard,
	taskSvc *task.Service,
	taskStore store.TaskStore,
	repoFinder refcache.RepoFinder,
	aiTaskStore store.AITaskStore,
	reporter *aitaskevents.Reporter,
	gitspaceConfigs store.GitspaceConfigStore,
	registry *agentmodel.Registry,
	decide *aidecide.Service,
	assign *taskassign.Service,
	triageObserver *triage.Observer,
	aiTaskEvents *aitaskevent.Service,
) *Launcher {
	l := New(Deps{
		Policy:     policy,
		Models:     gateway,
		Guard:      guard,
		Tasks:      taskSvc,
		TaskFinder: taskStore,
		Repos:      repoFinder,
		AITasks:    aiTaskStore,
		Events:     reporter,
		Gitspaces:  StoreGitspaceResolver{Configs: gitspaceConfigs},
		Registry:   registry,
		Decider:    decide,
		Assigner:   assign,
		Redactor:   gateway,
		Principal:  bootstrap.GetAgentServicePrincipal,
	})
	triageObserver.SetAutofixObserver(l)
	aiTaskEvents.SetFinishObserver(l)
	return l
}
