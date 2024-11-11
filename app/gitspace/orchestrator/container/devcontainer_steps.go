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

package container

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/harness/gitness/app/gitspace/orchestrator/common"
	"github.com/harness/gitness/app/gitspace/orchestrator/devcontainer"
	"github.com/harness/gitness/app/gitspace/orchestrator/git"
	"github.com/harness/gitness/app/gitspace/orchestrator/ide"
	orchestratorTypes "github.com/harness/gitness/app/gitspace/orchestrator/types"
	"github.com/harness/gitness/app/gitspace/orchestrator/user"
	"github.com/harness/gitness/app/gitspace/scm"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

// setupGitspaceAndIDE initializes Gitspace and IDE by registering and executing the setup steps.
func (e *EmbeddedDockerOrchestrator) setupGitspaceAndIDE(
	ctx context.Context,
	exec *devcontainer.Exec,
	gitspaceLogger orchestratorTypes.GitspaceLogger,
	ideService ide.IDE,
	gitspaceConfig types.GitspaceConfig,
	resolvedRepoDetails scm.ResolvedDetails,
	defaultBaseImage string,
) error {
	homeDir := GetUserHomeDir(gitspaceConfig.GitspaceUser.Identifier)
	devcontainerConfig := resolvedRepoDetails.DevcontainerConfig
	codeRepoDir := filepath.Join(homeDir, resolvedRepoDetails.RepoName)

	// Register setup steps
	e.RegisterStep("Validate Supported OS", ValidateSupportedOS, true)
	e.RegisterStep("Manage User",
		func(ctx context.Context, exec *devcontainer.Exec, gitspaceLogger orchestratorTypes.GitspaceLogger) error {
			return ManageUser(ctx, exec, e.userService, gitspaceLogger)
		}, true)
	e.RegisterStep("Install Tools",
		func(ctx context.Context, exec *devcontainer.Exec, gitspaceLogger orchestratorTypes.GitspaceLogger) error {
			return InstallTools(ctx, exec, gitspaceLogger, gitspaceConfig.IDE)
		}, true)
	e.RegisterStep("Setup IDE",
		func(ctx context.Context, exec *devcontainer.Exec, gitspaceLogger orchestratorTypes.GitspaceLogger) error {
			return SetupIDE(ctx, exec, ideService, gitspaceLogger)
		}, true)
	e.RegisterStep("Run IDE",
		func(ctx context.Context, exec *devcontainer.Exec, gitspaceLogger orchestratorTypes.GitspaceLogger) error {
			return RunIDE(ctx, exec, ideService, gitspaceLogger)
		}, true)
	e.RegisterStep("Install Git",
		func(ctx context.Context, exec *devcontainer.Exec, gitspaceLogger orchestratorTypes.GitspaceLogger) error {
			return InstallGit(ctx, exec, e.gitService, gitspaceLogger)
		}, true)
	e.RegisterStep("Setup Git Credentials",
		func(ctx context.Context, exec *devcontainer.Exec, gitspaceLogger orchestratorTypes.GitspaceLogger) error {
			if resolvedRepoDetails.ResolvedCredentials.Credentials != nil {
				return SetupGitCredentials(ctx, exec, resolvedRepoDetails, e.gitService, gitspaceLogger)
			}
			return nil
		}, true)
	e.RegisterStep("Clone Code",
		func(ctx context.Context, exec *devcontainer.Exec, gitspaceLogger orchestratorTypes.GitspaceLogger) error {
			return CloneCode(ctx, exec, defaultBaseImage, resolvedRepoDetails, e.gitService, gitspaceLogger)
		}, true)

	// Register the Execute Command steps (PostCreate and PostStart)
	e.RegisterStep("Execute PostCreate Command",
		func(ctx context.Context, exec *devcontainer.Exec, gitspaceLogger orchestratorTypes.GitspaceLogger) error {
			command := ExtractCommand(PostCreateAction, devcontainerConfig)
			return ExecuteCommand(ctx, exec, codeRepoDir, gitspaceLogger, command, PostCreateAction)
		}, false)
	e.RegisterStep("Execute PostStart Command",
		func(ctx context.Context, exec *devcontainer.Exec, gitspaceLogger orchestratorTypes.GitspaceLogger) error {
			command := ExtractCommand(PostStartAction, devcontainerConfig)
			return ExecuteCommand(ctx, exec, codeRepoDir, gitspaceLogger, command, PostStartAction)
		}, false)

	// Execute the registered steps
	if err := e.ExecuteSteps(ctx, exec, gitspaceLogger); err != nil {
		return err
	}
	return nil
}

func InstallTools(
	ctx context.Context,
	exec *devcontainer.Exec,
	gitspaceLogger orchestratorTypes.GitspaceLogger,
	ideType enum.IDEType,
) error {
	output, err := common.InstallTools(ctx, exec, ideType)
	if err != nil {
		return logStreamWrapError(gitspaceLogger, "Error while installing tools inside container", err)
	}
	gitspaceLogger.Info("Tools installation output...\n" + string(output))
	return nil
}

func ValidateSupportedOS(
	ctx context.Context,
	exec *devcontainer.Exec,
	gitspaceLogger orchestratorTypes.GitspaceLogger,
) error {
	output, err := common.ValidateSupportedOS(ctx, exec)
	if err != nil {
		return logStreamWrapError(gitspaceLogger, "Error while detecting OS inside container", err)
	}
	gitspaceLogger.Info("Validate supported OSes...\n" + string(output))
	return nil
}

func ExecuteCommand(
	ctx context.Context,
	exec *devcontainer.Exec,
	codeRepoDir string,
	gitspaceLogger orchestratorTypes.GitspaceLogger,
	command string,
	actionType PostAction,
) error {
	if command == "" {
		gitspaceLogger.Info(fmt.Sprintf("No %s command provided, skipping execution", actionType))
	}
	gitspaceLogger.Info(fmt.Sprintf("Executing %s command: %s", actionType, command))
	output, err := exec.ExecuteCommand(ctx, command, true, false, codeRepoDir)
	if err != nil {
		return logStreamWrapError(
			gitspaceLogger, fmt.Sprintf("Error while executing %s command", actionType), err)
	}
	gitspaceLogger.Info("Post create command execution output...\n" + string(output))
	gitspaceLogger.Info(fmt.Sprintf("Successfully executed %s command", actionType))
	return nil
}

func CloneCode(
	ctx context.Context,
	exec *devcontainer.Exec,
	defaultBaseImage string,
	resolvedRepoDetails scm.ResolvedDetails,
	gitService git.Service,
	gitspaceLogger orchestratorTypes.GitspaceLogger,
) error {
	output, err := gitService.CloneCode(ctx, exec, resolvedRepoDetails, defaultBaseImage)
	if err != nil {
		return logStreamWrapError(gitspaceLogger, "Error while cloning code inside container", err)
	}
	gitspaceLogger.Info("Clone output...\n" + string(output))
	return nil
}

func InstallGit(
	ctx context.Context,
	exec *devcontainer.Exec,
	gitService git.Service,
	gitspaceLogger orchestratorTypes.GitspaceLogger,
) error {
	output, err := gitService.Install(ctx, exec)
	if err != nil {
		return logStreamWrapError(gitspaceLogger, "Error while installing git inside container", err)
	}

	gitspaceLogger.Info("Install git output...\n" + string(output))
	return nil
}

func SetupGitCredentials(
	ctx context.Context,
	exec *devcontainer.Exec,
	resolvedRepoDetails scm.ResolvedDetails,
	gitService git.Service,
	gitspaceLogger orchestratorTypes.GitspaceLogger,
) error {
	output, err := gitService.SetupCredentials(ctx, exec, resolvedRepoDetails)
	if err != nil {
		return logStreamWrapError(
			gitspaceLogger, "Error while setting up git credentials inside container", err)
	}

	gitspaceLogger.Info("Setting up git credentials output...\n" + string(output))
	return nil
}

func ManageUser(
	ctx context.Context,
	exec *devcontainer.Exec,
	userService user.Service,
	gitspaceLogger orchestratorTypes.GitspaceLogger,
) error {
	output, err := userService.Manage(ctx, exec)
	if err != nil {
		return logStreamWrapError(gitspaceLogger, "Error while creating user inside container", err)
	}
	gitspaceLogger.Info("Managing user output...\n" + string(output))
	return nil
}

func SetupIDE(
	ctx context.Context,
	exec *devcontainer.Exec,
	ideService ide.IDE,
	gitspaceLogger orchestratorTypes.GitspaceLogger,
) error {
	gitspaceLogger.Info("Setting up IDE inside container: " + string(ideService.Type()))

	output, err := ideService.Setup(ctx, exec)
	if err != nil {
		return logStreamWrapError(gitspaceLogger, "Error while setting up IDE inside container", err)
	}

	gitspaceLogger.Info("IDE setup output...\n" + string(output))
	gitspaceLogger.Info("Successfully set up IDE inside container")
	return nil
}

func RunIDE(
	ctx context.Context,
	exec *devcontainer.Exec,
	ideService ide.IDE,
	gitspaceLogger orchestratorTypes.GitspaceLogger,
) error {
	gitspaceLogger.Info("Running the IDE inside container: " + string(ideService.Type()))
	output, err := ideService.Run(ctx, exec)
	if err != nil {
		return logStreamWrapError(gitspaceLogger, "Error while running IDE inside container", err)
	}

	gitspaceLogger.Info("IDE run output...\n" + string(output))
	gitspaceLogger.Info("Successfully run the IDE inside container")
	return nil
}
