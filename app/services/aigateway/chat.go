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

package aigateway

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/rs/zerolog/log"
)

const chatPath = "/v1/chat/completions"

var ErrEmptyCompletion = errors.New("completion has no choices")

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatRequest struct {
	Model       string        `json:"model"`
	Messages    []ChatMessage `json:"messages"`
	MaxTokens   int           `json:"max_tokens,omitempty"`
	Temperature *float64      `json:"temperature,omitempty"`
}

type chatWireRequest struct {
	ChatRequest
	Provider *ProviderPreferences `json:"provider"`
	Usage    struct {
		Include bool `json:"include"`
	} `json:"usage"`
}

type ChatUsage struct {
	PromptTokens     int     `json:"prompt_tokens"`
	CompletionTokens int     `json:"completion_tokens"`
	Cost             float64 `json:"cost"`
}

type ChatResponse struct {
	ID      string `json:"id"`
	Model   string `json:"model"`
	Choices []struct {
		Message      ChatMessage `json:"message"`
		FinishReason string      `json:"finish_reason"`
	} `json:"choices"`
	Usage ChatUsage `json:"usage"`
}

// Content returns the text of the first choice.
func (r *ChatResponse) Content() string {
	if len(r.Choices) == 0 {
		return ""
	}
	return r.Choices[0].Message.Content
}

// Chat sends a redacted chat completion to a free model. There is no fallback to paid models:
// if the model is not free, or no quota is left, the call fails and the caller decides whether to wait.
func (g *Gateway) Chat(ctx context.Context, class Class, req ChatRequest) (*ChatResponse, error) {
	if !g.Enabled() {
		return nil, ErrDisabled
	}
	if err := g.guard.Check(req.Model); err != nil {
		return nil, err
	}
	if len(req.Messages) == 0 {
		return nil, fmt.Errorf("chat request has no messages")
	}

	wire := chatWireRequest{ChatRequest: req, Provider: FreeOnlyProvider()}
	wire.Usage.Include = true
	wire.Messages = make([]ChatMessage, len(req.Messages))
	for i, m := range req.Messages {
		wire.Messages[i] = ChatMessage{Role: m.Role, Content: g.redactor.Redact(m.Content)}
	}

	if err := g.planner.Reserve(class); err != nil {
		return nil, err
	}

	var resp ChatResponse
	if err := g.client.doJSON(ctx, http.MethodPost, chatPath, wire, &resp); err != nil {
		return nil, err
	}
	if err := g.guard.VerifyCost(req.Model, resp.Usage.Cost); err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("ai gateway: free-only guarantee violated")
		return nil, err
	}
	if len(resp.Choices) == 0 {
		return nil, ErrEmptyCompletion
	}
	return &resp, nil
}

// AgentEnv returns the environment that points a coding agent (Claude Code) at OpenRouter
// with a free model. The model is checked against the guard like any other request.
func (g *Gateway) AgentEnv(model string) (map[string]string, error) {
	if !g.Enabled() {
		return nil, ErrDisabled
	}
	if err := g.guard.Check(model); err != nil {
		return nil, err
	}
	if m, ok := g.catalog.Get(model); ok && !m.SupportsTools() {
		return nil, fmt.Errorf("model %s doesn't support tool calling", model)
	}
	return map[string]string{
		"ANTHROPIC_BASE_URL":   g.config.BaseURL,
		"ANTHROPIC_AUTH_TOKEN": g.config.APIKey,
		"ANTHROPIC_API_KEY":    "",
		"ANTHROPIC_MODEL":      model,
	}, nil
}
