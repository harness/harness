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
	"github.com/harness/gitness/types/enum"
)

const (
	templateClaudeCodeInstallScript = "install_claude_code.sh"
)

type installAgentFun func(context.Context, *devcontainer.Exec, types.GitspaceLogger) error

var installationMap = map[enum.AIAgent]installAgentFun{
	enum.AIAgentClaudeCode: installClaudeCode,
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
