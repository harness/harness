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
	"io"
	"path/filepath"
	goruntime "runtime"
	"strconv"
	"strings"

	"github.com/harness/gitness/app/gitspace/logutil"
	"github.com/harness/gitness/app/gitspace/orchestrator/common"
	"github.com/harness/gitness/app/gitspace/orchestrator/devcontainer"
	"github.com/harness/gitness/app/gitspace/orchestrator/git"
	"github.com/harness/gitness/app/gitspace/orchestrator/ide"
	"github.com/harness/gitness/app/gitspace/orchestrator/user"
	"github.com/harness/gitness/app/gitspace/scm"
	"github.com/harness/gitness/infraprovider"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/rs/zerolog/log"
)

var _ Orchestrator = (*EmbeddedDockerOrchestrator)(nil)

const (
	loggingKey            = "gitspace.container"
	catchAllIP            = "0.0.0.0"
	containerStateRunning = "running"
	containerStateRemoved = "removed"
	containerStateStopped = "exited"
	mountType             = mount.TypeVolume
)

type EmbeddedDockerOrchestrator struct {
	dockerClientFactory *infraprovider.DockerClientFactory
	statefulLogger      *logutil.StatefulLogger
	gitService          git.Service
	userService         user.Service
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

	log := log.Ctx(ctx).With().Str(loggingKey, containerName).Logger()

	var accessKey string
	if gitspaceConfig.GitspaceInstance != nil &&
		gitspaceConfig.GitspaceInstance.AccessKey != nil {
		accessKey = *gitspaceConfig.GitspaceInstance.AccessKey
	} else {
		return nil, fmt.Errorf("no access key is configured: %s", gitspaceConfig.Identifier)
	}

	dockerClient, err := e.dockerClientFactory.NewDockerClient(ctx, infra)
	if err != nil {
		return nil, fmt.Errorf("error getting docker client from docker client factory: %w", err)
	}

	defer func() {
		closingErr := dockerClient.Close()
		if closingErr != nil {
			log.Warn().Err(closingErr).Msg("failed to close docker client")
		}
	}()

	log.Debug().Msg("checking current state of gitspace")
	state, err := e.containerState(ctx, containerName, dockerClient)
	if err != nil {
		return nil, err
	}

	homeDir := GetUserHomeDir(gitspaceConfig.GitspaceUser.Identifier)
	codeRepoDir := filepath.Join(homeDir, resolvedRepoDetails.RepoName)

	switch state {
	case containerStateRunning:
		log.Debug().Msg("gitspace is already running")

	case containerStateStopped:
		log.Debug().Msg("gitspace is stopped, starting it")

		logStreamInstance, loggerErr := e.statefulLogger.CreateLogStream(ctx, gitspaceConfig.ID)
		if loggerErr != nil {
			return nil, fmt.Errorf("error getting log stream for gitspace ID %d: %w", gitspaceConfig.ID, loggerErr)
		}

		defer func() {
			loggerErr = logStreamInstance.Flush()
			if loggerErr != nil {
				log.Warn().Err(loggerErr).Msgf("failed to flush log stream for gitspace ID %d", gitspaceConfig.ID)
			}
		}()

		startErr := e.startContainer(ctx, dockerClient, containerName, logStreamInstance)
		if startErr != nil {
			return nil, startErr
		}

		exec := &devcontainer.Exec{
			ContainerName:  containerName,
			DockerClient:   dockerClient,
			HomeDir:        homeDir,
			UserIdentifier: gitspaceConfig.GitspaceUser.Identifier,
			AccessKey:      accessKey,
			AccessType:     gitspaceConfig.GitspaceInstance.AccessType,
		}

		if resolvedRepoDetails.Credentials != nil {
			authErr := e.setupGitCredentials(ctx, exec, resolvedRepoDetails, logStreamInstance)
			if authErr != nil {
				return nil, authErr
			}
		}

		err = e.runIDE(ctx, exec, ideService, logStreamInstance)
		if err != nil {
			return nil, err
		}

		// TODO: Add gitspace status reporting.
		log.Debug().Msg("started gitspace")

	case containerStateRemoved:
		log.Debug().Msg("gitspace is removed, creating it...")

		logStreamInstance, loggerErr := e.statefulLogger.CreateLogStream(ctx, gitspaceConfig.ID)
		if loggerErr != nil {
			return nil,
				fmt.Errorf("error getting log stream for gitspace ID %d: %w", gitspaceConfig.ID, loggerErr)
		}

		defer func() {
			loggerErr = logStreamInstance.Flush()
			if loggerErr != nil {
				log.Warn().Err(loggerErr).Msgf("failed to flush log stream for gitspace ID %d", gitspaceConfig.ID)
			}
		}()

		startErr := e.startGitspace(
			ctx,
			gitspaceConfig,
			containerName,
			dockerClient,
			ideService,
			infra.Storage,
			resolvedRepoDetails,
			infra.GitspacePortMappings,
			defaultBaseImage,
			homeDir,
			codeRepoDir,
			logStreamInstance,
			accessKey,
		)
		if startErr != nil {
			return nil, fmt.Errorf("failed to start gitspace %s: %w", containerName, startErr)
		}

		// TODO: Add gitspace status reporting.
		log.Debug().Msg("started gitspace")

	default:
		return nil, fmt.Errorf("gitspace %s is in a bad state: %s", containerName, state)
	}

	id, ports, startErr := e.getContainerInfo(ctx, containerName, dockerClient, infra.GitspacePortMappings)
	if startErr != nil {
		return nil, startErr
	}

	return &StartResponse{
		ContainerID:      id,
		ContainerName:    containerName,
		PublishedPorts:   ports,
		AbsoluteRepoPath: codeRepoDir,
	}, nil
}

func (e *EmbeddedDockerOrchestrator) startGitspace(
	ctx context.Context,
	gitspaceConfig types.GitspaceConfig,
	containerName string,
	dockerClient *client.Client,
	ideService ide.IDE,
	volumeName string,
	resolvedRepoDetails scm.ResolvedDetails,
	portMappings map[int]*types.PortMapping,
	defaultBaseImage string,
	homeDir string,
	codeRepoDir string,
	logStreamInstance *logutil.LogStreamInstance,
	accessKey string,
) error {
	var imageName = resolvedRepoDetails.DevcontainerConfig.Image
	if imageName == "" {
		imageName = defaultBaseImage
	}

	err := e.pullImage(ctx, imageName, dockerClient, logStreamInstance)
	if err != nil {
		return err
	}

	err = e.createContainer(
		ctx,
		dockerClient,
		imageName,
		containerName,
		logStreamInstance,
		volumeName,
		homeDir,
		portMappings,
	)
	if err != nil {
		return err
	}

	err = e.startContainer(ctx, dockerClient, containerName, logStreamInstance)
	if err != nil {
		return err
	}

	var exec = &devcontainer.Exec{
		ContainerName:  containerName,
		DockerClient:   dockerClient,
		HomeDir:        homeDir,
		UserIdentifier: gitspaceConfig.GitspaceUser.Identifier,
		AccessKey:      accessKey,
		AccessType:     gitspaceConfig.GitspaceInstance.AccessType,
	}

	err = e.validateSupportedOS(ctx, exec, logStreamInstance)
	if err != nil {
		return err
	}

	err = e.manageUser(ctx, exec, logStreamInstance)
	if err != nil {
		return err
	}

	err = e.installTools(ctx, exec, logStreamInstance, gitspaceConfig.IDE)
	if err != nil {
		return err
	}

	err = e.setupIDE(ctx, exec, ideService, logStreamInstance)
	if err != nil {
		return err
	}

	err = e.runIDE(ctx, exec, ideService, logStreamInstance)
	if err != nil {
		return err
	}

	err = e.installGit(ctx, exec, logStreamInstance)
	if err != nil {
		return err
	}

	if resolvedRepoDetails.ResolvedCredentials.Credentials != nil {
		err = e.setupGitCredentials(ctx, exec, resolvedRepoDetails, logStreamInstance)
		if err != nil {
			return err
		}
	}

	err = e.cloneCode(ctx, exec, defaultBaseImage, resolvedRepoDetails, logStreamInstance)
	if err != nil {
		return err
	}

	err = e.executePostCreateCommand(ctx, resolvedRepoDetails.DevcontainerConfig, exec, codeRepoDir, logStreamInstance)
	if err != nil {
		return err
	}

	return nil
}

// TODO: Instead of explicitly running IDE related processes, we can explore service to run the service on boot.

func (e *EmbeddedDockerOrchestrator) runIDE(
	ctx context.Context,
	exec *devcontainer.Exec,
	ideService ide.IDE,
	logStreamInstance *logutil.LogStreamInstance,
) error {
	loggingErr := logStreamInstance.Write("Running the IDE inside container: " + string(ideService.Type()))
	if loggingErr != nil {
		return fmt.Errorf("logging error: %w", loggingErr)
	}

	output, err := ideService.Run(ctx, exec)
	if err != nil {
		loggingErr = logStreamInstance.Write("Error while running IDE inside container: " + err.Error())

		err = fmt.Errorf("failed to run the IDE for gitspace %s: %w", exec.ContainerName, err)

		if loggingErr != nil {
			err = fmt.Errorf("original error: %w; logging error: %w", err, loggingErr)
		}

		return err
	}

	loggingErr = logStreamInstance.Write("IDE run output...\n" + string(output))
	if loggingErr != nil {
		return fmt.Errorf("logging error: %w", loggingErr)
	}

	loggingErr = logStreamInstance.Write("Successfully run the IDE inside container")
	if loggingErr != nil {
		return fmt.Errorf("logging error: %w", loggingErr)
	}

	return nil
}

func (e *EmbeddedDockerOrchestrator) setupIDE(
	ctx context.Context,
	exec *devcontainer.Exec,
	ideService ide.IDE,
	logStreamInstance *logutil.LogStreamInstance,
) error {
	loggingErr := logStreamInstance.Write("Setting up IDE inside container: " + string(ideService.Type()))
	if loggingErr != nil {
		return fmt.Errorf("logging error: %w", loggingErr)
	}

	output, err := ideService.Setup(ctx, exec)
	if err != nil {
		loggingErr = logStreamInstance.Write("Error while setting up IDE inside container: " + err.Error())

		err = fmt.Errorf("failed to setup IDE for gitspace %s: %w", exec.ContainerName, err)

		if loggingErr != nil {
			err = fmt.Errorf("original error: %w; logging error: %w", err, loggingErr)
		}

		return err
	}

	loggingErr = logStreamInstance.Write("IDE setup output...\n" + string(output))
	if loggingErr != nil {
		return fmt.Errorf("logging error: %w", loggingErr)
	}

	loggingErr = logStreamInstance.Write("Successfully set up IDE inside container")
	if loggingErr != nil {
		return fmt.Errorf("logging error: %w", loggingErr)
	}

	return nil
}

func (e *EmbeddedDockerOrchestrator) getContainerInfo(
	ctx context.Context,
	containerName string,
	dockerClient *client.Client,
	portMappings map[int]*types.PortMapping,
) (string, map[int]string, error) {
	inspectResp, err := dockerClient.ContainerInspect(ctx, containerName)
	if err != nil {
		return "", nil, fmt.Errorf("could not inspect container %s: %w", containerName, err)
	}

	var usedPorts = make(map[int]string)
	for portAndProtocol, bindings := range inspectResp.NetworkSettings.Ports {
		portRaw := strings.Split(string(portAndProtocol), "/")[0]
		port, conversionErr := strconv.Atoi(portRaw)
		if conversionErr != nil {
			return "", nil, fmt.Errorf("could not convert port %s to int: %w", portRaw, err)
		}

		if portMappings[port] != nil {
			usedPorts[port] = bindings[0].HostPort
		}
	}

	return inspectResp.ID, usedPorts, nil
}

func (e *EmbeddedDockerOrchestrator) manageUser(
	ctx context.Context,
	exec *devcontainer.Exec,
	logStreamInstance *logutil.LogStreamInstance,
) error {
	output, err := e.userService.Manage(ctx, exec)
	if err != nil {
		loggingErr := logStreamInstance.Write("Error while creating user inside container: " + err.Error())

		err = fmt.Errorf("failed to create user gitspace %s: %w", exec.ContainerName, err)

		if loggingErr != nil {
			err = fmt.Errorf("original error: %w; logging error: %w", err, loggingErr)
		}

		return err
	}

	loggingErr := logStreamInstance.Write("Managing user output...\n" + string(output))
	if loggingErr != nil {
		return fmt.Errorf("logging error: %w", loggingErr)
	}

	return nil
}

func (e *EmbeddedDockerOrchestrator) installGit(
	ctx context.Context,
	exec *devcontainer.Exec,
	logStreamInstance *logutil.LogStreamInstance,
) error {
	output, err := e.gitService.Install(ctx, exec)
	if err != nil {
		loggingErr := logStreamInstance.Write("Error while installing git inside container: " + err.Error())

		err = fmt.Errorf("failed to install git for gitspace %s: %w", exec.ContainerName, err)

		if loggingErr != nil {
			err = fmt.Errorf("original error: %w; logging error: %w", err, loggingErr)
		}

		return err
	}

	loggingErr := logStreamInstance.Write("Install git output...\n" + string(output))
	if loggingErr != nil {
		return fmt.Errorf("logging error: %w", loggingErr)
	}

	return nil
}
func (e *EmbeddedDockerOrchestrator) setupGitCredentials(
	ctx context.Context,
	exec *devcontainer.Exec,
	resolvedRepoDetails scm.ResolvedDetails,
	logStreamInstance *logutil.LogStreamInstance,
) error {
	output, err := e.gitService.SetupCredentials(ctx, exec, resolvedRepoDetails)
	if err != nil {
		loggingErr := logStreamInstance.Write("Error while setting up git credentials inside container: " + err.Error())

		err = fmt.Errorf("failed to setup git credentials for gitspace %s: %w", exec.ContainerName, err)

		if loggingErr != nil {
			err = fmt.Errorf("original error: %w; logging error: %w", err, loggingErr)
		}

		return err
	}

	loggingErr := logStreamInstance.Write("Setting up git credentials output...\n" + string(output))
	if loggingErr != nil {
		return fmt.Errorf("logging error: %w", loggingErr)
	}

	return nil
}

func (e *EmbeddedDockerOrchestrator) cloneCode(
	ctx context.Context,
	exec *devcontainer.Exec,
	defaultBaseImage string,
	resolvedRepoDetails scm.ResolvedDetails,
	logStreamInstance *logutil.LogStreamInstance,
) error {
	output, err := e.gitService.CloneCode(ctx, exec, resolvedRepoDetails, defaultBaseImage)
	if err != nil {
		loggingErr := logStreamInstance.Write("Error while cloning code inside container: " + err.Error())

		err = fmt.Errorf("failed to clone code gitspace %s: %w", exec.ContainerName, err)

		if loggingErr != nil {
			err = fmt.Errorf("original error: %w; logging error: %w", err, loggingErr)
		}

		return err
	}

	loggingErr := logStreamInstance.Write("Clone output...\n" + string(output))
	if loggingErr != nil {
		return fmt.Errorf("logging error: %w", loggingErr)
	}

	return nil
}

func (e *EmbeddedDockerOrchestrator) executePostCreateCommand(
	ctx context.Context,
	devcontainerConfig types.DevcontainerConfig,
	exec *devcontainer.Exec,
	codeRepoDir string,
	logStreamInstance *logutil.LogStreamInstance,
) error {
	if devcontainerConfig.PostCreateCommand == "" {
		loggingErr := logStreamInstance.Write("No post-create command provided, skipping execution")
		if loggingErr != nil {
			return fmt.Errorf("logging error: %w", loggingErr)
		}

		return nil
	}

	loggingErr := logStreamInstance.Write("Executing postCreate command: " + devcontainerConfig.PostCreateCommand)
	if loggingErr != nil {
		return fmt.Errorf("logging error: %w", loggingErr)
	}

	output, err := exec.ExecuteCommand(ctx, devcontainerConfig.PostCreateCommand, true, false, codeRepoDir)
	if err != nil {
		loggingErr = logStreamInstance.Write("Error while executing postCreate command" + err.Error())

		err = fmt.Errorf("failed to execute postCreate command %q: %w", devcontainerConfig.PostCreateCommand, err)

		if loggingErr != nil {
			err = fmt.Errorf("original error: %w; logging error: %w", err, loggingErr)
		}

		return err
	}

	loggingErr = logStreamInstance.Write("Post create command execution output...\n" + string(output))
	if loggingErr != nil {
		return fmt.Errorf("logging error: %w", loggingErr)
	}

	loggingErr = logStreamInstance.Write("Successfully executed postCreate command")
	if loggingErr != nil {
		return fmt.Errorf("logging error: %w", loggingErr)
	}

	return nil
}

func (e *EmbeddedDockerOrchestrator) startContainer(
	ctx context.Context,
	dockerClient *client.Client,
	containerName string,
	logStreamInstance *logutil.LogStreamInstance,
) error {
	loggingErr := logStreamInstance.Write("Starting container: " + containerName)
	if loggingErr != nil {
		return fmt.Errorf("logging error: %w", loggingErr)
	}

	err := dockerClient.ContainerStart(ctx, containerName, container.StartOptions{})
	if err != nil {
		loggingErr = logStreamInstance.Write("Error while creating container: " + err.Error())

		err = fmt.Errorf("could not start container %s: %w", containerName, err)

		if loggingErr != nil {
			err = fmt.Errorf("original error: %w; logging error: %w", err, loggingErr)
		}

		return err
	}

	loggingErr = logStreamInstance.Write("Successfully started container")
	if loggingErr != nil {
		return fmt.Errorf("logging error: %w", loggingErr)
	}

	return nil
}

func (e *EmbeddedDockerOrchestrator) createContainer(
	ctx context.Context,
	dockerClient *client.Client,
	imageName string,
	containerName string,
	logStreamInstance *logutil.LogStreamInstance,
	volumeName string,
	homeDir string,
	portMappings map[int]*types.PortMapping,
) error {
	exposedPorts := nat.PortSet{}
	portBindings := nat.PortMap{}

	for port, mapping := range portMappings {
		natPort := nat.Port(strconv.Itoa(port) + "/tcp")
		hostPortBindings := []nat.PortBinding{
			{
				HostIP:   catchAllIP,
				HostPort: strconv.Itoa(mapping.PublishedPort),
			},
		}
		exposedPorts[natPort] = struct{}{}
		portBindings[natPort] = hostPortBindings
	}

	loggingErr := logStreamInstance.Write("Creating container: " + containerName)
	if loggingErr != nil {
		return fmt.Errorf("logging error: %w", loggingErr)
	}

	hostConfig := &container.HostConfig{
		PortBindings: portBindings,
		Mounts: []mount.Mount{
			{
				Type:   mountType,
				Source: volumeName,
				Target: homeDir,
			},
		},
	}

	if goruntime.GOOS == "linux" {
		extraHosts := []string{"host.docker.internal:host-gateway"}
		hostConfig.ExtraHosts = extraHosts
	}

	containerConfig := &container.Config{
		Image:        imageName,
		Entrypoint:   []string{"/bin/sh"},
		Cmd:          []string{"-c", "trap 'exit 0' 15; sleep infinity & wait $!"},
		ExposedPorts: exposedPorts,
	}
	_, err := dockerClient.ContainerCreate(ctx, containerConfig,
		hostConfig, nil, nil, containerName)

	if err != nil {
		loggingErr = logStreamInstance.Write("Error while creating container: " + err.Error())

		err = fmt.Errorf("could not create container %s: %w", containerName, err)

		if loggingErr != nil {
			err = fmt.Errorf("original error: %w; logging error: %w", err, loggingErr)
		}

		return err
	}

	return nil
}

func (e *EmbeddedDockerOrchestrator) pullImage(
	ctx context.Context,
	imageName string,
	dockerClient *client.Client,
	logStreamInstance *logutil.LogStreamInstance,
) error {
	loggingErr := logStreamInstance.Write("Pulling image: " + imageName)
	if loggingErr != nil {
		return fmt.Errorf("logging error: %w", loggingErr)
	}

	pullResponse, err := dockerClient.ImagePull(ctx, imageName, image.PullOptions{})

	defer func() {
		closingErr := pullResponse.Close()
		if closingErr != nil {
			log.Warn().Err(closingErr).Msg("failed to close image pull response")
		}
	}()

	if err != nil {
		loggingErr = logStreamInstance.Write("Error while pulling image: " + err.Error())

		err = fmt.Errorf("could not pull image %s: %w", imageName, err)

		if loggingErr != nil {
			err = fmt.Errorf("original error: %w; logging error: %w", err, loggingErr)
		}

		return err
	}

	// NOTE: It is necessary to read all the data in pullResponse to ensure the image has been completely downloaded.
	// If the execution proceeds before the response is completed, the container will not find the required image.
	output, err := io.ReadAll(pullResponse)
	if err != nil {
		loggingErr = logStreamInstance.Write("Error while parsing image pull response: " + err.Error())

		err = fmt.Errorf("error while parsing pull image output %s: %w", imageName, err)

		if loggingErr != nil {
			err = fmt.Errorf("original error: %w; logging error: %w", err, loggingErr)
		}

		return err
	}

	loggingErr = logStreamInstance.Write(string(output))
	if loggingErr != nil {
		return fmt.Errorf("logging error: %w", loggingErr)
	}

	loggingErr = logStreamInstance.Write("Successfully pulled image")
	if loggingErr != nil {
		return fmt.Errorf("logging error: %w", loggingErr)
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

	log := log.Ctx(ctx).With().Str(loggingKey, containerName).Logger()

	dockerClient, err := e.dockerClientFactory.NewDockerClient(ctx, infra)
	if err != nil {
		return fmt.Errorf("error getting docker client from docker client factory: %w", err)
	}

	defer func() {
		closingErr := dockerClient.Close()
		if closingErr != nil {
			log.Warn().Err(closingErr).Msg("failed to close docker client")
		}
	}()

	log.Debug().Msg("checking current state of gitspace")
	state, err := e.containerState(ctx, containerName, dockerClient)
	if err != nil {
		return err
	}

	if state == containerStateRemoved {
		return fmt.Errorf("gitspace %s is removed", containerName)
	}

	if state == containerStateStopped {
		log.Debug().Msg("gitspace is already stopped")
		return nil
	}

	log.Debug().Msg("stopping gitspace")

	logStreamInstance, loggerErr := e.statefulLogger.CreateLogStream(ctx, gitspaceConfig.ID)
	if loggerErr != nil {
		return fmt.Errorf("error getting log stream for gitspace ID %d: %w", gitspaceConfig.ID, loggerErr)
	}

	defer func() {
		loggerErr = logStreamInstance.Flush()
		if loggerErr != nil {
			log.Warn().Err(loggerErr).Msgf("failed to flush log stream for gitspace ID %d", gitspaceConfig.ID)
		}
	}()

	err = e.stopContainer(ctx, containerName, dockerClient, logStreamInstance)
	if err != nil {
		return fmt.Errorf("failed to stop gitspace %s: %w", containerName, err)
	}

	log.Debug().Msg("stopped gitspace")

	return nil
}

func (e *EmbeddedDockerOrchestrator) stopContainer(
	ctx context.Context,
	containerName string,
	dockerClient *client.Client,
	logStreamInstance *logutil.LogStreamInstance,
) error {
	loggingErr := logStreamInstance.Write("Stopping container: " + containerName)
	if loggingErr != nil {
		return fmt.Errorf("logging error: %w", loggingErr)
	}

	err := dockerClient.ContainerStop(ctx, containerName, container.StopOptions{})
	if err != nil {
		loggingErr = logStreamInstance.Write("Error while stopping container: " + err.Error())

		err = fmt.Errorf("could not stop container %s: %w", containerName, err)

		if loggingErr != nil {
			err = fmt.Errorf("original error: %w; logging error: %w", err, loggingErr)
		}

		return err
	}

	loggingErr = logStreamInstance.Write("Successfully stopped container")
	if loggingErr != nil {
		return fmt.Errorf("logging error: %w", loggingErr)
	}

	return nil
}

// Status is NOOP for EmbeddedDockerOrchestrator as the docker host is verified by the infra provisioner.
func (e *EmbeddedDockerOrchestrator) Status(_ context.Context, _ types.Infrastructure) error {
	return nil
}

func (e *EmbeddedDockerOrchestrator) containerState(
	ctx context.Context,
	containerName string,
	dockerClient *client.Client,
) (string, error) {
	var args = filters.NewArgs()
	args.Add("name", containerName)

	containers, err := dockerClient.ContainerList(ctx, container.ListOptions{All: true, Filters: args})
	if err != nil {
		return "", fmt.Errorf("could not list container %s: %w", containerName, err)
	}

	if len(containers) == 0 {
		return containerStateRemoved, nil
	}

	return containers[0].State, nil
}

// StopAndRemoveGitspace stops the container if not stopped and removes it.
// If the container is already removed, it returns.
func (e *EmbeddedDockerOrchestrator) StopAndRemoveGitspace(
	ctx context.Context,
	gitspaceConfig types.GitspaceConfig,
	infra types.Infrastructure,
) error {
	containerName := GetGitspaceContainerName(gitspaceConfig)

	log := log.Ctx(ctx).With().Str(loggingKey, containerName).Logger()

	dockerClient, err := e.dockerClientFactory.NewDockerClient(ctx, infra)
	if err != nil {
		return fmt.Errorf("error getting docker client from docker client factory: %w", err)
	}

	defer func() {
		closingErr := dockerClient.Close()
		if closingErr != nil {
			log.Warn().Err(closingErr).Msg("failed to close docker client")
		}
	}()

	log.Debug().Msg("checking current state of gitspace")
	state, err := e.containerState(ctx, containerName, dockerClient)
	if err != nil {
		return err
	}

	if state == containerStateRemoved {
		log.Debug().Msg("gitspace is already removed")
		return nil
	}

	logStreamInstance, loggerErr := e.statefulLogger.CreateLogStream(ctx, gitspaceConfig.ID)
	if loggerErr != nil {
		return fmt.Errorf("error getting log stream for gitspace ID %d: %w", gitspaceConfig.ID, loggerErr)
	}

	defer func() {
		loggerErr = logStreamInstance.Flush()
		if loggerErr != nil {
			log.Warn().Err(loggerErr).Msgf("failed to flush log stream for gitspace ID %d", gitspaceConfig.ID)
		}
	}()

	if state != containerStateStopped {
		log.Debug().Msg("stopping gitspace")

		err = e.stopContainer(ctx, containerName, dockerClient, logStreamInstance)
		if err != nil {
			return fmt.Errorf("failed to stop gitspace %s: %w", containerName, err)
		}

		log.Debug().Msg("stopped gitspace")
	}

	log.Debug().Msg("removing gitspace")

	err = e.removeContainer(ctx, containerName, dockerClient, logStreamInstance)
	if err != nil {
		return fmt.Errorf("failed to remove gitspace %s: %w", containerName, err)
	}

	log.Debug().Msg("removed gitspace")

	return nil
}

func (e *EmbeddedDockerOrchestrator) removeContainer(
	ctx context.Context,
	containerName string,
	dockerClient *client.Client,
	logStreamInstance *logutil.LogStreamInstance,
) error {
	loggingErr := logStreamInstance.Write("Removing container: " + containerName)
	if loggingErr != nil {
		return fmt.Errorf("logging error: %w", loggingErr)
	}

	err := dockerClient.ContainerRemove(ctx, containerName, container.RemoveOptions{Force: true})
	if err != nil {
		loggingErr = logStreamInstance.Write("Error while removing container: " + err.Error())

		err = fmt.Errorf("could not remove container %s: %w", containerName, err)

		if loggingErr != nil {
			err = fmt.Errorf("original error: %w; logging error: %w", err, loggingErr)
		}

		return err
	}

	loggingErr = logStreamInstance.Write("Successfully removed container")
	if loggingErr != nil {
		return fmt.Errorf("logging error: %w", loggingErr)
	}

	return nil
}

func (e *EmbeddedDockerOrchestrator) validateSupportedOS(
	ctx context.Context,
	exec *devcontainer.Exec,
	logStreamInstance *logutil.LogStreamInstance,
) error {
	output, err := common.ValidateSupportedOS(ctx, exec)
	if err != nil {
		loggingErr := logStreamInstance.Write("Error while detecting os inside container: " + err.Error())

		err = fmt.Errorf("failed to detect os in %s: %w", exec.ContainerName, err)

		if loggingErr != nil {
			err = fmt.Errorf("original error: %w; logging error: %w", err, loggingErr)
		}

		return err
	}

	loggingErr := logStreamInstance.Write("Validate supported OSes...\n" + string(output))
	if loggingErr != nil {
		return fmt.Errorf("logging error: %w", loggingErr)
	}

	return nil
}

func (e *EmbeddedDockerOrchestrator) installTools(
	ctx context.Context,
	exec *devcontainer.Exec,
	logStreamInstance *logutil.LogStreamInstance,
	ideType enum.IDEType,
) error {
	output, err := common.InstallTools(ctx, exec, ideType)
	if err != nil {
		loggingErr := logStreamInstance.Write("Error while installing tools inside container: " + err.Error())

		err = fmt.Errorf("failed to install tools in %s: %w", exec.ContainerName, err)

		if loggingErr != nil {
			err = fmt.Errorf("original error: %w; logging error: %w", err, loggingErr)
		}

		return err
	}

	loggingErr := logStreamInstance.Write("Tools installation output...\n" + string(output))
	if loggingErr != nil {
		return fmt.Errorf("logging error: %w", loggingErr)
	}

	return nil
}

func (e *EmbeddedDockerOrchestrator) StreamLogs(
	_ context.Context,
	_ types.GitspaceConfig,
	_ types.Infrastructure) (string, error) {
	return "", fmt.Errorf("not implemented")
}
