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

package types

import (
	"encoding/json"
	"errors"
	"strings"
)

type DevcontainerConfig struct {
	Image             string                           `json:"image"`
	PostCreateCommand LifecycleCommand                 `json:"postCreateCommand"` //nolint:tagliatelle
	PostStartCommand  LifecycleCommand                 `json:"postStartCommand"`  //nolint:tagliatelle
	ForwardPorts      []json.Number                    `json:"forwardPorts"`      //nolint:tagliatelle
	ContainerEnv      map[string]string                `json:"containerEnv"`      //nolint:tagliatelle
	Customizations    DevContainerConfigCustomizations `json:"customizations"`
}

// LifecycleCommand supports multiple formats for lifecycle commands.
type LifecycleCommand struct {
	CommandString string
	CommandArray  []string
	CommandList   map[string]string // Map to store commands by tags
}

// UnmarshalJSON custom unmarshal method for LifecycleCommand.
func (lc *LifecycleCommand) UnmarshalJSON(data []byte) error {
	// Try to unmarshal as a single string
	var commandStr string
	if err := json.Unmarshal(data, &commandStr); err == nil {
		lc.CommandString = commandStr
		return nil
	}

	// Try to unmarshal as an array of strings
	var commandArr []string
	if err := json.Unmarshal(data, &commandArr); err == nil {
		lc.CommandArray = commandArr
		return nil
	}

	// Try to unmarshal as a map of commands (tags to commands)
	var commandMap map[string]interface{}
	if err := json.Unmarshal(data, &commandMap); err == nil {
		validatedCommands := make(map[string]string)
		for tag, value := range commandMap {
			switch v := value.(type) {
			case string:
				validatedCommands[tag] = v
			case []interface{}:
				var strArray []string
				for _, item := range v {
					if str, ok := item.(string); ok {
						strArray = append(strArray, str)
					} else {
						return errors.New("invalid array type in command map")
					}
				}
				validatedCommands[tag] = strings.Join(strArray, " ")
			default:
				return errors.New("map values must be string or []string")
			}
		}
		lc.CommandList = validatedCommands
		return nil
	}

	return errors.New("invalid format: must be string, []string, or map[string]string | map[string][]string")
}

// ToCommandArray converts the LifecycleCommand into a slice of full commands.
func (lc *LifecycleCommand) ToCommandArray() []string {
	switch {
	case lc.CommandString != "":
		return []string{lc.CommandString}
	case lc.CommandArray != nil:
		return []string{strings.Join(lc.CommandArray, " ")}
	case lc.CommandList != nil:
		var commands []string
		for _, command := range lc.CommandList {
			commands = append(commands, command)
		}
		return commands
	default:
		return nil
	}
}
