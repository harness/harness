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

	"github.com/harness/gitness/app/gitspace/orchestrator/devcontainer"
	"github.com/harness/gitness/app/gitspace/orchestrator/template"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

var _ IDE = (*VSCode)(nil)

const templateSetupSSHServer string = "setup_ssh_server.sh"
const templateRunSSHServer string = "run_ssh_server.sh"

type VSCodeConfig struct {
	Port int
}

type VSCode struct {
	config *VSCodeConfig
}

func NewVsCodeService(config *VSCodeConfig) *VSCode {
	return &VSCode{config: config}
}

// Setup installs the SSH server inside the container.
func (v *VSCode) Setup(
	ctx context.Context,
	devcontainer *devcontainer.Exec,
	gitspaceInstance *types.GitspaceInstance,
) ([]byte, error) {
	sshServerScript, err := template.GenerateScriptFromTemplate(
		templateSetupSSHServer, &template.SetupSSHServerPayload{
			Username:         "harness",
			Password:         *gitspaceInstance.AccessKey,
			WorkingDirectory: devcontainer.WorkingDir,
		})
	if err != nil {
		return nil, fmt.Errorf(
			"failed to generate scipt to setup ssh server from template %s: %w", templateSetupSSHServer, err)
	}

	output := "Installing ssh-server inside container\n"

	_, err = devcontainer.ExecuteCommand(ctx, sshServerScript, false, rootUser)
	if err != nil {
		return nil, fmt.Errorf("failed to setup SSH serverr: %w", err)
	}

	output += "Successfully installed ssh-server\n"

	return []byte(output), nil
}

// Run runs the SSH server inside the container.
func (v *VSCode) Run(ctx context.Context, devcontainer *devcontainer.Exec) ([]byte, error) {
	var output = ""

	runSSHScript, err := template.GenerateScriptFromTemplate(
		templateRunSSHServer, &template.RunSSHServerPayload{
			Port: strconv.Itoa(v.config.Port),
		})
	if err != nil {
		return nil, fmt.Errorf(
			"failed to generate scipt to run ssh server from template %s: %w", templateRunSSHServer, err)
	}

	execOutput, err := devcontainer.ExecuteCommand(ctx, runSSHScript, false, rootUser)
	if err != nil {
		return nil, fmt.Errorf("failed to run SSH serverr: %w", err)
	}

	output += "SSH server run output...\n" + string(execOutput) + "\nSuccessfully run ssh-server\n"

	return []byte(output), nil
}

// Port returns the port on which the ssh-server is listening.
func (v *VSCode) Port() int {
	return v.config.Port
}

func (v *VSCode) Type() enum.IDEType {
	return enum.IDETypeVSCode
}
