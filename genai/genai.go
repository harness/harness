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
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"

	capabilitiesctrl "github.com/harness/gitness/app/api/controller/capabilities"
	capabilitieshandler "github.com/harness/gitness/app/api/handler/capabilities"
	"github.com/harness/gitness/app/services/capabilities"
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

type PipelineContext struct {
	Yaml string `json:"yaml"`
}

func (c PipelineContext) GetName() string {
	return "pipeline_context"
}

const AIContextPayloadTypePipelineContext capabilities2.AIContextPayloadType = "other"

func (PipelineContext) GetType() capabilities2.AIContextPayloadType {
	return AIContextPayloadTypePipelineContext
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

type ChatRequest struct {
	Prompt          string                              `json:"prompt"`
	ConversationID  string                              `json:"conversation_id"`
	ConversationRaw string                              `json:"conversation_raw"`
	Context         []capabilities2.AIContext           `json:"context"`
	Capabilities    []capabilities2.CapabilityReference `json:"capabilities"`
}

func CallAIFoundation(ctx context.Context, cr *capabilities.Registry,
	req *ChatRequest) (*capabilitiesctrl.RunCapabilitiesRequest, error) {
	url := "http://localhost:8000/chat/gitness"

	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	newReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	newReq.Header.Add("Authorization", "Bearer "+os.Getenv(AIFoundationServiceToken))

	client := http.DefaultClient
	resp, err := client.Do(newReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	in, err := capabilitieshandler.UnmarshalRunCapabilitiesRequest(cr, data)
	if err != nil {
		return nil, err
	}

	return in, nil
}
