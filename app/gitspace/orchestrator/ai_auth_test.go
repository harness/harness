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
	"strings"
	"testing"

	"github.com/harness/gitness/app/services/aigateway"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

type fakeAgentGateway struct {
	enabled bool
}

func (g fakeAgentGateway) Enabled() bool { return g.enabled }

func (g fakeAgentGateway) AgentEnv(model string) (map[string]string, error) {
	if !strings.HasSuffix(model, ":free") {
		return nil, errors.New("model is not free")
	}
	return map[string]string{
		agentEnvBaseURL: "https://gateway.example/api",
		agentEnvToken:   "gateway-key",
	}, nil
}

type fakePlatformSecret struct {
	calls int
}

func (f *fakePlatformSecret) FetchSecret(context.Context, string, string) (*types.MaskSecret, error) {
	f.calls++
	s := types.NewMaskSecret("anthropic-key")
	return &s, nil
}

type fixedSelector struct{ model string }

func (s fixedSelector) SelectModel(context.Context) (string, error) { return s.model, nil }

func selectorWith(models ...aigateway.Model) AgentModelSelector {
	return deterministicModelSelector{
		candidates: func(aigateway.Requirements) []aigateway.Model { return models },
	}
}

func TestDecryptAIAuth_GatewayDisabledUsesAnthropic(t *testing.T) {
	secrets := &fakePlatformSecret{}
	o := Orchestrator{
		platformSecret:     secrets,
		aiGateway:          fakeAgentGateway{enabled: false},
		agentModelSelector: fixedSelector{model: "vendor/m:free"},
	}

	auth, err := o.decryptAIAuth(context.Background(), "space", "ref")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := auth[enum.AIAgentClaudeCode]
	if got.AuthType != enum.AnthropicAPIKeyAuth || got.APIKey.Value.Value() != "anthropic-key" ||
		got.BaseURL != "" || got.Model != "" || secrets.calls != 1 {
		t.Errorf("unexpected auth: %+v", got)
	}
}

func TestDecryptAIAuth_GatewayEnabledUsesOpenRouter(t *testing.T) {
	secrets := &fakePlatformSecret{}
	o := Orchestrator{
		platformSecret:     secrets,
		aiGateway:          fakeAgentGateway{enabled: true},
		agentModelSelector: selectorWith(aigateway.Model{ID: "vendor/coder:free"}, aigateway.Model{ID: "z/other:free"}),
	}

	auth, err := o.decryptAIAuth(context.Background(), "space", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := auth[enum.AIAgentClaudeCode]
	if got.AuthType != enum.OpenRouterAPIKeyAuth || got.Model != "vendor/coder:free" ||
		got.BaseURL != "https://gateway.example/api" || got.APIKey.Value.Value() != "gateway-key" {
		t.Errorf("unexpected auth: %+v", got)
	}
	if secrets.calls != 0 {
		t.Errorf("platform secret must not be used when the gateway is enabled")
	}
	if strings.Contains(fmt.Sprintf("%v %+v", got, got), "gateway-key") {
		t.Errorf("gateway key leaked in formatted output")
	}
}

func TestDecryptAIAuth_GatewayEnabledNoCandidates(t *testing.T) {
	secrets := &fakePlatformSecret{}
	o := Orchestrator{
		platformSecret:     secrets,
		aiGateway:          fakeAgentGateway{enabled: true},
		agentModelSelector: selectorWith(),
	}

	_, err := o.decryptAIAuth(context.Background(), "space", "ref")
	if !errors.Is(err, ErrNoAgentModel) {
		t.Fatalf("expected ErrNoAgentModel, got %v", err)
	}
	if secrets.calls != 0 {
		t.Errorf("must not fall back to Anthropic when the gateway is enabled")
	}
}

func TestDecryptAIAuth_GatewayRejectsPaidModel(t *testing.T) {
	o := Orchestrator{
		aiGateway:          fakeAgentGateway{enabled: true},
		agentModelSelector: fixedSelector{model: "vendor/paid"},
	}
	if _, err := o.decryptAIAuth(context.Background(), "space", ""); err == nil {
		t.Fatal("expected error for non-free model")
	}
}
