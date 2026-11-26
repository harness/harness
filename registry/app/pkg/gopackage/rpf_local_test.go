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

package gopackage

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/gob"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/harness/gitness/app/api/request"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/services/refcache"
	urlprovider "github.com/harness/gitness/app/url"
	"github.com/harness/gitness/events"
	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	storagedriver "github.com/harness/gitness/registry/app/driver"
	registryevents "github.com/harness/gitness/registry/app/events/artifact"
	"github.com/harness/gitness/registry/app/events/asyncprocessing"
	"github.com/harness/gitness/registry/app/metadata"
	"github.com/harness/gitness/registry/app/pkg"
	"github.com/harness/gitness/registry/app/pkg/base"
	"github.com/harness/gitness/registry/app/pkg/commons"
	"github.com/harness/gitness/registry/app/pkg/filemanager"
	gopackagetype "github.com/harness/gitness/registry/app/pkg/types/gopackage"
	"github.com/harness/gitness/registry/app/storage"
	"github.com/harness/gitness/registry/app/store"
	"github.com/harness/gitness/registry/types"
	gitnessstore "github.com/harness/gitness/store"
	gtypes "github.com/harness/gitness/types"

	"github.com/stretchr/testify/require"
)

type rpfFakeDriver struct {
	storagedriver.StorageDriver
	content []byte
}

func (d *rpfFakeDriver) Reader(_ context.Context, _ string, _ int64) (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewReader(d.content)), nil
}

func rpfFileReader(ctx context.Context, t *testing.T, content []byte) *storage.FileReader {
	t.Helper()
	fr, err := storage.NewFileReader(ctx, &rpfFakeDriver{content: content}, "p", int64(len(content)))
	require.NoError(t, err)
	return fr
}

type rpfArtifactRepo struct {
	store.ArtifactRepository
	artifact *types.Artifact
	err      error
}

func (r *rpfArtifactRepo) GetByRegistryImageAndVersion(
	context.Context, int64, string, string,
) (*types.Artifact, error) {
	return r.artifact, r.err
}

func (r *rpfArtifactRepo) GetLatestByImageID(context.Context, int64) (*types.Artifact, error) {
	return r.artifact, r.err
}

func (r *rpfArtifactRepo) GetByName(context.Context, int64, string, ...types.QueryOption) (*types.Artifact, error) {
	return r.artifact, r.err
}

type rpfImageRepo struct {
	store.ImageRepository
	image *types.Image
	err   error
}

func (r *rpfImageRepo) GetByName(context.Context, int64, string, ...types.QueryOption) (*types.Image, error) {
	return r.image, r.err
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
	streams []string
}

func (p *rpfProducer) Send(_ context.Context, streamID string, _ map[string]any) (string, error) {
	p.streams = append(p.streams, streamID)
	return "1", nil
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
		producer, rpfCollector{})
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
	byPathErr error
	gotPath   string
	t         *testing.T
}

func (f *rpfFileManager) DownloadFileByPath(
	ctx context.Context, path string, _ int64, _ string, _ string, _ bool,
) (*storage.FileReader, int64, string, error) {
	f.gotPath = path
	if f.byPathErr != nil {
		return nil, 0, "", f.byPathErr
	}
	return rpfFileReader(ctx, f.t, []byte("content")), 7, "https://redirect.test/x", nil
}

type rpfLocalBase struct {
	base.LocalBase
	existsErr    error
	versionExist bool
	uploadErrOn  string
	uploaded     []string
	existsResult bool
	existsPath   string
}

func (b *rpfLocalBase) CheckIfVersionExists(context.Context, pkg.PackageArtifactInfo) (bool, error) {
	return b.versionExist, b.existsErr
}

func (b *rpfLocalBase) Upload(
	_ context.Context, _ pkg.ArtifactInfo, fileName, _, _ string, file io.ReadCloser, _ metadata.Metadata,
) (*commons.ResponseHeaders, string, error) {
	_, _ = io.Copy(io.Discard, file)
	b.uploaded = append(b.uploaded, fileName)
	if b.uploadErrOn != "" && fileName == b.uploadErrOn {
		return nil, "", errors.New("upload failed")
	}
	return &commons.ResponseHeaders{Code: http.StatusCreated}, "sum", nil
}

func (b *rpfLocalBase) Exists(_ context.Context, _ pkg.ArtifactInfo, path string) bool {
	b.existsPath = path
	return b.existsResult
}

const rpfModFile = "v1.0.0.mod"

func rpfInfo() gopackagetype.ArtifactInfo {
	return gopackagetype.ArtifactInfo{
		ArtifactInfo: pkg.ArtifactInfo{
			BaseInfo:      &pkg.BaseInfo{ParentID: 1, RootIdentifier: "root"},
			RegIdentifier: "reg",
			RegistryID:    3,
			Image:         "example.com/hello",
		},
		Version:  "v1.0.0",
		FileName: "v1.0.0.zip",
	}
}

func rpfNewLocal(
	t *testing.T, lb *rpfLocalBase, fm *rpfFileManager, artifacts *rpfArtifactRepo, images *rpfImageRepo,
) (*localRegistry, *rpfTaskRepo, *rpfProducer) {
	t.Helper()
	reporter, producer := rpfArtifactReporter(t)
	post, tasks := rpfPostReporter(t)
	if fm != nil {
		fm.t = t
	}
	reg := NewLocalRegistry(lb, fm, nil, nil, nil, images, artifacts, rpfURLProvider{}, reporter, post)
	local, ok := reg.(*localRegistry)
	require.True(t, ok)
	return local, tasks, producer
}

func TestRpfLocalUploadPackage(t *testing.T) {
	ctx := rpfAuthedContext()
	mod := func() io.ReadCloser { return io.NopCloser(bytes.NewBufferString("module example.com/hello")) }
	zipf := func() io.ReadCloser { return io.NopCloser(bytes.NewBufferString("zip")) }

	t.Run("success uploads all files and triggers post-processing", func(t *testing.T) {
		lb := &rpfLocalBase{}
		l, tasks, producer := rpfNewLocal(t, lb, nil, nil, nil)
		headers, err := l.UploadPackage(ctx, rpfInfo(), mod(), zipf())
		require.NoError(t, err)
		require.Equal(t, http.StatusCreated, headers.Code)
		require.Equal(t, []string{"v1.0.0.zip", rpfModFile, "v1.0.0.info"}, lb.uploaded)
		require.Len(t, producer.streams, 1)
		require.Len(t, tasks.keys, 2)
	})

	t.Run("version already exists", func(t *testing.T) {
		l, _, _ := rpfNewLocal(t, &rpfLocalBase{versionExist: true}, nil, nil, nil)
		_, err := l.UploadPackage(ctx, rpfInfo(), mod(), zipf())
		require.ErrorContains(t, err, "already exists")
	})

	t.Run("version lookup failure", func(t *testing.T) {
		l, _, _ := rpfNewLocal(t, &rpfLocalBase{existsErr: errors.New("db down")}, nil, nil, nil)
		_, err := l.UploadPackage(ctx, rpfInfo(), mod(), zipf())
		require.ErrorContains(t, err, "failed to check if version exists")
	})

	t.Run("a missing version record is fine", func(t *testing.T) {
		l, _, _ := rpfNewLocal(t, &rpfLocalBase{existsErr: gitnessstore.ErrResourceNotFound}, nil, nil, nil)
		_, err := l.UploadPackage(ctx, rpfInfo(), mod(), zipf())
		require.NoError(t, err)
	})

	for _, failing := range []string{"v1.0.0.zip", rpfModFile, "v1.0.0.info"} {
		t.Run("upload failure on "+failing, func(t *testing.T) {
			l, _, _ := rpfNewLocal(t, &rpfLocalBase{uploadErrOn: failing}, nil, nil, nil)
			_, err := l.UploadPackage(ctx, rpfInfo(), mod(), zipf())
			require.ErrorContains(t, err, "failed to upload")
		})
	}
}

func TestRpfLocalDownloadPackageFile(t *testing.T) {
	ctx := context.Background()
	art := &types.Artifact{Version: "v1.0.0"}
	images := &rpfImageRepo{image: &types.Image{ID: 5}}

	t.Run("regular file", func(t *testing.T) {
		fm := &rpfFileManager{}
		l, _, _ := rpfNewLocal(t, nil, fm, &rpfArtifactRepo{artifact: art}, images)
		headers, reader, rc, redirect, err := l.DownloadPackageFile(ctx, rpfInfo())
		require.NoError(t, err)
		require.NotNil(t, headers)
		require.NotNil(t, reader)
		require.Nil(t, rc)
		require.Equal(t, "https://redirect.test/x", redirect)
		require.Equal(t, "/example.com/hello/@v/v1.0.0/v1.0.0.zip", fm.gotPath)
	})

	t.Run("artifact missing or deleted", func(t *testing.T) {
		l, _, _ := rpfNewLocal(t, nil, &rpfFileManager{}, &rpfArtifactRepo{err: errors.New("gone")}, images)
		_, _, _, _, err := l.DownloadPackageFile(ctx, rpfInfo())
		require.ErrorContains(t, err, "artifact not found or deleted")
	})

	t.Run("storage failure", func(t *testing.T) {
		l, _, _ := rpfNewLocal(t, nil, &rpfFileManager{byPathErr: errors.New("x")}, &rpfArtifactRepo{artifact: art}, images)
		_, _, _, _, err := l.DownloadPackageFile(ctx, rpfInfo())
		require.ErrorContains(t, err, "failed to download file")
	})

	t.Run("index file when the version is empty", func(t *testing.T) {
		fm := &rpfFileManager{}
		l, _, _ := rpfNewLocal(t, nil, fm, &rpfArtifactRepo{}, images)
		info := rpfInfo()
		info.Version = ""
		_, reader, _, _, err := l.DownloadPackageFile(ctx, info)
		require.NoError(t, err)
		require.NotNil(t, reader)
		require.Equal(t, "/example.com/hello/index", fm.gotPath)

		l, _, _ = rpfNewLocal(t, nil, &rpfFileManager{byPathErr: errors.New("x")}, &rpfArtifactRepo{}, images)
		_, _, _, _, err = l.DownloadPackageFile(ctx, info)
		require.ErrorContains(t, err, "failed to download package index")
	})

	t.Run("latest resolves to the newest info file", func(t *testing.T) {
		fm := &rpfFileManager{}
		l, _, _ := rpfNewLocal(t, nil, fm, &rpfArtifactRepo{artifact: art}, images)
		info := rpfInfo()
		info.Version = LatestVersionKey
		_, reader, _, _, err := l.DownloadPackageFile(ctx, info)
		require.NoError(t, err)
		require.NotNil(t, reader)
		require.Equal(t, "/example.com/hello/@v/v1.0.0/v1.0.0.info", fm.gotPath)
	})

	t.Run("latest failures", func(t *testing.T) {
		info := rpfInfo()
		info.Version = LatestVersionKey
		l, _, _ := rpfNewLocal(t, nil, &rpfFileManager{}, &rpfArtifactRepo{artifact: art},
			&rpfImageRepo{err: errors.New("no image")})
		_, _, _, _, err := l.DownloadPackageFile(ctx, info)
		require.ErrorContains(t, err, "failed to get image")

		l, _, _ = rpfNewLocal(t, nil, &rpfFileManager{}, &rpfArtifactRepo{err: errors.New("none")}, images)
		_, _, _, _, err = l.DownloadPackageFile(ctx, info)
		require.ErrorContains(t, err, "failed to get latest artifact")

		weird := &types.Artifact{Version: LatestVersionKey}
		l, _, _ = rpfNewLocal(t, nil, &rpfFileManager{}, &rpfArtifactRepo{artifact: weird}, images)
		_, _, _, _, err = l.DownloadPackageFile(ctx, info)
		require.ErrorContains(t, err, "cannot be")
	})
}

func TestRpfLocalRegenerate(t *testing.T) {
	ctx := rpfAuthedContext()
	l, tasks, _ := rpfNewLocal(t, nil, nil, nil, nil)
	require.Equal(t, artifact.RegistryTypeVIRTUAL, l.GetArtifactType())
	require.Equal(t, []artifact.PackageType{artifact.PackageTypeGO}, l.GetPackageTypes())

	headers, err := l.RegeneratePackageIndex(ctx, rpfInfo())
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, headers.Code)
	headers, err = l.RegeneratePackageMetadata(ctx, rpfInfo())
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, headers.Code)
	require.Equal(t, []string{
		"package_3_example.com/hello",
		"package_3_example.com/hello_v1.0.0_metadata",
	}, tasks.keys)
}

type rpfFakeLocalRegistry struct {
	LocalRegistry
	called bool
}

func (r *rpfFakeLocalRegistry) DownloadPackageFile(
	context.Context, gopackagetype.ArtifactInfo,
) (*commons.ResponseHeaders, *storage.FileReader, io.ReadCloser, string, error) {
	r.called = true
	return &commons.ResponseHeaders{Code: http.StatusOK}, nil, nil, "redirect", nil
}

func TestRpfLocalRegistryHelper(t *testing.T) {
	ctx := rpfAuthedContext()
	lb := &rpfLocalBase{existsResult: true}
	reg := &rpfFakeLocalRegistry{}
	post, tasks := rpfPostReporter(t)
	h := NewLocalRegistryHelper(reg, lb, post)
	info := rpfInfo()

	require.True(t, h.FileExists(ctx, info, "/example.com/hello/@v/list"))
	require.Equal(t, "example.com/hello/@v/list", lb.existsPath)

	h.RegeneratePackageIndex(ctx, info)
	h.RegeneratePackageMetadata(ctx, info)
	require.Len(t, tasks.keys, 2)

	headers, _, _, redirect, err := h.DownloadFile(ctx, info)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, headers.Code)
	require.Equal(t, "redirect", redirect)
	require.True(t, reg.called)
}

type rpfFakeLocalHelper struct {
	exists      bool
	downloadErr error
	done        chan struct{}
	indexCalls  int
	metaCalls   int
}

func (h *rpfFakeLocalHelper) FileExists(context.Context, gopackagetype.ArtifactInfo, string) bool {
	return h.exists
}

func (h *rpfFakeLocalHelper) RegeneratePackageIndex(context.Context, gopackagetype.ArtifactInfo) {
	h.indexCalls++
}

func (h *rpfFakeLocalHelper) RegeneratePackageMetadata(context.Context, gopackagetype.ArtifactInfo) {
	h.metaCalls++
	if h.done != nil {
		close(h.done)
	}
}

func (h *rpfFakeLocalHelper) DownloadFile(
	context.Context, gopackagetype.ArtifactInfo,
) (*commons.ResponseHeaders, *storage.FileReader, io.ReadCloser, string, error) {
	if h.downloadErr != nil {
		return nil, nil, nil, "", h.downloadErr
	}
	return &commons.ResponseHeaders{Code: http.StatusOK}, nil, nil, "local-redirect", nil
}

func rpfGoUpstream(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/example.com/hello/@v/list", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, "v1.0.0\n")
	})
	mux.HandleFunc("/example.com/hello/@latest", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"Version":"v1.0.0"}`)
	})
	mux.HandleFunc("/example.com/hello/@v/v1.0.0.info", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"Version":"v1.0.0"}`)
	})
	mux.HandleFunc("/example.com/hello/@v/v1.0.0.mod", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, "module example.com/hello")
	})
	mux.HandleFunc("/example.com/hello/@v/v1.0.0.zip", func(w http.ResponseWriter, _ *http.Request) {
		var buf bytes.Buffer
		zw := zip.NewWriter(&buf)
		_, _ = zw.Create("example.com/hello@v1.0.0/main.go")
		_ = zw.Close()
		_, _ = w.Write(buf.Bytes())
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func rpfNewProxy(repoURL string, lb *rpfLocalBase, helper *rpfFakeLocalHelper, storeErr error) *proxy {
	up := &types.UpstreamProxy{
		RepoKey: "up", RepoURL: repoURL, RepoAuthType: string(artifact.AuthTypeAnonymous),
		PackageType: artifact.PackageTypeGO,
	}
	p := NewProxy(lb, nil, &rpfProxyStore{proxy: up, err: storeErr}, nil, nil, nil, nil, rpfURLProvider{},
		refcache.SpaceFinder{}, nil, nil, helper)
	px, ok := p.(*proxy)
	if !ok {
		return nil
	}
	return px
}

func TestRpfProxyDownloads(t *testing.T) {
	ctx := context.Background()
	srv := rpfGoUpstream(t)

	t.Run("static", func(t *testing.T) {
		p := rpfNewProxy(srv.URL, &rpfLocalBase{}, &rpfFakeLocalHelper{}, nil)
		require.Equal(t, artifact.RegistryTypeUPSTREAM, p.GetArtifactType())
		require.Equal(t, []artifact.PackageType{artifact.PackageTypeGO}, p.GetPackageTypes())
		_, err := p.UploadPackage(ctx, rpfInfo(), nil, nil)
		require.Error(t, err)
	})

	t.Run("index is always fetched from upstream", func(t *testing.T) {
		p := rpfNewProxy(srv.URL, &rpfLocalBase{}, &rpfFakeLocalHelper{}, nil)
		info := rpfInfo()
		info.Version = ""
		headers, _, rc, _, err := p.DownloadPackageFile(ctx, info)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, headers.Code)
		data, err := io.ReadAll(rc)
		require.NoError(t, err)
		require.Equal(t, "v1.0.0\n", string(data))
	})

	t.Run("latest is always fetched from upstream", func(t *testing.T) {
		p := rpfNewProxy(srv.URL, &rpfLocalBase{}, &rpfFakeLocalHelper{}, nil)
		info := rpfInfo()
		info.Version = LatestVersionKey
		_, _, rc, _, err := p.DownloadPackageFile(ctx, info)
		require.NoError(t, err)
		data, err := io.ReadAll(rc)
		require.NoError(t, err)
		require.Contains(t, string(data), "v1.0.0")
	})

	t.Run("zip is streamed and every file is cached", func(t *testing.T) {
		lb := &rpfLocalBase{}
		helper := &rpfFakeLocalHelper{done: make(chan struct{})}
		p := rpfNewProxy(srv.URL, lb, helper, nil)
		_, _, rc, _, err := p.DownloadPackageFile(ctx, rpfInfo())
		require.NoError(t, err)
		_, err = io.Copy(io.Discard, rc)
		require.NoError(t, err)
		select {
		case <-helper.done:
		case <-time.After(10 * time.Second):
			t.Fatal("package was not cached locally")
		}
		require.Equal(t, []string{"v1.0.0.info", rpfModFile, "v1.0.0.zip"}, lb.uploaded)
		require.Equal(t, 1, helper.indexCalls)
	})

	t.Run("non zip files are not cached", func(t *testing.T) {
		lb := &rpfLocalBase{}
		p := rpfNewProxy(srv.URL, lb, &rpfFakeLocalHelper{}, nil)
		info := rpfInfo()
		info.FileName = rpfModFile
		_, _, rc, _, err := p.DownloadPackageFile(ctx, info)
		require.NoError(t, err)
		require.NoError(t, rc.Close())
		require.Empty(t, lb.uploaded)
	})

	t.Run("local copy is preferred", func(t *testing.T) {
		p := rpfNewProxy(srv.URL, &rpfLocalBase{}, &rpfFakeLocalHelper{exists: true}, nil)
		headers, _, _, redirect, err := p.DownloadPackageFile(ctx, rpfInfo())
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, headers.Code)
		require.Equal(t, "local-redirect", redirect)
	})

	t.Run("broken local copy falls back to upstream", func(t *testing.T) {
		helper := &rpfFakeLocalHelper{exists: true, downloadErr: errors.New("corrupt")}
		p := rpfNewProxy(srv.URL, &rpfLocalBase{}, helper, nil)
		info := rpfInfo()
		info.FileName = rpfModFile
		_, _, rc, redirect, err := p.DownloadPackageFile(ctx, info)
		require.NoError(t, err)
		require.NotNil(t, rc)
		require.Empty(t, redirect)
	})

	t.Run("failures", func(t *testing.T) {
		info := rpfInfo()
		info.FileName = rpfModFile
		p := rpfNewProxy(srv.URL, &rpfLocalBase{}, &rpfFakeLocalHelper{}, errors.New("db down"))
		_, _, _, _, err := p.DownloadPackageFile(ctx, info)
		require.ErrorContains(t, err, "failed to get upstream proxy")

		missing := info
		missing.FileName = "v9.9.9.mod"
		p = rpfNewProxy(srv.URL, &rpfLocalBase{}, &rpfFakeLocalHelper{}, nil)
		_, _, _, _, err = p.DownloadPackageFile(ctx, missing)
		require.ErrorContains(t, err, "failed to download package file from upstream")

		noIndex := rpfInfo()
		noIndex.Version = ""
		noIndex.Image = "missing.example/pkg"
		_, _, _, _, err = p.DownloadPackageFile(ctx, noIndex)
		require.ErrorContains(t, err, "failed to download package index")

		noLatest := noIndex
		noLatest.Version = LatestVersionKey
		_, _, _, _, err = p.DownloadPackageFile(ctx, noLatest)
		require.ErrorContains(t, err, "failed to download package latest version")
	})

	t.Run("regenerate delegates to the local helper", func(t *testing.T) {
		helper := &rpfFakeLocalHelper{}
		p := rpfNewProxy(srv.URL, &rpfLocalBase{}, helper, nil)
		headers, err := p.RegeneratePackageIndex(ctx, rpfInfo())
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, headers.Code)
		headers, err = p.RegeneratePackageMetadata(ctx, rpfInfo())
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, headers.Code)
		require.Equal(t, 1, helper.indexCalls)
		require.Equal(t, 1, helper.metaCalls)
	})
}

func TestRpfProxyCacheFailure(t *testing.T) {
	ctx := context.Background()
	srv := rpfGoUpstream(t)
	p := rpfNewProxy(srv.URL, &rpfLocalBase{uploadErrOn: rpfModFile}, &rpfFakeLocalHelper{}, nil)
	helper, err := NewRemoteRegistryHelper(ctx, refcache.SpaceFinder{}, types.UpstreamProxy{
		RepoURL: srv.URL, RepoAuthType: string(artifact.AuthTypeAnonymous), PackageType: artifact.PackageTypeGO,
	}, nil)
	require.NoError(t, err)
	info := rpfInfo()
	err = p.putFileToLocal(ctx, &info, helper)
	require.ErrorContains(t, err, "failed to cache mod file")

	info = rpfInfo()
	info.Image = "missing.example/pkg"
	err = p.putFileToLocal(ctx, &info, helper)
	require.ErrorContains(t, err, "failed to cache info file")
}
