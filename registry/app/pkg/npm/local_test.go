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
	"encoding/json"
	"io"
	"mime/multipart"
	"testing"

	urlprovider "github.com/harness/gitness/app/url"
	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/registry/app/driver/filesystem"
	"github.com/harness/gitness/registry/app/metadata"
	npmmeta "github.com/harness/gitness/registry/app/metadata/npm"
	"github.com/harness/gitness/registry/app/pkg"
	"github.com/harness/gitness/registry/app/pkg/base"
	"github.com/harness/gitness/registry/app/pkg/commons"
	"github.com/harness/gitness/registry/app/pkg/filemanager"
	"github.com/harness/gitness/registry/app/pkg/types/npm"
	"github.com/harness/gitness/registry/app/storage"
	"github.com/harness/gitness/registry/app/store"
	"github.com/harness/gitness/registry/types"

	"github.com/stretchr/testify/assert"
)

// -----------------------
// Test Mocks (lightweight)
// -----------------------

type mockLocalBase struct {
	checkIfVersionExists func(ctx context.Context, info pkg.PackageArtifactInfo) (bool, error)
	download             func(
		ctx context.Context,
		info pkg.ArtifactInfo, version, filename string,
	) (*commons.ResponseHeaders, *storage.FileReader, string, error)
	deletePackage     func(ctx context.Context, info pkg.PackageArtifactInfo) error
	deleteVersion     func(ctx context.Context, info pkg.PackageArtifactInfo) error
	moveTempAndCreate func(
		ctx context.Context, info pkg.ArtifactInfo,
		tmp, version, path string,
		md metadata.Metadata, fi types.FileInfo, failOnConflict bool,
	) (*commons.ResponseHeaders, string, int64, bool, error)
}

func TestParseAndUploadNPMPackage_WithAttachment_UploadsData(t *testing.T) {
	ctx := context.Background()
	info := sampleArtifactInfo()

	// Prepare a temp filesystem-backed storage service
	tmpDir := t.TempDir()
	drv, err := filesystem.FromParameters(map[string]any{"rootdirectory": tmpDir})
	if err != nil {
		t.Fatalf("filesystem driver init failed: %v", err)
	}
	svc, err := storage.NewStorageService(drv, nil)
	if err != nil {
		t.Fatalf("storage service init failed: %v", err)
	}
	fm := filemanager.NewFileManager(nil, nil, nil, nil, nil, svc, nil, nil)

	// Wire local registry with a real file manager
	lr := newLocalForTests(&mockLocalBase{}, nil, nil, nil, nil)
	lr.fileManager = fm
	// Prepare blob store for verification of upload
	blobStore := svc.GenericBlobsStore(nil, info.RootIdentifier)

	// Prepare base64 data for attachment
	content := []byte("test tarball bytes")
	b64 := base64.StdEncoding.EncodeToString(content)

	body := `{
        "_id": "pkg",
        "name": "pkg",
        "description": "A sample package",
        "keywords": ["x", "y"],
        "homepage": "http://example",
        "maintainers": [{"name": "dev"}],
        "repository": {"type": "git", "url": "git://example/repo.git"},
        "license": "MIT", 
        "readme": "# Readme",
        "readmeFilename": "README.md",
        "users": {"u": true},
        "time": {"created": "2024-01-01T00:00:00.000Z"},
        "versions": {
            "1.0.0": {
                "name": "pkg",
                "version": "1.0.0",
                "dist": {"shasum": "abc", "integrity": "xyz"}
            }
        },
        "dist-tags": {"latest": "1.0.0"},
        "_attachments": {
            "pkg-1.0.0.tgz": {
                "content_type":"application/octet-stream",
                "length":"18",
                "data":"` + b64 + `"
            }
        }
    }`

	var meta npmmeta.PackageMetadata
	fi, tmp, err := lr.parseAndUploadNPMPackage(ctx, info, bytes.NewReader([]byte(body)), &meta)
	assert.NoError(t, err)

	// Metadata should be parsed
	assert.Equal(t, "pkg", meta.Name)
	assert.Equal(t, "1.0.0", meta.Versions["1.0.0"].Version)

	// Upload should have occurred
	assert.NotEqual(t, "", tmp)
	assert.Greater(t, fi.Size, int64(0))

	// Verify data present at temp path in storage
	tmpPath := "/" + info.RootIdentifier + "/tmp/" + tmp
	sz, statErr := blobStore.Stat(ctx, tmpPath)
	assert.NoError(t, statErr)
	assert.Equal(t, int64(len(content)), sz)
}

// Satisfy base.LocalBase interface with exact signatures.
func (m *mockLocalBase) UploadFile(
	context.Context,
	pkg.ArtifactInfo, string, string, string,
	multipart.File, metadata.Metadata,
) (*commons.ResponseHeaders, string, error) {
	panic("not implemented in tests")
}

func (m *mockLocalBase) Upload(
	context.Context,
	pkg.ArtifactInfo, string, string,
	string, io.ReadCloser, metadata.Metadata,
) (*commons.ResponseHeaders, string, error) {
	panic("not implemented in tests")
}

func (m *mockLocalBase) MoveTempFileAndCreateArtifact(
	ctx context.Context,
	info pkg.ArtifactInfo, tmp,
	version, p string, md metadata.Metadata, fi types.FileInfo, foC bool,
) (*commons.ResponseHeaders, string, int64, bool, error) {
	return m.moveTempAndCreate(ctx, info, tmp, version, p, md, fi, foC)
}

func (m *mockLocalBase) Download(
	ctx context.Context,
	info pkg.ArtifactInfo, version, filename string,
) (*commons.ResponseHeaders, *storage.FileReader, string, error) {
	return m.download(ctx, info, version, filename)
}

func (m *mockLocalBase) Exists(context.Context, pkg.ArtifactInfo, string) bool { return false }
func (m *mockLocalBase) ExistsE(context.Context, pkg.PackageArtifactInfo, string) (*commons.ResponseHeaders, error) {
	return nil, nil //nolint:nilnil
}
func (m *mockLocalBase) DeleteFile(context.Context, pkg.PackageArtifactInfo, string) (*commons.ResponseHeaders, error) {
	return nil, nil //nolint:nilnil
}
func (m *mockLocalBase) ExistsByFilePath(context.Context, int64, string) (bool, error) {
	return false, nil
}
func (m *mockLocalBase) CheckIfVersionExists(ctx context.Context, info pkg.PackageArtifactInfo) (bool, error) {
	return m.checkIfVersionExists(ctx, info)
}
func (m *mockLocalBase) DeletePackage(ctx context.Context, info pkg.PackageArtifactInfo) error {
	return m.deletePackage(ctx, info)
}
func (m *mockLocalBase) DeleteVersion(ctx context.Context, info pkg.PackageArtifactInfo) error {
	return m.deleteVersion(ctx, info)
}
func (m *mockLocalBase) MoveMultipleTempFilesAndCreateArtifact(
	context.Context,
	*pkg.ArtifactInfo, string, metadata.Metadata,
	*[]types.FileInfo, func(info *pkg.ArtifactInfo, fileInfo *types.FileInfo) string, string,
) error {
	return nil
}

type mockTagsDAO struct {
	findByImageNameAndRegID func(ctx context.Context, image string, regID int64) ([]*types.PackageTagMetadata, error)
	create                  func(ctx context.Context, tag *types.PackageTag) (string, error)
	deleteByTagAndImageName func(ctx context.Context, tag string, image string, regID int64) error
}

func (m *mockTagsDAO) FindByImageNameAndRegID(
	ctx context.Context,
	image string, regID int64,
) ([]*types.PackageTagMetadata, error) {
	return m.findByImageNameAndRegID(ctx, image, regID)
}
func (m *mockTagsDAO) Create(ctx context.Context, tag *types.PackageTag) (string, error) {
	return m.create(ctx, tag)
}
func (m *mockTagsDAO) DeleteByTagAndImageName(ctx context.Context, tag string, image string, regID int64) error {
	return m.deleteByTagAndImageName(ctx, tag, image, regID)
}
func (m *mockTagsDAO) DeleteByImageNameAndRegID(context.Context, string, int64) error { return nil } //nolint:nilnil

type mockImageDAO struct {
	getByRepoAndName func(ctx context.Context, parentID int64, repo, name string) (*types.Image, error)
}

func (m *mockImageDAO) DuplicateImage(
	_ context.Context,
	_ *types.Image,
	_ int64,
) (*types.Image, error) {
	return nil, nil //nolint:nilnil
}

func (m *mockImageDAO) GetByUUID(context.Context, string) (*types.Image, error) {
	return nil, nil //nolint:nilnil
}
func (m *mockImageDAO) Get(context.Context, int64) (*types.Image, error) { return nil, nil } //nolint:nilnil
func (m *mockImageDAO) GetByName(context.Context, int64, string) (*types.Image, error) {
	return nil, nil //nolint:nilnil
}
func (m *mockImageDAO) GetByNameAndType(context.Context, int64, string, *artifact.ArtifactType) (*types.Image, error) {
	return nil, nil //nolint:nilnil
}
func (m *mockImageDAO) GetLabelsByParentIDAndRepo(context.Context, int64, string, int, int, string) ([]string, error) {
	return nil, nil //nolint:nilnil
}
func (m *mockImageDAO) CountLabelsByParentIDAndRepo(context.Context, int64, string, string) (int64, error) {
	return 0, nil
}
func (m *mockImageDAO) GetByRepoAndName(ctx context.Context, parentID int64, repo, name string) (*types.Image, error) {
	return m.getByRepoAndName(ctx, parentID, repo, name)
}
func (m *mockImageDAO) CreateOrUpdate(context.Context, *types.Image) error             { return nil }
func (m *mockImageDAO) Update(context.Context, *types.Image) error                     { return nil }
func (m *mockImageDAO) UpdateStatus(context.Context, *types.Image) error               { return nil }
func (m *mockImageDAO) DeleteByImageNameAndRegID(context.Context, int64, string) error { return nil }
func (m *mockImageDAO) DeleteByImageNameIfNoLinkedArtifacts(context.Context, int64, string) error {
	return nil
}

type mockArtifactDAO struct {
	getByUUID               func(ctx context.Context, uuid string) (*types.Artifact, error)
	get                     func(ctx context.Context, id int64) (*types.Artifact, error)
	getByName               func(ctx context.Context, imageID int64, version string) (*types.Artifact, error)
	getByRegistryIDAndImage func(ctx context.Context, registryID int64, image string) (*[]types.Artifact, error)
	searchLatestByName      func(
		ctx context.Context,
		regID int64, name string, limit int, offset int,
	) (*[]types.Artifact, error)
	countLatestByName func(ctx context.Context, regID int64, name string) (int64, error)
}

func (m *mockArtifactDAO) DuplicateArtifact(
	_ context.Context,
	_ *types.Artifact,
	_ int64,
) (*types.Artifact, error) {
	return nil, nil //nolint:nilnil
}

func (m *mockArtifactDAO) GetLatestArtifactsByRepo(
	_ context.Context,
	_ int64, _ int, _ int64,
) (*[]types.ArtifactMetadata, error) {
	// TODO implement me
	panic("implement me")
}

func (m *mockArtifactDAO) GetByUUID(
	ctx context.Context,
	uuid string,
) (*types.Artifact, error) {
	return m.getByUUID(ctx, uuid)
}

func (m *mockArtifactDAO) GetByName(
	ctx context.Context,
	imageID int64, version string,
) (*types.Artifact, error) {
	return m.getByName(ctx, imageID, version)
}
func (m *mockArtifactDAO) GetByRegistryImageAndVersion(
	context.Context,
	int64, string, string,
) (*types.Artifact, error) {
	return nil, nil //nolint:nilnil
}
func (m *mockArtifactDAO) GetByRegistryImageVersionAndArtifactType(
	ctx context.Context, registryID int64, image string, version string, artifactType string,
) (*types.Artifact, error) {
	return nil, nil //nolint:nilnil
}
func (m *mockArtifactDAO) CreateOrUpdate(context.Context, *types.Artifact) (int64, error) {
	return 0, nil
}
func (m *mockArtifactDAO) Count(context.Context) (int64, error) { return 0, nil }
func (m *mockArtifactDAO) GetAllArtifactsByParentID(
	context.Context,
	int64, *[]string, string, string, int, int,
	string, bool, []string,
) (*[]types.ArtifactMetadata, error) {
	return &[]types.ArtifactMetadata{}, nil
}
func (m *mockArtifactDAO) CountAllArtifactsByParentID(
	context.Context,
	int64, *[]string, string, bool, []string,
) (int64, error) {
	return 0, nil
}
func (m *mockArtifactDAO) GetArtifactsByRepo(
	context.Context,
	int64, string, string, string,
	int, int, string, []string, *artifact.ArtifactType,
) (*[]types.ArtifactMetadata, error) {
	return &[]types.ArtifactMetadata{}, nil
}
func (m *mockArtifactDAO) CountArtifactsByRepo(
	context.Context,
	int64, string, string, []string, *artifact.ArtifactType,
) (int64, error) {
	return 0, nil
}
func (m *mockArtifactDAO) GetLatestArtifactMetadata(
	context.Context,
	int64, string, string,
) (*types.ArtifactMetadata, error) {
	return &types.ArtifactMetadata{}, nil
}
func (m *mockArtifactDAO) GetAllVersionsByRepoAndImage(
	context.Context,
	int64, string, string, string,
	int, int, string, *artifact.ArtifactType,
) (*[]types.NonOCIArtifactMetadata, error) {
	return &[]types.NonOCIArtifactMetadata{}, nil
}
func (m *mockArtifactDAO) CountAllVersionsByRepoAndImage(
	context.Context,
	int64, string, string, string, *artifact.ArtifactType,
) (int64, error) {
	return 0, nil
}
func (m *mockArtifactDAO) GetArtifactMetadata(
	context.Context,
	int64, string, string, string, *artifact.ArtifactType,
) (*types.ArtifactMetadata, error) {
	return &types.ArtifactMetadata{}, nil
}
func (m *mockArtifactDAO) UpdateArtifactMetadata(context.Context, json.RawMessage, int64) error {
	return nil //nolint:nilnil
}
func (m *mockArtifactDAO) GetByRegistryIDAndImage(
	ctx context.Context,
	registryID int64, image string,
) (*[]types.Artifact, error) {
	return m.getByRegistryIDAndImage(ctx, registryID, image)
}
func (m *mockArtifactDAO) Get(
	ctx context.Context,
	id int64,
) (*types.Artifact, error) {
	return m.get(ctx, id)
}
func (m *mockArtifactDAO) DeleteByImageNameAndRegistryID(context.Context, int64, string) error {
	return nil //nolint:nilnil
}
func (m *mockArtifactDAO) DeleteByVersionAndImageName(context.Context, string, string, int64) error {
	return nil //nolint:nilnil
}
func (m *mockArtifactDAO) GetLatestByImageID(context.Context, int64) (*types.Artifact, error) {
	return nil, nil //nolint:nilnil
}
func (m *mockArtifactDAO) GetAllArtifactsByRepo(
	context.Context,
	int64, int, int64,
) (*[]types.ArtifactMetadata, error) {
	return nil, nil //nolint:nilnil
}
func (m *mockArtifactDAO) GetArtifactsByRepoAndImageBatch(
	context.Context,
	int64, string, int, int64,
) (*[]types.ArtifactMetadata, error) {
	return &[]types.ArtifactMetadata{}, nil
}
func (m *mockArtifactDAO) SearchLatestByName(
	ctx context.Context,
	regID int64, name string, limit int, offset int,
) (*[]types.Artifact, error) {
	return m.searchLatestByName(ctx, regID, name, limit, offset)
}
func (m *mockArtifactDAO) CountLatestByName(
	ctx context.Context,
	regID int64, name string,
) (int64, error) {
	return m.countLatestByName(ctx, regID, name)
}
func (m *mockArtifactDAO) SearchByImageName(
	context.Context,
	int64, string, int, int,
) (*[]types.ArtifactMetadata, error) {
	return &[]types.ArtifactMetadata{}, nil
}
func (m *mockArtifactDAO) CountByImageName(context.Context, int64, string) (int64, error) {
	return 0, nil
}

type mockURLProvider struct {
	pkgURL func(ctx context.Context, regRef string, pkgType string, params ...string) string
}

func (m *mockURLProvider) GetInternalAPIURL(_ context.Context) string { return "" }
func (m *mockURLProvider) GenerateContainerGITCloneURL(_ context.Context, _ string) string {
	return ""
}
func (m *mockURLProvider) GenerateGITCloneURL(_ context.Context, _ string) string { return "" }
func (m *mockURLProvider) GenerateGITCloneSSHURL(_ context.Context, _ string) string {
	return ""
}
func (m *mockURLProvider) GenerateUIRepoURL(_ context.Context, _ string) string { return "" }
func (m *mockURLProvider) GenerateUIPRURL(_ context.Context, _ string, _ int64) string {
	return ""
}
func (m *mockURLProvider) GenerateUICompareURL(_ context.Context, _ string, _ string, _ string) string {
	return ""
}
func (m *mockURLProvider) GenerateUIRefURL(_ context.Context, _ string, _ string) string {
	return ""
}
func (m *mockURLProvider) GetAPIHostname(_ context.Context) string { return "" }
func (m *mockURLProvider) GenerateUIBuildURL(_ context.Context, _, _ string, _ int64) string {
	return ""
}
func (m *mockURLProvider) GetGITHostname(_ context.Context) string           { return "" }
func (m *mockURLProvider) GetAPIProto(_ context.Context) string              { return "" }
func (m *mockURLProvider) RegistryURL(_ context.Context, _ ...string) string { return "" }
func (m *mockURLProvider) PackageURL(ctx context.Context, regRef string, pkgType string, params ...string) string {
	if m.pkgURL != nil {
		return m.pkgURL(ctx, regRef, pkgType, params...)
	}
	return ""
}
func (m *mockURLProvider) GetUIBaseURL(_ context.Context, _ ...string) string { return "" }
func (m *mockURLProvider) GenerateUIRegistryURL(_ context.Context, _ string, _ string) string {
	return ""
}

// -----------------------
// Helpers
// -----------------------

func newLocalForTests(
	lb base.LocalBase,
	tags store.PackageTagRepository,
	img store.ImageRepository, art store.ArtifactRepository, urlp urlprovider.Provider,
) *localRegistry {
	return &localRegistry{
		localBase:   lb,
		fileManager: filemanager.FileManager{},
		proxyStore:  nil,
		tx:          nil,
		registryDao: nil,
		imageDao:    img,
		tagsDao:     tags,
		nodesDao:    nil,
		artifactDao: art,
		urlProvider: urlp,
	}
}

func sampleArtifactInfo() npm.ArtifactInfo {
	return npm.ArtifactInfo{
		ArtifactInfo: pkg.ArtifactInfo{
			BaseInfo:      &pkg.BaseInfo{RootParentID: 1, RootIdentifier: "root"},
			Registry:      types.Registry{ID: 10, Name: "reg"},
			RegistryID:    10,
			RegIdentifier: "reg",
			Image:         "pkg",
		},
		Version:  "1.0.0",
		Filename: "pkg-1.0.0.tgz",
	}
}

// -----------------------
// Tests
// -----------------------

func TestGetArtifactTypeAndPackageTypes(t *testing.T) {
	lr := newLocalForTests(nil, nil, nil, nil, nil)
	assert.Equal(t, artifact.RegistryTypeVIRTUAL, lr.GetArtifactType())
	assert.Equal(t, []artifact.PackageType{artifact.PackageTypeNPM}, lr.GetPackageTypes())
}

func TestHeadAndDownloadAndDeleteDelegation(t *testing.T) {
	ctx := context.Background()
	info := sampleArtifactInfo()

	lb := &mockLocalBase{
		checkIfVersionExists: func(_ context.Context, _ pkg.PackageArtifactInfo) (bool, error) { return true, nil },
		download: func(
			_ context.Context,
			_ pkg.ArtifactInfo, _, _ string,
		) (*commons.ResponseHeaders, *storage.FileReader, string, error) {
			return &commons.ResponseHeaders{Code: 200}, &storage.FileReader{}, "", nil
		},
		deletePackage: func(context.Context, pkg.PackageArtifactInfo) error { return nil },
		deleteVersion: func(context.Context, pkg.PackageArtifactInfo) error { return nil },
	}
	lr := newLocalForTests(lb, nil, nil, nil, nil)

	ok, err := lr.HeadPackageMetadata(ctx, info)
	assert.NoError(t, err)
	assert.True(t, ok)

	_, fr, body, redirect, err := lr.DownloadPackageFile(ctx, info)
	assert.NoError(t, err)
	assert.NotNil(t, fr)
	assert.Nil(t, body)
	assert.Equal(t, "", redirect)

	assert.NoError(t, lr.DeletePackage(ctx, info))
	assert.NoError(t, lr.DeleteVersion(ctx, info))
}

func TestListTags(t *testing.T) {
	ctx := context.Background()
	info := sampleArtifactInfo()
	tags := []*types.PackageTagMetadata{{Name: "latest", Version: "1.0.0"}, {Name: "beta", Version: "2.0.0"}}
	tdao := &mockTagsDAO{
		findByImageNameAndRegID: func(_ context.Context, _ string, _ int64) ([]*types.PackageTagMetadata, error) {
			return tags, nil
		},
	}
	lr := newLocalForTests(nil, tdao, nil, nil, nil)
	m, err := lr.ListTags(ctx, info)
	assert.NoError(t, err)
	assert.Equal(t, map[string]string{"latest": "1.0.0", "beta": "2.0.0"}, m)
}

func TestAddTag_Success(t *testing.T) {
	ctx := context.Background()
	info := sampleArtifactInfo()
	info.DistTags = []string{"latest"}
	imgDAO := &mockImageDAO{
		getByRepoAndName: func(_ context.Context, _ int64, _, _ string) (*types.Image, error) {
			return &types.Image{ID: 5}, nil
		},
	}
	artDAO := &mockArtifactDAO{
		getByName: func(_ context.Context, _ int64, version string) (*types.Artifact, error) {
			return &types.Artifact{ID: 7, Version: version}, nil
		},
	}
	created := false
	tdao := &mockTagsDAO{
		create: func(
			_ context.Context,
			tag *types.PackageTag,
		) (string, error) {
			created = true
			return tag.ID, nil
		},
		findByImageNameAndRegID: func(_ context.Context, _ string, _ int64) ([]*types.PackageTagMetadata, error) {
			return []*types.PackageTagMetadata{{Name: "latest", Version: info.Version}}, nil
		},
	}
	lr := newLocalForTests(nil, tdao, imgDAO, artDAO, nil)
	m, err := lr.AddTag(ctx, info)
	assert.NoError(t, err)
	assert.True(t, created)
	assert.Equal(t, map[string]string{"latest": info.Version}, m)
}

func TestAddTag_NoDistTags(t *testing.T) {
	ctx := context.Background()
	info := sampleArtifactInfo()
	imgDAO := &mockImageDAO{
		getByRepoAndName: func(_ context.Context, _ int64, _, _ string) (*types.Image, error) {
			return &types.Image{ID: 5}, nil
		},
	}
	artDAO := &mockArtifactDAO{
		getByName: func(_ context.Context, _ int64, version string) (*types.Artifact, error) {
			return &types.Artifact{ID: 7, Version: version}, nil
		},
	}
	lr := newLocalForTests(nil, &mockTagsDAO{}, imgDAO, artDAO, nil)
	_, err := lr.AddTag(ctx, info)
	assert.Error(t, err)
}

func TestDeleteTag_Success(t *testing.T) {
	ctx := context.Background()
	info := sampleArtifactInfo()
	info.DistTags = []string{"beta"}
	deleted := false
	tdao := &mockTagsDAO{
		deleteByTagAndImageName: func(
			_ context.Context,
			_ string,
			_ string,
			_ int64,
		) error {
			deleted = true
			return nil
		},
		findByImageNameAndRegID: func(_ context.Context, _ string, _ int64) ([]*types.PackageTagMetadata, error) {
			return []*types.PackageTagMetadata{}, nil
		},
	}
	lr := newLocalForTests(nil, tdao, nil, nil, nil)
	m, err := lr.DeleteTag(ctx, info)
	assert.NoError(t, err)
	assert.True(t, deleted)
	assert.Equal(t, map[string]string{}, m)
}

func TestGetPackageMetadata_Success(t *testing.T) {
	ctx := context.Background()
	info := sampleArtifactInfo()
	// Prepare artifact with npm metadata
	meta := &npmmeta.NpmMetadata{
		PackageMetadata: npmmeta.PackageMetadata{
			Name: "pkg",
			Versions: map[string]*npmmeta.PackageMetadataVersion{
				"1.0.0": {Name: "pkg", Version: "1.0.0", Dist: npmmeta.PackageDistribution{}},
			},
		},
	}
	metaBytes, _ := json.Marshal(meta)
	arts := []types.Artifact{{Version: "1.0.0", Metadata: metaBytes}}
	artDAO := &mockArtifactDAO{
		getByRegistryIDAndImage: func(_ context.Context, _ int64, _ string) (*[]types.Artifact, error) {
			return &arts, nil
		},
	}
	tdao := &mockTagsDAO{
		findByImageNameAndRegID: func(_ context.Context, _ string, _ int64) ([]*types.PackageTagMetadata, error) {
			return []*types.PackageTagMetadata{{Name: "latest", Version: "1.0.0"}}, nil
		},
	}
	urlp := &mockURLProvider{
		pkgURL: func(_ context.Context, _ string, _ string, _ ...string) string {
			return "http://example.local/registry"
		},
	}
	lr := newLocalForTests(nil, tdao, nil, artDAO, urlp)
	pm, err := lr.GetPackageMetadata(ctx, info)
	assert.NoError(t, err)
	assert.Equal(t, "pkg", pm.Name)
	assert.Contains(t, pm.Versions, "1.0.0")
	assert.Equal(t, "1.0.0", pm.DistTags["latest"])
}

func TestSearchPackage_Success(t *testing.T) {
	ctx := context.Background()
	info := sampleArtifactInfo()
	verMeta := &npmmeta.PackageMetadataVersion{Name: "pkg", Version: "1.2.3"}
	artMeta := &npmmeta.NpmMetadata{
		PackageMetadata: npmmeta.PackageMetadata{
			Name:     "pkg",
			Versions: map[string]*npmmeta.PackageMetadataVersion{"1.2.3": verMeta},
		},
	}
	artBytes, _ := json.Marshal(artMeta)
	list := []types.Artifact{{Metadata: artBytes}}
	artDAO := &mockArtifactDAO{
		searchLatestByName: func(_ context.Context, _ int64, _ string, _ int, _ int) (*[]types.Artifact, error) {
			return &list, nil
		},
		countLatestByName: func(_ context.Context, _ int64, _ string) (int64, error) { return 1, nil },
	}
	urlp := &mockURLProvider{
		pkgURL: func(_ context.Context, _ string, _ string, _ ...string) string {
			return "http://example.local/r"
		},
	}
	lr := newLocalForTests(nil, nil, nil, artDAO, urlp)
	res, err := lr.SearchPackage(ctx, info, 10, 0)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), res.Total)
	assert.Len(t, res.Objects, 1)
	assert.Equal(t, "pkg", res.Objects[0].Package.Name)
}

func TestMapHelpers(t *testing.T) {
	// getScope
	assert.Equal(t, "unscoped", getScope("pkg"))
	assert.Equal(t, "myscope", getScope("@myscope/pkg"))

	// getValueOrDefault
	assert.Equal(t, 5, getValueOrDefault(nil, 5))
	assert.Equal(t, 3, getValueOrDefault(3, 5))

	// CreatePackageMetadataVersion
	ver := &npmmeta.PackageMetadataVersion{
		Name:    "pkg",
		Version: "1.0.0",
		Dist: npmmeta.PackageDistribution{
			Integrity: "int",
		},
	}
	pmv := CreatePackageMetadataVersion("http://reg", ver)
	assert.Equal(t, "pkg@1.0.0", pmv.ID)
	assert.Contains(t, pmv.Dist.Tarball, "/pkg/-/1.0.0/pkg-1.0.0.tgz")
}

func TestParseAndUploadNPMPackage_ParseOnly_NoAttachments(t *testing.T) {
	ctx := context.Background()
	info := sampleArtifactInfo()
	// JSON without _attachments to avoid storage interaction
	body := map[string]any{
		"_id":  "pkg",
		"name": "pkg",
		"versions": map[string]any{
			"1.0.0": map[string]any{
				"name":    "pkg",
				"version": "1.0.0", "dist": map[string]any{"shasum": "abc", "integrity": "xyz"},
			},
		},
		"dist-tags": map[string]string{"latest": "1.0.0"},
	}
	buf, _ := json.Marshal(body)
	lr := newLocalForTests(&mockLocalBase{}, nil, nil, nil, nil)
	var meta npmmeta.PackageMetadata
	fi, tmp, err := lr.parseAndUploadNPMPackage(ctx, info, bytes.NewReader(buf), &meta)
	assert.NoError(t, err)
	assert.Equal(t, types.FileInfo{}, fi)
	assert.Equal(t, "", tmp)
	assert.Equal(t, "pkg", meta.Name)
	assert.Equal(t, "1.0.0", meta.Versions["1.0.0"].Version)
}

func TestProcessAttachmentsOptimized_NoData(t *testing.T) {
	ctx := context.Background()
	info := sampleArtifactInfo()
	// _attachments exists but no data field -> expect error
	jsonStr := `{"_attachments": {"pkg-1.0.0.tgz": {"length": 12}}}`
	bufReader := bufio.NewReader(bytes.NewReader([]byte(jsonStr)))
	dec := json.NewDecoder(bufReader)
	lr := newLocalForTests(&mockLocalBase{}, nil, nil, nil, nil)
	_, _, err := lr.processAttachmentsOptimized(ctx, info, dec, bufReader)
	assert.Error(t, err)
}
