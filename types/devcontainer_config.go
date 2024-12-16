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

//nolint:tagliatelle
type DevcontainerConfig struct {
	Image             string                           `json:"image,omitempty"`
	PostCreateCommand LifecycleCommand                 `json:"postCreateCommand,omitempty"`
	PostStartCommand  LifecycleCommand                 `json:"postStartCommand,omitempty"`
	ForwardPorts      []json.Number                    `json:"forwardPorts,omitempty"`
	ContainerEnv      map[string]string                `json:"containerEnv,omitempty"`
	Customizations    DevContainerConfigCustomizations `json:"customizations,omitempty"`
	RunArgs           []string                         `json:"runArgs,omitempty"`
	ContainerUser     string                           `json:"containerUser,omitempty"`
	RemoteUser        string                           `json:"remoteUser,omitempty"`
}

//nolint:tagliatelle
type LifecycleCommand struct {
	CommandString string   `json:"commandString,omitempty"`
	CommandArray  []string `json:"commandArray,omitempty"`
	// Map to store commands by tags
	CommandMap      map[string]string   `json:"commandMap,omitempty"`
	CommandMapArray map[string][]string `json:"commandMapArray,omitempty"`
	Discriminator   string              `json:"-"` // Tracks the original type for proper re-marshaling
}

// Constants for discriminator values.
const (
	TypeString           = "string"
	TypeArray            = "array"
	TypeCommandMapString = "commandMap"
	TypeCommandMapArray  = "commandMapArray"
)

func (lc *LifecycleCommand) UnmarshalJSON(data []byte) error {
	// Try to unmarshal as a single string
	var commandStr string
	if err := json.Unmarshal(data, &commandStr); err == nil {
		lc.CommandString = commandStr
		lc.Discriminator = TypeString
		return nil
	}
	// Try to unmarshal as an array of strings
	var commandArr []string
	if err := json.Unmarshal(data, &commandArr); err == nil {
		lc.CommandArray = commandArr
		lc.Discriminator = TypeArray
		return nil
	}
	// Try to unmarshal as a map of commands (tags to commands)
	var commandMap map[string]string
	if err := json.Unmarshal(data, &commandMap); err == nil {
		lc.CommandMap = commandMap
		lc.Discriminator = TypeCommandMapString
		return nil
	}
	// Try to unmarshal as a CommandMapArray
	var commandMapArray map[string][]string
	if err := json.Unmarshal(data, &commandMapArray); err == nil {
		lc.CommandMapArray = commandMapArray
		lc.Discriminator = TypeCommandMapArray
		return nil
	}
	return errors.New("invalid format: must be string, []string, map[string]string, or map[string][]string")
}

func (lc *LifecycleCommand) MarshalJSON() ([]byte, error) {
	switch lc.Discriminator {
	case TypeString:
		return json.Marshal(lc.CommandString)
	case TypeArray:
		return json.Marshal(lc.CommandArray)
	case TypeCommandMapString:
		return json.Marshal(lc.CommandMap)
	case TypeCommandMapArray:
		return json.Marshal(lc.CommandMapArray)
	default:
		return nil, errors.New("unknown type for LifecycleCommand")
	}
}

// ToCommandArray converts the LifecycleCommand into a slice of full commands.
func (lc *LifecycleCommand) ToCommandArray() []string {
	switch {
	case lc.CommandString != "":
		return []string{lc.CommandString}
	case lc.CommandArray != nil:
		return []string{strings.Join(lc.CommandArray, " ")}
	case lc.CommandMap != nil:
		var commands []string
		for _, command := range lc.CommandMap {
			commands = append(commands, command)
		}
		return commands
	case lc.CommandMapArray != nil:
		var commands []string
		for _, commandArray := range lc.CommandMapArray {
			commands = append(commands, strings.Join(commandArray, " "))
		}
		return commands
	default:
		return nil
	}
}
