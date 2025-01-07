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

var _ IDE = (*Intellij)(nil)

const (
	templateSetupIntellij        string = "setup_intellij.sh"
	templateRunRemoteIDEIntellij string = "run_intellij.sh"

	intellijURLScheme string = "jetbrains-gateway"
)

type IntellijConfig struct {
	Port int
}

type Intellij struct {
	config IntellijConfig
}

func NewIntellijService(config *IntellijConfig) *Intellij {
	return &Intellij{config: *config}
}

// Setup installs the SSH server inside the container.
func (ij *Intellij) Setup(
	ctx context.Context,
	exec *devcontainer.Exec,
	args map[gitspaceTypes.IDEArg]interface{},
	gitspaceLogger gitspaceTypes.GitspaceLogger,
) error {
	gitspaceLogger.Info("Installing ssh-server inside container")
	err := ij.setupSSHServer(ctx, exec, gitspaceLogger)
	if err != nil {
		return fmt.Errorf("failed to setup SSH server: %w", err)
	}
	gitspaceLogger.Info("Successfully installed ssh-server")

	gitspaceLogger.Info("Installing intelliJ IDE inside container")
	gitspaceLogger.Info("IDE setup output...")
	err = ij.setupIntellijIDE(ctx, exec, args, gitspaceLogger)
	if err != nil {
		return fmt.Errorf("failed to setup IntelliJ IDE: %w", err)
	}
	gitspaceLogger.Info("Successfully installed IntelliJ IDE")
	gitspaceLogger.Info("Successfully set up IDE inside container")

	return nil
}

func (ij *Intellij) setupSSHServer(
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

func (ij *Intellij) setupIntellijIDE(
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
		templateSetupIntellij, &payload)
	if err != nil {
		return fmt.Errorf(
			"failed to generate scipt to setup intellij idea from template %s: %w",
			templateSetupIntellij,
			err,
		)
	}

	err = exec.ExecuteCommandInHomeDirAndLog(ctx, intellijIDEScript,
		false, gitspaceLogger, true)
	if err != nil {
		return fmt.Errorf("failed to setup intellij IDE: %w", err)
	}

	return nil
}

// Run runs the SSH server inside the container.
func (ij *Intellij) Run(
	ctx context.Context,
	exec *devcontainer.Exec,
	args map[gitspaceTypes.IDEArg]interface{},
	gitspaceLogger gitspaceTypes.GitspaceLogger,
) error {
	gitspaceLogger.Info("SSH server run output...")
	err := ij.runSSHServer(ctx, exec, args, gitspaceLogger)
	if err != nil {
		return err
	}
	gitspaceLogger.Info("Successfully run ssh-server")
	gitspaceLogger.Info("Run Remote IntelliJ IDE...")
	err = ij.runRemoteIDE(ctx, exec, args, gitspaceLogger)
	if err != nil {
		return err
	}
	gitspaceLogger.Info("Successfully Run Remote IntelliJ IDE")

	return nil
}

func (ij *Intellij) runSSHServer(
	ctx context.Context,
	exec *devcontainer.Exec,
	_ map[gitspaceTypes.IDEArg]interface{},
	gitspaceLogger gitspaceTypes.GitspaceLogger,
) error {
	payload := gitspaceTypes.RunSSHServerPayload{
		Port: strconv.Itoa(ij.config.Port),
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

func (ij *Intellij) runRemoteIDE(
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
		templateRunRemoteIDEIntellij, &payload)
	if err != nil {
		return fmt.Errorf(
			"failed to generate scipt to run intelliJ IDE from template %s: %w", templateRunSSHServer, err)
	}

	err = exec.ExecuteCommandInHomeDirAndLog(ctx, runSSHScript, false, gitspaceLogger, true)
	if err != nil {
		return fmt.Errorf("failed to run intelliJ IDE: %w", err)
	}

	return nil
}

// Port returns the port on which the ssh-server is listening.
func (ij *Intellij) Port() *types.GitspacePort {
	return &types.GitspacePort{
		Port:     ij.config.Port,
		Protocol: enum.CommunicationProtocolSSH,
	}
}

// GenerateURL returns the url to redirect user to ide(here to jetbrains gateway application).
func (ij *Intellij) GenerateURL(absoluteRepoPath, host, port, user string) string {
	homePath := getHomePath(absoluteRepoPath)
	idePath := path.Join(homePath, ".cache", "JetBrains", "RemoteDev", "dist", "intellij")
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

func (ij *Intellij) Type() enum.IDEType {
	return enum.IDETypeIntellij
}
