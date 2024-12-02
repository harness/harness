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

package git

import (
	"context"
	"fmt"
	"net/url"

	"github.com/harness/gitness/app/gitspace/orchestrator/common"
	"github.com/harness/gitness/app/gitspace/orchestrator/devcontainer"
	"github.com/harness/gitness/app/gitspace/orchestrator/template"
	"github.com/harness/gitness/app/gitspace/scm"
	"github.com/harness/gitness/app/gitspace/types"
)

var _ Service = (*ServiceImpl)(nil)

const templateGitInstallScript string = "install_git.sh"
const templateSetupGitCredentials = "setup_git_credentials.sh" // nolint:gosec
const templateCloneCode = "clone_code.sh"

type ServiceImpl struct {
}

func NewGitServiceImpl() Service {
	return &ServiceImpl{}
}

func (g *ServiceImpl) Install(
	ctx context.Context,
	exec *devcontainer.Exec,
	gitspaceLogger types.GitspaceLogger,
) error {
	script, err := template.GenerateScriptFromTemplate(
		templateGitInstallScript, &template.SetupGitInstallPayload{
			OSInfoScript: common.GetOSInfoScript(),
		})
	if err != nil {
		return fmt.Errorf(
			"failed to generate scipt to setup git install from template %s: %w", templateGitInstallScript, err)
	}
	gitspaceLogger.Info("Install git output...")
	gitspaceLogger.Info("Setting up git inside container")
	err = common.ExecuteCommandInHomeDirAndLog(ctx, exec, script, true, gitspaceLogger, false)
	if err != nil {
		return fmt.Errorf("failed to setup git: %w", err)
	}
	gitspaceLogger.Info("Successfully setup git")

	return nil
}

func (g *ServiceImpl) SetupCredentials(
	ctx context.Context,
	exec *devcontainer.Exec,
	resolvedRepoDetails scm.ResolvedDetails,
	gitspaceLogger types.GitspaceLogger,
) error {
	script, err := template.GenerateScriptFromTemplate(
		templateSetupGitCredentials, &template.SetupGitCredentialsPayload{
			CloneURLWithCreds: resolvedRepoDetails.CloneURL.Value(),
		})
	if err != nil {
		return fmt.Errorf(
			"failed to generate scipt to setup git credentials from template %s: %w", templateSetupGitCredentials, err)
	}
	gitspaceLogger.Info("Setting up git credentials output...")
	gitspaceLogger.Info("Setting up git credentials inside container")
	err = common.ExecuteCommandInHomeDirAndLog(ctx, exec, script, false, gitspaceLogger, true)
	if err != nil {
		return fmt.Errorf("failed to setup git credentials: %w", err)
	}
	gitspaceLogger.Info("Successfully setup git credentials")
	return nil
}

func (g *ServiceImpl) CloneCode(
	ctx context.Context,
	exec *devcontainer.Exec,
	resolvedRepoDetails scm.ResolvedDetails,
	defaultBaseImage string,
	gitspaceLogger types.GitspaceLogger,
) error {
	cloneURL, err := url.Parse(resolvedRepoDetails.CloneURL.Value())
	if err != nil {
		return fmt.Errorf(
			"failed to parse clone url %s: %w", resolvedRepoDetails.CloneURL, err)
	}
	cloneURL.User = nil
	data := &template.CloneCodePayload{
		RepoURL:  cloneURL.String(),
		Image:    defaultBaseImage,
		Branch:   resolvedRepoDetails.Branch,
		RepoName: resolvedRepoDetails.RepoName,
	}
	if resolvedRepoDetails.ResolvedCredentials.Credentials != nil {
		data.Email = resolvedRepoDetails.Credentials.Email
		data.Name = resolvedRepoDetails.Credentials.Name.Value()
	}
	script, err := template.GenerateScriptFromTemplate(
		templateCloneCode, data)
	if err != nil {
		return fmt.Errorf(
			"failed to generate scipt to clone code from template %s: %w", templateCloneCode, err)
	}
	gitspaceLogger.Info("Cloning code inside container")
	err = common.ExecuteCommandInHomeDirAndLog(ctx, exec, script, false, gitspaceLogger, true)
	if err != nil {
		return fmt.Errorf("failed to clone code: %w", err)
	}
	gitspaceLogger.Info("Successfully clone code")

	return nil
}
