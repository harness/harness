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
	"strconv"
	"strings"
	"time"

	apiauth "github.com/harness/gitness/app/api/auth"
	"github.com/harness/gitness/app/auth"
	events "github.com/harness/gitness/app/events/gitspace"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/check"
	"github.com/harness/gitness/types/enum"

	gonanoid "github.com/matoous/go-nanoid"
	"github.com/rs/zerolog/log"
)

const defaultAccessKey = "Harness@123"
const defaultMachineUser = "harness"
const gitspaceTimedOutInMintues = 5

type ActionInput struct {
	Action     enum.GitspaceActionType `json:"action"`
	Identifier string                  `json:"-"`
	SpaceRef   string                  `json:"-"` // Ref of the parent space
}

func (c *Controller) Action(
	ctx context.Context,
	session *auth.Session,
	in *ActionInput,
) (*types.GitspaceConfig, error) {
	if err := c.sanitizeActionInput(in); err != nil {
		return nil, fmt.Errorf("failed to sanitize input: %w", err)
	}
	space, err := c.spaceStore.FindByRef(ctx, in.SpaceRef)
	if err != nil {
		return nil, fmt.Errorf("failed to find space: %w", err)
	}
	err = apiauth.CheckGitspace(ctx, c.authorizer, session, space.Path, in.Identifier, enum.PermissionGitspaceAccess)
	if err != nil {
		return nil, fmt.Errorf("failed to authorize: %w", err)
	}

	gitspaceConfig, err := c.gitspaceConfigStore.FindByIdentifier(ctx, space.ID, in.Identifier)
	gitspaceConfig.SpacePath = space.Path
	gitspaceConfig.SpaceID = space.ID
	if err != nil {
		return nil, fmt.Errorf("failed to find gitspace config: %w", err)
	}

	// check if it's an internal repo
	if gitspaceConfig.CodeRepoType == enum.CodeRepoTypeGitness {
		if gitspaceConfig.CodeRepoRef == nil {
			return nil, fmt.Errorf("couldn't fetch repo for the user, no ref found: %w", err)
		}
		repo, err := c.repoStore.FindByRef(ctx, *gitspaceConfig.CodeRepoRef)
		if err != nil {
			return nil, fmt.Errorf("couldn't fetch repo for the user: %w", err)
		}
		if err = apiauth.CheckRepo(
			ctx,
			c.authorizer,
			session,
			repo,
			enum.PermissionRepoView); err != nil {
			return nil, err
		}
	}
	// All the actions should be idempotent.
	switch in.Action {
	case enum.GitspaceActionTypeStart:
		c.emitGitspaceConfigEvent(ctx, gitspaceConfig, enum.GitspaceEventTypeGitspaceActionStart)
		err = c.startGitspaceAction(ctx, gitspaceConfig)
		return gitspaceConfig, err
	case enum.GitspaceActionTypeStop:
		c.emitGitspaceConfigEvent(ctx, gitspaceConfig, enum.GitspaceEventTypeGitspaceActionStop)
		err = c.stopGitspaceAction(ctx, gitspaceConfig)
		return gitspaceConfig, err
	default:
		return nil, fmt.Errorf("unknown action %s on gitspace : %s", string(in.Action), gitspaceConfig.Identifier)
	}
}

func (c *Controller) startGitspaceAction(
	ctx context.Context,
	config *types.GitspaceConfig,
) error {
	savedGitspaceInstance, err := c.gitspaceInstanceStore.FindLatestByGitspaceConfigID(ctx, config.ID, config.SpaceID)
	const resourceNotFoundErr = "Failed to find gitspace: resource not found"
	if err != nil && err.Error() != resourceNotFoundErr {
		return fmt.Errorf("failed to find gitspace instance for config ID : %s %w", config.Identifier, err)
	}
	config.GitspaceInstance = savedGitspaceInstance
	err = c.gitspaceBusyOperation(ctx, config)
	if err != nil {
		return err
	}
	if savedGitspaceInstance == nil || savedGitspaceInstance.State.IsFinalStatus() {
		gitspaceInstance, err := c.createGitspaceInstance(config)
		if err != nil {
			return err
		}
		if err = c.gitspaceInstanceStore.Create(ctx, gitspaceInstance); err != nil {
			return fmt.Errorf("failed to create gitspace instance for %s %w", config.Identifier, err)
		}
	}
	newGitspaceInstance, err := c.gitspaceInstanceStore.FindLatestByGitspaceConfigID(ctx, config.ID, config.SpaceID)
	newGitspaceInstance.SpacePath = config.SpacePath
	if err != nil {
		return fmt.Errorf("failed to find gitspace with config ID : %s %w", config.Identifier, err)
	}
	config.GitspaceInstance = newGitspaceInstance
	config.State, _ = enum.GetGitspaceStateFromInstance(newGitspaceInstance.State)
	c.submitAsyncOps(ctx, config, enum.GitspaceActionTypeStart)
	return nil
}

func (c *Controller) asyncOperation(
	ctxWithTimedOut context.Context,
	config types.GitspaceConfig,
	action enum.GitspaceActionType,
	stateChannel chan enum.GitspaceInstanceStateType,
	errChannel chan error,
) {
	defer close(stateChannel)
	defer close(errChannel)

	var orchestrateErr error
	var instanceState enum.GitspaceInstanceStateType

	switch action {
	case enum.GitspaceActionTypeStart:
		instanceState, orchestrateErr = c.orchestrator.TriggerStartGitspace(ctxWithTimedOut, config)
	case enum.GitspaceActionTypeStop:
		instanceState, orchestrateErr = c.orchestrator.TriggerStopGitspace(ctxWithTimedOut, config)
	}

	if orchestrateErr != nil {
		errChannel <- fmt.Errorf("failed to start/stop gitspace: %s %w", config.Identifier, orchestrateErr)
	}
	stateChannel <- instanceState
}

func (c *Controller) submitAsyncOps(
	ctx context.Context,
	config *types.GitspaceConfig,
	action enum.GitspaceActionType,
) {
	errChannel := make(chan error)
	stateChannel := make(chan enum.GitspaceInstanceStateType)

	submitCtx := context.WithoutCancel(ctx)
	ttlExecuteContext, cancel := context.WithTimeout(submitCtx, gitspaceTimedOutInMintues*time.Minute)

	go c.asyncOperation(ttlExecuteContext, *config, action, stateChannel, errChannel)

	var err error
	var instanceState enum.GitspaceInstanceStateType

	go func() {
		select {
		case <-ttlExecuteContext.Done():
			if ttlExecuteContext.Err() != nil {
				err = ttlExecuteContext.Err()
			}
		case err = <-errChannel:
		case instanceState = <-stateChannel:
		}
		if err != nil {
			log.Err(err).Msgf("error during async execution for %s", config.GitspaceInstance.Identifier)
			switch action {
			case enum.GitspaceActionTypeStart:
				c.emitGitspaceConfigEvent(ttlExecuteContext, config, enum.GitspaceEventTypeGitspaceActionStartFailed)
			case enum.GitspaceActionTypeStop:
				c.emitGitspaceConfigEvent(ttlExecuteContext, config, enum.GitspaceEventTypeGitspaceActionStopFailed)
			}
		}

		if instanceState == "" {
			instanceState = enum.GitspaceInstanceStateError
		}

		config.GitspaceInstance.State = instanceState

		c.updateGitspaceInstance(submitCtx, config.GitspaceInstance)

		cancel()
	}()
}

func (c *Controller) createGitspaceInstance(config *types.GitspaceConfig) (*types.GitspaceInstance, error) {
	codeServerPassword := defaultAccessKey
	gitspaceMachineUser := defaultMachineUser
	now := time.Now().UnixMilli()
	suffixUID, err := gonanoid.Generate(allowedUIDAlphabet, 6)
	if err != nil {
		return nil, fmt.Errorf("could not generate UID for gitspace config : %q %w", config.Identifier, err)
	}
	identifier := strings.ToLower(config.Identifier + "-" + suffixUID)
	var gitspaceInstance = &types.GitspaceInstance{
		GitSpaceConfigID: config.ID,
		Identifier:       identifier,
		State:            enum.GitspaceInstanceStateStarting,
		UserID:           config.UserID,
		SpaceID:          config.SpaceID,
		SpacePath:        config.SpacePath,
		Created:          now,
		Updated:          now,
		TotalTimeUsed:    0,
	}
	if config.IDE == enum.IDETypeVSCodeWeb || config.IDE == enum.IDETypeVSCode {
		gitspaceInstance.AccessKey = &codeServerPassword
		gitspaceInstance.AccessType = enum.GitspaceAccessTypeUserCredentials
		gitspaceInstance.MachineUser = &gitspaceMachineUser
	}
	return gitspaceInstance, nil
}

func (c *Controller) gitspaceBusyOperation(
	ctx context.Context,
	config *types.GitspaceConfig,
) error {
	if config.GitspaceInstance == nil {
		return nil
	}
	if config.GitspaceInstance.State.IsBusyStatus() &&
		time.Since(time.UnixMilli(config.GitspaceInstance.Updated)).Milliseconds() <= (gitspaceTimedOutInMintues*60*1000) {
		return fmt.Errorf("gitspace start/stop is already pending for : %q", config.Identifier)
	} else if config.GitspaceInstance.State.IsBusyStatus() {
		config.GitspaceInstance.State = enum.GitspaceInstanceStateError
		if err := c.gitspaceInstanceStore.Update(ctx, config.GitspaceInstance); err != nil {
			return fmt.Errorf("failed to update gitspace config for %s %w", config.Identifier, err)
		}
	}
	return nil
}

func (c *Controller) stopGitspaceAction(
	ctx context.Context,
	config *types.GitspaceConfig,
) error {
	savedGitspaceInstance, err := c.gitspaceInstanceStore.FindLatestByGitspaceConfigID(ctx, config.ID, config.SpaceID)
	if err != nil {
		return fmt.Errorf("failed to find gitspace with config ID : %s %w", config.Identifier, err)
	}
	if savedGitspaceInstance.State.IsFinalStatus() {
		return fmt.Errorf(
			"gitspace instance cannot be stopped with ID %s %w", savedGitspaceInstance.Identifier, err)
	}
	config.GitspaceInstance = savedGitspaceInstance
	err = c.gitspaceBusyOperation(ctx, config)
	if err != nil {
		return err
	}
	config.GitspaceInstance.State = enum.GitspaceInstanceStateStopping
	if err = c.gitspaceInstanceStore.Update(ctx, config.GitspaceInstance); err != nil {
		return fmt.Errorf("failed to update gitspace config for stopping %s %w", config.Identifier, err)
	}
	config.State, _ = enum.GetGitspaceStateFromInstance(savedGitspaceInstance.State)
	c.submitAsyncOps(ctx, config, enum.GitspaceActionTypeStop)
	return nil
}

func (c *Controller) sanitizeActionInput(in *ActionInput) error {
	if err := check.Identifier(in.Identifier); err != nil {
		return err
	}
	parentRefAsID, err := strconv.ParseInt(in.SpaceRef, 10, 64)
	if (err == nil && parentRefAsID <= 0) || (len(strings.TrimSpace(in.SpaceRef)) == 0) {
		return ErrGitspaceRequiresParent
	}
	return nil
}

func (c *Controller) emitGitspaceConfigEvent(
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

func (c *Controller) updateGitspaceInstance(
	ctx context.Context,
	instance *types.GitspaceInstance,
) {
	err := c.gitspaceInstanceStore.Update(ctx, instance)
	if err != nil {
		log.Err(err).Msgf(
			"failed to update gitspace instance during exec %q", instance.Identifier)
	}
}
