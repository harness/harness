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

package common

import (
	"context"
	"fmt"

	"github.com/harness/gitness/app/gitspace/orchestrator/devcontainer"
	"github.com/harness/gitness/app/gitspace/orchestrator/template"
	"github.com/harness/gitness/types/enum"
)

const templateSupportedOSDistribution = "supported_os_distribution.sh"
const templateVsCodeWebToolsInstallation = "install_tools_vs_code_web.sh"
const templateVsCodeToolsInstallation = "install_tools_vs_code.sh"

func ValidateSupportedOS(ctx context.Context, exec *devcontainer.Exec) ([]byte, error) {
	// TODO: Currently not supporting arch, freebsd and alpine.
	// For alpine wee need to install multiple things from
	// https://github.com/microsoft/vscode/wiki/How-to-Contribute#prerequisites
	script, err := template.GenerateScriptFromTemplate(
		templateSupportedOSDistribution, &template.SupportedOSDistributionPayload{
			OSInfoScript: osDetectScript,
		})
	if err != nil {
		return nil, fmt.Errorf("failed to generate scipt to validate supported os distribution from template %s: %w",
			templateSupportedOSDistribution, err)
	}

	output, err := exec.ExecuteCommandInHomeDirectory(ctx, script, true, false)
	if err != nil {
		return nil, fmt.Errorf("error while detecting os distribution: %w", err)
	}
	return output, nil
}

func InstallTools(ctx context.Context, exec *devcontainer.Exec, ideType enum.IDEType) ([]byte, error) {
	switch ideType {
	case enum.IDETypeVSCodeWeb:
		{
			output, err := InstallToolsForVsCodeWeb(ctx, exec)
			if err != nil {
				return []byte(output), err
			}
			return []byte(output), nil
		}
	case enum.IDETypeVSCode:
		{
			output, err := InstallToolsForVsCode(ctx, exec)
			if err != nil {
				return []byte(output), err
			}
			return []byte(output), nil
		}
	}
	return nil, nil
}

func InstallToolsForVsCodeWeb(ctx context.Context, exec *devcontainer.Exec) (string, error) {
	script, err := template.GenerateScriptFromTemplate(
		templateVsCodeWebToolsInstallation, &template.InstallToolsPayload{
			OSInfoScript: osDetectScript,
		})
	if err != nil {
		return "", fmt.Errorf(
			"failed to generate scipt to install tools for vs code web from template %s: %w",
			templateVsCodeWebToolsInstallation, err)
	}

	output := "Installing tools for vs code web inside container\n"
	_, err = exec.ExecuteCommandInHomeDirectory(ctx, script, true, false)
	if err != nil {
		return "", fmt.Errorf("failed to install tools for vs code web: %w", err)
	}

	output += "Successfully installed tools for vs code web\n"

	return output, nil
}

func InstallToolsForVsCode(ctx context.Context, exec *devcontainer.Exec) (string, error) {
	script, err := template.GenerateScriptFromTemplate(
		templateVsCodeToolsInstallation, &template.InstallToolsPayload{
			OSInfoScript: osDetectScript,
		})
	if err != nil {
		return "", fmt.Errorf(
			"failed to generate scipt to install tools for vs code from template %s: %w",
			templateVsCodeToolsInstallation, err)
	}

	output := "Installing tools for vs code inside container\n"
	_, err = exec.ExecuteCommandInHomeDirectory(ctx, script, true, false)
	if err != nil {
		return "", fmt.Errorf("failed to install tools for vs code: %w", err)
	}

	output += "Successfully installed tools for vs code\n"

	return output, nil
}
