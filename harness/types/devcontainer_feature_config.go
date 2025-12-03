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
	"fmt"
	"slices"
	"strconv"

	"github.com/harness/gitness/types/enum"
)

//nolint:tagliatelle
type DevcontainerFeatureConfig struct {
	ID                string            `json:"id,omitempty"`
	Version           string            `json:"version,omitempty"`
	Name              string            `json:"name,omitempty"`
	Options           *Options          `json:"options,omitempty"`
	DependsOn         *Features         `json:"dependsOn,omitempty"`
	ContainerEnv      map[string]string `json:"containerEnv,omitempty"`
	Privileged        bool              `json:"privileged,omitempty"`
	Init              bool              `json:"init,omitempty"`
	CapAdd            []string          `json:"capAdd,omitempty"`
	SecurityOpt       []string          `json:"securityOpt,omitempty"`
	Entrypoint        string            `json:"entrypoint,omitempty"`
	InstallsAfter     []string          `json:"installsAfter,omitempty"`
	Mounts            []*Mount          `json:"mounts,omitempty"`
	PostCreateCommand LifecycleCommand  `json:"postCreateCommand,omitempty"`
	PostStartCommand  LifecycleCommand  `json:"postStartCommand,omitempty"`
}

type Options map[string]*OptionDefinition

type OptionDefinition struct {
	Type        enum.FeatureOptionValueType `json:"type,omitempty"`
	Proposals   []string                    `json:"proposals,omitempty"`
	Enum        []string                    `json:"enum,omitempty"`
	Default     any                         `json:"default,omitempty"`
	Description string                      `json:"description,omitempty"`
}

// ValidateValue checks if the value matches the type defined in the definition. For string types,
// it also checks if it is allowed ie it is present in the enum array for the option.
// Reference: https://containers.dev/implementors/features/#options-property
func (o *OptionDefinition) ValidateValue(optionValue any, optionKey string, featureSource string) (string, error) {
	switch o.Type {
	case enum.FeatureOptionValueTypeBoolean:
		boolValue, ok := optionValue.(bool)
		if ok {
			return strconv.FormatBool(boolValue), nil
		}
		stringValue, ok := optionValue.(string)
		if ok {
			parsedBoolValue, err := strconv.ParseBool(stringValue)
			if err == nil {
				return strconv.FormatBool(parsedBoolValue), nil
			}
		}
		return "", fmt.Errorf("error during resolving feature %s, option Id %s "+
			"expects boolean, got %s ", featureSource, optionKey, optionValue)
	case enum.FeatureOptionValueTypeString:
		stringValue, ok := optionValue.(string)
		if !ok {
			return "", fmt.Errorf("error during resolving feature %s, option Id %s "+
				"expects string, got %s ", featureSource, optionKey, optionValue)
		}
		if len(o.Enum) > 0 && !slices.Contains(o.Enum, stringValue) {
			return "", fmt.Errorf("error during resolving feature %s, option value %s "+
				"not allowed for Id %s ", featureSource, stringValue, optionKey)
		}
		return stringValue, nil
	default:
		return "", fmt.Errorf("unsupported option type %s", o.Type)
	}
}
