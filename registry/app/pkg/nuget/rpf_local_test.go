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

package nuget

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/gob"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"testing"

	"github.com/harness/gitness/app/api/request"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/events"
	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	storagedriver "github.com/harness/gitness/registry/app/driver"
	registryevents "github.com/harness/gitness/registry/app/events/artifact"
	"github.com/harness/gitness/registry/app/metadata"
	nugetmetadata "github.com/harness/gitness/registry/app/metadata/nuget"
	"github.com/harness/gitness/registry/app/pkg"
	"github.com/harness/gitness/registry/app/pkg/base"
	"github.com/harness/gitness/registry/app/pkg/commons"
	"github.com/harness/gitness/registry/app/pkg/filemanager"
	nugettype "github.com/harness/gitness/registry/app/pkg/types/nuget"
	"github.com/harness/gitness/registry/app/storage"
	"github.com/harness/gitness/registry/app/store"
	"github.com/harness/gitness/registry/types"
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
	artifacts    []types.Artifact
	artifact     *types.Artifact
	searchResult []types.ArtifactMetadata
	count        int64
	err          error
	countErr     error
}

func (r *rpfArtifactRepo) GetByRegistryIDAndImage(context.Context, int64, string) (*[]types.Artifact, error) {
	if r.err != nil {
		return nil, r.err
	}
	return &r.artifacts, nil
}

func (r *rpfArtifactRepo) CountByImageName(context.Context, int64, string) (int64, error) {
	if r.countErr != nil {
		return 0, r.countErr
	}
	return r.count, r.err
}

func (r *rpfArtifactRepo) SearchByImageName(
	context.Context, int64, string, int, int,
) (*[]types.ArtifactMetadata, error) {
	if r.err != nil {
		return nil, r.err
	}
	return &r.searchResult, nil
}

func (r *rpfArtifactRepo) GetByName(context.Context, int64, string, ...types.QueryOption) (*types.Artifact, error) {
	return r.artifact, r.err
}

func (r *rpfArtifactRepo) GetByRegistryImageAndVersion(
	context.Context, int64, string, string,
) (*types.Artifact, error) {
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

type rpfFileManager struct {
	filemanager.FileManager
	uploadErr   error
	downloadErr error
	content     []byte
	findErr     error
	byPathErr   error
	t           *testing.T
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
	return rpfFileReader(ctx, f.t, f.content), nil
}

func (f *rpfFileManager) FindLatestFilePath(context.Context, int64, string, string) (string, error) {
	return "/demo/1.0.0/demo.1.0.0.nupkg", f.findErr
}

func (f *rpfFileManager) DownloadFileByPath(
	ctx context.Context, _ string, _ int64, _ string, _ string, _ bool,
) (*storage.FileReader, int64, string, error) {
	if f.byPathErr != nil {
		return nil, 0, "", f.byPathErr
	}
	return rpfFileReader(ctx, f.t, []byte("content")), 7, "https://redirect.test/x", nil
}

type rpfLocalBase struct {
	base.LocalBase
	versionExists bool
	checkErr      error
	updateErr     error
	deleteErr     error
	gotPath       string
	gotMetadata   metadata.Metadata
	existsPath    string
	existsResult  bool
	downloadCalls []string
}

func (b *rpfLocalBase) CheckIfVersionExists(context.Context, pkg.PackageArtifactInfo) (bool, error) {
	return b.versionExists, b.checkErr
}

func (b *rpfLocalBase) UpdateFileManagerAndCreateArtifact(
	_ context.Context, _ pkg.ArtifactInfo, _, path string, md metadata.Metadata, _ types.FileInfo, _ bool,
) (*commons.ResponseHeaders, string, int64, bool, error) {
	b.gotPath = path
	b.gotMetadata = md
	return &commons.ResponseHeaders{Code: http.StatusCreated}, "sum", 1, false, b.updateErr
}

func (b *rpfLocalBase) DeleteVersion(context.Context, pkg.PackageArtifactInfo) error {
	return b.deleteErr
}

func (b *rpfLocalBase) Exists(_ context.Context, _ pkg.ArtifactInfo, path string) bool {
	b.existsPath = path
	return b.existsResult
}

func (b *rpfLocalBase) Download(
	_ context.Context, _ pkg.ArtifactInfo, version string, fileName string,
) (*commons.ResponseHeaders, *storage.FileReader, string, error) {
	b.downloadCalls = append(b.downloadCalls, version+"/"+fileName)
	return &commons.ResponseHeaders{Code: http.StatusOK}, nil, "redirect", nil
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

func rpfReporter(t *testing.T) (*registryevents.Reporter, *rpfProducer) {
	t.Helper()
	gob.Register(&registryevents.CommonArtifact{})
	producer := &rpfProducer{}
	system, err := events.NewSystem(
		func(string, string) (events.StreamConsumer, error) { return nil, errors.New("unused") },
		producer, rpfCollector{}, false)
	require.NoError(t, err)
	reporter, err := registryevents.NewReporter(system)
	require.NoError(t, err)
	return reporter, producer
}

func rpfNewLocal(
	t *testing.T, lb *rpfLocalBase, fm *rpfFileManager, artifacts *rpfArtifactRepo, images *rpfImageRepo,
) *localRegistry {
	t.Helper()
	reporter, _ := rpfReporter(t)
	if fm != nil {
		fm.t = t
	}
	reg := NewLocalRegistry(lb, fm, nil, nil, nil, images, artifacts, rpfURLProvider{}, reporter)
	local, ok := reg.(*localRegistry)
	require.True(t, ok)
	return local
}

func rpfNupkg(t *testing.T, files [][2]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, f := range files {
		w, err := zw.CreateHeader(&zip.FileHeader{Name: f[0], Method: zip.Deflate})
		require.NoError(t, err)
		_, err = w.Write([]byte(f[1]))
		require.NoError(t, err)
	}
	require.NoError(t, zw.Close())
	return buf.Bytes()
}

func rpfNuspec(id, version string) string {
	return `<?xml version="1.0"?><package><metadata><id>` + id + `</id><version>` + version +
		`</version><authors>alice</authors><description>demo description</description></metadata></package>`
}

func rpfStoredArtifact(t *testing.T) types.Artifact {
	t.Helper()
	m := rpfSampleMetadata()
	raw, err := json.Marshal(m)
	require.NoError(t, err)
	return types.Artifact{Version: "1.0.0", Metadata: raw}
}

func TestRpfLocalStatic(t *testing.T) {
	ctx := context.Background()
	l := rpfNewLocal(t, &rpfLocalBase{}, &rpfFileManager{}, &rpfArtifactRepo{}, &rpfImageRepo{})
	info := rpfProxyInfo()
	require.Equal(t, artifact.RegistryTypeVIRTUAL, l.GetArtifactType())
	require.Equal(t, []artifact.PackageType{artifact.PackageTypeNUGET}, l.GetPackageTypes())
	require.Equal(t, "https://harness.test/pkg/root/reg/nuget/query", l.GetServiceEndpoint(ctx, info).Resources[0].ID)
	require.Equal(t, "https://harness.test/pkg/root/reg/nuget", l.GetServiceEndpointV2(ctx, info).Base)
	require.NotNil(t, l.GetServiceMetadataV2(ctx, info))
}

func TestRpfLocalListAndMetadata(t *testing.T) {
	ctx := context.Background()
	info := rpfProxyInfo()
	art := rpfStoredArtifact(t)
	good := &rpfArtifactRepo{artifacts: []types.Artifact{art, {Version: "2.0.0", Metadata: art.Metadata}}}
	empty := &rpfArtifactRepo{}
	failing := &rpfArtifactRepo{err: errors.New("db down")}

	t.Run("ListPackageVersion", func(t *testing.T) {
		got, err := rpfNewLocal(t, nil, nil, good, nil).ListPackageVersion(ctx, info)
		require.NoError(t, err)
		require.Equal(t, []string{"1.0.0", "2.0.0"}, got.Versions)
		_, err = rpfNewLocal(t, nil, nil, empty, nil).ListPackageVersion(ctx, info)
		require.ErrorContains(t, err, "no artifacts found")
		_, err = rpfNewLocal(t, nil, nil, failing, nil).ListPackageVersion(ctx, info)
		require.ErrorContains(t, err, "failed to get artifacts")
	})

	t.Run("ListPackageVersionV2", func(t *testing.T) {
		feed, err := rpfNewLocal(t, nil, nil, good, nil).ListPackageVersionV2(ctx, info)
		require.NoError(t, err)
		require.Len(t, feed.Entries, 2)
		_, err = rpfNewLocal(t, nil, nil, empty, nil).ListPackageVersionV2(ctx, info)
		require.ErrorContains(t, err, "no artifacts found")
		_, err = rpfNewLocal(t, nil, nil, failing, nil).ListPackageVersionV2(ctx, info)
		require.ErrorContains(t, err, "failed to get artifacts")
	})

	t.Run("GetPackageMetadata", func(t *testing.T) {
		got, err := rpfNewLocal(t, nil, nil, good, nil).GetPackageMetadata(ctx, info)
		require.NoError(t, err)
		resp, ok := got.(*nugettype.RegistrationIndexResponse)
		require.True(t, ok)
		require.Len(t, resp.Pages[0].Items, 2)
		_, err = rpfNewLocal(t, nil, nil, empty, nil).GetPackageMetadata(ctx, info)
		require.ErrorContains(t, err, "no artifacts found")
		_, err = rpfNewLocal(t, nil, nil, failing, nil).GetPackageMetadata(ctx, info)
		require.ErrorContains(t, err, "failed to get artifacts")
	})
}

func TestRpfLocalCountAndSearch(t *testing.T) {
	ctx := context.Background()
	info := rpfProxyInfo()
	meta := rpfMetadataJSON(t, rpfSampleMetadata())
	repo := &rpfArtifactRepo{
		count:        3,
		searchResult: []types.ArtifactMetadata{{Name: "demo", Version: "1.0.0", Metadata: meta}},
	}

	t.Run("counts", func(t *testing.T) {
		l := rpfNewLocal(t, nil, nil, repo, nil)
		n, err := l.CountPackageVersionV2(ctx, info)
		require.NoError(t, err)
		require.Equal(t, int64(3), n)
		n, err = l.CountPackageV2(ctx, info, "DEMO")
		require.NoError(t, err)
		require.Equal(t, int64(3), n)

		bad := rpfNewLocal(t, nil, nil, &rpfArtifactRepo{countErr: errors.New("boom")}, nil)
		_, err = bad.CountPackageVersionV2(ctx, info)
		require.ErrorContains(t, err, "failed to get artifacts count")
		_, err = bad.CountPackageV2(ctx, info, "x")
		require.ErrorContains(t, err, "failed to get artifacts count")
	})

	t.Run("search v2", func(t *testing.T) {
		feed, err := rpfNewLocal(t, nil, nil, repo, nil).SearchPackageV2(ctx, info, "demo", 10, 0)
		require.NoError(t, err)
		require.Len(t, feed.Entries, 1)
		_, err = rpfNewLocal(t, nil, nil, &rpfArtifactRepo{err: errors.New("boom")}, nil).
			SearchPackageV2(ctx, info, "demo", 10, 0)
		require.Error(t, err)
	})

	t.Run("search v3", func(t *testing.T) {
		resp, err := rpfNewLocal(t, nil, nil, repo, nil).SearchPackage(ctx, info, "demo", 10, 0)
		require.NoError(t, err)
		require.Equal(t, int64(3), resp.TotalHits)
		require.Len(t, resp.Data, 1)

		_, err = rpfNewLocal(t, nil, nil, &rpfArtifactRepo{err: errors.New("boom")}, nil).
			SearchPackage(ctx, info, "demo", 10, 0)
		require.ErrorContains(t, err, "failed to get artifacts")

		// The search succeeds but counting the hits fails.
		_, err = rpfNewLocal(t, nil, nil, &rpfArtifactRepo{countErr: errors.New("boom")}, nil).
			SearchPackage(ctx, info, "demo", 10, 0)
		require.ErrorContains(t, err, "failed to get artifacts count")
	})
}

func TestRpfLocalVersionMetadata(t *testing.T) {
	ctx := context.Background()
	info := rpfProxyInfo()
	art := rpfStoredArtifact(t)
	images := &rpfImageRepo{image: &types.Image{ID: 4}}

	t.Run("v2 feed entry", func(t *testing.T) {
		entry, err := rpfNewLocal(t, nil, nil, &rpfArtifactRepo{artifact: &art}, images).
			GetPackageVersionMetadataV2(ctx, info)
		require.NoError(t, err)
		require.Equal(t, "demo", entry.Properties.ID)
	})

	t.Run("v3 leaf", func(t *testing.T) {
		leaf, err := rpfNewLocal(t, nil, nil, &rpfArtifactRepo{artifact: &art}, images).
			GetPackageVersionMetadata(ctx, info)
		require.NoError(t, err)
		require.True(t, leaf.Listed)
	})

	t.Run("missing image and artifact", func(t *testing.T) {
		noImage := &rpfImageRepo{err: errors.New("no image")}
		l := rpfNewLocal(t, nil, nil, &rpfArtifactRepo{artifact: &art}, noImage)
		_, err := l.GetPackageVersionMetadataV2(ctx, info)
		require.ErrorContains(t, err, "failed to get image")
		_, err = l.GetPackageVersionMetadata(ctx, info)
		require.ErrorContains(t, err, "failed to get image")

		noArtifact := rpfNewLocal(t, nil, nil, &rpfArtifactRepo{err: errors.New("no artifact")}, images)
		_, err = noArtifact.GetPackageVersionMetadataV2(ctx, info)
		require.ErrorContains(t, err, "failed to get artifacts")
		_, err = noArtifact.GetPackageVersionMetadata(ctx, info)
		require.ErrorContains(t, err, "failed to get artifacts")
	})
}

func TestRpfLocalGetReadme(t *testing.T) {
	ctx := context.Background()
	info := rpfProxyInfo()
	images := &rpfImageRepo{image: &types.Image{ID: 4}}

	withReadme := rpfSampleMetadata()
	withReadme.PackageMetadata.Readme = "# Demo"
	raw, err := json.Marshal(withReadme)
	require.NoError(t, err)
	got, err := rpfNewLocal(t, nil, nil, &rpfArtifactRepo{artifact: &types.Artifact{Metadata: raw}}, images).
		GetReadme(ctx, info)
	require.NoError(t, err)
	require.Equal(t, "# Demo", got)

	noReadme := rpfStoredArtifact(t)
	_, err = rpfNewLocal(t, nil, nil, &rpfArtifactRepo{artifact: &noReadme}, images).GetReadme(ctx, info)
	require.ErrorContains(t, err, "readme not found")

	_, err = rpfNewLocal(t, nil, nil, &rpfArtifactRepo{artifact: &types.Artifact{Metadata: []byte("{")}}, images).
		GetReadme(ctx, info)
	require.ErrorContains(t, err, "failed to unmarshal metadata")

	_, err = rpfNewLocal(t, nil, nil, &rpfArtifactRepo{artifact: &noReadme}, &rpfImageRepo{err: errors.New("x")}).
		GetReadme(ctx, info)
	require.ErrorContains(t, err, "failed to get image")

	_, err = rpfNewLocal(t, nil, nil, &rpfArtifactRepo{err: errors.New("x")}, images).GetReadme(ctx, info)
	require.ErrorContains(t, err, "failed to get artifact")
}

func TestRpfLocalDownloadPackage(t *testing.T) {
	ctx := context.Background()
	info := rpfProxyInfo()
	art := rpfStoredArtifact(t)

	l := rpfNewLocal(t, nil, &rpfFileManager{}, &rpfArtifactRepo{artifact: &art}, nil)
	headers, reader, redirect, body, err := l.DownloadPackage(ctx, info)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, headers.Code)
	require.Equal(t, "application/octet-stream", headers.Headers["Content-Type"])
	require.Equal(t, "7", headers.Headers["Content-Length"])
	require.NotNil(t, reader)
	require.Equal(t, "https://redirect.test/x", redirect)
	require.Nil(t, body)

	_, _, _, _, err = rpfNewLocal(t, nil, &rpfFileManager{}, &rpfArtifactRepo{err: errors.New("gone")}, nil).
		DownloadPackage(ctx, info)
	require.ErrorContains(t, err, "artifact not found or deleted")

	findFails := &rpfFileManager{findErr: errors.New("x")}
	_, _, _, _, err = rpfNewLocal(t, nil, findFails, &rpfArtifactRepo{artifact: &art}, nil).
		DownloadPackage(ctx, info)
	require.ErrorContains(t, err, "failed to find file node")

	pathFails := &rpfFileManager{byPathErr: errors.New("x")}
	_, _, _, _, err = rpfNewLocal(t, nil, pathFails, &rpfArtifactRepo{artifact: &art}, nil).
		DownloadPackage(ctx, info)
	require.Error(t, err)
}

func TestRpfLocalDeletePackage(t *testing.T) {
	ctx := context.Background()
	info := rpfProxyInfo()
	headers, err := rpfNewLocal(t, &rpfLocalBase{}, nil, nil, nil).DeletePackage(ctx, info)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, headers.Code)

	headers, err = rpfNewLocal(t, &rpfLocalBase{deleteErr: errors.New("x")}, nil, nil, nil).DeletePackage(ctx, info)
	require.ErrorContains(t, err, "failed to delete package version")
	require.Zero(t, headers.Code)
}

func rpfUploadInfo() nugettype.ArtifactInfo {
	info := rpfProxyInfo()
	info.RegistryID = 3
	info.Version = ""
	info.Filename = ""
	return info
}

func rpfAuthedContext() context.Context {
	return request.WithAuthSession(context.Background(), &auth.Session{Principal: gtypes.Principal{ID: 9}})
}

func TestRpfLocalUploadPackage(t *testing.T) {
	ctx := rpfAuthedContext()
	nupkg := rpfNupkg(t, [][2]string{
		{"Demo.Pkg.nuspec", rpfNuspec("Demo.Pkg", "1.2")},
		{"README.md", "# Readme"},
		{"lib/net6.0/demo.dll", "binary"},
	})

	t.Run("dependency package", func(t *testing.T) {
		lb := &rpfLocalBase{}
		l := rpfNewLocal(t, lb, &rpfFileManager{content: nupkg}, nil, nil)
		reporter, producer := rpfReporter(t)
		l.artifactEventReporter = reporter
		headers, sum, err := l.UploadPackage(ctx, rpfUploadInfo(), io.NopCloser(bytes.NewReader(nupkg)), DependencyFile)
		require.NoError(t, err)
		require.Equal(t, http.StatusCreated, headers.Code)
		require.Equal(t, "sum", sum)
		require.Equal(t, "demo.pkg/1.2.0/demo.pkg.1.2.0.nupkg", lb.gotPath)
		md, ok := lb.gotMetadata.(*nugetmetadata.NugetMetadata)
		require.True(t, ok)
		require.Equal(t, "# Readme", md.PackageMetadata.Readme)
		require.Len(t, producer.streams, 1)
	})

	t.Run("nested path", func(t *testing.T) {
		lb := &rpfLocalBase{}
		l := rpfNewLocal(t, lb, &rpfFileManager{content: nupkg}, nil, nil)
		info := rpfUploadInfo()
		info.NestedPath = "nested"
		_, _, err := l.UploadPackage(ctx, info, io.NopCloser(bytes.NewReader(nupkg)), DependencyFile)
		require.NoError(t, err)
		require.Equal(t, "demo.pkg/1.2.0/nested/demo.pkg.1.2.0.nupkg", lb.gotPath)
	})

	t.Run("symbols package requires an existing version", func(t *testing.T) {
		lb := &rpfLocalBase{versionExists: true}
		l := rpfNewLocal(t, lb, &rpfFileManager{content: nupkg}, nil, nil)
		_, _, err := l.UploadPackage(ctx, rpfUploadInfo(), io.NopCloser(bytes.NewReader(nupkg)), SymbolsFile)
		require.NoError(t, err)
		require.Equal(t, "demo.pkg/1.2.0/demo.pkg.1.2.0.snupkg", lb.gotPath)

		l = rpfNewLocal(t, &rpfLocalBase{}, &rpfFileManager{content: nupkg}, nil, nil)
		_, _, err = l.UploadPackage(ctx, rpfUploadInfo(), io.NopCloser(bytes.NewReader(nupkg)), SymbolsFile)
		require.ErrorContains(t, err, "can't push symbol package")

		l = rpfNewLocal(t, &rpfLocalBase{checkErr: errors.New("x")}, &rpfFileManager{content: nupkg}, nil, nil)
		_, _, err = l.UploadPackage(ctx, rpfUploadInfo(), io.NopCloser(bytes.NewReader(nupkg)), SymbolsFile)
		require.ErrorContains(t, err, "failed to check package version existence")
	})

	t.Run("failures", func(t *testing.T) {
		upload := func(l *localRegistry, body []byte) error {
			_, _, err := l.UploadPackage(ctx, rpfUploadInfo(), io.NopCloser(bytes.NewReader(body)), DependencyFile)
			return err
		}
		require.ErrorContains(t,
			upload(rpfNewLocal(t, &rpfLocalBase{}, &rpfFileManager{uploadErr: errors.New("x")}, nil, nil), nupkg),
			"failed to upload file")
		require.ErrorContains(t,
			upload(rpfNewLocal(t, &rpfLocalBase{}, &rpfFileManager{downloadErr: errors.New("x")}, nil, nil), nupkg),
			"failed to download file")

		badZip := rpfNupkg(t, [][2]string{{"a.nuspec", "<not-closed"}})
		require.ErrorContains(t,
			upload(rpfNewLocal(t, &rpfLocalBase{}, &rpfFileManager{content: badZip}, nil, nil), badZip),
			"failed to build metadata")

		badVersion := rpfNupkg(t, [][2]string{{"a.nuspec", rpfNuspec("demo", "not.a.version")}})
		require.ErrorContains(t,
			upload(rpfNewLocal(t, &rpfLocalBase{}, &rpfFileManager{content: badVersion}, nil, nil), badVersion),
			"invalid version")

		require.Error(t,
			upload(rpfNewLocal(t, &rpfLocalBase{updateErr: errors.New("x")}, &rpfFileManager{content: nupkg}, nil, nil),
				nupkg))
	})
}

func TestRpfLocalBuildMetadata(t *testing.T) {
	l := rpfNewLocal(t, nil, nil, nil, nil)

	t.Run("description is the readme fallback", func(t *testing.T) {
		data := rpfNupkg(t, [][2]string{{"a.nuspec", rpfNuspec("demo", "1.0.0")}})
		md, err := l.buildMetadata(bytes.NewReader(data))
		require.NoError(t, err)
		require.Equal(t, "demo description", md.PackageMetadata.Readme)
	})

	t.Run("explicit readme wins", func(t *testing.T) {
		data := rpfNupkg(t, [][2]string{{"a.nuspec", rpfNuspec("demo", "1.0.0")}, {"docs/Readme.MD", "# hi"}})
		md, err := l.buildMetadata(bytes.NewReader(data))
		require.NoError(t, err)
		require.Equal(t, "# hi", md.PackageMetadata.Readme)
	})

	t.Run("invalid package id", func(t *testing.T) {
		data := rpfNupkg(t, [][2]string{{"a.nuspec", rpfNuspec("bad id!", "1.0.0")}})
		_, err := l.buildMetadata(bytes.NewReader(data))
		require.ErrorContains(t, err, "invalid package id")
	})

	t.Run("no nuspec", func(t *testing.T) {
		data := rpfNupkg(t, [][2]string{{"a.txt", "x"}})
		md, err := l.buildMetadata(bytes.NewReader(data))
		require.NoError(t, err)
		require.Empty(t, md.PackageMetadata.ID)
	})
}

func TestRpfLocalRegistryHelper(t *testing.T) {
	ctx := context.Background()
	lb := &rpfLocalBase{existsResult: true}
	reg := &rpfFakeLocalRegistry{}
	h := NewLocalRegistryHelper(reg, lb)
	info := rpfProxyInfo()

	require.True(t, h.FileExists(ctx, info))
	require.Equal(t, "demo/1.0.0/demo.1.0.0.nupkg", lb.existsPath)

	headers, _, redirect, err := h.DownloadFile(ctx, info)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, headers.Code)
	require.Equal(t, "redirect", redirect)
	require.Equal(t, []string{"1.0.0/demo.1.0.0.nupkg"}, lb.downloadCalls)

	require.NoError(t, h.DeletePackage(ctx, info))
	lb.deleteErr = errors.New("x")
	require.Error(t, h.DeletePackage(ctx, info))

	_, _, err = h.UploadPackageFile(ctx, info, nil)
	require.NoError(t, err)
	require.Equal(t, DependencyFile, reg.bundle)
	info.Filename = "demo.1.0.0.snupkg"
	_, _, err = h.UploadPackageFile(ctx, info, nil)
	require.NoError(t, err)
	require.Equal(t, SymbolsFile, reg.bundle)
}

type rpfFakeLocalRegistry struct {
	LocalRegistry
	bundle FileBundleType
}

func (r *rpfFakeLocalRegistry) UploadPackage(
	_ context.Context, _ nugettype.ArtifactInfo, _ io.ReadCloser, bundle FileBundleType,
) (*commons.ResponseHeaders, string, error) {
	r.bundle = bundle
	return nil, "", nil
}
