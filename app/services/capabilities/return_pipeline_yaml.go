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

var ReturnPipelineYamlType capabilities.Type = "return_pipeline_yaml"
var ReturnPipelineYamlVersion capabilities.Version = "0"

type ReturnPipelineYamlInput struct {
	Yaml string `json:"pipeline_yaml"`
}

func (ReturnPipelineYamlInput) IsCapabilityInput() {}

type ReturnPipelineYamlOutput struct {
	Yaml string `json:"pipeline_yaml"`
}

func (ReturnPipelineYamlOutput) IsCapabilityOutput() {}

const AIContextPayloadTypeReturnPipelineYaml capabilities.AIContextPayloadType = "other"

func (ReturnPipelineYamlOutput) GetType() capabilities.AIContextPayloadType {
	return AIContextPayloadTypeReturnPipelineYaml
}

func (ReturnPipelineYamlOutput) GetName() string {
	return string(ReturnPipelineYamlType)
}

func (r *Registry) RegisterReturnPipelineYamlCapability(
	logic func(ctx context.Context, input *ReturnPipelineYamlInput) (*ReturnPipelineYamlOutput, error),
) error {
	return r.register(
		capabilities.Capability{
			Type:         ReturnPipelineYamlType,
			NewInput:     func() capabilities.Input { return &ReturnPipelineYamlInput{} },
			Logic:        newLogic(logic),
			Version:      ReturnPipelineYamlVersion,
			ReturnToUser: true,
		},
	)
}

// ReturnPipelineYaml could take in, eg repoStore store.RepoStore, git git.Interface, as arguments.
func ReturnPipelineYaml() func(ctx context.Context, input *ReturnPipelineYamlInput) (*ReturnPipelineYamlOutput, error) {
	return func(_ context.Context, input *ReturnPipelineYamlInput) (*ReturnPipelineYamlOutput, error) {
		return &ReturnPipelineYamlOutput{
			Yaml: input.Yaml,
		}, nil
	}
}
