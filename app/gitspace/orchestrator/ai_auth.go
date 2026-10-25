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

package orchestrator

import (
	"context"
	"errors"
	"fmt"

	"github.com/harness/gitness/app/gitspace/agentmodel"
	"github.com/harness/gitness/app/services/aigateway"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

const (
	// agentMinContext is the minimum context window required by a coding agent.
	agentMinContext = 32000

	agentEnvBaseURL = "ANTHROPIC_BASE_URL"
	agentEnvToken   = "ANTHROPIC_AUTH_TOKEN"
)

// ErrNoAgentModel is returned when the AI gateway is enabled but no free model with tool calling is available.
var ErrNoAgentModel = errors.New("no free model with tool calling is available for the AI agent")

// AgentModelSelector chooses the model a coding agent runs with when the AI gateway is enabled.
// It is a seam so a smarter selection can replace the deterministic default.
type AgentModelSelector interface {
	SelectModel(ctx context.Context) (string, error)
}

// ModelHints tells the model chosen for a gitspace config (see agentmodel.Registry).
type ModelHints interface {
	Get(gitspaceConfigID int64) (string, bool)
}

// hintedModelSelector uses the model chosen by the agent launcher for the gitspace config in the context
// (agentmodel.WithConfigID) and falls back to another selector when there is none.
type hintedModelSelector struct {
	hints    ModelHints
	fallback AgentModelSelector
}

func (s hintedModelSelector) SelectModel(ctx context.Context) (string, error) {
	if s.hints != nil {
		if id, ok := agentmodel.ConfigIDFrom(ctx); ok {
			if m, found := s.hints.Get(id); found && m != "" {
				return m, nil
			}
		}
	}
	return s.fallback.SelectModel(ctx)
}

// agentGateway is the subset of the AI gateway used to configure a coding agent.
type agentGateway interface {
	Enabled() bool
	AgentEnv(model string) (map[string]string, error)
}

// deterministicModelSelector picks the first free tool-calling model (sorted by ID) of the catalog.
type deterministicModelSelector struct {
	candidates func(aigateway.Requirements) []aigateway.Model
}

func newDeterministicModelSelector(gateway *aigateway.Gateway) AgentModelSelector {
	return deterministicModelSelector{
		candidates: func(req aigateway.Requirements) []aigateway.Model {
			if !gateway.Enabled() || gateway.Catalog() == nil {
				return nil
			}
			return gateway.Catalog().Candidates(req)
		},
	}
}

func (s deterministicModelSelector) SelectModel(_ context.Context) (string, error) {
	candidates := s.candidates(aigateway.Requirements{Tools: true, MinContext: agentMinContext})
	if len(candidates) == 0 {
		return "", ErrNoAgentModel
	}
	return candidates[0].ID, nil
}

// openRouterAIAuth builds the OpenRouter auth of the AI agent. The model is validated by the gateway
// (free and tool calling). The key is wrapped in a MaskSecret so it is never printed.
func openRouterAIAuth(
	ctx context.Context,
	gateway agentGateway,
	selector AgentModelSelector,
) (map[enum.AIAgent]types.AIAgentAuth, error) {
	model, err := selector.SelectModel(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to select AI agent model: %w", err)
	}

	env, err := gateway.AgentEnv(model)
	if err != nil {
		return nil, fmt.Errorf("AI gateway rejected AI agent model %q: %w", model, err)
	}

	apiKey := types.NewMaskSecret(env[agentEnvToken])

	return map[enum.AIAgent]types.AIAgentAuth{
		enum.AIAgentClaudeCode: {
			AuthType: enum.OpenRouterAPIKeyAuth,
			APIKey:   types.APIKey{Value: &apiKey},
			BaseURL:  env[agentEnvBaseURL],
			Model:    model,
		},
	}, nil
}

func (o Orchestrator) aiGatewayEnabled() bool {
	return o.aiGateway != nil && o.aiGateway.Enabled()
}
