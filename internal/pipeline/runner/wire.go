// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package runner

import (
	"github.com/harness/gitness/internal/pipeline/manager"
	"github.com/harness/gitness/types"

	runnerclient "github.com/drone/runner-go/client"
	"github.com/drone/runner-go/pipeline/runtime"
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
	manager manager.ExecutionManager,
) (*runtime.Runner, error) {
	return NewExecutionRunner(config, client, manager)
}

// ProvideExecutionPoller provides a poller which can poll the manager
// for new builds and execute them.
func ProvideExecutionPoller(
	runner *runtime.Runner,
	config *types.Config,
	client runnerclient.Client,
) *poller.Poller {
	return NewExecutionPoller(runner, config, client)
}
