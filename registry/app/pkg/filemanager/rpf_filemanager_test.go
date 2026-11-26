//  Copyright 2023 Harness, Inc.
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

package filemanager

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/registry/app/events/replication"
	"github.com/harness/gitness/registry/app/store"
	"github.com/harness/gitness/registry/types"
	gitnesstypes "github.com/harness/gitness/types"

	"github.com/stretchr/testify/require"
)

type rpfNodes struct {
	store.NodesRepository
	created       []*types.Node
	createErr     error
	createErrAt   int
	byPath        *types.Node
	byPathErr     error
	byBlob        *types.Node
	byBlobErr     error
	latest        *types.Node
	latestErr     error
	prefixNodes   []types.Node
	prefixErr     error
	deleteErr     error
	deletedPaths  []string
	deletedLeaves []string
	count         int64
	countErr      error
	files         []types.FileNodeMetadata
	filesErr      error
}

func (r *rpfNodes) Create(_ context.Context, node *types.Node) error {
	if r.createErr != nil && len(r.created) == r.createErrAt {
		return r.createErr
	}
	node.ID = "node-" + node.Name
	r.created = append(r.created, node)
	return nil
}

func (r *rpfNodes) GetByPathAndRegistryID(context.Context, int64, string) (*types.Node, error) {
	return r.byPath, r.byPathErr
}

func (r *rpfNodes) GetByBlobIDAndRegistryID(context.Context, string, int64) (*types.Node, error) {
	return r.byBlob, r.byBlobErr
}

func (r *rpfNodes) FindByPathAndRegistryID(context.Context, int64, string, string) (*types.Node, error) {
	return r.latest, r.latestErr
}

func (r *rpfNodes) GetAllFileNodesByPathPrefixAndRegistryID(
	context.Context, int64, string,
) (*[]types.Node, error) {
	if r.prefixErr != nil {
		return nil, r.prefixErr
	}
	return &r.prefixNodes, nil
}

func (r *rpfNodes) DeleteByNodePathAndRegistryID(_ context.Context, path string, _ int64) error {
	r.deletedPaths = append(r.deletedPaths, path)
	return r.deleteErr
}

func (r *rpfNodes) DeleteByLeafNodePathAndRegistryID(_ context.Context, path string, _ int64) error {
	r.deletedLeaves = append(r.deletedLeaves, path)
	return r.deleteErr
}

func (r *rpfNodes) CountByPathAndRegistryID(context.Context, int64, string) (int64, error) {
	return r.count, r.countErr
}

func (r *rpfNodes) GetFilesMetadataByPathAndRegistryID(
	context.Context, int64, string, string, string, int, int, string,
) (*[]types.FileNodeMetadata, error) {
	if r.filesErr != nil {
		return nil, r.filesErr
	}
	return &r.files, nil
}

type rpfBlobs struct {
	store.GenericBlobRepository
	blob       *types.GenericBlob
	findErr    error
	bySha      *types.GenericBlob
	byShaErr   error
	createID   string
	createdNew bool
	createErr  error
	gotCreate  *types.GenericBlob
}

func (r *rpfBlobs) FindByID(context.Context, string) (*types.GenericBlob, error) {
	return r.blob, r.findErr
}

func (r *rpfBlobs) FindBySha256AndRootParentID(context.Context, string, int64) (*types.GenericBlob, error) {
	return r.bySha, r.byShaErr
}

func (r *rpfBlobs) Create(_ context.Context, gb *types.GenericBlob) (string, bool, error) {
	r.gotCreate = gb
	return r.createID, r.createdNew, r.createErr
}

type rpfTx struct{ err error }

func (t rpfTx) WithTx(ctx context.Context, fn func(ctx context.Context) error, _ ...any) error {
	if t.err != nil {
		return t.err
	}
	return fn(ctx)
}

type rpfReplication struct {
	events []string
}

func (r *rpfReplication) ReportEventAsync(
	_ context.Context, _ string, _ replication.BlobAction, _ int64, genericBlobID string, _ string,
	_ *gitnesstypes.Config, _ []replication.CloudLocation,
) {
	r.events = append(r.events, genericBlobID)
}

func rpfManager(nodes *rpfNodes, blobs *rpfBlobs) (*fileManager, *rpfReplication) {
	reporter := &rpfReplication{}
	return &fileManager{
		nodesDao:            nodes,
		genericBlobDao:      blobs,
		tx:                  rpfTx{},
		replicationReporter: reporter,
		config:              &gitnesstypes.Config{},
	}, reporter
}

func TestRpfCreateNodesBuildsTheHierarchy(t *testing.T) {
	ctx := context.Background()
	nodes := &rpfNodes{}
	fm, _ := rpfManager(nodes, &rpfBlobs{})

	require.NoError(t, fm.createNodes(ctx, "/dir//sub/file.txt", "blob-1", 3, 9))
	require.Len(t, nodes.created, 3)
	require.Equal(t, "dir", nodes.created[0].Name)
	require.Equal(t, "/dir", nodes.created[0].NodePath)
	require.Empty(t, nodes.created[0].ParentNodeID)
	require.False(t, nodes.created[0].IsFile)
	require.Equal(t, "node-dir", nodes.created[1].ParentNodeID)
	require.Equal(t, "/dir/sub", nodes.created[1].NodePath)
	require.Equal(t, "node-sub", nodes.created[2].ParentNodeID)
	require.Equal(t, "/dir/sub/file.txt", nodes.created[2].NodePath)
	require.True(t, nodes.created[2].IsFile)
	require.Equal(t, "blob-1", nodes.created[2].BlobID)
	require.Equal(t, int64(3), nodes.created[2].RegistryID)
	require.Equal(t, int64(9), nodes.created[2].CreatedBy)

	failing := &rpfNodes{createErr: errors.New("db down"), createErrAt: 1}
	fm, _ = rpfManager(failing, &rpfBlobs{})
	err := fm.createNodes(ctx, "/dir/sub/file.txt", "blob-1", 3, 9)
	require.ErrorContains(t, err, "failed to create the node: sub")
}

func TestRpfCreateNodesWithoutFileNode(t *testing.T) {
	ctx := context.Background()
	nodes := &rpfNodes{}
	fm, _ := rpfManager(nodes, &rpfBlobs{})
	require.NoError(t, fm.CreateNodesWithoutFileNode(ctx, "/a/b", 3, 9))
	require.Len(t, nodes.created, 2)
	for _, n := range nodes.created {
		require.False(t, n.IsFile)
		require.Empty(t, n.BlobID)
	}

	fm, _ = rpfManager(&rpfNodes{createErr: errors.New("x")}, &rpfBlobs{})
	require.Error(t, fm.CreateNodesWithoutFileNode(ctx, "/a/b", 3, 9))
}

func TestRpfSaveNodesAndNode(t *testing.T) {
	ctx := context.Background()
	nodes := &rpfNodes{}
	fm, _ := rpfManager(nodes, &rpfBlobs{bySha: &types.GenericBlob{ID: "blob-7"}})
	require.NoError(t, fm.SaveNodes(ctx, "/a/file", 3, 2, 9, "sha"))
	require.Equal(t, "blob-7", nodes.created[1].BlobID)

	fm, _ = rpfManager(&rpfNodes{}, &rpfBlobs{byShaErr: errors.New("missing")})
	require.ErrorContains(t, fm.SaveNodes(ctx, "/a/file", 3, 2, 9, "sha"), "failed to find generic blob")

	fm, _ = rpfManager(&rpfNodes{createErr: errors.New("x")}, &rpfBlobs{bySha: &types.GenericBlob{ID: "b"}})
	require.ErrorContains(t, fm.SaveNodes(ctx, "/a/file", 3, 2, 9, "sha"), "failed to save nodes")

	id, err := fm.SaveNode(ctx, "/a", "", 3, "a", "", "/a", false, 9)
	require.Error(t, err)
	require.Empty(t, id)
	fm, _ = rpfManager(&rpfNodes{}, &rpfBlobs{})
	id, err = fm.SaveNode(ctx, "/a", "", 3, "a", "", "/a", false, 9)
	require.NoError(t, err)
	require.Equal(t, "node-a", id)
}

func TestRpfCopyNodes(t *testing.T) {
	ctx := context.Background()
	source := []types.Node{{BlobID: "b1", NodePath: "/p/one", CreatedBy: 4}, {BlobID: "b2", NodePath: "/p/two"}}

	nodes := &rpfNodes{prefixNodes: source}
	fm, _ := rpfManager(nodes, &rpfBlobs{blob: &types.GenericBlob{Sha256: "sha"}, bySha: &types.GenericBlob{ID: "b"}})
	require.NoError(t, fm.CopyNodes(ctx, 2, 3, 4, []string{"/p"}))
	require.Len(t, nodes.created, 4)

	fm, _ = rpfManager(&rpfNodes{prefixErr: errors.New("x")}, &rpfBlobs{})
	require.ErrorContains(t, fm.CopyNodes(ctx, 2, 3, 4, []string{"/p"}), "failed to get nodes from source registry")

	fm, _ = rpfManager(&rpfNodes{}, &rpfBlobs{})
	require.ErrorContains(t, fm.CopyNodes(ctx, 2, 3, 4, []string{"/p"}), "failed to get nodes from source registry")

	fm, _ = rpfManager(&rpfNodes{prefixNodes: source}, &rpfBlobs{findErr: errors.New("x")})
	require.ErrorContains(t, fm.CopyNodes(ctx, 2, 3, 4, []string{"/p"}), "failed to get blob")

	fm, _ = rpfManager(&rpfNodes{prefixNodes: source},
		&rpfBlobs{blob: &types.GenericBlob{Sha256: "sha"}, byShaErr: errors.New("x")})
	require.ErrorContains(t, fm.CopyNodes(ctx, 2, 3, 4, []string{"/p"}), "failed to save nodes")
}

func TestRpfDeletes(t *testing.T) {
	ctx := context.Background()
	nodes := &rpfNodes{}
	fm, _ := rpfManager(nodes, &rpfBlobs{})

	require.NoError(t, fm.DeleteFile(ctx, 3, "/a/b"))
	require.NoError(t, fm.DeleteLeafNode(ctx, 3, "a/b"))
	require.NoError(t, fm.DeleteLeafNode(ctx, 3, "/c"))
	require.NoError(t, fm.DeleteLeafNode(ctx, 3, ""))
	require.Equal(t, []string{"/a/b"}, nodes.deletedPaths)
	require.Equal(t, []string{"/a/b", "/c", ""}, nodes.deletedLeaves)

	failing, _ := rpfManager(&rpfNodes{deleteErr: errors.New("x")}, &rpfBlobs{})
	require.ErrorContains(t, failing.DeleteFile(ctx, 3, "/a"), "failed to delete file")
	require.ErrorContains(t, failing.DeleteLeafNode(ctx, 3, "/a"), "failed to delete file")
}

func TestRpfLookups(t *testing.T) {
	ctx := context.Background()
	node := &types.Node{Name: "file.txt", BlobID: "b1", NodePath: "/a/file.txt"}
	blob := &types.GenericBlob{ID: "b1", Sha256: "sha256-value", Sha1: "sha1-value", Size: 12}

	fm, _ := rpfManager(&rpfNodes{byPath: node, byBlob: node, latest: node}, &rpfBlobs{blob: blob, bySha: blob})

	got, err := fm.GetNode(ctx, 3, "/a/file.txt")
	require.NoError(t, err)
	require.Same(t, node, got)

	sha, size, err := fm.HeadFile(ctx, "/a/file.txt", 3)
	require.NoError(t, err)
	require.Equal(t, "sha256-value", sha)
	require.Equal(t, int64(12), size)

	path, err := fm.GetFilePath(ctx, "sha256-value", 3, 2)
	require.NoError(t, err)
	require.Equal(t, "/a/file.txt", path)

	path, err = fm.FindLatestFilePath(ctx, 3, "/a", "file.txt")
	require.NoError(t, err)
	require.Equal(t, "/a/file.txt", path)

	info, err := fm.GetFileMetadata(ctx, 3, "/a/file.txt")
	require.NoError(t, err)
	require.Equal(t, types.FileInfo{
		Sha1: "sha1-value", Sha256: "sha256-value", Size: 12, Filename: "file.txt",
	}, info)

	missingNode, _ := rpfManager(
		&rpfNodes{
			byPathErr: errors.New("x"), byBlobErr: errors.New("x"), latestErr: errors.New("x"),
		}, &rpfBlobs{blob: blob, bySha: blob})
	_, err = missingNode.GetNode(ctx, 3, "/p")
	require.Error(t, err)
	_, _, err = missingNode.HeadFile(ctx, "/p", 3)
	require.ErrorContains(t, err, "failed to get the node path mapping")
	_, err = missingNode.GetFilePath(ctx, "sha", 3, 2)
	require.ErrorContains(t, err, "failed to get the node for blob id")
	_, err = missingNode.FindLatestFilePath(ctx, 3, "/a", "f")
	require.ErrorContains(t, err, "failed to get the node for path")
	_, err = missingNode.GetFileMetadata(ctx, 3, "/p")
	require.ErrorContains(t, err, "failed to get the node path mapping")

	missingBlob, _ := rpfManager(&rpfNodes{byPath: node, byBlob: node},
		&rpfBlobs{findErr: errors.New("x"), byShaErr: errors.New("x")})
	_, _, err = missingBlob.HeadFile(ctx, "/p", 3)
	require.ErrorContains(t, err, "failed to get the blob")
	_, err = missingBlob.GetFilePath(ctx, "sha", 3, 2)
	require.ErrorContains(t, err, "failed to get the blob for sha256")
	_, err = missingBlob.GetFileMetadata(ctx, 3, "/p")
	require.ErrorContains(t, err, "failed to get the blob")

	// A missing blob without an error is treated as a failure too.
	noBlob, _ := rpfManager(&rpfNodes{}, &rpfBlobs{})
	_, err = noBlob.GetFilePath(ctx, "sha", 3, 2)
	require.Error(t, err)
}

func TestRpfFilesMetadataAndCount(t *testing.T) {
	ctx := context.Background()
	fm, _ := rpfManager(&rpfNodes{files: []types.FileNodeMetadata{{Name: "a"}}, count: 5}, &rpfBlobs{})
	files, err := fm.GetFilesMetadata(ctx, "/a", 3, "name", "ASC", 10, 0, "")
	require.NoError(t, err)
	require.Len(t, *files, 1)
	count, err := fm.CountFilesByPath(ctx, "/a", 3)
	require.NoError(t, err)
	require.Equal(t, int64(5), count)

	failing, _ := rpfManager(&rpfNodes{filesErr: errors.New("x"), countErr: errors.New("x")}, &rpfBlobs{})
	files, err = failing.GetFilesMetadata(ctx, "/a", 3, "name", "ASC", 10, 0, "")
	require.ErrorContains(t, err, "failed to get the files")
	require.Empty(t, *files)
	count, err = failing.CountFilesByPath(ctx, "/a", 3)
	require.ErrorContains(t, err, "failed to get the count of files")
	require.Equal(t, int64(-1), count)
}

func TestRpfDBSaveFile(t *testing.T) {
	ctx := context.Background()
	info := types.FileInfo{Sha256: "sha", Size: 4, Filename: "f.txt"}

	t.Run("saves the blob and nodes then runs the commit callback", func(t *testing.T) {
		nodes := &rpfNodes{}
		blobs := &rpfBlobs{createID: "blob-1", createdNew: true}
		fm, _ := rpfManager(nodes, blobs)
		called := false
		id, created, err := fm.dbSaveFile(ctx, "/a/f.txt", 3, 2, info, 9, func(context.Context) error {
			called = true
			return nil
		})
		require.NoError(t, err)
		require.Equal(t, "blob-1", id)
		require.True(t, created)
		require.True(t, called)
		require.Equal(t, "sha", blobs.gotCreate.Sha256)
		require.Equal(t, int64(2), blobs.gotCreate.RootParentID)
		require.Len(t, nodes.created, 2)
	})

	t.Run("failures", func(t *testing.T) {
		fm, _ := rpfManager(&rpfNodes{}, &rpfBlobs{createErr: errors.New("x")})
		_, _, err := fm.dbSaveFile(ctx, "/a/f.txt", 3, 2, info, 9, nil)
		require.ErrorContains(t, err, "failed to save generic blob")

		fm, _ = rpfManager(&rpfNodes{createErr: errors.New("x")}, &rpfBlobs{})
		_, _, err = fm.dbSaveFile(ctx, "/a/f.txt", 3, 2, info, 9, nil)
		require.Error(t, err)

		fm, _ = rpfManager(&rpfNodes{}, &rpfBlobs{})
		_, _, err = fm.dbSaveFile(ctx, "/a/f.txt", 3, 2, info, 9, func(context.Context) error {
			return errors.New("callback failed")
		})
		require.ErrorContains(t, err, "failed to execute commit callback")

		fm, _ = rpfManager(&rpfNodes{}, &rpfBlobs{})
		fm.tx = rpfTx{err: errors.New("tx failed")}
		_, _, err = fm.dbSaveFile(ctx, "/a/f.txt", 3, 2, info, 9, nil)
		require.ErrorContains(t, err, "tx failed")
	})
}

func TestRpfPostFileUpload(t *testing.T) {
	ctx := context.Background()
	info := types.FileInfo{Sha256: "sha", Filename: "f.txt"}

	fm, reporter := rpfManager(&rpfNodes{}, &rpfBlobs{createID: "blob-1", createdNew: true})
	require.NoError(t, fm.PostFileUpload(ctx, "/a/f.txt", 3, 2, "root", info, 9))
	require.Equal(t, []string{"blob-1"}, reporter.events)

	fm, reporter = rpfManager(&rpfNodes{}, &rpfBlobs{createID: "blob-1"})
	require.NoError(t, fm.PostFileUpload(ctx, "/a/f.txt", 3, 2, "root", info, 9))
	require.Empty(t, reporter.events)

	fm, _ = rpfManager(&rpfNodes{}, &rpfBlobs{createErr: errors.New("x")})
	require.Error(t, fm.PostFileUpload(ctx, "/a/f.txt", 3, 2, "root", info, 9))
}

func TestRpfDownloadFileByPathFailures(t *testing.T) {
	ctx := context.Background()
	notFound := func(err error) {
		t.Helper()
		var userErr *usererror.Error
		require.ErrorAs(t, err, &userErr)
		require.Equal(t, http.StatusNotFound, userErr.Status)
	}

	fm, _ := rpfManager(&rpfNodes{byPathErr: errors.New("x")}, &rpfBlobs{})
	_, _, _, err := fm.DownloadFileByPath(ctx, "/a", 3, "reg", "root", true)
	notFound(err)

	fm, _ = rpfManager(&rpfNodes{byPath: &types.Node{BlobID: "b"}}, &rpfBlobs{findErr: errors.New("x")})
	_, _, _, err = fm.DownloadFileByPath(ctx, "/a", 3, "reg", "root", true)
	notFound(err)

	fm, _ = rpfManager(&rpfNodes{byPath: &types.Node{BlobID: "b"}},
		&rpfBlobs{blob: &types.GenericBlob{ID: "not-a-uuid"}})
	_, _, _, err = fm.DownloadFileByPath(ctx, "/a", 3, "reg", "root", false)
	notFound(err)
}
