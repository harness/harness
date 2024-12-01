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

	"github.com/harness/gitness/app/gitspace/orchestrator/common"
	"github.com/harness/gitness/app/gitspace/orchestrator/devcontainer"
	"github.com/harness/gitness/app/gitspace/orchestrator/template"
	gitspaceTypes "github.com/harness/gitness/app/gitspace/types"
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
	exec *devcontainer.Exec,
	gitspaceLogger gitspaceTypes.GitspaceLogger,
) error {
	osInfoScript := common.GetOSInfoScript()
	sshServerScript, err := template.GenerateScriptFromTemplate(
		templateSetupSSHServer, &template.SetupSSHServerPayload{
			Username:     exec.RemoteUser,
			AccessType:   exec.AccessType,
			OSInfoScript: osInfoScript,
		})
	if err != nil {
		return fmt.Errorf(
			"failed to generate scipt to setup ssh server from template %s: %w", templateSetupSSHServer, err)
	}

	gitspaceLogger.Info("Installing ssh-server inside container")
	gitspaceLogger.Info("IDE setup output...")
	err = common.ExecuteCommandInHomeDirAndLog(ctx, exec, sshServerScript, true, gitspaceLogger)
	if err != nil {
		return fmt.Errorf("failed to setup SSH serverr: %w", err)
	}
	gitspaceLogger.Info("Successfully installed ssh-server")
	gitspaceLogger.Info("Successfully set up IDE inside container")
	return nil
}

// Run runs the SSH server inside the container.
func (v *VSCode) Run(
	ctx context.Context,
	exec *devcontainer.Exec,
	args map[string]interface{},
	gitspaceLogger gitspaceTypes.GitspaceLogger,
) error {
	payload := template.RunSSHServerPayload{
		Port: strconv.Itoa(v.config.Port),
	}
	runSSHScript, err := template.GenerateScriptFromTemplate(
		templateRunSSHServer, &payload)
	if err != nil {
		return fmt.Errorf(
			"failed to generate scipt to run ssh server from template %s: %w", templateRunSSHServer, err)
	}
	if args != nil {
		if customization, exists := args["customization"]; exists {
			// Perform a type assertion to ensure customization is a VSCodeCustomizationSpecs
			if vsCodeCustomizationSpecs, ok := customization.(types.VSCodeCustomizationSpecs); ok {
				gitspaceLogger.Info(fmt.Sprintf("VSCode Customizations %v", vsCodeCustomizationSpecs))
			} else {
				return fmt.Errorf("customization is not of type VSCodeCustomizationSpecs")
			}
		}
	}
	gitspaceLogger.Info("SSH server run output...")
	err = common.ExecuteCommandInHomeDirAndLog(ctx, exec, runSSHScript, true, gitspaceLogger)
	if err != nil {
		return fmt.Errorf("failed to run SSH server: %w", err)
	}
	gitspaceLogger.Info("Successfully run ssh-server")

	return nil
}

// Port returns the port on which the ssh-server is listening.
func (v *VSCode) Port() *types.GitspacePort {
	return &types.GitspacePort{
		Port:     v.config.Port,
		Protocol: enum.CommunicationProtocolSSH,
	}
}

func (v *VSCode) Type() enum.IDEType {
	return enum.IDETypeVSCode
}
