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

package gitspace

import (
	"context"
	"fmt"
	"time"

	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

func (c *Service) CleanupGitspace(ctx context.Context, config types.GitspaceConfig) error {
	if config.GitspaceInstance.State == enum.GitSpaceInstanceStateCleaning &&
		time.Since(time.UnixMilli(config.GitspaceInstance.Updated)).Milliseconds() <=
			(gitspaceInstanceCleaningTimedOutMins*60*1000) {
		log.Ctx(ctx).Warn().Msgf("gitspace cleaning is already pending for : %q",
			config.GitspaceInstance.Identifier)
		return fmt.Errorf("gitspace is already pending for : %q", config.GitspaceInstance.Identifier)
	}

	config.GitspaceInstance.State = enum.GitSpaceInstanceStateCleaning

	err := c.UpdateInstance(ctx, config.GitspaceInstance)
	if err != nil {
		log.Ctx(ctx).Err(err).Msgf("failed to update instance %s before triggering cleanup",
			config.GitspaceInstance.Identifier)
		return fmt.Errorf("failed to update instance %s before triggering cleanup: %w",
			config.GitspaceInstance.Identifier,
			err,
		)
	}

	err = c.orchestrator.TriggerCleanupInstanceResources(ctx, config)
	if err != nil {
		log.Ctx(ctx).Err(err).Msgf("error during triggering cleanup for gitspace instance %s",
			config.GitspaceInstance.Identifier)

		config.GitspaceInstance.State = enum.GitspaceInstanceStateError
		if updateErr := c.UpdateInstance(ctx, config.GitspaceInstance); updateErr != nil {
			log.Ctx(ctx).Err(updateErr).Msgf("failed to update instance %s after error in triggering delete",
				config.GitspaceInstance.Identifier)
		}

		return fmt.Errorf("failed to trigger cleanup for gitspace instance %s: %w",
			config.GitspaceInstance.Identifier,
			err,
		)
	}

	log.Ctx(ctx).Debug().Msgf("successfully triggered cleanup for gitspace instance %s",
		config.GitspaceInstance.Identifier)

	return nil
}
