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

	_ "embed"
)

const (
	templateSupportedOSDistribution    = "supported_os_distribution.sh"
	templateVsCodeWebToolsInstallation = "install_tools_vs_code_web.sh"
	templateVsCodeToolsInstallation    = "install_tools_vs_code.sh"
	templateIntellijToolsInstallation  = "install_tools_intellij.sh"
	templateSetEnv                     = "set_env.sh"
	templateGitInstallScript           = "install_git.sh"
	templateSetupGitCredentials        = "setup_git_credentials.sh" // nolint:gosec
	templateCloneCode                  = "clone_code.sh"
	templateManagerUser                = "manage_user.sh"
)

//go:embed script/os_info.sh
var osDetectScript string

func GetOSInfoScript() (script string) {
	return osDetectScript
}

func ValidateSupportedOS(
	ctx context.Context,
	exec *devcontainer.Exec,
	gitspaceLogger types.GitspaceLogger,
) error {
	// TODO: Currently not supporting arch, freebsd and alpine.
	// For alpine wee need to install multiple things from
	// https://github.com/microsoft/vscode/wiki/How-to-Contribute#prerequisites
	script, err := GenerateScriptFromTemplate(
		templateSupportedOSDistribution, &types.SupportedOSDistributionPayload{
			OSInfoScript: osDetectScript,
		})
	if err != nil {
		return fmt.Errorf("failed to generate scipt to validate supported os distribution from template %s: %w",
			templateSupportedOSDistribution, err)
	}
	gitspaceLogger.Info("Validate supported OSes...")
	err = exec.ExecuteCommandInHomeDirAndLog(ctx, script, true, gitspaceLogger, false)
	if err != nil {
		return fmt.Errorf("error while detecting os distribution: %w", err)
	}
	return nil
}

func SetEnv(
	ctx context.Context,
	exec *devcontainer.Exec,
	gitspaceLogger types.GitspaceLogger,
	environment []string,
) error {
	if len(environment) > 0 {
		script, err := GenerateScriptFromTemplate(
			templateSetEnv, &types.SetEnvPayload{
				EnvVariables: environment,
			})
		if err != nil {
			return fmt.Errorf("failed to generate scipt to set env from template %s: %w",
				templateSetEnv, err)
		}
		gitspaceLogger.Info("Setting env...")
		err = exec.ExecuteCommandInHomeDirAndLog(ctx, script, true, gitspaceLogger, true)
		if err != nil {
			return fmt.Errorf("error while setting env vars: %w", err)
		}
	}
	return nil
}
