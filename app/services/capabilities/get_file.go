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

package capabilities

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"unicode/utf8"

	"github.com/harness/gitness/app/services/refcache"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/types/capabilities"
	"github.com/harness/gitness/types/check"

	"github.com/rs/zerolog/log"
)

const (
	// maxGetContentFileSize specifies the maximum number of bytes a file content response contains.
	// If a file is any larger, the content is truncated.
	maxGetContentFileSize = 1 << 22 // 4 MB
)

var GetFileType capabilities.Type = "get_file"
var GetFileVersion capabilities.Version = "1.0.0"

type GetFileInput struct {
	RepoREF string `json:"repo_ref"`
	GitREF  string `json:"git_ref"`
	Path    string `json:"path"`
}

func (GetFileInput) IsCapabilityInput() {}

type GetFileOutput struct {
	Content string `json:"content"`
}

func (GetFileOutput) IsCapabilityOutput() {}

func (GetFileOutput) GetName() string {
	return string(GetFileType)
}

const AIContextPayloadTypeGetFile capabilities.AIContextPayloadType = "other"

func (GetFileOutput) GetType() capabilities.AIContextPayloadType {
	return AIContextPayloadTypeGetFile
}

func (r *Registry) RegisterGetFileCapability(
	logic func(ctx context.Context, input *GetFileInput) (*GetFileOutput, error),
) error {
	return r.register(
		capabilities.Capability{
			Type:     GetFileType,
			NewInput: func() capabilities.Input { return &GetFileInput{} },
			Logic:    newLogic(logic),
		},
	)
}

func GetFile(
	repoFinder refcache.RepoFinder,
	gitI git.Interface) func(ctx context.Context, input *GetFileInput) (*GetFileOutput, error) {
	return func(ctx context.Context, input *GetFileInput) (*GetFileOutput, error) {
		if input.RepoREF == "" {
			return nil, check.NewValidationError("repo_ref is required")
		}
		if input.Path == "" {
			return nil, check.NewValidationError("path is required")
		}

		repo, err := repoFinder.FindByRef(ctx, input.RepoREF)
		if err != nil {
			return nil, fmt.Errorf("failed to find repo %q: %w", input.RepoREF, err)
		}

		// set gitRef to default branch in case an empty reference was provided
		gitRef := input.GitREF
		if gitRef == "" {
			gitRef = repo.DefaultBranch
		}

		readParams := git.CreateReadParams(repo)
		node, err := gitI.GetTreeNode(ctx, &git.GetTreeNodeParams{
			ReadParams:          readParams,
			GitREF:              gitRef,
			Path:                input.Path,
			IncludeLatestCommit: false,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to read tree node: %w", err)
		}

		// Todo: Handle symlinks
		return getContent(ctx, gitI, readParams, node)
	}
}

func getContent(
	ctx context.Context,
	gitI git.Interface,
	readParams git.ReadParams,
	node *git.GetTreeNodeOutput) (*GetFileOutput, error) {
	output, err := gitI.GetBlob(ctx, &git.GetBlobParams{
		ReadParams: readParams,
		SHA:        node.Node.SHA,
		SizeLimit:  maxGetContentFileSize,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get file content: %w", err)
	}

	defer func() {
		if err := output.Content.Close(); err != nil {
			log.Ctx(ctx).Warn().Err(err).Msgf("failed to close blob content reader.")
		}
	}()

	content, err := io.ReadAll(output.Content)
	if err != nil {
		return nil, fmt.Errorf("failed to read blob content: %w", err)
	}

	// throw error for binary file
	if i := bytes.IndexByte(content, '\x00'); i > 0 {
		if !utf8.Valid(content[:i]) {
			return nil, check.NewValidationError("file content is not valid UTF-8")
		}
	}

	return &GetFileOutput{
		Content: string(content),
	}, nil
}
