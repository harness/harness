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
	"net/url"
	"path"
	"strings"

	"github.com/harness/gitness/app/gitspace/orchestrator/devcontainer"
	"github.com/harness/gitness/app/gitspace/orchestrator/utils"
	gitspaceTypes "github.com/harness/gitness/app/gitspace/types"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

var _ IDE = (*JetBrainsIDE)(nil)

const (
	templateSetupJetBrainsIDE        string = "setup_jetbrains_ide.sh"
	templateSetupJetBrainsIDEPlugins string = "setup_jetbrains_plugins.sh"
	templateRunRemoteJetBrainsIDE    string = "run_jetbrains_ide.sh"

	intellijURLScheme string = "jetbrains-gateway"
)

type JetBrainsIDEConfig struct {
	IntelliJPort int
	GolandPort   int
	PyCharmPort  int
	WebStormPort int
	CLionPort    int
	PHPStormPort int
	RubyMinePort int
	RiderPort    int
}

type JetBrainsIDE struct {
	ideType enum.IDEType
	config  JetBrainsIDEConfig
}

func NewJetBrainsIDEService(config *JetBrainsIDEConfig, ideType enum.IDEType) *JetBrainsIDE {
	return &JetBrainsIDE{
		ideType: ideType,
		config:  *config,
	}
}

func (jb *JetBrainsIDE) port() int {
	switch jb.ideType {
	case enum.IDETypeIntelliJ:
		return jb.config.IntelliJPort
	case enum.IDETypeGoland:
		return jb.config.GolandPort
	case enum.IDETypePyCharm:
		return jb.config.PyCharmPort
	case enum.IDETypeWebStorm:
		return jb.config.WebStormPort
	case enum.IDETypeCLion:
		return jb.config.CLionPort
	case enum.IDETypePHPStorm:
		return jb.config.PHPStormPort
	case enum.IDETypeRubyMine:
		return jb.config.RubyMinePort
	case enum.IDETypeRider:
		return jb.config.RiderPort
	case enum.IDETypeVSCode, enum.IDETypeCursor, enum.IDETypeWindsurf:
		return 0
	case enum.IDETypeVSCodeWeb:
		// IDETypeVSCodeWeb is not JetBrainsIDE
		return 0
	default:
		return 0
	}
}

// Setup installs the SSH server inside the container.
func (jb *JetBrainsIDE) Setup(
	ctx context.Context,
	exec *devcontainer.Exec,
	args map[gitspaceTypes.IDEArg]interface{},
	gitspaceLogger gitspaceTypes.GitspaceLogger,
) error {
	gitspaceLogger.Info("Installing ssh-server inside container...")
	err := setupSSHServer(ctx, exec, gitspaceLogger)
	if err != nil {
		return fmt.Errorf("failed to setup %s IDE: %w", jb.ideType, err)
	}
	gitspaceLogger.Info("Successfully installed ssh-server")

	gitspaceLogger.Info(fmt.Sprintf("Installing %s IDE inside container...", jb.ideType))
	gitspaceLogger.Info("IDE setup output...")
	err = jb.setupJetbrainsIDE(ctx, exec, args, gitspaceLogger)
	if err != nil {
		return fmt.Errorf("failed to setup %s IDE: %w", jb.ideType, err)
	}
	gitspaceLogger.Info(fmt.Sprintf("Successfully installed %s IDE binary", jb.ideType))

	gitspaceLogger.Info("Installing JetBrains plugins inside container...")
	err = jb.setupJetbrainsPlugins(ctx, exec, args, gitspaceLogger)
	if err != nil {
		return fmt.Errorf("failed to setup %s IDE: %w", jb.ideType, err)
	}
	gitspaceLogger.Info(fmt.Sprintf("Successfully installed JetBrains plugins for %s IDE", jb.ideType))

	gitspaceLogger.Info(fmt.Sprintf("Successfully set up %s IDE inside container", jb.ideType))

	return nil
}

func (jb *JetBrainsIDE) setupJetbrainsIDE(
	ctx context.Context,
	exec *devcontainer.Exec,
	args map[gitspaceTypes.IDEArg]interface{},
	gitspaceLogger gitspaceTypes.GitspaceLogger,
) error {
	payload := gitspaceTypes.SetupJetBrainsIDEPayload{
		Username: exec.RemoteUser,
	}

	// get Download URL
	downloadURL, err := getIDEDownloadURL(args)
	if err != nil {
		return err
	}
	payload.IdeDownloadURLArm64 = downloadURL.Arm64
	payload.IdeDownloadURLAmd64 = downloadURL.Amd64

	// get DIR name
	dirName, err := getIDEDirName(args)
	if err != nil {
		return err
	}
	payload.IdeDirName = dirName

	intellijIDEScript, err := utils.GenerateScriptFromTemplate(
		templateSetupJetBrainsIDE, &payload)
	if err != nil {
		return fmt.Errorf(
			"failed to generate script to setup JetBrains idea from template %s: %w",
			templateSetupJetBrainsIDE,
			err,
		)
	}

	err = exec.ExecuteCommandInHomeDirAndLog(ctx, intellijIDEScript,
		false, gitspaceLogger, true)
	if err != nil {
		return fmt.Errorf("failed to setup JetBrains IdeType: %w", err)
	}

	return nil
}

func (jb *JetBrainsIDE) setupJetbrainsPlugins(
	ctx context.Context,
	exec *devcontainer.Exec,
	args map[gitspaceTypes.IDEArg]interface{},
	gitspaceLogger gitspaceTypes.GitspaceLogger,
) error {
	payload := gitspaceTypes.SetupJetBrainsPluginPayload{
		Username: exec.RemoteUser,
	}

	// get DIR name
	dirName, err := getIDEDirName(args)
	if err != nil {
		return err
	}
	payload.IdeDirName = dirName

	// get jetbrains plugins
	customization, exists := args[gitspaceTypes.JetBrainsCustomizationArg]
	if !exists {
		return nil
	}

	jetBrainsCustomization, ok := customization.(types.JetBrainsCustomizationSpecs)
	if !ok {
		return fmt.Errorf("customization is not of type JetBrainsCustomizationSpecs")
	}

	payload.IdePluginsName = strings.Join(jetBrainsCustomization.Plugins, " ")

	gitspaceLogger.Info(fmt.Sprintf(
		"JetBrains Customizations : Plugins %v", jetBrainsCustomization.Plugins))

	intellijIDEScript, err := utils.GenerateScriptFromTemplate(
		templateSetupJetBrainsIDEPlugins, &payload)
	if err != nil {
		return fmt.Errorf(
			"failed to generate script to setup Jetbrains plugins from template %s: %w",
			templateSetupJetBrainsIDEPlugins,
			err,
		)
	}

	err = exec.ExecuteCommandInHomeDirAndLog(ctx, intellijIDEScript,
		false, gitspaceLogger, true)
	if err != nil {
		return fmt.Errorf("failed to setup Jetbrains plugins: %w", err)
	}

	return nil
}

// Run runs the SSH server inside the container.
func (jb *JetBrainsIDE) Run(
	ctx context.Context,
	exec *devcontainer.Exec,
	args map[gitspaceTypes.IDEArg]interface{},
	gitspaceLogger gitspaceTypes.GitspaceLogger,
) error {
	gitspaceLogger.Info("Running ssh-server...")
	err := runSSHServer(ctx, exec, jb.port(), gitspaceLogger)
	if err != nil {
		return fmt.Errorf("failed to run %s IDE: %w", jb.ideType, err)
	}
	gitspaceLogger.Info("Successfully run ssh-server")
	gitspaceLogger.Info(fmt.Sprintf("Running remote %s IDE backend...", jb.ideType))
	err = jb.runRemoteIDE(ctx, exec, args, gitspaceLogger)
	if err != nil {
		return err
	}
	gitspaceLogger.Info(fmt.Sprintf("Successfully run %s IDE backend", jb.ideType))

	return nil
}

func (jb *JetBrainsIDE) runRemoteIDE(
	ctx context.Context,
	exec *devcontainer.Exec,
	args map[gitspaceTypes.IDEArg]interface{},
	gitspaceLogger gitspaceTypes.GitspaceLogger,
) error {
	payload := gitspaceTypes.RunIntellijIDEPayload{
		Username: exec.RemoteUser,
	}
	// get Repository Name
	repoName, err := getRepoName(args)
	if err != nil {
		return err
	}
	payload.RepoName = repoName

	// get DIR name
	dirName, err := getIDEDirName(args)
	if err != nil {
		return err
	}
	payload.IdeDirName = dirName

	runSSHScript, err := utils.GenerateScriptFromTemplate(
		templateRunRemoteJetBrainsIDE, &payload)
	if err != nil {
		return fmt.Errorf(
			"failed to generate scipt to run intelliJ IdeType from template %s: %w", templateRunSSHServer, err)
	}

	err = exec.ExecuteCommandInHomeDirAndLog(ctx, runSSHScript, false, gitspaceLogger, true)
	if err != nil {
		return fmt.Errorf("failed to run intelliJ IdeType: %w", err)
	}

	return nil
}

// Port returns the port on which the ssh-server is listening.
func (jb *JetBrainsIDE) Port() *types.GitspacePort {
	return &types.GitspacePort{
		Port:     jb.port(),
		Protocol: enum.CommunicationProtocolSSH,
	}
}

func (jb *JetBrainsIDE) Type() enum.IDEType {
	return jb.ideType
}

// GenerateURL returns the url to redirect user to ide(here to jetbrains gateway application).
func (jb *JetBrainsIDE) GenerateURL(absoluteRepoPath, host, port, user string) string {
	homePath := getHomePath(absoluteRepoPath)
	idePath := path.Join(homePath, ".cache", "JetBrains", "RemoteDev", "dist", jb.ideType.String())
	ideURL := url.URL{
		Scheme: intellijURLScheme,
		Host:   "", // Empty since we include the host and port in the path
		Path:   "connect",
		Fragment: fmt.Sprintf("idePath=%s&projectPath=%s&host=%s&port=%s&user=%s&type=%s&deploy=%s",
			idePath,
			absoluteRepoPath,
			host,
			port,
			user,
			"ssh",
			"false",
		),
	}

	return ideURL.String()
}

func (jb *JetBrainsIDE) GeneratePluginURL(_, _ string) string {
	return ""
}
