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

package utils

import (
	"strings"
	"testing"

	"github.com/harness/gitness/app/gitspace/types"
	gitnessTypes "github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

func authWithKey(authType enum.AIAgentAuth, model string) gitnessTypes.AIAgentAuth {
	key := gitnessTypes.NewMaskSecret("key-123")
	return gitnessTypes.AIAgentAuth{
		AuthType: authType,
		APIKey:   gitnessTypes.APIKey{Value: &key},
		Model:    model,
	}
}

func TestClaudeCodePayloadOpenRouterRequiresFreeModel(t *testing.T) {
	if _, err := claudeCodePayload(authWithKey(enum.OpenRouterAPIKeyAuth, "anthropic/claude-sonnet")); err == nil {
		t.Fatal("paid models must be rejected")
	}
	if _, err := claudeCodePayload(authWithKey(enum.OpenRouterAPIKeyAuth, "")); err == nil {
		t.Fatal("a model is required")
	}

	p, err := claudeCodePayload(authWithKey(enum.OpenRouterAPIKeyAuth, "vendor/coder:free"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.BaseURL != "https://openrouter.ai/api" || p.AuthToken != "key-123" || p.AnthropicAPIKey != "" {
		t.Errorf("unexpected payload: %+v", p)
	}
}

func TestClaudeCodeConfigureScript(t *testing.T) {
	script, err := GenerateScriptFromTemplate(templateClaudeCodeConfigureScript, &types.ConfigureClaudeCodePayload{
		BaseURL:   "https://openrouter.ai/api",
		AuthToken: "key-123",
		Model:     "vendor/coder:free",
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`"ANTHROPIC_BASE_URL": "https://openrouter.ai/api"`,
		`"ANTHROPIC_MODEL": "vendor/coder:free"`,
		`echo "$ANTHROPIC_AUTH_TOKEN"`,
	} {
		if !strings.Contains(script, want) {
			t.Errorf("script is missing %q:\n%s", want, script)
		}
	}

	script, err = GenerateScriptFromTemplate(templateClaudeCodeConfigureScript, &types.ConfigureClaudeCodePayload{
		AnthropicAPIKey: "sk-ant",
	})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(script, "ANTHROPIC_BASE_URL") || !strings.Contains(script, `echo "$ANTHROPIC_API_KEY"`) {
		t.Errorf("anthropic mode must keep the original configuration:\n%s", script)
	}
}
