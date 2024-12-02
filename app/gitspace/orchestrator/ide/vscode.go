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
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/harness/gitness/app/gitspace/orchestrator/common"
	"github.com/harness/gitness/app/gitspace/orchestrator/devcontainer"
	"github.com/harness/gitness/app/gitspace/orchestrator/template"
	gitspaceTypes "github.com/harness/gitness/app/gitspace/types"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
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
	args map[string]interface{},
	gitspaceLogger gitspaceTypes.GitspaceLogger,
) error {
	osInfoScript := common.GetOSInfoScript()
	payload := template.SetupSSHServerPayload{
		Username:     exec.RemoteUser,
		AccessType:   exec.AccessType,
		OSInfoScript: osInfoScript,
	}
	if err := v.updateVSCodeSetupPayload(args, gitspaceLogger, &payload); err != nil {
		return err
	}
	sshServerScript, err := template.GenerateScriptFromTemplate(
		templateSetupSSHServer, &payload)
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
	_ map[string]interface{},
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

func (v *VSCode) updateVSCodeSetupPayload(
	args map[string]interface{},
	gitspaceLogger gitspaceTypes.GitspaceLogger,
	payload *template.SetupSSHServerPayload,
) error {
	if args == nil {
		return nil
	}
	// Handle VSCode Customization
	if err := v.handleVSCodeCustomization(args, gitspaceLogger, payload); err != nil {
		return err
	}
	// Handle Repository Name
	if err := v.handleRepoName(args, payload); err != nil {
		return err
	}
	return nil
}

func (v *VSCode) handleVSCodeCustomization(
	args map[string]interface{},
	gitspaceLogger gitspaceTypes.GitspaceLogger,
	payload *template.SetupSSHServerPayload,
) error {
	customization, exists := args[gitspaceTypes.VSCodeCustomizationArg]
	if !exists {
		return nil // No customization found, nothing to do
	}

	// Perform type assertion to ensure it's the correct type
	vsCodeCustomizationSpecs, ok := customization.(types.VSCodeCustomizationSpecs)
	if !ok {
		return fmt.Errorf("customization is not of type VSCodeCustomizationSpecs")
	}

	// Log customization details
	gitspaceLogger.Info(fmt.Sprintf("VSCode Customizations %v", vsCodeCustomizationSpecs))

	// Marshal extensions and set payload
	jsonData, err := json.Marshal(vsCodeCustomizationSpecs.Extensions)
	if err != nil {
		log.Warn().Msg("Error marshalling JSON for VSCode extensions")
		return err
	}
	payload.Extensions = string(jsonData)

	return nil
}

func (v *VSCode) handleRepoName(
	args map[string]interface{},
	payload *template.SetupSSHServerPayload,
) error {
	repoName, exists := args[gitspaceTypes.IDERepoNameArg]
	if !exists {
		return nil // No repo name found, nothing to do
	}

	repoNameStr, ok := repoName.(string)
	if !ok {
		return fmt.Errorf("repo name is not of type string")
	}
	payload.RepoName = repoNameStr

	return nil
}
