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

package container

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/harness/gitness/app/gitspace/logutil"
	"github.com/harness/gitness/app/gitspace/orchestrator/devcontainer"
	"github.com/harness/gitness/app/gitspace/orchestrator/git"
	"github.com/harness/gitness/app/gitspace/orchestrator/ide"
	orchestratorTypes "github.com/harness/gitness/app/gitspace/orchestrator/types"
	"github.com/harness/gitness/app/gitspace/orchestrator/user"
	"github.com/harness/gitness/app/gitspace/scm"
	"github.com/harness/gitness/infraprovider"
	"github.com/harness/gitness/types"

	"github.com/docker/docker/client"
	"github.com/rs/zerolog/log"
)

var _ Orchestrator = (*EmbeddedDockerOrchestrator)(nil)

const (
	loggingKey = "gitspace.container"
)

type EmbeddedDockerOrchestrator struct {
	steps               []orchestratorTypes.Step // Steps registry
	dockerClientFactory *infraprovider.DockerClientFactory
	statefulLogger      *logutil.StatefulLogger
	gitService          git.Service
	userService         user.Service
}

// RegisterStep registers a new setup step with an option to stop or continue on failure.
func (e *EmbeddedDockerOrchestrator) RegisterStep(
	name string,
	execute func(ctx context.Context, exec *devcontainer.Exec, gitspaceLogger orchestratorTypes.GitspaceLogger) error,
	stopOnFailure bool,
) {
	step := orchestratorTypes.Step{
		Name:          name,
		Execute:       execute,
		StopOnFailure: stopOnFailure,
	}
	e.steps = append(e.steps, step)
}

// ExecuteSteps executes all registered steps in sequence, respecting stopOnFailure flag.
func (e *EmbeddedDockerOrchestrator) ExecuteSteps(
	ctx context.Context,
	exec *devcontainer.Exec,
	gitspaceLogger orchestratorTypes.GitspaceLogger,
) error {
	for _, step := range e.steps {
		// Execute the step
		if err := step.Execute(ctx, exec, gitspaceLogger); err != nil {
			// Log the error and decide whether to stop or continue based on stopOnFailure flag
			if step.StopOnFailure {
				return fmt.Errorf("error executing step %s: %w (stopping due to failure)", step.Name, err)
			}
			// Log that we continue despite the failure
			gitspaceLogger.Info(fmt.Sprintf("Step %s failed:", step.Name))
		}
	}
	return nil
}

func NewEmbeddedDockerOrchestrator(
	dockerClientFactory *infraprovider.DockerClientFactory,
	statefulLogger *logutil.StatefulLogger,
	gitService git.Service,
	userService user.Service,
) Orchestrator {
	return &EmbeddedDockerOrchestrator{
		dockerClientFactory: dockerClientFactory,
		statefulLogger:      statefulLogger,
		gitService:          gitService,
		userService:         userService,
	}
}

// CreateAndStartGitspace starts an exited container and starts a new container if the container is removed.
// If the container is newly created, it clones the code, sets up the IDE and executes the postCreateCommand.
// It returns the container ID, name and ports used.
// It returns an error if the container is not running, exited or removed.
func (e *EmbeddedDockerOrchestrator) CreateAndStartGitspace(
	ctx context.Context,
	gitspaceConfig types.GitspaceConfig,
	infra types.Infrastructure,
	resolvedRepoDetails scm.ResolvedDetails,
	defaultBaseImage string,
	ideService ide.IDE,
) (*StartResponse, error) {
	containerName := GetGitspaceContainerName(gitspaceConfig)
	logger := log.Ctx(ctx).With().Str(loggingKey, containerName).Logger()

	// Step 1: Validate access key
	accessKey, err := e.getAccessKey(gitspaceConfig)
	if err != nil {
		return nil, err
	}

	// Step 2: Get Docker client
	dockerClient, err := e.getDockerClient(ctx, infra)
	if err != nil {
		return nil, err
	}
	defer e.closeDockerClient(dockerClient)

	// Step 3: Check the current state of the container
	state, err := e.checkContainerState(ctx, dockerClient, containerName)
	if err != nil {
		return nil, err
	}

	// Step 4: Handle different container states
	switch state {
	case ContainerStateRunning:
		logger.Debug().Msg("gitspace is already running")

	case ContainerStateStopped:
		if err := e.startStoppedGitspace(
			ctx,
			gitspaceConfig,
			dockerClient,
			resolvedRepoDetails,
			accessKey,
			ideService,
		); err != nil {
			return nil, err
		}
	case ContainerStateRemoved:
		if err := e.createAndStartNewGitspace(
			ctx,
			gitspaceConfig,
			dockerClient,
			resolvedRepoDetails,
			infra,
			defaultBaseImage,
			ideService); err != nil {
			return nil, err
		}

	default:
		return nil, fmt.Errorf("gitspace %s is in a bad state: %s", containerName, state)
	}

	homeDir := GetUserHomeDir(gitspaceConfig.GitspaceUser.Identifier)
	codeRepoDir := filepath.Join(homeDir, resolvedRepoDetails.RepoName)

	// Step 5: Retrieve container information and return response
	return e.getContainerResponse(ctx, dockerClient, containerName, infra.GitspacePortMappings, codeRepoDir)
}

// startStoppedGitspace starts the Gitspace container if it was stopped.
func (e *EmbeddedDockerOrchestrator) startStoppedGitspace(
	ctx context.Context,
	gitspaceConfig types.GitspaceConfig,
	dockerClient *client.Client,
	resolvedRepoDetails scm.ResolvedDetails,
	accessKey string,
	ideService ide.IDE,
) error {
	logStreamInstance, err := e.statefulLogger.CreateLogStream(ctx, gitspaceConfig.ID)
	containerName := GetGitspaceContainerName(gitspaceConfig)

	if err != nil {
		return fmt.Errorf("error getting log stream for gitspace ID %d: %w", gitspaceConfig.ID, err)
	}
	defer e.flushLogStream(logStreamInstance, gitspaceConfig.ID)

	startErr := e.manageContainer(ctx, ContainerActionStart, containerName, dockerClient, logStreamInstance)
	if startErr != nil {
		return startErr
	}

	homeDir := GetUserHomeDir(gitspaceConfig.GitspaceUser.Identifier)
	codeRepoDir := filepath.Join(homeDir, resolvedRepoDetails.RepoName)

	exec := &devcontainer.Exec{
		ContainerName:  containerName,
		DockerClient:   dockerClient,
		HomeDir:        homeDir,
		UserIdentifier: gitspaceConfig.GitspaceUser.Identifier,
		AccessKey:      accessKey,
		AccessType:     gitspaceConfig.GitspaceInstance.AccessType,
	}

	// Set up git credentials if needed
	if resolvedRepoDetails.Credentials != nil {
		if err := SetupGitCredentials(ctx, exec, resolvedRepoDetails, e.gitService, logStreamInstance); err != nil {
			return err
		}
	}

	// Run IDE setup
	if err := RunIDE(ctx, exec, ideService, logStreamInstance); err != nil {
		return err
	}

	// Execute post-start command
	devcontainerConfig := resolvedRepoDetails.DevcontainerConfig
	command := ExtractCommand(PostStartAction, devcontainerConfig)
	startErr = ExecuteCommand(ctx, exec, codeRepoDir, logStreamInstance, command, PostStartAction)
	if startErr != nil {
		log.Warn().Msgf("Error is post-start command, continuing : %s", startErr.Error())
	}
	return nil
}

// StopGitspace stops a container. If it is removed, it returns an error.
func (e *EmbeddedDockerOrchestrator) StopGitspace(
	ctx context.Context,
	gitspaceConfig types.GitspaceConfig,
	infra types.Infrastructure,
) error {
	containerName := GetGitspaceContainerName(gitspaceConfig)
	logger := log.Ctx(ctx).With().Str(loggingKey, containerName).Logger()

	// Step 1: Get Docker client
	dockerClient, err := e.getDockerClient(ctx, infra)
	if err != nil {
		return err
	}
	defer e.closeDockerClient(dockerClient)

	// Step 2: Check the current state of the container
	state, err := e.checkContainerState(ctx, dockerClient, containerName)
	if err != nil {
		return err
	}

	// Step 3: Handle container states
	switch state {
	case ContainerStateRemoved:
		return fmt.Errorf("gitspace %s is removed", containerName)

	case ContainerStateStopped:
		logger.Debug().Msg("gitspace is already stopped")
		return nil

	case ContainerStateRunning:
		logger.Debug().Msg("stopping gitspace")
		if err := e.stopRunningGitspace(ctx, gitspaceConfig, containerName, dockerClient); err != nil {
			return err
		}

	default:
		return fmt.Errorf("gitspace %s is in a bad state: %s", containerName, state)
	}

	logger.Debug().Msg("stopped gitspace")
	return nil
}

// stopRunningGitspace handles stopping the container when it is in a running state.
func (e *EmbeddedDockerOrchestrator) stopRunningGitspace(
	ctx context.Context,
	gitspaceConfig types.GitspaceConfig,
	containerName string,
	dockerClient *client.Client,
) error {
	// Step 4: Create log stream for stopping the container
	logStreamInstance, err := e.statefulLogger.CreateLogStream(ctx, gitspaceConfig.ID)
	if err != nil {
		return fmt.Errorf("error getting log stream for gitspace ID %d: %w", gitspaceConfig.ID, err)
	}
	defer e.flushLogStream(logStreamInstance, gitspaceConfig.ID)

	// Step 5: Stop the container
	return e.manageContainer(ctx, ContainerActionStop, containerName, dockerClient, logStreamInstance)
}

// Status is NOOP for EmbeddedDockerOrchestrator as the docker host is verified by the infra provisioner.
func (e *EmbeddedDockerOrchestrator) Status(_ context.Context, _ types.Infrastructure) error {
	return nil
}

// StopAndRemoveGitspace stops the container if not stopped and removes it.
// If the container is already removed, it returns.
func (e *EmbeddedDockerOrchestrator) StopAndRemoveGitspace(
	ctx context.Context,
	gitspaceConfig types.GitspaceConfig,
	infra types.Infrastructure,
) error {
	containerName := GetGitspaceContainerName(gitspaceConfig)
	logger := log.Ctx(ctx).With().Str(loggingKey, containerName).Logger()

	// Step 1: Get Docker client
	dockerClient, err := e.getDockerClient(ctx, infra)
	if err != nil {
		return err
	}
	defer e.closeDockerClient(dockerClient)

	// Step 2: Check the current state of the container
	state, err := e.checkContainerState(ctx, dockerClient, containerName)
	if err != nil {
		return err
	}

	// Step 3: Handle container states
	if state == ContainerStateRemoved {
		logger.Debug().Msg("gitspace is already removed")
		return nil
	}

	// Step 4: Create logger stream for stopping and removing the container
	logStreamInstance, err := e.createLogStream(ctx, gitspaceConfig.ID)
	if err != nil {
		return err
	}
	defer e.flushLogStream(logStreamInstance, gitspaceConfig.ID)

	// Step 5: Stop the container if it's not already stopped
	if state != ContainerStateStopped {
		logger.Debug().Msg("stopping gitspace")
		if err := e.manageContainer(
			ctx, ContainerActionStop, containerName, dockerClient, logStreamInstance); err != nil {
			return fmt.Errorf("failed to stop gitspace %s: %w", containerName, err)
		}
		logger.Debug().Msg("stopped gitspace")
	}

	// Step 6: Remove the container
	logger.Debug().Msg("removing gitspace")
	if err := e.manageContainer(
		ctx, ContainerActionRemove, containerName, dockerClient, logStreamInstance); err != nil {
		return fmt.Errorf("failed to remove gitspace %s: %w", containerName, err)
	}

	logger.Debug().Msg("removed gitspace")
	return nil
}

func (e *EmbeddedDockerOrchestrator) StreamLogs(
	_ context.Context,
	_ types.GitspaceConfig,
	_ types.Infrastructure) (string, error) {
	return "", fmt.Errorf("not implemented")
}

// getAccessKey retrieves the access key from the Gitspace config, returns an error if not found.
func (e *EmbeddedDockerOrchestrator) getAccessKey(gitspaceConfig types.GitspaceConfig) (string, error) {
	if gitspaceConfig.GitspaceInstance != nil && gitspaceConfig.GitspaceInstance.AccessKey != nil {
		return *gitspaceConfig.GitspaceInstance.AccessKey, nil
	}
	return "", fmt.Errorf("no access key is configured: %s", gitspaceConfig.Identifier)
}

// getDockerClient creates and returns a new Docker client using the factory.
func (e *EmbeddedDockerOrchestrator) getDockerClient(
	ctx context.Context,
	infra types.Infrastructure,
) (*client.Client, error) {
	dockerClient, err := e.dockerClientFactory.NewDockerClient(ctx, infra)
	if err != nil {
		return nil, fmt.Errorf("error getting docker client from docker client factory: %w", err)
	}
	return dockerClient, nil
}

// closeDockerClient safely closes the Docker client.
func (e *EmbeddedDockerOrchestrator) closeDockerClient(dockerClient *client.Client) {
	if err := dockerClient.Close(); err != nil {
		log.Warn().Err(err).Msg("failed to close docker client")
	}
}

// checkContainerState checks the current state of the Docker container.
func (e *EmbeddedDockerOrchestrator) checkContainerState(
	ctx context.Context,
	dockerClient *client.Client,
	containerName string,
) (State, error) {
	log.Debug().Msg("checking current state of gitspace")
	state, err := e.containerState(ctx, containerName, dockerClient)
	if err != nil {
		return "", err
	}
	return state, nil
}

// createAndStartNewGitspace creates a new Gitspace if it was removed.
func (e *EmbeddedDockerOrchestrator) createAndStartNewGitspace(
	ctx context.Context,
	gitspaceConfig types.GitspaceConfig,
	dockerClient *client.Client,
	resolvedRepoDetails scm.ResolvedDetails,
	infrastructure types.Infrastructure,
	defaultBaseImage string,
	ideService ide.IDE,
) error {
	logStreamInstance, err := e.statefulLogger.CreateLogStream(ctx, gitspaceConfig.ID)
	if err != nil {
		return fmt.Errorf("error getting log stream for gitspace ID %d: %w", gitspaceConfig.ID, err)
	}
	defer e.flushLogStream(logStreamInstance, gitspaceConfig.ID)

	startErr := e.runGitspaceSetupSteps(
		ctx,
		gitspaceConfig,
		dockerClient,
		ideService,
		infrastructure,
		resolvedRepoDetails,
		defaultBaseImage,
		logStreamInstance,
	)
	if startErr != nil {
		return fmt.Errorf("failed to start gitspace %s: %w", gitspaceConfig.Identifier, startErr)
	}
	return nil
}

// createLogStream creates and returns a log stream for the given gitspace ID.
func (e *EmbeddedDockerOrchestrator) createLogStream(
	ctx context.Context,
	gitspaceID int64,
) (*logutil.LogStreamInstance, error) {
	logStreamInstance, err := e.statefulLogger.CreateLogStream(ctx, gitspaceID)
	if err != nil {
		return nil, fmt.Errorf("error getting log stream for gitspace ID %d: %w", gitspaceID, err)
	}
	return logStreamInstance, nil
}

func (e *EmbeddedDockerOrchestrator) flushLogStream(logStreamInstance *logutil.LogStreamInstance, gitspaceID int64) {
	if err := logStreamInstance.Flush(); err != nil {
		log.Warn().Err(err).Msgf("failed to flush log stream for gitspace ID %d", gitspaceID)
	}
}
