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

type PipelineData struct {
	PipelineYaml string `json:"yaml_pipeline"`
}

// create.
type GeneratePipelineInput struct {
	Prompt       string            `json:"prompt"`
	Metadata     map[string]string `json:"metadata"`
	Conversation []Conversation    `json:"conversation"`
}

type GeneratePipelineOutput struct {
	Error string       `json:"error"`
	Data  PipelineData `json:"data"`
}

// suggest.
type SuggestPipelineInput struct {
	Pipeline string `json:"pipeline"`
}

type SuggestPipelineOutput struct {
	Suggestions []Suggestion
}

// update.
type UpdatePipelineInput struct {
	Prompt   string `json:"prompt"`
	Pipeline string `json:"pipeline"`
}

type UpdatePipelineOutput struct {
	Error string       `json:"error"`
	Data  PipelineData `json:"data"`
}

func (in *GeneratePipelineInput) GetConversation() []Conversation {
	return in.Conversation
}

func (in *GeneratePipelineInput) GetPrompt() string {
	return in.Prompt
}

func (in *GeneratePipelineInput) GetValidationPrompt() string {
	return "Create a pipeline-yaml with the following query: " + in.GetPrompt()
}
