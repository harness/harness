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
	"net/url"
	"path/filepath"
	"strings"

	"github.com/harness/gitness/app/gitspace/orchestrator/devcontainer"
	"github.com/harness/gitness/app/gitspace/orchestrator/utils"
	gitspaceTypes "github.com/harness/gitness/app/gitspace/types"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

var _ IDE = (*VSCode)(nil)

const (
	templateSetupVSCodeExtensions string = "setup_vscode_extensions.sh"

	vSCodeURLScheme string = "vscode-remote"
)

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
	args map[gitspaceTypes.IDEArg]interface{},
	gitspaceLogger gitspaceTypes.GitspaceLogger,
) error {
	gitspaceLogger.Info("Installing ssh-server inside container...")
	err := setupSSHServer(ctx, exec, gitspaceLogger)
	if err != nil {
		return fmt.Errorf("failed to setup %s IDE: %w", enum.IDETypeVSCode, err)
	}
	gitspaceLogger.Info("Successfully installed ssh-server")

	gitspaceLogger.Info("Installing vs-code extensions inside container...")
	err = v.setupVSCodeExtensions(ctx, exec, args, gitspaceLogger)
	if err != nil {
		return fmt.Errorf("failed to setup vs code extensions: %w", err)
	}
	gitspaceLogger.Info("Successfully installed vs-code extensions")
	gitspaceLogger.Info(fmt.Sprintf("Successfully set up %s IDE inside container", enum.IDETypeVSCode))

	return nil
}

func (v *VSCode) setupVSCodeExtensions(
	ctx context.Context,
	exec *devcontainer.Exec,
	args map[gitspaceTypes.IDEArg]interface{},
	gitspaceLogger gitspaceTypes.GitspaceLogger,
) error {
	payload := gitspaceTypes.SetupVSCodeExtensionsPayload{
		Username: exec.RemoteUser,
	}
	if err := v.updateVSCodeSetupPayload(args, gitspaceLogger, &payload); err != nil {
		return err
	}

	vscodeExtensionsScript, err := utils.GenerateScriptFromTemplate(
		templateSetupVSCodeExtensions, &payload)
	if err != nil {
		return fmt.Errorf(
			"failed to generate scipt to setup vscode extensions from template %s: %w",
			templateSetupVSCodeExtensions,
			err,
		)
	}

	err = exec.ExecuteCommandInHomeDirAndLog(ctx, vscodeExtensionsScript,
		true, gitspaceLogger, false)
	if err != nil {
		return fmt.Errorf("failed to setup vs-code extensions: %w", err)
	}

	return nil
}

// Run runs the SSH server inside the container.
func (v *VSCode) Run(
	ctx context.Context,
	exec *devcontainer.Exec,
	_ map[gitspaceTypes.IDEArg]interface{},
	gitspaceLogger gitspaceTypes.GitspaceLogger,
) error {
	gitspaceLogger.Info("Running ssh-server...")
	err := runSSHServer(ctx, exec, v.config.Port, gitspaceLogger)
	if err != nil {
		return fmt.Errorf("failed to run %s IDE: %w", enum.IDETypeVSCode, err)
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
	args map[gitspaceTypes.IDEArg]interface{},
	gitspaceLogger gitspaceTypes.GitspaceLogger,
	payload *gitspaceTypes.SetupVSCodeExtensionsPayload,
) error {
	if args == nil {
		return nil
	}
	// Handle VSCode Customization
	if err := v.handleVSCodeCustomization(args, gitspaceLogger, payload); err != nil {
		return err
	}
	// Handle Repository Name
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

func (v *VSCode) handleVSCodeCustomization(
	args map[gitspaceTypes.IDEArg]interface{},
	gitspaceLogger gitspaceTypes.GitspaceLogger,
	payload *gitspaceTypes.SetupVSCodeExtensionsPayload,
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
	gitspaceLogger.Info(fmt.Sprintf(
		"VSCode Customizations : Extensions %v", vsCodeCustomizationSpecs.Extensions))

	// Marshal extensions and set payload
	jsonData, err := json.Marshal(vsCodeCustomizationSpecs.Extensions)
	if err != nil {
		log.Warn().Msg("Error marshalling JSON for VSCode extensions")
		return err
	}
	payload.Extensions = string(jsonData)

	return nil
}

// GenerateURL returns the url to redirect user to ide(here to vscode application).
func (v *VSCode) GenerateURL(absoluteRepoPath, host, port, user string) string {
	relativeRepoPath := strings.TrimPrefix(absoluteRepoPath, "/")
	ideURL := url.URL{
		Scheme: vSCodeURLScheme,
		Host:   "", // Empty since we include the host and port in the path
		Path: fmt.Sprintf(
			"ssh-remote+%s@%s:%s",
			user,
			host,
			filepath.Join(port, relativeRepoPath),
		),
	}

	return ideURL.String()
}
