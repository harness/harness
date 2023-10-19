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

package codeowners

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/gitrpc"
	"github.com/harness/gitness/types"
)

const (
	// maxGetContentFileSize specifies the maximum number of bytes a file content response contains.
	// If a file is any larger, the content is truncated.
	maxGetContentFileSize = 1 << 20 // 1 MB
)

type Config struct {
	FilePath string
}

type Service struct {
	repoStore store.RepoStore
	git       gitrpc.Interface
	config    Config
}

type codeOwnerFile struct {
	Content string
	SHA     string
}

type CodeOwners struct {
	FileSHA string
	Entries []Entry
}

type Entry struct {
	Pattern string
	Owners  []string
}

func New(
	repoStore store.RepoStore,
	git gitrpc.Interface,
	config Config,
) (*Service, error) {
	service := &Service{
		repoStore: repoStore,
		git:       git,
		config:    config,
	}
	return service, nil
}

func (s *Service) Get(ctx context.Context,
	repoID int64) (*CodeOwners, error) {
	repo, err := s.repoStore.Find(ctx, repoID)
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve repo %w", err)
	}
	codeOwnerFile, err := s.getCodeOwnerFile(ctx, repo)
	if err != nil {
		return nil, fmt.Errorf("unable to get codeowner file %w", err)
	}

	owner, err := s.parseCodeOwner(codeOwnerFile.Content)
	if err != nil {
		return nil, fmt.Errorf("unable to parse codeowner %w", err)
	}

	return &CodeOwners{
		FileSHA: codeOwnerFile.SHA,
		Entries: owner,
	}, nil
}

func (s *Service) parseCodeOwner(codeOwnersContent string) ([]Entry, error) {
	var codeOwners []Entry
	scanner := bufio.NewScanner(strings.NewReader(codeOwnersContent))
	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.Split(line, " ")
		if len(parts) < 2 {
			return nil, fmt.Errorf("line has invalid format: '%s'", line)
		}

		pattern := parts[0]
		owners := parts[1:]

		codeOwner := Entry{
			Pattern: pattern,
			Owners:  owners,
		}

		codeOwners = append(codeOwners, codeOwner)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading input: %w", err)
	}

	return codeOwners, nil
}

func (s *Service) getCodeOwnerFile(ctx context.Context,
	repo *types.Repository,
) (*codeOwnerFile, error) {
	params := gitrpc.CreateRPCReadParams(repo)
	node, err := s.git.GetTreeNode(ctx, &gitrpc.GetTreeNodeParams{
		ReadParams: params,
		GitREF:     "refs/heads/" + repo.DefaultBranch,
		Path:       s.config.FilePath,
	})
	if err != nil {
		// TODO: check for path not found and return empty codeowners
		return nil, fmt.Errorf("unable to retrieve codeowner file %w", err)
	}

	if node.Node.Mode != gitrpc.TreeNodeModeFile {
		return nil, fmt.Errorf(
			"codeowner file is of format '%s' but expected to be of format '%s'",
			node.Node.Mode,
			gitrpc.TreeNodeModeFile,
		)
	}

	output, err := s.git.GetBlob(ctx, &gitrpc.GetBlobParams{
		ReadParams: params,
		SHA:        node.Node.SHA,
		SizeLimit:  maxGetContentFileSize,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get file content: %w", err)
	}

	content, err := io.ReadAll(output.Content)
	if err != nil {
		return nil, fmt.Errorf("failed to read blob content: %w", err)
	}

	return &codeOwnerFile{
		Content: string(content),
		SHA:     output.SHA,
	}, nil
}
