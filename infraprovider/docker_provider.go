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
	"net/http"
	"path/filepath"

	"github.com/harness/gitness/infraprovider/enum"

	"github.com/docker/docker/client"
	"github.com/docker/go-connections/tlsconfig"
	"github.com/rs/zerolog/log"
)

var _ InfraProvider = (*DockerProvider)(nil)

type Config struct {
	DockerHost       string
	DockerAPIVersion string
	DockerCertPath   string
	DockerTLSVerify  string
}

type DockerProvider struct {
	config *Config
}

func NewDockerProvider(config *Config) *DockerProvider {
	return &DockerProvider{
		config: config,
	}
}

// Provision assumes a docker engine is already running on the Gitness host machine and re-uses that as infra.
// It does not start docker engine.
func (d DockerProvider) Provision(ctx context.Context, _ string, params []Parameter) (Infrastructure, error) {
	dockerClient, closeFunc, err := d.getClient(params)
	if err != nil {
		return Infrastructure{}, err
	}
	defer closeFunc(ctx)
	info, err := dockerClient.Info(ctx)
	if err != nil {
		return Infrastructure{}, fmt.Errorf("unable to connect to docker engine: %w", err)
	}
	return Infrastructure{
		Identifier:   info.ID,
		ProviderType: enum.InfraProviderTypeDocker,
		Status:       enum.InfraStatusProvisioned,
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

// Client returns a new docker client created using params.
func (d DockerProvider) Client(_ context.Context, infra Infrastructure) (Client, error) {
	dockerClient, closeFunc, err := d.getClient(infra.Parameters)
	if err != nil {
		return nil, err
	}
	return &DockerClient{
		dockerClient: dockerClient,
		closeFunc:    closeFunc,
	}, nil
}

// getClient returns a new docker client created using values from gitness docker config.
func (d DockerProvider) getClient(_ []Parameter) (*client.Client, func(context.Context), error) {
	var opts []client.Opt

	opts = append(opts, client.WithHost(d.config.DockerHost))

	opts = append(opts, client.WithVersion(d.config.DockerAPIVersion))

	if d.config.DockerCertPath != "" {
		httpsClient, err := d.getHTTPSClient()
		if err != nil {
			return nil, nil, fmt.Errorf("unable to create https client for docker client: %w", err)
		}
		opts = append(opts, client.WithHTTPClient(httpsClient))
	}

	dockerClient, err := client.NewClientWithOpts(opts...)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to create docker client: %w", err)
	}

	closeFunc := func(ctx context.Context) {
		closingErr := dockerClient.Close()
		if closingErr != nil {
			log.Ctx(ctx).Warn().Err(closingErr).Msg("failed to close docker client")
		}
	}

	return dockerClient, closeFunc, nil
}

func (d DockerProvider) getHTTPSClient() (*http.Client, error) {
	options := tlsconfig.Options{
		CAFile:             filepath.Join(d.config.DockerCertPath, "ca.pem"),
		CertFile:           filepath.Join(d.config.DockerCertPath, "cert.pem"),
		KeyFile:            filepath.Join(d.config.DockerCertPath, "key.pem"),
		InsecureSkipVerify: d.config.DockerTLSVerify == "",
	}
	tlsc, err := tlsconfig.Client(options)
	if err != nil {
		return nil, err
	}
	return &http.Client{
		Transport:     &http.Transport{TLSClientConfig: tlsc},
		CheckRedirect: client.CheckRedirect,
	}, nil
}
