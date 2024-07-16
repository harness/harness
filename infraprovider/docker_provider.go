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
	"io"
	"os"
	"path/filepath"

	"github.com/harness/gitness/infraprovider/enum"

	"github.com/rs/zerolog/log"
)

var _ InfraProvider = (*DockerProvider)(nil)

const gitspacesDir = "gitspaces"

type DockerProviderConfig struct {
	MountSourceBasePath string
}

type DockerProvider struct {
	config               *DockerConfig
	dockerClientFactory  *DockerClientFactory
	dockerProviderConfig *DockerProviderConfig
}

func NewDockerProvider(
	config *DockerConfig,
	dockerClientFactory *DockerClientFactory,
	dockerProviderConfig *DockerProviderConfig,
) *DockerProvider {
	return &DockerProvider{
		config:               config,
		dockerClientFactory:  dockerClientFactory,
		dockerProviderConfig: dockerProviderConfig,
	}
}

// Provision assumes a docker engine is already running on the Gitness host machine and re-uses that as infra.
// It does not start docker engine. It creates a directory in the host machine using the given resource key.
func (d DockerProvider) Provision(
	ctx context.Context,
	spacePath string,
	resourceKey string,
	params []Parameter,
) (Infrastructure, error) {
	dockerClient, err := d.dockerClientFactory.NewDockerClient(ctx, &Infrastructure{
		ProviderType: enum.InfraProviderTypeDocker,
		Parameters:   params,
	})
	if err != nil {
		return Infrastructure{}, fmt.Errorf("error getting docker client from docker client factory: %w", err)
	}

	defer func() {
		closingErr := dockerClient.Close()
		if closingErr != nil {
			log.Ctx(ctx).Warn().Err(closingErr).Msg("failed to close docker client")
		}
	}()

	info, err := dockerClient.Info(ctx)
	if err != nil {
		return Infrastructure{}, fmt.Errorf("unable to connect to docker engine: %w", err)
	}

	err = d.createMountSourceDirectory(spacePath, resourceKey)
	if err != nil {
		return Infrastructure{}, err
	}

	return Infrastructure{
		Identifier:   info.ID,
		ProviderType: enum.InfraProviderTypeDocker,
		Status:       enum.InfraStatusProvisioned,
		Host:         d.config.DockerMachineHostName,
	}, nil
}

func (d DockerProvider) createMountSourceDirectory(spacePath string, resourceKey string) error {
	mountSourcePath := d.getMountSourceDirectoryPath(spacePath, resourceKey)

	err := os.MkdirAll(mountSourcePath, os.ModePerm)
	if err != nil {
		return fmt.Errorf(
			"could not create bind mount source path %s: %w", mountSourcePath, err)
	}

	return nil
}

func (d DockerProvider) deleteMountSourceDirectory(spacePath string, resourceKey string) error {
	mountSourcePath := d.getMountSourceDirectoryPath(spacePath, resourceKey)

	err := os.RemoveAll(mountSourcePath)
	if err != nil {
		return fmt.Errorf(
			"could not delete bind mount source path %s: %w", mountSourcePath, err)
	}

	return nil
}

func (d DockerProvider) getMountSourceDirectoryPath(spacePath string, resourceKey string) string {
	return filepath.Join(
		d.dockerProviderConfig.MountSourceBasePath,
		gitspacesDir,
		spacePath,
		resourceKey,
	)
}

func (d DockerProvider) Find(_ context.Context, _ string, _ []Parameter) (Infrastructure, error) {
	// TODO implement me
	panic("implement me")
}

// Stop is NOOP as this provider uses already running docker engine. It does not stop the docker engine.
func (d DockerProvider) Stop(_ context.Context, infra Infrastructure) (Infrastructure, error) {
	return infra, nil
}

// Deprovision deletes the host machine directory created by Provision. It does not stop the docker engine.
func (d DockerProvider) Deprovision(_ context.Context, infra Infrastructure) (Infrastructure, error) {
	err := d.deleteMountSourceDirectory(infra.SpacePath, infra.ResourceKey)
	if err != nil {
		return Infrastructure{}, err
	}

	return infra, nil
}

func (d DockerProvider) Status(_ context.Context, _ Infrastructure) (enum.InfraStatus, error) {
	// TODO implement me
	panic("implement me")
}

// AvailableParams returns empty slice as no params are defined.
func (d DockerProvider) AvailableParams() []ParameterSchema {
	return []ParameterSchema{}
}

// ValidateParams returns nil as no params are defined.
func (d DockerProvider) ValidateParams(_ []Parameter) error {
	return nil
}

// TemplateParams returns nil as no template params are used.
func (d DockerProvider) TemplateParams() []ParameterSchema {
	return nil
}

// ProvisioningType returns existing as docker provider doesn't create new resources.
func (d DockerProvider) ProvisioningType() enum.InfraProvisioningType {
	return enum.InfraProvisioningTypeExisting
}

func (d DockerProvider) Exec(_ context.Context, _ Infrastructure, _ []string) (io.Reader, io.Reader, error) {
	// TODO implement me
	panic("implement me")
}
