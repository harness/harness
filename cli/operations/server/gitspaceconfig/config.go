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

// Package gitspaceconfig holds the config providers for the docker/oras-heavy
// gitspace orchestrator, ide and infra-provisioner packages. It is kept
// separate from cli/operations/server so that binaries which only need the
// other server config providers (e.g. code-api) do not transitively import the
// heavy gitspace packages.
package gitspaceconfig

import (
	"fmt"
	"net/url"

	"github.com/harness/gitness/app/gitspace/infrastructure"
	"github.com/harness/gitness/app/gitspace/orchestrator"
	"github.com/harness/gitness/app/gitspace/orchestrator/ide"
	"github.com/harness/gitness/infraprovider"
	"github.com/harness/gitness/types"
)

// ProvideDockerConfig loads config for Docker.
func ProvideDockerConfig(config *types.Config) (*infraprovider.DockerConfig, error) {
	if config.Docker.MachineHostName == "" {
		gitnessBaseURL, err := url.Parse(config.URL.Base)
		if err != nil {
			return nil, fmt.Errorf("unable to parse Harness base URL %s: %w", gitnessBaseURL, err)
		}
		config.Docker.MachineHostName = gitnessBaseURL.Hostname()
	}

	return &infraprovider.DockerConfig{
		DockerHost:            config.Docker.Host,
		DockerAPIVersion:      config.Docker.APIVersion,
		DockerCertPath:        config.Docker.CertPath,
		DockerTLSVerify:       config.Docker.TLSVerify,
		DockerMachineHostName: config.Docker.MachineHostName,
	}, nil
}

// ProvideIDEVSCodeWebConfig loads the VSCode Web IDE config from the main config.
func ProvideIDEVSCodeWebConfig(config *types.Config) *ide.VSCodeWebConfig {
	return &ide.VSCodeWebConfig{
		Port: config.IDE.VSCodeWeb.Port,
	}
}

// ProvideIDEVSCodeConfig loads the VSCode IDE config from the main config.
func ProvideIDEVSCodeConfig(config *types.Config) *ide.VSCodeConfig {
	return &ide.VSCodeConfig{
		Port:       config.IDE.VSCode.Port,
		PluginName: config.IDE.VSCode.PluginName,
	}
}

// ProvideIDECursorConfig loads the Cursor IDE config from the main config.
func ProvideIDECursorConfig(config *types.Config) *ide.CursorConfig {
	return &ide.CursorConfig{
		Port: config.IDE.Cursor.Port,
	}
}

// ProvideIDEWindsurfConfig loads the Windsurf IDE config from the main config.
func ProvideIDEWindsurfConfig(config *types.Config) *ide.WindsurfConfig {
	return &ide.WindsurfConfig{
		Port: config.IDE.Windsurf.Port,
	}
}

// ProvideIDEJetBrainsConfig loads the IdeType IDE config from the main config.
func ProvideIDEJetBrainsConfig(config *types.Config) *ide.JetBrainsIDEConfig {
	return &ide.JetBrainsIDEConfig{
		IntelliJPort: config.IDE.Intellij.Port,
		GolandPort:   config.IDE.Goland.Port,
		PyCharmPort:  config.IDE.PyCharm.Port,
		WebStormPort: config.IDE.WebStorm.Port,
		PHPStormPort: config.IDE.PHPStorm.Port,
		CLionPort:    config.IDE.CLion.Port,
		RubyMinePort: config.IDE.RubyMine.Port,
		RiderPort:    config.IDE.Rider.Port,
	}
}

// ProvideGitspaceOrchestratorConfig loads the Gitspace orchestrator config from the main config.
func ProvideGitspaceOrchestratorConfig(config *types.Config) *orchestrator.Config {
	return &orchestrator.Config{
		DefaultBaseImage: config.Gitspace.DefaultBaseImage,
	}
}

// ProvideGitspaceInfraProvisionerConfig loads the Gitspace infra provisioner config from the main config.
func ProvideGitspaceInfraProvisionerConfig(config *types.Config) *infrastructure.Config {
	return &infrastructure.Config{
		AgentPort: config.Gitspace.AgentPort,
	}
}
