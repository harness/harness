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

	"github.com/harness/gitness/app/gitspace/logutil"
	"github.com/harness/gitness/app/gitspace/orchestrator/common"
	"github.com/harness/gitness/app/gitspace/orchestrator/devcontainer"
	"github.com/harness/gitness/app/gitspace/orchestrator/ide"
	"github.com/harness/gitness/app/gitspace/scm"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

// setupGitspaceAndIDE initializes Gitspace and IDE by registering and executing the setup steps.
func (e *EmbeddedDockerOrchestrator) setupGitspaceAndIDE(
	ctx context.Context,
	exec *devcontainer.Exec,
	logStreamInstance *logutil.LogStreamInstance,
	ideService ide.IDE,
	gitspaceConfig types.GitspaceConfig,
	resolvedRepoDetails scm.ResolvedDetails,
	defaultBaseImage string,
) error {
	homeDir := GetUserHomeDir(gitspaceConfig.GitspaceUser.Identifier)
	devcontainerConfig := resolvedRepoDetails.DevcontainerConfig
	codeRepoDir := filepath.Join(homeDir, resolvedRepoDetails.RepoName)

	// Register setup steps
	e.RegisterStep("Validate Supported OS", e.validateSupportedOS, true)
	e.RegisterStep("Manage User", e.manageUser, true)
	e.RegisterStep("Install Tools",
		func(ctx context.Context, exec *devcontainer.Exec, logStreamInstance *logutil.LogStreamInstance) error {
			return e.installTools(ctx, exec, logStreamInstance, gitspaceConfig.IDE)
		}, true)
	e.RegisterStep("Setup IDE",
		func(ctx context.Context, exec *devcontainer.Exec, logStreamInstance *logutil.LogStreamInstance) error {
			return e.setupIDE(ctx, exec, ideService, logStreamInstance)
		}, true)
	e.RegisterStep("Run IDE",
		func(ctx context.Context, exec *devcontainer.Exec, logStreamInstance *logutil.LogStreamInstance) error {
			return e.runIDE(ctx, exec, ideService, logStreamInstance)
		}, true)
	e.RegisterStep("Install Git", e.installGit, true)
	e.RegisterStep("Setup Git Credentials",
		func(ctx context.Context, exec *devcontainer.Exec, logStreamInstance *logutil.LogStreamInstance) error {
			if resolvedRepoDetails.ResolvedCredentials.Credentials != nil {
				return e.setupGitCredentials(ctx, exec, resolvedRepoDetails, logStreamInstance)
			}
			return nil
		}, true)
	e.RegisterStep("Clone Code",
		func(ctx context.Context, exec *devcontainer.Exec, logStreamInstance *logutil.LogStreamInstance) error {
			return e.cloneCode(ctx, exec, defaultBaseImage, resolvedRepoDetails, logStreamInstance)
		}, true)

	// Register the Execute Command steps (PostCreate and PostStart)
	e.RegisterStep("Execute PostCreate Command",
		func(ctx context.Context, exec *devcontainer.Exec, logStreamInstance *logutil.LogStreamInstance) error {
			command := ExtractCommand(PostCreateAction, devcontainerConfig)
			return e.executeCommand(ctx, exec, codeRepoDir, logStreamInstance, command, PostCreateAction)
		}, false)
	e.RegisterStep("Execute PostStart Command",
		func(ctx context.Context, exec *devcontainer.Exec, logStreamInstance *logutil.LogStreamInstance) error {
			command := ExtractCommand(PostStartAction, devcontainerConfig)
			return e.executeCommand(ctx, exec, codeRepoDir, logStreamInstance, command, PostStartAction)
		}, false)

	// Execute the registered steps
	if err := e.ExecuteSteps(ctx, exec, logStreamInstance); err != nil {
		return err
	}
	return nil
}

func (e *EmbeddedDockerOrchestrator) installTools(
	ctx context.Context,
	exec *devcontainer.Exec,
	logStreamInstance *logutil.LogStreamInstance,
	ideType enum.IDEType,
) error {
	output, err := common.InstallTools(ctx, exec, ideType)
	if err != nil {
		return logStreamWrapError(logStreamInstance, "Error while installing tools inside container", err)
	}

	if err := logStreamSuccess(logStreamInstance, "Tools installation output...\n"+string(output)); err != nil {
		return err
	}

	return nil
}

func (e *EmbeddedDockerOrchestrator) validateSupportedOS(
	ctx context.Context,
	exec *devcontainer.Exec,
	logStreamInstance *logutil.LogStreamInstance,
) error {
	output, err := common.ValidateSupportedOS(ctx, exec)
	if err != nil {
		return logStreamWrapError(logStreamInstance, "Error while detecting OS inside container", err)
	}

	if err := logStreamSuccess(logStreamInstance, "Validate supported OSes...\n"+string(output)); err != nil {
		return err
	}

	return nil
}

func (e *EmbeddedDockerOrchestrator) executeCommand(
	ctx context.Context,
	exec *devcontainer.Exec,
	codeRepoDir string,
	logStreamInstance *logutil.LogStreamInstance,
	command string,
	actionType PostAction,
) error {
	if command == "" {
		if err := logStreamSuccess(
			logStreamInstance,
			fmt.Sprintf("No %s command provided, skipping execution", actionType)); err != nil {
			return err
		}
		return nil
	}

	if err := logStreamSuccess(
		logStreamInstance, fmt.Sprintf("Executing %s command: %s", actionType, command)); err != nil {
		return err
	}

	output, err := exec.ExecuteCommand(ctx, command, true, false, codeRepoDir)
	if err != nil {
		return logStreamWrapError(
			logStreamInstance, fmt.Sprintf("Error while executing %s command", actionType), err)
	}

	if err := logStreamSuccess(
		logStreamInstance, "Post create command execution output...\n"+string(output)); err != nil {
		return err
	}

	return logStreamSuccess(
		logStreamInstance, "Successfully executed postCreate command")
}

func (e *EmbeddedDockerOrchestrator) cloneCode(
	ctx context.Context,
	exec *devcontainer.Exec,
	defaultBaseImage string,
	resolvedRepoDetails scm.ResolvedDetails,
	logStreamInstance *logutil.LogStreamInstance,
) error {
	output, err := e.gitService.CloneCode(ctx, exec, resolvedRepoDetails, defaultBaseImage)
	if err != nil {
		return logStreamWrapError(logStreamInstance, "Error while cloning code inside container", err)
	}

	if err := logStreamSuccess(logStreamInstance, "Clone output...\n"+string(output)); err != nil {
		return err
	}

	return nil
}

func (e *EmbeddedDockerOrchestrator) installGit(
	ctx context.Context,
	exec *devcontainer.Exec,
	logStreamInstance *logutil.LogStreamInstance,
) error {
	output, err := e.gitService.Install(ctx, exec)
	if err != nil {
		return logStreamWrapError(logStreamInstance, "Error while installing git inside container", err)
	}

	if err := logStreamSuccess(logStreamInstance, "Install git output...\n"+string(output)); err != nil {
		return err
	}

	return nil
}

func (e *EmbeddedDockerOrchestrator) setupGitCredentials(
	ctx context.Context,
	exec *devcontainer.Exec,
	resolvedRepoDetails scm.ResolvedDetails,
	logStreamInstance *logutil.LogStreamInstance,
) error {
	output, err := e.gitService.SetupCredentials(ctx, exec, resolvedRepoDetails)
	if err != nil {
		return logStreamWrapError(
			logStreamInstance, "Error while setting up git credentials inside container", err)
	}

	if err := logStreamSuccess(logStreamInstance, "Setting up git credentials output...\n"+string(output)); err != nil {
		return err
	}

	return nil
}

func (e *EmbeddedDockerOrchestrator) manageUser(
	ctx context.Context,
	exec *devcontainer.Exec,
	logStreamInstance *logutil.LogStreamInstance,
) error {
	output, err := e.userService.Manage(ctx, exec)
	if err != nil {
		return logStreamWrapError(logStreamInstance, "Error while creating user inside container", err)
	}

	if err := logStreamSuccess(logStreamInstance, "Managing user output...\n"+string(output)); err != nil {
		return err
	}

	return nil
}

func (e *EmbeddedDockerOrchestrator) setupIDE(
	ctx context.Context,
	exec *devcontainer.Exec,
	ideService ide.IDE,
	logStreamInstance *logutil.LogStreamInstance,
) error {
	if err := logStreamSuccess(
		logStreamInstance, "Setting up IDE inside container: "+string(ideService.Type())); err != nil {
		return err
	}

	output, err := ideService.Setup(ctx, exec)
	if err != nil {
		return logStreamWrapError(logStreamInstance, "Error while setting up IDE inside container", err)
	}

	if err := logStreamSuccess(logStreamInstance, "IDE setup output...\n"+string(output)); err != nil {
		return err
	}

	return logStreamSuccess(logStreamInstance, "Successfully set up IDE inside container")
}

func (e *EmbeddedDockerOrchestrator) runIDE(
	ctx context.Context,
	exec *devcontainer.Exec,
	ideService ide.IDE,
	logStreamInstance *logutil.LogStreamInstance,
) error {
	if err := logStreamSuccess(
		logStreamInstance, "Running the IDE inside container: "+string(ideService.Type())); err != nil {
		return err
	}

	output, err := ideService.Run(ctx, exec)
	if err != nil {
		return logStreamWrapError(logStreamInstance, "Error while running IDE inside container", err)
	}

	if err := logStreamSuccess(logStreamInstance, "IDE run output...\n"+string(output)); err != nil {
		return err
	}

	return logStreamSuccess(logStreamInstance, "Successfully run the IDE inside container")
}
