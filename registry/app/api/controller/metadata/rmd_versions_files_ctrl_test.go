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
	"errors"
	"testing"
	"time"

	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/registry/types"

	"github.com/stretchr/testify/require"
)

func rmdInfoWithPackage(pkg artifact.PackageType) func(*APIController) {
	return func(c *APIController) {
		info := rmdBaseInfo()
		info.PackageType = pkg
		c.RegistryMetadataHelper = rmdHelper{info: info}
	}
}

func TestRmdGetAllArtifactVersions(t *testing.T) {
	size := artifact.PageSize(10)
	req := artifact.GetAllArtifactVersionsRequestObject{
		RegistryRef: "root/reg", Artifact: "img", Params: artifact.GetAllArtifactVersionsParams{Size: &size},
	}
	call := func(r artifact.GetAllArtifactVersionsRequestObject) rmdCallFn {
		return func(c *APIController, ctx context.Context) (any, error) { return c.GetAllArtifactVersions(ctx, r) }
	}
	rmdRunPrelude(t, rmdPrelude{name: "versions", call: call(req), space: 400, anon: 403, forbidden: 403})

	ociVersion := types.OciVersionMetadata{
		Name: "v1", Digest: rmdDigest1, PackageType: artifact.PackageTypeDOCKER, SchemaVersion: 2,
		MediaType: "application/vnd.docker.distribution.manifest.v2+json", Payload: rmdSchema2Payload(),
		ModifiedAt: time.UnixMilli(5),
	}
	versions := func() *[]types.OciVersionMetadata {
		v := []types.OciVersionMetadata{ociVersion}
		return &v
	}
	oci := func(untagged bool, ts rmdTagStore) func(*APIController) {
		return rmdWithAll(rmdRegistryRepo(artifact.PackageTypeDOCKER), rmdUntagged(untagged), func(c *APIController) {
			c.ImageStore = &rmdImageStore{image: rmdImage()}
			c.TagStore = ts
		})
	}
	okTags := func() rmdTagStore {
		return rmdTagStore{
			versions: versions(), manifestCounts: map[string]int64{rmdDigest1: 12}, countTags: 1, countOci: 1,
		}
	}
	with := func(mod func(ts *rmdTagStore)) rmdTagStore {
		ts := okTags()
		mod(&ts)
		return ts
	}

	rmdRunCases(t, call(req), []rmdCase{
		{name: "registry_not_found", want: 404, mutate: func(c *APIController) {
			c.RegistryRepository = &rmdRegRepo{getErr: errRmdNotFound}
		}},
		{name: "registry_error", want: 500, mutate: func(c *APIController) {
			c.RegistryRepository = &rmdRegRepo{getErr: errors.New("db")}
		}},
		{name: "image_not_found", want: 404, mutate: rmdWithAll(rmdRegistryRepo(artifact.PackageTypeDOCKER),
			func(c *APIController) { c.ImageStore = &rmdImageStore{imageErr: errRmdNotFound} })},
		{name: "image_error", want: 500, mutate: rmdWithAll(rmdRegistryRepo(artifact.PackageTypeDOCKER),
			func(c *APIController) { c.ImageStore = &rmdImageStore{imageErr: errors.New("db")} })},
		{name: "tagged_success", want: 200, mutate: oci(false, okTags())},
		{name: "untagged_success", want: 200, mutate: oci(true, okTags())},
		{name: "tagged_versions_error", want: 500, mutate: oci(false, with(func(ts *rmdTagStore) {
			ts.versions = nil
			ts.versionsErr = errors.New("db")
		}))},
		{name: "download_counts_error", want: 500, mutate: oci(false, with(func(ts *rmdTagStore) {
			ts.manifestCountsErr = errors.New("db")
		}))},
		{name: "quarantine_lookup_error", want: 500, mutate: oci(false, with(func(ts *rmdTagStore) {
			ts.quarantineInfoErr = errors.New("db")
		}))},
		{name: "tagged_count_error", want: 500, mutate: oci(false, with(func(ts *rmdTagStore) {
			ts.countTagsErr = errors.New("db")
		}))},
		{name: "untagged_count_error", want: 500, mutate: oci(true, with(func(ts *rmdTagStore) {
			ts.countOciErr = errors.New("db")
		}))},
		{name: "unreadable_manifest_payload", want: 500, mutate: oci(false, with(func(ts *rmdTagStore) {
			bad := ociVersion
			bad.Payload = types.Payload("not json")
			ts.versions = &[]types.OciVersionMetadata{bad}
		}))},
		{name: "non_oci_success", want: 200, mutate: rmdWithAll(rmdRegistryRepo(artifact.PackageTypeNPM),
			rmdUntagged(false), func(c *APIController) {
				c.ImageStore = &rmdImageStore{image: rmdImage()}
				c.ArtifactStore = rmdArtifactStore{
					versions:      &[]types.NonOCIArtifactMetadata{{Name: "1.0", Size: "10", PackageType: artifact.PackageTypeNPM}},
					countVersions: 1,
				}
			})},
		{name: "non_oci_store_error", want: 500, mutate: rmdWithAll(rmdRegistryRepo(artifact.PackageTypeNPM),
			rmdUntagged(false), func(c *APIController) {
				c.ImageStore = &rmdImageStore{image: rmdImage()}
				c.ArtifactStore = rmdArtifactStore{versionsErr: errors.New("db")}
			})},
		{name: "generic_registry_url", want: 200, mutate: rmdWithAll(rmdRegistryRepo(artifact.PackageTypeGENERIC),
			rmdUntagged(false), func(c *APIController) {
				c.ImageStore = &rmdImageStore{image: rmdImage()}
				c.ArtifactStore = rmdArtifactStore{versions: &[]types.NonOCIArtifactMetadata{}}
			})},
	})

	t.Run("quarantine_and_download_counts_are_merged", func(t *testing.T) {
		ts := okTags()
		ts.quarantineInfo = map[types.ArtifactIdentifier]*types.QuarantineInfo{
			{Name: "img", Version: rmdDigest1, RegistryName: "reg"}: {Reason: "malware"},
		}
		resp, err := rmdNewCtrl(oci(false, ts)).GetAllArtifactVersions(rmdAdminCtx(), req)
		require.NoError(t, err)
		ok, isOK := resp.(artifact.GetAllArtifactVersions200JSONResponse)
		require.True(t, isOK)
		items := *ok.Data.ArtifactVersions
		require.Len(t, items, 1)
		require.Equal(t, int64(12), *items[0].DownloadsCount)
		require.True(t, *items[0].IsQuarantined)
		require.Equal(t, "malware", *items[0].QuarantineReason)
		require.Equal(t, 1, *items[0].DigestCount)
	})

	t.Run("negative_size_and_invalid_artifact_type", func(t *testing.T) {
		neg := artifact.PageSize(-1)
		r := req
		r.Params.Size = &neg
		resp, err := rmdNewCtrl().GetAllArtifactVersions(rmdAdminCtx(), r)
		require.NoError(t, err)
		require.Equal(t, 400, rmdStatus(t, resp))

		bad := artifact.GetAllArtifactVersionsParamsArtifactType("space")
		r2 := req
		r2.Params.ArtifactType = &bad
		resp, err = rmdNewCtrl(rmdRegistryRepo(artifact.PackageTypeNPM)).GetAllArtifactVersions(rmdAdminCtx(), r2)
		require.NoError(t, err)
		require.Equal(t, 400, rmdStatus(t, resp))
	})
}

func TestRmdUpdateQuarantineInfoAndDigestCount(t *testing.T) {
	ctx := context.Background()
	t.Run("empty_versions_skip_lookup", func(t *testing.T) {
		c := rmdNewCtrl(func(c *APIController) { c.TagStore = rmdTagStore{quarantineInfoErr: errors.New("x")} })
		require.NoError(t, c.updateQuarantineInfo(ctx, nil, "img", "reg", 2))
		require.NoError(t, c.updateQuarantineInfo(ctx, &[]types.OciVersionMetadata{}, "img", "reg", 2))
	})

	t.Run("manifest_list_digest_count", func(t *testing.T) {
		payload := types.Payload(`{"schemaVersion":2,` +
			`"mediaType":"application/vnd.docker.distribution.manifest.list.v2+json",` +
			`"manifests":[{"mediaType":"application/vnd.docker.distribution.manifest.v2+json","size":1,"digest":"` +
			rmdDigest1 + `","platform":{"architecture":"amd64","os":"linux"}},` +
			`{"mediaType":"application/vnd.docker.distribution.manifest.v2+json","size":1,"digest":"` +
			rmdDigest2 + `","platform":{"architecture":"arm64","os":"linux"}}]}`)
		versions := []types.OciVersionMetadata{{
			SchemaVersion: 2, MediaType: "application/vnd.docker.distribution.manifest.list.v2+json", Payload: payload,
		}, {
			SchemaVersion: 2, MediaType: "application/vnd.docker.distribution.manifest.v2+json",
			Payload: rmdSchema2Payload(),
		}}
		require.NoError(t, setDigestCount(ctx, versions))
		require.Equal(t, 2, versions[0].DigestCount)
		require.Equal(t, 1, versions[1].DigestCount)
	})

	t.Run("unsupported_schema_version", func(t *testing.T) {
		versions := []types.OciVersionMetadata{{SchemaVersion: 1}}
		require.Error(t, setDigestCount(ctx, versions))
	})
}

func rmdFilesCtrl(pkg artifact.PackageType, fm rmdFileMgr) func(*APIController) {
	return rmdWithAll(rmdRegistryRepo(pkg), func(c *APIController) {
		c.ImageStore = &rmdImageStore{image: rmdImage()}
		c.ArtifactStore = rmdArtifactStore{byName: &types.Artifact{ID: 3, Version: "1.0"}}
		c.fileManager = fm
	})
}

func TestRmdGetArtifactFiles(t *testing.T) {
	size := artifact.PageSize(10)
	req := artifact.GetArtifactFilesRequestObject{
		RegistryRef: "root/reg", Artifact: "img", Version: "1.0", Params: artifact.GetArtifactFilesParams{Size: &size},
	}
	call := func(r artifact.GetArtifactFilesRequestObject) rmdCallFn {
		return func(c *APIController, ctx context.Context) (any, error) { return c.GetArtifactFiles(ctx, r) }
	}
	rmdRunPrelude(t, rmdPrelude{name: "files", call: call(req), base: 400, space: 400, anon: 401, forbidden: 403})

	files := &[]types.FileNodeMetadata{{Path: "/img/1.0/a.bin", Size: 5}}
	fm := rmdFileMgr{files: files, count: 1}

	for _, pkg := range []artifact.PackageType{
		artifact.PackageTypeGENERIC, artifact.PackageTypePYTHON, artifact.PackageTypeNPM, artifact.PackageTypeRPM,
		artifact.PackageTypeNUGET, artifact.PackageTypeGO, artifact.PackageTypeMAVEN,
		artifact.PackageTypeCARGO,
	} {
		t.Run("success_"+string(pkg), func(t *testing.T) {
			resp, err := rmdNewCtrl(rmdFilesCtrl(pkg, fm)).GetArtifactFiles(rmdAdminCtx(), req)
			require.NoError(t, err)
			require.Equal(t, 200, rmdStatus(t, resp))
		})
	}

	rmdRunCases(t, call(req), []rmdCase{
		{name: "registry_not_found", want: 404, mutate: func(c *APIController) {
			c.RegistryRepository = &rmdRegRepo{byParentNameE: errRmdNotFound}
		}},
		{name: "registry_error", want: 500, mutate: func(c *APIController) {
			c.RegistryRepository = &rmdRegRepo{byParentNameE: errors.New("db")}
		}},
		{name: "image_not_found", want: 404, mutate: rmdWithAll(rmdRegistryRepo(artifact.PackageTypeNPM),
			func(c *APIController) { c.ImageStore = &rmdImageStore{imageErr: errRmdNotFound} })},
		{name: "image_error", want: 500, mutate: rmdWithAll(rmdRegistryRepo(artifact.PackageTypeNPM),
			func(c *APIController) { c.ImageStore = &rmdImageStore{imageErr: errors.New("db")} })},
		{name: "version_not_found", want: 404, mutate: rmdWithAll(rmdFilesCtrl(artifact.PackageTypeNPM, fm),
			func(c *APIController) { c.ArtifactStore = rmdArtifactStore{byNameErr: errRmdNotFound} })},
		{name: "version_error", want: 500, mutate: rmdWithAll(rmdFilesCtrl(artifact.PackageTypeNPM, fm),
			func(c *APIController) { c.ArtifactStore = rmdArtifactStore{byNameErr: errors.New("db")} })},
		{name: "file_listing_error", want: 500, mutate: rmdFilesCtrl(artifact.PackageTypeNPM,
			rmdFileMgr{filesErr: errors.New("db")})},
		{name: "file_count_error", want: 500, mutate: rmdFilesCtrl(artifact.PackageTypeNPM,
			rmdFileMgr{files: files, countErr: errors.New("db")})},
		{name: "unsupported_package_file_layout", want: 500, mutate: rmdFilesCtrl(artifact.PackageTypeDOCKER, fm)},
		{name: "unknown_package_type_is_rejected", want: 400, mutate: rmdWithAll(
			rmdFilesCtrl(artifact.PackageTypeCARGO, fm),
			func(c *APIController) { c.PackageWrapper = rmdPkgWrapper{invalidPkgType: true} })},
		{name: "wrapper_supplies_package_url_and_path", want: 200, mutate: rmdWithAll(
			rmdFilesCtrl(artifact.PackageTypeCARGO, fm),
			func(c *APIController) {
				c.PackageWrapper = rmdPkgWrapper{filePath: "/img/1.0"}
			})},
	})

	t.Run("invalid_artifact_type_for_package", func(t *testing.T) {
		bad := artifact.GetArtifactFilesParamsArtifactType("space")
		r := req
		r.Params.ArtifactType = &bad
		resp, err := rmdNewCtrl(rmdFilesCtrl(artifact.PackageTypeHUGGINGFACE, fm)).GetArtifactFiles(rmdAdminCtx(), r)
		require.NoError(t, err)
		require.Equal(t, 400, rmdStatus(t, resp))

		good := artifact.GetArtifactFilesParamsArtifactTypeModel
		r.Params.ArtifactType = &good
		resp, err = rmdNewCtrl(rmdFilesCtrl(artifact.PackageTypeHUGGINGFACE, fm)).GetArtifactFiles(rmdAdminCtx(), r)
		require.NoError(t, err)
		require.Equal(t, 200, rmdStatus(t, resp))
	})

	t.Run("file_payload", func(t *testing.T) {
		resp, err := rmdNewCtrl(rmdFilesCtrl(artifact.PackageTypeNPM, fm)).GetArtifactFiles(rmdAdminCtx(), req)
		require.NoError(t, err)
		ok, isOK := resp.(artifact.GetArtifactFiles200JSONResponse)
		require.True(t, isOK)
		require.Len(t, ok.Data.Files, 1)
		require.Equal(t, "a.bin", ok.Data.Files[0].Name)
		require.Contains(t, ok.Data.Files[0].DownloadCommand, "/npm")
	})
}

func TestRmdGetArtifactFile(t *testing.T) {
	req := artifact.GetArtifactFileRequestObject{
		RegistryRef: "root/reg", Artifact: "img", Version: "1.0", FileName: "a.bin",
	}
	call := func(r artifact.GetArtifactFileRequestObject) rmdCallFn {
		return func(c *APIController, ctx context.Context) (any, error) { return c.GetArtifactFile(ctx, r) }
	}
	rmdRunPrelude(t, rmdPrelude{name: "file", call: call(req), base: 400, space: 400, anon: 401, forbidden: 403})

	fm := rmdFileMgr{fileInfo: types.FileInfo{Filename: "a.bin"}}
	rmdRunCases(t, call(req), []rmdCase{
		{name: "registry_error", want: 500, mutate: func(c *APIController) {
			c.RegistryRepository = &rmdRegRepo{byParentNameE: errors.New("db")}
		}},
		{name: "image_not_found", want: 404, mutate: rmdWithAll(rmdRegistryRepo(artifact.PackageTypeNPM),
			func(c *APIController) { c.ImageStore = &rmdImageStore{imageErr: errRmdNotFound} })},
		{name: "image_error", want: 500, mutate: rmdWithAll(rmdRegistryRepo(artifact.PackageTypeNPM),
			func(c *APIController) { c.ImageStore = &rmdImageStore{imageErr: errors.New("db")} })},
		{name: "version_not_found", want: 404, mutate: rmdWithAll(rmdFilesCtrl(artifact.PackageTypeNPM, fm),
			func(c *APIController) { c.ArtifactStore = rmdArtifactStore{byNameErr: errRmdNotFound} })},
		{name: "version_error", want: 500, mutate: rmdWithAll(rmdFilesCtrl(artifact.PackageTypeNPM, fm),
			func(c *APIController) { c.ArtifactStore = rmdArtifactStore{byNameErr: errors.New("db")} })},
		{name: "unsupported_package_layout", want: 500, mutate: rmdFilesCtrl(artifact.PackageTypeDOCKER, fm)},
		{name: "file_not_found", want: 404, mutate: rmdFilesCtrl(artifact.PackageTypeNPM,
			rmdFileMgr{fileInfoErr: errRmdNotFound})},
		{name: "file_error", want: 500, mutate: rmdFilesCtrl(artifact.PackageTypeNPM,
			rmdFileMgr{fileInfoErr: errors.New("db")})},
		{name: "success", want: 200, mutate: rmdFilesCtrl(artifact.PackageTypeNPM, fm)},
	})

	t.Run("download_url_is_built_from_file_info", func(t *testing.T) {
		resp, err := rmdNewCtrl(rmdFilesCtrl(artifact.PackageTypeNPM, fm)).GetArtifactFile(rmdAdminCtx(), req)
		require.NoError(t, err)
		ok, isOK := resp.(artifact.GetArtifactFile200JSONResponse)
		require.True(t, isOK)
		require.Equal(t, rmdRegistryBase+"/pkg/root/reg/npm/img/1.0/a.bin", ok.DownloadUrl)
	})

	t.Run("invalid_artifact_type", func(t *testing.T) {
		bad := artifact.GetArtifactFileParamsArtifactType("space")
		r := req
		r.Params.ArtifactType = &bad
		resp, err := rmdNewCtrl(rmdFilesCtrl(artifact.PackageTypeNPM, fm)).GetArtifactFile(rmdAdminCtx(), r)
		require.NoError(t, err)
		require.Equal(t, 400, rmdStatus(t, resp))
	})
}

func TestRmdQuarantineFilePath(t *testing.T) {
	version := "1.0"
	filePath := "dir/a.bin"
	newReq := func(mod func(b *artifact.QuarantineFilePathJSONRequestBody)) artifact.QuarantineFilePathRequestObject {
		body := artifact.QuarantineFilePathJSONRequestBody{Artifact: "img", Reason: "bad", Version: &version}
		mod(&body)
		return artifact.QuarantineFilePathRequestObject{RegistryRef: "root/reg", Body: &body}
	}
	req := newReq(func(*artifact.QuarantineFilePathJSONRequestBody) {})
	call := func(r artifact.QuarantineFilePathRequestObject) rmdCallFn {
		return func(c *APIController, ctx context.Context) (any, error) { return c.QuarantineFilePath(ctx, r) }
	}
	rmdRunPrelude(t, rmdPrelude{name: "quarantine", call: call(req), base: 400, space: 400, anon: 401, forbidden: 403})

	var evicted []string
	var created []*types.QuarantineArtifact
	setup := func(pkg artifact.PackageType, is *rmdImageStore, as rmdArtifactStore, fm rmdFileMgr,
		qr rmdQuarantineRepo) func(*APIController) {
		return rmdWithAll(rmdInfoWithPackage(pkg), func(c *APIController) {
			c.ImageStore = is
			c.ArtifactStore = as
			c.fileManager = fm
			qr.created = &created
			c.QuarantineArtifactRepository = qr
			c.QuarantineFinder = rmdQuarantineFinder{evicted: &evicted}
		})
	}
	img := &rmdImageStore{image: rmdImage()}
	art := rmdArtifactStore{byName: &types.Artifact{ID: 3, Version: "1.0"}}
	node := rmdFileMgr{node: &types.Node{ID: "node-1"}}

	rmdRunCases(t, call(req), []rmdCase{
		{name: "image_not_found", want: 400, mutate: setup(artifact.PackageTypeNPM,
			&rmdImageStore{imageErr: errRmdNotFound}, art, node, rmdQuarantineRepo{})},
		{name: "image_error", want: 500, mutate: setup(artifact.PackageTypeNPM,
			&rmdImageStore{imageErr: errors.New("db")}, art, node, rmdQuarantineRepo{})},
		{name: "version_not_found", want: 400, mutate: setup(artifact.PackageTypeNPM, img,
			rmdArtifactStore{byNameErr: errRmdNotFound}, node, rmdQuarantineRepo{})},
		{name: "version_error", want: 500, mutate: setup(artifact.PackageTypeNPM, img,
			rmdArtifactStore{byNameErr: errors.New("db")}, node, rmdQuarantineRepo{})},
		{name: "create_error", want: 500, mutate: setup(artifact.PackageTypeNPM, img, art, node,
			rmdQuarantineRepo{createErr: errors.New("db")})},
		{name: "version_quarantine", want: 200, mutate: setup(artifact.PackageTypeNPM, img, art, node,
			rmdQuarantineRepo{})},
		{name: "docker_version_must_be_a_digest", want: 500, mutate: setup(artifact.PackageTypeDOCKER, img, art, node,
			rmdQuarantineRepo{})},
	})

	t.Run("file_quarantine", func(t *testing.T) {
		fileReq := newReq(func(b *artifact.QuarantineFilePathJSONRequestBody) { b.FilePath = &filePath })
		rmdRunCases(t, call(fileReq), []rmdCase{
			{name: "success", want: 200, mutate: setup(artifact.PackageTypeNPM, img, art, node, rmdQuarantineRepo{})},
			{name: "node_not_found", want: 400, mutate: setup(artifact.PackageTypeNPM, img, art,
				rmdFileMgr{nodeErr: errRmdNotFound}, rmdQuarantineRepo{})},
			{name: "node_error", want: 500, mutate: setup(artifact.PackageTypeNPM, img, art,
				rmdFileMgr{nodeErr: errors.New("db")}, rmdQuarantineRepo{})},
			{name: "unsupported_package_layout", want: 500, mutate: setup(artifact.PackageTypeHELM, img, art, node,
				rmdQuarantineRepo{})},
		})
		noVersion := newReq(func(b *artifact.QuarantineFilePathJSONRequestBody) {
			b.FilePath = &filePath
			b.Version = nil
		})
		rmdRunCases(t, call(noVersion), []rmdCase{
			{name: "file_without_version", want: 400, mutate: setup(artifact.PackageTypeNPM, img, art, node,
				rmdQuarantineRepo{})},
		})
	})

	t.Run("quarantine_record_and_cache_eviction", func(t *testing.T) {
		evicted = nil
		created = nil
		fileReq := newReq(func(b *artifact.QuarantineFilePathJSONRequestBody) { b.FilePath = &filePath })
		resp, err := rmdNewCtrl(setup(artifact.PackageTypeNPM, img, art, node, rmdQuarantineRepo{})).
			QuarantineFilePath(rmdAdminCtx(), fileReq)
		require.NoError(t, err)
		ok, isOK := resp.(artifact.QuarantineFilePath200JSONResponse)
		require.True(t, isOK)
		require.Equal(t, "qid", ok.Data.Id)
		require.Equal(t, "bad", ok.Data.Reason)
		require.Equal(t, int64(3), *ok.Data.VersionId)
		require.Len(t, created, 1)
		require.Equal(t, "node-1", *created[0].NodeID)
		require.Equal(t, int64(9), created[0].ImageID)
		require.Equal(t, []string{"img@1.0"}, evicted)
	})

	t.Run("artifact_type_is_forwarded", func(t *testing.T) {
		at := "model"
		r := newReq(func(b *artifact.QuarantineFilePathJSONRequestBody) { b.ArtifactType = &at })
		resp, err := rmdNewCtrl(setup(artifact.PackageTypeHUGGINGFACE, img, art, node, rmdQuarantineRepo{})).
			QuarantineFilePath(rmdAdminCtx(), r)
		require.NoError(t, err)
		require.Equal(t, 200, rmdStatus(t, resp))
	})

	t.Run("quarantine_without_version", func(t *testing.T) {
		r := newReq(func(b *artifact.QuarantineFilePathJSONRequestBody) { b.Version = nil })
		evicted = nil
		resp, err := rmdNewCtrl(setup(artifact.PackageTypeNPM, img, art, node, rmdQuarantineRepo{})).
			QuarantineFilePath(rmdAdminCtx(), r)
		require.NoError(t, err)
		require.Equal(t, 200, rmdStatus(t, resp))
		require.Empty(t, evicted, "package level quarantine does not evict a version")
	})
}

func TestRmdDeleteQuarantineFilePath(t *testing.T) {
	name := artifact.ArtifactParam("img")
	version := artifact.VersionParam("1.0")
	file := artifact.FilePathParam("dir/a.bin")
	newReq := func(mod func(p *artifact.DeleteQuarantineFilePathParams)) artifact.DeleteQuarantineFilePathRequestObject {
		p := artifact.DeleteQuarantineFilePathParams{Artifact: &name, Version: &version}
		mod(&p)
		return artifact.DeleteQuarantineFilePathRequestObject{RegistryRef: "root/reg", Params: p}
	}
	req := newReq(func(*artifact.DeleteQuarantineFilePathParams) {})
	call := func(r artifact.DeleteQuarantineFilePathRequestObject) rmdCallFn {
		return func(c *APIController, ctx context.Context) (any, error) { return c.DeleteQuarantineFilePath(ctx, r) }
	}
	rmdRunPrelude(t, rmdPrelude{
		name: "deletequarantine", call: call(req), base: 400, space: 400, anon: 401, forbidden: 403,
	})

	var evicted []string
	setup := func(pkg artifact.PackageType, is *rmdImageStore, as rmdArtifactStore, fm rmdFileMgr,
		qr rmdQuarantineRepo) func(*APIController) {
		return rmdWithAll(rmdInfoWithPackage(pkg), func(c *APIController) {
			c.ImageStore = is
			c.ArtifactStore = as
			c.fileManager = fm
			c.QuarantineArtifactRepository = qr
			c.QuarantineFinder = rmdQuarantineFinder{evicted: &evicted}
		})
	}
	img := &rmdImageStore{image: rmdImage()}
	art := rmdArtifactStore{byName: &types.Artifact{ID: 3, Version: "1.0"}}
	node := rmdFileMgr{node: &types.Node{ID: "node-1"}}

	rmdRunCases(t, call(req), []rmdCase{
		{name: "image_error", want: 500, mutate: setup(artifact.PackageTypeNPM,
			&rmdImageStore{imageErr: errors.New("db")}, art, node, rmdQuarantineRepo{})},
		{name: "version_error", want: 500, mutate: setup(artifact.PackageTypeNPM, img,
			rmdArtifactStore{byNameErr: errors.New("db")}, node, rmdQuarantineRepo{})},
		{name: "delete_error", want: 500, mutate: setup(artifact.PackageTypeNPM, img, art, node,
			rmdQuarantineRepo{deleteErr: errors.New("db")})},
		{name: "success", want: 200, mutate: setup(artifact.PackageTypeNPM, img, art, node, rmdQuarantineRepo{})},
		{name: "docker_version_must_be_a_digest", want: 500, mutate: setup(artifact.PackageTypeDOCKER, img, art, node,
			rmdQuarantineRepo{})},
	})

	t.Run("file_quarantine", func(t *testing.T) {
		fileReq := newReq(func(p *artifact.DeleteQuarantineFilePathParams) { p.FilePath = &file })
		rmdRunCases(t, call(fileReq), []rmdCase{
			{name: "success", want: 200, mutate: setup(artifact.PackageTypeNPM, img, art, node, rmdQuarantineRepo{})},
			{name: "node_error", want: 500, mutate: setup(artifact.PackageTypeNPM, img, art,
				rmdFileMgr{nodeErr: errors.New("db")}, rmdQuarantineRepo{})},
			{name: "unsupported_package_layout", want: 500, mutate: setup(artifact.PackageTypeHELM, img, art, node,
				rmdQuarantineRepo{})},
		})
	})

	t.Run("cache_is_evicted_for_versions", func(t *testing.T) {
		evicted = nil
		resp, err := rmdNewCtrl(setup(artifact.PackageTypeNPM, img, art, node, rmdQuarantineRepo{})).
			DeleteQuarantineFilePath(rmdAdminCtx(), req)
		require.NoError(t, err)
		require.Equal(t, 200, rmdStatus(t, resp))
		require.Equal(t, []string{"img@1.0"}, evicted)
	})

	t.Run("without_version_nothing_is_evicted", func(t *testing.T) {
		evicted = nil
		r := newReq(func(p *artifact.DeleteQuarantineFilePathParams) { p.Version = nil })
		resp, err := rmdNewCtrl(setup(artifact.PackageTypeNPM, img, art, node, rmdQuarantineRepo{})).
			DeleteQuarantineFilePath(rmdAdminCtx(), r)
		require.NoError(t, err)
		require.Equal(t, 200, rmdStatus(t, resp))
		require.Empty(t, evicted)
	})
}
