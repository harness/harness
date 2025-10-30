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
	"net/http"
	"path/filepath"

	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/docker/docker/client"
	"github.com/docker/go-connections/tlsconfig"
)

type DockerClientFactory struct {
	config *DockerConfig
}

func NewDockerClientFactory(config *DockerConfig) *DockerClientFactory {
	return &DockerClientFactory{config: config}
}

// NewDockerClient returns a new docker client created using the docker config and infra.
func (d *DockerClientFactory) NewDockerClient(
	_ context.Context,
	infra types.Infrastructure,
) (*client.Client, error) {
	if infra.ProviderType != enum.InfraProviderTypeDocker {
		return nil, fmt.Errorf("infra provider type %s not supported", infra.ProviderType)
	}
	dockerClient, err := d.getClient(infra.InputParameters)
	if err != nil {
		return nil, fmt.Errorf("error creating docker client using infra %+v: %w", infra, err)
	}
	return dockerClient, nil
}

func (d *DockerClientFactory) getClient(_ []types.InfraProviderParameter) (*client.Client, error) {
	overrides, err := d.dockerOpts(d.config)
	if err != nil {
		return nil, fmt.Errorf("unable to create docker opts overrides: %w", err)
	}
	opts := append([]client.Opt{client.FromEnv}, overrides...)
	dockerClient, err := client.NewClientWithOpts(opts...)
	if err != nil {
		return nil, fmt.Errorf("unable to create docker client: %w", err)
	}

	return dockerClient, nil
}

func (d *DockerClientFactory) getHTTPSClient() (*http.Client, error) {
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

// dockerOpts returns back the options to be overridden from docker options set
// in the environment. If values are specified in gitness, they get preference.
func (d *DockerClientFactory) dockerOpts(config *DockerConfig) ([]client.Opt, error) {
	var overrides []client.Opt
	if config.DockerHost != "" {
		overrides = append(overrides, client.WithHost(config.DockerHost))
	}
	if config.DockerAPIVersion != "" {
		overrides = append(overrides, client.WithVersion(config.DockerAPIVersion))
	}
	if config.DockerCertPath != "" {
		httpsClient, err := d.getHTTPSClient()
		if err != nil {
			return nil, fmt.Errorf("unable to create https client for docker client: %w", err)
		}
		overrides = append(overrides, client.WithHTTPClient(httpsClient))
	}
	return overrides, nil
}
