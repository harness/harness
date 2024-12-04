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
	"path/filepath"

	"github.com/harness/gitness/types"

	types2 "github.com/docker/docker/api/types"
)

const (
	linuxHome               = "/home"
	deprecatedRemoteUser    = "harness"
	gitspaceRemoteUserLabel = "gitspace.remote.user"
)

func GetGitspaceContainerName(config types.GitspaceConfig) string {
	return "gitspace-" + config.GitspaceUser.Identifier + "-" + config.Identifier
}

func GetUserHomeDir(userIdentifier string) string {
	if userIdentifier == "root" {
		return "/root"
	}
	return filepath.Join(linuxHome, userIdentifier)
}

func GetImage(devcontainerConfig types.DevcontainerConfig, defaultBaseImage string) string {
	imageName := devcontainerConfig.Image
	if imageName == "" {
		imageName = defaultBaseImage
	}
	return imageName
}

func GetContainerUser(
	runArgsMap map[types.RunArg]*types.RunArgValue,
	devcontainerConfig types.DevcontainerConfig,
	metadataFromImage map[string]any,
	imageUser string,
) string {
	if containerUser := getUser(runArgsMap); containerUser != "" {
		return containerUser
	}
	if devcontainerConfig.ContainerUser != "" {
		return devcontainerConfig.ContainerUser
	}
	if containerUser, ok := metadataFromImage["containerUser"].(string); ok {
		return containerUser
	}
	return imageUser
}

func ExtractRemoteUserFromLabels(inspectResp types2.ContainerJSON) string {
	remoteUser := deprecatedRemoteUser

	if remoteUserValue, ok := inspectResp.Config.Labels[gitspaceRemoteUserLabel]; ok {
		remoteUser = remoteUserValue
	}
	return remoteUser
}

func GetRemoteUser(
	devcontainerConfig types.DevcontainerConfig,
	metadataFromImage map[string]any,
	containerUser string,
) string {
	if devcontainerConfig.RemoteUser != "" {
		return devcontainerConfig.RemoteUser
	}
	if remoteUser, ok := metadataFromImage["remoteUser"].(string); ok {
		return remoteUser
	}
	return containerUser
}
