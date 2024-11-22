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

package aiagent

import (
	"context"
	"fmt"

	capabilitiesctrl "github.com/harness/gitness/app/api/controller/capabilities"
	"github.com/harness/gitness/app/auth/authz"
	"github.com/harness/gitness/app/services/capabilities"
	"github.com/harness/gitness/genai"
	"github.com/harness/gitness/types"
	capabilitytypes "github.com/harness/gitness/types/capabilities"

	"github.com/google/uuid"
)

type HarnessIntelligence struct {
	authorizer authz.Authorizer
	cr         *capabilities.Registry
	cc         *capabilitiesctrl.Controller
}

type Pipeline interface {
	Generate(ctx context.Context, req *types.PipelineGenerateRequest) (*types.PipelineGenerateResponse, error)
}

func capabilityResponseToChatContext(
	ranCapabilities *capabilitiesctrl.CapabilityRunResponse) []capabilitytypes.AIContext {
	var aiContexts []capabilitytypes.AIContext
	for _, value := range ranCapabilities.CapabilitiesRan {
		aiContext := capabilitytypes.AIContext{
			Type:    capabilitytypes.AIContextPayloadType("other"),
			Payload: value.Result,
			Name:    string(value.Type),
		}
		aiContexts = append(aiContexts, aiContext)
	}
	return aiContexts
}

func (s *HarnessIntelligence) Generate(
	ctx context.Context,
	req *types.PipelineGenerateRequest,
	repo *types.Repository) (*types.PipelineGenerateResponse, error) {
	if req.RepoRef == "" {
		return nil, fmt.Errorf("no repo ref specified")
	}

	conversationID := uuid.New()
	chatRequest := &genai.ChatRequest{
		Prompt:          req.Prompt,
		ConversationID:  conversationID.String(),
		ConversationRaw: "",
		Context: genai.GenerateAIContext(
			genai.RepoRef{
				Ref: repo.Path,
			},
		),
		Capabilities: s.cr.Capabilities(),
	}

	resp, err := s.CapabilitiesLoop(ctx, chatRequest)
	if err != nil {
		return nil, err
	}

	var yaml string
	for _, value := range resp.Context {
		out, ok := value.Payload.(*capabilities.DisplayYamlOutput)
		if ok {
			yaml = out.Yaml
		}
	}
	return &types.PipelineGenerateResponse{
		YAML: yaml,
	}, nil
}

func (s *HarnessIntelligence) GenerateStep(
	ctx context.Context,
	req *types.PipelineStepGenerateRequest,
	repo *types.Repository) (*types.PipelineStepGenerateResponse, error) {
	if req.RepoRef == "" {
		return nil, fmt.Errorf("no repo ref specified")
	}

	conversationID := uuid.New()
	chatRequest := &genai.ChatRequest{
		Prompt:          req.Prompt,
		ConversationID:  conversationID.String(),
		ConversationRaw: "",
		Context: genai.GenerateAIContext(
			genai.RepoRef{
				Ref: repo.Path,
			},
		),
		Capabilities: s.cr.Capabilities(),
	}

	resp, err := s.CapabilitiesLoop(ctx, chatRequest)
	if err != nil {
		return nil, err
	}

	var yaml string
	for _, value := range resp.Context {
		out, ok := value.Payload.(*capabilities.DisplayYamlOutput)
		if ok {
			yaml = out.Yaml
		}
	}
	return &types.PipelineStepGenerateResponse{
		YAML: yaml,
	}, nil
}

// TODO fix naming
type PipelineYaml struct {
	Yaml string `yaml:"yaml"`
}

// CapabilitiesLoop TODO: this should be replaced with an async model for Harness Enterprise, but remain for Harness.
func (s *HarnessIntelligence) CapabilitiesLoop(
	ctx context.Context, req *genai.ChatRequest) (*genai.ChatRequest, error) {
	returnToUser := false
	for !returnToUser {
		capToRun, err := genai.CallAIFoundation(ctx, s.cr, req)
		if err != nil {
			return nil, fmt.Errorf("failed to call local chat: %w", err)
		}

		resp, err := s.cc.RunCapabilities(ctx, capToRun)
		if err != nil {
			return nil, fmt.Errorf("failed to run capabilities: %w", err)
		}

		prevChatRequest := req
		req = &genai.ChatRequest{
			Prompt:          "",
			ConversationID:  prevChatRequest.ConversationID,
			ConversationRaw: capToRun.ConversationRaw,
			Context:         capabilityResponseToChatContext(resp),
			Capabilities:    s.cr.Capabilities(),
		}

		for _, value := range resp.CapabilitiesRan {
			if value.ReturnToUser {
				returnToUser = true
			}
		}
	}
	return req, nil
}

func (s *HarnessIntelligence) Update(
	ctx context.Context,
	req *types.PipelineUpdateRequest, repo *types.Repository) (*types.PipelineUpdateResponse, error) {
	if req.RepoRef == "" {
		return nil, fmt.Errorf("no repo ref specified")
	}

	conversationID := uuid.New()
	chatRequest := &genai.ChatRequest{
		Prompt:          req.Prompt,
		ConversationID:  conversationID.String(),
		ConversationRaw: "",
		Context: genai.GenerateAIContext(
			genai.RepoRef{
				Ref: repo.Path,
			},
			genai.PipelineContext{
				Yaml: req.Pipeline,
			},
		),
		Capabilities: s.cr.Capabilities(),
	}

	resp, err := s.CapabilitiesLoop(ctx, chatRequest)
	if err != nil {
		return nil, err
	}

	var yaml string
	for _, value := range resp.Context {
		out, ok := value.Payload.(*capabilities.DisplayYamlOutput)
		if ok {
			yaml = out.Yaml
		}
	}

	updateResponse := &types.PipelineUpdateResponse{
		YAML: yaml,
	}
	return updateResponse, nil
}

func (s *HarnessIntelligence) Suggest(
	_ context.Context,
	_ *types.PipelineSuggestionsRequest) (*types.PipelineSuggestionsResponse, error) {
	return &types.PipelineSuggestionsResponse{
		Suggestions: []types.Suggestion{},
	}, nil
}
