// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

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

func new(gitRPCClient gitrpc.Interface) FileService {
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
