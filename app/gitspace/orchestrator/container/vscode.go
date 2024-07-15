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

	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

var _ IDE = (*VSCode)(nil)

const sshPort = "22/tcp"

type VSCode struct{}

func NewVsCodeService() *VSCode {
	return &VSCode{}
}

// Setup installs and runs SSH server inside the container.
func (v *VSCode) Setup(
	ctx context.Context,
	devcontainer *Devcontainer,
	gitspaceInstance *types.GitspaceInstance,
) ([]byte, error) {
	var output = ""

	sshServerScript, err := GenerateScriptFromTemplate(
		templateSetupSSHServer, &SetupSSHServerPayload{
			Username:         "harness",
			Password:         *gitspaceInstance.AccessKey,
			WorkingDirectory: devcontainer.WorkingDir,
		})
	if err != nil {
		return nil, fmt.Errorf(
			"failed to generate scipt to setup ssh server from template %s: %w", templateSetupSSHServer, err)
	}

	output += "Installing ssh-server inside container\n"

	execOutput, err := devcontainer.ExecuteCommand(ctx, sshServerScript, false)
	if err != nil {
		return nil, fmt.Errorf("failed to setup SSH serverr: %w", err)
	}

	output += "SSH server installation output...\n" + string(execOutput) + "\nSuccessfully installed ssh-server\n"

	return []byte(output), nil
}

// PortAndProtocol returns the port on which the ssh-server is listening.
func (v *VSCode) PortAndProtocol() string {
	return sshPort
}

func (v *VSCode) Type() enum.IDEType {
	return enum.IDETypeVSCode
}
