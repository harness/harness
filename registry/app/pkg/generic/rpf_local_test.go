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

package generic

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/harness/gitness/app/services/refcache"
	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/registry/app/metadata"
	"github.com/harness/gitness/registry/app/pkg"
	"github.com/harness/gitness/registry/app/pkg/base"
	"github.com/harness/gitness/registry/app/pkg/commons"
	generictype "github.com/harness/gitness/registry/app/pkg/types/generic"
	"github.com/harness/gitness/registry/app/storage"
	"github.com/harness/gitness/registry/app/store"
	"github.com/harness/gitness/registry/types"

	"github.com/stretchr/testify/require"
)

type rpfArtifactRepo struct {
	store.ArtifactRepository
	err error
}

func (r *rpfArtifactRepo) GetByRegistryImageAndVersion(
	context.Context, int64, string, string,
) (*types.Artifact, error) {
	return &types.Artifact{}, r.err
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

type rpfLocalBase struct {
	base.LocalBase
	err     error
	calls   []string
	gotPath string
}

func (b *rpfLocalBase) Upload(
	_ context.Context, _ pkg.ArtifactInfo, _, _, path string, _ io.ReadCloser, _ metadata.Metadata,
) (*commons.ResponseHeaders, string, error) {
	b.calls = append(b.calls, "upload")
	b.gotPath = path
	return &commons.ResponseHeaders{Code: http.StatusCreated}, "sum", b.err
}

func (b *rpfLocalBase) UploadRawFile(
	_ context.Context, _ pkg.ArtifactInfo, filePath string, _ io.ReadCloser, _ bool,
) (*commons.ResponseHeaders, string, error) {
	b.calls = append(b.calls, "uploadRaw")
	b.gotPath = filePath
	return &commons.ResponseHeaders{Code: http.StatusCreated}, "raw-sum", b.err
}

func (b *rpfLocalBase) Download(
	_ context.Context, _ pkg.ArtifactInfo, _ string, fileName string,
) (*commons.ResponseHeaders, *storage.FileReader, string, error) {
	b.calls = append(b.calls, "download")
	b.gotPath = fileName
	return &commons.ResponseHeaders{Code: http.StatusOK}, nil, "redirect", b.err
}

func (b *rpfLocalBase) DownloadRawFile(
	_ context.Context, _ pkg.ArtifactInfo, filePath string,
) (*commons.ResponseHeaders, *storage.FileReader, string, error) {
	b.calls = append(b.calls, "downloadRaw")
	b.gotPath = filePath
	return &commons.ResponseHeaders{Code: http.StatusOK}, nil, "raw-redirect", b.err
}

func (b *rpfLocalBase) DeleteFile(
	_ context.Context, _ pkg.PackageArtifactInfo, filePath string,
) (*commons.ResponseHeaders, error) {
	b.calls = append(b.calls, "delete")
	b.gotPath = filePath
	return &commons.ResponseHeaders{Code: http.StatusNoContent}, b.err
}

func (b *rpfLocalBase) ExistsE(
	_ context.Context, _ pkg.PackageArtifactInfo, path string,
) (*commons.ResponseHeaders, error) {
	b.calls = append(b.calls, "exists")
	b.gotPath = path
	return &commons.ResponseHeaders{Code: http.StatusOK}, b.err
}

func rpfGenericInfo(pkgType artifact.PackageType) generictype.ArtifactInfo {
	return generictype.ArtifactInfo{
		ArtifactInfo: pkg.ArtifactInfo{
			BaseInfo:      &pkg.BaseInfo{ParentID: 1, RootIdentifier: "root"},
			RegIdentifier: "reg",
			RegistryID:    3,
			Image:         "demo",
			Registry:      types.Registry{PackageType: pkgType},
		},
		Version:  "1.0",
		FileName: "a.txt",
		FilePath: "dir/a.txt",
	}
}

func rpfNewLocal(lb *rpfLocalBase, artifacts *rpfArtifactRepo) *localRegistry {
	reg := NewLocalRegistry(lb, nil, nil, nil, nil, nil, artifacts, nil)
	local, ok := reg.(*localRegistry)
	if !ok {
		return nil
	}
	return local
}

func TestRpfLocalPutFile(t *testing.T) {
	ctx := context.Background()
	body := func() io.ReadCloser { return io.NopCloser(strings.NewReader("x")) }

	lb := &rpfLocalBase{}
	l := rpfNewLocal(lb, &rpfArtifactRepo{})
	require.Equal(t, artifact.RegistryTypeVIRTUAL, l.GetArtifactType())
	require.Equal(t, []artifact.PackageType{artifact.PackageTypeGENERIC}, l.GetPackageTypes())

	headers, sum, err := l.PutFile(ctx, rpfGenericInfo(artifact.PackageTypeGENERIC), body(), "text/plain")
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, headers.Code)
	require.Equal(t, "sum", sum)
	require.Equal(t, "demo/1.0/dir/a.txt", lb.gotPath)

	// Other package types are stored as raw files.
	lb = &rpfLocalBase{}
	l = rpfNewLocal(lb, &rpfArtifactRepo{})
	_, sum, err = l.PutFile(ctx, rpfGenericInfo(artifact.PackageTypeHUGGINGFACE), body(), "")
	require.NoError(t, err)
	require.Equal(t, "raw-sum", sum)
	require.Equal(t, []string{"uploadRaw"}, lb.calls)
	require.Equal(t, "dir/a.txt", lb.gotPath)

	l = rpfNewLocal(&rpfLocalBase{err: errors.New("x")}, &rpfArtifactRepo{})
	_, _, err = l.PutFile(ctx, rpfGenericInfo(artifact.PackageTypeGENERIC), body(), "")
	require.ErrorContains(t, err, "failed to upload file")
	_, _, err = l.PutFile(ctx, rpfGenericInfo(artifact.PackageTypeHUGGINGFACE), body(), "")
	require.ErrorContains(t, err, "failed to upload raw file")
}

func TestRpfLocalDownloadFile(t *testing.T) {
	ctx := context.Background()

	lb := &rpfLocalBase{}
	l := rpfNewLocal(lb, &rpfArtifactRepo{})
	headers, _, rc, redirect, err := l.DownloadFile(ctx, rpfGenericInfo(artifact.PackageTypeGENERIC), "dir/a.txt")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, headers.Code)
	require.Nil(t, rc)
	require.Equal(t, "redirect", redirect)
	require.Equal(t, "dir/a.txt", lb.gotPath)

	lb = &rpfLocalBase{}
	l = rpfNewLocal(lb, &rpfArtifactRepo{})
	_, _, _, redirect, err = l.DownloadFile(ctx, rpfGenericInfo(artifact.PackageTypeHUGGINGFACE), "dir/a.txt")
	require.NoError(t, err)
	require.Equal(t, "raw-redirect", redirect)
	require.Equal(t, "/dir/a.txt", lb.gotPath)

	// Without a version the soft-delete check is skipped.
	noVersion := rpfGenericInfo(artifact.PackageTypeGENERIC)
	noVersion.Version = ""
	l = rpfNewLocal(&rpfLocalBase{}, &rpfArtifactRepo{err: errors.New("gone")})
	_, _, _, _, err = l.DownloadFile(ctx, noVersion, "dir/a.txt")
	require.NoError(t, err)
	_, _, _, _, err = l.DownloadFile(ctx, rpfGenericInfo(artifact.PackageTypeGENERIC), "dir/a.txt")
	require.ErrorContains(t, err, "artifact not found or deleted")

	l = rpfNewLocal(&rpfLocalBase{err: errors.New("x")}, &rpfArtifactRepo{})
	_, _, _, _, err = l.DownloadFile(ctx, rpfGenericInfo(artifact.PackageTypeGENERIC), "dir/a.txt")
	require.ErrorContains(t, err, "failed to download file")
	_, _, _, _, err = l.DownloadFile(ctx, rpfGenericInfo(artifact.PackageTypeHUGGINGFACE), "dir/a.txt")
	require.ErrorContains(t, err, "failed to download raw file")
}

func TestRpfLocalDeleteAndHead(t *testing.T) {
	ctx := context.Background()
	lb := &rpfLocalBase{}
	l := rpfNewLocal(lb, &rpfArtifactRepo{})
	info := rpfGenericInfo(artifact.PackageTypeGENERIC)

	headers, err := l.DeleteFile(ctx, info)
	require.NoError(t, err)
	require.Equal(t, http.StatusNoContent, headers.Code)
	require.Equal(t, "dir/a.txt", lb.gotPath)

	headers, err = l.HeadFile(ctx, info, "other/b.txt")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, headers.Code)
	require.Equal(t, "other/b.txt", lb.gotPath)
}

type rpfFakeLocalRegistry struct {
	LocalRegistry
	putType string
}

func (r *rpfFakeLocalRegistry) PutFile(
	_ context.Context, _ generictype.ArtifactInfo, _ io.ReadCloser, contentType string,
) (*commons.ResponseHeaders, string, error) {
	r.putType = contentType
	return &commons.ResponseHeaders{Code: http.StatusCreated}, "sum", nil
}

func (r *rpfFakeLocalRegistry) DeleteFile(
	context.Context, generictype.ArtifactInfo,
) (*commons.ResponseHeaders, error) {
	return &commons.ResponseHeaders{Code: http.StatusNoContent}, nil
}

func TestRpfLocalRegistryHelper(t *testing.T) {
	ctx := context.Background()
	lb := &rpfLocalBase{}
	reg := &rpfFakeLocalRegistry{}
	h := NewLocalRegistryHelper(reg, lb)
	info := rpfGenericInfo(artifact.PackageTypeGENERIC)

	headers, err := h.FileExists(ctx, info)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, headers.Code)
	require.Equal(t, "dir/a.txt", lb.gotPath)

	headers, _, redirect, err := h.DownloadFile(ctx, info)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, headers.Code)
	require.Equal(t, "redirect", redirect)

	headers, sum, err := h.PutFile(ctx, info, nil, "text/plain")
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, headers.Code)
	require.Equal(t, "sum", sum)
	require.Equal(t, "text/plain", reg.putType)

	headers, err = h.DeleteFile(ctx, info)
	require.NoError(t, err)
	require.Equal(t, http.StatusNoContent, headers.Code)
}

type rpfFakeLocalHelper struct {
	existsErr   error
	downloadErr error
	putErr      error
	put         chan string
}

func (h *rpfFakeLocalHelper) FileExists(context.Context, generictype.ArtifactInfo) (*commons.ResponseHeaders, error) {
	return &commons.ResponseHeaders{Code: http.StatusOK, Headers: map[string]string{"X-Local": "1"}}, h.existsErr
}

func (h *rpfFakeLocalHelper) DownloadFile(
	context.Context, generictype.ArtifactInfo,
) (*commons.ResponseHeaders, *storage.FileReader, string, error) {
	if h.downloadErr != nil {
		return nil, nil, "", h.downloadErr
	}
	return &commons.ResponseHeaders{Code: http.StatusOK}, nil, "local-redirect", nil
}

func (h *rpfFakeLocalHelper) PutFile(
	_ context.Context, _ generictype.ArtifactInfo, r io.ReadCloser, _ string,
) (*commons.ResponseHeaders, string, error) {
	data, _ := io.ReadAll(r)
	if h.put != nil {
		h.put <- string(data)
	}
	return nil, "sha", h.putErr
}

func (h *rpfFakeLocalHelper) DeleteFile(context.Context, generictype.ArtifactInfo) (*commons.ResponseHeaders, error) {
	return &commons.ResponseHeaders{Code: http.StatusNoContent}, nil
}

func rpfGenericUpstream(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/dir/a.txt", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Length", "5")
		_, _ = io.WriteString(w, "hello")
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func rpfNewProxy(repoURL string, helper *rpfFakeLocalHelper, storeErr error) *proxy {
	up := &types.UpstreamProxy{
		RepoKey: "up", RepoURL: repoURL, RepoAuthType: string(artifact.AuthTypeAnonymous),
		PackageType: artifact.PackageTypeGENERIC,
	}
	p := NewProxy(nil, &rpfProxyStore{proxy: up, err: storeErr}, nil, nil, nil, nil, nil,
		refcache.SpaceFinder{}, nil, helper)
	px, ok := p.(*proxy)
	if !ok {
		return nil
	}
	return px
}

func TestRpfProxy(t *testing.T) {
	ctx := context.Background()
	srv := rpfGenericUpstream(t)
	info := rpfGenericInfo(artifact.PackageTypeGENERIC)

	t.Run("static", func(t *testing.T) {
		p := rpfNewProxy(srv.URL, &rpfFakeLocalHelper{}, nil)
		require.Equal(t, artifact.RegistryTypeUPSTREAM, p.GetArtifactType())
		require.Equal(t, []artifact.PackageType{artifact.PackageTypeGENERIC}, p.GetPackageTypes())
		_, _, err := p.PutFile(ctx, info, nil, "")
		require.Error(t, err)
		headers, err := p.DeleteFile(ctx, info)
		require.NoError(t, err)
		require.Equal(t, http.StatusNoContent, headers.Code)
	})

	t.Run("download streams from upstream and caches", func(t *testing.T) {
		helper := &rpfFakeLocalHelper{existsErr: errors.New("missing"), put: make(chan string, 1)}
		p := rpfNewProxy(srv.URL, helper, nil)
		headers, _, rc, _, err := p.DownloadFile(ctx, info, "dir/a.txt")
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, headers.Code)
		data, err := io.ReadAll(rc)
		require.NoError(t, err)
		require.Equal(t, "hello", string(data))
		select {
		case cached := <-helper.put:
			require.Equal(t, "hello", cached)
		case <-time.After(10 * time.Second):
			t.Fatal("file was not cached locally")
		}
	})

	t.Run("download prefers the local copy", func(t *testing.T) {
		p := rpfNewProxy(srv.URL, &rpfFakeLocalHelper{}, nil)
		_, _, _, redirect, err := p.DownloadFile(ctx, info, "dir/a.txt")
		require.NoError(t, err)
		require.Equal(t, "local-redirect", redirect)
	})

	t.Run("broken local copy falls back to upstream", func(t *testing.T) {
		helper := &rpfFakeLocalHelper{downloadErr: errors.New("corrupt"), put: make(chan string, 1)}
		p := rpfNewProxy(srv.URL, helper, nil)
		_, _, rc, _, err := p.DownloadFile(ctx, info, "dir/a.txt")
		require.NoError(t, err)
		require.NotNil(t, rc)
		select {
		case <-helper.put:
		case <-time.After(10 * time.Second):
			t.Fatal("file was not cached locally")
		}
	})

	t.Run("download failures", func(t *testing.T) {
		_, _, _, _, err := rpfNewProxy(srv.URL, &rpfFakeLocalHelper{}, errors.New("db")).DownloadFile(ctx, info, "x")
		require.Error(t, err)

		helper := &rpfFakeLocalHelper{existsErr: errors.New("missing")}
		_, _, _, _, err = rpfNewProxy(srv.URL, helper, nil).DownloadFile(ctx, info, "dir/missing.txt")
		require.ErrorContains(t, err, "filepath not found")
	})

	t.Run("head", func(t *testing.T) {
		headers, err := rpfNewProxy(srv.URL, &rpfFakeLocalHelper{}, nil).HeadFile(ctx, info, "dir/a.txt")
		require.NoError(t, err)
		require.Equal(t, "1", headers.Headers["X-Local"])

		missing := &rpfFakeLocalHelper{existsErr: errors.New("missing")}
		headers, err = rpfNewProxy(srv.URL, missing, nil).HeadFile(ctx, info, "dir/a.txt")
		require.NoError(t, err)
		require.NotNil(t, headers)

		_, err = rpfNewProxy(srv.URL, missing, nil).HeadFile(ctx, info, "dir/missing.txt")
		require.ErrorContains(t, err, "HeadFile Failure")

		_, err = rpfNewProxy(srv.URL, missing, errors.New("db")).HeadFile(ctx, info, "dir/a.txt")
		require.Error(t, err)
	})

	t.Run("put to local failures", func(t *testing.T) {
		remote, err := NewRemoteRegistryHelper(ctx, refcache.SpaceFinder{}, types.UpstreamProxy{
			RepoURL: srv.URL, RepoAuthType: string(artifact.AuthTypeAnonymous), PackageType: artifact.PackageTypeGENERIC,
		}, nil)
		require.NoError(t, err)
		p := rpfNewProxy(srv.URL, &rpfFakeLocalHelper{}, nil)
		require.NoError(t, p.putFileToLocal(ctx, info, "dir/a.txt", remote))
		require.Error(t, p.putFileToLocal(ctx, info, "dir/missing.txt", remote))

		p = rpfNewProxy(srv.URL, &rpfFakeLocalHelper{putErr: errors.New("x")}, nil)
		require.Error(t, p.putFileToLocal(ctx, info, "dir/a.txt", remote))
	})
}
