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

package file

import (
	"context"
	"fmt"
	"io"

	"github.com/harness/gitness/gitrpc"
	"github.com/harness/gitness/types"
)

type service struct {
	gitRPCClient gitrpc.Interface
}

func newService(gitRPCClient gitrpc.Interface) Service {
	return &service{gitRPCClient: gitRPCClient}
}

func (f *service) Get(
	ctx context.Context,
	repo *types.Repository,
	path string,
	ref string,
) (*File, error) {
	readParams := gitrpc.ReadParams{
		RepoUID: repo.GitUID,
	}
	treeNodeOutput, err := f.gitRPCClient.GetTreeNode(ctx, &gitrpc.GetTreeNodeParams{
		ReadParams:          readParams,
		GitREF:              ref,
		Path:                path,
		IncludeLatestCommit: false,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to read tree node: %w", err)
	}
	// viewing Raw content is only supported for blob content
	if treeNodeOutput.Node.Type != gitrpc.TreeNodeTypeBlob {
		return nil, fmt.Errorf("path content is not of blob type: %s", treeNodeOutput.Node.Type)
	}

	blobReader, err := f.gitRPCClient.GetBlob(ctx, &gitrpc.GetBlobParams{
		ReadParams: readParams,
		SHA:        treeNodeOutput.Node.SHA,
		SizeLimit:  0, // no size limit, we stream whatever data there is
	})
	if err != nil {
		return nil, fmt.Errorf("failed to read blob from gitrpc: %w", err)
	}

	buf, err := io.ReadAll(blobReader.Content)
	if err != nil {
		return nil, fmt.Errorf("could not read blob content from file: %w", err)
	}

	return &File{
		Data: buf,
	}, nil
}
