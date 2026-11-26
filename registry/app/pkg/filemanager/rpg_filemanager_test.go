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
	"io"
	"strings"
	"sync"
	"testing"

	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/registry/app/driver/filesystem"
	"github.com/harness/gitness/registry/app/events/replication"
	"github.com/harness/gitness/registry/app/pkg/docker"
	"github.com/harness/gitness/registry/app/services/hook"
	"github.com/harness/gitness/registry/app/storage"
	"github.com/harness/gitness/registry/app/store"
	"github.com/harness/gitness/registry/types"
	gitnesstypes "github.com/harness/gitness/types"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

var errRpgBoom = errors.New("boom")

type rpgNodes struct {
	store.NodesRepository
	created   []*types.Node
	createErr error
	byPath    *types.Node
	byPathErr error
}

func (r *rpgNodes) Create(_ context.Context, node *types.Node) error {
	if r.createErr != nil {
		return r.createErr
	}
	node.ID = "node-" + node.Name
	r.created = append(r.created, node)
	return nil
}

func (r *rpgNodes) GetByPathAndRegistryID(context.Context, int64, string) (*types.Node, error) {
	return r.byPath, r.byPathErr
}

type rpgBlobs struct {
	store.GenericBlobRepository
	blob       *types.GenericBlob
	findErr    error
	createID   string
	createdNew bool
	createErr  error
}

func (r *rpgBlobs) FindByID(context.Context, string) (*types.GenericBlob, error) {
	return r.blob, r.findErr
}

func (r *rpgBlobs) Create(_ context.Context, _ *types.GenericBlob) (string, bool, error) {
	return r.createID, r.createdNew, r.createErr
}

type rpgTx struct{}

func (rpgTx) WithTx(ctx context.Context, fn func(ctx context.Context) error, _ ...any) error {
	return fn(ctx)
}

type rpgReplication struct {
	mu     sync.Mutex
	events []string
}

func (r *rpgReplication) ReportEventAsync(
	_ context.Context, _ string, _ replication.BlobAction, _ int64, genericBlobID string, _ string,
	_ *gitnesstypes.Config, _ []replication.CloudLocation,
) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = append(r.events, genericBlobID)
}

type rpgHook struct {
	commits int
	reads   int
	err     error
}

func (h *rpgHook) OnCommit(context.Context, hook.BlobCommitEvent) error {
	h.commits++
	return h.err
}

func (h *rpgHook) OnRead(context.Context, hook.BlobReadEvent) error {
	h.reads++
	return h.err
}

type rpgBuckets struct {
	store *docker.BlobStore
	calls int
}

func (b *rpgBuckets) GetBlobStore(context.Context, string, string, any, string, string) *docker.BlobStore {
	b.calls++
	return b.store
}

// rpgRedirectStore serves every blob through a redirect.
type rpgRedirectStore struct {
	storage.GenericBlobStore
}

func (rpgRedirectStore) GetGeneric(
	context.Context, int64, string, string, string,
) (*storage.FileReader, string, error) {
	return nil, "https://cdn.example.com/file", nil
}

func (rpgRedirectStore) BucketKey() types.BucketKey { return "bucket" }

type rpgFailingResolver struct{}

func (rpgFailingResolver) Resolve(context.Context, types.StorageLookup) (storage.StorageTarget, error) {
	return storage.StorageTarget{}, errRpgBoom
}

type rpgEnv struct {
	manager *fileManager
	nodes   *rpgNodes
	blobs   *rpgBlobs
	repl    *rpgReplication
	hook    *rpgHook
	buckets *rpgBuckets
}

func rpgNewEnv(t *testing.T) *rpgEnv {
	t.Helper()
	resolver := storage.NewStaticStorageResolver(
		filesystem.New(filesystem.DriverParameters{RootDirectory: t.TempDir(), MaxThreads: 10}),
	)
	service, err := storage.NewStorageService(resolver)
	require.NoError(t, err)

	env := &rpgEnv{
		nodes:   &rpgNodes{},
		blobs:   &rpgBlobs{createID: uuid.NewString(), createdNew: true},
		repl:    &rpgReplication{},
		hook:    &rpgHook{},
		buckets: &rpgBuckets{},
	}
	env.manager = &fileManager{
		config:              &gitnesstypes.Config{},
		storageService:      service,
		genericBlobDao:      env.blobs,
		nodesDao:            env.nodes,
		tx:                  rpgTx{},
		bucketService:       env.buckets,
		replicationReporter: env.repl,
		blobActionHook:      env.hook,
	}
	return env
}

const rpgContent = "file manager content"

func TestRpgUploadAndDownloadFile(t *testing.T) {
	ctx := context.Background()
	env := rpgNewEnv(t)

	info, err := env.manager.UploadFile(ctx, "/dir/file.txt", 5, 1, "root", nil, strings.NewReader(rpgContent), 42)
	require.NoError(t, err)
	require.Equal(t, int64(len(rpgContent)), info.Size)
	require.NotEmpty(t, info.Sha256)
	require.Len(t, env.nodes.created, 2)
	require.Equal(t, 1, env.hook.commits)
	require.Equal(t, []string{env.blobs.createID}, env.repl.events)

	env.nodes.byPath = &types.Node{Name: "file.txt", BlobID: env.blobs.createID}
	env.blobs.blob = &types.GenericBlob{
		ID: env.blobs.createID, Sha256: info.Sha256, Size: info.Size, RootParentID: 1,
	}

	// Download without redirects reads straight from storage.
	reader, size, redirect, err := env.manager.DownloadFileByPath(ctx, "/dir/file.txt", 5, "reg", "root", false)
	require.NoError(t, err)
	require.Empty(t, redirect)
	require.Equal(t, info.Size, size)
	body, err := io.ReadAll(reader)
	require.NoError(t, err)
	require.Equal(t, rpgContent, string(body))
	require.Positive(t, env.buckets.calls)

	// Downloads through the redirect-aware path emit a read event.
	reader, _, _, err = env.manager.DownloadFileByPath(ctx, "/dir/file.txt", 5, "reg", "root", true)
	require.NoError(t, err)
	require.NotNil(t, reader)
	require.Equal(t, 1, env.hook.reads)

	// A bucket service can take over the read.
	env.buckets.store = &docker.BlobStore{GenericStore: rpgRedirectStore{}}
	reader, _, redirect, err = env.manager.DownloadFileByPath(ctx, "/dir/file.txt", 5, "reg", "root", true)
	require.NoError(t, err)
	require.Nil(t, reader)
	require.Equal(t, "https://cdn.example.com/file", redirect)
	require.Equal(t, 2, env.hook.reads)

	// By digest.
	env.buckets.store = nil
	exists, size, err := env.manager.HeadByDigest(ctx, "root", info, 1, 5)
	require.NoError(t, err)
	require.True(t, exists)
	require.Equal(t, info.Size, size)

	reader, err = env.manager.DownloadFileByDigest(ctx, "root", info, 1, 5)
	require.NoError(t, err)
	body, err = io.ReadAll(reader)
	require.NoError(t, err)
	require.Equal(t, rpgContent, string(body))

	missing := info
	missing.Sha256 = strings.Repeat("0", 64)
	exists, _, err = env.manager.HeadByDigest(ctx, "root", missing, 1, 5)
	require.Error(t, err)
	require.False(t, exists)
}

func TestRpgUploadFileFailures(t *testing.T) {
	ctx := context.Background()

	t.Run("storage unavailable", func(t *testing.T) {
		env := rpgNewEnv(t)
		env.manager.storageService, _ = storage.NewStorageService(rpgFailingResolver{})
		_, err := env.manager.UploadFile(ctx, "/f", 5, 1, "root", nil, strings.NewReader("x"), 1)
		require.ErrorContains(t, err, "failed to get blob context")
		_, err = env.manager.UploadFileNoDBUpdate(ctx, "root", nil, strings.NewReader("x"), 1, 5)
		require.ErrorContains(t, err, "failed to get blob context")
		_, _, err = env.manager.HeadByDigest(ctx, "root", types.FileInfo{}, 1, 5)
		require.ErrorContains(t, err, "failed to get blob context")
		_, err = env.manager.DownloadFileByDigest(ctx, "root", types.FileInfo{}, 1, 5)
		require.ErrorContains(t, err, "failed to get blob context")
	})
	t.Run("body cannot be read", func(t *testing.T) {
		env := rpgNewEnv(t)
		_, err := env.manager.UploadFile(ctx, "/f", 5, 1, "root", nil, rpgFailingReader{}, 1)
		require.ErrorContains(t, err, "failed to upload the file on temporary location")
	})
	t.Run("database failure", func(t *testing.T) {
		env := rpgNewEnv(t)
		env.blobs.createErr = errRpgBoom
		_, err := env.manager.UploadFile(ctx, "/f", 5, 1, "root", nil, strings.NewReader("x"), 1)
		require.ErrorIs(t, err, errRpgBoom)
	})
	t.Run("existing blob does not emit an event", func(t *testing.T) {
		env := rpgNewEnv(t)
		env.blobs.createdNew = false
		_, err := env.manager.UploadFile(ctx, "/f", 5, 1, "root", nil, strings.NewReader("x"), 1)
		require.NoError(t, err)
		require.Empty(t, env.repl.events)
	})
	t.Run("upload without database update", func(t *testing.T) {
		env := rpgNewEnv(t)
		info, err := env.manager.UploadFileNoDBUpdate(ctx, "root", nil, strings.NewReader("x"), 1, 5)
		require.NoError(t, err)
		require.Equal(t, int64(1), info.Size)
		require.Empty(t, env.nodes.created)
		require.Equal(t, 1, env.hook.commits)

		env.hook.err = errRpgBoom
		_, err = env.manager.UploadFileNoDBUpdate(ctx, "root", nil, strings.NewReader("y"), 1, 5)
		require.ErrorContains(t, err, "failed to execute commit callback")
	})
	t.Run("commit callback failure rolls back the save", func(t *testing.T) {
		env := rpgNewEnv(t)
		env.hook.err = errRpgBoom
		_, err := env.manager.UploadFile(ctx, "/f", 5, 1, "root", nil, strings.NewReader("x"), 1)
		require.ErrorContains(t, err, "failed to execute commit callback")
	})
}

type rpgFailingReader struct{}

func (rpgFailingReader) Read([]byte) (int, error) { return 0, errRpgBoom }

func TestRpgDownloadFileByPathFailures(t *testing.T) {
	ctx := context.Background()
	env := rpgNewEnv(t)

	env.nodes.byPathErr = errRpgBoom
	_, _, _, err := env.manager.DownloadFileByPath(ctx, "/f", 5, "reg", "root", false)
	var userErr *usererror.Error
	require.ErrorAs(t, err, &userErr)
	require.Equal(t, 404, userErr.Status)

	env.nodes.byPathErr = nil
	env.nodes.byPath = &types.Node{Name: "f", BlobID: "b"}
	env.blobs.findErr = errRpgBoom
	_, _, _, err = env.manager.DownloadFileByPath(ctx, "/f", 5, "reg", "root", false)
	require.ErrorAs(t, err, &userErr)

	env.blobs.findErr = nil
	env.blobs.blob = &types.GenericBlob{ID: "not-a-uuid"}
	_, _, _, err = env.manager.DownloadFileByPath(ctx, "/f", 5, "reg", "root", false)
	require.ErrorAs(t, err, &userErr)

	env.blobs.blob = &types.GenericBlob{ID: uuid.NewString(), Sha256: strings.Repeat("a", 64), Size: 3}

	env.manager.storageService, _ = storage.NewStorageService(rpgFailingResolver{})
	_, _, _, err = env.manager.DownloadFileByPath(ctx, "/f", 5, "reg", "root", false)
	require.ErrorContains(t, err, "failed to get blob context")
}

func TestRpgContextAndConstructors(t *testing.T) {
	type ctxKey string
	ctx := &Context{Context: context.WithValue(context.Background(), ctxKey("k"), "v")}
	require.Equal(t, "v", ctx.Value(ctxKey("k")))

	fm := NewFileManager(nil, nil, nil, rpgTx{}, &gitnesstypes.Config{}, nil, nil, nil, nil)
	require.NotNil(t, fm)
	provided := Provider(nil, nil, nil, rpgTx{}, &gitnesstypes.Config{}, nil, nil, nil, nil)
	require.NotNil(t, provided)
}
