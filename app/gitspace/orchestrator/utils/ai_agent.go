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
	"context"
	"fmt"

	"github.com/harness/gitness/app/gitspace/orchestrator/devcontainer"
	"github.com/harness/gitness/app/gitspace/types"
	"github.com/harness/gitness/app/services/aigateway"
	gitnessTypes "github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

const (
	templateClaudeCodeInstallScript   = "install_claude_code.sh"
	templateClaudeCodeConfigureScript = "configure_claude_code.sh"
)

type (
	installAgentFun func(context.Context, *devcontainer.Exec, types.GitspaceLogger) error

	configureAgentFun func(
		ctx context.Context,
		exec *devcontainer.Exec,
		aiAgentAuth map[enum.AIAgent]gitnessTypes.AIAgentAuth,
		gitspaceLogger types.GitspaceLogger) error
)

var installationMap = map[enum.AIAgent]installAgentFun{
	enum.AIAgentClaudeCode: installClaudeCode,
}

var configurationMap = map[enum.AIAgent]configureAgentFun{
	enum.AIAgentClaudeCode: configureClaudeCode,
}

func InstallAIAgents(
	ctx context.Context,
	exec *devcontainer.Exec,
	gitspaceLogger types.GitspaceLogger,
	aiAgents []enum.AIAgent,
) error {
	gitspaceLogger.Info(fmt.Sprintf("Installing ai agents: %v...", aiAgents))
	for _, aiAgent := range aiAgents {
		installFun, ok := installationMap[aiAgent]
		if !ok {
			return fmt.Errorf("installation not available for %s", aiAgent)
		}

		if err := installFun(ctx, exec, gitspaceLogger); err != nil {
			return err
		}
	}

	gitspaceLogger.Info("Successfully installed ai agents")
	return nil
}

func ConfigureAIAgent(ctx context.Context,
	exec *devcontainer.Exec,
	gitspaceLogger types.GitspaceLogger,
	aiAgents []enum.AIAgent,
	aiAgentAuth map[enum.AIAgent]gitnessTypes.AIAgentAuth,
) error {
	gitspaceLogger.Info(fmt.Sprintf("configuring ai agents: %v...", aiAgents))
	for _, aiAgent := range aiAgents {
		configureFun, ok := configurationMap[aiAgent]
		if !ok {
			gitspaceLogger.Info(fmt.Sprintf("configuration not available for %s", aiAgent))
			continue
		}

		if err := configureFun(ctx, exec, aiAgentAuth, gitspaceLogger); err != nil {
			return err
		}
	}

	gitspaceLogger.Info("Successfully configured ai agents")
	return nil
}

func installClaudeCode(
	ctx context.Context,
	exec *devcontainer.Exec,
	gitspaceLogger types.GitspaceLogger,
) error {
	script, err := GenerateScriptFromTemplate(
		templateClaudeCodeInstallScript, &types.SetupClaudeCodePayload{
			OSInfoScript: GetOSInfoScript(),
		})
	if err != nil {
		return fmt.Errorf(
			"failed to generate script to install claude code from template %s: %w",
			templateClaudeCodeInstallScript, err)
	}
	gitspaceLogger.Info("Installing claude code output...")
	gitspaceLogger.Info("Installing claude code inside container")
	err = exec.ExecuteCommandInHomeDirAndLog(ctx, script, true, gitspaceLogger, true)
	if err != nil {
		return fmt.Errorf("failed to install claude code : %w", err)
	}
	gitspaceLogger.Info("Successfully Installed claude code")
	return nil
}

func configureClaudeCode(
	ctx context.Context,
	exec *devcontainer.Exec,
	aiAgentAuth map[enum.AIAgent]gitnessTypes.AIAgentAuth,
	gitspaceLogger types.GitspaceLogger,
) error {
	claudeAuth, ok := aiAgentAuth[enum.AIAgentClaudeCode]
	if !ok {
		gitspaceLogger.Info("auth is not available for claude code")
		return fmt.Errorf("auth not available for %s", enum.AIAgentClaudeCode)
	}

	payload, err := claudeCodePayload(claudeAuth)
	if err != nil {
		gitspaceLogger.Info(fmt.Sprintf("%s is not available for claude code: %s", claudeAuth.AuthType, err))
		return fmt.Errorf("auth type not available for %s: %w", enum.AIAgentClaudeCode, err)
	}

	script, err := GenerateScriptFromTemplate(templateClaudeCodeConfigureScript, payload)
	if err != nil {
		return fmt.Errorf(
			"failed to generate script to configure claude code from template %s: %w",
			templateClaudeCodeInstallScript, err)
	}
	gitspaceLogger.Info("configuring claude code output...")
	gitspaceLogger.Info(fmt.Sprintf("claude code is setup using %s", claudeAuth.AuthType))
	err = exec.ExecuteCommandInHomeDirAndLog(ctx, script, false, gitspaceLogger, true)
	if err != nil {
		return fmt.Errorf("failed to configure claude code : %w", err)
	}
	gitspaceLogger.Info("Successfully configured claude code")
	return nil
}

func claudeCodePayload(auth gitnessTypes.AIAgentAuth) (*types.ConfigureClaudeCodePayload, error) {
	if auth.APIKey.Value == nil {
		return nil, fmt.Errorf("api key is missing")
	}

	switch auth.AuthType {
	case enum.AnthropicAPIKeyAuth:
		return &types.ConfigureClaudeCodePayload{
			AnthropicAPIKey: auth.APIKey.Value.Value(),
		}, nil
	case enum.OpenRouterAPIKeyAuth:
		// only free models may run through OpenRouter
		if !aigateway.IsFreeModelID(auth.Model) {
			return nil, fmt.Errorf("model %q is not a free OpenRouter model", auth.Model)
		}
		baseURL := auth.BaseURL
		if baseURL == "" {
			baseURL = aigateway.DefaultBaseURL
		}
		return &types.ConfigureClaudeCodePayload{
			BaseURL:   baseURL,
			AuthToken: auth.APIKey.Value.Value(),
			Model:     auth.Model,
		}, nil
	default:
		return nil, fmt.Errorf("unsupported auth type %q", auth.AuthType)
	}
}
