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
	"strconv"

	"github.com/harness/gitness/app/gitspace/orchestrator/devcontainer"
	"github.com/harness/gitness/app/gitspace/orchestrator/utils"
	gitspaceTypes "github.com/harness/gitness/app/gitspace/types"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

var _ IDE = (*JetBrainsIDE)(nil)

const (
	templateSetupJetBrainsIDE     string = "setup_jetbrains_ide.sh"
	templateRunRemoteJetBrainsIDE string = "run_jetbrains_ide.sh"

	intellijURLScheme string = "jetbrains-gateway"
)

type JetBrainsIDEConfig struct {
	Port int
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

// Setup installs the SSH server inside the container.
func (jb *JetBrainsIDE) Setup(
	ctx context.Context,
	exec *devcontainer.Exec,
	args map[gitspaceTypes.IDEArg]interface{},
	gitspaceLogger gitspaceTypes.GitspaceLogger,
) error {
	gitspaceLogger.Info("Installing ssh-server inside container")
	err := jb.setupSSHServer(ctx, exec, gitspaceLogger)
	if err != nil {
		return fmt.Errorf("failed to setup SSH server: %w", err)
	}
	gitspaceLogger.Info("Successfully installed ssh-server")

	gitspaceLogger.Info(fmt.Sprintf("Installing %s IdeType inside container...", jb.ideType))
	gitspaceLogger.Info("IDE setup output...")
	err = jb.setupIntellijIDE(ctx, exec, args, gitspaceLogger)
	if err != nil {
		return fmt.Errorf("failed to setup %s IdeType: %w", jb.ideType, err)
	}
	gitspaceLogger.Info(fmt.Sprintf("Successfully installed %s IdeType", jb.ideType))
	gitspaceLogger.Info("Successfully set up IDE inside container")

	return nil
}

func (jb *JetBrainsIDE) setupSSHServer(
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

func (jb *JetBrainsIDE) setupIntellijIDE(
	ctx context.Context,
	exec *devcontainer.Exec,
	args map[gitspaceTypes.IDEArg]interface{},
	gitspaceLogger gitspaceTypes.GitspaceLogger,
) error {
	payload := gitspaceTypes.SetupIntellijIDEPayload{
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
			"failed to generate scipt to setup intellij idea from template %s: %w",
			templateSetupJetBrainsIDE,
			err,
		)
	}

	err = exec.ExecuteCommandInHomeDirAndLog(ctx, intellijIDEScript,
		false, gitspaceLogger, true)
	if err != nil {
		return fmt.Errorf("failed to setup intellij IdeType: %w", err)
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
	gitspaceLogger.Info("SSH server run output...")
	err := jb.runSSHServer(ctx, exec, args, gitspaceLogger)
	if err != nil {
		return err
	}
	gitspaceLogger.Info("Successfully run ssh-server")
	gitspaceLogger.Info("Run Remote IntelliJ IdeType...")
	err = jb.runRemoteIDE(ctx, exec, args, gitspaceLogger)
	if err != nil {
		return err
	}
	gitspaceLogger.Info("Successfully Run Remote IntelliJ IdeType")

	return nil
}

func (jb *JetBrainsIDE) runSSHServer(
	ctx context.Context,
	exec *devcontainer.Exec,
	_ map[gitspaceTypes.IDEArg]interface{},
	gitspaceLogger gitspaceTypes.GitspaceLogger,
) error {
	payload := gitspaceTypes.RunSSHServerPayload{
		Port: strconv.Itoa(jb.config.Port),
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
	gitspaceLogger.Info("Successfully run ssh-server")

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
		Port:     jb.config.Port,
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
