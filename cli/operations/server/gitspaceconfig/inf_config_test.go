// Copyright 2026 Harness, Inc.
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

package gitspaceconfig

import (
	"testing"

	"github.com/harness/gitness/types"

	"github.com/stretchr/testify/require"
)

func TestInfProvideDockerConfig(t *testing.T) {
	t.Run("host name is derived from the base url", func(t *testing.T) {
		cfg := &types.Config{}
		cfg.URL.Base = "https://git.example.invalid:8443/some/path"
		cfg.Docker.Host = "tcp://docker:2375"
		cfg.Docker.APIVersion = "1.44"
		cfg.Docker.CertPath = "/certs"
		cfg.Docker.TLSVerify = "1"

		got, err := ProvideDockerConfig(cfg)
		require.NoError(t, err)
		require.Equal(t, "git.example.invalid", got.DockerMachineHostName)
		require.Equal(t, "git.example.invalid", cfg.Docker.MachineHostName, "the config is updated in place")
		require.Equal(t, "tcp://docker:2375", got.DockerHost)
		require.Equal(t, "1.44", got.DockerAPIVersion)
		require.Equal(t, "/certs", got.DockerCertPath)
		require.Equal(t, "1", got.DockerTLSVerify)
	})
	t.Run("explicit machine host name wins", func(t *testing.T) {
		cfg := &types.Config{}
		cfg.URL.Base = "https://git.example.invalid"
		cfg.Docker.MachineHostName = "machine.example.invalid"
		got, err := ProvideDockerConfig(cfg)
		require.NoError(t, err)
		require.Equal(t, "machine.example.invalid", got.DockerMachineHostName)
	})
	t.Run("invalid base url", func(t *testing.T) {
		cfg := &types.Config{}
		cfg.URL.Base = "://bad url"
		_, err := ProvideDockerConfig(cfg)
		require.ErrorContains(t, err, "unable to parse Harness base URL")
	})
}

func TestInfProvideIDEConfigs(t *testing.T) {
	cfg := &types.Config{}
	cfg.IDE.VSCodeWeb.Port = 1
	cfg.IDE.VSCode.Port = 2
	cfg.IDE.VSCode.PluginName = "plugin"
	cfg.IDE.Cursor.Port = 3
	cfg.IDE.Windsurf.Port = 4
	cfg.IDE.Intellij.Port = 5
	cfg.IDE.Goland.Port = 6
	cfg.IDE.PyCharm.Port = 7
	cfg.IDE.WebStorm.Port = 8
	cfg.IDE.PHPStorm.Port = 9
	cfg.IDE.CLion.Port = 10
	cfg.IDE.RubyMine.Port = 11
	cfg.IDE.Rider.Port = 12

	require.Equal(t, 1, ProvideIDEVSCodeWebConfig(cfg).Port)

	vscode := ProvideIDEVSCodeConfig(cfg)
	require.Equal(t, 2, vscode.Port)
	require.Equal(t, "plugin", vscode.PluginName)

	require.Equal(t, 3, ProvideIDECursorConfig(cfg).Port)
	require.Equal(t, 4, ProvideIDEWindsurfConfig(cfg).Port)

	jb := ProvideIDEJetBrainsConfig(cfg)
	require.Equal(t, 5, jb.IntelliJPort)
	require.Equal(t, 6, jb.GolandPort)
	require.Equal(t, 7, jb.PyCharmPort)
	require.Equal(t, 8, jb.WebStormPort)
	require.Equal(t, 9, jb.PHPStormPort)
	require.Equal(t, 10, jb.CLionPort)
	require.Equal(t, 11, jb.RubyMinePort)
	require.Equal(t, 12, jb.RiderPort)
}

func TestInfProvideGitspaceConfigs(t *testing.T) {
	cfg := &types.Config{}
	cfg.Gitspace.DefaultBaseImage = "example.invalid/base:1"
	cfg.Gitspace.AgentPort = 9000

	require.Equal(t, "example.invalid/base:1", ProvideGitspaceOrchestratorConfig(cfg).DefaultBaseImage)
	require.Equal(t, 9000, ProvideGitspaceInfraProvisionerConfig(cfg).AgentPort)
}
