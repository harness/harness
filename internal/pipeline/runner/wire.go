// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package runner

import (
	"github.com/harness/gitness/internal/pipeline/manager"
	"github.com/harness/gitness/internal/pipeline/plugin"
	"github.com/harness/gitness/types"

	runtime2 "github.com/drone-runners/drone-runner-docker/engine2/runtime"
	runnerclient "github.com/drone/runner-go/client"
	"github.com/drone/runner-go/poller"
	"github.com/google/wire"
)

// WireSet provides a wire set for this package.
var WireSet = wire.NewSet(
	ProvideExecutionRunner,
	ProvideExecutionPoller,
)

// ProvideExecutionRunner provides an execution runner.
func ProvideExecutionRunner(
	config *types.Config,
	client runnerclient.Client,
	pluginManager *plugin.PluginManager,
	manager manager.ExecutionManager,
) (*runtime2.Runner, error) {
	return NewExecutionRunner(config, client, pluginManager, manager)
}

// ProvideExecutionPoller provides a poller which can poll the manager
// for new builds and execute them.
func ProvideExecutionPoller(
	runner *runtime2.Runner,
	config *types.Config,
	client runnerclient.Client,
) *poller.Poller {
	return NewExecutionPoller(runner, config, client)
}
