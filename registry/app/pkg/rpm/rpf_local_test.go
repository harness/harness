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

package rpm

import (
	"bytes"
	"context"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/harness/gitness/app/services/refcache"
	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	storagedriver "github.com/harness/gitness/registry/app/driver"
	"github.com/harness/gitness/registry/app/pkg"
	"github.com/harness/gitness/registry/app/pkg/base"
	"github.com/harness/gitness/registry/app/pkg/commons"
	"github.com/harness/gitness/registry/app/pkg/filemanager"
	rpmtype "github.com/harness/gitness/registry/app/pkg/types/rpm"
	"github.com/harness/gitness/registry/app/storage"
	"github.com/harness/gitness/registry/app/store"
	"github.com/harness/gitness/registry/types"

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
	err      error
	gotImage string
	gotVer   string
}

func (r *rpfArtifactRepo) GetByRegistryImageAndVersion(
	_ context.Context, _ int64, image string, version string,
) (*types.Artifact, error) {
	r.gotImage, r.gotVer = image, version
	return &types.Artifact{}, r.err
}

type rpfProxyStore struct {
	store.UpstreamProxyConfigRepository
	proxy *types.UpstreamProxy
	err   error
}

func (s *rpfProxyStore) Get(context.Context, int64, ...types.QueryOption) (*types.UpstreamProxy, error) {
	return s.proxy, s.err
}

type rpfLocalBase struct {
	base.LocalBase
	err         error
	gotVersion  string
	gotFileName string
}

func (b *rpfLocalBase) Download(
	_ context.Context, _ pkg.ArtifactInfo, version string, fileName string,
) (*commons.ResponseHeaders, *storage.FileReader, string, error) {
	b.gotVersion, b.gotFileName = version, fileName
	if b.err != nil {
		return nil, nil, "", b.err
	}
	return &commons.ResponseHeaders{Code: http.StatusOK}, nil, "redirect", nil
}

type rpfFileManager struct {
	filemanager.FileManager
	byPathErr   error
	uploadErr   error
	downloadErr error
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
	return rpfFileReader(ctx, f.t, []byte("repodata")), 8, "https://redirect.test/x", nil
}

func (f *rpfFileManager) UploadFileNoDBUpdate(
	context.Context, string, multipart.File, io.Reader, int64, int64,
) (types.FileInfo, error) {
	return types.FileInfo{}, f.uploadErr
}

func (f *rpfFileManager) DownloadFileByDigest(
	ctx context.Context, _ string, _ types.FileInfo, _ int64, _ int64,
) (*storage.FileReader, error) {
	if f.downloadErr != nil {
		return nil, f.downloadErr
	}
	return rpfFileReader(ctx, f.t, []byte("this is not an rpm")), nil
}

func rpfRpmInfo(version string) rpmtype.ArtifactInfo {
	return rpmtype.ArtifactInfo{
		ArtifactInfo: pkg.ArtifactInfo{
			BaseInfo:      &pkg.BaseInfo{ParentID: 1, RootIdentifier: "root"},
			RegIdentifier: "reg",
			RegistryID:    3,
			Image:         "demo",
		},
		Version:     version,
		Arch:        "x86_64",
		FileName:    "demo-1.0-1.x86_64.rpm",
		PackagePath: "Packages/demo-1.0-1.x86_64.rpm",
	}
}

func TestRpfDownloadPackageFile(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name        string
		version     string
		wantVersion string
	}{
		{"without epoch", "1.0-1.x86_64", "1.0-1/x86_64"},
		{"with epoch", "2:1.0-1.x86_64", "1.0-1/x86_64/2"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			lb := &rpfLocalBase{}
			artifacts := &rpfArtifactRepo{}
			headers, _, rc, redirect, err := downloadPackageFile(ctx, rpfRpmInfo(tc.version), lb, artifacts)
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, headers.Code)
			require.Nil(t, rc)
			require.Equal(t, "redirect", redirect)
			require.Equal(t, tc.wantVersion, lb.gotVersion)
			require.Equal(t, "demo-1.0-1.x86_64.rpm", lb.gotFileName)
			require.Equal(t, tc.version, artifacts.gotVer)
		})
	}

	_, _, _, _, err := downloadPackageFile(ctx, rpfRpmInfo("noseparator"), &rpfLocalBase{}, &rpfArtifactRepo{})
	require.ErrorContains(t, err, "invalid version")

	gone := &rpfArtifactRepo{err: errors.New("gone")}
	_, _, _, _, err = downloadPackageFile(ctx, rpfRpmInfo("1.0.x86_64"), &rpfLocalBase{}, gone)
	require.ErrorContains(t, err, "artifact not found or deleted")

	failing := &rpfLocalBase{err: errors.New("x")}
	_, _, _, _, err = downloadPackageFile(ctx, rpfRpmInfo("1.0.x86_64"), failing, &rpfArtifactRepo{})
	require.Error(t, err)
}

func TestRpfGetRepoData(t *testing.T) {
	ctx := context.Background()
	fm := &rpfFileManager{t: t}
	headers, reader, rc, redirect, err := getRepoData(ctx, rpfRpmInfo("1.0.x86_64"), "repomd.xml", fm)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, headers.Code)
	require.NotNil(t, reader)
	require.Nil(t, rc)
	require.Equal(t, "https://redirect.test/x", redirect)
	require.Equal(t, "/repodata/repomd.xml", fm.gotPath)

	failingFM := &rpfFileManager{byPathErr: errors.New("x")}
	headers, _, _, _, err = getRepoData(ctx, rpfRpmInfo("1.0.x86_64"), "repomd.xml", failingFM)
	require.Error(t, err)
	require.Zero(t, headers.Code)
}

type rpfFakeRegistryHelper struct {
	called chan struct{}
	err    error
}

func (h *rpfFakeRegistryHelper) UploadPackage(
	_ context.Context, _ rpmtype.ArtifactInfo, file io.Reader,
) (*commons.ResponseHeaders, string, error) {
	_, _ = io.Copy(io.Discard, file)
	if h.called != nil {
		close(h.called)
	}
	return nil, "sha", h.err
}

func TestRpfLocalRegistry(t *testing.T) {
	ctx := context.Background()
	info := rpfRpmInfo("1.0.x86_64")
	helper := &rpfFakeRegistryHelper{}
	fm := &rpfFileManager{t: t}
	lb := &rpfLocalBase{}
	l := NewLocalRegistry(lb, fm, nil, nil, nil, nil, &rpfArtifactRepo{}, nil, helper)

	require.Equal(t, artifact.RegistryTypeVIRTUAL, l.GetArtifactType())
	require.Equal(t, []artifact.PackageType{artifact.PackageTypeRPM}, l.GetPackageTypes())

	_, sha, err := l.UploadPackageFile(ctx, info, strings.NewReader("x"), "demo.rpm")
	require.NoError(t, err)
	require.Equal(t, "sha", sha)

	_, _, _, redirect, err := l.GetRepoData(ctx, info, "repomd.xml")
	require.NoError(t, err)
	require.Equal(t, "https://redirect.test/x", redirect)

	_, _, _, redirect, err = l.DownloadPackageFile(ctx, info)
	require.NoError(t, err)
	require.Equal(t, "redirect", redirect)
}

func rpfRpmUpstream(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/Packages/demo-1.0-1.x86_64.rpm", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, "rpm-bytes")
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func rpfNewProxy(repoURL string, lb *rpfLocalBase, artifacts *rpfArtifactRepo, helper *rpfFakeRegistryHelper,
	storeErr error) Proxy {
	up := &types.UpstreamProxy{
		RepoKey: "up", RepoURL: repoURL, RepoAuthType: string(artifact.AuthTypeAnonymous),
		PackageType: artifact.PackageTypeRPM,
	}
	fm := &rpfFileManager{byPathErr: errors.New("no repodata")}
	return NewProxy(fm, &rpfProxyStore{proxy: up, err: storeErr}, nil, nil, nil, artifacts, nil,
		lb, helper, refcache.SpaceFinder{}, nil)
}

func TestRpfProxy(t *testing.T) {
	ctx := context.Background()
	srv := rpfRpmUpstream(t)
	info := rpfRpmInfo("1.0.x86_64")

	t.Run("static", func(t *testing.T) {
		p := rpfNewProxy(srv.URL, &rpfLocalBase{}, &rpfArtifactRepo{}, &rpfFakeRegistryHelper{}, nil)
		require.Equal(t, artifact.RegistryTypeUPSTREAM, p.GetArtifactType())
		require.Equal(t, []artifact.PackageType{artifact.PackageTypeRPM}, p.GetPackageTypes())
		_, _, err := p.UploadPackageFile(ctx, info, nil, "x")
		require.Error(t, err)
	})

	t.Run("local copy is preferred", func(t *testing.T) {
		p := rpfNewProxy(srv.URL, &rpfLocalBase{}, &rpfArtifactRepo{}, &rpfFakeRegistryHelper{}, nil)
		_, _, _, redirect, err := p.DownloadPackageFile(ctx, info)
		require.NoError(t, err)
		require.Equal(t, "redirect", redirect)
	})

	t.Run("missing locally is streamed from upstream and cached", func(t *testing.T) {
		helper := &rpfFakeRegistryHelper{called: make(chan struct{})}
		p := rpfNewProxy(srv.URL, &rpfLocalBase{}, &rpfArtifactRepo{err: errors.New("gone")}, helper, nil)
		headers, _, rc, _, err := p.DownloadPackageFile(ctx, info)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, headers.Code)
		data, err := io.ReadAll(rc)
		require.NoError(t, err)
		require.Equal(t, "rpm-bytes", string(data))
		select {
		case <-helper.called:
		case <-time.After(10 * time.Second):
			t.Fatal("package was not cached locally")
		}
	})

	t.Run("failures", func(t *testing.T) {
		gone := &rpfArtifactRepo{err: errors.New("gone")}
		noPath := info
		noPath.PackagePath = ""
		p := rpfNewProxy(srv.URL, &rpfLocalBase{}, gone, &rpfFakeRegistryHelper{}, nil)
		_, _, _, _, err := p.DownloadPackageFile(ctx, noPath)
		require.Error(t, err)

		p = rpfNewProxy(srv.URL, &rpfLocalBase{}, gone, &rpfFakeRegistryHelper{}, errors.New("db"))
		_, _, _, _, err = p.DownloadPackageFile(ctx, info)
		require.Error(t, err)

		missing := info
		missing.PackagePath = "Packages/missing.rpm"
		p = rpfNewProxy(srv.URL, &rpfLocalBase{}, gone, &rpfFakeRegistryHelper{}, nil)
		_, _, _, _, err = p.DownloadPackageFile(ctx, missing)
		require.Error(t, err)
	})

	t.Run("repo data comes from the local file manager", func(t *testing.T) {
		p := rpfNewProxy(srv.URL, &rpfLocalBase{}, &rpfArtifactRepo{}, &rpfFakeRegistryHelper{}, nil)
		_, _, _, _, err := p.GetRepoData(ctx, info, "repomd.xml")
		require.Error(t, err)
	})
}

func TestRpfRegistryHelperUploadFailures(t *testing.T) {
	ctx := context.Background()
	info := rpfRpmInfo("1.0.x86_64")

	h := NewRegistryHelper(nil, &rpfFileManager{t: t, uploadErr: errors.New("x")}, nil)
	_, _, err := h.UploadPackage(ctx, info, strings.NewReader("x"))
	require.Error(t, err)

	h = NewRegistryHelper(nil, &rpfFileManager{t: t, downloadErr: errors.New("x")}, nil)
	_, _, err = h.UploadPackage(ctx, info, strings.NewReader("x"))
	require.Error(t, err)

	// The downloaded content is not a valid rpm.
	h = NewRegistryHelper(nil, &rpfFileManager{t: t}, nil)
	_, _, err = h.UploadPackage(ctx, info, strings.NewReader("x"))
	require.Error(t, err)
}
