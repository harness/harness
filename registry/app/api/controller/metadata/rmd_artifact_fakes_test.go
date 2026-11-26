//  Copyright 2026 Harness, Inc.
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

package metadata

import (
	"context"
	"reflect"
	"regexp"
	"strconv"
	"testing"

	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/registry/app/pkg/filemanager"
	"github.com/harness/gitness/registry/app/pkg/quarantine"
	"github.com/harness/gitness/registry/app/store"
	"github.com/harness/gitness/registry/types"

	"github.com/stretchr/testify/require"
)

var rmdStatusRe = regexp.MustCompile(`(\d{3})JSONResponse$`)

// rmdStatus derives the HTTP status from a generated response type name.
func rmdStatus(t *testing.T, resp any) int {
	t.Helper()
	require.NotNil(t, resp)
	name := reflect.TypeOf(resp).Name()
	m := rmdStatusRe.FindStringSubmatch(name)
	require.NotNil(t, m, "unexpected response type %s", name)
	code, err := strconv.Atoi(m[1])
	require.NoError(t, err)
	return code
}

// rmdTagStore is a tag repository fake.
type rmdTagStore struct {
	store.TagRepository
	artifactsByRepo    *[]types.ArtifactMetadata
	artifactsByRepoErr error
	countArtifacts     int64
	quarantineStatuses []bool
	quarantineErr      error
	ociTags            *[]types.TagInfo
	ociTagsErr         error
	countTags          int64
	countTagsErr       error
	latestTagMeta      *types.ArtifactMetadata
	latestTagMetaErr   error
	downloadCount      int64
	downloadCountErr   error
	latestTag          *types.Tag
	latestTagErr       error
	tagMetadata        *types.OciVersionMetadata
	tagMetadataErr     error
	tagsByManifest     *[]string
	tagsByManifestErr  error
	tagDetail          *types.TagDetail
	tagDetailErr       error
	versions           *[]types.OciVersionMetadata
	versionsErr        error
	manifestCounts     map[string]int64
	manifestCountsErr  error
	countOci           int64
	countOciErr        error
	quarantineInfo     map[types.ArtifactIdentifier]*types.QuarantineInfo
	quarantineInfoErr  error
}

func (s rmdTagStore) GetAllArtifactsByRepo(
	_ context.Context, _ int64, _ string, _ string, _ string, _ int, _ int, _ string, _ []string,
) (*[]types.ArtifactMetadata, error) {
	return s.artifactsByRepo, s.artifactsByRepoErr
}

func (s rmdTagStore) CountAllArtifactsByRepo(
	_ context.Context, _ int64, _ string, _ string, _ []string,
) (int64, error) {
	return s.countArtifacts, nil
}

func (s rmdTagStore) GetQuarantineStatusForImages(
	_ context.Context, _ []string, _ int64,
) ([]bool, error) {
	return s.quarantineStatuses, s.quarantineErr
}

func (s rmdTagStore) GetOciTagsInfo(
	_ context.Context, _ int64, _ string, _ int, _ int, _ string,
) (*[]types.TagInfo, error) {
	return s.ociTags, s.ociTagsErr
}

func (s rmdTagStore) CountAllTagsByRepoAndImage(
	_ context.Context, _ int64, _ string, _ string, _ string,
) (int64, error) {
	return s.countTags, s.countTagsErr
}

func (s rmdTagStore) GetLatestTagMetadata(
	_ context.Context, _ int64, _ string, _ string, _ ...types.QueryOption,
) (*types.ArtifactMetadata, error) {
	return s.latestTagMeta, s.latestTagMetaErr
}

func (s rmdTagStore) GetDownloadCountByImageID(_ context.Context, _ int64) (int64, error) {
	return s.downloadCount, s.downloadCountErr
}

func (s rmdTagStore) GetLatestTag(
	_ context.Context, _ int64, _ string, _ ...types.QueryOption,
) (*types.Tag, error) {
	return s.latestTag, s.latestTagErr
}

func (s rmdTagStore) GetTagMetadata(
	_ context.Context, _ int64, _ string, _ string, _ string,
) (*types.OciVersionMetadata, error) {
	return s.tagMetadata, s.tagMetadataErr
}

func (s rmdTagStore) GetTagsByManifestID(_ context.Context, _ int64) (*[]string, error) {
	return s.tagsByManifest, s.tagsByManifestErr
}

func (s rmdTagStore) GetTagDetail(_ context.Context, _ int64, _ string, _ string) (*types.TagDetail, error) {
	return s.tagDetail, s.tagDetailErr
}

func (s rmdTagStore) GetAllTagsByRepoAndImage(
	_ context.Context, _ int64, _ string, _ string, _ string, _ string, _ int, _ int, _ string,
) (*[]types.OciVersionMetadata, error) {
	return s.versions, s.versionsErr
}

func (s rmdTagStore) GetAllOciVersionsByRepoAndImage(
	_ context.Context, _ int64, _ string, _ string, _ string, _ string, _ int, _ int, _ string,
) (*[]types.OciVersionMetadata, error) {
	return s.versions, s.versionsErr
}

func (s rmdTagStore) GetDownloadCountByManifests(
	_ context.Context, _ []string, _ int64,
) (map[string]int64, error) {
	return s.manifestCounts, s.manifestCountsErr
}

func (s rmdTagStore) CountOciVersionByRepoAndImage(
	_ context.Context, _ int64, _ string, _ string, _ string,
) (int64, error) {
	return s.countOci, s.countOciErr
}

func (s rmdTagStore) GetQuarantineInfoForArtifacts(
	_ context.Context, _ []types.ArtifactIdentifier, _ int64,
) (map[types.ArtifactIdentifier]*types.QuarantineInfo, error) {
	return s.quarantineInfo, s.quarantineInfoErr
}

// rmdManifestStore is a manifest repository fake.
type rmdManifestStore struct {
	store.ManifestRepository
	latest     *types.Manifest
	latestErr  error
	byDigest   *types.Manifest
	byDigestE  error
	byTag      *types.Manifest
	byTagErr   error
	payload    *types.Payload
	payloadErr error
}

func (s rmdManifestStore) GetLatestManifest(_ context.Context, _ int64, _ string) (*types.Manifest, error) {
	return s.latest, s.latestErr
}

func (s rmdManifestStore) FindManifestByDigest(
	_ context.Context, _ int64, _ string, _ types.Digest,
) (*types.Manifest, error) {
	return s.byDigest, s.byDigestE
}

func (s rmdManifestStore) FindManifestByTagName(
	_ context.Context, _ int64, _ string, _ string,
) (*types.Manifest, error) {
	return s.byTag, s.byTagErr
}

func (s rmdManifestStore) FindManifestPayloadByTagName(
	_ context.Context, _ int64, _ string, _ string, _ string,
) (*types.Payload, error) {
	return s.payload, s.payloadErr
}

func (s rmdManifestStore) GetManifestPayload(
	_ context.Context, _ int64, _ string, _ string, _ types.Digest,
) (*types.Payload, error) {
	return s.payload, s.payloadErr
}

// rmdImageStore is an image repository fake.
type rmdImageStore struct {
	store.ImageRepository
	image      *types.Image
	imageErr   error
	labels     []string
	labelsErr  error
	labelCount int64
	updateErr  error
	updated    *types.Image
}

func (s *rmdImageStore) Get(_ context.Context, _ int64, _ ...types.QueryOption) (*types.Image, error) {
	return s.image, s.imageErr
}

func (s *rmdImageStore) GetByName(
	_ context.Context, _ int64, _ string, _ ...types.QueryOption,
) (*types.Image, error) {
	return s.image, s.imageErr
}

func (s *rmdImageStore) GetByNameAndType(
	_ context.Context, _ int64, _ string, _ *artifact.ArtifactType, _ ...types.QueryOption,
) (*types.Image, error) {
	return s.image, s.imageErr
}

func (s *rmdImageStore) GetByRepoAndName(
	_ context.Context, _ int64, _ string, _ string,
) (*types.Image, error) {
	if s.imageErr != nil && s.image == nil {
		return &types.Image{}, s.imageErr
	}
	return s.image, s.imageErr
}

func (s *rmdImageStore) GetLabelsByParentIDAndRepo(
	_ context.Context, _ int64, _ string, _ int, _ int, _ string,
) ([]string, error) {
	return s.labels, s.labelsErr
}

func (s *rmdImageStore) CountLabelsByParentIDAndRepo(
	_ context.Context, _ int64, _ string, _ string,
) (int64, error) {
	return s.labelCount, nil
}

func (s *rmdImageStore) Update(_ context.Context, img *types.Image) error {
	s.updated = img
	return s.updateErr
}

// rmdArtifactStore is an artifact repository fake.
type rmdArtifactStore struct {
	store.ArtifactRepository
	byName           *types.Artifact
	byNameErr        error
	metadata         *types.ArtifactMetadata
	metadataErr      error
	latestByImage    *types.Artifact
	latestByImageErr error
	byRepo           *[]types.ArtifactMetadata
	byRepoErr        error
	countByRepo      int64
	versions         *[]types.NonOCIArtifactMetadata
	versionsErr      error
	countVersions    int64
	downloadCount    int64
	downloadCountErr error
}

func (s rmdArtifactStore) GetDownloadCountByID(_ context.Context, _ int64) (int64, error) {
	return s.downloadCount, s.downloadCountErr
}

func (s rmdArtifactStore) GetByName(
	_ context.Context, _ int64, _ string, _ ...types.QueryOption,
) (*types.Artifact, error) {
	return s.byName, s.byNameErr
}

func (s rmdArtifactStore) GetArtifactMetadata(
	_ context.Context, _ int64, _ string, _ string, _ string, _ *artifact.ArtifactType, _ ...types.QueryOption,
) (*types.ArtifactMetadata, error) {
	return s.metadata, s.metadataErr
}

func (s rmdArtifactStore) GetLatestByImageID(_ context.Context, _ int64) (*types.Artifact, error) {
	return s.latestByImage, s.latestByImageErr
}

func (s rmdArtifactStore) GetArtifactsByRepo(
	_ context.Context, _ int64, _ string, _ string, _ string, _ int, _ int, _ string, _ []string,
	_ *artifact.ArtifactType,
) (*[]types.ArtifactMetadata, error) {
	return s.byRepo, s.byRepoErr
}

func (s rmdArtifactStore) CountArtifactsByRepo(
	_ context.Context, _ int64, _ string, _ string, _ []string, _ *artifact.ArtifactType,
) (int64, error) {
	return s.countByRepo, nil
}

func (s rmdArtifactStore) GetAllVersionsByRepoAndImage(
	_ context.Context, _ int64, _ string, _ string, _ string, _ int, _ int, _ string,
	_ *artifact.ArtifactType,
) (*[]types.NonOCIArtifactMetadata, error) {
	return s.versions, s.versionsErr
}

func (s rmdArtifactStore) CountAllVersionsByRepoAndImage(
	_ context.Context, _ int64, _ string, _ string, _ string, _ *artifact.ArtifactType,
) (int64, error) {
	return s.countVersions, nil
}

// rmdQuarantineRepo is a quarantine artifact repository fake.
type rmdQuarantineRepo struct {
	store.QuarantineArtifactRepository
	createErr error
	created   *[]*types.QuarantineArtifact
	byPath    []*types.QuarantineArtifact
	byPathErr error
	deleteErr error
}

func (r rmdQuarantineRepo) Create(_ context.Context, q *types.QuarantineArtifact) error {
	if r.created != nil {
		*r.created = append(*r.created, q)
	}
	q.ID = "qid"
	return r.createErr
}

func (r rmdQuarantineRepo) GetByFilePath(
	_ context.Context, _ string, _ int64, _ string, _ string, _ *artifact.ArtifactType,
) ([]*types.QuarantineArtifact, error) {
	return r.byPath, r.byPathErr
}

func (r rmdQuarantineRepo) DeleteByRegistryIDArtifactAndFilePath(
	_ context.Context, _ int64, _ *int64, _ int64, _ *string,
) error {
	return r.deleteErr
}

// rmdQuarantineFinder records cache evictions.
type rmdQuarantineFinder struct {
	quarantine.Finder
	evicted *[]string
}

func (f rmdQuarantineFinder) EvictCache(
	_ context.Context, _ int64, image string, version string, _ *artifact.ArtifactType,
) {
	if f.evicted != nil {
		*f.evicted = append(*f.evicted, image+"@"+version)
	}
}

// rmdFileMgr is a file manager fake.
type rmdFileMgr struct {
	filemanager.FileManager
	node        *types.Node
	nodeErr     error
	fileInfo    types.FileInfo
	fileInfoErr error
	files       *[]types.FileNodeMetadata
	filesErr    error
	count       int64
	countErr    error
}

func (m rmdFileMgr) GetNode(_ context.Context, _ int64, _ string) (*types.Node, error) {
	return m.node, m.nodeErr
}

func (m rmdFileMgr) GetFileMetadata(_ context.Context, _ int64, _ string) (types.FileInfo, error) {
	return m.fileInfo, m.fileInfoErr
}

func (m rmdFileMgr) GetFilesMetadata(
	_ context.Context, _ string, _ int64, _ string, _ string, _ int, _ int, _ string,
) (*[]types.FileNodeMetadata, error) {
	return m.files, m.filesErr
}

func (m rmdFileMgr) CountFilesByPath(_ context.Context, _ string, _ int64) (int64, error) {
	return m.count, m.countErr
}

// rmdCallFn invokes an endpoint with a given controller and context.
type rmdCallFn func(c *APIController, ctx context.Context) (any, error)

// rmdPrelude describes the status codes an endpoint returns before touching any store.
type rmdPrelude struct {
	name      string
	call      rmdCallFn
	base      int
	space     int
	anon      int
	forbidden int
}

func rmdRunPrelude(t *testing.T, p rmdPrelude) {
	t.Helper()
	run := func(name string, want int, mutate func(*APIController), anon bool) {
		t.Run(p.name+"_"+name, func(t *testing.T) {
			if want == 0 {
				t.Skip("not applicable")
			}
			ctx := rmdAdminCtx()
			if anon {
				ctx = rmdAnonCtx()
			}
			resp, err := p.call(rmdNewCtrl(mutate), ctx)
			require.NoError(t, err)
			require.Equal(t, want, rmdStatus(t, resp))
		})
	}
	run("base_info_error", p.base, func(c *APIController) {
		c.RegistryMetadataHelper = rmdHelper{infoErr: errRmdNotAvailable}
	}, false)
	run("space_error", p.space, func(c *APIController) {
		c.SpaceFinder = rmdSpaceFinder{err: errRmdNotAvailable}
	}, false)
	run("anonymous_denied", p.anon, func(c *APIController) {
		c.Authorizer = rmdAuthorizer{allow: false}
	}, true)
	run("forbidden", p.forbidden, func(c *APIController) {
		c.Authorizer = rmdAuthorizer{allow: false}
	}, false)
}
