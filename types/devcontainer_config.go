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
	"fmt"
	"net/url"
	"strings"

	"github.com/harness/gitness/types/enum"

	"oras.land/oras-go/v2/registry"
)

const FeatureDefaultTag = "latest"

//nolint:tagliatelle
type DevcontainerConfig struct {
	Image                       string                           `json:"image,omitempty"`
	PostCreateCommand           LifecycleCommand                 `json:"postCreateCommand,omitempty"`
	PostStartCommand            LifecycleCommand                 `json:"postStartCommand,omitempty"`
	ForwardPorts                []json.Number                    `json:"forwardPorts,omitempty"`
	ContainerEnv                map[string]string                `json:"containerEnv,omitempty"`
	Customizations              DevContainerConfigCustomizations `json:"customizations,omitempty"`
	RunArgs                     []string                         `json:"runArgs,omitempty"`
	ContainerUser               string                           `json:"containerUser,omitempty"`
	RemoteUser                  string                           `json:"remoteUser,omitempty"`
	Features                    *Features                        `json:"features,omitempty"`
	OverrideFeatureInstallOrder []string                         `json:"overrideFeatureInstallOrder,omitempty"`
	Privileged                  bool                             `json:"privileged,omitempty"`
	Init                        bool                             `json:"init,omitempty"`
	CapAdd                      []string                         `json:"capAdd,omitempty"`
	SecurityOpt                 []string                         `json:"securityOpt,omitempty"`
	Mounts                      []*Mount                         `json:"mounts,omitempty"`
}

// Constants for discriminator values.
const (
	TypeString     = "string"
	TypeArray      = "array"
	TypeCommandMap = "commandMap"
)

//nolint:tagliatelle
type LifecycleCommand struct {
	CommandString string         `json:"commandString,omitempty"`
	CommandArray  []string       `json:"commandArray,omitempty"`
	CommandMap    map[string]any `json:"commandMap,omitempty"`
	Discriminator string         `json:"-"` // Tracks the original type for proper re-marshaling
}

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

	// Try to unmarshal as a map with mixed types
	var rawMap map[string]any
	if err := json.Unmarshal(data, &rawMap); err == nil {
		for key, value := range rawMap {
			switch v := value.(type) {
			case string:
				// Valid string value
			case []interface{}:
				// Convert []interface{} to []string
				var strArray []string
				for _, item := range v {
					if str, ok := item.(string); ok {
						strArray = append(strArray, str)
					} else {
						return fmt.Errorf("invalid format: array contains non-string value")
					}
				}
				rawMap[key] = strArray
			default:
				return fmt.Errorf("invalid format: map contains unsupported type")
			}
		}
		lc.CommandMap = rawMap
		lc.Discriminator = TypeCommandMap
		return nil
	}

	return fmt.Errorf("invalid format: must be string, []string, or map[string]any")
}

func (lc *LifecycleCommand) MarshalJSON() ([]byte, error) {
	// If Discriminator is empty, return an empty JSON object (i.e., {} or no content)
	if lc.Discriminator == "" || lc == nil {
		return []byte("{}"), nil
	}
	switch lc.Discriminator {
	case TypeString:
		return json.Marshal(lc.CommandString)
	case TypeArray:
		return json.Marshal(lc.CommandArray)
	case TypeCommandMap:
		return json.Marshal(lc.CommandMap)
	default:
		return nil, fmt.Errorf("unknown type for LifecycleCommand")
	}
}

// ToCommandArray converts the LifecycleCommand into a slice of full commands.
func (lc *LifecycleCommand) ToCommandArray() []string {
	// If Discriminator is empty, return nil
	if lc.Discriminator == "" || lc == nil {
		return nil
	}
	switch lc.Discriminator {
	case TypeString:
		return []string{lc.CommandString}
	case TypeArray:
		return []string{strings.Join(lc.CommandArray, " ")}
	case TypeCommandMap:
		var commands []string
		for _, value := range lc.CommandMap {
			switch v := value.(type) {
			case string:
				commands = append(commands, v)
			case []string:
				commands = append(commands, strings.Join(v, " "))
			}
		}
		return commands
	default:
		return nil
	}
}

type Features map[string]*FeatureValue

type FeatureValue struct {
	Source     string                 `json:"source,omitempty"`
	SourceType enum.FeatureSourceType `json:"source_type,omitempty"`
	Options    map[string]any         `json:"options,omitempty"`
}

func (f *FeatureValue) UnmarshalJSON(data []byte) error {
	var version string
	if err := json.Unmarshal(data, &version); err == nil {
		f.Options = make(map[string]any)
		f.Options["version"] = version
		return nil
	}

	var options map[string]any
	if err := json.Unmarshal(data, &options); err == nil {
		for key, value := range options {
			switch value.(type) {
			case string, bool:
				continue
			default:
				return fmt.Errorf("invalid type for option '%s': must be string or boolean, got %T", key, value)
			}
		}
		f.Options = options
		return nil
	}

	return nil
}

func (f *Features) UnmarshalJSON(data []byte) error {
	if *f == nil {
		*f = make(Features)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	for key, value := range raw {
		sanitizedSource, sourceType, validationErr := validateFeatureSource(key)
		if validationErr != nil {
			return validationErr
		}
		feature := &FeatureValue{Source: sanitizedSource, SourceType: sourceType}
		if err := json.Unmarshal(value, feature); err != nil {
			return fmt.Errorf("failed to unmarshal feature '%s': %w", key, err)
		}
		(*f)[sanitizedSource] = feature
	}

	return nil
}

func validateFeatureSource(source string) (string, enum.FeatureSourceType, error) {
	if _, err := registry.ParseReference(source); err == nil {
		indexOfSeparator := strings.Index(source, ":")
		if indexOfSeparator == -1 {
			source += ":" + FeatureDefaultTag
		}
		return source, enum.FeatureSourceTypeOCI, nil
	}
	if err := validateTarballURL(source); err == nil {
		return source, enum.FeatureSourceTypeTarball, nil
	}
	return source, enum.FeatureSourceTypeLocal, fmt.Errorf("unsupported feature source: %s", source)
}

func validateTarballURL(source string) error {
	tarballURL, err := url.Parse(source)
	if err != nil {
		return fmt.Errorf("parsing feature URL: %w", err)
	}
	if tarballURL.Scheme != "http" && tarballURL.Scheme != "https" {
		return fmt.Errorf("invalid feature URL: %s", tarballURL.String())
	}
	if !strings.HasSuffix(tarballURL.Path, ".tgz") {
		return fmt.Errorf("invalid feature URL: %s", tarballURL.String())
	}
	return nil
}

type Mount struct {
	Source string `json:"source,omitempty"`
	Target string `json:"target,omitempty"`
	Type   string `json:"type,omitempty"`
}

func (m *Mount) UnmarshalJSON(data []byte) error {
	// TODO: Add support for unmarshalling mount data from a string input
	var mount Mount
	err := json.Unmarshal(data, &mount)
	if err != nil {
		return err
	}
	return nil
}
