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
	"fmt"
	"io"
	"net/url"
	"os"
	"path"
	"strings"

	"github.com/harness/gitness/git/command"
	"github.com/harness/gitness/types"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

type GenericSCM struct {
}

func NewGenericSCM() *GenericSCM {
	return &GenericSCM{}
}

func (s GenericSCM) GetFileContent(ctx context.Context,
	gitspaceConfig types.GitspaceConfig,
	filePath string,
) ([]byte, error) {
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

	log.Info().Msg("Cloning the repository...")
	cmd := command.New("clone",
		command.WithFlag("--branch", gitspaceConfig.Branch),
		command.WithFlag("--no-checkout"),
		command.WithFlag("--depth", "1"),
		command.WithArg(gitspaceConfig.CodeRepoURL),
		command.WithArg(cloneDir),
	)
	if err := cmd.Run(ctx, command.WithDir(cloneDir)); err != nil {
		return nil, fmt.Errorf("failed to clone repository %s: %w", gitspaceConfig.CodeRepoURL, err)
	}

	var lsTreeOutput bytes.Buffer
	lsTreeCmd := command.New("ls-tree",
		command.WithArg("HEAD"),
		command.WithArg(filePath),
	)

	if err := lsTreeCmd.Run(ctx, command.WithDir(cloneDir), command.WithStdout(&lsTreeOutput)); err != nil {
		return nil, fmt.Errorf("failed to list files in repository %s: %w", cloneDir, err)
	}

	if lsTreeOutput.Len() == 0 {
		log.Info().Msg("File not found, returning empty devcontainerConfig")
		return nil, nil
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
		return nil, fmt.Errorf("failed to read devcontainer file from path %s: %w", filePath, err)
	}
	return catFileOutput.Bytes(), nil
}

func (s GenericSCM) ResolveCredentials(
	_ context.Context,
	gitspaceConfig types.GitspaceConfig,
) (*ResolvedCredentials, error) {
	var resolvedCredentials = &ResolvedCredentials{
		Branch:   gitspaceConfig.Branch,
		CloneURL: gitspaceConfig.CodeRepoURL,
	}
	repoURL, err := url.Parse(gitspaceConfig.CodeRepoURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse repository URL %s: %w", gitspaceConfig.CodeRepoURL, err)
	}
	repoName := strings.TrimSuffix(path.Base(repoURL.Path), ".git")
	resolvedCredentials.RepoName = repoName
	return resolvedCredentials, err
}
