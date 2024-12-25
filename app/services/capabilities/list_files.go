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
	"context"
	"fmt"

	"github.com/harness/gitness/app/services/refcache"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/capabilities"
	"github.com/harness/gitness/types/check"
)

var ListFilesType capabilities.Type = "list_files"
var ListFilesVersion capabilities.Version = "0"

type ListFilesInput struct {
	RepoREF string `json:"repo_ref"`
	GitRef  string `json:"git_ref"`
	Path    string `json:"path"`
}

func (ListFilesInput) IsCapabilityInput() {}

type ListFilesOutput struct {
	Files       []string `json:"files"`
	Directories []string `json:"directories"`
}

func (ListFilesOutput) IsCapabilityOutput() {}

const AIContextPayloadTypeListFiles capabilities.AIContextPayloadType = "other"

func (ListFilesOutput) GetType() capabilities.AIContextPayloadType {
	return AIContextPayloadTypeListFiles
}

func (ListFilesOutput) GetName() string {
	return string(ListFilesType)
}

func (r *Registry) RegisterListFilesCapability(
	logic func(ctx context.Context, input *ListFilesInput) (*ListFilesOutput, error),
) error {
	return r.register(
		capabilities.Capability{
			Type:         ListFilesType,
			NewInput:     func() capabilities.Input { return &ListFilesInput{} },
			Logic:        newLogic(logic),
			Version:      ListFilesVersion,
			ReturnToUser: false,
		},
	)
}

func ListFiles(
	repoFinder refcache.RepoFinder,
	gitI git.Interface) func(
	ctx context.Context,
	input *ListFilesInput) (*ListFilesOutput, error) {
	return func(ctx context.Context, input *ListFilesInput) (*ListFilesOutput, error) {
		if input.RepoREF == "" {
			return nil, check.NewValidationError("repo_ref is required")
		}

		repo, err := repoFinder.FindByRef(ctx, input.RepoREF)
		if err != nil {
			return nil, fmt.Errorf("failed to find repo %q: %w", input.RepoREF, err)
		}

		// set gitRef to default branch in case an empty reference was provided
		gitRef := input.GitRef
		if gitRef == "" {
			gitRef = repo.DefaultBranch
		}

		return listFiles(ctx, gitI, repo, gitRef, input.Path)
	}
}

func listFiles(
	ctx context.Context,
	gitI git.Interface,
	repo *types.Repository,
	gitRef string,
	path string) (*ListFilesOutput, error) {
	files := make([]string, 0)
	directories := make([]string, 0)
	tree, err := gitI.ListTreeNodes(ctx, &git.ListTreeNodeParams{
		ReadParams: git.CreateReadParams(repo),
		GitREF:     gitRef,
		Path:       path,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list tree nodes: %w", err)
	}
	for _, v := range tree.Nodes {
		if v.Type == git.TreeNodeTypeBlob {
			files = append(files, v.Path)
		} else if v.Type == git.TreeNodeTypeTree {
			directories = append(directories, v.Path)
		}
	}

	return &ListFilesOutput{
		Files:       files,
		Directories: directories,
	}, nil
}
