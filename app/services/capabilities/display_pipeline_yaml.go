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

var DisplayPipelineYamlType capabilities.Type = "display_pipeline_yaml"
var DisplayPipelineYamlVersion capabilities.Version = "0"

type DisplayPipelineYamlInput struct {
	Yaml string `json:"pipeline_yaml"`
}

func (DisplayPipelineYamlInput) IsCapabilityInput() {}

type DisplayPipelineYamlOutput struct {
	Yaml string `json:"pipeline_yaml"`
}

func (DisplayPipelineYamlOutput) IsCapabilityOutput() {}

const AIContextPayloadTypeDisplayPipelineYaml capabilities.AIContextPayloadType = "other"

func (DisplayPipelineYamlOutput) GetType() capabilities.AIContextPayloadType {
	return AIContextPayloadTypeDisplayPipelineYaml
}

func (DisplayPipelineYamlOutput) GetName() string {
	return string(DisplayPipelineYamlType)
}

func (r *Registry) RegisterDisplayPipelineYamlCapability(
	logic func(ctx context.Context, input *DisplayPipelineYamlInput) (*DisplayPipelineYamlOutput, error),
) error {
	return r.register(
		capabilities.Capability{
			Type:         DisplayPipelineYamlType,
			NewInput:     func() capabilities.Input { return &DisplayPipelineYamlInput{} },
			Logic:        newLogic(logic),
			Version:      DisplayPipelineYamlVersion,
			ReturnToUser: true,
		},
	)
}

// ReturnPipelineYaml could take in, eg repoStore store.RepoStore, git git.Interface, as arguments.
func DisplayPipelineYaml() func(
	ctx context.Context,
	input *DisplayPipelineYamlInput) (*DisplayPipelineYamlOutput, error) {
	return func(_ context.Context, input *DisplayPipelineYamlInput) (*DisplayPipelineYamlOutput, error) {
		return &DisplayPipelineYamlOutput{
			Yaml: input.Yaml,
		}, nil
	}
}
