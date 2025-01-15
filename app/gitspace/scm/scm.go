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

package scm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/harness/gitness/git/command"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/tidwall/jsonc"
)

var (
	ErrNoDefaultBranch = errors.New("no default branch")
)

const devcontainerDefaultPath = ".devcontainer/devcontainer.json"

type SCM struct {
	scmProviderFactory Factory
}

func NewSCM(factory Factory) *SCM {
	return &SCM{scmProviderFactory: factory}
}

// CheckValidCodeRepo checks if the current URL is a valid and accessible code repo,
// input can be connector info, user token etc.
func (s *SCM) CheckValidCodeRepo(
	ctx context.Context,
	codeRepositoryRequest CodeRepositoryRequest,
) (*CodeRepositoryResponse, error) {
	codeRepositoryResponse := &CodeRepositoryResponse{
		URL:               codeRepositoryRequest.URL,
		CodeRepoIsPrivate: true,
	}

	branch, err := s.detectBranch(ctx, codeRepositoryRequest.URL)
	if err == nil {
		codeRepositoryResponse.Branch = branch
		codeRepositoryResponse.CodeRepoIsPrivate = false
		return codeRepositoryResponse, nil
	}

	scmProvider, err := s.getSCMProvider(codeRepositoryRequest.RepoType)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve SCM provider: %w", err)
	}

	resolvedCreds, err := s.resolveRepoCredentials(ctx, scmProvider, codeRepositoryRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve repo credentials and URL: %w", err)
	}

	if branch, err = s.detectBranch(ctx, resolvedCreds.CloneURL.Value()); err == nil {
		codeRepositoryResponse.Branch = branch
	}
	return codeRepositoryResponse, nil
}

// GetSCMRepoDetails fetches repository name, credentials & devcontainer config file from the given repo and branch.
func (s *SCM) GetSCMRepoDetails(
	ctx context.Context,
	gitspaceConfig types.GitspaceConfig,
) (*ResolvedDetails, error) {
	scmProvider, err := s.getSCMProvider(gitspaceConfig.CodeRepo.Type)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve SCM provider: %w", err)
	}

	resolvedCredentials, err := scmProvider.ResolveCredentials(ctx, gitspaceConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve repo credentials and url: %w", err)
	}

	devcontainerConfig, err := s.getDevcontainerConfig(ctx, scmProvider, gitspaceConfig, resolvedCredentials)
	if err != nil {
		return nil, fmt.Errorf("failed to read or parse devcontainer config: %w", err)
	}
	var resolvedDetails = &ResolvedDetails{
		ResolvedCredentials: *resolvedCredentials,
		DevcontainerConfig:  devcontainerConfig,
	}
	return resolvedDetails, nil
}

func detectDefaultGitBranch(ctx context.Context, gitRepoDir string) (string, error) {
	cmd := command.New("ls-remote",
		command.WithFlag("--symref"),
		command.WithFlag("-q"),
		command.WithArg(gitRepoDir),
		command.WithArg("HEAD"),
	)
	output := &bytes.Buffer{}
	if err := cmd.Run(ctx, command.WithStdout(output)); err != nil {
		return "", fmt.Errorf("failed to ls remote repo")
	}
	var lsRemoteHeadRegexp = regexp.MustCompile(`ref: refs/heads/([^\s]+)\s+HEAD`)
	match := lsRemoteHeadRegexp.FindStringSubmatch(strings.TrimSpace(output.String()))
	if match == nil {
		return "", ErrNoDefaultBranch
	}
	return match[1], nil
}

func (s *SCM) GetBranchURL(
	spacePath string,
	repoType enum.GitspaceCodeRepoType,
	repoURL string,
	branch string,
) (string, error) {
	scmProvider, err := s.scmProviderFactory.GetSCMProvider(repoType)
	if err != nil {
		return "", fmt.Errorf("failed to resolve scm provider while generating branch url: %w", err)
	}
	return scmProvider.GetBranchURL(spacePath, repoURL, branch)
}

// detectBranch tries to detect the default Git branch for a given URL.
func (s *SCM) detectBranch(ctx context.Context, repoURL string) (string, error) {
	defaultBranch, err := detectDefaultGitBranch(ctx, repoURL)
	if err != nil {
		return "", err
	}
	if defaultBranch == "" {
		return "main", nil
	}
	return defaultBranch, nil
}

func (s *SCM) getSCMProvider(repoType enum.GitspaceCodeRepoType) (Provider, error) {
	if repoType == "" {
		repoType = enum.CodeRepoTypeUnknown
	}
	return s.scmProviderFactory.GetSCMProvider(repoType)
}

func (s *SCM) resolveRepoCredentials(
	ctx context.Context,
	scmProvider Provider,
	codeRepositoryRequest CodeRepositoryRequest,
) (*ResolvedCredentials, error) {
	codeRepo := types.CodeRepo{URL: codeRepositoryRequest.URL}
	gitspaceUser := types.GitspaceUser{Identifier: codeRepositoryRequest.UserIdentifier}
	gitspaceConfig := types.GitspaceConfig{
		CodeRepo:     codeRepo,
		SpacePath:    codeRepositoryRequest.SpacePath,
		GitspaceUser: gitspaceUser,
	}
	return scmProvider.ResolveCredentials(ctx, gitspaceConfig)
}

func (s *SCM) getDevcontainerConfig(
	ctx context.Context,
	scmProvider Provider,
	gitspaceConfig types.GitspaceConfig,
	resolvedCredentials *ResolvedCredentials,
) (types.DevcontainerConfig, error) {
	config := types.DevcontainerConfig{}
	filePath := devcontainerDefaultPath
	catFileOutputBytes, err := scmProvider.GetFileContent(ctx, gitspaceConfig, filePath, resolvedCredentials)
	if err != nil {
		return config, fmt.Errorf("failed to read devcontainer file: %w", err)
	}

	if len(catFileOutputBytes) == 0 {
		return config, nil // Return an empty config if the file is empty
	}

	if err = json.Unmarshal(jsonc.ToJSON(catFileOutputBytes), &config); err != nil {
		return config, fmt.Errorf("failed to parse devcontainer JSON: %w", err)
	}

	return config, nil
}
