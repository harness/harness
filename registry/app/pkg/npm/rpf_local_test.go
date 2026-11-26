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

package npm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/gob"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
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
	npmmetadata "github.com/harness/gitness/registry/app/metadata/npm"
	"github.com/harness/gitness/registry/app/pkg"
	"github.com/harness/gitness/registry/app/pkg/base"
	"github.com/harness/gitness/registry/app/pkg/commons"
	"github.com/harness/gitness/registry/app/pkg/filemanager"
	npmtype "github.com/harness/gitness/registry/app/pkg/types/npm"
	"github.com/harness/gitness/registry/app/storage"
	"github.com/harness/gitness/registry/app/store"
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
	searchErr error
	countErr  error
	count     int64
}

func (r *rpfArtifactRepo) GetByRegistryImageAndVersion(
	context.Context, int64, string, string,
) (*types.Artifact, error) {
	return r.artifact, r.err
}

func (r *rpfArtifactRepo) GetByName(context.Context, int64, string, ...types.QueryOption) (*types.Artifact, error) {
	return r.artifact, r.err
}

func (r *rpfArtifactRepo) GetByRegistryIDAndImage(context.Context, int64, string) (*[]types.Artifact, error) {
	if r.listErr != nil {
		return nil, r.listErr
	}
	return &r.artifacts, nil
}

func (r *rpfArtifactRepo) SearchLatestByName(
	context.Context, int64, string, int, int,
) (*[]types.Artifact, error) {
	if r.searchErr != nil {
		return nil, r.searchErr
	}
	return &r.artifacts, nil
}

func (r *rpfArtifactRepo) CountLatestByName(context.Context, int64, string) (int64, error) {
	return r.count, r.countErr
}

type rpfTagRepo struct {
	store.PackageTagRepository
	tags      []*types.PackageTagMetadata
	findErr   error
	createErr error
	deleteErr error
	created   []*types.PackageTag
	deleted   []string
}

func (r *rpfTagRepo) FindByImageNameAndRegID(
	context.Context, string, int64, *string,
) ([]*types.PackageTagMetadata, error) {
	return r.tags, r.findErr
}

func (r *rpfTagRepo) Create(_ context.Context, tag *types.PackageTag) (string, error) {
	r.created = append(r.created, tag)
	return tag.ID, r.createErr
}

func (r *rpfTagRepo) DeleteByTagAndImageName(_ context.Context, tag string, _ string, _ int64) error {
	r.deleted = append(r.deleted, tag)
	return r.deleteErr
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

func rpfAuthedContext() context.Context {
	return request.WithAuthSession(context.Background(), &auth.Session{Principal: gtypes.Principal{ID: 9}})
}

type rpfFileManager struct {
	filemanager.FileManager
	uploadErr error
	received  []byte
}

func (f *rpfFileManager) UploadFileNoDBUpdate(
	_ context.Context, _ string, _ multipart.File, r io.Reader, _ int64, _ int64,
) (types.FileInfo, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return types.FileInfo{}, err
	}
	f.received = data
	return types.FileInfo{Sha256: "abc"}, f.uploadErr
}

type rpfLocalBase struct {
	base.LocalBase
	versionExists bool
	err           error
	updateErr     error
	deleteErr     error
	gotPath       string
	uploaded      []string
	existsResult  bool
	existsPath    string
}

func (b *rpfLocalBase) CheckIfVersionExists(context.Context, pkg.PackageArtifactInfo) (bool, error) {
	return b.versionExists, b.err
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
	b.gotPath = path
	if b.updateErr != nil {
		return nil, "", b.updateErr
	}
	return &commons.ResponseHeaders{Code: http.StatusCreated}, "sum", nil
}

func (b *rpfLocalBase) UpdateFileManagerAndCreateArtifact(
	_ context.Context, _ pkg.ArtifactInfo, _, path string, _ metadata.Metadata, _ types.FileInfo, _ bool,
) (*commons.ResponseHeaders, string, int64, bool, error) {
	b.gotPath = path
	return &commons.ResponseHeaders{Code: http.StatusCreated}, "sum", 1, false, b.updateErr
}

func (b *rpfLocalBase) DeletePackage(context.Context, pkg.PackageArtifactInfo) error {
	return b.deleteErr
}
func (b *rpfLocalBase) DeleteVersion(context.Context, pkg.PackageArtifactInfo) error {
	return b.deleteErr
}

func (b *rpfLocalBase) Exists(_ context.Context, _ pkg.ArtifactInfo, path string) bool {
	b.existsPath = path
	return b.existsResult
}

const rpfMissing = "missing"

func rpfNpmInfo() npmtype.ArtifactInfo {
	return npmtype.ArtifactInfo{
		ArtifactInfo: pkg.ArtifactInfo{
			BaseInfo:      &pkg.BaseInfo{ParentID: 1, RootIdentifier: "root", RootParentID: 2},
			RegIdentifier: "reg",
			RegistryID:    3,
			Image:         "demo",
			Registry:      types.Registry{Name: "reg"},
		},
		Version:  "1.0.0",
		Filename: "demo-1.0.0.tgz",
		DistTags: []string{"latest"},
	}
}

type rpfDeps struct {
	lb        *rpfLocalBase
	fm        *rpfFileManager
	artifacts *rpfArtifactRepo
	images    *rpfImageRepo
	tags      *rpfTagRepo
}

func rpfNewLocal(t *testing.T, d rpfDeps) (*localRegistry, *rpfProducer) {
	t.Helper()
	reporter, producer := rpfArtifactReporter(t)
	if d.images == nil {
		d.images = &rpfImageRepo{image: &types.Image{ID: 7}}
	}
	if d.artifacts == nil {
		d.artifacts = &rpfArtifactRepo{artifact: &types.Artifact{ID: 11, Version: "1.0.0"}}
	}
	if d.tags == nil {
		d.tags = &rpfTagRepo{}
	}
	if d.lb == nil {
		d.lb = &rpfLocalBase{}
	}
	reg := NewLocalRegistry(d.lb, d.fm, nil, nil, nil, d.tags, d.images, d.artifacts, nil, rpfURLProvider{}, reporter)
	local, ok := reg.(*localRegistry)
	require.True(t, ok)
	return local, producer
}

func rpfStoredVersion(t *testing.T, version string, author any) types.Artifact {
	t.Helper()
	m := npmmetadata.NpmMetadata{PackageMetadata: npmmetadata.PackageMetadata{
		Name: "@scope/demo",
		Versions: map[string]*npmmetadata.PackageMetadataVersion{
			version: {Name: "@scope/demo", Version: version, Author: author, Description: "d"},
		},
	}}
	raw, err := json.Marshal(m)
	require.NoError(t, err)
	return types.Artifact{Version: version, Metadata: raw, CreatedAt: time.Unix(1700000000, 0)}
}

func TestRpfLocalBasics(t *testing.T) {
	ctx := context.Background()
	info := rpfNpmInfo()

	l, _ := rpfNewLocal(t, rpfDeps{lb: &rpfLocalBase{versionExists: true}})
	exists, err := l.HeadPackageMetadata(ctx, info)
	require.NoError(t, err)
	require.True(t, exists)
	require.Equal(t, artifact.RegistryTypeVIRTUAL, l.GetArtifactType())
	require.Equal(t, []artifact.PackageType{artifact.PackageTypeNPM}, l.GetPackageTypes())

	require.NoError(t, l.DeletePackage(ctx, info))
	require.NoError(t, l.DeleteVersion(ctx, info))
	l, _ = rpfNewLocal(t, rpfDeps{lb: &rpfLocalBase{deleteErr: errors.New("x")}})
	require.Error(t, l.DeletePackage(ctx, info))
	require.Error(t, l.DeleteVersion(ctx, info))
}

func TestRpfLocalDownloadPackageFile(t *testing.T) {
	ctx := context.Background()
	info := rpfNpmInfo()

	l, _ := rpfNewLocal(t, rpfDeps{})
	headers, _, rc, redirect, err := l.DownloadPackageFile(ctx, info)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, headers.Code)
	require.Nil(t, rc)
	require.Equal(t, "redirect:1.0.0/demo-1.0.0.tgz", redirect)

	l, _ = rpfNewLocal(t, rpfDeps{artifacts: &rpfArtifactRepo{err: errors.New("gone")}})
	_, _, _, _, err = l.DownloadPackageFile(ctx, info)
	require.ErrorContains(t, err, "artifact not found or deleted")

	l, _ = rpfNewLocal(t, rpfDeps{lb: &rpfLocalBase{err: errors.New("x")}})
	_, _, _, _, err = l.DownloadPackageFile(ctx, info)
	require.Error(t, err)
}

func TestRpfLocalGetPackageMetadata(t *testing.T) {
	ctx := context.Background()
	info := rpfNpmInfo()
	tagRepo := &rpfTagRepo{tags: []*types.PackageTagMetadata{{Name: "latest", Version: "2.0.0"}}}

	t.Run("merges versions and tags", func(t *testing.T) {
		repo := &rpfArtifactRepo{artifacts: []types.Artifact{
			rpfStoredVersion(t, "1.0.0", nil), rpfStoredVersion(t, "2.0.0", nil),
		}}
		l, _ := rpfNewLocal(t, rpfDeps{artifacts: repo, tags: tagRepo})
		got, err := l.GetPackageMetadata(ctx, info)
		require.NoError(t, err)
		require.Equal(t, "@scope/demo", got.Name)
		require.Len(t, got.Versions, 2)
		require.Equal(t, "https://harness.test/pkg/root/reg/npm/@scope/demo/-/2.0.0/@scope/demo-2.0.0.tgz",
			got.Versions["2.0.0"].Dist.Tarball)
		require.Equal(t, map[string]string{"latest": "2.0.0"}, got.DistTags)
	})

	t.Run("failures", func(t *testing.T) {
		l, _ := rpfNewLocal(t, rpfDeps{artifacts: &rpfArtifactRepo{listErr: errors.New("db")}})
		_, err := l.GetPackageMetadata(ctx, info)
		require.Error(t, err)

		l, _ = rpfNewLocal(t, rpfDeps{artifacts: &rpfArtifactRepo{}})
		_, err = l.GetPackageMetadata(ctx, info)
		require.Error(t, err)

		bad := &rpfArtifactRepo{artifacts: []types.Artifact{{Version: "1", Metadata: []byte("{")}}}
		l, _ = rpfNewLocal(t, rpfDeps{artifacts: bad})
		_, err = l.GetPackageMetadata(ctx, info)
		require.Error(t, err)

		good := &rpfArtifactRepo{artifacts: []types.Artifact{rpfStoredVersion(t, "1.0.0", nil)}}
		l, _ = rpfNewLocal(t, rpfDeps{artifacts: good, tags: &rpfTagRepo{findErr: errors.New("tags down")}})
		_, err = l.GetPackageMetadata(ctx, info)
		require.ErrorContains(t, err, "tags down")
	})
}

func TestRpfLocalSearchPackage(t *testing.T) {
	ctx := context.Background()
	info := rpfNpmInfo()
	author := map[string]string{"name": "alice", "email": "a@example.test"}

	t.Run("maps stored metadata to search results", func(t *testing.T) {
		repo := &rpfArtifactRepo{
			artifacts: []types.Artifact{rpfStoredVersion(t, "1.0.0", author)},
			count:     5,
		}
		l, _ := rpfNewLocal(t, rpfDeps{artifacts: repo})
		got, err := l.SearchPackage(ctx, info, 10, 0)
		require.NoError(t, err)
		require.Equal(t, int64(5), got.Total)
		require.Len(t, got.Objects, 1)
		pkgInfo := got.Objects[0].Package
		require.Equal(t, "@scope/demo", pkgInfo.Name)
		require.Equal(t, "scope", pkgInfo.Scope)
		require.Equal(t, npmmetadata.User{Username: "alice"}, pkgInfo.Author)
		require.Equal(t, "https://harness.test/pkg/root/reg/npm", pkgInfo.Links.Registry)
		require.NotNil(t, pkgInfo.Keywords)
	})

	t.Run("failures", func(t *testing.T) {
		l, _ := rpfNewLocal(t, rpfDeps{artifacts: &rpfArtifactRepo{searchErr: errors.New("x")}})
		_, err := l.SearchPackage(ctx, info, 10, 0)
		require.Error(t, err)

		l, _ = rpfNewLocal(t, rpfDeps{artifacts: &rpfArtifactRepo{countErr: errors.New("x")}})
		_, err = l.SearchPackage(ctx, info, 10, 0)
		require.Error(t, err)

		bad := &rpfArtifactRepo{artifacts: []types.Artifact{{Metadata: []byte("{")}}}
		l, _ = rpfNewLocal(t, rpfDeps{artifacts: bad})
		_, err = l.SearchPackage(ctx, info, 10, 0)
		require.Error(t, err)
	})
}

func TestRpfMapToPackageSearch(t *testing.T) {
	_, err := mapToPackageSearch(types.Artifact{Metadata: []byte("{")}, "u")
	require.Error(t, err)

	empty, err := json.Marshal(npmmetadata.NpmMetadata{PackageMetadata: npmmetadata.PackageMetadata{Name: "demo"}})
	require.NoError(t, err)
	_, err = mapToPackageSearch(types.Artifact{Metadata: empty}, "u")
	require.ErrorContains(t, err, "no version found")

	// An author that is not an object cannot be mapped.
	art := rpfStoredVersion(t, "1.0.0", "just a string")
	_, err = mapToPackageSearch(art, "u")
	require.Error(t, err)

	art = rpfStoredVersion(t, "1.0.0", nil)
	got, err := mapToPackageSearch(art, "u")
	require.NoError(t, err)
	require.Equal(t, npmmetadata.User{}, got.Package.Author)
}

func TestRpfScopeAndDefaults(t *testing.T) {
	require.Equal(t, "scope", getScope("@scope/name"))
	require.Equal(t, "unscoped", getScope("name"))
	require.Equal(t, "unscoped", getScope("@broken"))
	require.Equal(t, "x", getValueOrDefault("x", "d"))
	require.Equal(t, "d", getValueOrDefault(nil, "d"))
}

func TestRpfLocalTags(t *testing.T) {
	ctx := context.Background()
	info := rpfNpmInfo()

	t.Run("list", func(t *testing.T) {
		tags := &rpfTagRepo{tags: []*types.PackageTagMetadata{
			{Name: "latest", Version: "1.0.0"}, {Name: "beta", Version: "2"},
		}}
		l, _ := rpfNewLocal(t, rpfDeps{tags: tags})
		got, err := l.ListTags(ctx, info)
		require.NoError(t, err)
		require.Equal(t, map[string]string{"latest": "1.0.0", "beta": "2"}, got)

		l, _ = rpfNewLocal(t, rpfDeps{tags: &rpfTagRepo{findErr: errors.New("x")}})
		_, err = l.ListTags(ctx, info)
		require.Error(t, err)
	})

	t.Run("add", func(t *testing.T) {
		tags := &rpfTagRepo{tags: []*types.PackageTagMetadata{{Name: "latest", Version: "1.0.0"}}}
		l, _ := rpfNewLocal(t, rpfDeps{tags: tags})
		got, err := l.AddTag(ctx, info)
		require.NoError(t, err)
		require.Equal(t, map[string]string{"latest": "1.0.0"}, got)
		require.Len(t, tags.created, 1)
		require.Equal(t, "latest", tags.created[0].Name)
		require.Equal(t, int64(11), tags.created[0].ArtifactID)

		noTags := info
		noTags.DistTags = nil
		_, err = l.AddTag(ctx, noTags)
		require.Error(t, err)

		blank := info
		blank.DistTags = []string{"  "}
		got, err = l.AddTag(ctx, blank)
		require.NoError(t, err)
		require.Empty(t, got)
		require.Len(t, tags.created, 1)

		l, _ = rpfNewLocal(t, rpfDeps{images: &rpfImageRepo{err: errors.New("x")}})
		_, err = l.AddTag(ctx, info)
		require.Error(t, err)

		l, _ = rpfNewLocal(t, rpfDeps{artifacts: &rpfArtifactRepo{err: errors.New("x")}})
		_, err = l.AddTag(ctx, info)
		require.Error(t, err)

		l, _ = rpfNewLocal(t, rpfDeps{tags: &rpfTagRepo{createErr: errors.New("x")}})
		_, err = l.AddTag(ctx, info)
		require.Error(t, err)
	})

	t.Run("delete", func(t *testing.T) {
		tags := &rpfTagRepo{}
		l, _ := rpfNewLocal(t, rpfDeps{tags: tags})
		_, err := l.DeleteTag(ctx, info)
		require.NoError(t, err)
		require.Equal(t, []string{"latest"}, tags.deleted)

		noTags := info
		noTags.DistTags = nil
		_, err = l.DeleteTag(ctx, noTags)
		require.Error(t, err)

		l, _ = rpfNewLocal(t, rpfDeps{tags: &rpfTagRepo{deleteErr: errors.New("x")}})
		_, err = l.DeleteTag(ctx, info)
		require.Error(t, err)
	})
}

func rpfPublishBody(tarball []byte, extra string) string {
	return `{"_id":"demo","_rev":"1-x","name":"demo","description":"d","readme":"r",` +
		`"dist-tags":{"latest":"1.0.0"},"versions":{"1.0.0":{"name":"demo","version":"1.0.0"}},` +
		`"maintainers":[{"name":"alice"}],"contributors":[],"time":{"1.0.0":"2024-01-01T00:00:00Z"},` +
		`"homepage":"h","keywords":["k"],"repository":{"type":"git"},"author":{"name":"alice"},` +
		`"bugs":{"url":"b"},"readmeFilename":"README.md","users":{"alice":true},"license":"MIT",` +
		`"unknown":{"a":[1,2]},` + extra +
		`"_attachments":{"demo-1.0.0.tgz":{"content_type":"application/octet-stream","length":` +
		fmt.Sprint(len(tarball)) + `,"data":"` + base64.StdEncoding.EncodeToString(tarball) + `"}}}`
}

func TestRpfLocalUploadPackageFile(t *testing.T) {
	ctx := rpfAuthedContext()
	tarball := bytes.Repeat([]byte("tarball-data-"), 1000)

	t.Run("publishes the decoded tarball", func(t *testing.T) {
		fm := &rpfFileManager{}
		lb := &rpfLocalBase{}
		tags := &rpfTagRepo{}
		l, producer := rpfNewLocal(t, rpfDeps{fm: fm, lb: lb, tags: tags})
		_, sum, err := l.UploadPackageFile(ctx, rpfNpmInfo(), io.NopCloser(strings.NewReader(rpfPublishBody(tarball, ""))))
		require.NoError(t, err)
		require.Equal(t, "sum", sum)
		require.Equal(t, tarball, fm.received)
		require.Equal(t, "demo/1.0.0/demo-1.0.0.tgz", lb.gotPath)
		require.Len(t, tags.created, 1)
		require.Len(t, producer.streams, 1)
	})

	t.Run("failures", func(t *testing.T) {
		upload := func(l *localRegistry, body string) error {
			_, _, err := l.UploadPackageFile(ctx, rpfNpmInfo(), io.NopCloser(strings.NewReader(body)))
			return err
		}
		newLocal := func(fm *rpfFileManager, lb *rpfLocalBase, tags *rpfTagRepo) *localRegistry {
			l, _ := rpfNewLocal(t, rpfDeps{fm: fm, lb: lb, tags: tags})
			return l
		}
		body := rpfPublishBody(tarball, "")

		require.Error(t, upload(newLocal(&rpfFileManager{}, nil, nil), "{not json"))
		require.Error(t, upload(newLocal(&rpfFileManager{uploadErr: errors.New("x")}, nil, nil), body))
		require.Error(t, upload(newLocal(&rpfFileManager{}, &rpfLocalBase{updateErr: errors.New("x")}, nil), body))
		require.Error(t, upload(newLocal(&rpfFileManager{}, nil, &rpfTagRepo{createErr: errors.New("x")}), body))

		for _, field := range []string{
			`"_id":1`, `"_rev":1`, `"name":1`, `"description":1`, `"dist-tags":1`, `"versions":1`, `"readme":1`,
			`"maintainers":1`, `"time":1`, `"homepage":1`, `"keywords":1`, `"users":1`, `"readmeFilename":1`,
		} {
			err := upload(newLocal(&rpfFileManager{}, nil, nil), "{"+field+"}")
			require.Error(t, err, field)
		}
	})

	t.Run("malformed attachments", func(t *testing.T) {
		l, _ := rpfNewLocal(t, rpfDeps{fm: &rpfFileManager{}})
		cases := map[string]string{
			"not an object":    `{"_attachments":[]}`,
			"no data":          `{"_attachments":{"a.tgz":{"length":3}}}`,
			"empty":            `{"_attachments":{}}`,
			"value not object": `{"_attachments":{"a.tgz":1}}`,
			"bad separator":    `{"_attachments":{"a.tgz":{"data":1}}}`,
			"unterminated":     `{"_attachments":{"a.tgz":{"data":"QUJD`,
			"missing value":    `{"_attachments":{"a.tgz":{"data"`,
		}
		for name, body := range cases {
			_, _, err := l.UploadPackageFile(ctx, rpfNpmInfo(), io.NopCloser(strings.NewReader(body)))
			require.Error(t, err, name)
		}
	})
}

func TestRpfJSONStringStreamReader(t *testing.T) {
	read := func(in string) (string, error) {
		data, err := io.ReadAll(NewOptimizedJSONStringStreamReader(bufio.NewReader(strings.NewReader(in))))
		return string(data), err
	}

	got, err := read(`abc"trailing`)
	require.NoError(t, err)
	require.Equal(t, "abc", got)

	// An escaped quote does not terminate the string.
	got, err = read(`a\"b"rest`)
	require.NoError(t, err)
	require.Equal(t, `a\"b`, got)

	long := strings.Repeat("x", 10000)
	got, err = read(long + `"tail`)
	require.NoError(t, err)
	require.Equal(t, long, got)

	_, err = read("no closing quote")
	require.ErrorContains(t, err, "missing closing quote")

	_, err = read("")
	require.ErrorContains(t, err, "missing closing quote")
}

func TestRpfLocalUploadWithoutParsing(t *testing.T) {
	ctx := rpfAuthedContext()
	info := rpfNpmInfo()

	lb := &rpfLocalBase{}
	tags := &rpfTagRepo{}
	l, producer := rpfNewLocal(t, rpfDeps{lb: lb, tags: tags})
	headers, sum, err := l.UploadPackageFileWithoutParsing(ctx, info, io.NopCloser(strings.NewReader("tgz")))
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, headers.Code)
	require.Equal(t, "sum", sum)
	require.Equal(t, "demo/1.0.0/demo-1.0.0.tgz", lb.gotPath)
	require.Len(t, tags.created, 1)
	require.Len(t, producer.streams, 1)

	l, _ = rpfNewLocal(t, rpfDeps{lb: &rpfLocalBase{updateErr: errors.New("x")}})
	_, _, err = l.UploadPackageFileWithoutParsing(ctx, info, io.NopCloser(strings.NewReader("tgz")))
	require.Error(t, err)

	l, _ = rpfNewLocal(t, rpfDeps{tags: &rpfTagRepo{createErr: errors.New("x")}})
	_, _, err = l.UploadPackageFileWithoutParsing(ctx, info, io.NopCloser(strings.NewReader("tgz")))
	require.Error(t, err)
}

type rpfFakeLocalRegistry struct {
	LocalRegistry
	called bool
}

func (r *rpfFakeLocalRegistry) UploadPackageFileWithoutParsing(
	context.Context, npmtype.ArtifactInfo, io.ReadCloser,
) (*commons.ResponseHeaders, string, error) {
	r.called = true
	return nil, "sha", nil
}

func TestRpfLocalRegistryHelper(t *testing.T) {
	ctx := context.Background()
	lb := &rpfLocalBase{existsResult: true}
	reg := &rpfFakeLocalRegistry{}
	h := NewLocalRegistryHelper(reg, lb)
	info := rpfNpmInfo()

	require.True(t, h.FileExists(ctx, info))
	require.Equal(t, "demo/1.0.0/demo-1.0.0.tgz", lb.existsPath)
	headers, _, redirect, err := h.DownloadFile(ctx, info)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, headers.Code)
	require.Equal(t, "redirect:1.0.0/demo-1.0.0.tgz", redirect)
	_, sha, err := h.UploadPackageFile(ctx, info, nil)
	require.NoError(t, err)
	require.Equal(t, "sha", sha)
	require.True(t, reg.called)
}

type rpfFakeLocalHelper struct {
	exists      bool
	downloadErr error
	uploaded    chan npmtype.ArtifactInfo
}

func (h *rpfFakeLocalHelper) FileExists(context.Context, npmtype.ArtifactInfo) bool { return h.exists }

func (h *rpfFakeLocalHelper) DownloadFile(
	context.Context, npmtype.ArtifactInfo,
) (*commons.ResponseHeaders, *storage.FileReader, string, error) {
	if h.downloadErr != nil {
		return nil, nil, "", h.downloadErr
	}
	return &commons.ResponseHeaders{Code: http.StatusOK}, nil, "local-redirect", nil
}

func (h *rpfFakeLocalHelper) UploadPackageFile(
	_ context.Context, info npmtype.ArtifactInfo, file io.ReadCloser,
) (*commons.ResponseHeaders, string, error) {
	_, _ = io.Copy(io.Discard, file)
	if h.uploaded != nil {
		h.uploaded <- info
	}
	return nil, "sha", nil
}

func rpfNpmUpstream(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/demo", func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprintf(w, `{"name":"demo","versions":{"1.0.0":{"name":"demo","version":"1.0.0",`+
			`"dist":{"tarball":"http://%s/demo/-/demo-1.0.0.tgz"}}},"dist-tags":{"latest":"1.0.0"}}`, r.Host)
	})
	mux.HandleFunc("/demo/-/demo-1.0.0.tgz", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, "tarball-bytes")
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func rpfNewProxy(repoURL string, helper *rpfFakeLocalHelper, storeErr error) *proxy {
	up := &types.UpstreamProxy{
		RepoKey: "up", RepoURL: repoURL, RepoAuthType: string(artifact.AuthTypeAnonymous),
		PackageType: artifact.PackageTypeNPM,
	}
	p := NewProxy(nil, &rpfProxyStore{proxy: up, err: storeErr}, nil, nil, nil, nil, rpfURLProvider{},
		refcache.SpaceFinder{}, nil, helper)
	px, ok := p.(*proxy)
	if !ok {
		return nil
	}
	return px
}

func TestRpfProxyStubs(t *testing.T) {
	ctx := context.Background()
	p := rpfNewProxy("http://unused", &rpfFakeLocalHelper{}, nil)
	info := rpfNpmInfo()

	require.Equal(t, artifact.RegistryTypeUPSTREAM, p.GetArtifactType())
	require.Equal(t, []artifact.PackageType{artifact.PackageTypeNPM}, p.GetPackageTypes())

	_, err := p.SearchPackage(ctx, info, 1, 0)
	require.ErrorIs(t, err, commons.ErrNotSupported)
	_, _, err = p.UploadPackageFileReader(ctx, info)
	require.ErrorIs(t, err, commons.ErrNotSupported)
	_, err = p.HeadPackageMetadata(ctx, info)
	require.ErrorIs(t, err, commons.ErrNotSupported)
	_, err = p.ListTags(ctx, info)
	require.ErrorIs(t, err, commons.ErrNotSupported)
	_, err = p.AddTag(ctx, info)
	require.ErrorIs(t, err, commons.ErrNotSupported)
	_, err = p.DeleteTag(ctx, info)
	require.ErrorIs(t, err, commons.ErrNotSupported)
	require.ErrorIs(t, p.DeletePackage(ctx, info), commons.ErrNotSupported)
	require.ErrorIs(t, p.DeleteVersion(ctx, info), commons.ErrNotSupported)
	_, _, err = p.UploadPackageFile(ctx, info, nil)
	require.Error(t, err)
	_, _, err = p.UploadPackageFileWithoutParsing(ctx, info, nil)
	require.Error(t, err)
}

func TestRpfProxyGetPackageMetadata(t *testing.T) {
	ctx := context.Background()
	srv := rpfNpmUpstream(t)
	info := rpfNpmInfo()
	info.RootIdentifier = "root"
	info.ParentRegIdentifier = "parent"

	got, err := rpfNewProxy(srv.URL, &rpfFakeLocalHelper{}, nil).GetPackageMetadata(ctx, info)
	require.NoError(t, err)
	require.Equal(t, "demo", got.Name)
	require.Len(t, got.Versions, 1)
	require.Equal(t, "https://harness.test/pkg/root/parent/npm/demo/-/1.0.0/demo-1.0.0.tgz",
		got.Versions["1.0.0"].Dist.Tarball)

	_, err = rpfNewProxy(srv.URL, &rpfFakeLocalHelper{}, errors.New("db")).GetPackageMetadata(ctx, info)
	require.Error(t, err)

	missing := info
	missing.Image = rpfMissing
	_, err = rpfNewProxy(srv.URL, &rpfFakeLocalHelper{}, nil).GetPackageMetadata(ctx, missing)
	require.Error(t, err)
}

func TestRpfProxyDownloadPackageFile(t *testing.T) {
	ctx := context.Background()
	srv := rpfNpmUpstream(t)
	info := rpfNpmInfo()

	t.Run("streams from upstream and caches", func(t *testing.T) {
		helper := &rpfFakeLocalHelper{uploaded: make(chan npmtype.ArtifactInfo, 1)}
		_, _, rc, _, err := rpfNewProxy(srv.URL, helper, nil).DownloadPackageFile(ctx, info)
		require.NoError(t, err)
		data, err := io.ReadAll(rc)
		require.NoError(t, err)
		require.Equal(t, "tarball-bytes", string(data))
		select {
		case cached := <-helper.uploaded:
			require.Equal(t, "demo", cached.Metadata.Name)
		case <-time.After(10 * time.Second):
			t.Fatal("tarball was not cached locally")
		}
	})

	t.Run("local copy is preferred", func(t *testing.T) {
		p := rpfNewProxy(srv.URL, &rpfFakeLocalHelper{exists: true}, nil)
		_, _, _, redirect, err := p.DownloadPackageFile(ctx, info)
		require.NoError(t, err)
		require.Equal(t, "local-redirect", redirect)
	})

	t.Run("broken local copy falls back to upstream", func(t *testing.T) {
		helper := &rpfFakeLocalHelper{
			exists: true, downloadErr: errors.New("corrupt"), uploaded: make(chan npmtype.ArtifactInfo, 1),
		}
		_, _, rc, _, err := rpfNewProxy(srv.URL, helper, nil).DownloadPackageFile(ctx, info)
		require.NoError(t, err)
		require.NotNil(t, rc)
		select {
		case <-helper.uploaded:
		case <-time.After(10 * time.Second):
			t.Fatal("tarball was not cached locally")
		}
	})

	t.Run("failures", func(t *testing.T) {
		_, _, _, _, err := rpfNewProxy(srv.URL, &rpfFakeLocalHelper{}, errors.New("db")).
			DownloadPackageFile(ctx, info)
		require.Error(t, err)

		missing := info
		missing.Image = rpfMissing
		_, _, _, _, err = rpfNewProxy(srv.URL, &rpfFakeLocalHelper{}, nil).DownloadPackageFile(ctx, missing)
		require.Error(t, err)

		noVersion := info
		noVersion.Version = "9.9.9"
		_, _, _, _, err = rpfNewProxy(srv.URL, &rpfFakeLocalHelper{}, nil).DownloadPackageFile(ctx, noVersion)
		require.Error(t, err)
	})
}

func TestRpfProxyPutFileToLocalFailures(t *testing.T) {
	ctx := context.Background()
	srv := rpfNpmUpstream(t)
	remote, err := NewRemoteRegistryHelper(ctx, refcache.SpaceFinder{}, types.UpstreamProxy{
		RepoURL: srv.URL, RepoAuthType: string(artifact.AuthTypeAnonymous), PackageType: artifact.PackageTypeNPM,
	}, nil)
	require.NoError(t, err)
	p := rpfNewProxy(srv.URL, &rpfFakeLocalHelper{}, nil)

	require.NoError(t, p.putFileToLocal(ctx, rpfNpmInfo(), remote))

	missing := rpfNpmInfo()
	missing.Image = rpfMissing
	require.Error(t, p.putFileToLocal(ctx, missing, remote))

	noVersion := rpfNpmInfo()
	noVersion.Version = "9.9.9"
	require.Error(t, p.putFileToLocal(ctx, noVersion, remote))
}
