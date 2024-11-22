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

package capabilities

import (
	"context"

	"github.com/harness/gitness/types/capabilities"
)

var DisplayYamlType capabilities.Type = "display_yaml"
var DisplayYamlVersion capabilities.Version = "0"

type DisplayYamlInput struct {
	Yaml string `json:"yaml"`
}

func (DisplayYamlInput) IsCapabilityInput() {}

type DisplayYamlOutput struct {
	Yaml string `json:"yaml"`
}

func (DisplayYamlOutput) IsCapabilityOutput() {}

const AIContextPayloadTypeDisplayPipelineYaml capabilities.AIContextPayloadType = "other"

func (DisplayYamlOutput) GetType() capabilities.AIContextPayloadType {
	return AIContextPayloadTypeDisplayPipelineYaml
}

func (DisplayYamlOutput) GetName() string {
	return string(DisplayYamlType)
}

func (r *Registry) RegisterDisplayYamlCapability(
	logic func(ctx context.Context, input *DisplayYamlInput) (*DisplayYamlOutput, error),
) error {
	return r.register(
		capabilities.Capability{
			Type:         DisplayYamlType,
			NewInput:     func() capabilities.Input { return &DisplayYamlInput{} },
			Logic:        newLogic(logic),
			Version:      DisplayYamlVersion,
			ReturnToUser: true,
		},
	)
}

// ReturnPipelineYaml could take in, eg repoStore store.RepoStore, git git.Interface, as arguments.
func DisplayYaml() func(
	ctx context.Context,
	input *DisplayYamlInput) (*DisplayYamlOutput, error) {
	return func(_ context.Context, input *DisplayYamlInput) (*DisplayYamlOutput, error) {
		return &DisplayYamlOutput{
			Yaml: input.Yaml,
		}, nil
	}
}
