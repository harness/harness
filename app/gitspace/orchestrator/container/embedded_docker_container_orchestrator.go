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
	"github.com/harness/gitness/app/gitspace/orchestrator/runarg"
	"github.com/harness/gitness/app/gitspace/orchestrator/user"
	"github.com/harness/gitness/app/gitspace/scm"
	gitspaceTypes "github.com/harness/gitness/app/gitspace/types"
	"github.com/harness/gitness/infraprovider"
	"github.com/harness/gitness/types"

	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
	"github.com/rs/zerolog/log"
)

var _ Orchestrator = (*EmbeddedDockerOrchestrator)(nil)

const (
	loggingKey = "gitspace.container"
)

type EmbeddedDockerOrchestrator struct {
	dockerClientFactory *infraprovider.DockerClientFactory
	statefulLogger      *logutil.StatefulLogger
	gitService          git.Service
	userService         user.Service
	runArgProvider      runarg.Provider
}

// ExecuteSteps executes all registered steps in sequence, respecting stopOnFailure flag.
func (e *EmbeddedDockerOrchestrator) ExecuteSteps(
	ctx context.Context,
	exec *devcontainer.Exec,
	gitspaceLogger gitspaceTypes.GitspaceLogger,
	steps []gitspaceTypes.Step,
) error {
	for _, step := range steps {
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
	runArgProvider runarg.Provider,
) Orchestrator {
	return &EmbeddedDockerOrchestrator{
		dockerClientFactory: dockerClientFactory,
		statefulLogger:      statefulLogger,
		gitService:          gitService,
		userService:         userService,
		runArgProvider:      runArgProvider,
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

	// todo : update the code when private repository integration is supported in gitness
	imagAuthMap := make(map[string]gitspaceTypes.DockerRegistryAuth)

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
			ideService,
			imagAuthMap); err != nil {
			return nil, err
		}
	case ContainerStatePaused, ContainerStateCreated, ContainerStateUnknown, ContainerStateDead:
		// TODO handle the following states
		return nil, fmt.Errorf("gitspace %s is in a unhandled state: %s", containerName, state)

	default:
		return nil, fmt.Errorf("gitspace %s is in a bad state: %s", containerName, state)
	}

	// Step 5: Retrieve container information and return response
	return GetContainerResponse(ctx, dockerClient, containerName, infra.GitspacePortMappings, resolvedRepoDetails.RepoName)
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
		return fmt.Errorf("error getting log stream for gitspace instance %s: %w",
			gitspaceConfig.GitspaceInstance.Identifier, err)
	}
	defer e.flushLogStream(logStreamInstance, gitspaceConfig.ID)

	remoteUser, err := GetRemoteUserFromContainerLabel(ctx, containerName, dockerClient)
	if err != nil {
		return fmt.Errorf("error getting remote user for gitspace instance %s: %w",
			gitspaceConfig.GitspaceInstance.Identifier, err)
	}

	homeDir := GetUserHomeDir(remoteUser)

	startErr := ManageContainer(ctx, ContainerActionStart, containerName, dockerClient, logStreamInstance)
	if startErr != nil {
		return startErr
	}

	codeRepoDir := filepath.Join(homeDir, resolvedRepoDetails.RepoName)

	exec := &devcontainer.Exec{
		ContainerName:     containerName,
		DockerClient:      dockerClient,
		DefaultWorkingDir: homeDir,
		RemoteUser:        remoteUser,
		AccessKey:         accessKey,
		AccessType:        gitspaceConfig.GitspaceInstance.AccessType,
	}

	// Set up git credentials if needed
	if resolvedRepoDetails.Credentials != nil {
		if err := SetupGitCredentials(ctx, exec, resolvedRepoDetails, e.gitService, logStreamInstance); err != nil {
			return err
		}
	}

	// Run IDE setup
	if err := RunIDEWithArgs(ctx, exec, ideService, nil, logStreamInstance); err != nil {
		return err
	}

	// Execute post-start command
	devcontainerConfig := resolvedRepoDetails.DevcontainerConfig
	command := ExtractLifecycleCommands(PostStartAction, devcontainerConfig)
	startErr = ExecuteLifecycleCommands(ctx, *exec, codeRepoDir, logStreamInstance, command, PostStartAction)
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
	case ContainerStatePaused, ContainerStateCreated, ContainerStateUnknown, ContainerStateDead:
		// TODO handle the following states
		return fmt.Errorf("gitspace %s is in a unhandled state: %s", containerName, state)
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
	return ManageContainer(ctx, ContainerActionStop, containerName, dockerClient, logStreamInstance)
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
		if err := ManageContainer(
			ctx, ContainerActionStop, containerName, dockerClient, logStreamInstance); err != nil {
			return fmt.Errorf("failed to stop gitspace %s: %w", containerName, err)
		}
		logger.Debug().Msg("stopped gitspace")
	}

	// Step 6: Remove the container
	logger.Debug().Msg("removing gitspace")
	if err := ManageContainer(
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

func (e *EmbeddedDockerOrchestrator) runGitspaceSetupSteps(
	ctx context.Context,
	gitspaceConfig types.GitspaceConfig,
	dockerClient *client.Client,
	ideService ide.IDE,
	infrastructure types.Infrastructure,
	resolvedRepoDetails scm.ResolvedDetails,
	defaultBaseImage string,
	gitspaceLogger gitspaceTypes.GitspaceLogger,
	imageAuthMap map[string]gitspaceTypes.DockerRegistryAuth,
) error {
	containerName := GetGitspaceContainerName(gitspaceConfig)

	devcontainerConfig := resolvedRepoDetails.DevcontainerConfig
	imageName := GetImage(devcontainerConfig, defaultBaseImage)

	runArgsMap, err := ExtractRunArgsWithLogging(ctx, gitspaceConfig.SpaceID, e.runArgProvider,
		devcontainerConfig.RunArgs, gitspaceLogger)
	if err != nil {
		return err
	}

	// Pull the required image
	if err := PullImage(ctx, imageName, dockerClient, runArgsMap, gitspaceLogger, imageAuthMap); err != nil {
		return err
	}

	metadataFromImage, imageUser, err := ExtractMetadataAndUserFromImage(ctx, imageName, dockerClient)
	if err != nil {
		return err
	}

	portMappings := infrastructure.GitspacePortMappings
	forwardPorts := ExtractForwardPorts(devcontainerConfig)
	if len(forwardPorts) > 0 {
		for _, port := range forwardPorts {
			portMappings[port] = &types.PortMapping{
				PublishedPort: port,
				ForwardedPort: port,
			}
		}
		gitspaceLogger.Info(fmt.Sprintf("Forwarding ports : %v", forwardPorts))
	}

	storage := infrastructure.Storage
	environment := ExtractEnv(devcontainerConfig, runArgsMap)
	if len(environment) > 0 {
		gitspaceLogger.Info(fmt.Sprintf("Setting Environment : %v", environment))
	}

	containerUser := GetContainerUser(runArgsMap, devcontainerConfig, metadataFromImage, imageUser)
	remoteUser := GetRemoteUser(devcontainerConfig, metadataFromImage, containerUser)

	homeDir := GetUserHomeDir(remoteUser)

	gitspaceLogger.Info(fmt.Sprintf("Container user: %s", containerUser))
	gitspaceLogger.Info(fmt.Sprintf("Remote user: %s", remoteUser))

	// Create the container
	err = CreateContainer(
		ctx,
		dockerClient,
		imageName,
		containerName,
		gitspaceLogger,
		storage,
		homeDir,
		mount.TypeVolume,
		portMappings,
		environment,
		runArgsMap,
		containerUser,
		remoteUser,
	)
	if err != nil {
		return err
	}

	// Start the container
	if err := ManageContainer(ctx, ContainerActionStart, containerName, dockerClient, gitspaceLogger); err != nil {
		return err
	}

	// Setup and run commands
	exec := &devcontainer.Exec{
		ContainerName:     containerName,
		DockerClient:      dockerClient,
		DefaultWorkingDir: homeDir,
		RemoteUser:        remoteUser,
		AccessKey:         *gitspaceConfig.GitspaceInstance.AccessKey,
		AccessType:        gitspaceConfig.GitspaceInstance.AccessType,
	}

	if err := e.setupGitspaceAndIDE(
		ctx,
		exec,
		gitspaceLogger,
		ideService,
		gitspaceConfig,
		resolvedRepoDetails,
		defaultBaseImage,
		environment,
	); err != nil {
		return err
	}

	return nil
}

// buildSetupSteps constructs the steps to be executed in the setup process.
func (e *EmbeddedDockerOrchestrator) buildSetupSteps(
	_ context.Context,
	ideService ide.IDE,
	gitspaceConfig types.GitspaceConfig,
	resolvedRepoDetails scm.ResolvedDetails,
	defaultBaseImage string,
	environment []string,
	devcontainerConfig types.DevcontainerConfig,
	codeRepoDir string,
) []gitspaceTypes.Step {
	return []gitspaceTypes.Step{
		{
			Name:          "Validate Supported OS",
			Execute:       ValidateSupportedOS,
			StopOnFailure: true,
		},
		{
			Name: "Manage User",
			Execute: func(
				ctx context.Context,
				exec *devcontainer.Exec,
				gitspaceLogger gitspaceTypes.GitspaceLogger,
			) error {
				return ManageUser(ctx, exec, e.userService, gitspaceLogger)
			},
			StopOnFailure: true,
		},
		{
			Name: "Set environment",
			Execute: func(
				ctx context.Context,
				exec *devcontainer.Exec,
				gitspaceLogger gitspaceTypes.GitspaceLogger,
			) error {
				return SetEnv(ctx, exec, gitspaceLogger, environment)
			},
			StopOnFailure: true,
		},
		{
			Name: "Install Tools",
			Execute: func(
				ctx context.Context,
				exec *devcontainer.Exec,
				gitspaceLogger gitspaceTypes.GitspaceLogger,
			) error {
				return InstallTools(ctx, exec, gitspaceLogger, gitspaceConfig.IDE)
			},
			StopOnFailure: true,
		},
		{
			Name: "Install Git",
			Execute: func(
				ctx context.Context,
				exec *devcontainer.Exec,
				gitspaceLogger gitspaceTypes.GitspaceLogger,
			) error {
				return InstallGit(ctx, exec, e.gitService, gitspaceLogger)
			},
			StopOnFailure: true,
		},
		{
			Name: "Setup Git Credentials",
			Execute: func(
				ctx context.Context,
				exec *devcontainer.Exec,
				gitspaceLogger gitspaceTypes.GitspaceLogger,
			) error {
				if resolvedRepoDetails.ResolvedCredentials.Credentials != nil {
					return SetupGitCredentials(ctx, exec, resolvedRepoDetails, e.gitService, gitspaceLogger)
				}
				return nil
			},
			StopOnFailure: true,
		},
		{
			Name: "Clone Code",
			Execute: func(
				ctx context.Context,
				exec *devcontainer.Exec,
				gitspaceLogger gitspaceTypes.GitspaceLogger,
			) error {
				return CloneCode(ctx, exec, defaultBaseImage, resolvedRepoDetails, e.gitService, gitspaceLogger)
			},
			StopOnFailure: true,
		},
		{
			Name: "Setup IDE",
			Execute: func(
				ctx context.Context,
				exec *devcontainer.Exec,
				gitspaceLogger gitspaceTypes.GitspaceLogger,
			) error {
				// Run IDE setup
				args := ExtractIDECustomizations(ideService, resolvedRepoDetails.DevcontainerConfig)
				args[gitspaceTypes.IDERepoNameArg] = resolvedRepoDetails.RepoName
				return SetupIDE(ctx, exec, ideService, args, gitspaceLogger)
			},
			StopOnFailure: true,
		},
		{
			Name: "Run IDE",
			Execute: func(
				ctx context.Context,
				exec *devcontainer.Exec,
				gitspaceLogger gitspaceTypes.GitspaceLogger,
			) error {
				return RunIDEWithArgs(ctx, exec, ideService, nil, gitspaceLogger)
			},
			StopOnFailure: true,
		},
		// Post-create and Post-start steps
		{
			Name: "Execute PostCreate Command",
			Execute: func(
				ctx context.Context,
				exec *devcontainer.Exec,
				gitspaceLogger gitspaceTypes.GitspaceLogger,
			) error {
				command := ExtractLifecycleCommands(PostCreateAction, devcontainerConfig)
				return ExecuteLifecycleCommands(ctx, *exec, codeRepoDir, gitspaceLogger, command, PostCreateAction)
			},
			StopOnFailure: false,
		},
		{
			Name: "Execute PostStart Command",
			Execute: func(
				ctx context.Context,
				exec *devcontainer.Exec,
				gitspaceLogger gitspaceTypes.GitspaceLogger,
			) error {
				command := ExtractLifecycleCommands(PostStartAction, devcontainerConfig)
				return ExecuteLifecycleCommands(ctx, *exec, codeRepoDir, gitspaceLogger, command, PostStartAction)
			},
			StopOnFailure: false,
		},
	}
}

// setupGitspaceAndIDE initializes Gitspace and IDE by registering and executing the setup steps.
func (e *EmbeddedDockerOrchestrator) setupGitspaceAndIDE(
	ctx context.Context,
	exec *devcontainer.Exec,
	gitspaceLogger gitspaceTypes.GitspaceLogger,
	ideService ide.IDE,
	gitspaceConfig types.GitspaceConfig,
	resolvedRepoDetails scm.ResolvedDetails,
	defaultBaseImage string,
	environment []string,
) error {
	homeDir := GetUserHomeDir(exec.RemoteUser)
	devcontainerConfig := resolvedRepoDetails.DevcontainerConfig
	codeRepoDir := filepath.Join(homeDir, resolvedRepoDetails.RepoName)

	steps := e.buildSetupSteps(
		ctx,
		ideService,
		gitspaceConfig,
		resolvedRepoDetails,
		defaultBaseImage,
		environment,
		devcontainerConfig,
		codeRepoDir)

	// Execute the registered steps
	if err := e.ExecuteSteps(ctx, exec, gitspaceLogger, steps); err != nil {
		return err
	}
	return nil
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
	state, err := FetchContainerState(ctx, containerName, dockerClient)
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
	imageAuthMap map[string]gitspaceTypes.DockerRegistryAuth,
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
		imageAuthMap,
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
