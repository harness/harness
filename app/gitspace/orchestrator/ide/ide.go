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

package ide

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/harness/gitness/app/gitspace/orchestrator/devcontainer"
	"github.com/harness/gitness/app/gitspace/orchestrator/utils"
	gitspaceTypes "github.com/harness/gitness/app/gitspace/types"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

const (
	templateSetupSSHServer string = "setup_ssh_server.sh"
	templateRunSSHServer   string = "run_ssh_server.sh"
)

type IDE interface {
	// Setup is responsible for doing all the operations for setting up the IDE in the container e.g. installation,
	// copying settings and configurations.
	Setup(
		ctx context.Context,
		exec *devcontainer.Exec,
		args map[gitspaceTypes.IDEArg]any,
		gitspaceLogger gitspaceTypes.GitspaceLogger,
	) error

	// Run runs the IDE and supporting services.
	Run(
		ctx context.Context,
		exec *devcontainer.Exec,
		args map[gitspaceTypes.IDEArg]any,
		gitspaceLogger gitspaceTypes.GitspaceLogger,
	) error

	// Port provides the port which will be used by this IDE.
	Port() *types.GitspacePort

	// Type provides the IDE type to which the service belongs.
	Type() enum.IDEType

	// GenerateURL returns the url to redirect user to ide from gitspace
	GenerateURL(absoluteRepoPath, host, port, user string) string

	// GenerateURL returns the url to redirect user to ide from gitspace
	GeneratePluginURL(projectName, gitspaceInstaceUID string) string
}

func getHomePath(absoluteRepoPath string) string {
	pathList := strings.Split(absoluteRepoPath, "/")
	return strings.Join(pathList[:len(pathList)-1], "/")
}

// setupSSHServer is responsible for setting up the SSH server inside the container.
// This is a common operation done by most of the IDEs that require SSH to connect.
func setupSSHServer(
	ctx context.Context,
	exec *devcontainer.Exec,
	gitspaceLogger gitspaceTypes.GitspaceLogger,
) error {
	osInfoScript := utils.GetOSInfoScript()
	payload := gitspaceTypes.SetupSSHServerPayload{
		Username:     exec.RemoteUser,
		AccessType:   exec.AccessType,
		OSInfoScript: osInfoScript,
	}
	sshServerScript, err := utils.GenerateScriptFromTemplate(
		templateSetupSSHServer, &payload)
	if err != nil {
		return fmt.Errorf(
			"failed to generate scipt to setup ssh server from template %s: %w", templateSetupSSHServer, err)
	}
	err = exec.ExecuteCommandInHomeDirAndLog(ctx, sshServerScript, true, gitspaceLogger, false)
	if err != nil {
		return fmt.Errorf("failed to setup SSH serverr: %w", err)
	}

	return nil
}

// runSSHServer runs the SSH server inside the container.
// This is a common operation done by most of the IDEs that require ssh connection.
func runSSHServer(
	ctx context.Context,
	exec *devcontainer.Exec,
	port int,
	gitspaceLogger gitspaceTypes.GitspaceLogger,
) error {
	payload := gitspaceTypes.RunSSHServerPayload{
		Port: strconv.Itoa(port),
	}
	runSSHScript, err := utils.GenerateScriptFromTemplate(
		templateRunSSHServer, &payload)
	if err != nil {
		return fmt.Errorf(
			"failed to generate scipt to run ssh server from template %s: %w", templateRunSSHServer, err)
	}

	err = exec.ExecuteCommandInHomeDirAndLog(ctx, runSSHScript, true, gitspaceLogger, true)
	if err != nil {
		return fmt.Errorf("failed to run SSH server: %w", err)
	}
	return nil
}
