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

	"github.com/harness/gitness/infraprovider/enum"

	"github.com/rs/zerolog/log"
)

var _ InfraProvider = (*DockerProvider)(nil)

type DockerProvider struct {
	config              *DockerConfig
	dockerClientFactory *DockerClientFactory
}

func NewDockerProvider(config *DockerConfig, dockerClientFactory *DockerClientFactory) *DockerProvider {
	return &DockerProvider{
		config:              config,
		dockerClientFactory: dockerClientFactory,
	}
}

// Provision assumes a docker engine is already running on the Gitness host machine and re-uses that as infra.
// It does not start docker engine.
func (d DockerProvider) Provision(ctx context.Context, _ string, params []Parameter) (Infrastructure, error) {
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
	return Infrastructure{
		Identifier:   info.ID,
		ProviderType: enum.InfraProviderTypeDocker,
		Status:       enum.InfraStatusProvisioned,
		Host:         d.config.DockerMachineHostName,
	}, nil
}

func (d DockerProvider) Find(_ context.Context, _ string, _ []Parameter) (Infrastructure, error) {
	// TODO implement me
	panic("implement me")
}

// Stop is NOOP as this provider uses already running docker engine. It does not stop the docker engine.
func (d DockerProvider) Stop(_ context.Context, infra Infrastructure) (Infrastructure, error) {
	return infra, nil
}

// Destroy is NOOP as this provider uses already running docker engine. It does not stop the docker engine.
func (d DockerProvider) Destroy(_ context.Context, infra Infrastructure) (Infrastructure, error) {
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
