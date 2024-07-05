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
	"os"
	"path/filepath"

	"github.com/harness/gitness/infraprovider"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	dockerTypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/strslice"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/rs/zerolog/log"
)

var _ Orchestrator = (*EmbeddedDockerOrchestrator)(nil)

const (
	loggingKey             = "gitspace.container"
	sshPort                = "22/tcp"
	catchAllIP             = "0.0.0.0"
	catchAllPort           = "0"
	containerStateRunning  = "running"
	containerStateRemoved  = "removed"
	templateCloneGit       = "clone_git.sh"
	templateSetupSSHServer = "setup_ssh_server.sh"
)

type Config struct {
	DefaultBaseImage               string
	DefaultBindMountTargetPath     string
	DefaultBindMountSourceBasePath string
}

type EmbeddedDockerOrchestrator struct {
	dockerClientFactory *infraprovider.DockerClientFactory
	vsCodeService       *VSCode
	vsCodeWebService    *VSCodeWeb
	config              *Config
}

func NewEmbeddedDockerOrchestrator(
	dockerClientFactory *infraprovider.DockerClientFactory,
	vsCodeService *VSCode,
	vsCodeWebService *VSCodeWeb,
	config *Config,
) Orchestrator {
	return &EmbeddedDockerOrchestrator{
		dockerClientFactory: dockerClientFactory,
		vsCodeService:       vsCodeService,
		vsCodeWebService:    vsCodeWebService,
		config:              config,
	}
}

// StartGitspace checks if the Gitspace is already running by checking its entry in a map. If is it running,
// it returns, else, it creates a new Gitspace container by using the provided image. If the provided image is
// nil, it uses a default image read from Gitness config. Post creation it runs the postCreate command and clones
// the code inside the container. It uses the IDE service to setup the relevant IDE and also installs SSH server
// inside the container.
func (e *EmbeddedDockerOrchestrator) StartGitspace(
	ctx context.Context,
	gitspaceConfig *types.GitspaceConfig,
	devcontainerConfig *types.DevcontainerConfig,
	infra *infraprovider.Infrastructure,
) (map[enum.IDEType]string, error) {
	containerName := getGitspaceContainerName(gitspaceConfig)

	log := log.Ctx(ctx).With().Str(loggingKey, containerName).Logger()

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

	var usedPorts map[enum.IDEType]string

	switch state {
	case containerStateRunning:
		log.Debug().Msg("gitspace is already running")

		ideService, startErr := e.getIDEService(gitspaceConfig)
		if startErr != nil {
			return nil, startErr
		}

		ports, startErr := e.getUsedPorts(ctx, containerName, dockerClient, ideService)
		if startErr != nil {
			return nil, startErr
		}
		usedPorts = ports

	case containerStateRemoved:
		log.Debug().Msg("gitspace is not running, starting it...")

		ideService, startErr := e.getIDEService(gitspaceConfig)
		if startErr != nil {
			return nil, startErr
		}

		startErr = e.startGitspace(
			ctx,
			gitspaceConfig,
			devcontainerConfig,
			containerName,
			dockerClient,
			ideService,
		)
		if startErr != nil {
			return nil, fmt.Errorf("failed to start gitspace %s: %w", containerName, startErr)
		}
		ports, startErr := e.getUsedPorts(ctx, containerName, dockerClient, ideService)
		if startErr != nil {
			return nil, startErr
		}
		usedPorts = ports

		// TODO: Add gitspace status reporting.
		log.Debug().Msg("started gitspace")

	default:
		return nil, fmt.Errorf("gitspace %s is in a bad state: %s", containerName, state)
	}

	return usedPorts, nil
}

func (e *EmbeddedDockerOrchestrator) startGitspace(
	ctx context.Context,
	gitspaceConfig *types.GitspaceConfig,
	devcontainerConfig *types.DevcontainerConfig,
	containerName string,
	dockerClient *client.Client,
	ideService IDE,
) error {
	var imageName = devcontainerConfig.Image
	if imageName == "" {
		imageName = e.config.DefaultBaseImage
	}

	err := e.pullImage(ctx, imageName, dockerClient)
	if err != nil {
		return err
	}

	err = e.createContainer(ctx, gitspaceConfig, dockerClient, imageName, containerName, ideService)
	if err != nil {
		return err
	}

	var devcontainer = &Devcontainer{
		ContainerName: containerName,
		DockerClient:  dockerClient,
		WorkingDir:    e.config.DefaultBindMountTargetPath,
	}

	err = e.executePostCreateCommand(ctx, devcontainerConfig, devcontainer)
	if err != nil {
		return err
	}

	err = e.cloneCode(ctx, gitspaceConfig, devcontainerConfig, devcontainer)
	if err != nil {
		return err
	}

	err = e.setupSSHServer(ctx, gitspaceConfig.GitspaceInstance, devcontainer)
	if err != nil {
		return err
	}

	err = ideService.Setup(ctx, devcontainer, gitspaceConfig.GitspaceInstance)
	if err != nil {
		return fmt.Errorf("failed to setup IDE for gitspace %s: %w", containerName, err)
	}

	return nil
}

func (e *EmbeddedDockerOrchestrator) getUsedPorts(
	ctx context.Context,
	containerName string,
	dockerClient *client.Client,
	ideService IDE,
) (map[enum.IDEType]string, error) {
	inspectResp, err := dockerClient.ContainerInspect(ctx, containerName)
	if err != nil {
		return nil, fmt.Errorf("could not inspect container %s: %w", containerName, err)
	}

	usedPorts := map[enum.IDEType]string{}
	for port, bindings := range inspectResp.NetworkSettings.Ports {
		if port == sshPort {
			usedPorts[enum.IDETypeVSCode] = bindings[0].HostPort
		}
		if port == nat.Port(ideService.PortAndProtocol()) {
			usedPorts[ideService.Type()] = bindings[0].HostPort
		}
	}

	return usedPorts, nil
}

func (e *EmbeddedDockerOrchestrator) getIDEService(gitspaceConfig *types.GitspaceConfig) (IDE, error) {
	var ideService IDE

	switch gitspaceConfig.IDE {
	case enum.IDETypeVSCode:
		ideService = e.vsCodeService
	case enum.IDETypeVSCodeWeb:
		ideService = e.vsCodeWebService
	default:
		return nil, fmt.Errorf("unsupported IDE: %s", gitspaceConfig.IDE)
	}

	return ideService, nil
}

func (e *EmbeddedDockerOrchestrator) setupSSHServer(
	ctx context.Context,
	gitspaceInstance *types.GitspaceInstance,
	devcontainer *Devcontainer,
) error {
	sshServerScript, err := GenerateScriptFromTemplate(
		templateSetupSSHServer, &SetupSSHServerPayload{
			Username:         "harness",
			Password:         gitspaceInstance.AccessKey.String,
			WorkingDirectory: devcontainer.WorkingDir,
		})
	if err != nil {
		return fmt.Errorf(
			"failed to generate scipt to setup ssh server from template %s: %w", templateSetupSSHServer, err)
	}

	_, err = devcontainer.ExecuteCommand(ctx, sshServerScript, false)
	if err != nil {
		return fmt.Errorf("failed to setup SSH server: %w", err)
	}

	return nil
}

func (e *EmbeddedDockerOrchestrator) cloneCode(
	ctx context.Context,
	gitspaceConfig *types.GitspaceConfig,
	devcontainerConfig *types.DevcontainerConfig,
	devcontainer *Devcontainer,
) error {
	var devcontainerPresent = "true"
	if devcontainerConfig.Image == "" {
		devcontainerPresent = "false"
	}

	gitCloneScript, err := GenerateScriptFromTemplate(
		templateCloneGit, &CloneGitPayload{
			RepoURL:             gitspaceConfig.CodeRepoURL,
			DevcontainerPresent: devcontainerPresent,
			Image:               e.config.DefaultBaseImage,
		})
	if err != nil {
		return fmt.Errorf("failed to generate scipt to clone git from template %s: %w", templateCloneGit, err)
	}

	_, err = devcontainer.ExecuteCommand(ctx, gitCloneScript, false)
	if err != nil {
		return fmt.Errorf("failed to clone code: %w", err)
	}

	return nil
}

func (e *EmbeddedDockerOrchestrator) executePostCreateCommand(
	ctx context.Context,
	devcontainerConfig *types.DevcontainerConfig,
	devcontainer *Devcontainer,
) error {
	if devcontainerConfig.PostCreateCommand == "" {
		return nil
	}

	_, err := devcontainer.ExecuteCommand(ctx, devcontainerConfig.PostCreateCommand, false)
	if err != nil {
		return fmt.Errorf("post create command failed %q: %w", devcontainerConfig.PostCreateCommand, err)
	}

	return nil
}

func (e *EmbeddedDockerOrchestrator) createContainer(
	ctx context.Context,
	gitspaceConfig *types.GitspaceConfig,
	dockerClient *client.Client,
	imageName string,
	containerName string,
	ideService IDE,
) error {
	portUsedByIDE := ideService.PortAndProtocol()

	exposedPorts := nat.PortSet{
		sshPort: struct{}{},
	}

	hostPortBindings := []nat.PortBinding{
		{
			HostIP:   catchAllIP,
			HostPort: catchAllPort,
		},
	}

	portBindings := nat.PortMap{
		sshPort: hostPortBindings,
	}

	if portUsedByIDE != "" {
		natPort := nat.Port(portUsedByIDE)
		exposedPorts[natPort] = struct{}{}
		portBindings[natPort] = hostPortBindings
	}

	entryPoint := make(strslice.StrSlice, 0)
	entryPoint = append(entryPoint, "sleep")

	commands := make(strslice.StrSlice, 0)
	commands = append(commands, "infinity")

	bindMountSourcePath :=
		filepath.Join(
			e.config.DefaultBindMountSourceBasePath,
			gitspaceConfig.SpacePath,
			gitspaceConfig.Identifier,
		)
	err := os.MkdirAll(bindMountSourcePath, 0600)
	if err != nil {
		return fmt.Errorf(
			"could not create bind mount source path %s: %w", bindMountSourcePath, err)
	}

	resp2, err := dockerClient.ContainerCreate(ctx, &container.Config{
		Image:        imageName,
		Entrypoint:   entryPoint,
		Cmd:          commands,
		ExposedPorts: exposedPorts,
	}, &container.HostConfig{
		PortBindings: portBindings,
		Mounts: []mount.Mount{
			{
				Type:   mount.TypeBind,
				Source: bindMountSourcePath,
				Target: e.config.DefaultBindMountTargetPath,
			},
		},
	}, nil, containerName)
	if err != nil {
		return fmt.Errorf("could not create container %s: %w", containerName, err)
	}

	err = dockerClient.ContainerStart(ctx, resp2.ID, dockerTypes.ContainerStartOptions{})
	if err != nil {
		return fmt.Errorf("could not start container %s: %w", containerName, err)
	}

	return nil
}

func (e *EmbeddedDockerOrchestrator) pullImage(
	ctx context.Context,
	imageName string,
	dockerClient *client.Client,
) error {
	resp, err := dockerClient.ImagePull(ctx, imageName, dockerTypes.ImagePullOptions{})
	defer func() {
		closingErr := resp.Close()
		if closingErr != nil {
			log.Warn().Err(closingErr).Msg("failed to close image pull response")
		}
	}()
	if err != nil {
		return fmt.Errorf("could not pull image %s: %w", imageName, err)
	}

	return nil
}

// StopGitspace checks if the Gitspace container is running. If yes, it stops and removes the container.
// Else it returns.
func (e EmbeddedDockerOrchestrator) StopGitspace(
	ctx context.Context,
	gitspaceConfig *types.GitspaceConfig,
	infra *infraprovider.Infrastructure,
) error {
	containerName := getGitspaceContainerName(gitspaceConfig)

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
		log.Debug().Msg("gitspace is already stopped")
		return nil
	}

	log.Debug().Msg("stopping gitspace")
	err = e.stopGitspace(ctx, containerName, dockerClient)
	if err != nil {
		return fmt.Errorf("failed to stop gitspace %s: %w", containerName, err)
	}

	log.Debug().Msg("stopped gitspace")

	return nil
}

func (e EmbeddedDockerOrchestrator) stopGitspace(
	ctx context.Context,
	containerName string,
	dockerClient *client.Client,
) error {
	err := dockerClient.ContainerStop(ctx, containerName, nil)
	if err != nil {
		return fmt.Errorf("could not stop container %s: %w", containerName, err)
	}

	err = dockerClient.ContainerRemove(ctx, containerName, dockerTypes.ContainerRemoveOptions{Force: true})
	if err != nil {
		return fmt.Errorf("could not remove container %s: %w", containerName, err)
	}

	return nil
}

func getGitspaceContainerName(config *types.GitspaceConfig) string {
	return "gitspace-" + config.UserID + "-" + config.Identifier
}

// Status is NOOP for EmbeddedDockerOrchestrator as the docker host is verified by the infra provisioner.
func (e *EmbeddedDockerOrchestrator) Status(_ context.Context, _ *infraprovider.Infrastructure) error {
	return nil
}

func (e *EmbeddedDockerOrchestrator) containerState(
	ctx context.Context,
	containerName string,
	dockerClient *client.Client,
) (string, error) {
	var args = filters.NewArgs()
	args.Add("name", containerName)

	containers, err := dockerClient.ContainerList(ctx, dockerTypes.ContainerListOptions{All: true, Filters: args})
	if err != nil {
		return "", fmt.Errorf("could not list container %s: %w", containerName, err)
	}

	if len(containers) == 0 {
		return containerStateRemoved, nil
	}

	return containers[0].State, nil
}
