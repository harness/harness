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
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	goruntime "runtime"
	"strconv"
	"strings"

	"github.com/harness/gitness/app/gitspace/orchestrator/container/response"
	"github.com/harness/gitness/app/gitspace/orchestrator/runarg"
	"github.com/harness/gitness/app/gitspace/orchestrator/utils"
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
	"github.com/gotidy/ptr"
	"github.com/rs/zerolog/log"
)

const (
	catchAllIP             = "0.0.0.0"
	imagePullRunArgMissing = "missing"
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
	features []*types.ResolvedFeature,
	devcontainerConfig types.DevcontainerConfig,
	metadataFromImage map[string]any,
) (map[PostAction][]*LifecycleHookStep, error) {
	exposedPorts, portBindings := applyPortMappings(portMappings)

	gitspaceLogger.Info(fmt.Sprintf("Creating container %s with image %s", containerName, imageName))

	hostConfig, err := prepareHostConfig(bindMountSource, bindMountTarget, mountType, portBindings, runArgsMap,
		features, devcontainerConfig, metadataFromImage)
	if err != nil {
		return nil, err
	}
	healthCheckConfig, err := getHealthCheckConfig(runArgsMap)
	if err != nil {
		return nil, err
	}
	stopTimeout, err := getStopTimeout(runArgsMap)
	if err != nil {
		return nil, err
	}

	entrypoint := mergeEntrypoints(features, runArgsMap)
	var cmd strslice.StrSlice
	if len(entrypoint) == 0 {
		entrypoint = []string{"/bin/sh"}
		cmd = []string{"-c", "trap 'exit 0' 15; sleep infinity & wait $!"}
	}

	lifecycleHookSteps := mergeLifeCycleHooks(devcontainerConfig, features)
	lifecycleHookStepsStr, err := json.Marshal(lifecycleHookSteps)
	if err != nil {
		return nil, err
	}

	labels := getLabels(runArgsMap)

	// Setting the following so that it can be read later to form gitspace URL.
	labels[gitspaceRemoteUserLabel] = remoteUser

	// Setting the following so that it can be read later to run the postStartCommands during restarts.
	labels[gitspaceLifeCycleHooksLabel] = string(lifecycleHookStepsStr)

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
		return nil, logStreamWrapError(gitspaceLogger, "Error while creating container", err)
	}

	return lifecycleHookSteps, nil
}

func mergeLifeCycleHooks(
	devcontainerConfig types.DevcontainerConfig,
	features []*types.ResolvedFeature,
) map[PostAction][]*LifecycleHookStep {
	var postCreateHooks []*LifecycleHookStep
	var postStartHooks []*LifecycleHookStep
	for _, feature := range features {
		if len(feature.DownloadedFeature.DevcontainerFeatureConfig.PostCreateCommand.ToCommandArray()) > 0 {
			postCreateHooks = append(postCreateHooks, &LifecycleHookStep{
				Source:        feature.DownloadedFeature.Source,
				Command:       feature.DownloadedFeature.DevcontainerFeatureConfig.PostCreateCommand,
				ActionType:    PostCreateAction,
				StopOnFailure: true,
			})
		}
		if len(feature.DownloadedFeature.DevcontainerFeatureConfig.PostStartCommand.ToCommandArray()) > 0 {
			postStartHooks = append(postStartHooks, &LifecycleHookStep{
				Source:        feature.DownloadedFeature.Source,
				Command:       feature.DownloadedFeature.DevcontainerFeatureConfig.PostStartCommand,
				ActionType:    PostStartAction,
				StopOnFailure: true,
			})
		}
	}

	if len(devcontainerConfig.PostCreateCommand.ToCommandArray()) > 0 {
		postCreateHooks = append(postCreateHooks, &LifecycleHookStep{
			Source:        "devcontainer.json",
			Command:       devcontainerConfig.PostCreateCommand,
			ActionType:    PostCreateAction,
			StopOnFailure: false,
		})
	}

	if len(devcontainerConfig.PostStartCommand.ToCommandArray()) > 0 {
		postStartHooks = append(postStartHooks, &LifecycleHookStep{
			Source:        "devcontainer.json",
			Command:       devcontainerConfig.PostStartCommand,
			ActionType:    PostStartAction,
			StopOnFailure: false,
		})
	}

	return map[PostAction][]*LifecycleHookStep{
		PostCreateAction: postCreateHooks,
		PostStartAction:  postStartHooks,
	}
}

func mergeEntrypoints(
	features []*types.ResolvedFeature,
	runArgsMap map[types.RunArg]*types.RunArgValue,
) strslice.StrSlice {
	entrypoints := strslice.StrSlice{}
	for _, feature := range features {
		entrypoint := feature.DownloadedFeature.DevcontainerFeatureConfig.Entrypoint
		if entrypoint != "" {
			entrypoints = append(entrypoints, entrypoint)
		}
	}
	entrypoints = append(entrypoints, getEntrypoint(runArgsMap)...)
	return entrypoints
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
	features []*types.ResolvedFeature,
	devcontainerConfig types.DevcontainerConfig,
	metadataFromImage map[string]any,
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

	defaultMount := mount.Mount{
		Type:   mountType,
		Source: bindMountSource,
		Target: bindMountTarget,
	}

	mergedMounts, err := mergeMounts(devcontainerConfig, runArgsMap, features, defaultMount, metadataFromImage)
	if err != nil {
		return nil, fmt.Errorf("failed to merge mounts: %w", err)
	}

	hostConfig := &container.HostConfig{
		PortBindings:  portBindings,
		Mounts:        mergedMounts,
		Resources:     hostResources,
		Annotations:   getAnnotations(runArgsMap),
		ExtraHosts:    extraHosts,
		NetworkMode:   getNetworkMode(runArgsMap),
		RestartPolicy: restartPolicy,
		AutoRemove:    getAutoRemove(runArgsMap),
		CapAdd:        mergeCapAdd(devcontainerConfig, runArgsMap, features, metadataFromImage),
		CapDrop:       getCapDrop(runArgsMap),
		CgroupnsMode:  getCgroupNSMode(runArgsMap),
		DNS:           getDNS(runArgsMap),
		DNSOptions:    getDNSOptions(runArgsMap),
		DNSSearch:     getDNSSearch(runArgsMap),
		IpcMode:       getIPCMode(runArgsMap),
		Isolation:     getIsolation(runArgsMap),
		Init:          mergeInit(devcontainerConfig, runArgsMap, features, metadataFromImage),
		Links:         getLinks(runArgsMap),
		OomScoreAdj:   oomScoreAdj,
		PidMode:       getPIDMode(runArgsMap),
		Privileged:    mergePriviledged(devcontainerConfig, runArgsMap, features, metadataFromImage),
		Runtime:       getRuntime(runArgsMap),
		SecurityOpt:   mergeSecurityOpts(devcontainerConfig, runArgsMap, features, metadataFromImage),
		StorageOpt:    getStorageOpt(runArgsMap),
		ShmSize:       shmSize,
		Sysctls:       getSysctls(runArgsMap),
	}

	return hostConfig, nil
}

func mergeMounts(
	devcontainerConfig types.DevcontainerConfig,
	runArgsMap map[types.RunArg]*types.RunArgValue,
	features []*types.ResolvedFeature,
	defaultMount mount.Mount,
	metadataFromImage map[string]any,
) ([]mount.Mount, error) {
	var allMountsRaw []*types.Mount
	for _, feature := range features {
		if len(feature.DownloadedFeature.DevcontainerFeatureConfig.Mounts) > 0 {
			allMountsRaw = append(allMountsRaw, feature.DownloadedFeature.DevcontainerFeatureConfig.Mounts...)
		}
	}

	mountsFromRunArgs, err := getMounts(runArgsMap)
	if err != nil {
		return nil, err
	}

	// First check if mounts have been overridden in the runArgs, then check if the devcontainer.json
	// provides any security options, if not, only then check the image metadata.
	switch {
	case len(mountsFromRunArgs) > 0:
		allMountsRaw = append(allMountsRaw, mountsFromRunArgs...)
	case len(devcontainerConfig.Mounts) > 0:
		allMountsRaw = append(allMountsRaw, devcontainerConfig.Mounts...)
	default:
		if values, ok := metadataFromImage["mounts"].([]any); ok {
			parsedMounts, err := types.ParseMountsFromRawSlice(values)
			if err != nil {
				return nil, err
			}
			allMountsRaw = append(allMountsRaw, parsedMounts...)
		}
	}

	var allMounts []mount.Mount
	for _, rawMount := range allMountsRaw {
		if rawMount.Type == "" {
			rawMount.Type = string(mount.TypeVolume)
		}
		parsedMount := mount.Mount{
			Type:   mount.Type(rawMount.Type),
			Source: rawMount.Source,
			Target: rawMount.Target,
		}
		allMounts = append(allMounts, parsedMount)
	}

	allMounts = append(allMounts, defaultMount)

	return allMounts, nil
}

func mergeSecurityOpts(
	devcontainerConfig types.DevcontainerConfig,
	runArgsMap map[types.RunArg]*types.RunArgValue,
	features []*types.ResolvedFeature,
	metadataFromImage map[string]any,
) []string {
	var allOpts []string
	for _, feature := range features {
		allOpts = append(allOpts, feature.DownloadedFeature.DevcontainerFeatureConfig.SecurityOpt...)
	}

	// First check if security options have been overridden in the runArgs, then check if the devcontainer.json
	// provides any security options, if not, only then check the image metadata.
	securityOptsFromRunArgs := getSecurityOpt(runArgsMap)
	switch {
	case len(securityOptsFromRunArgs) > 0:
		allOpts = append(allOpts, securityOptsFromRunArgs...)
	case len(devcontainerConfig.SecurityOpt) > 0:
		allOpts = append(allOpts, devcontainerConfig.SecurityOpt...)
	default:
		if value, ok := metadataFromImage["securityOpt"].([]string); ok {
			allOpts = append(allOpts, value...)
		}
	}

	return allOpts
}

func mergeCapAdd(
	devcontainerConfig types.DevcontainerConfig,
	runArgsMap map[types.RunArg]*types.RunArgValue,
	features []*types.ResolvedFeature,
	metadataFromImage map[string]any,
) strslice.StrSlice {
	allCaps := strslice.StrSlice{}
	for _, feature := range features {
		allCaps = append(allCaps, feature.DownloadedFeature.DevcontainerFeatureConfig.CapAdd...)
	}

	// First check if capAdd have been overridden in the runArgs, then check if the devcontainer.json
	// provides any capAdd, if not, only then check the image metadata.
	capAddFromRunArgs := getCapAdd(runArgsMap)
	switch {
	case len(capAddFromRunArgs) > 0:
		allCaps = append(allCaps, capAddFromRunArgs...)
	case len(devcontainerConfig.CapAdd) > 0:
		allCaps = append(allCaps, devcontainerConfig.CapAdd...)
	default:
		if value, ok := metadataFromImage["capAdd"].([]string); ok {
			allCaps = append(allCaps, value...)
		}
	}

	return allCaps
}

func mergeInit(
	devcontainerConfig types.DevcontainerConfig,
	runArgsMap map[types.RunArg]*types.RunArgValue,
	features []*types.ResolvedFeature,
	metadataFromImage map[string]any,
) *bool {
	// First check if init has been overridden in the runArgs, if not, then check in the devcontainer.json
	// lastly check in the image metadata.
	var initPtr = getInit(runArgsMap)

	if initPtr == nil {
		if devcontainerConfig.Init != nil {
			initPtr = devcontainerConfig.Init
		} else {
			if value, ok := metadataFromImage["init"].(bool); ok {
				initPtr = ptr.Bool(value)
			}
		}
	}

	var init = ptr.ToBool(initPtr)

	// Merge this valye with the value from the features.
	if !init {
		for _, feature := range features {
			if feature.DownloadedFeature.DevcontainerFeatureConfig.Init {
				init = true
				break
			}
		}
	}

	return ptr.Bool(init)
}

func mergePriviledged(
	devcontainerConfig types.DevcontainerConfig,
	runArgsMap map[types.RunArg]*types.RunArgValue,
	features []*types.ResolvedFeature,
	metadataFromImage map[string]any,
) bool {
	// First check if privileged has been overridden in the runArgs, if not, then check in the devcontainer.json
	// lastly check in the image metadata.
	var privilegedPtr = getPrivileged(runArgsMap)

	if privilegedPtr == nil {
		if devcontainerConfig.Privileged != nil {
			privilegedPtr = devcontainerConfig.Privileged
		} else {
			if value, ok := metadataFromImage["privileged"].(bool); ok {
				privilegedPtr = ptr.Bool(value)
			}
		}
	}

	var privileged = ptr.ToBool(privilegedPtr)

	// Merge this valye with the value from the features.
	if !privileged {
		for _, feature := range features {
			if feature.DownloadedFeature.DevcontainerFeatureConfig.Privileged {
				privileged = true
				break
			}
		}
	}

	return privileged
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

func ExtractMetadataAndUserFromImage(
	ctx context.Context,
	imageName string,
	dockerClient *client.Client,
) (map[string]any, string, error) {
	imageInspect, _, err := dockerClient.ImageInspectWithRaw(ctx, imageName)
	if err != nil {
		return nil, "", fmt.Errorf("error while inspecting image: %w", err)
	}
	imageUser := imageInspect.Config.User
	if imageUser == "" {
		imageUser = "root"
	}
	metadataMap := map[string]any{}
	if metadata, ok := imageInspect.Config.Labels["devcontainer.metadata"]; ok {
		dst := []map[string]any{}
		unmarshalErr := json.Unmarshal([]byte(metadata), &dst)
		if unmarshalErr != nil {
			return nil, imageUser, fmt.Errorf("error while unmarshalling metadata: %w", err)
		}
		for _, values := range dst {
			for k, v := range values {
				metadataMap[k] = v
			}
		}
	}
	return metadataMap, imageUser, nil
}

func CopyImage(
	ctx context.Context,
	imageName string,
	dockerClient *client.Client,
	runArgsMap map[types.RunArg]*types.RunArgValue,
	gitspaceLogger gitspaceTypes.GitspaceLogger,
	imageAuthMap map[string]gitspaceTypes.DockerRegistryAuth,
	httpProxyURL types.MaskSecret,
	httpsProxyURL types.MaskSecret,
) error {
	gitspaceLogger.Info("Copying image " + imageName + " to local")
	imagePullRunArg := getImagePullPolicy(runArgsMap)
	if imagePullRunArg == "never" {
		return nil
	}
	if imagePullRunArg == imagePullRunArgMissing {
		imagePresentLocally, err := utils.IsImagePresentLocally(ctx, imageName, dockerClient)
		if err != nil {
			return err
		}

		if imagePresentLocally {
			return nil
		}
	}

	// Build skopeo command
	platform := getPlatform(runArgsMap)
	args := []string{"copy", "--debug", "--src-tls-verify=false"}

	if platform != "" {
		args = append(args, "--override-os", platform)
	}

	// Add credentials if available
	if auth, ok := imageAuthMap[imageName]; ok && auth.Password != nil {
		gitspaceLogger.Info("Using credentials for registry: " + auth.RegistryURL)
		args = append(args, "--src-creds", auth.Username.Value()+":"+auth.Password.Value())
	} else {
		gitspaceLogger.Warn("No credentials found for registry. Proceeding without authentication.")
	}

	// Source and destination
	source := "docker://" + imageName
	image, tag := getImageAndTag(imageName)
	destination := "docker-daemon:" + image + ":" + tag
	args = append(args, source, destination)

	cmd := exec.CommandContext(ctx, "skopeo", args...)

	// Set proxy environment variables if provided
	env := cmd.Env
	if httpProxyURL.Value() != "" {
		env = append(env, "HTTP_PROXY="+httpProxyURL.Value())
		log.Info().Msg("HTTP_PROXY set in environment")
	}
	if httpsProxyURL.Value() != "" {
		env = append(env, "HTTPS_PROXY="+httpsProxyURL.Value())
		log.Info().Msg("HTTPS_PROXY set in environment")
	}
	cmd.Env = env

	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	gitspaceLogger.Info("Executing image copy command: " + cmd.String())
	cmdErr := cmd.Run()

	response, err := io.ReadAll(&outBuf)
	if err != nil {
		return logStreamWrapError(gitspaceLogger, "Error while reading image output", err)
	}
	gitspaceLogger.Info("Image copy output: " + string(response))
	errResponse, err := io.ReadAll(&errBuf)
	if err != nil {
		return logStreamWrapError(gitspaceLogger, "Error while reading image output", err)
	}
	combinedOutput := string(response) + "\n" + string(errResponse)
	gitspaceLogger.Info("Image copy combined output: " + combinedOutput)

	if cmdErr != nil {
		return logStreamWrapError(gitspaceLogger, "Error while pulling image using skopeo", cmdErr)
	}

	gitspaceLogger.Info("Image copy completed successfully using skopeo")
	return nil
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
	if imagePullRunArg == imagePullRunArgMissing {
		gitspaceLogger.Info("Checking if image " + imageName + " is present locally")
		imagePresentLocally, err := utils.IsImagePresentLocally(ctx, imageName, dockerClient)
		if err != nil {
			gitspaceLogger.Error("Error listing images locally", err)
			return err
		}

		if imagePresentLocally {
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
	if err = processImagePullResponse(pullResponse, gitspaceLogger); err != nil {
		return err
	}
	gitspaceLogger.Info("Image pull completed successfully")
	return nil
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
) (*response.StartResponse, error) {
	id, ports, remoteUser, err := GetContainerInfo(ctx, containerName, dockerClient, portMappings)
	if err != nil {
		return nil, err
	}

	homeDir := GetUserHomeDir(remoteUser)
	codeRepoDir := filepath.Join(homeDir, repoName)

	return &response.StartResponse{
		Status:           response.SuccessStatus,
		ContainerID:      id,
		ContainerName:    containerName,
		PublishedPorts:   ports,
		AbsoluteRepoPath: codeRepoDir,
		RemoteUser:       remoteUser,
	}, nil
}

func GetGitspaceInfoFromContainerLabels(
	ctx context.Context,
	containerName string,
	dockerClient *client.Client,
) (string, map[PostAction][]*LifecycleHookStep, error) {
	inspectResp, err := dockerClient.ContainerInspect(ctx, containerName)
	if err != nil {
		return "", nil, fmt.Errorf("could not inspect container %s: %w", containerName, err)
	}

	remoteUser := ExtractRemoteUserFromLabels(inspectResp)
	lifecycleHooks, err := ExtractLifecycleHooksFromLabels(inspectResp)
	if err != nil {
		return "", nil, fmt.Errorf("could not extract lifecycle hooks: %w", err)
	}
	return remoteUser, lifecycleHooks, nil
}

// Helper function to encode the AuthConfig into a Base64 string.
func encodeAuthToBase64(authConfig registry.AuthConfig) (string, error) {
	authJSON, err := json.Marshal(authConfig)
	if err != nil {
		return "", fmt.Errorf("encoding auth config: %w", err)
	}
	return base64.URLEncoding.EncodeToString(authJSON), nil
}

func processImagePullResponse(pullResponse io.ReadCloser, gitspaceLogger gitspaceTypes.GitspaceLogger) error {
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
	return nil
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

// getImageAndTag separates the image name and tag correctly.
func getImageAndTag(image string) (string, string) {
	// Split the image on the last slash
	lastSlashIndex := strings.LastIndex(image, "/")
	var name, tag string

	if lastSlashIndex != -1 { //nolint:nestif
		// If there's a slash, the portion before the last slash is part of the registry/repository
		// and the portion after the last slash will be considered for the tag handling.
		// Now check if there is a colon in the string
		lastColonIndex := strings.LastIndex(image, ":")
		if lastColonIndex != -1 && lastColonIndex > lastSlashIndex {
			// There is a tag after the last colon
			name = image[:lastColonIndex]
			tag = image[lastColonIndex+1:]
		} else {
			// No colon, treat it as the image and assume "latest" tag
			name = image
			tag = "latest"
		}
	} else {
		// If no slash is present, split on the last colon for image and tag
		lastColonIndex := strings.LastIndex(image, ":")
		if lastColonIndex != -1 {
			name = image[:lastColonIndex]
			tag = image[lastColonIndex+1:]
		} else {
			name = image
			tag = "latest"
		}
	}

	return name, tag
}
