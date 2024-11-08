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

func (g *ServiceImpl) Install(ctx context.Context, exec *devcontainer.Exec) ([]byte, error) {
	script, err := template.GenerateScriptFromTemplate(
		templateGitInstallScript, &template.SetupGitInstallPayload{
			OSInfoScript: common.GetOSInfoScript(),
		})
	if err != nil {
		return nil, fmt.Errorf(
			"failed to generate scipt to setup git install from template %s: %w", templateGitInstallScript, err)
	}
	output := "Setting up git inside container\n"
	_, err = exec.ExecuteCommandInHomeDirectory(ctx, script, true, false)
	if err != nil {
		return nil, fmt.Errorf("failed to setup git: %w", err)
	}

	output += "Successfully setup git\n"

	return []byte(output), nil
}

func (g *ServiceImpl) SetupCredentials(
	ctx context.Context,
	exec *devcontainer.Exec,
	resolvedRepoDetails scm.ResolvedDetails,
) ([]byte, error) {
	script, err := template.GenerateScriptFromTemplate(
		templateSetupGitCredentials, &template.SetupGitCredentialsPayload{
			CloneURLWithCreds: resolvedRepoDetails.CloneURL,
		})
	if err != nil {
		return nil, fmt.Errorf(
			"failed to generate scipt to setup git credentials from template %s: %w", templateSetupGitCredentials, err)
	}

	output := "Setting up git credentials inside container\n"

	_, err = exec.ExecuteCommandInHomeDirectory(ctx, script, false, false)
	if err != nil {
		return nil, fmt.Errorf("failed to setup git credentials: %w", err)
	}

	output += "Successfully setup git credentials\n"

	return []byte(output), nil
}

func (g *ServiceImpl) CloneCode(
	ctx context.Context,
	exec *devcontainer.Exec,
	resolvedRepoDetails scm.ResolvedDetails,
	defaultBaseImage string,
) ([]byte, error) {
	cloneURL, err := url.Parse(resolvedRepoDetails.CloneURL)
	if err != nil {
		return nil, fmt.Errorf(
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
		data.Name = resolvedRepoDetails.Credentials.Name
	}
	script, err := template.GenerateScriptFromTemplate(
		templateCloneCode, data)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to generate scipt to clone code from template %s: %w", templateCloneCode, err)
	}

	output := "Cloning code inside container\n"

	_, err = exec.ExecuteCommandInHomeDirectory(ctx, script, false, false)
	if err != nil {
		return nil, fmt.Errorf("failed to clone code: %w", err)
	}

	output += "Successfully clone code\n"

	return []byte(output), nil
}
