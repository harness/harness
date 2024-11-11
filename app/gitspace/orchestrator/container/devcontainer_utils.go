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
	"fmt"

	"github.com/harness/gitness/types"

	"github.com/rs/zerolog/log"
)

func ExtractEnv(devcontainerConfig types.DevcontainerConfig) []string {
	var env []string
	for key, value := range devcontainerConfig.ContainerEnv {
		env = append(env, fmt.Sprintf("%s=%s", key, value))
	}
	return env
}

func ExtractForwardPorts(devcontainerConfig types.DevcontainerConfig) []int {
	var ports []int
	for _, strPort := range devcontainerConfig.ForwardPorts {
		portAsInt, err := strPort.Int64() // Using Atoi to convert string to int
		if err != nil {
			log.Warn().Msgf("Error converting port string '%s' to int: %v", strPort, err)
			continue // Skip the invalid port
		}
		ports = append(ports, int(portAsInt))
	}
	return ports
}

func ExtractCommand(actionType PostAction, devcontainerConfig types.DevcontainerConfig) string {
	switch actionType {
	case PostCreateAction:
		return devcontainerConfig.PostCreateCommand
	case PostStartAction:
		return devcontainerConfig.PostStartCommand
	default:
		return "" // Return empty string if actionType is not recognized
	}
}
