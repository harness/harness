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

type GeneratePipelineStageInput struct {
	Prompt       string            `json:"prompt"`
	Metadata     map[string]string `json:"metadata"`
	Conversation []Conversation    `json:"conversation"`
}

type PipelineStageData struct {
	StageYaml string `json:"yaml_stage"`
}

type GeneratePipelineStageOutput struct {
	Error string            `json:"error"`
	Data  PipelineStageData `json:"data"`
}

func (in *GeneratePipelineStageInput) GetConversation() []Conversation {
	return in.Conversation
}

func (in *GeneratePipelineStageInput) GetPrompt() string {
	return in.Prompt
}

func (in *GeneratePipelineStageInput) GetValidationPrompt() string {
	return "Create a stage-yaml with the following query: " + in.GetPrompt()
}
