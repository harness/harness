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
	"sync"

	"github.com/harness/gitness/app/gitspace/orchestrator/devcontainer"
	"github.com/harness/gitness/app/gitspace/orchestrator/ide"
	"github.com/harness/gitness/app/gitspace/orchestrator/utils"
	"github.com/harness/gitness/app/gitspace/scm"
	gitspaceTypes "github.com/harness/gitness/app/gitspace/types"
	"github.com/harness/gitness/types/enum"
)

func InstallTools(
	ctx context.Context,
	exec *devcontainer.Exec,
	gitspaceLogger gitspaceTypes.GitspaceLogger,
	ideType enum.IDEType,
) error {
	err := utils.InstallTools(ctx, exec, ideType, gitspaceLogger)
	if err != nil {
		return logStreamWrapError(gitspaceLogger, "Error while installing tools inside container", err)
	}
	return nil
}

func ValidateSupportedOS(
	ctx context.Context,
	exec *devcontainer.Exec,
	gitspaceLogger gitspaceTypes.GitspaceLogger,
) error {
	err := utils.ValidateSupportedOS(ctx, exec, gitspaceLogger)
	if err != nil {
		return logStreamWrapError(gitspaceLogger, "Error while detecting OS inside container", err)
	}
	return nil
}

// ExecuteLifecycleCommands executes commands in parallel, logs with numbers, and prefixes all logs.
func ExecuteLifecycleCommands(
	ctx context.Context,
	exec devcontainer.Exec,
	codeRepoDir string,
	gitspaceLogger gitspaceTypes.GitspaceLogger,
	commands []string,
	actionType PostAction,
) error {
	if len(commands) == 0 {
		gitspaceLogger.Info(fmt.Sprintf("No %s commands provided, skipping execution", actionType))
		return nil
	}
	gitspaceLogger.Info(fmt.Sprintf("Executing %s commands: %v", actionType, commands))

	// Create a WaitGroup to wait for all goroutines to finish.
	var wg sync.WaitGroup

	// Iterate over commands and execute them in parallel using goroutines.
	for index, command := range commands {
		// Increment the WaitGroup counter.
		wg.Add(1)

		// Execute each command in a new goroutine.
		go func(index int, command string) {
			// Decrement the WaitGroup counter when the goroutine finishes.
			defer wg.Done()

			// Number the command in the logs and prefix all logs.
			commandNumber := index + 1 // Starting from 1 for numbering
			logPrefix := fmt.Sprintf("Command #%d - ", commandNumber)

			// Log command execution details.
			gitspaceLogger.Info(fmt.Sprintf("%sExecuting %s command: %s", logPrefix, actionType, command))
			exec.DefaultWorkingDir = codeRepoDir
			err := exec.ExecuteCommandInHomeDirAndLog(ctx, command, false, gitspaceLogger, true)
			if err != nil {
				// Log the error if there is any issue with executing the command.
				_ = logStreamWrapError(gitspaceLogger, fmt.Sprintf("%sError while executing %s command: %s",
					logPrefix, actionType, command), err)
				return
			}

			// Log completion of the command execution.
			gitspaceLogger.Info(fmt.Sprintf(
				"%sCompleted execution %s command: %s", logPrefix, actionType, command))
		}(index, command)
	}

	// Wait for all goroutines to finish.
	wg.Wait()

	return nil
}

func CloneCode(
	ctx context.Context,
	exec *devcontainer.Exec,
	defaultBaseImage string,
	resolvedRepoDetails scm.ResolvedDetails,
	gitspaceLogger gitspaceTypes.GitspaceLogger,
) error {
	err := utils.CloneCode(ctx, exec, resolvedRepoDetails, defaultBaseImage, gitspaceLogger)
	if err != nil {
		return logStreamWrapError(gitspaceLogger, "Error while cloning code inside container", err)
	}
	return nil
}

func InstallGit(
	ctx context.Context,
	exec *devcontainer.Exec,
	gitspaceLogger gitspaceTypes.GitspaceLogger,
) error {
	err := utils.InstallGit(ctx, exec, gitspaceLogger)
	if err != nil {
		return logStreamWrapError(gitspaceLogger, "Error while installing git inside container", err)
	}
	return nil
}

func SetupGitCredentials(
	ctx context.Context,
	exec *devcontainer.Exec,
	resolvedRepoDetails scm.ResolvedDetails,
	gitspaceLogger gitspaceTypes.GitspaceLogger,
) error {
	err := utils.SetupGitCredentials(ctx, exec, resolvedRepoDetails, gitspaceLogger)
	if err != nil {
		return logStreamWrapError(
			gitspaceLogger, "Error while setting up git credentials inside container", err)
	}
	return nil
}

func ManageUser(
	ctx context.Context,
	exec *devcontainer.Exec,
	gitspaceLogger gitspaceTypes.GitspaceLogger,
) error {
	err := utils.ManageUser(ctx, exec, gitspaceLogger)
	if err != nil {
		return logStreamWrapError(gitspaceLogger, "Error while creating user inside container", err)
	}
	return nil
}

func SetupIDE(
	ctx context.Context,
	exec *devcontainer.Exec,
	ideService ide.IDE,
	args map[gitspaceTypes.IDEArg]interface{},
	gitspaceLogger gitspaceTypes.GitspaceLogger,
) error {
	gitspaceLogger.Info("Setting up IDE inside container: " + string(ideService.Type()))
	err := ideService.Setup(ctx, exec, args, gitspaceLogger)
	if err != nil {
		return logStreamWrapError(gitspaceLogger, "Error while setting up IDE inside container", err)
	}
	return nil
}

func RunIDEWithArgs(
	ctx context.Context,
	exec *devcontainer.Exec,
	ideService ide.IDE,
	args map[gitspaceTypes.IDEArg]interface{},
	gitspaceLogger gitspaceTypes.GitspaceLogger,
) error {
	gitspaceLogger.Info("Running the IDE inside container: " + string(ideService.Type()))
	err := ideService.Run(ctx, exec, args, gitspaceLogger)
	if err != nil {
		return logStreamWrapError(gitspaceLogger, "Error while running IDE inside container", err)
	}
	return nil
}

func SetEnv(
	ctx context.Context,
	exec *devcontainer.Exec,
	gitspaceLogger gitspaceTypes.GitspaceLogger,
	environment []string,
) error {
	if len(environment) > 0 {
		err := utils.SetEnv(ctx, exec, gitspaceLogger, environment)
		if err != nil {
			return logStreamWrapError(gitspaceLogger, "Error while installing tools inside container", err)
		}
	}
	return nil
}
