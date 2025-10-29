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
	"strings"

	events "github.com/harness/gitness/app/events/gitspaceoperations"
	"github.com/harness/gitness/app/gitspace/logutil"
	"github.com/harness/gitness/app/gitspace/orchestrator/container/response"
	"github.com/harness/gitness/app/gitspace/orchestrator/devcontainer"
	"github.com/harness/gitness/app/gitspace/orchestrator/ide"
	"github.com/harness/gitness/app/gitspace/orchestrator/runarg"
	"github.com/harness/gitness/app/gitspace/orchestrator/utils"
	"github.com/harness/gitness/app/gitspace/scm"
	gitspaceTypes "github.com/harness/gitness/app/gitspace/types"
	"github.com/harness/gitness/infraprovider"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
	"github.com/rs/zerolog/log"
)

var _ Orchestrator = (*EmbeddedDockerOrchestrator)(nil)
var loggingDivider = "\n" + strings.Repeat("=", 100) + "\n"

const (
	loggingKey = "gitspace.container"
)

type EmbeddedDockerOrchestrator struct {
	dockerClientFactory *infraprovider.DockerClientFactory
	statefulLogger      *logutil.StatefulLogger
	runArgProvider      runarg.Provider
	eventReporter       *events.Reporter
}

// Step represents a single setup action.
type step struct {
	Name          string
	Execute       func(ctx context.Context, exec *devcontainer.Exec, gitspaceLogger gitspaceTypes.GitspaceLogger) error
	StopOnFailure bool // Flag to control whether execution should stop on failure
}

type LifecycleHookStep struct {
	Source        string                 `json:"source,omitempty"`
	Command       types.LifecycleCommand `json:"command,omitzero"`
	ActionType    PostAction             `json:"action_type,omitempty"`
	StopOnFailure bool                   `json:"stop_on_failure,omitempty"`
}

// ExecuteSteps executes all registered steps in sequence, respecting stopOnFailure flag.
func (e *EmbeddedDockerOrchestrator) ExecuteSteps(
	ctx context.Context,
	exec *devcontainer.Exec,
	gitspaceLogger gitspaceTypes.GitspaceLogger,
	steps []step,
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
	runArgProvider runarg.Provider,
	eventReporter *events.Reporter,
) EmbeddedDockerOrchestrator {
	return EmbeddedDockerOrchestrator{
		dockerClientFactory: dockerClientFactory,
		statefulLogger:      statefulLogger,
		runArgProvider:      runArgProvider,
		eventReporter:       eventReporter,
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
) error {
	containerName := GetGitspaceContainerName(gitspaceConfig)
	logger := log.Ctx(ctx).With().Str(loggingKey, containerName).Logger()

	// Step 1: Validate access key
	accessKey, err := e.getAccessKey(gitspaceConfig)
	if err != nil {
		return err
	}

	// Step 2: Get Docker client
	dockerClient, err := e.getDockerClient(ctx, infra)
	if err != nil {
		return err
	}
	defer e.closeDockerClient(dockerClient)

	// todo : update the code when private repository integration is supported in gitness
	imagAuthMap := make(map[string]gitspaceTypes.DockerRegistryAuth)

	// Step 3: Check the current state of the container
	state, err := e.checkContainerState(ctx, dockerClient, containerName)
	if err != nil {
		return err
	}

	// Step 4: Handle different container states
	switch state {
	case ContainerStateRunning:
		logger.Debug().Msg("gitspace is already running")

	case ContainerStateStopped:
		if err = e.startStoppedGitspace(
			ctx,
			gitspaceConfig,
			dockerClient,
			resolvedRepoDetails,
			accessKey,
			ideService,
		); err != nil {
			return err
		}
	case ContainerStateRemoved:
		if err = e.createAndStartNewGitspace(
			ctx,
			gitspaceConfig,
			dockerClient,
			resolvedRepoDetails,
			infra,
			defaultBaseImage,
			ideService,
			imagAuthMap); err != nil {
			return err
		}
	case ContainerStatePaused, ContainerStateCreated, ContainerStateUnknown, ContainerStateDead:
		// TODO handle the following states
		return fmt.Errorf("gitspace %s is in a unhandled state: %s", containerName, state)

	default:
		return fmt.Errorf("gitspace %s is in a bad state: %s", containerName, state)
	}

	// Step 5: Retrieve container information and return response
	startResponse, err := GetContainerResponse(
		ctx,
		dockerClient,
		containerName,
		infra.GitspacePortMappings,
		resolvedRepoDetails.RepoName,
	)
	if err != nil {
		return err
	}

	return e.eventReporter.EmitGitspaceOperationsEvent(
		ctx,
		events.GitspaceOperationsEvent,
		&events.GitspaceOperationsEventPayload{
			Type:     enum.GitspaceOperationsEventStart,
			Infra:    infra,
			Response: *startResponse,
		},
	)
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

	remoteUser, lifecycleHooks, err := GetGitspaceInfoFromContainerLabels(ctx, containerName, dockerClient)
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
	if resolvedRepoDetails.UserPasswordCredentials != nil {
		if err = utils.SetupGitCredentials(ctx, exec, resolvedRepoDetails, logStreamInstance); err != nil {
			return err
		}
	}

	// Run IDE setup
	runIDEArgs := make(map[gitspaceTypes.IDEArg]any)
	runIDEArgs[gitspaceTypes.IDERepoNameArg] = resolvedRepoDetails.RepoName
	runIDEArgs = AddIDEDirNameArg(ideService, runIDEArgs)
	if err = ideService.Run(ctx, exec, runIDEArgs, logStreamInstance); err != nil {
		return err
	}

	if len(lifecycleHooks) > 0 && len(lifecycleHooks[PostStartAction]) > 0 {
		for _, lifecycleHook := range lifecycleHooks[PostStartAction] {
			startErr = ExecuteLifecycleCommands(ctx, *exec, codeRepoDir, logStreamInstance,
				lifecycleHook.Command.ToCommandArray(), PostStartAction)
			if startErr != nil {
				log.Warn().Msgf("Error in post-start command, continuing : %s", startErr.Error())
			}
		}
	} else {
		// Execute post-start command for the containers before this label was introduced
		devcontainerConfig := resolvedRepoDetails.DevcontainerConfig
		command := ExtractLifecycleCommands(PostStartAction, devcontainerConfig)
		startErr = ExecuteLifecycleCommands(ctx, *exec, codeRepoDir, logStreamInstance, command, PostStartAction)
		if startErr != nil {
			log.Warn().Msgf("Error in post-start command, continuing : %s", startErr.Error())
		}
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
		if err = e.stopRunningGitspace(ctx, gitspaceConfig, containerName, dockerClient); err != nil {
			return err
		}
	case ContainerStatePaused, ContainerStateCreated, ContainerStateUnknown, ContainerStateDead:
		// TODO handle the following states
		return fmt.Errorf("gitspace %s is in a unhandled state: %s", containerName, state)
	default:
		return fmt.Errorf("gitspace %s is in a bad state: %s", containerName, state)
	}

	stopResponse := &response.StopResponse{
		Status: response.SuccessStatus,
	}

	err = e.eventReporter.EmitGitspaceOperationsEvent(
		ctx,
		events.GitspaceOperationsEvent,
		&events.GitspaceOperationsEventPayload{
			Type:     enum.GitspaceOperationsEventStop,
			Infra:    infra,
			Response: stopResponse,
		},
	)
	logger.Debug().Msg("stopped gitspace")
	return err
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

// RemoveGitspace force removes the container and the operation is idempotent.
// If the container is already removed, it returns.
func (e *EmbeddedDockerOrchestrator) RemoveGitspace(
	ctx context.Context,
	gitspaceConfig types.GitspaceConfig,
	infra types.Infrastructure,
	canDeleteUserData bool,
) error {
	containerName := GetGitspaceContainerName(gitspaceConfig)
	logger := log.Ctx(ctx).With().Str(loggingKey, containerName).Logger()

	// Step 1: Get Docker client
	dockerClient, err := e.getDockerClient(ctx, infra)
	if err != nil {
		return err
	}
	defer e.closeDockerClient(dockerClient)

	// Step 4: Create logger stream for stopping and removing the container
	logStreamInstance, err := e.createLogStream(ctx, gitspaceConfig.ID)
	if err != nil {
		return err
	}
	defer e.flushLogStream(logStreamInstance, gitspaceConfig.ID)

	logger.Debug().Msg("removing gitspace")
	if err = ManageContainer(
		ctx, ContainerActionRemove, containerName, dockerClient, logStreamInstance); err != nil {
		if client.IsErrNotFound(err) {
			logger.Debug().Msg("gitspace is already removed")
		} else {
			return fmt.Errorf("failed to remove gitspace %s: %w", containerName, err)
		}
	}

	err = e.eventReporter.EmitGitspaceOperationsEvent(
		ctx,
		events.GitspaceOperationsEvent,
		&events.GitspaceOperationsEventPayload{
			Type:  enum.GitspaceOperationsEventDelete,
			Infra: infra,
			Response: &response.DeleteResponse{
				Status:            response.SuccessStatus,
				CanDeleteUserData: canDeleteUserData,
			},
		},
	)
	logger.Debug().Msg("removed gitspace")
	return err
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
	imageName := getImage(devcontainerConfig, defaultBaseImage)

	runArgsMap, err := ExtractRunArgsWithLogging(ctx, gitspaceConfig.SpaceID, e.runArgProvider,
		devcontainerConfig.RunArgs, gitspaceLogger)
	if err != nil {
		return err
	}

	// Pull the required image
	if err = PullImage(ctx, imageName, dockerClient, runArgsMap, gitspaceLogger, imageAuthMap); err != nil {
		return err
	}

	imageData, err := ExtractImageData(ctx, imageName, dockerClient)
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

	containerUser := GetContainerUser(runArgsMap, devcontainerConfig, imageData.Metadata, imageData.User)
	remoteUser := GetRemoteUser(devcontainerConfig, imageData.Metadata, containerUser)

	containerUserHomeDir := GetUserHomeDir(containerUser)
	remoteUserHomeDir := GetUserHomeDir(remoteUser)

	gitspaceLogger.Info(fmt.Sprintf("Container user: %s", containerUser))
	gitspaceLogger.Info(fmt.Sprintf("Remote user: %s", remoteUser))
	var features []*types.ResolvedFeature
	if devcontainerConfig.Features != nil && len(*devcontainerConfig.Features) > 0 {
		sortedFeatures, newImageName, err := InstallFeatures(ctx, gitspaceConfig.GitspaceInstance.Identifier,
			dockerClient, *devcontainerConfig.Features, devcontainerConfig.OverrideFeatureInstallOrder, imageName,
			containerUser, remoteUser, containerUserHomeDir, remoteUserHomeDir, gitspaceLogger)
		if err != nil {
			return err
		}
		features = sortedFeatures
		imageName = newImageName
	} else {
		gitspaceLogger.Info("No features found")
	}

	// Create the container
	lifecycleHookSteps, err := CreateContainer(
		ctx,
		dockerClient,
		imageName,
		containerName,
		gitspaceLogger,
		storage,
		remoteUserHomeDir,
		mount.TypeVolume,
		portMappings,
		environment,
		runArgsMap,
		containerUser,
		remoteUser,
		features,
		resolvedRepoDetails.DevcontainerConfig,
		imageData.Metadata,
	)
	if err != nil {
		return err
	}

	// Start the container
	if err = ManageContainer(ctx, ContainerActionStart, containerName, dockerClient, gitspaceLogger); err != nil {
		return err
	}

	// Setup and run commands
	exec := &devcontainer.Exec{
		ContainerName:     containerName,
		DockerClient:      dockerClient,
		DefaultWorkingDir: remoteUserHomeDir,
		RemoteUser:        remoteUser,
		AccessKey:         *gitspaceConfig.GitspaceInstance.AccessKey,
		AccessType:        gitspaceConfig.GitspaceInstance.AccessType,
		Arch:              imageData.Arch,
		OS:                imageData.OS,
	}

	if err = e.setupGitspaceAndIDE(
		ctx,
		exec,
		gitspaceLogger,
		ideService,
		gitspaceConfig,
		resolvedRepoDetails,
		defaultBaseImage,
		environment,
		lifecycleHookSteps,
	); err != nil {
		return logStreamWrapError(gitspaceLogger, "Error while setting up gitspace", err)
	}

	return nil
}

func InstallFeatures(
	ctx context.Context,
	gitspaceInstanceIdentifier string,
	dockerClient *client.Client,
	features types.Features,
	overrideFeatureInstallOrder []string,
	imageName string,
	containerUser string,
	remoteUser string,
	containerUserHomeDir string,
	remoteUserHomeDir string,
	gitspaceLogger gitspaceTypes.GitspaceLogger,
) ([]*types.ResolvedFeature, string, error) {
	gitspaceLogger.Info("Downloading features...")
	downloadedFeatures, err := utils.DownloadFeatures(ctx, gitspaceInstanceIdentifier, features)
	if err != nil {
		return nil, "", logStreamWrapError(gitspaceLogger, "Error downloading features", err)
	}
	gitspaceLogger.Info(fmt.Sprintf("Downloaded %d features", len(*downloadedFeatures)))

	gitspaceLogger.Info("Resolving features...")
	resolvedFeatures, err := utils.ResolveFeatures(features, *downloadedFeatures)
	if err != nil {
		return nil, "", logStreamWrapError(gitspaceLogger, "Error resolving features", err)
	}
	gitspaceLogger.Info(fmt.Sprintf("Resolved to %d features", len(resolvedFeatures)))

	gitspaceLogger.Info("Determining feature installation order...")
	sortedFeatures, err := utils.SortFeatures(resolvedFeatures, overrideFeatureInstallOrder)
	if err != nil {
		return nil, "", logStreamWrapError(gitspaceLogger, "Error sorting features", err)
	}
	gitspaceLogger.Info("Feature installation order is:")
	for index, feature := range sortedFeatures {
		gitspaceLogger.Info(fmt.Sprintf("%d. %s", index, feature.Print()))
	}

	gitspaceLogger.Info("Installing features...")
	newImageName, dockerFileContent, err := utils.BuildWithFeatures(ctx, dockerClient, imageName, sortedFeatures,
		gitspaceInstanceIdentifier, containerUser, remoteUser, containerUserHomeDir, remoteUserHomeDir)
	gitspaceLogger.Info(fmt.Sprintf("Using dockerfile%s%s%s", loggingDivider, dockerFileContent, loggingDivider))
	if err != nil {
		return nil, "", logStreamWrapError(gitspaceLogger, "Error building with features", err)
	}
	gitspaceLogger.Info(fmt.Sprintf("Installed features, built new docker image %s", newImageName))

	return sortedFeatures, newImageName, nil
}

// buildSetupSteps constructs the steps to be executed in the setup process.
func (e *EmbeddedDockerOrchestrator) buildSetupSteps(
	ideService ide.IDE,
	gitspaceConfig types.GitspaceConfig,
	resolvedRepoDetails scm.ResolvedDetails,
	defaultBaseImage string,
	environment []string,
	codeRepoDir string,
	lifecycleHookSteps map[PostAction][]*LifecycleHookStep,
) []step {
	steps := []step{
		{
			Name:          "Validate Supported OS",
			Execute:       utils.ValidateSupportedOS,
			StopOnFailure: true,
		},
		{
			Name:          "Manage User",
			Execute:       utils.ManageUser,
			StopOnFailure: true,
		},
		{
			Name: "Set environment",
			Execute: func(
				ctx context.Context,
				exec *devcontainer.Exec,
				gitspaceLogger gitspaceTypes.GitspaceLogger,
			) error {
				return utils.SetEnv(ctx, exec, gitspaceLogger, environment)
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
				return utils.InstallTools(ctx, exec, gitspaceConfig.IDE, gitspaceLogger)
			},
			StopOnFailure: true,
		},
		{
			Name:          "Install Git",
			Execute:       utils.InstallGit,
			StopOnFailure: true,
		},
		{
			Name: "Setup Git Credentials",
			Execute: func(
				ctx context.Context,
				exec *devcontainer.Exec,
				gitspaceLogger gitspaceTypes.GitspaceLogger,
			) error {
				if resolvedRepoDetails.ResolvedCredentials.UserPasswordCredentials != nil {
					return utils.SetupGitCredentials(ctx, exec, resolvedRepoDetails, gitspaceLogger)
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
				return utils.CloneCode(ctx, exec, resolvedRepoDetails, defaultBaseImage, gitspaceLogger)
			},
			StopOnFailure: true,
		},
		{
			Name: "Install AI agents",
			Execute: func(
				ctx context.Context,
				exec *devcontainer.Exec,
				gitspaceLogger gitspaceTypes.GitspaceLogger,
			) error {
				return utils.InstallAIAgents(ctx, exec, gitspaceLogger, gitspaceConfig.AIAgents)
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
				args := make(map[gitspaceTypes.IDEArg]any)
				args = AddIDECustomizationsArg(ideService, resolvedRepoDetails.DevcontainerConfig, args)
				args[gitspaceTypes.IDERepoNameArg] = resolvedRepoDetails.RepoName
				args = AddIDEDownloadURLArg(ideService, args)
				args = AddIDEDirNameArg(ideService, args)

				return ideService.Setup(ctx, exec, args, gitspaceLogger)
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
				args := make(map[gitspaceTypes.IDEArg]any)
				args[gitspaceTypes.IDERepoNameArg] = resolvedRepoDetails.RepoName
				args = AddIDEDirNameArg(ideService, args)
				return ideService.Run(ctx, exec, args, gitspaceLogger)
			},
			StopOnFailure: true,
		}}

	// Add the postCreateCommand lifecycle hooks to the steps
	for _, lifecycleHook := range lifecycleHookSteps[PostCreateAction] {
		steps = append(steps, step{
			Name: fmt.Sprintf("Execute postCreateCommand from %s", lifecycleHook.Source),
			Execute: func(
				ctx context.Context,
				exec *devcontainer.Exec,
				gitspaceLogger gitspaceTypes.GitspaceLogger,
			) error {
				return ExecuteLifecycleCommands(ctx, *exec, codeRepoDir, gitspaceLogger,
					lifecycleHook.Command.ToCommandArray(), PostCreateAction)
			},
			StopOnFailure: lifecycleHook.StopOnFailure,
		})
	}

	// Add the postStartCommand lifecycle hooks to the steps
	for _, lifecycleHook := range lifecycleHookSteps[PostStartAction] {
		steps = append(steps, step{
			Name: fmt.Sprintf("Execute postStartCommand from %s", lifecycleHook.Source),
			Execute: func(
				ctx context.Context,
				exec *devcontainer.Exec,
				gitspaceLogger gitspaceTypes.GitspaceLogger,
			) error {
				return ExecuteLifecycleCommands(ctx, *exec, codeRepoDir, gitspaceLogger,
					lifecycleHook.Command.ToCommandArray(), PostStartAction)
			},
			StopOnFailure: lifecycleHook.StopOnFailure,
		})
	}

	return steps
}

// setupGitspaceAndIDE initializes Gitspace and IdeType by registering and executing the setup steps.
func (e *EmbeddedDockerOrchestrator) setupGitspaceAndIDE(
	ctx context.Context,
	exec *devcontainer.Exec,
	gitspaceLogger gitspaceTypes.GitspaceLogger,
	ideService ide.IDE,
	gitspaceConfig types.GitspaceConfig,
	resolvedRepoDetails scm.ResolvedDetails,
	defaultBaseImage string,
	environment []string,
	lifecycleHookSteps map[PostAction][]*LifecycleHookStep,
) error {
	homeDir := GetUserHomeDir(exec.RemoteUser)
	codeRepoDir := filepath.Join(homeDir, resolvedRepoDetails.RepoName)

	steps := e.buildSetupSteps(
		ideService,
		gitspaceConfig,
		resolvedRepoDetails,
		defaultBaseImage,
		environment,
		codeRepoDir,
		lifecycleHookSteps,
	)

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

func getImage(devcontainerConfig types.DevcontainerConfig, defaultBaseImage string) string {
	imageName := devcontainerConfig.Image
	if imageName == "" {
		imageName = defaultBaseImage
	}
	return imageName
}

func (e *EmbeddedDockerOrchestrator) RetryCreateAndStartGitspaceIfRequired(_ context.Context) {
	// Nothing to do here as the event will be published from CreateAndStartGitspace itself.
}
