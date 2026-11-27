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

package cargo

import (
	"bytes"
	"context"
	"encoding/gob"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/harness/gitness/app/api/request"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/services/refcache"
	urlprovider "github.com/harness/gitness/app/url"
	"github.com/harness/gitness/events"
	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	registryevents "github.com/harness/gitness/registry/app/events/artifact"
	"github.com/harness/gitness/registry/app/events/asyncprocessing"
	"github.com/harness/gitness/registry/app/metadata"
	cargometadata "github.com/harness/gitness/registry/app/metadata/cargo"
	"github.com/harness/gitness/registry/app/pkg"
	"github.com/harness/gitness/registry/app/pkg/base"
	"github.com/harness/gitness/registry/app/pkg/commons"
	"github.com/harness/gitness/registry/app/pkg/filemanager"
	cargotype "github.com/harness/gitness/registry/app/pkg/types/cargo"
	"github.com/harness/gitness/registry/app/storage"
	"github.com/harness/gitness/registry/app/store"
	"github.com/harness/gitness/registry/types"
	gtypes "github.com/harness/gitness/types"

	"github.com/stretchr/testify/require"
)

type rpfArtifactRepo struct {
	store.ArtifactRepository
	artifact *types.Artifact
	saved    *types.Artifact
	err      error
	saveErr  error
}

func (r *rpfArtifactRepo) GetByRegistryImageAndVersion(
	context.Context, int64, string, string,
) (*types.Artifact, error) {
	return r.artifact, r.err
}

func (r *rpfArtifactRepo) CreateOrUpdate(_ context.Context, a *types.Artifact) (int64, error) {
	r.saved = a
	return 1, r.saveErr
}

type rpfURLProvider struct {
	urlprovider.Provider
}

func (rpfURLProvider) PackageURL(_ context.Context, regRef string, pkgType string, _ ...string) string {
	return "https://harness.test/pkg/" + regRef + "/" + pkgType
}

type rpfProxyStore struct {
	store.UpstreamProxyConfigRepository
	proxy *types.UpstreamProxy
	err   error
}

func (s *rpfProxyStore) GetByRegistryIdentifier(
	context.Context, int64, string, ...types.QueryOption,
) (*types.UpstreamProxy, error) {
	return s.proxy, s.err
}

type rpfProducer struct {
	mu      sync.Mutex
	streams []string
}

func (p *rpfProducer) Send(_ context.Context, streamID string, _ map[string]any) (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.streams = append(p.streams, streamID)
	return "1", nil
}

func (p *rpfProducer) count() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.streams)
}

type rpfCollector struct{}

func (rpfCollector) RecordError(string, string)   {}
func (rpfCollector) RecordDiscard(string, string) {}

func rpfEventSystem(t *testing.T) (*events.System, *rpfProducer) {
	t.Helper()
	gob.Register(&registryevents.CommonArtifact{})
	producer := &rpfProducer{}
	system, err := events.NewSystem(
		func(string, string) (events.StreamConsumer, error) { return nil, errors.New("unused") },
		producer, rpfCollector{}, false)
	require.NoError(t, err)
	return system, producer
}

func rpfArtifactReporter(t *testing.T) (*registryevents.Reporter, *rpfProducer) {
	t.Helper()
	system, producer := rpfEventSystem(t)
	reporter, err := registryevents.NewReporter(system)
	require.NoError(t, err)
	return reporter, producer
}

type rpfTx struct{}

func (rpfTx) WithTx(ctx context.Context, fn func(ctx context.Context) error, _ ...any) error {
	return fn(ctx)
}

type rpfTaskRepo struct {
	store.TaskRepository
	keys []string
}

func (r *rpfTaskRepo) UpsertTask(_ context.Context, task *types.Task) error {
	r.keys = append(r.keys, task.Key)
	return nil
}

func (r *rpfTaskRepo) LockForUpdate(context.Context, *types.Task) (types.TaskStatus, error) {
	return types.TaskStatusPending, nil
}

func (r *rpfTaskRepo) UpdateStatus(context.Context, string, types.TaskStatus) error { return nil }

type rpfTaskSourceRepo struct {
	store.TaskSourceRepository
}

func (rpfTaskSourceRepo) InsertSource(context.Context, string, types.SourceRef) error { return nil }

type rpfTaskEventRepo struct {
	store.TaskEventRepository
}

func (rpfTaskEventRepo) LogTaskEvent(context.Context, string, string, []byte) error { return nil }

func rpfPostReporter(t *testing.T) (*asyncprocessing.Reporter, *rpfTaskRepo) {
	t.Helper()
	system, _ := rpfEventSystem(t)
	tasks := &rpfTaskRepo{}
	reporter, err := asyncprocessing.NewReporter(rpfTx{}, system, tasks, rpfTaskSourceRepo{}, rpfTaskEventRepo{})
	require.NoError(t, err)
	return reporter, tasks
}

func rpfAuthedContext() context.Context {
	return request.WithAuthSession(context.Background(), &auth.Session{Principal: gtypes.Principal{ID: 9}})
}

type rpfFileManager struct {
	filemanager.FileManager
	byPathErr   error
	uploadErr   error
	downloadErr error
	content     []byte
	gotPath     string
	t           *testing.T
}

func (f *rpfFileManager) DownloadFileByPath(
	ctx context.Context, path string, _ int64, _ string, _ string, _ bool,
) (*storage.FileReader, int64, string, error) {
	f.gotPath = path
	if f.byPathErr != nil {
		return nil, 0, "", f.byPathErr
	}
	return rpfNewFileReader(ctx, f.t, []byte("content")), 7, "https://redirect.test/x", nil
}

func (f *rpfFileManager) UploadFileNoDBUpdate(
	context.Context, string, multipart.File, io.Reader, int64, int64,
) (types.FileInfo, error) {
	return types.FileInfo{Sha256: "abc"}, f.uploadErr
}

func (f *rpfFileManager) DownloadFileByDigest(
	ctx context.Context, _ string, _ types.FileInfo, _ int64, _ int64,
) (*storage.FileReader, error) {
	if f.downloadErr != nil {
		return nil, f.downloadErr
	}
	return rpfNewFileReader(ctx, f.t, f.content), nil
}

type rpfLocalBase struct {
	base.LocalBase
	uploadErr    error
	uploaded     []string
	uploadedPath string
	existsResult bool
	existsPath   string
	moved        bool
}

func (b *rpfLocalBase) Upload(
	_ context.Context, _ pkg.ArtifactInfo, fileName, _, path string, file io.ReadCloser, _ metadata.Metadata,
) (*commons.ResponseHeaders, string, error) {
	_, _ = io.Copy(io.Discard, file)
	b.uploaded = append(b.uploaded, fileName)
	b.uploadedPath = path
	if b.uploadErr != nil {
		return nil, "", b.uploadErr
	}
	return &commons.ResponseHeaders{Code: http.StatusCreated}, "sum", nil
}

func (b *rpfLocalBase) Exists(_ context.Context, _ pkg.ArtifactInfo, path string) bool {
	b.existsPath = path
	return b.existsResult
}

func (b *rpfLocalBase) UpdateFileManagerAndCreateArtifact(
	_ context.Context, _ pkg.ArtifactInfo, _, path string, _ metadata.Metadata, _ types.FileInfo, _ bool,
) (*commons.ResponseHeaders, string, int64, bool, error) {
	b.moved = true
	b.uploadedPath = path
	return &commons.ResponseHeaders{Code: http.StatusCreated}, "sum", 5, false, b.uploadErr
}

func rpfRegInfo() cargotype.ArtifactInfo {
	return cargotype.ArtifactInfo{
		ArtifactInfo: pkg.ArtifactInfo{
			BaseInfo:      &pkg.BaseInfo{ParentID: 1, RootIdentifier: "root"},
			RegIdentifier: "reg",
			RegistryID:    3,
			Image:         "demo",
		},
		Version: "1.0.0",
	}
}

func rpfNewLocal(
	t *testing.T, lb *rpfLocalBase, fm *rpfFileManager, artifacts *rpfArtifactRepo,
) (*localRegistry, *rpfTaskRepo, *rpfProducer) {
	t.Helper()
	reporter, producer := rpfArtifactReporter(t)
	post, tasks := rpfPostReporter(t)
	if fm != nil {
		fm.t = t
	}
	reg := NewLocalRegistry(lb, fm, nil, nil, nil, nil, artifacts, rpfURLProvider{}, reporter, post)
	local, ok := reg.(*localRegistry)
	require.True(t, ok)
	return local, tasks, producer
}

func TestRpfLocalUploadPackage(t *testing.T) {
	ctx := rpfAuthedContext()
	crate := func() io.ReadCloser { return io.NopCloser(bytes.NewBufferString("crate")) }

	lb := &rpfLocalBase{}
	l, tasks, producer := rpfNewLocal(t, lb, nil, nil)
	headers, err := l.UploadPackage(ctx, rpfRegInfo(), &cargometadata.VersionMetadata{Name: "demo"}, crate())
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, headers.Code)
	require.Equal(t, []string{"demo-1.0.0.crate"}, lb.uploaded)
	require.Equal(t, "/crates/demo/1.0.0/demo-1.0.0.crate", lb.uploadedPath)
	require.Len(t, producer.streams, 1)
	require.Equal(t, []string{"package_3_demo"}, tasks.keys)

	l, _, _ = rpfNewLocal(t, &rpfLocalBase{uploadErr: errors.New("x")}, nil, nil)
	_, err = l.UploadPackage(ctx, rpfRegInfo(), &cargometadata.VersionMetadata{}, crate())
	require.ErrorContains(t, err, "failed to upload crate file")
}

func TestRpfLocalDownloads(t *testing.T) {
	ctx := context.Background()
	art := &types.Artifact{Version: "1.0.0"}

	t.Run("index", func(t *testing.T) {
		fm := &rpfFileManager{}
		l, _, _ := rpfNewLocal(t, nil, fm, nil)
		headers, reader, rc, redirect, err := l.DownloadPackageIndex(ctx, rpfRegInfo(), "de/mo/demo")
		require.NoError(t, err)
		require.NotNil(t, headers)
		require.NotNil(t, reader)
		require.Nil(t, rc)
		require.Equal(t, "https://redirect.test/x", redirect)
		require.Equal(t, "/index/de/mo/demo", fm.gotPath)

		l, _, _ = rpfNewLocal(t, nil, &rpfFileManager{byPathErr: errors.New("x")}, nil)
		_, _, _, _, err = l.DownloadPackageIndex(ctx, rpfRegInfo(), "de/mo/demo")
		require.ErrorContains(t, err, "failed to download package index file")
	})

	t.Run("crate", func(t *testing.T) {
		fm := &rpfFileManager{}
		l, _, _ := rpfNewLocal(t, nil, fm, &rpfArtifactRepo{artifact: art})
		_, reader, _, _, err := l.DownloadPackage(ctx, rpfRegInfo())
		require.NoError(t, err)
		require.NotNil(t, reader)
		require.Equal(t, "/crates/demo/1.0.0/demo-1.0.0.crate", fm.gotPath)

		l, _, _ = rpfNewLocal(t, nil, &rpfFileManager{}, &rpfArtifactRepo{err: errors.New("gone")})
		_, _, _, _, err = l.DownloadPackage(ctx, rpfRegInfo())
		require.ErrorContains(t, err, "artifact not found or deleted")

		l, _, _ = rpfNewLocal(t, nil, &rpfFileManager{byPathErr: errors.New("x")}, &rpfArtifactRepo{artifact: art})
		_, _, _, _, err = l.DownloadPackage(ctx, rpfRegInfo())
		require.ErrorContains(t, err, "failed to download package file")
	})
}

func TestRpfLocalYankAndRegenerate(t *testing.T) {
	ctx := rpfAuthedContext()
	meta, err := json.Marshal(cargometadata.VersionMetadataDB{
		VersionMetadata: cargometadata.VersionMetadata{Name: "demo"},
	})
	require.NoError(t, err)

	t.Run("yank", func(t *testing.T) {
		repo := &rpfArtifactRepo{artifact: &types.Artifact{Version: "1.0.0", Metadata: meta}}
		l, tasks, _ := rpfNewLocal(t, nil, nil, repo)
		headers, err := l.UpdateYank(ctx, rpfRegInfo(), true)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, headers.Code)
		var saved cargometadata.VersionMetadataDB
		require.NoError(t, json.Unmarshal(repo.saved.Metadata, &saved))
		require.True(t, saved.Yanked)
		require.Len(t, tasks.keys, 1)

		_, err = l.UpdateYank(ctx, rpfRegInfo(), false)
		require.NoError(t, err)
		require.NoError(t, json.Unmarshal(repo.saved.Metadata, &saved))
		require.False(t, saved.Yanked)
	})

	t.Run("yank failures", func(t *testing.T) {
		l, _, _ := rpfNewLocal(t, nil, nil, &rpfArtifactRepo{err: errors.New("none")})
		_, err := l.UpdateYank(ctx, rpfRegInfo(), true)
		require.ErrorContains(t, err, "failed to get artifact by image and version")

		l, _, _ = rpfNewLocal(t, nil, nil,
			&rpfArtifactRepo{artifact: &types.Artifact{Version: "1.0.0", Metadata: []byte("{")}})
		_, err = l.UpdateYank(ctx, rpfRegInfo(), true)
		require.ErrorContains(t, err, "failed to unmarshal metadata")

		l, _, _ = rpfNewLocal(t, nil, nil,
			&rpfArtifactRepo{artifact: &types.Artifact{Version: "1.0.0", Metadata: meta}, saveErr: errors.New("x")})
		_, err = l.UpdateYank(ctx, rpfRegInfo(), true)
		require.ErrorContains(t, err, "failed to update artifact")
	})

	t.Run("regenerate and static", func(t *testing.T) {
		l, tasks, _ := rpfNewLocal(t, nil, nil, nil)
		headers, err := l.RegeneratePackageIndex(ctx, rpfRegInfo())
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, headers.Code)
		require.Len(t, tasks.keys, 1)
		require.Equal(t, artifact.RegistryTypeVIRTUAL, l.GetArtifactType())
		require.Equal(t, []artifact.PackageType{artifact.PackageTypeCARGO}, l.GetPackageTypes())
	})
}

type rpfFakeLocalRegistry struct {
	LocalRegistry
}

func (rpfFakeLocalRegistry) DownloadPackage(
	context.Context, cargotype.ArtifactInfo,
) (*commons.ResponseHeaders, *storage.FileReader, io.ReadCloser, string, error) {
	return &commons.ResponseHeaders{Code: http.StatusOK}, nil, nil, "redirect", nil
}

func TestRpfLocalRegistryHelper(t *testing.T) {
	ctx := rpfAuthedContext()
	lb := &rpfLocalBase{existsResult: true}
	post, tasks := rpfPostReporter(t)
	h := NewLocalRegistryHelper(rpfFakeLocalRegistry{}, lb, post)
	info := rpfRegInfo()

	require.True(t, h.FileExists(ctx, info))
	require.Equal(t, "crates/demo/1.0.0/demo-1.0.0.crate", lb.existsPath)

	headers, _, _, redirect, err := h.DownloadFile(ctx, info)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, headers.Code)
	require.Equal(t, "redirect", redirect)

	h.UpdatePackageIndex(ctx, info)
	require.Equal(t, []string{"package_3_demo"}, tasks.keys)

	_, _, size, existent, err := h.MoveTempFile(ctx, &info, types.FileInfo{}, &cargometadata.VersionMetadata{})
	require.NoError(t, err)
	require.Equal(t, int64(5), size)
	require.False(t, existent)
	require.True(t, lb.moved)
	require.Equal(t, "/crates/demo/1.0.0/demo-1.0.0.crate", lb.uploadedPath)
}

type rpfFakeLocalHelper struct {
	exists       bool
	downloadErr  error
	moveErr      error
	moved        chan struct{}
	indexUpdates int
}

func (h *rpfFakeLocalHelper) FileExists(context.Context, cargotype.ArtifactInfo) bool {
	return h.exists
}

func (h *rpfFakeLocalHelper) DownloadFile(
	context.Context, cargotype.ArtifactInfo,
) (*commons.ResponseHeaders, *storage.FileReader, io.ReadCloser, string, error) {
	if h.downloadErr != nil {
		return nil, nil, nil, "", h.downloadErr
	}
	return &commons.ResponseHeaders{Code: http.StatusOK}, nil, nil, "local-redirect", nil
}

func (h *rpfFakeLocalHelper) MoveTempFile(
	context.Context, *cargotype.ArtifactInfo, types.FileInfo, *cargometadata.VersionMetadata,
) (*commons.ResponseHeaders, string, int64, bool, error) {
	if h.moved != nil {
		defer close(h.moved)
	}
	return nil, "", 0, false, h.moveErr
}

func (h *rpfFakeLocalHelper) UpdatePackageIndex(context.Context, cargotype.ArtifactInfo) {
	h.indexUpdates++
}

func rpfCargoUpstream(t *testing.T, crate []byte) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/config.json", func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprintf(w, `{"dl":"http://%s/dl","api":"http://%s"}`, r.Host, r.Host)
	})
	mux.HandleFunc("/dl/demo/1.0.0/download", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(crate)
	})
	mux.HandleFunc("/de/mo/demo", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"name":"demo","vers":"1.0.0"}`+"\n")
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func rpfNewProxy(
	repoURL string, fm *rpfFileManager, helper *rpfFakeLocalHelper, storeErr error, reporter *registryevents.Reporter,
) *proxy {
	up := &types.UpstreamProxy{
		RepoKey: "up", RepoURL: repoURL, RepoAuthType: string(artifact.AuthTypeAnonymous),
		PackageType: artifact.PackageTypeCARGO,
	}
	p := NewProxy(fm, &rpfProxyStore{proxy: up, err: storeErr}, nil, nil, nil, nil, rpfURLProvider{},
		refcache.SpaceFinder{}, nil, helper, reporter)
	px, ok := p.(*proxy)
	if !ok {
		return nil
	}
	return px
}

func TestRpfProxy(t *testing.T) {
	ctx := rpfAuthedContext()
	crate := rpfBuildCrate(t, map[string]string{
		"demo-1.0.0/Cargo.toml": "[package]\nname = \"demo\"\nversion = \"1.0.0\"\n",
	})
	srv := rpfCargoUpstream(t, crate)
	reporter, producer := rpfArtifactReporter(t)

	t.Run("static", func(t *testing.T) {
		p := rpfNewProxy(srv.URL, nil, &rpfFakeLocalHelper{}, nil, reporter)
		require.Equal(t, artifact.RegistryTypeUPSTREAM, p.GetArtifactType())
		require.Equal(t, []artifact.PackageType{artifact.PackageTypeCARGO}, p.GetPackageTypes())
		_, err := p.UploadPackage(ctx, rpfRegInfo(), nil, nil)
		require.Error(t, err)
		_, err = p.UpdateYank(ctx, rpfRegInfo(), true)
		require.Error(t, err)
		_, err = p.RegeneratePackageIndex(ctx, rpfRegInfo())
		require.Error(t, err)
	})

	t.Run("index", func(t *testing.T) {
		p := rpfNewProxy(srv.URL, nil, &rpfFakeLocalHelper{}, nil, reporter)
		headers, _, rc, _, err := p.DownloadPackageIndex(ctx, rpfRegInfo(), "de/mo/demo")
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, headers.Code)
		data, err := io.ReadAll(rc)
		require.NoError(t, err)
		require.Contains(t, string(data), `"vers":"1.0.0"`)

		_, _, _, _, err = p.DownloadPackageIndex(ctx, rpfRegInfo(), "nope")
		require.ErrorContains(t, err, "failed to get package index file")

		p = rpfNewProxy(srv.URL, nil, &rpfFakeLocalHelper{}, errors.New("db down"), reporter)
		_, _, _, _, err = p.DownloadPackageIndex(ctx, rpfRegInfo(), "de/mo/demo")
		require.ErrorContains(t, err, "failed to get upstream proxy")
	})

	t.Run("crate is streamed then cached locally", func(t *testing.T) {
		helper := &rpfFakeLocalHelper{moved: make(chan struct{})}
		fm := &rpfFileManager{content: crate}
		fm.t = t
		p := rpfNewProxy(srv.URL, fm, helper, nil, reporter)
		headers, _, rc, _, err := p.DownloadPackage(ctx, rpfRegInfo())
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, headers.Code)
		data, err := io.ReadAll(rc)
		require.NoError(t, err)
		require.Equal(t, crate, data)
		select {
		case <-helper.moved:
		case <-time.After(10 * time.Second):
			t.Fatal("crate was not cached locally")
		}
		require.Eventually(t, func() bool { return producer.count() > 0 }, 5*time.Second, 10*time.Millisecond)
	})

	t.Run("local copy is preferred", func(t *testing.T) {
		p := rpfNewProxy(srv.URL, nil, &rpfFakeLocalHelper{exists: true}, nil, reporter)
		_, _, _, redirect, err := p.DownloadPackage(ctx, rpfRegInfo())
		require.NoError(t, err)
		require.Equal(t, "local-redirect", redirect)
	})

	t.Run("failures", func(t *testing.T) {
		p := rpfNewProxy(srv.URL, nil, &rpfFakeLocalHelper{}, errors.New("db down"), reporter)
		_, _, _, _, err := p.DownloadPackage(ctx, rpfRegInfo())
		require.ErrorContains(t, err, "failed to get upstream proxy")

		dead := httptest.NewServer(http.NotFoundHandler())
		url := dead.URL
		dead.Close()
		p = rpfNewProxy(url, nil, &rpfFakeLocalHelper{}, nil, reporter)
		_, _, _, _, err = p.DownloadPackage(ctx, rpfRegInfo())
		require.ErrorContains(t, err, "failed to get registry config")
	})
}

func TestRpfProxyPutFileToLocal(t *testing.T) {
	ctx := rpfAuthedContext()
	crate := rpfBuildCrate(t, map[string]string{
		"demo-1.0.0/Cargo.toml": "[package]\nname = \"demo\"\nversion = \"1.0.0\"\n",
	})
	srv := rpfCargoUpstream(t, crate)
	reporter, producer := rpfArtifactReporter(t)
	remote, err := NewRemoteRegistryHelper(ctx, refcache.SpaceFinder{}, types.UpstreamProxy{
		RepoURL: srv.URL + "/dl", RepoAuthType: string(artifact.AuthTypeAnonymous), PackageType: artifact.PackageTypeCARGO,
	}, nil, "")
	require.NoError(t, err)

	run := func(fm *rpfFileManager, helper *rpfFakeLocalHelper) error {
		fm.t = t
		p := rpfNewProxy(srv.URL, fm, helper, nil, reporter)
		info := rpfRegInfo()
		return p.putFileToLocal(ctx, &info, remote)
	}

	helper := &rpfFakeLocalHelper{}
	require.NoError(t, run(&rpfFileManager{content: crate}, helper))
	require.Equal(t, 1, helper.indexUpdates)
	require.Len(t, producer.streams, 1)

	require.ErrorContains(t, run(&rpfFileManager{uploadErr: errors.New("x")}, &rpfFakeLocalHelper{}),
		"failed to upload with registry")
	require.ErrorContains(t, run(&rpfFileManager{downloadErr: errors.New("x")}, &rpfFakeLocalHelper{}),
		"failed to download with registry")
	require.ErrorContains(t, run(&rpfFileManager{content: []byte("garbage")}, &rpfFakeLocalHelper{}),
		"failed to generate metadata")
	require.ErrorContains(t, run(&rpfFileManager{content: crate}, &rpfFakeLocalHelper{moveErr: errors.New("x")}),
		"failed to move temp file")

	badRemote, err := NewRemoteRegistryHelper(ctx, refcache.SpaceFinder{}, types.UpstreamProxy{
		RepoURL: srv.URL + "/none", RepoAuthType: string(artifact.AuthTypeAnonymous), PackageType: artifact.PackageTypeCARGO,
	}, nil, "")
	require.NoError(t, err)
	p := rpfNewProxy(srv.URL, &rpfFileManager{t: t}, &rpfFakeLocalHelper{}, nil, reporter)
	info := rpfRegInfo()
	require.ErrorContains(t, p.putFileToLocal(ctx, &info, badRemote), "failed to get registry config")
}
