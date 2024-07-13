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
	"io"
	"net/url"
	"os"
	"path"
	"regexp"
	"strings"

	"github.com/harness/gitness/git/command"
	"github.com/harness/gitness/types"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

var (
	ErrNoDefaultBranch = errors.New("no default branch")
)

var _ SCM = (*scm)(nil)

type SCM interface {
	// RepoNameAndDevcontainerConfig fetches repository name & devcontainer config file from the given repo and branch.
	RepoNameAndDevcontainerConfig(
		ctx context.Context,
		gitspaceConfig *types.GitspaceConfig,
	) (string, *types.DevcontainerConfig, error)

	// CheckValidCodeRepo checks if the current URL is a valid and accessible code repo,
	// input can be connector info, user token etc.
	CheckValidCodeRepo(ctx context.Context, request CodeRepositoryRequest) (*CodeRepositoryResponse, error)
}

type scm struct{}

func (s scm) CheckValidCodeRepo(ctx context.Context, request CodeRepositoryRequest) (*CodeRepositoryResponse, error) {
	err := validateURL(request)
	if err != nil {
		return nil, fmt.Errorf("invalid URL, %w", err)
	}
	codeRepositoryResponse := &CodeRepositoryResponse{
		URL:               request.URL,
		CodeRepoIsPrivate: true,
	}
	defaultBranch, err := detectDefaultGitBranch(ctx, request.URL)
	if err == nil {
		branch := "main"
		if defaultBranch != "" {
			branch = defaultBranch
		}
		codeRepositoryResponse = &CodeRepositoryResponse{
			URL:               request.URL,
			Branch:            branch,
			CodeRepoIsPrivate: false,
		}
	}
	return codeRepositoryResponse, nil
}

func NewSCM() SCM {
	return &scm{}
}

func (s scm) RepoNameAndDevcontainerConfig(
	ctx context.Context,
	gitspaceConfig *types.GitspaceConfig,
) (string, *types.DevcontainerConfig, error) {
	repoURL, err := url.Parse(gitspaceConfig.CodeRepoURL)
	if err != nil {
		return "", nil, fmt.Errorf("failed to parse repository URL %s: %w", gitspaceConfig.CodeRepoURL, err)
	}
	repoName := strings.TrimSuffix(path.Base(repoURL.Path), ".git")

	gitWorkingDirectory := "/tmp/git/"

	cloneDir := gitWorkingDirectory + uuid.New().String()

	err = os.MkdirAll(cloneDir, os.ModePerm)
	if err != nil {
		return "", nil, fmt.Errorf("error creating directory %s: %w", cloneDir, err)
	}

	defer func() {
		err = os.RemoveAll(cloneDir)
		if err != nil {
			log.Ctx(ctx).Warn().Err(err).Msg("Unable to remove working directory")
		}
	}()

	filePath := ".devcontainer/devcontainer.json"
	err = validateArgs(gitspaceConfig)
	if err != nil {
		return "", nil, fmt.Errorf("invalid branch or url: %w", err)
	}

	log.Info().Msg("Cloning the repository...")
	cmd := command.New("clone",
		command.WithFlag("--branch", gitspaceConfig.Branch),
		command.WithFlag("--no-checkout"),
		command.WithFlag("--depth", "1"),
		command.WithArg(gitspaceConfig.CodeRepoURL),
		command.WithArg(cloneDir),
	)
	err = cmd.Run(
		ctx,
		command.WithDir(cloneDir),
	)
	if err != nil {
		return "", nil, fmt.Errorf("failed to clone repository %s: %w", gitspaceConfig.CodeRepoURL, err)
	}

	var lsTreeOutput bytes.Buffer
	lsTreeCmd := command.New("ls-tree",
		command.WithArg("HEAD"),
		command.WithArg(filePath),
	)
	err = lsTreeCmd.Run(
		ctx,
		command.WithDir(cloneDir),
		command.WithStdout(&lsTreeOutput),
	)
	if err != nil {
		return "", nil, fmt.Errorf("failed to list files in repository %s: %w", cloneDir, err)
	}

	if lsTreeOutput.Len() == 0 {
		log.Info().Msg("File not found, returning empty devcontainerConfig")
		emptyConfig := &types.DevcontainerConfig{}
		return repoName, emptyConfig, nil
	}

	fields := strings.Fields(lsTreeOutput.String())
	blobSHA := fields[2]

	var catFileOutput bytes.Buffer
	catFileCmd := command.New("cat-file", command.WithFlag("-p"), command.WithArg(blobSHA))
	err = catFileCmd.Run(
		ctx,
		command.WithDir(cloneDir),
		command.WithStderr(io.Discard),
		command.WithStdout(&catFileOutput),
	)
	if err != nil {
		return "", nil, fmt.Errorf("failed to read devcontainer file from path %s: %w", filePath, err)
	}

	sanitizedJSON := removeComments(catFileOutput.Bytes())

	var config types.DevcontainerConfig
	err = json.Unmarshal(sanitizedJSON, &config)
	if err != nil {
		return "", nil, fmt.Errorf("failed to parse devcontainer json: %w", err)
	}

	return repoName, &config, nil
}

func removeComments(input []byte) []byte {
	blockCommentRegex := regexp.MustCompile(`(?s)/\*.*?\*/`)
	input = blockCommentRegex.ReplaceAll(input, nil)
	lineCommentRegex := regexp.MustCompile(`//.*`)
	return lineCommentRegex.ReplaceAll(input, nil)
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

func validateURL(request CodeRepositoryRequest) error {
	if _, err := url.ParseRequestURI(request.URL); err != nil {
		return err
	}
	return nil
}

func validateArgs(_ *types.GitspaceConfig) error {
	// TODO Validate the args
	return nil
}
