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

package infraprovider

import (
	"context"
	"fmt"
	"strings"

	events "github.com/harness/gitness/app/events/gitspaceinfra"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
	"github.com/rs/zerolog/log"
)

var _ InfraProvider = (*DockerProvider)(nil)

type DockerProvider struct {
	config              *DockerConfig
	dockerClientFactory *DockerClientFactory
	eventReporter       *events.Reporter
}

func NewDockerProvider(
	config *DockerConfig,
	dockerClientFactory *DockerClientFactory,
	eventReporter *events.Reporter,
) *DockerProvider {
	return &DockerProvider{
		config:              config,
		dockerClientFactory: dockerClientFactory,
		eventReporter:       eventReporter,
	}
}

// Provision assumes a docker engine is already running on the Harness host machine and re-uses that as infra.
// It does not start docker engine. It creates a docker volume using the given gitspace config identifier.
func (d DockerProvider) Provision(
	ctx context.Context,
	spaceID int64,
	spacePath string,
	gitspaceConfigIdentifier string,
	gitspaceInstanceIdentifier string,
	_ int64,
	_ int,
	requiredGitspacePorts []types.GitspacePort,
	inputParameters []types.InfraProviderParameter,
	_ map[string]any,
	_ types.Infrastructure,
) error {
	dockerClient, err := d.dockerClientFactory.NewDockerClient(ctx, types.Infrastructure{
		ProviderType:    enum.InfraProviderTypeDocker,
		InputParameters: inputParameters,
	})
	if err != nil {
		return fmt.Errorf("error getting docker client from docker client factory: %w", err)
	}

	defer func() {
		closingErr := dockerClient.Close()
		if closingErr != nil {
			log.Ctx(ctx).Warn().Err(closingErr).Msg("failed to close docker client")
		}
	}()

	infrastructure, err := d.dockerHostInfo(ctx, dockerClient)
	if err != nil {
		return err
	}

	infrastructure.SpaceID = spaceID
	infrastructure.SpacePath = spacePath
	infrastructure.GitspaceConfigIdentifier = gitspaceConfigIdentifier
	infrastructure.GitspaceInstanceIdentifier = gitspaceInstanceIdentifier

	storageName, err := d.createNamedVolume(ctx, spacePath, gitspaceConfigIdentifier, dockerClient)
	if err != nil {
		return err
	}

	infrastructure.Storage = storageName

	var portMappings = make(map[int]*types.PortMapping, len(requiredGitspacePorts))

	for _, requiredPort := range requiredGitspacePorts {
		portMapping := &types.PortMapping{
			PublishedPort: 0,
			ForwardedPort: 0,
		}

		portMappings[requiredPort.Port] = portMapping
	}

	infrastructure.GitspacePortMappings = portMappings

	event := &events.GitspaceInfraEventPayload{
		Infra: *infrastructure,
		Type:  enum.InfraEventProvision,
	}

	err = d.eventReporter.EmitGitspaceInfraEvent(ctx, events.GitspaceInfraEvent, event)
	if err != nil {
		return fmt.Errorf("error emitting gitspace infra event for provisioning: %w", err)
	}

	return nil
}

// Find fetches the infrastructure with the current state, the method has no side effects on the infra.
func (d DockerProvider) Find(
	ctx context.Context,
	spaceID int64,
	spacePath string,
	gitspaceConfigIdentifier string,
	inputParameters []types.InfraProviderParameter,
) (*types.Infrastructure, error) {
	dockerClient, err := d.dockerClientFactory.NewDockerClient(ctx, types.Infrastructure{
		ProviderType:    enum.InfraProviderTypeDocker,
		InputParameters: inputParameters,
	})

	if err != nil {
		return nil, fmt.Errorf("error getting docker client from docker client factory: %w", err)
	}

	defer func() {
		closingErr := dockerClient.Close()
		if closingErr != nil {
			log.Ctx(ctx).Warn().Err(closingErr).Msg("failed to close docker client")
		}
	}()

	infrastructure, err := d.dockerHostInfo(ctx, dockerClient)
	if err != nil {
		return nil, err
	}

	infrastructure.SpaceID = spaceID
	infrastructure.SpacePath = spacePath
	infrastructure.GitspaceConfigIdentifier = gitspaceConfigIdentifier
	infrastructure.Storage = volumeName(spacePath, gitspaceConfigIdentifier)

	return infrastructure, nil
}

func (d DockerProvider) FindInfraStatus(_ context.Context,
	_ string,
	_ string,
	_ []types.InfraProviderParameter) (*enum.InfraStatus, error) {
	return nil, nil //nolint:nilnil
}

// Stop is NOOP as this provider uses already running docker engine. It does not stop the docker engine.
func (d DockerProvider) Stop(ctx context.Context, infra types.Infrastructure, _ map[string]any) error {
	infra.Status = enum.InfraStatusDestroyed

	event := &events.GitspaceInfraEventPayload{
		Infra: infra,
		Type:  enum.InfraEventStop,
	}

	err := d.eventReporter.EmitGitspaceInfraEvent(ctx, events.GitspaceInfraEvent, event)
	if err != nil {
		return fmt.Errorf("error emitting gitspace infra event for stopping: %w", err)
	}

	return nil
}

// CleanupInstanceResources is NOOP as this provider does not utilise infra exclusively associated to a gitspace
// instance.
func (d DockerProvider) CleanupInstanceResources(ctx context.Context, infra types.Infrastructure) error {
	event := &events.GitspaceInfraEventPayload{
		Infra: infra,
		Type:  enum.InfraEventCleanup,
	}

	infra.Status = enum.InfraStatusStopped

	err := d.eventReporter.EmitGitspaceInfraEvent(ctx, events.GitspaceInfraEvent, event)
	if err != nil {
		return fmt.Errorf("error emitting gitspace infra event for cleanup: %w", err)
	}

	return nil
}

// Deprovision is NOOP if canDeleteUserData = false
// Deprovision deletes the volume created by Provision method if canDeleteUserData = false.
// Deprovision does not stop the docker engine in any case.
func (d DockerProvider) Deprovision(
	ctx context.Context,
	infra types.Infrastructure,
	canDeleteUserData bool,
	_ map[string]any,
	_ []types.InfraProviderParameter,
) error {
	if canDeleteUserData {
		err := d.deleteVolume(ctx, infra)
		if err != nil {
			return fmt.Errorf("couldn't delete volume for %s : %w", infra.Storage, err)
		}
	}

	infra.Status = enum.InfraStatusDestroyed

	event := &events.GitspaceInfraEventPayload{
		Infra: infra,
		Type:  enum.InfraEventDeprovision,
	}

	err := d.eventReporter.EmitGitspaceInfraEvent(ctx, events.GitspaceInfraEvent, event)
	if err != nil {
		return fmt.Errorf("error emitting gitspace infra event for deprovisioning: %w", err)
	}

	return nil
}

func (d DockerProvider) deleteVolume(ctx context.Context, infra types.Infrastructure) error {
	dockerClient, err := d.dockerClientFactory.NewDockerClient(ctx, types.Infrastructure{
		ProviderType:    enum.InfraProviderTypeDocker,
		InputParameters: infra.InputParameters,
	})
	if err != nil {
		return fmt.Errorf("error getting docker client from docker client factory: %w", err)
	}

	defer func() {
		closingErr := dockerClient.Close()
		if closingErr != nil {
			log.Ctx(ctx).Warn().Err(closingErr).Msg("failed to close docker client")
		}
	}()

	// check if volume is available
	volumeList, err := dockerClient.VolumeList(ctx, volume.ListOptions{})
	if err != nil {
		return fmt.Errorf("couldn't list the volume: %w", err)
	}

	if !findVolume(infra.Storage, volumeList.Volumes) {
		// given volume does not exist, return nil
		return nil
	}

	err = dockerClient.VolumeRemove(ctx, infra.Storage, true)
	if err != nil {
		return fmt.Errorf("couldn't delete volume for %s : %w", infra.Storage, err)
	}

	return nil
}

func findVolume(target string, volumes []*volume.Volume) bool {
	for _, vol := range volumes {
		if vol == nil {
			continue
		}

		if vol.Name == target {
			return true
		}
	}

	return false
}

// AvailableParams returns empty slice as no params are defined.
func (d DockerProvider) AvailableParams() []types.InfraProviderParameterSchema {
	return []types.InfraProviderParameterSchema{}
}

// ValidateParams returns nil as no params are defined.
func (d DockerProvider) ValidateParams(_ []types.InfraProviderParameter) error {
	return nil
}

func (d DockerProvider) UpdateParams(ip []types.InfraProviderParameter,
	_ map[string]any) ([]types.InfraProviderParameter, error) {
	return ip, nil
}

// TemplateParams returns nil as no template params are used.
func (d DockerProvider) TemplateParams() []types.InfraProviderParameterSchema {
	return nil
}

// ProvisioningType returns existing as docker provider doesn't create new resources.
func (d DockerProvider) ProvisioningType() enum.InfraProvisioningType {
	return enum.InfraProvisioningTypeExisting
}

func (d DockerProvider) dockerHostInfo(
	ctx context.Context,
	dockerClient *client.Client,
) (*types.Infrastructure, error) {
	info, err := dockerClient.Info(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to docker engine: %w", err)
	}
	return &types.Infrastructure{
		Identifier:     info.ID,
		ProviderType:   enum.InfraProviderTypeDocker,
		Status:         enum.InfraStatusProvisioned,
		GitspaceHost:   d.config.DockerMachineHostName,
		GitspaceScheme: "http",
	}, nil
}

func (d DockerProvider) createNamedVolume(
	ctx context.Context,
	spacePath string,
	resourceKey string,
	dockerClient *client.Client,
) (string, error) {
	name := volumeName(spacePath, resourceKey)
	dockerVolume, err := dockerClient.VolumeCreate(ctx, volume.CreateOptions{
		Name:       name,
		Driver:     "local",
		Labels:     nil,
		DriverOpts: nil})
	if err != nil {
		return "", fmt.Errorf(
			"could not create name volume %s: %w", name, err)
	}
	log.Info().Msgf("created volume %s", dockerVolume.Name)
	return dockerVolume.Name, nil
}

func volumeName(spacePath string, resourceKey string) string {
	name := "gitspace-" + strings.ReplaceAll(spacePath, "/", "-") + "-" + resourceKey
	return name
}

func (d DockerProvider) ValidateConfig(_ *types.InfraProviderConfig) error {
	return nil
}

func (d DockerProvider) GenerateSetupYAML(_ *types.InfraProviderConfig) (string, error) {
	return "", nil
}
