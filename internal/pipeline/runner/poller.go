// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package runner

import (
	"context"
	"fmt"
	"runtime/debug"

	"github.com/harness/gitness/types"

	"github.com/drone-runners/drone-runner-docker/engine/resource"
	"github.com/drone/drone-go/drone"
	runnerclient "github.com/drone/runner-go/client"
	"github.com/drone/runner-go/pipeline/runtime"
	"github.com/drone/runner-go/poller"
)

func NewExecutionPoller(
	runner *runtime.Runner,
	config *types.Config,
	client runnerclient.Client,
) *poller.Poller {
	// taking the cautious approach of recovering in case of panics
	runWithRecovery := func(ctx context.Context, stage *drone.Stage) (err error) {
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("panic received while executing run: %s", debug.Stack())
			}
		}()
		return runner.Run(ctx, stage)
	}

	return &poller.Poller{
		Client:   client,
		Dispatch: runWithRecovery,
		Filter: &runnerclient.Filter{
			Kind: resource.Kind,
			Type: resource.Type,
			// TODO: Check if other parameters are needed.
		},
	}
}
