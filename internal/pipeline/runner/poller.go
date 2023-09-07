// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package runner

import (
	"github.com/harness/gitness/types"

	"github.com/drone-runners/drone-runner-docker/engine/resource"
	runnerclient "github.com/drone/runner-go/client"
	"github.com/drone/runner-go/pipeline/runtime"
	"github.com/drone/runner-go/poller"
)

func NewExecutionPoller(
	runner *runtime.Runner,
	config *types.Config,
	client runnerclient.Client,
) *poller.Poller {
	return &poller.Poller{
		Client:   client,
		Dispatch: runner.Run,
		Filter: &runnerclient.Filter{
			Kind: resource.Kind,
			Type: resource.Type,
			// TODO: Check if other parameters are needed.
		},
	}
}
