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

package repo

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/harness/gitness/app/services/settings"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/git"
	gitnessstore "github.com/harness/gitness/store"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// crpSettingsStore is a store.SettingsStore serving raw setting values by key.
type crpSettingsStore struct {
	store.SettingsStore
	values    map[string]json.RawMessage
	err       error
	upsertErr error
}

func (s crpSettingsStore) Upsert(
	_ context.Context, _ enum.SettingsScope, _ int64, key string, value json.RawMessage,
) error {
	if s.upsertErr != nil {
		return s.upsertErr
	}

	s.values[key] = value

	return nil
}

func (s crpSettingsStore) Find(
	_ context.Context, _ enum.SettingsScope, _ int64, key string,
) (json.RawMessage, error) {
	if s.err != nil {
		return nil, s.err
	}

	value, ok := s.values[key]
	if !ok {
		return nil, gitnessstore.ErrResourceNotFound
	}

	return value, nil
}

func crpBlob(content string) *git.GetBlobOutput {
	return &git.GetBlobOutput{
		SHA:         crpTestSHA,
		Size:        int64(len(content)),
		ContentSize: int64(len(content)),
		Content:     io.NopCloser(strings.NewReader(content)),
	}
}

func crpTreeNode(mode git.TreeNodeMode, typ git.TreeNodeType, name string) git.TreeNode {
	return git.TreeNode{Type: typ, Mode: mode, SHA: crpTestSHA.String(), Name: name, Path: "dir/" + name}
}

func TestCrpMapNodeModeToContentType(t *testing.T) {
	tests := []struct {
		mode    git.TreeNodeMode
		want    ContentType
		wantErr bool
	}{
		{mode: git.TreeNodeModeFile, want: ContentTypeFile},
		{mode: git.TreeNodeModeExec, want: ContentTypeFile},
		{mode: git.TreeNodeModeSymlink, want: ContentTypeSymlink},
		{mode: git.TreeNodeModeCommit, want: ContentTypeSubmodule},
		{mode: git.TreeNodeModeTree, want: ContentTypeDir},
		{mode: git.TreeNodeMode("weird"), wantErr: true},
	}

	for _, tt := range tests {
		t.Run(string(tt.mode), func(t *testing.T) {
			got, err := mapNodeModeToContentType(tt.mode)

			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCrpGetContent(t *testing.T) {
	ctx := context.Background()
	commit := &git.Commit{SHA: crpTestSHA, Title: "last"}

	t.Run("file content is base64 encoded", func(t *testing.T) {
		e := crpNewEnv(t)
		e.git.On("GetTreeNode", mock.Anything, mock.MatchedBy(func(p *git.GetTreeNodeParams) bool {
			return p.GitREF == crpDefaultBranch && p.Path == "dir/a.txt" && p.IncludeLatestCommit
		})).Return(&git.GetTreeNodeOutput{
			Node:   crpTreeNode(git.TreeNodeModeFile, git.TreeNodeTypeBlob, "a.txt"),
			Commit: commit,
		}, nil)
		e.git.On("GetBlob", mock.Anything, mock.Anything).Return(crpBlob("hello"), nil)

		out, err := e.c.GetContent(ctx, crpSession(), crpRepoRef, "", "dir/a.txt", true, false)

		require.NoError(t, err)
		assert.Equal(t, ContentTypeFile, out.Type)
		require.NotNil(t, out.LatestCommit)
		assert.Equal(t, "last", out.LatestCommit.Title)
		fileContent, ok := out.Content.(*FileContent)
		require.True(t, ok)
		assert.Equal(t, base64.StdEncoding.EncodeToString([]byte("hello")), fileContent.Data)
		assert.EqualValues(t, 5, fileContent.Size)
		assert.Empty(t, fileContent.LFSObjectID)
	})

	t.Run("lfs pointers are flagged", func(t *testing.T) {
		e := crpNewEnv(t)
		pointer := "version https://git-lfs.github.com/spec/v1\n" +
			"oid sha256:4d7a214614ab2935c943f9e0ff69d22eadbb8f32b1258daaa5e2ca24d17e2393\nsize 12345\n"
		e.git.On("GetTreeNode", mock.Anything, mock.Anything).Return(&git.GetTreeNodeOutput{
			Node: crpTreeNode(git.TreeNodeModeFile, git.TreeNodeTypeBlob, "big.bin"),
		}, nil)
		e.git.On("GetBlob", mock.Anything, mock.Anything).Return(crpBlob(pointer), nil)

		out, err := e.c.GetContent(ctx, crpSession(), crpRepoRef, crpFeatureBranch, "big.bin", false, false)

		require.NoError(t, err)
		fileContent, ok := out.Content.(*FileContent)
		require.True(t, ok)
		assert.Equal(t, "4d7a214614ab2935c943f9e0ff69d22eadbb8f32b1258daaa5e2ca24d17e2393", fileContent.LFSObjectID)
		assert.EqualValues(t, 12345, fileContent.LFSObjectSize)
		assert.Nil(t, out.LatestCommit)
	})

	t.Run("directory entries are listed", func(t *testing.T) {
		e := crpNewEnv(t)
		e.git.On("GetTreeNode", mock.Anything, mock.Anything).Return(&git.GetTreeNodeOutput{
			Node: crpTreeNode(git.TreeNodeModeTree, git.TreeNodeTypeTree, "dir"),
		}, nil)
		e.git.On("ListTreeNodes", mock.Anything, mock.MatchedBy(func(p *git.ListTreeNodeParams) bool {
			return p.FlattenDirectories && p.Path == "dir"
		})).Return(&git.ListTreeNodeOutput{Nodes: []git.TreeNode{
			crpTreeNode(git.TreeNodeModeFile, git.TreeNodeTypeBlob, "a.txt"),
			crpTreeNode(git.TreeNodeModeTree, git.TreeNodeTypeTree, "sub"),
		}}, nil)

		out, err := e.c.GetContent(ctx, crpSession(), crpRepoRef, "", "dir", false, true)

		require.NoError(t, err)
		dirContent, ok := out.Content.(*DirContent)
		require.True(t, ok)
		require.Len(t, dirContent.Entries, 2)
		assert.Equal(t, ContentTypeFile, dirContent.Entries[0].Type)
		assert.Equal(t, ContentTypeDir, dirContent.Entries[1].Type)
	})

	t.Run("directory entry with an unsupported mode", func(t *testing.T) {
		e := crpNewEnv(t)
		e.git.On("GetTreeNode", mock.Anything, mock.Anything).Return(&git.GetTreeNodeOutput{
			Node: crpTreeNode(git.TreeNodeModeTree, git.TreeNodeTypeTree, "dir"),
		}, nil)
		e.git.On("ListTreeNodes", mock.Anything, mock.Anything).Return(&git.ListTreeNodeOutput{Nodes: []git.TreeNode{
			crpTreeNode(git.TreeNodeMode("weird"), git.TreeNodeTypeBlob, "x"),
		}}, nil)

		_, err := e.c.GetContent(ctx, crpSession(), crpRepoRef, "", "dir", false, false)

		require.Error(t, err)
	})

	t.Run("symlink target is returned", func(t *testing.T) {
		e := crpNewEnv(t)
		e.git.On("GetTreeNode", mock.Anything, mock.Anything).Return(&git.GetTreeNodeOutput{
			Node: crpTreeNode(git.TreeNodeModeSymlink, git.TreeNodeTypeBlob, "link"),
		}, nil)
		e.git.On("GetBlob", mock.Anything, mock.Anything).Return(crpBlob("target.txt"), nil)

		out, err := e.c.GetContent(ctx, crpSession(), crpRepoRef, "", "link", false, false)

		require.NoError(t, err)
		link, ok := out.Content.(*SymlinkContent)
		require.True(t, ok)
		assert.Equal(t, "target.txt", link.Target)
	})

	t.Run("submodule url is returned", func(t *testing.T) {
		e := crpNewEnv(t)
		e.git.On("GetTreeNode", mock.Anything, mock.Anything).Return(&git.GetTreeNodeOutput{
			Node: crpTreeNode(git.TreeNodeModeCommit, git.TreeNodeTypeCommit, "mod"),
		}, nil)
		e.git.On("GetSubmodule", mock.Anything, mock.Anything).Return(&git.GetSubmoduleOutput{
			Submodule: git.Submodule{Name: "mod", URL: "https://example.com/mod.git"},
		}, nil)

		out, err := e.c.GetContent(ctx, crpSession(), crpRepoRef, "", "mod", false, false)

		require.NoError(t, err)
		sub, ok := out.Content.(*SubmoduleContent)
		require.True(t, ok)
		assert.Equal(t, "https://example.com/mod.git", sub.URL)
		assert.Equal(t, crpTestSHA.String(), sub.CommitSHA)
	})

	failures := []struct {
		name  string
		mode  git.TreeNodeMode
		typ   git.TreeNodeType
		setup func(e *crpEnv)
	}{
		{name: "tree node failure", setup: func(e *crpEnv) {
			e.git.On("GetTreeNode", mock.Anything, mock.Anything).Return(nil, errCrpBoom)
		}},
		{name: "unsupported node mode", mode: "weird", typ: git.TreeNodeTypeBlob},
		{name: "file blob failure", mode: git.TreeNodeModeFile, typ: git.TreeNodeTypeBlob, setup: func(e *crpEnv) {
			e.git.On("GetBlob", mock.Anything, mock.Anything).Return(nil, errCrpBoom)
		}},
		{name: "symlink blob failure", mode: git.TreeNodeModeSymlink, typ: git.TreeNodeTypeBlob, setup: func(e *crpEnv) {
			e.git.On("GetBlob", mock.Anything, mock.Anything).Return(nil, errCrpBoom)
		}},
		{name: "dir failure", mode: git.TreeNodeModeTree, typ: git.TreeNodeTypeTree, setup: func(e *crpEnv) {
			e.git.On("ListTreeNodes", mock.Anything, mock.Anything).Return(nil, errCrpBoom)
		}},
		{name: "submodule failure", mode: git.TreeNodeModeCommit, typ: git.TreeNodeTypeCommit, setup: func(e *crpEnv) {
			e.git.On("GetSubmodule", mock.Anything, mock.Anything).Return(nil, errCrpBoom)
		}},
	}
	for _, tt := range failures {
		t.Run(tt.name, func(t *testing.T) {
			e := crpNewEnv(t)
			if tt.mode != "" {
				e.git.On("GetTreeNode", mock.Anything, mock.Anything).Return(&git.GetTreeNodeOutput{
					Node: crpTreeNode(tt.mode, tt.typ, "x"),
				}, nil)
			}
			if tt.setup != nil {
				tt.setup(e)
			}

			_, err := e.c.GetContent(ctx, crpSession(), crpRepoRef, "", "x", false, false)

			require.Error(t, err)
		})
	}
}

func TestCrpPathsDetails(t *testing.T) {
	ctx := context.Background()

	t.Run("no paths", func(t *testing.T) {
		e := crpNewEnv(t)

		out, err := e.c.PathsDetails(ctx, crpSession(), crpRepoRef, "", PathsDetailsInput{})

		require.NoError(t, err)
		assert.Empty(t, out.Details)
	})

	t.Run("too many paths", func(t *testing.T) {
		e := crpNewEnv(t)

		_, err := e.c.PathsDetails(ctx, crpSession(), crpRepoRef, "", PathsDetailsInput{Paths: make([]string, 51)})

		require.Error(t, err)
		assert.Equal(t, http.StatusBadRequest, crpUserErrStatus(t, err))
	})

	t.Run("maps last commits", func(t *testing.T) {
		e := crpNewEnv(t)
		e.git.On("PathsDetails", mock.Anything, mock.MatchedBy(func(p git.PathsDetailsParams) bool {
			return p.GitREF == crpDefaultBranch && len(p.Paths) == 2
		})).Return(git.PathsDetailsOutput{Details: []git.PathDetails{
			{Path: "a", LastCommit: &git.Commit{SHA: crpTestSHA, Title: "a commit"}},
			{Path: "b"},
		}}, nil)

		out, err := e.c.PathsDetails(ctx, crpSession(), crpRepoRef, "", PathsDetailsInput{Paths: []string{"a", "b"}})

		require.NoError(t, err)
		require.Len(t, out.Details, 2)
		require.NotNil(t, out.Details[0].LastCommit)
		assert.Equal(t, "a commit", out.Details[0].LastCommit.Title)
		assert.Nil(t, out.Details[1].LastCommit)
	})

	t.Run("git failure", func(t *testing.T) {
		e := crpNewEnv(t)
		e.git.On("PathsDetails", mock.Anything, mock.Anything).Return(git.PathsDetailsOutput{}, errCrpBoom)

		_, err := e.c.PathsDetails(ctx, crpSession(), crpRepoRef, crpFeatureBranch, PathsDetailsInput{Paths: []string{"a"}})

		require.ErrorIs(t, err, errCrpBoom)
	})
}

func TestCrpRaw(t *testing.T) {
	ctx := context.Background()
	blobNode := &git.GetTreeNodeOutput{Node: crpTreeNode(git.TreeNodeModeFile, git.TreeNodeTypeBlob, "a.txt")}

	newEnv := func(t *testing.T, lfs *bool) *crpEnv {
		t.Helper()
		e := crpNewEnv(t)
		values := map[string]json.RawMessage{}
		if lfs != nil {
			raw, err := json.Marshal(*lfs)
			require.NoError(t, err)
			values[string(settings.KeyGitLFSEnabled)] = raw
		}
		e.c.settings = settings.NewService(crpSettingsStore{values: values}, e.c.spaceFinder)

		return e
	}

	t.Run("returns the blob stream with lfs disabled", func(t *testing.T) {
		disabled := false
		e := newEnv(t, &disabled)
		e.git.On("GetTreeNode", mock.Anything, mock.MatchedBy(func(p *git.GetTreeNodeParams) bool {
			return p.GitREF == crpDefaultBranch && !p.IncludeLatestCommit
		})).Return(blobNode, nil)
		e.git.On("GetBlob", mock.Anything, mock.MatchedBy(func(p *git.GetBlobParams) bool {
			return p.SizeLimit == 0
		})).Return(crpBlob("raw data"), nil)

		out, err := e.c.Raw(ctx, crpSession(), crpRepoRef, "", "a.txt")

		require.NoError(t, err)
		data, err := io.ReadAll(out.Data)
		require.NoError(t, err)
		assert.Equal(t, "raw data", string(data))
		assert.EqualValues(t, 8, out.Size)
		assert.Equal(t, crpTestSHA, out.SHA)
	})

	t.Run("non pointer content is passed through when lfs is enabled", func(t *testing.T) {
		e := newEnv(t, nil)
		e.git.On("GetTreeNode", mock.Anything, mock.Anything).Return(blobNode, nil)
		e.git.On("GetBlob", mock.Anything, mock.Anything).Return(crpBlob("plain file content"), nil)

		out, err := e.c.Raw(ctx, crpSession(), crpRepoRef, crpFeatureBranch, "a.txt")

		require.NoError(t, err)
		data, err := io.ReadAll(out.Data)
		require.NoError(t, err)
		assert.Equal(t, "plain file content", string(data))
		require.NoError(t, out.Data.Close())
	})

	t.Run("only blobs can be viewed raw", func(t *testing.T) {
		e := newEnv(t, nil)
		e.git.On("GetTreeNode", mock.Anything, mock.Anything).Return(&git.GetTreeNodeOutput{
			Node: crpTreeNode(git.TreeNodeModeTree, git.TreeNodeTypeTree, "dir"),
		}, nil)

		_, err := e.c.Raw(ctx, crpSession(), crpRepoRef, "", "dir")

		require.Error(t, err)
		assert.Equal(t, http.StatusBadRequest, crpUserErrStatus(t, err))
	})

	t.Run("tree node failure", func(t *testing.T) {
		e := newEnv(t, nil)
		e.git.On("GetTreeNode", mock.Anything, mock.Anything).Return(nil, errCrpBoom)

		_, err := e.c.Raw(ctx, crpSession(), crpRepoRef, "", "a.txt")

		require.ErrorIs(t, err, errCrpBoom)
	})

	t.Run("blob failure", func(t *testing.T) {
		e := newEnv(t, nil)
		e.git.On("GetTreeNode", mock.Anything, mock.Anything).Return(blobNode, nil)
		e.git.On("GetBlob", mock.Anything, mock.Anything).Return(nil, errCrpBoom)

		_, err := e.c.Raw(ctx, crpSession(), crpRepoRef, "", "a.txt")

		require.ErrorIs(t, err, errCrpBoom)
	})

	t.Run("settings failure", func(t *testing.T) {
		e := newEnv(t, nil)
		e.c.settings = settings.NewService(crpSettingsStore{err: errCrpBoom}, e.c.spaceFinder)
		e.git.On("GetTreeNode", mock.Anything, mock.Anything).Return(blobNode, nil)
		e.git.On("GetBlob", mock.Anything, mock.Anything).Return(crpBlob("x"), nil)

		_, err := e.c.Raw(ctx, crpSession(), crpRepoRef, "", "a.txt")

		require.ErrorIs(t, err, errCrpBoom)
	})
}
