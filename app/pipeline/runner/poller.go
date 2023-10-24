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

	"github.com/harness/gitness/app/pipeline/logger"

	"github.com/drone-runners/drone-runner-docker/engine/resource"
	runtime2 "github.com/drone-runners/drone-runner-docker/engine2/runtime"
	"github.com/drone/drone-go/drone"
	runnerclient "github.com/drone/runner-go/client"
	"github.com/drone/runner-go/poller"
	"github.com/rs/zerolog/log"
)

func NewExecutionPoller(
	runner *runtime2.Runner,
	client runnerclient.Client,
) *poller.Poller {
	runWithRecovery := func(ctx context.Context, stage *drone.Stage) (err error) {
		ctx = logger.WithUnwrappedZerolog(ctx)
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("panic received: %s", debug.Stack())
			}

			// the caller of this method (poller.Poller) discards the error - log it here
			if err != nil {
				log.Ctx(ctx).Error().Err(err).Msgf("An error occurred while calling runner.Run in Poller")
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
