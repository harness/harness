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

package python

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
	"testing"
	"time"

	"github.com/harness/gitness/app/api/request"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/services/refcache"
	urlprovider "github.com/harness/gitness/app/url"
	"github.com/harness/gitness/events"
	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	registryevents "github.com/harness/gitness/registry/app/events/artifact"
	"github.com/harness/gitness/registry/app/metadata"
	pythonmetadata "github.com/harness/gitness/registry/app/metadata/python"
	"github.com/harness/gitness/registry/app/pkg"
	"github.com/harness/gitness/registry/app/pkg/base"
	"github.com/harness/gitness/registry/app/pkg/commons"
	pythontype "github.com/harness/gitness/registry/app/pkg/types/python"
	registryrefcache "github.com/harness/gitness/registry/app/services/refcache"
	"github.com/harness/gitness/registry/app/storage"
	"github.com/harness/gitness/registry/app/store"
	request2 "github.com/harness/gitness/registry/request"
	"github.com/harness/gitness/registry/types"
	gtypes "github.com/harness/gitness/types"

	"github.com/stretchr/testify/require"
)

type rpfArtifactRepo struct {
	store.ArtifactRepository
	artifact  *types.Artifact
	artifacts []types.Artifact
	err       error
	listErr   error
}

func (r *rpfArtifactRepo) GetByRegistryImageAndVersion(
	context.Context, int64, string, string,
) (*types.Artifact, error) {
	return r.artifact, r.err
}

func (r *rpfArtifactRepo) GetByRegistryIDAndImage(context.Context, int64, string) (*[]types.Artifact, error) {
	if r.listErr != nil {
		return nil, r.listErr
	}
	return &r.artifacts, nil
}

type rpfURLProvider struct {
	urlprovider.Provider
	err error
}

func (p rpfURLProvider) PackagePathFor(_ context.Context, spec urlprovider.PackagePathSpec) (string, error) {
	if p.err != nil {
		return "", p.err
	}
	py, ok := spec.(urlprovider.PythonFilePathSpec)
	if !ok {
		return "", errors.New("unexpected spec")
	}
	return "files/" + py.Version + "/" + py.Filename, nil
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

func rpfAuthedContext() context.Context {
	return request.WithAuthSession(context.Background(), &auth.Session{Principal: gtypes.Principal{ID: 9}})
}

type rpfRegistryFinder struct {
	registryrefcache.RegistryFinder
	registry *types.Registry
	err      error
}

func (f *rpfRegistryFinder) FindByRootParentID(
	context.Context, int64, string, ...types.QueryOption,
) (*types.Registry, error) {
	return f.registry, f.err
}

type rpfLocalBase struct {
	base.LocalBase
	err        error
	uploaded   []string
	uploadPath string
}

func (b *rpfLocalBase) Download(
	_ context.Context, _ pkg.ArtifactInfo, version string, fileName string,
) (*commons.ResponseHeaders, *storage.FileReader, string, error) {
	if b.err != nil {
		return nil, nil, "", b.err
	}
	return &commons.ResponseHeaders{Code: http.StatusOK}, nil, "redirect:" + version + "/" + fileName, nil
}

func (b *rpfLocalBase) Upload(
	_ context.Context, _ pkg.ArtifactInfo, fileName, _, path string, file io.ReadCloser, _ metadata.Metadata,
) (*commons.ResponseHeaders, string, error) {
	_, _ = io.Copy(io.Discard, file)
	b.uploaded = append(b.uploaded, fileName)
	b.uploadPath = path
	return &commons.ResponseHeaders{Code: http.StatusCreated}, "sum", b.err
}

func (b *rpfLocalBase) UploadFile(
	_ context.Context, _ pkg.ArtifactInfo, fileName, _, path string, _ multipart.File, _ metadata.Metadata,
) (*commons.ResponseHeaders, string, error) {
	b.uploaded = append(b.uploaded, fileName)
	b.uploadPath = path
	return &commons.ResponseHeaders{Code: http.StatusCreated}, "sum", b.err
}

func rpfPyInfo() pythontype.ArtifactInfo {
	return pythontype.ArtifactInfo{
		ArtifactInfo: pkg.ArtifactInfo{
			BaseInfo:      &pkg.BaseInfo{ParentID: 1, RootIdentifier: "root", RootParentID: 2},
			RegIdentifier: "reg",
			RegistryID:    3,
			Image:         "demo",
		},
		Version:  "1.0",
		Filename: "demo-1.0-py3-none-any.whl",
		Metadata: pythonmetadata.Metadata{Version: "1.0"},
	}
}

func rpfNewLocal(
	t *testing.T, lb *rpfLocalBase, artifacts *rpfArtifactRepo, finder *rpfRegistryFinder, urls rpfURLProvider,
) (*localRegistry, *rpfProducer) {
	t.Helper()
	reporter, producer := rpfArtifactReporter(t)
	reg := NewLocalRegistry(lb, nil, nil, nil, nil, finder, nil, artifacts, urls, reporter)
	local, ok := reg.(*localRegistry)
	require.True(t, ok)
	return local, producer
}

func rpfStoredFiles(t *testing.T, requiresPython string, files ...string) json.RawMessage {
	t.Helper()
	m := pythonmetadata.PythonMetadata{Metadata: pythonmetadata.Metadata{RequiresPython: requiresPython}}
	for _, f := range files {
		m.Files = append(m.Files, metadata.File{Filename: f})
	}
	raw, err := json.Marshal(m)
	require.NoError(t, err)
	return raw
}

func TestRpfLocalDownloadPackageFile(t *testing.T) {
	ctx := context.Background()
	info := rpfPyInfo()
	art := &types.Artifact{Version: "1.0"}

	l, _ := rpfNewLocal(t, &rpfLocalBase{}, &rpfArtifactRepo{artifact: art}, nil, rpfURLProvider{})
	headers, reader, rc, redirect, err := l.DownloadPackageFile(ctx, info)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, headers.Code)
	require.Nil(t, reader)
	require.Nil(t, rc)
	require.Equal(t, "redirect:1.0/demo-1.0-py3-none-any.whl", redirect)

	l, _ = rpfNewLocal(t, &rpfLocalBase{}, &rpfArtifactRepo{err: errors.New("gone")}, nil, rpfURLProvider{})
	_, _, _, _, err = l.DownloadPackageFile(ctx, info)
	require.ErrorContains(t, err, "artifact not found or deleted")

	l, _ = rpfNewLocal(t, &rpfLocalBase{err: errors.New("x")}, &rpfArtifactRepo{artifact: art}, nil, rpfURLProvider{})
	_, _, _, _, err = l.DownloadPackageFile(ctx, info)
	require.Error(t, err)

	require.Equal(t, artifact.RegistryTypeVIRTUAL, l.GetArtifactType())
	require.Equal(t, []artifact.PackageType{artifact.PackageTypePYTHON}, l.GetPackageTypes())
}

func TestRpfLocalGetPackageMetadata(t *testing.T) {
	ctx := context.Background()
	info := rpfPyInfo()
	finder := &rpfRegistryFinder{registry: &types.Registry{ID: 3}}

	t.Run("files are sorted by numeric version", func(t *testing.T) {
		repo := &rpfArtifactRepo{artifacts: []types.Artifact{
			{Version: "10.0", Metadata: rpfStoredFiles(t, ">=3.8", "demo-10.0.tar.gz")},
			{Version: "2.0", Metadata: rpfStoredFiles(t, "", "demo-2.0.tar.gz", "demo-2.0-py3-none-any.whl")},
			{Version: "1.0", Metadata: rpfStoredFiles(t, "", "demo-1.0.tar.gz")},
		}}
		l, _ := rpfNewLocal(t, nil, repo, finder, rpfURLProvider{})
		got, err := l.GetPackageMetadata(ctx, info)
		require.NoError(t, err)
		require.Equal(t, "demo", got.Name)
		names := make([]string, 0, len(got.Files))
		for _, f := range got.Files {
			names = append(names, f.Name)
		}
		require.Equal(t, []string{
			"demo-1.0.tar.gz", "demo-2.0-py3-none-any.whl", "demo-2.0.tar.gz", "demo-10.0.tar.gz",
		}, names)
		require.Equal(t, "files/1.0/demo-1.0.tar.gz", got.Files[0].FileURL)
		require.Equal(t, ">=3.8", got.Files[3].RequiresPython)
	})

	t.Run("failures", func(t *testing.T) {
		l, _ := rpfNewLocal(t, nil, &rpfArtifactRepo{}, &rpfRegistryFinder{err: errors.New("no reg")}, rpfURLProvider{})
		_, err := l.GetPackageMetadata(ctx, info)
		require.ErrorContains(t, err, "no reg")

		l, _ = rpfNewLocal(t, nil, &rpfArtifactRepo{listErr: errors.New("db")}, finder, rpfURLProvider{})
		_, err = l.GetPackageMetadata(ctx, info)
		require.ErrorContains(t, err, "db")

		l, _ = rpfNewLocal(t, nil, &rpfArtifactRepo{}, finder, rpfURLProvider{})
		_, err = l.GetPackageMetadata(ctx, info)
		require.ErrorContains(t, err, "no artifacts found")

		bad := &rpfArtifactRepo{artifacts: []types.Artifact{{Version: "1.0", Metadata: []byte("{")}}}
		l, _ = rpfNewLocal(t, nil, bad, finder, rpfURLProvider{})
		_, err = l.GetPackageMetadata(ctx, info)
		require.Error(t, err)

		good := &rpfArtifactRepo{artifacts: []types.Artifact{{Version: "1.0", Metadata: rpfStoredFiles(t, "", "a.whl")}}}
		l, _ = rpfNewLocal(t, nil, good, finder, rpfURLProvider{err: errors.New("no url")})
		_, err = l.GetPackageMetadata(ctx, info)
		require.ErrorContains(t, err, "no url")
	})
}

func TestRpfParseVersion(t *testing.T) {
	ctx := context.Background()
	require.Equal(t, []int{1, 2, 3}, parseVersion(ctx, "1.2.3"))
	require.Equal(t, []int{1, 0, 4}, parseVersion(ctx, "1.rc.4"))
	require.Equal(t, []int{12}, parseVersion(ctx, "12rc1"))
}

func TestRpfLocalUpload(t *testing.T) {
	ctx := rpfAuthedContext()
	info := rpfPyInfo()

	t.Run("reader upload publishes an event", func(t *testing.T) {
		lb := &rpfLocalBase{}
		l, producer := rpfNewLocal(t, lb, nil, nil, rpfURLProvider{})
		headers, sum, err := l.UploadPackageFileReader(ctx, info, io.NopCloser(bytes.NewBufferString("x")), "demo.whl")
		require.NoError(t, err)
		require.Equal(t, http.StatusCreated, headers.Code)
		require.Equal(t, "sum", sum)
		require.Equal(t, "demo/1.0/demo.whl", lb.uploadPath)
		require.Len(t, producer.streams, 1)
	})

	t.Run("multipart upload publishes an event", func(t *testing.T) {
		lb := &rpfLocalBase{}
		l, producer := rpfNewLocal(t, lb, nil, nil, rpfURLProvider{})
		_, _, err := l.UploadPackageFile(ctx, info, nil, "demo.whl")
		require.NoError(t, err)
		require.Equal(t, []string{"demo.whl"}, lb.uploaded)
		require.Len(t, producer.streams, 1)
	})

	t.Run("failures publish nothing", func(t *testing.T) {
		l, producer := rpfNewLocal(t, &rpfLocalBase{err: errors.New("x")}, nil, nil, rpfURLProvider{})
		_, _, err := l.UploadPackageFileReader(ctx, info, io.NopCloser(bytes.NewBufferString("x")), "demo.whl")
		require.Error(t, err)
		_, _, err = l.UploadPackageFile(ctx, info, nil, "demo.whl")
		require.Error(t, err)
		require.Empty(t, producer.streams)
	})
}

type rpfFakeLocalHelper struct {
	exists      bool
	downloadErr error
	uploaded    chan pythontype.ArtifactInfo
}

func (h *rpfFakeLocalHelper) FileExists(context.Context, pythontype.ArtifactInfo) bool {
	return h.exists
}

func (h *rpfFakeLocalHelper) DownloadFile(
	context.Context, pythontype.ArtifactInfo,
) (*commons.ResponseHeaders, *storage.FileReader, string, error) {
	if h.downloadErr != nil {
		return nil, nil, "", h.downloadErr
	}
	return &commons.ResponseHeaders{Code: http.StatusOK}, nil, "local-redirect", nil
}

func (h *rpfFakeLocalHelper) UploadPackageFile(
	_ context.Context, info pythontype.ArtifactInfo, file io.ReadCloser, _ string,
) (*commons.ResponseHeaders, string, error) {
	_, _ = io.Copy(io.Discard, file)
	if h.uploaded != nil {
		h.uploaded <- info
	}
	return nil, "sha", nil
}

const rpfSimpleIndex = `<!DOCTYPE html><html><head><meta name="pypi:repository-version" content="1.0">` +
	`<title>Links for demo</title></head><body>` +
	`<a href="../../packages/demo-1.0-py3-none-any.whl#sha256=abc">demo-1.0-py3-none-any.whl</a>` +
	`<a href="../../packages/demo-2.0.tar.gz" data-requires-python="&gt;=3.8">demo-2.0.tar.gz</a>` +
	`</body></html>`

func rpfPyUpstream(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/simple/demo/", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, rpfSimpleIndex)
	})
	mux.HandleFunc("/packages/demo-1.0-py3-none-any.whl", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, "wheel-bytes")
	})
	mux.HandleFunc("/pypi/demo/1.0/json", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprint(w, `{"info":{"name":"demo","version":"1.0","summary":"a demo"}}`)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func rpfNewProxy(repoURL string, helper *rpfFakeLocalHelper, storeErr error, urls rpfURLProvider) *proxy {
	up := &types.UpstreamProxy{
		RepoKey: "up", RepoURL: repoURL, RepoAuthType: string(artifact.AuthTypeAnonymous),
		PackageType: artifact.PackageTypePYTHON,
	}
	p := NewProxy(nil, &rpfProxyStore{proxy: up, err: storeErr}, nil, nil, nil, nil, urls,
		refcache.SpaceFinder{}, nil, helper)
	px, ok := p.(*proxy)
	if !ok {
		return nil
	}
	return px
}

func TestRpfProxyGetPackageMetadata(t *testing.T) {
	ctx := context.Background()
	srv := rpfPyUpstream(t)
	info := rpfPyInfo()

	p := rpfNewProxy(srv.URL, &rpfFakeLocalHelper{}, nil, rpfURLProvider{})
	require.Equal(t, artifact.RegistryTypeUPSTREAM, p.GetArtifactType())
	require.Equal(t, []artifact.PackageType{artifact.PackageTypePYTHON}, p.GetPackageTypes())

	got, err := p.GetPackageMetadata(ctx, info)
	require.NoError(t, err)
	require.Equal(t, "demo", got.Name)
	require.Len(t, got.Files, 2)
	require.Equal(t, "demo-1.0-py3-none-any.whl", got.Files[0].Name)
	require.Equal(t, "files/1.0/demo-1.0-py3-none-any.whl", got.Files[0].FileURL)
	require.Equal(t, ">=3.8", got.Files[1].RequiresPython)

	_, err = rpfNewProxy(srv.URL, &rpfFakeLocalHelper{}, errors.New("db"), rpfURLProvider{}).
		GetPackageMetadata(ctx, info)
	require.Error(t, err)

	_, err = rpfNewProxy(srv.URL, &rpfFakeLocalHelper{}, nil, rpfURLProvider{err: errors.New("no url")}).
		GetPackageMetadata(ctx, info)
	require.ErrorContains(t, err, "no url")

	missing := info
	missing.Image = "missing"
	_, err = p.GetPackageMetadata(ctx, missing)
	require.Error(t, err)

	_, _, err = p.UploadPackageFile(ctx, info, nil, "x")
	require.Error(t, err)
	_, _, err = p.UploadPackageFileReader(ctx, info, nil, "x")
	require.Error(t, err)
}

func TestRpfProxyDownloadPackageFile(t *testing.T) {
	srv := rpfPyUpstream(t)
	info := rpfPyInfo()

	t.Run("streams from upstream and caches", func(t *testing.T) {
		helper := &rpfFakeLocalHelper{uploaded: make(chan pythontype.ArtifactInfo, 1)}
		p := rpfNewProxy(srv.URL, helper, nil, rpfURLProvider{})
		ctx := request2.WithArtifactInfo(context.Background(), &info)
		_, _, rc, _, err := p.DownloadPackageFile(ctx, info)
		require.NoError(t, err)
		data, err := io.ReadAll(rc)
		require.NoError(t, err)
		require.Equal(t, "wheel-bytes", string(data))
		select {
		case cached := <-helper.uploaded:
			require.Equal(t, "demo-1.0-py3-none-any.whl", cached.Filename)
			require.Equal(t, "a demo", cached.Metadata.Summary)
		case <-time.After(10 * time.Second):
			t.Fatal("wheel was not cached locally")
		}
	})

	t.Run("local copy is preferred", func(t *testing.T) {
		p := rpfNewProxy(srv.URL, &rpfFakeLocalHelper{exists: true}, nil, rpfURLProvider{})
		_, _, _, redirect, err := p.DownloadPackageFile(context.Background(), info)
		require.NoError(t, err)
		require.Equal(t, "local-redirect", redirect)
	})

	t.Run("broken local copy falls back to upstream", func(t *testing.T) {
		helper := &rpfFakeLocalHelper{exists: true, downloadErr: errors.New("corrupt")}
		p := rpfNewProxy(srv.URL, helper, nil, rpfURLProvider{})
		// No artifact info in the context, so the background caching step bails out.
		_, _, rc, _, err := p.DownloadPackageFile(context.Background(), info)
		require.NoError(t, err)
		require.NotNil(t, rc)
	})

	t.Run("failures", func(t *testing.T) {
		_, _, _, _, err := rpfNewProxy(srv.URL, &rpfFakeLocalHelper{}, errors.New("db"), rpfURLProvider{}).
			DownloadPackageFile(context.Background(), info)
		require.Error(t, err)

		missing := info
		missing.Filename = "demo-9.9.tar.gz"
		_, _, _, _, err = rpfNewProxy(srv.URL, &rpfFakeLocalHelper{}, nil, rpfURLProvider{}).
			DownloadPackageFile(context.Background(), missing)
		require.Error(t, err)
	})
}

func TestRpfProxyPutFileToLocal(t *testing.T) {
	srv := rpfPyUpstream(t)
	remote, err := NewRemoteRegistryHelper(context.Background(), refcache.SpaceFinder{}, types.UpstreamProxy{
		RepoURL: srv.URL, RepoAuthType: string(artifact.AuthTypeAnonymous), PackageType: artifact.PackageTypePYTHON,
	}, nil)
	require.NoError(t, err)
	p := rpfNewProxy(srv.URL, &rpfFakeLocalHelper{}, nil, rpfURLProvider{})

	// Metadata cannot be fetched (version unknown), so a minimal one is synthesised.
	info := rpfPyInfo()
	ctx := request2.WithArtifactInfo(context.Background(), &info)
	require.NoError(t, p.putFileToLocal(ctx, "demo", "demo-1.0-py3-none-any.whl", remote))

	require.Error(t, p.putFileToLocal(context.Background(), "demo", "demo-1.0-py3-none-any.whl", remote))
	require.Error(t, p.putFileToLocal(ctx, "demo", "demo-9.9.tar.gz", remote))
}
