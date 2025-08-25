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

func InstallTools(
	ctx context.Context,
	exec *devcontainer.Exec,
	ideType enum.IDEType,
	gitspaceLogger types.GitspaceLogger,
) error {
	switch ideType {
	case enum.IDETypeVSCodeWeb:
		err := InstallToolsForVsCodeWeb(ctx, exec, gitspaceLogger)
		if err != nil {
			return err
		}
		return nil
	case enum.IDETypeVSCode:
		err := InstallToolsForVsCode(ctx, exec, gitspaceLogger)
		if err != nil {
			return err
		}
		return nil
	case enum.IDETypeCursor:
		err := InstallToolsForCursor(ctx, exec, gitspaceLogger)
		if err != nil {
			return err
		}
		return nil
	case enum.IDETypeWindsurf:
		err := InstallToolsForWindsurf(ctx, exec, gitspaceLogger)
		if err != nil {
			return err
		}
		return nil
	case enum.IDETypeIntelliJ, enum.IDETypePyCharm, enum.IDETypeGoland, enum.IDETypeWebStorm, enum.IDETypeCLion,
		enum.IDETypePHPStorm, enum.IDETypeRubyMine, enum.IDETypeRider:
		err := InstallToolsForJetBrains(ctx, exec, ideType, gitspaceLogger)
		if err != nil {
			return err
		}
		return nil
	}
	return nil
}

func InstallToolsForVsCodeWeb(
	ctx context.Context,
	exec *devcontainer.Exec,
	gitspaceLogger types.GitspaceLogger,
) error {
	script, err := GenerateScriptFromTemplate(
		templateVsCodeWebToolsInstallation, &types.InstallToolsPayload{
			OSInfoScript: osDetectScript,
		})
	if err != nil {
		return fmt.Errorf(
			"failed to generate scipt to install tools for vs code web from template %s: %w",
			templateVsCodeWebToolsInstallation, err)
	}

	gitspaceLogger.Info("Installing tools for vs code web inside container")
	gitspaceLogger.Info("Tools installation output...")
	err = exec.ExecuteCommandInHomeDirAndLog(ctx, script, true, gitspaceLogger, false)
	if err != nil {
		return fmt.Errorf("failed to install tools for vs code web: %w", err)
	}
	gitspaceLogger.Info("Successfully installed tools for vs code web")
	return nil
}

func InstallToolsForVsCode(
	ctx context.Context,
	exec *devcontainer.Exec,
	gitspaceLogger types.GitspaceLogger,
) error {
	script, err := GenerateScriptFromTemplate(
		templateVsCodeToolsInstallation, &types.InstallToolsPayload{
			OSInfoScript: osDetectScript,
		})
	if err != nil {
		return fmt.Errorf(
			"failed to generate scipt to install tools for vs code from template %s: %w",
			templateVsCodeToolsInstallation, err)
	}

	gitspaceLogger.Info("Installing tools for vs code in container")
	err = exec.ExecuteCommandInHomeDirAndLog(ctx, script, true, gitspaceLogger, false)
	if err != nil {
		return fmt.Errorf("failed to install tools for vs code: %w", err)
	}
	gitspaceLogger.Info("Successfully installed tools for vs code")
	return nil
}

func InstallToolsForJetBrains(
	ctx context.Context,
	exec *devcontainer.Exec,
	ideType enum.IDEType,
	gitspaceLogger types.GitspaceLogger,
) error {
	script, err := GenerateScriptFromTemplate(
		templateIntellijToolsInstallation, &types.InstallToolsPayload{
			OSInfoScript: osDetectScript,
		})
	if err != nil {
		return fmt.Errorf(
			"failed to generate scipt to install tools for %s from template %s: %w",
			ideType, templateIntellijToolsInstallation, err)
	}

	gitspaceLogger.Info(fmt.Sprintf("Installing tools for %s in container", ideType))
	err = exec.ExecuteCommandInHomeDirAndLog(ctx, script, true, gitspaceLogger, false)
	if err != nil {
		return fmt.Errorf("failed to install tools for %s: %w", ideType, err)
	}
	gitspaceLogger.Info(fmt.Sprintf("Successfully installed tools for %s in container", ideType))
	return nil
}

func InstallToolsForWindsurf(
	ctx context.Context,
	exec *devcontainer.Exec,
	gitspaceLogger types.GitspaceLogger,
) error {
	script, err := GenerateScriptFromTemplate(
		templateWindsurfToolsInstallation, &types.InstallToolsPayload{
			OSInfoScript: osDetectScript,
		})
	if err != nil {
		return fmt.Errorf(
			"failed to generate scipt to install tools for windsurf from template %s: %w",
			templateWindsurfToolsInstallation, err)
	}

	gitspaceLogger.Info("Installing tools for windsurf in container")
	err = exec.ExecuteCommandInHomeDirAndLog(ctx, script, true, gitspaceLogger, false)
	if err != nil {
		return fmt.Errorf("failed to install tools for windsurf: %w", err)
	}
	gitspaceLogger.Info("Successfully installed tools for windsurf")
	return nil
}

func InstallToolsForCursor(
	ctx context.Context,
	exec *devcontainer.Exec,
	gitspaceLogger types.GitspaceLogger,
) error {
	script, err := GenerateScriptFromTemplate(
		templateCursorToolsInstallation, &types.InstallToolsPayload{
			OSInfoScript: osDetectScript,
		})
	if err != nil {
		return fmt.Errorf(
			"failed to generate scipt to install tools for cursor from template %s: %w",
			templateCursorToolsInstallation, err)
	}

	gitspaceLogger.Info("Installing tools for cursor in container")
	err = exec.ExecuteCommandInHomeDirAndLog(ctx, script, true, gitspaceLogger, false)
	if err != nil {
		return fmt.Errorf("failed to install tools for cursor: %w", err)
	}
	gitspaceLogger.Info("Successfully installed tools for cursor")
	return nil
}
