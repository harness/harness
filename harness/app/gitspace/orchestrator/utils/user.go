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
)

func ManageUser(
	ctx context.Context,
	exec *devcontainer.Exec,
	gitspaceLogger types.GitspaceLogger,
) error {
	script, err := GenerateScriptFromTemplate(
		templateManagerUser, &types.SetupUserPayload{
			Username:   exec.RemoteUser,
			AccessKey:  exec.AccessKey,
			AccessType: exec.AccessType,
			HomeDir:    exec.DefaultWorkingDir,
		})
	if err != nil {
		return fmt.Errorf(
			"failed to generate scipt to manager user from template %s: %w", templateManagerUser, err)
	}

	gitspaceLogger.Info("Configuring user directory and credentials inside container")
	err = exec.ExecuteCommandInHomeDirAndLog(ctx, script, true, gitspaceLogger, true)
	if err != nil {
		return fmt.Errorf("failed to setup user: %w", err)
	}

	gitspaceLogger.Info("Successfully configured the user directory and credentials.")

	return nil
}
