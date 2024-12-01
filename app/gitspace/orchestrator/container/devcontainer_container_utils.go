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
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	goruntime "runtime"
	"strconv"
	"strings"

	"github.com/harness/gitness/app/gitspace/orchestrator/runarg"
	gitspaceTypes "github.com/harness/gitness/app/gitspace/types"
	"github.com/harness/gitness/types"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/registry"
	"github.com/docker/docker/api/types/strslice"
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
	runArgsMap map[types.RunArg]*types.RunArgValue,
	containerUser string,
	remoteUser string,
) error {
	exposedPorts, portBindings := applyPortMappings(portMappings)

	gitspaceLogger.Info("Creating container: " + containerName)

	hostConfig, err := prepareHostConfig(bindMountSource, bindMountTarget, mountType, portBindings, runArgsMap)
	if err != nil {
		return err
	}
	healthCheckConfig, err := getHealthCheckConfig(runArgsMap)
	if err != nil {
		return err
	}
	stopTimeout, err := getStopTimeout(runArgsMap)
	if err != nil {
		return err
	}

	entrypoint := getEntrypoint(runArgsMap)
	var cmd strslice.StrSlice
	if len(entrypoint) == 0 {
		entrypoint = []string{"/bin/sh"}
		cmd = []string{"-c", "trap 'exit 0' 15; sleep infinity & wait $!"}
	}

	labels := getLabels(runArgsMap)
	// Setting the following so that it can be read later to form gitspace URL.
	labels[gitspaceRemoteUserLabel] = remoteUser

	// Create the container
	containerConfig := &container.Config{
		Hostname:     getHostname(runArgsMap),
		Domainname:   getDomainname(runArgsMap),
		Image:        imageName,
		Env:          env,
		Entrypoint:   entrypoint,
		Cmd:          cmd,
		ExposedPorts: exposedPorts,
		Labels:       labels,
		Healthcheck:  healthCheckConfig,
		MacAddress:   getMACAddress(runArgsMap),
		StopSignal:   getStopSignal(runArgsMap),
		StopTimeout:  stopTimeout,
		User:         containerUser,
	}

	_, err = dockerClient.ContainerCreate(ctx, containerConfig, hostConfig, nil, nil, containerName)
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
	runArgsMap map[types.RunArg]*types.RunArgValue,
) (*container.HostConfig, error) {
	hostResources, err := getHostResources(runArgsMap)
	if err != nil {
		return nil, err
	}

	extraHosts := getExtraHosts(runArgsMap)
	if goruntime.GOOS == "linux" {
		extraHosts = append(extraHosts, "host.docker.internal:host-gateway")
	}

	restartPolicy, err := getRestartPolicy(runArgsMap)
	if err != nil {
		return nil, err
	}

	oomScoreAdj, err := getOomScoreAdj(runArgsMap)
	if err != nil {
		return nil, err
	}

	shmSize, err := getSHMSize(runArgsMap)
	if err != nil {
		return nil, err
	}

	hostConfig := &container.HostConfig{
		PortBindings: portBindings,
		Mounts: []mount.Mount{
			{
				Type:   mountType,
				Source: bindMountSource,
				Target: bindMountTarget,
			},
		},
		Resources:     hostResources,
		Annotations:   getAnnotations(runArgsMap),
		ExtraHosts:    extraHosts,
		NetworkMode:   getNetworkMode(runArgsMap),
		RestartPolicy: restartPolicy,
		AutoRemove:    getAutoRemove(runArgsMap),
		CapDrop:       getCapDrop(runArgsMap),
		CgroupnsMode:  getCgroupNSMode(runArgsMap),
		DNS:           getDNS(runArgsMap),
		DNSOptions:    getDNSOptions(runArgsMap),
		DNSSearch:     getDNSSearch(runArgsMap),
		IpcMode:       getIPCMode(runArgsMap),
		Isolation:     getIsolation(runArgsMap),
		Init:          getInit(runArgsMap),
		Links:         getLinks(runArgsMap),
		OomScoreAdj:   oomScoreAdj,
		PidMode:       getPIDMode(runArgsMap),
		Runtime:       getRuntime(runArgsMap),
		SecurityOpt:   getSecurityOpt(runArgsMap),
		StorageOpt:    getStorageOpt(runArgsMap),
		ShmSize:       shmSize,
		Sysctls:       getSysctls(runArgsMap),
	}

	return hostConfig, nil
}

func GetContainerInfo(
	ctx context.Context,
	containerName string,
	dockerClient *client.Client,
	portMappings map[int]*types.PortMapping,
) (string, map[int]string, string, error) {
	inspectResp, err := dockerClient.ContainerInspect(ctx, containerName)
	if err != nil {
		return "", nil, "", fmt.Errorf("could not inspect container %s: %w", containerName, err)
	}

	usedPorts := make(map[int]string)
	for portAndProtocol, bindings := range inspectResp.NetworkSettings.Ports {
		portRaw := strings.Split(string(portAndProtocol), "/")[0]
		port, conversionErr := strconv.Atoi(portRaw)
		if conversionErr != nil {
			return "", nil, "", fmt.Errorf("could not convert port %s to int: %w", portRaw, conversionErr)
		}

		if portMappings[port] != nil {
			usedPorts[port] = bindings[0].HostPort
		}
	}

	remoteUser := ExtractRemoteUserFromLabels(inspectResp)

	return inspectResp.ID, usedPorts, remoteUser, nil
}

func ExtractMetadataFromImage(
	ctx context.Context,
	imageName string,
	dockerClient *client.Client,
) (map[string]any, error) {
	imageInspect, _, err := dockerClient.ImageInspectWithRaw(ctx, imageName)
	if err != nil {
		return nil, fmt.Errorf("error while inspecting image: %w", err)
	}
	metadataMap := map[string]any{}
	if metadata, ok := imageInspect.Config.Labels["devcontainer.metadata"]; ok {
		dst := []map[string]any{}
		unmarshalErr := json.Unmarshal([]byte(metadata), &dst)
		if unmarshalErr != nil {
			return nil, fmt.Errorf("error while unmarshalling metadata: %w", err)
		}
		for _, values := range dst {
			for k, v := range values {
				metadataMap[k] = v
			}
		}
	}
	return metadataMap, nil
}

func PullImage(
	ctx context.Context,
	imageName string,
	dockerClient *client.Client,
	runArgsMap map[types.RunArg]*types.RunArgValue,
	gitspaceLogger gitspaceTypes.GitspaceLogger,
	imageAuthMap map[string]gitspaceTypes.DockerRegistryAuth,
) error {
	imagePullRunArg := getImagePullPolicy(runArgsMap)
	gitspaceLogger.Info("Image pull policy is: " + imagePullRunArg)
	if imagePullRunArg == "never" {
		return nil
	}
	if imagePullRunArg == "missing" {
		gitspaceLogger.Info("Checking if image " + imageName + " is present locally")
		ok, err := isImagePresentLocally(ctx, imageName, dockerClient)
		if err != nil {
			gitspaceLogger.Error("Error listing images locally", err)
			return err
		}

		if ok {
			gitspaceLogger.Info("Image " + imageName + " is present locally")
			return nil
		}

		gitspaceLogger.Info("Image " + imageName + " is not present locally")
	}

	gitspaceLogger.Info("Pulling image: " + imageName)

	pullOpts, err := buildImagePullOptions(imageName, getPlatform(runArgsMap), imageAuthMap)
	if err != nil {
		return logStreamWrapError(gitspaceLogger, "Error building image pull options", err)
	}

	pullResponse, err := dockerClient.ImagePull(ctx, imageName, pullOpts)
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
	defer func() {
		if pullResponse != nil {
			if closingErr := pullResponse.Close(); closingErr != nil {
				log.Warn().Err(closingErr).Msg("Failed to close image pull response")
			}
		}
	}()
	// Process JSON stream
	decoder := json.NewDecoder(pullResponse)
	layerStatus := make(map[string]string) // Track last status of each layer

	for {
		var pullEvent map[string]interface{}
		if err := decoder.Decode(&pullEvent); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return logStreamWrapError(gitspaceLogger, "Error while decoding image pull response", err)
		}
		// Extract relevant fields from the JSON object
		layerID, _ := pullEvent["id"].(string)    // Layer ID (if available)
		status, _ := pullEvent["status"].(string) // Current status
		// Update logs only when the status changes
		if layerID != "" {
			if lastStatus, exists := layerStatus[layerID]; !exists || lastStatus != status {
				layerStatus[layerID] = status
				gitspaceLogger.Info(fmt.Sprintf("Layer %s: %s", layerID, status))
			}
		} else if status != "" {
			// Log non-layer-specific status
			gitspaceLogger.Info(status)
		}
	}
	gitspaceLogger.Info("Image pull completed successfully")
	return nil
}

func isImagePresentLocally(ctx context.Context, imageName string, dockerClient *client.Client) (bool, error) {
	filterArgs := filters.NewArgs()
	filterArgs.Add("reference", imageName)

	images, err := dockerClient.ImageList(ctx, image.ListOptions{Filters: filterArgs})
	if err != nil {
		return false, err
	}

	return len(images) > 0, nil
}

func buildImagePullOptions(
	imageName,
	platform string,
	imageAuthMap map[string]gitspaceTypes.DockerRegistryAuth,
) (image.PullOptions, error) {
	pullOpts := image.PullOptions{Platform: platform}
	if imageAuth, ok := imageAuthMap[imageName]; ok {
		authConfig := registry.AuthConfig{
			Username:      imageAuth.Username.Value(),
			Password:      imageAuth.Password.Value(),
			ServerAddress: imageAuth.RegistryURL,
		}
		auth, err := encodeAuthToBase64(authConfig)
		if err != nil {
			return image.PullOptions{}, fmt.Errorf("encoding auth for docker registry: %w", err)
		}

		pullOpts.RegistryAuth = auth
	}

	return pullOpts, nil
}

func ExtractRunArgsWithLogging(
	ctx context.Context,
	spaceID int64,
	runArgProvider runarg.Provider,
	runArgsRaw []string,
	gitspaceLogger gitspaceTypes.GitspaceLogger,
) (map[types.RunArg]*types.RunArgValue, error) {
	runArgsMap, err := ExtractRunArgs(ctx, spaceID, runArgProvider, runArgsRaw)
	if err != nil {
		return nil, logStreamWrapError(gitspaceLogger, "Error while extracting runArgs", err)
	}
	if len(runArgsMap) > 0 {
		st := ""
		for key, value := range runArgsMap {
			st = fmt.Sprintf("%s%s: %s\n", st, key, value)
		}
		gitspaceLogger.Info(fmt.Sprintf("Using the following runArgs\n%v", st))
	} else {
		gitspaceLogger.Info("No runArgs found")
	}
	return runArgsMap, nil
}

// GetContainerResponse retrieves container information and prepares the start response.
func GetContainerResponse(
	ctx context.Context,
	dockerClient *client.Client,
	containerName string,
	portMappings map[int]*types.PortMapping,
	repoName string,
) (*StartResponse, error) {
	id, ports, remoteUser, err := GetContainerInfo(ctx, containerName, dockerClient, portMappings)
	if err != nil {
		return nil, err
	}

	homeDir := GetUserHomeDir(remoteUser)
	codeRepoDir := filepath.Join(homeDir, repoName)

	return &StartResponse{
		ContainerID:      id,
		ContainerName:    containerName,
		PublishedPorts:   ports,
		AbsoluteRepoPath: codeRepoDir,
		RemoteUser:       remoteUser,
	}, nil
}

func GetRemoteUserFromContainerLabel(
	ctx context.Context,
	containerName string,
	dockerClient *client.Client,
) (string, error) {
	inspectResp, err := dockerClient.ContainerInspect(ctx, containerName)
	if err != nil {
		return "", fmt.Errorf("could not inspect container %s: %w", containerName, err)
	}

	return ExtractRemoteUserFromLabels(inspectResp), nil
}

// Helper function to encode the AuthConfig into a Base64 string.
func encodeAuthToBase64(authConfig registry.AuthConfig) (string, error) {
	authJSON, err := json.Marshal(authConfig)
	if err != nil {
		return "", fmt.Errorf("encoding auth config: %w", err)
	}
	return base64.URLEncoding.EncodeToString(authJSON), nil
}
