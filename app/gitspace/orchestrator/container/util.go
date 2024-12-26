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
	"sync"

	"github.com/harness/gitness/app/gitspace/orchestrator/devcontainer"
	gitspaceTypes "github.com/harness/gitness/app/gitspace/types"
	"github.com/harness/gitness/types"

	dockerTypes "github.com/docker/docker/api/types"
)

const (
	linuxHome               = "/home"
	deprecatedRemoteUser    = "harness"
	gitspaceRemoteUserLabel = "gitspace.remote.user"
)

func GetGitspaceContainerName(config types.GitspaceConfig) string {
	return "gitspace-" + config.GitspaceUser.Identifier + "-" + config.Identifier
}

func GetUserHomeDir(userIdentifier string) string {
	if userIdentifier == "root" {
		return "/root"
	}
	return filepath.Join(linuxHome, userIdentifier)
}

func ExtractRemoteUserFromLabels(inspectResp dockerTypes.ContainerJSON) string {
	remoteUser := deprecatedRemoteUser

	if remoteUserValue, ok := inspectResp.Config.Labels[gitspaceRemoteUserLabel]; ok {
		remoteUser = remoteUserValue
	}
	return remoteUser
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
