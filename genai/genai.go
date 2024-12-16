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

package genai

import (
	capabilities2 "github.com/harness/gitness/types/capabilities"
)

const (
	AIFoundationServiceToken = "AI_FOUNDATION_SERVICE_TOKEN"
)

func GenerateAIContext(payloads ...capabilities2.AIContextPayload) []capabilities2.AIContext {
	out := make([]capabilities2.AIContext, len(payloads))
	for i := range payloads {
		out[i] = capabilities2.AIContext{
			Type:    payloads[i].GetType(),
			Payload: payloads[i],
			Name:    payloads[i].GetName(),
		}
	}
	return out
}

var _ capabilities2.AIContextPayload = (*PipelineContext)(nil)
var _ capabilities2.AIContextPayload = (*StepContext)(nil)
var _ capabilities2.AIContextPayload = (*RepoRef)(nil)

type PipelineContext struct {
	Yaml string `json:"pipeline_yaml"`
}

func (c PipelineContext) GetName() string {
	return "pipeline_context"
}

const AIContextPayloadTypePipelineContext capabilities2.AIContextPayloadType = "other"

func (PipelineContext) GetType() capabilities2.AIContextPayloadType {
	return AIContextPayloadTypePipelineContext
}

const AIContextPayloadTypeStepContext capabilities2.AIContextPayloadType = "other"

type StepContext struct {
	Yaml string `json:"step_yaml"`
}

func (StepContext) GetName() string {
	return "step_context"
}

func (StepContext) GetType() capabilities2.AIContextPayloadType {
	return AIContextPayloadTypeStepContext
}

type RepoRef struct {
	Ref string `json:"ref"`
}

func (r RepoRef) GetName() string {
	return "repo_ref"
}

const AIContextPayloadTypeRepoRef capabilities2.AIContextPayloadType = "other"

func (RepoRef) GetType() capabilities2.AIContextPayloadType {
	return AIContextPayloadTypeRepoRef
}
