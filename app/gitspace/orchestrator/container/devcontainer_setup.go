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
	goruntime "runtime"
	"strconv"
	"strings"

	gitspaceTypes "github.com/harness/gitness/app/gitspace/types"
	"github.com/harness/gitness/types"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/rs/zerolog/log"
)

const (
	catchAllIP = "0.0.0.0"
)

var containerStateMapping = map[string]State{
	"running": ContainerStateRunning,
	"exited":  ContainerStateStopped,
	"dead":    ContainerStateDead,
	"created": ContainerStateCreated,
	"paused":  ContainerStatePaused,
}

// Helper function to log messages and handle error wrapping.
func logStreamWrapError(gitspaceLogger gitspaceTypes.GitspaceLogger, msg string, err error) error {
	gitspaceLogger.Error(msg, err)
	return fmt.Errorf("%s: %w", msg, err)
}

// Generalized Docker Container Management.
func ManageContainer(
	ctx context.Context,
	action Action,
	containerName string,
	dockerClient *client.Client,
	gitspaceLogger gitspaceTypes.GitspaceLogger,
) error {
	var err error
	switch action {
	case ContainerActionStop:
		err = dockerClient.ContainerStop(ctx, containerName, container.StopOptions{})
		if err != nil {
			return logStreamWrapError(gitspaceLogger, "Error while stopping container", err)
		}
		gitspaceLogger.Info("Successfully stopped container")

	case ContainerActionStart:
		err = dockerClient.ContainerStart(ctx, containerName, container.StartOptions{})
		if err != nil {
			return logStreamWrapError(gitspaceLogger, "Error while starting container", err)
		}
		gitspaceLogger.Info("Successfully started container")
	case ContainerActionRemove:
		err = dockerClient.ContainerRemove(ctx, containerName, container.RemoveOptions{Force: true})
		if err != nil {
			return logStreamWrapError(gitspaceLogger, "Error while removing container", err)
		}
		gitspaceLogger.Info("Successfully removed container")
	default:
		return fmt.Errorf("unknown action: %s", action)
	}
	return nil
}

func FetchContainerState(
	ctx context.Context,
	containerName string,
	dockerClient *client.Client,
) (State, error) {
	args := filters.NewArgs()
	args.Add("name", containerName)

	containers, err := dockerClient.ContainerList(ctx, container.ListOptions{All: true, Filters: args})
	if err != nil {
		return "", fmt.Errorf("could not list container %s: %w", containerName, err)
	}

	if len(containers) == 0 {
		return ContainerStateRemoved, nil
	}
	containerState := ContainerStateUnknown
	for _, value := range containers {
		name, _ := strings.CutPrefix(value.Names[0], "/")
		if name == containerName {
			if state, ok := containerStateMapping[value.State]; ok {
				containerState = state
			}
			break
		}
	}

	return containerState, nil
}

// Create a new Docker container.
func CreateContainer(
	ctx context.Context,
	dockerClient *client.Client,
	imageName string,
	containerName string,
	gitspaceLogger gitspaceTypes.GitspaceLogger,
	bindMountSource string,
	bindMountTarget string,
	mountType mount.Type,
	portMappings map[int]*types.PortMapping,
	env []string,
) error {
	exposedPorts, portBindings := applyPortMappings(portMappings)

	gitspaceLogger.Info("Creating container: " + containerName)

	hostConfig := prepareHostConfig(bindMountSource, bindMountTarget, mountType, portBindings)

	// Create the container
	containerConfig := &container.Config{
		Image:        imageName,
		Env:          env,
		Entrypoint:   []string{"/bin/sh"},
		Cmd:          []string{"-c", "trap 'exit 0' 15; sleep infinity & wait $!"},
		ExposedPorts: exposedPorts,
	}
	_, err := dockerClient.ContainerCreate(ctx, containerConfig, hostConfig, nil, nil, containerName)
	if err != nil {
		return logStreamWrapError(gitspaceLogger, "Error while creating container", err)
	}

	return nil
}

// Prepare port mappings for container creation.
func applyPortMappings(portMappings map[int]*types.PortMapping) (nat.PortSet, nat.PortMap) {
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
	return exposedPorts, portBindings
}

// Prepare the host configuration for container creation.
func prepareHostConfig(
	bindMountSource string,
	bindMountTarget string,
	mountType mount.Type,
	portBindings nat.PortMap,
) *container.HostConfig {
	hostConfig := &container.HostConfig{
		PortBindings: portBindings,
		Mounts: []mount.Mount{
			{
				Type:   mountType,
				Source: bindMountSource,
				Target: bindMountTarget,
			},
		},
	}

	if goruntime.GOOS == "linux" {
		hostConfig.ExtraHosts = []string{"host.docker.internal:host-gateway"}
	}

	return hostConfig
}
func GetContainerInfo(
	ctx context.Context,
	containerName string,
	dockerClient *client.Client,
	portMappings map[int]*types.PortMapping,
) (string, map[int]string, error) {
	inspectResp, err := dockerClient.ContainerInspect(ctx, containerName)
	if err != nil {
		return "", nil, fmt.Errorf("could not inspect container %s: %w", containerName, err)
	}

	usedPorts := make(map[int]string)
	for portAndProtocol, bindings := range inspectResp.NetworkSettings.Ports {
		portRaw := strings.Split(string(portAndProtocol), "/")[0]
		port, conversionErr := strconv.Atoi(portRaw)
		if conversionErr != nil {
			return "", nil, fmt.Errorf("could not convert port %s to int: %w", portRaw, conversionErr)
		}

		if portMappings[port] != nil {
			usedPorts[port] = bindings[0].HostPort
		}
	}

	return inspectResp.ID, usedPorts, nil
}

func PullImage(
	ctx context.Context,
	imageName string,
	dockerClient *client.Client,
	gitspaceLogger gitspaceTypes.GitspaceLogger,
) error {
	gitspaceLogger.Info("Pulling image: " + imageName)

	pullResponse, err := dockerClient.ImagePull(ctx, imageName, image.PullOptions{})
	defer func() {
		if pullResponse == nil {
			return
		}
		closingErr := pullResponse.Close()
		if closingErr != nil {
			log.Warn().Err(closingErr).Msg("failed to close image pull response")
		}
	}()

	if err != nil {
		return logStreamWrapError(gitspaceLogger, "Error while pulling image", err)
	}

	// Ensure the image has been fully pulled by reading the response
	output, err := io.ReadAll(pullResponse)
	if err != nil {
		return logStreamWrapError(gitspaceLogger, "Error while parsing image pull response", err)
	}

	gitspaceLogger.Info(string(output))
	return nil
}

// getContainerResponse retrieves container information and prepares the start response.
func GetContainerResponse(
	ctx context.Context,
	dockerClient *client.Client,
	containerName string,
	portMappings map[int]*types.PortMapping,
	codeRepoDir string,
) (*StartResponse, error) {
	id, ports, err := GetContainerInfo(ctx, containerName, dockerClient, portMappings)
	if err != nil {
		return nil, err
	}
	return &StartResponse{
		ContainerID:      id,
		ContainerName:    containerName,
		PublishedPorts:   ports,
		AbsoluteRepoPath: codeRepoDir,
	}, nil
}
