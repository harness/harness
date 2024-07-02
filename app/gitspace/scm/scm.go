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
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/harness/gitness/git/command"
	"github.com/harness/gitness/types"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

var _ SCM = (*scm)(nil)

type SCM interface {
	// DevcontainerConfig fetches devcontainer config file from the given repo and branch.
	DevcontainerConfig(ctx context.Context, gitspaceConfig *types.GitspaceConfig) (*types.DevcontainerConfig, error)
}

type scm struct{}

func NewSCM() SCM {
	return &scm{}
}

func (s scm) DevcontainerConfig(
	ctx context.Context,
	gitspaceConfig *types.GitspaceConfig,
) (*types.DevcontainerConfig, error) {
	gitWorkingDirectory := "/tmp/git/"
	cloneDir := gitWorkingDirectory + uuid.New().String()
	err := os.MkdirAll(cloneDir, os.ModePerm)
	if err != nil {
		return nil, fmt.Errorf("error creating directory %s: %w", cloneDir, err)
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
		return nil, fmt.Errorf("invalid branch or url: %w", err)
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
		return nil, fmt.Errorf("failed to clone repository %s: %w", gitspaceConfig.CodeRepoURL, err)
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
		return nil, fmt.Errorf("failed to list files in repository %s: %w", cloneDir, err)
	}

	if lsTreeOutput.Len() == 0 {
		log.Info().Msg("File not found, returning empty devcontainerConfig")
		emptyConfig := &types.DevcontainerConfig{}
		return emptyConfig, nil
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
		return nil, fmt.Errorf("failed to checkout devcontainer file from path %s: %w", filePath, err)
	}

	sanitizedJSON := removeComments(catFileOutput.Bytes())

	var config types.DevcontainerConfig
	err = json.Unmarshal(sanitizedJSON, &config)
	if err != nil {
		return nil, fmt.Errorf("failed to parse devcontainer json: %w", err)
	}

	return &config, nil
}

func removeComments(input []byte) []byte {
	blockCommentRegex := regexp.MustCompile(`(?s)/\*.*?\*/`)
	input = blockCommentRegex.ReplaceAll(input, nil)
	lineCommentRegex := regexp.MustCompile(`//.*`)
	return lineCommentRegex.ReplaceAll(input, nil)
}

func validateArgs(_ *types.GitspaceConfig) error {
	// TODO Validate the args
	return nil
}
