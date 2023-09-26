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

package runner

import (
	"context"
	"fmt"
	"runtime/debug"

	"github.com/harness/gitness/types"

	"github.com/drone-runners/drone-runner-docker/engine/resource"
	runtime2 "github.com/drone-runners/drone-runner-docker/engine2/runtime"
	"github.com/drone/drone-go/drone"
	runnerclient "github.com/drone/runner-go/client"
	"github.com/drone/runner-go/poller"
)

func NewExecutionPoller(
	runner *runtime2.Runner,
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
