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
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/harness/gitness/app/api/usererror"
	events "github.com/harness/gitness/app/events/gitspace"
	"github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	gonanoid "github.com/matoous/go-nanoid"
	"github.com/rs/zerolog/log"
)

const defaultPasswordRef = "harness_password"
const defaultMachineUser = "harness"
const AllowedUIDAlphabet = "abcdefghijklmnopqrstuvwxyz0123456789"

func (c *Service) StopGitspaceAction(
	ctx context.Context,
	config *types.GitspaceConfig,
	now time.Time,
) error {
	savedGitspaceInstance, err := c.gitspaceInstanceStore.FindLatestByGitspaceConfigID(ctx, config.ID)
	if err != nil {
		return fmt.Errorf("failed to find gitspace with config ID : %s %w", config.Identifier, err)
	}
	if savedGitspaceInstance.State.IsFinalStatus() {
		return fmt.Errorf("gitspace instance cannot be stopped with ID %s", savedGitspaceInstance.Identifier)
	}
	config.GitspaceInstance = savedGitspaceInstance
	err = c.gitspaceBusyOperation(ctx, config)
	if err != nil {
		return err
	}

	activeTimeEnded := now.UnixMilli()
	config.GitspaceInstance.ActiveTimeEnded = &activeTimeEnded
	config.GitspaceInstance.TotalTimeUsed =
		*(config.GitspaceInstance.ActiveTimeEnded) - *(config.GitspaceInstance.ActiveTimeStarted)
	config.GitspaceInstance.State = enum.GitspaceInstanceStateStopping
	if err = c.UpdateInstance(ctx, config.GitspaceInstance); err != nil {
		return fmt.Errorf("failed to update gitspace config for stopping %s %w", config.Identifier, err)
	}
	c.submitAsyncOps(ctx, config, enum.GitspaceActionTypeStop)
	return nil
}

func (c *Service) GitspaceAutostopAction(
	ctx context.Context,
	config *types.GitspaceConfig,
	now time.Time,
) error {
	c.EmitGitspaceConfigEvent(ctx, config, enum.GitspaceEventTypeGitspaceAutoStop)
	if err := c.StopGitspaceAction(ctx, config, now); err != nil {
		return err
	}
	return nil
}

func (c *Service) gitspaceBusyOperation(
	ctx context.Context,
	config *types.GitspaceConfig,
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
	config *types.GitspaceConfig,
	action enum.GitspaceActionType,
) {
	errChannel := make(chan error)

	submitCtx := context.WithoutCancel(ctx)
	gitspaceTimedOutInMins := time.Duration(c.config.Gitspace.ProvisionTimeoutInMins) * time.Minute
	ttlExecuteContext, cancel := context.WithTimeout(submitCtx, gitspaceTimedOutInMins)

	go c.asyncOperation(ttlExecuteContext, *config, action, errChannel)

	var err error

	go func() {
		select {
		case <-ttlExecuteContext.Done():
			if ttlExecuteContext.Err() != nil {
				err = ttlExecuteContext.Err()
			}
		case err = <-errChannel:
		}
		if err != nil {
			log.Err(err).Msgf("error during async execution for %s", config.GitspaceInstance.Identifier)

			config.GitspaceInstance.State = enum.GitspaceInstanceStateError
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
			}
		}

		cancel()
	}()
}

func (c *Service) StartGitspaceAction(
	ctx context.Context,
	config *types.GitspaceConfig,
) error {
	savedGitspaceInstance, err := c.gitspaceInstanceStore.FindLatestByGitspaceConfigID(ctx, config.ID)
	if err != nil && !errors.Is(err, store.ErrResourceNotFound) {
		return err
	}
	config.GitspaceInstance = savedGitspaceInstance
	err = c.gitspaceBusyOperation(ctx, config)
	if err != nil {
		return err
	}
	if savedGitspaceInstance == nil || savedGitspaceInstance.State.IsFinalStatus() {
		gitspaceInstance, err := c.buildGitspaceInstance(config)
		if err != nil {
			return err
		}

		if savedGitspaceInstance != nil {
			gitspaceInstance.HasGitChanges = savedGitspaceInstance.HasGitChanges
		}

		if err = c.gitspaceInstanceStore.Create(ctx, gitspaceInstance); err != nil {
			return fmt.Errorf("failed to create gitspace instance for %s %w", config.Identifier, err)
		}
	}
	newGitspaceInstance, err := c.gitspaceInstanceStore.FindLatestByGitspaceConfigID(ctx, config.ID)
	newGitspaceInstance.SpacePath = config.SpacePath
	if err != nil {
		return fmt.Errorf("failed to find gitspace with config ID : %s %w", config.Identifier, err)
	}
	config.GitspaceInstance = newGitspaceInstance
	c.submitAsyncOps(ctx, config, enum.GitspaceActionTypeStart)
	return nil
}

func (c *Service) asyncOperation(
	ctxWithTimedOut context.Context,
	config types.GitspaceConfig,
	action enum.GitspaceActionType,
	errChannel chan error,
) {
	defer close(errChannel)

	var orchestrateErr error

	switch action {
	case enum.GitspaceActionTypeStart:
		config.GitspaceInstance.State = enum.GitspaceInstanceStateStarting
		err := c.UpdateInstance(ctxWithTimedOut, config.GitspaceInstance)
		if err != nil {
			log.Err(err).Msgf(
				"failed to update gitspace instance during exec %q", config.GitspaceInstance.Identifier)
		}
		orchestrateErr = c.orchestrator.TriggerStartGitspace(ctxWithTimedOut, config)
	case enum.GitspaceActionTypeStop:
		config.GitspaceInstance.State = enum.GitspaceInstanceStateStopping
		err := c.UpdateInstance(ctxWithTimedOut, config.GitspaceInstance)
		if err != nil {
			log.Err(err).Msgf(
				"failed to update gitspace instance during exec %q", config.GitspaceInstance.Identifier)
		}
		orchestrateErr = c.orchestrator.TriggerStopGitspace(ctxWithTimedOut, config)
	}

	if orchestrateErr != nil {
		errChannel <- fmt.Errorf("failed to start/stop gitspace: %s %w", config.Identifier, orchestrateErr)
	}
}

func (c *Service) buildGitspaceInstance(config *types.GitspaceConfig) (*types.GitspaceInstance, error) {
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
		LastUsed:         &now,
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
	config *types.GitspaceConfig,
	eventType enum.GitspaceEventType,
) {
	c.eventReporter.EmitGitspaceEvent(ctx, events.GitspaceEvent, &events.GitspaceEventPayload{
		QueryKey:   config.Identifier,
		EntityID:   config.ID,
		EntityType: enum.GitspaceEntityTypeGitspaceConfig,
		EventType:  eventType,
		Timestamp:  time.Now().UnixNano(),
	})
}
