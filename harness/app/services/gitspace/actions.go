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
	"net/http"
	"strings"
	"time"

	"github.com/harness/gitness/app/api/usererror"
	events "github.com/harness/gitness/app/events/gitspace"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	gonanoid "github.com/matoous/go-nanoid"
	"github.com/rs/zerolog/log"
)

const defaultPasswordRef = "harness_password"
const defaultMachineUser = "harness"
const AllowedUIDAlphabet = "abcdefghijklmnopqrstuvwxyz0123456789"

// gitspaceInstanceCleaningTimedOutMins is timeout for which a gitspace instance can be in cleaning state.
const gitspaceInstanceCleaningTimedOutMins = 10

func (c *Service) gitspaceBusyOperation(
	ctx context.Context,
	config types.GitspaceConfig,
) error {
	if config.GitspaceInstance == nil || !config.GitspaceInstance.State.IsBusyStatus() {
		return nil
	}

	busyStateTimeoutInMillis := int64(c.config.Gitspace.BusyActionInMins * 60 * 1000)
	if time.Since(time.UnixMilli(config.GitspaceInstance.Updated)).Milliseconds() <= busyStateTimeoutInMillis {
		return usererror.NewWithPayload(http.StatusForbidden, fmt.Sprintf(
			"Last session for this gitspace is still %s", config.GitspaceInstance.State))
	}
	config.GitspaceInstance.State = enum.GitspaceInstanceStateError
	if err := c.UpdateInstance(ctx, config.GitspaceInstance); err != nil {
		return fmt.Errorf("failed to update gitspace config for %s: %w", config.Identifier, err)
	}
	return nil
}

func (c *Service) submitAsyncOps(
	ctx context.Context,
	config types.GitspaceConfig,
	action enum.GitspaceActionType,
) {
	switch action {
	case enum.GitspaceActionTypeStart:
		config.GitspaceInstance.State = enum.GitspaceInstanceStateStarting
	case enum.GitspaceActionTypeStop:
		config.GitspaceInstance.State = enum.GitspaceInstanceStateStopping
	case enum.GitspaceActionTypeReset:
		config.GitspaceInstance.State = enum.GitSpaceInstanceStateResetting
	}
	if updateErr := c.UpdateInstance(ctx, config.GitspaceInstance); updateErr != nil {
		log.Err(updateErr).Msgf(
			"failed to update gitspace instance during exec %s", config.GitspaceInstance.Identifier)
	}
	errChannel := make(chan *types.GitspaceError)
	submitCtx := context.WithoutCancel(ctx)
	gitspaceTimedOutInMins := time.Duration(c.config.Gitspace.InfraTimeoutInMins) * time.Minute
	ttlExecuteContext, cancel := context.WithTimeout(submitCtx, gitspaceTimedOutInMins)

	go c.triggerOrchestrator(ttlExecuteContext, config, action, errChannel)
	var err *types.GitspaceError
	go func() {
		select {
		case <-ttlExecuteContext.Done():
			if ttlExecuteContext.Err() != nil {
				err = &types.GitspaceError{
					Error: ttlExecuteContext.Err(),
				}
			}
		case err = <-errChannel:
		}
		if err != nil {
			log.Err(err.Error).Msgf("error during async execution for %s", config.GitspaceInstance.Identifier)
			config.GitspaceInstance.State = enum.GitspaceInstanceStateError
			config.GitspaceInstance.ErrorMessage = err.ErrorMessage
			updateErr := c.UpdateInstance(submitCtx, config.GitspaceInstance)
			if updateErr != nil {
				log.Err(updateErr).Msgf(
					"failed to update gitspace instance during exec %q", config.GitspaceInstance.Identifier)
			}

			switch action {
			case enum.GitspaceActionTypeStart:
				c.EmitGitspaceConfigEvent(submitCtx, config, enum.GitspaceEventTypeGitspaceActionStartFailed)
			case enum.GitspaceActionTypeStop:
				c.EmitGitspaceConfigEvent(submitCtx, config, enum.GitspaceEventTypeGitspaceActionStopFailed)
			case enum.GitspaceActionTypeReset:
				c.EmitGitspaceConfigEvent(submitCtx, config, enum.GitspaceEventTypeGitspaceActionResetFailed)
			}
		}

		cancel()
	}()
}

func (c *Service) triggerOrchestrator(
	ctxWithTimedOut context.Context,
	config types.GitspaceConfig,
	action enum.GitspaceActionType,
	errChannel chan *types.GitspaceError,
) {
	defer close(errChannel)
	var orchestrateErr *types.GitspaceError

	switch action {
	case enum.GitspaceActionTypeStart:
		orchestrateErr = c.orchestrator.TriggerStartGitspace(ctxWithTimedOut, config)
	case enum.GitspaceActionTypeStop:
		orchestrateErr = c.orchestrator.TriggerStopGitspace(ctxWithTimedOut, config)
	case enum.GitspaceActionTypeReset:
		orchestrateErr = c.orchestrator.TriggerDeleteGitspace(ctxWithTimedOut, config, false)
	}
	if orchestrateErr != nil {
		orchestrateErr.Error =
			fmt.Errorf("failed to start/stop/reset gitspace: %s %w", config.Identifier, orchestrateErr.Error)
		errChannel <- orchestrateErr
	}
}

func (c *Service) buildGitspaceInstance(config types.GitspaceConfig) (*types.GitspaceInstance, error) {
	gitspaceMachineUser := defaultMachineUser
	now := time.Now().UnixMilli()
	suffixUID, err := gonanoid.Generate(AllowedUIDAlphabet, 6)
	if err != nil {
		return nil, fmt.Errorf("could not generate UID for gitspace config : %q %w", config.Identifier, err)
	}
	identifier := strings.ToLower(config.Identifier + "-" + suffixUID)
	var gitspaceInstance = &types.GitspaceInstance{
		GitSpaceConfigID: config.ID,
		Identifier:       identifier,
		State:            enum.GitspaceInstanceStateStarting,
		UserID:           config.GitspaceUser.Identifier,
		SpaceID:          config.SpaceID,
		SpacePath:        config.SpacePath,
		Created:          now,
		Updated:          now,
		TotalTimeUsed:    0,
	}
	if config.IDE == enum.IDETypeVSCodeWeb || config.IDE == enum.IDETypeVSCode {
		gitspaceInstance.MachineUser = &gitspaceMachineUser
	}
	gitspaceInstance.AccessType = enum.GitspaceAccessTypeSSHKey
	gitspaceInstance.AccessKeyRef = &config.SSHTokenIdentifier
	if len(config.SSHTokenIdentifier) == 0 {
		ref := strings.Clone(defaultPasswordRef)
		gitspaceInstance.AccessKeyRef = &ref
		gitspaceInstance.AccessType = enum.GitspaceAccessTypeUserCredentials
	}
	return gitspaceInstance, nil
}

func (c *Service) EmitGitspaceConfigEvent(
	ctx context.Context,
	config types.GitspaceConfig,
	eventType enum.GitspaceEventType,
) {
	c.gitspaceEventReporter.EmitGitspaceEvent(ctx, events.GitspaceEvent, &events.GitspaceEventPayload{
		QueryKey:   config.Identifier,
		EntityID:   config.ID,
		EntityType: enum.GitspaceEntityTypeGitspaceConfig,
		EventType:  eventType,
		Timestamp:  time.Now().UnixNano(),
	})
}
