// Copyright 2026 Harness, Inc.
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
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/harness/gitness/git"
	"github.com/harness/gitness/types"

	"github.com/stretchr/testify/require"
)

type infGit struct {
	git.Interface

	node    git.TreeNode
	nodeErr error
	blobErr error
	content io.ReadCloser
	gotNode *git.GetTreeNodeParams
	gotBlob *git.GetBlobParams
}

func (g *infGit) GetTreeNode(_ context.Context, p *git.GetTreeNodeParams) (*git.GetTreeNodeOutput, error) {
	g.gotNode = p
	if g.nodeErr != nil {
		return nil, g.nodeErr
	}
	return &git.GetTreeNodeOutput{Node: g.node}, nil
}

func (g *infGit) GetBlob(_ context.Context, p *git.GetBlobParams) (*git.GetBlobOutput, error) {
	g.gotBlob = p
	if g.blobErr != nil {
		return nil, g.blobErr
	}
	return &git.GetBlobOutput{Content: g.content}, nil
}

type infReadCloser struct {
	io.Reader
	closed *bool
}

func (r infReadCloser) Close() error {
	*r.closed = true
	return errors.New("close failed") // only logged by the service
}

type infErrReader struct{}

func (infErrReader) Read([]byte) (int, error) { return 0, errors.New("read failed") }

func TestInfService_Get(t *testing.T) {
	ctx := context.Background()
	repo := &types.Repository{GitUID: "git-uid"}

	t.Run("reads blob content", func(t *testing.T) {
		closed := false
		g := &infGit{
			node:    git.TreeNode{Type: git.TreeNodeTypeBlob, SHA: "abc"},
			content: infReadCloser{Reader: strings.NewReader("pipeline: yaml"), closed: &closed},
		}
		f, err := newService(g).Get(ctx, repo, ".harness/ci.yaml", "refs/heads/main")
		require.NoError(t, err)
		require.Equal(t, "pipeline: yaml", string(f.Data))
		require.True(t, closed)
		require.Equal(t, "git-uid", g.gotNode.RepoUID)
		require.Equal(t, "refs/heads/main", g.gotNode.GitREF)
		require.Equal(t, ".harness/ci.yaml", g.gotNode.Path)
		require.Equal(t, "abc", g.gotBlob.SHA)
	})
	t.Run("tree node error", func(t *testing.T) {
		g := &infGit{nodeErr: errors.New("no such path")}
		_, err := newService(g).Get(ctx, repo, "p", "ref")
		require.ErrorContains(t, err, "failed to read tree node")
	})
	t.Run("not a blob", func(t *testing.T) {
		g := &infGit{node: git.TreeNode{Type: git.TreeNodeTypeTree}}
		_, err := newService(g).Get(ctx, repo, "dir", "ref")
		require.ErrorContains(t, err, "path content is not of blob type")
	})
	t.Run("blob error", func(t *testing.T) {
		g := &infGit{node: git.TreeNode{Type: git.TreeNodeTypeBlob}, blobErr: errors.New("boom")}
		_, err := newService(g).Get(ctx, repo, "p", "ref")
		require.ErrorContains(t, err, "failed to read blob")
	})
	t.Run("read error", func(t *testing.T) {
		g := &infGit{
			node:    git.TreeNode{Type: git.TreeNodeTypeBlob},
			content: infReadCloser{Reader: infErrReader{}, closed: new(bool)},
		}
		_, err := newService(g).Get(ctx, repo, "p", "ref")
		require.ErrorContains(t, err, "could not read blob content from file")
	})
}
