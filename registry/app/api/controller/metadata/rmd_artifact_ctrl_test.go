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
	"fmt"
	"testing"
	"time"

	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/registry/types"
	gitnessstore "github.com/harness/gitness/store"

	"github.com/opencontainers/go-digest"
	"github.com/stretchr/testify/require"
)

const (
	rmdDigest1 = "sha256:1111111111111111111111111111111111111111111111111111111111111111"
	rmdDigest2 = "sha256:2222222222222222222222222222222222222222222222222222222222222222"
)

var errRmdNotFound = fmt.Errorf("lookup: %w", gitnessstore.ErrResourceNotFound)

func rmdSchema2Payload() types.Payload {
	return types.Payload(`{"schemaVersion":2,"mediaType":"application/vnd.docker.distribution.manifest.v2+json",` +
		`"config":{"mediaType":"application/vnd.docker.container.image.v1+json","size":10,"digest":"` + rmdDigest1 + `"},` +
		`"layers":[{"mediaType":"application/vnd.docker.image.rootfs.diff.tar.gzip","size":20,"digest":"` +
		rmdDigest2 + `"}]}`)
}

func rmdRegistryRepo(pkg artifact.PackageType) func(c *APIController) {
	return func(c *APIController) {
		c.RegistryRepository = &rmdRegRepo{
			byParentName: &types.Registry{ID: 5, UUID: "reg-uuid", Name: "reg", ParentID: 2, PackageType: pkg},
			byID: map[int64]*types.Registry{
				5: {ID: 5, UUID: "reg-uuid", Name: "reg", ParentID: 2, PackageType: pkg},
			},
		}
	}
}

func rmdWithAll(muts ...func(*APIController)) func(*APIController) {
	return func(c *APIController) {
		for _, m := range muts {
			m(c)
		}
	}
}

func rmdUntagged(enabled bool) func(*APIController) {
	return func(c *APIController) {
		c.UntaggedImagesEnabled = func(context.Context) bool { return enabled }
	}
}

func rmdImage() *types.Image {
	return &types.Image{ID: 9, UUID: "img-uuid", Name: "img", RegistryID: 5, CreatedAt: time.UnixMilli(1)}
}

type rmdCase struct {
	name   string
	mutate func(*APIController)
	want   int
	ctx    context.Context
}

func rmdRunCases(
	t *testing.T, call func(c *APIController, ctx context.Context) (any, error), cases []rmdCase,
) {
	t.Helper()
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			ctx := tt.ctx
			if ctx == nil {
				ctx = rmdAdminCtx()
			}
			resp, err := call(rmdNewCtrl(tt.mutate), ctx)
			require.NoError(t, err)
			require.Equal(t, tt.want, rmdStatus(t, resp))
		})
	}
}

func TestRmdGetArtifactSummary(t *testing.T) {
	req := artifact.GetArtifactSummaryRequestObject{RegistryRef: "root/reg", Artifact: "img"}
	call := func(c *APIController, ctx context.Context) (any, error) { return c.GetArtifactSummary(ctx, req) }
	rmdRunPrelude(t, rmdPrelude{name: "summary", call: call, base: 400, space: 400, anon: 401, forbidden: 403})

	hfBad := req
	badType := artifact.GetArtifactSummaryParamsArtifactType("space")
	hfBad.Params.ArtifactType = &badType

	docker := func(untagged bool, tags rmdTagStore, mf rmdManifestStore) func(*APIController) {
		return rmdWithAll(rmdRegistryRepo(artifact.PackageTypeDOCKER), rmdUntagged(untagged), func(c *APIController) {
			c.ImageStore = &rmdImageStore{image: rmdImage()}
			c.TagStore = tags
			c.ManifestStore = mf
		})
	}
	generic := func(as rmdArtifactStore) func(*APIController) {
		return rmdWithAll(rmdRegistryRepo(artifact.PackageTypeGENERIC), rmdUntagged(false), func(c *APIController) {
			c.ImageStore = &rmdImageStore{image: rmdImage()}
			c.TagStore = rmdTagStore{}
			c.ArtifactStore = as
		})
	}

	rmdRunCases(t, call, []rmdCase{
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
		{name: "download_count_error", want: 500, mutate: docker(false,
			rmdTagStore{downloadCountErr: errors.New("db")}, rmdManifestStore{})},
		{name: "docker_latest_tag", want: 200, mutate: docker(false,
			rmdTagStore{latestTag: &types.Tag{Name: "latest", UpdatedAt: time.UnixMilli(5)}}, rmdManifestStore{})},
		{name: "docker_latest_tag_error", want: 500, mutate: docker(false,
			rmdTagStore{latestTagErr: errors.New("db")}, rmdManifestStore{})},
		{name: "docker_untagged_latest_manifest", want: 200, mutate: docker(true, rmdTagStore{},
			rmdManifestStore{latest: &types.Manifest{Digest: digest.Digest(rmdDigest1)}})},
		{name: "docker_untagged_manifest_error", want: 500, mutate: docker(true, rmdTagStore{},
			rmdManifestStore{latestErr: errors.New("db")})},
		{name: "generic_latest_artifact", want: 200, mutate: generic(rmdArtifactStore{
			latestByImage: &types.Artifact{Version: "1.0", UpdatedAt: time.UnixMilli(5)},
		})},
		{name: "generic_without_versions", want: 200, mutate: generic(rmdArtifactStore{})},
		{name: "generic_error", want: 500, mutate: generic(rmdArtifactStore{latestByImageErr: errors.New("db")})},
	})

	t.Run("invalid_artifact_type", func(t *testing.T) {
		resp, err := rmdNewCtrl(rmdRegistryRepo(artifact.PackageTypeGENERIC)).GetArtifactSummary(rmdAdminCtx(), hfBad)
		require.NoError(t, err)
		require.Equal(t, 400, rmdStatus(t, resp))
	})

	t.Run("generic_payload", func(t *testing.T) {
		c := rmdNewCtrl(generic(rmdArtifactStore{latestByImage: &types.Artifact{Version: "1.0"}}),
			func(c *APIController) {
				c.TagStore = rmdTagStore{downloadCount: 42}
			})
		resp, err := c.GetArtifactSummary(rmdAdminCtx(), req)
		require.NoError(t, err)
		ok, isOK := resp.(artifact.GetArtifactSummary200JSONResponse)
		require.True(t, isOK)
		require.Equal(t, "img", ok.Data.ImageName)
		require.Equal(t, int64(42), *ok.Data.DownloadsCount)
		require.Equal(t, "reg-uuid", ok.Data.RegistryUUID)
	})
}

func TestRmdGetArtifactVersionSummary(t *testing.T) {
	base := artifact.GetArtifactVersionSummaryRequestObject{RegistryRef: "root/reg", Artifact: "img", Version: "v1"}
	call := func(r artifact.GetArtifactVersionSummaryRequestObject) rmdCallFn {
		return func(c *APIController, ctx context.Context) (any, error) { return c.GetArtifactVersionSummary(ctx, r) }
	}
	rmdRunPrelude(t, rmdPrelude{
		name: "versionsummary", call: call(base), base: 500, space: 500, anon: 401, forbidden: 403,
	})

	withDigest := base
	d := artifact.DigestOptParam(rmdDigest1)
	withDigest.Params.Digest = &d
	badType := artifact.GetArtifactVersionSummaryParamsArtifactType("space")
	withBadType := base
	withBadType.Params.ArtifactType = &badType

	quarantined := []*types.QuarantineArtifact{{Reason: "bad"}}
	docker := func(untagged bool, mut func(c *APIController)) func(*APIController) {
		return rmdWithAll(rmdRegistryRepo(artifact.PackageTypeDOCKER), rmdUntagged(untagged), mut)
	}

	t.Run("cases", func(t *testing.T) {
		rmdRunCases(t, call(base), []rmdCase{
			{name: "registry_error", want: 500, mutate: func(c *APIController) {
				c.RegistryRepository = &rmdRegRepo{getErr: errors.New("db")}
			}},
			{name: "docker_tag", want: 200, mutate: docker(false, func(c *APIController) {
				c.QuarantineArtifactRepository = rmdQuarantineRepo{byPath: quarantined}
				c.TagStore = rmdTagStore{tagMetadata: &types.OciVersionMetadata{
					Name: "v1", PackageType: artifact.PackageTypeDOCKER, ArtifactUUID: "au",
				}}
			})},
			{name: "docker_tag_lookup_error", want: 500, mutate: docker(false, func(c *APIController) {
				c.QuarantineArtifactRepository = rmdQuarantineRepo{byPathErr: errors.New("q")}
				c.TagStore = rmdTagStore{tagMetadataErr: errors.New("db")}
			})},
			{name: "generic_metadata", want: 200, mutate: rmdWithAll(rmdRegistryRepo(artifact.PackageTypeGENERIC),
				func(c *APIController) {
					c.QuarantineArtifactRepository = rmdQuarantineRepo{}
					c.ArtifactStore = rmdArtifactStore{metadata: &types.ArtifactMetadata{Name: "v1"}}
				})},
			{name: "generic_metadata_error", want: 500, mutate: rmdWithAll(rmdRegistryRepo(artifact.PackageTypeGENERIC),
				func(c *APIController) {
					c.QuarantineArtifactRepository = rmdQuarantineRepo{}
					c.ArtifactStore = rmdArtifactStore{metadataErr: errors.New("db")}
				})},
		})
	})

	t.Run("untagged_with_digest_version", func(t *testing.T) {
		digestReq := base
		digestReq.Version = rmdDigest1
		c := rmdNewCtrl(docker(true, func(c *APIController) {
			c.QuarantineArtifactRepository = rmdQuarantineRepo{}
			c.ArtifactStore = rmdArtifactStore{metadata: &types.ArtifactMetadata{
				PackageType: artifact.PackageTypeDOCKER, UUID: "au",
			}}
		}))
		resp, err := c.GetArtifactVersionSummary(rmdAdminCtx(), digestReq)
		require.NoError(t, err)
		ok, isOK := resp.(artifact.GetArtifactVersionSummary200JSONResponse)
		require.True(t, isOK)
		require.Equal(t, "au", ok.Data.Uuid)
		require.False(t, *ok.Data.IsQuarantined)
	})

	t.Run("untagged_with_explicit_digest_param", func(t *testing.T) {
		c := rmdNewCtrl(docker(true, func(c *APIController) {
			c.QuarantineArtifactRepository = rmdQuarantineRepo{byPath: quarantined}
			c.ArtifactStore = rmdArtifactStore{metadata: &types.ArtifactMetadata{PackageType: artifact.PackageTypeDOCKER}}
		}))
		resp, err := c.GetArtifactVersionSummary(rmdAdminCtx(), withDigest)
		require.NoError(t, err)
		ok, isOK := resp.(artifact.GetArtifactVersionSummary200JSONResponse)
		require.True(t, isOK)
		require.True(t, *ok.Data.IsQuarantined)
		require.Equal(t, "bad", *ok.Data.QuarantineReason)
	})

	t.Run("untagged_with_invalid_digest", func(t *testing.T) {
		c := rmdNewCtrl(docker(true, func(c *APIController) { c.QuarantineArtifactRepository = rmdQuarantineRepo{} }))
		resp, err := c.GetArtifactVersionSummary(rmdAdminCtx(), base)
		require.NoError(t, err)
		require.Equal(t, 500, rmdStatus(t, resp), "version is not a digest")
	})

	t.Run("invalid_artifact_type", func(t *testing.T) {
		c := rmdNewCtrl(rmdRegistryRepo(artifact.PackageTypeGENERIC))
		resp, err := c.GetArtifactVersionSummary(rmdAdminCtx(), withBadType)
		require.NoError(t, err)
		require.Equal(t, 500, rmdStatus(t, resp))
	})
}

func TestRmdGetHelmArtifactDetails(t *testing.T) {
	base := artifact.GetHelmArtifactDetailsRequestObject{RegistryRef: "root/reg", Artifact: "chart", Version: "1.0.0"}
	call := func(r artifact.GetHelmArtifactDetailsRequestObject) rmdCallFn {
		return func(c *APIController, ctx context.Context) (any, error) { return c.GetHelmArtifactDetails(ctx, r) }
	}
	rmdRunPrelude(t, rmdPrelude{name: "helmdetails", call: call(base), base: 400, space: 400, forbidden: 403})

	manifest := &types.Manifest{ID: 3, ImageName: "chart", Digest: digest.Digest(rmdDigest1), TotalSize: 100}
	tagType := artifact.GetHelmArtifactDetailsParamsVersionTypeTAG
	byTag := base
	byTag.Params.VersionType = &tagType
	byDigest := base
	byDigest.Version = rmdDigest1

	setup := func(untagged bool, ms rmdManifestStore, ts rmdTagStore, as rmdArtifactStore) func(*APIController) {
		return rmdWithAll(rmdRegistryRepo(artifact.PackageTypeHELM), rmdUntagged(untagged), func(c *APIController) {
			c.ManifestStore = ms
			c.TagStore = ts
			c.ArtifactStore = as
		})
	}
	art := rmdArtifactStore{metadata: &types.ArtifactMetadata{DownloadCount: 7}}

	rmdRunCases(t, call(base), []rmdCase{
		{name: "registry_not_found", want: 404, mutate: func(c *APIController) {
			c.RegistryRepository = &rmdRegRepo{byParentNameE: errRmdNotFound}
		}},
		{name: "registry_error", want: 500, mutate: func(c *APIController) {
			c.RegistryRepository = &rmdRegRepo{byParentNameE: errors.New("db")}
		}},
		{name: "tag_lookup_error", want: 500, mutate: setup(false, rmdManifestStore{},
			rmdTagStore{tagDetailErr: errors.New("db")}, art)},
		{name: "manifest_by_tag_missing", want: 500, mutate: setup(false,
			rmdManifestStore{byTagErr: errRmdNotFound}, rmdTagStore{tagDetail: &types.TagDetail{Name: "1.0.0"}}, art)},
		{name: "manifest_by_tag_error", want: 500, mutate: setup(false,
			rmdManifestStore{byTagErr: errors.New("db")}, rmdTagStore{tagDetail: &types.TagDetail{}}, art)},
		{name: "artifact_metadata_error", want: 500, mutate: setup(false, rmdManifestStore{byTag: manifest},
			rmdTagStore{tagDetail: &types.TagDetail{Name: "1.0.0"}},
			rmdArtifactStore{metadataErr: errors.New("db")})},
		{name: "by_tag_success", want: 200, mutate: setup(false, rmdManifestStore{byTag: manifest},
			rmdTagStore{tagDetail: &types.TagDetail{Name: "1.0.0"}}, art)},
		{name: "untagged_enabled_requires_a_digest_version", want: 500, mutate: setup(true, rmdManifestStore{},
			rmdTagStore{}, art)},
	})

	t.Run("explicit_tag_version_type_with_untagged_enabled", func(t *testing.T) {
		rmdRunCases(t, call(byTag), []rmdCase{
			{name: "success", want: 200, mutate: setup(true, rmdManifestStore{byTag: manifest},
				rmdTagStore{tagDetail: &types.TagDetail{Name: "1.0.0"}}, art)},
		})
	})

	t.Run("by_digest", func(t *testing.T) {
		rmdRunCases(t, call(byDigest), []rmdCase{
			{name: "success_with_tags", want: 200, mutate: setup(true, rmdManifestStore{byDigest: manifest},
				rmdTagStore{tagsByManifest: &[]string{"a", "b"}}, art)},
			{name: "manifest_missing", want: 500, mutate: setup(true, rmdManifestStore{byDigestE: errRmdNotFound},
				rmdTagStore{}, art)},
			{name: "manifest_error", want: 500, mutate: setup(true, rmdManifestStore{byDigestE: errors.New("db")},
				rmdTagStore{}, art)},
			{name: "tags_error", want: 500, mutate: setup(true, rmdManifestStore{byDigest: manifest},
				rmdTagStore{tagsByManifestErr: errors.New("db")}, art)},
		})

		bad := byDigest
		bad.Version = "not-a-digest"
		rmdRunCases(t, call(bad), []rmdCase{
			{name: "invalid_digest", want: 500, mutate: setup(true, rmdManifestStore{}, rmdTagStore{}, art)},
		})

		c := rmdNewCtrl(setup(true, rmdManifestStore{byDigest: manifest},
			rmdTagStore{tagsByManifest: &[]string{"a", "b"}}, art))
		resp, err := c.GetHelmArtifactDetails(rmdAdminCtx(), byDigest)
		require.NoError(t, err)
		ok, isOK := resp.(artifact.GetHelmArtifactDetails200JSONResponse)
		require.True(t, isOK)
		require.Equal(t, &[]string{"a", "b"}, (*ok.Data.Metadata)["tags"])
		require.Equal(t, int64(7), *ok.Data.DownloadsCount)
		require.Contains(t, *ok.Data.PullCommandByDigest, "chart@"+rmdDigest1)
	})
}

func TestRmdGetDockerArtifactDetails(t *testing.T) {
	base := artifact.GetDockerArtifactDetailsRequestObject{
		RegistryRef: "root/reg", Artifact: "img", Version: "v1",
		Params: artifact.GetDockerArtifactDetailsParams{Digest: rmdDigest1},
	}
	call := func(r artifact.GetDockerArtifactDetailsRequestObject) rmdCallFn {
		return func(c *APIController, ctx context.Context) (any, error) { return c.GetDockerArtifactDetails(ctx, r) }
	}
	rmdRunPrelude(t, rmdPrelude{name: "dockerdetails", call: call(base), base: 400, space: 400, forbidden: 403})

	manifest := &types.Manifest{ID: 3, ImageName: "img", Digest: digest.Digest(rmdDigest1), TotalSize: 100}
	art := rmdArtifactStore{metadata: &types.ArtifactMetadata{DownloadCount: 7}}
	tagType := artifact.GetDockerArtifactDetailsParamsVersionTypeTAG
	byTag := base
	byTag.Params.VersionType = &tagType

	setup := func(untagged bool, as rmdArtifactStore, ms rmdManifestStore, ts rmdTagStore,
		qr rmdQuarantineRepo) func(*APIController) {
		return rmdWithAll(rmdRegistryRepo(artifact.PackageTypeDOCKER), rmdUntagged(untagged), func(c *APIController) {
			c.ArtifactStore = as
			c.ManifestStore = ms
			c.TagStore = ts
			c.QuarantineArtifactRepository = qr
		})
	}
	tag := rmdTagStore{tagDetail: &types.TagDetail{Name: "v1", ImageName: "img"}}
	ok := rmdManifestStore{byDigest: manifest}

	rmdRunCases(t, call(base), []rmdCase{
		{name: "registry_not_found", want: 404, mutate: func(c *APIController) {
			c.RegistryRepository = &rmdRegRepo{byParentNameE: errRmdNotFound}
		}},
		{name: "registry_error", want: 500, mutate: func(c *APIController) {
			c.RegistryRepository = &rmdRegRepo{byParentNameE: errors.New("db")}
		}},
		{name: "artifact_metadata_error", want: 500, mutate: setup(false, rmdArtifactStore{metadataErr: errors.New("db")},
			ok, tag, rmdQuarantineRepo{})},
		{name: "manifest_missing", want: 500, mutate: setup(false, art, rmdManifestStore{byDigestE: errRmdNotFound}, tag,
			rmdQuarantineRepo{})},
		{name: "manifest_error", want: 500, mutate: setup(false, art, rmdManifestStore{byDigestE: errors.New("db")},
			tag, rmdQuarantineRepo{})},
		{name: "quarantine_lookup_error", want: 500, mutate: setup(false, art, ok, tag,
			rmdQuarantineRepo{byPathErr: errors.New("db")})},
		{name: "tag_lookup_error", want: 500, mutate: setup(false, art, ok,
			rmdTagStore{tagDetailErr: errors.New("db")}, rmdQuarantineRepo{})},
		{name: "by_tag_success", want: 200, mutate: setup(false, art, ok, tag, rmdQuarantineRepo{})},
		{name: "by_tag_quarantined_with_untagged_enabled", want: 200, mutate: setup(true, art, ok, tag,
			rmdQuarantineRepo{byPath: []*types.QuarantineArtifact{{Reason: "bad"}}})},
		{name: "by_digest_success", want: 200, mutate: setup(true, art, ok, rmdTagStore{}, rmdQuarantineRepo{})},
	})

	t.Run("explicit_tag_type_wins_over_untagged_flag", func(t *testing.T) {
		resp, err := rmdNewCtrl(setup(true, art, ok, tag, rmdQuarantineRepo{})).GetDockerArtifactDetails(rmdAdminCtx(), byTag)
		require.NoError(t, err)
		d, isOK := resp.(artifact.GetDockerArtifactDetails200JSONResponse)
		require.True(t, isOK)
		require.NotNil(t, d.Data.PullCommand)
		require.NotNil(t, d.Data.PullCommandByDigest)
	})

	t.Run("by_digest_tags_and_error", func(t *testing.T) {
		withTags := setup(true, art, ok, rmdTagStore{tagsByManifest: &[]string{"x"}}, rmdQuarantineRepo{})
		resp, err := rmdNewCtrl(withTags).GetDockerArtifactDetails(rmdAdminCtx(), base)
		require.NoError(t, err)
		d, isOK := resp.(artifact.GetDockerArtifactDetails200JSONResponse)
		require.True(t, isOK)
		require.Equal(t, &[]string{"x"}, (*d.Data.Metadata)["tags"])
		require.Equal(t, int64(7), *d.Data.DownloadsCount)

		rmdRunCases(t, call(base), []rmdCase{
			{name: "tags_error", want: 500, mutate: setup(true, art, ok,
				rmdTagStore{tagsByManifestErr: errors.New("db")}, rmdQuarantineRepo{})},
		})
	})

	t.Run("invalid_digest", func(t *testing.T) {
		bad := base
		bad.Params.Digest = "nope"
		rmdRunCases(t, call(bad), []rmdCase{
			{name: "invalid_digest", want: 500, mutate: setup(false, art, ok, tag, rmdQuarantineRepo{})},
		})
	})
}

func TestRmdArtifactManifests(t *testing.T) {
	payload := types.Payload(`{"manifest":true}`)

	t.Run("helm", func(t *testing.T) {
		req := artifact.GetHelmArtifactManifestRequestObject{RegistryRef: "root/reg", Artifact: "chart", Version: "1.0"}
		call := func(c *APIController, ctx context.Context) (any, error) { return c.GetHelmArtifactManifest(ctx, req) }
		rmdRunPrelude(t, rmdPrelude{name: "helmmanifest", call: call, base: 400, space: 400, forbidden: 403})
		withManifest := func(untagged bool, ms rmdManifestStore) func(*APIController) {
			return rmdWithAll(rmdUntagged(untagged), func(c *APIController) { c.ManifestStore = ms })
		}
		rmdRunCases(t, call, []rmdCase{
			{name: "by_tag", want: 200, mutate: withManifest(false, rmdManifestStore{payload: &payload})},
			{name: "by_tag_not_found", want: 400, mutate: withManifest(false, rmdManifestStore{payloadErr: errRmdNotFound})},
			{name: "by_tag_error", want: 500, mutate: withManifest(false,
				rmdManifestStore{payloadErr: errors.New("db")})},
			{name: "untagged_requires_digest", want: 400, mutate: withManifest(true, rmdManifestStore{})},
		})

		digestReq := req
		digestReq.Version = rmdDigest1
		resp, err := rmdNewCtrl(withManifest(true, rmdManifestStore{payload: &payload})).
			GetHelmArtifactManifest(rmdAdminCtx(), digestReq)
		require.NoError(t, err)
		ok, isOK := resp.(artifact.GetHelmArtifactManifest200JSONResponse)
		require.True(t, isOK)
		require.JSONEq(t, string(payload), ok.Data.Manifest)
	})

	t.Run("docker", func(t *testing.T) {
		req := artifact.GetDockerArtifactManifestRequestObject{
			RegistryRef: "root/reg", Artifact: "img", Version: "v1",
			Params: artifact.GetDockerArtifactManifestParams{Digest: rmdDigest1},
		}
		call := func(c *APIController, ctx context.Context) (any, error) { return c.GetDockerArtifactManifest(ctx, req) }
		rmdRunPrelude(t, rmdPrelude{name: "dockermanifest", call: call, base: 400, space: 400, forbidden: 403})
		withManifest := func(ms rmdManifestStore) func(*APIController) {
			return func(c *APIController) { c.ManifestStore = ms }
		}
		rmdRunCases(t, call, []rmdCase{
			{name: "success", want: 200, mutate: withManifest(rmdManifestStore{payload: &payload})},
			{name: "store_error", want: 500, mutate: withManifest(rmdManifestStore{payloadErr: errors.New("db")})},
		})
		bad := req
		bad.Params.Digest = "nope"
		resp, err := rmdNewCtrl().GetDockerArtifactManifest(rmdAdminCtx(), bad)
		require.NoError(t, err)
		require.Equal(t, 500, rmdStatus(t, resp))
	})
}

func TestRmdGetOciArtifactTags(t *testing.T) {
	size := artifact.PageSize(5)
	req := artifact.GetOciArtifactTagsRequestObject{
		RegistryRef: "root/reg", Artifact: "img", Params: artifact.GetOciArtifactTagsParams{Size: &size},
	}
	call := func(c *APIController, ctx context.Context) (any, error) { return c.GetOciArtifactTags(ctx, req) }
	rmdRunPrelude(t, rmdPrelude{name: "ocitags", call: call, base: 400, space: 400, forbidden: 403})

	setup := func(is *rmdImageStore, ts rmdTagStore) func(*APIController) {
		return func(c *APIController) {
			c.ImageStore = is
			c.TagStore = ts
		}
	}
	img := &rmdImageStore{image: rmdImage()}
	tags := &[]types.TagInfo{{Name: "v1", Digest: rmdDigest1}, {Name: "v2", Digest: rmdDigest2}}
	rmdRunCases(t, call, []rmdCase{
		{name: "image_missing", want: 404, mutate: setup(&rmdImageStore{imageErr: errRmdNotFound}, rmdTagStore{})},
		{name: "tags_error", want: 500, mutate: setup(img, rmdTagStore{ociTagsErr: errors.New("db")})},
		{name: "count_error", want: 500, mutate: setup(img, rmdTagStore{ociTags: tags, countTagsErr: errors.New("db")})},
		{name: "success", want: 200, mutate: setup(img, rmdTagStore{ociTags: tags, countTags: 11})},
	})

	resp, err := rmdNewCtrl(setup(img, rmdTagStore{ociTags: tags, countTags: 11})).GetOciArtifactTags(rmdAdminCtx(), req)
	require.NoError(t, err)
	ok, isOK := resp.(artifact.GetOciArtifactTags200JSONResponse)
	require.True(t, isOK)
	require.Len(t, ok.Data.OciArtifactTags, 2)
	require.Equal(t, int64(3), *ok.Data.PageCount)
	require.Equal(t, "v2", ok.Data.OciArtifactTags[1].Name)
}

func TestRmdGetAllArtifactsByRegistry(t *testing.T) {
	size := artifact.PageSize(10)
	req := artifact.GetAllArtifactsByRegistryRequestObject{
		RegistryRef: "root/reg", Params: artifact.GetAllArtifactsByRegistryParams{Size: &size},
	}
	call := func(r artifact.GetAllArtifactsByRegistryRequestObject) rmdCallFn {
		return func(c *APIController, ctx context.Context) (any, error) { return c.GetAllArtifactsByRegistry(ctx, r) }
	}
	rmdRunPrelude(t, rmdPrelude{name: "artifactsbyregistry", call: call(req), base: 400, space: 400, forbidden: 403})

	arts := &[]types.ArtifactMetadata{{Name: "a"}, {Name: "b"}, {Name: "a"}}
	setup := func(pkg artifact.PackageType, ts rmdTagStore, as rmdArtifactStore) func(*APIController) {
		return rmdWithAll(rmdRegistryRepo(pkg), func(c *APIController) {
			c.TagStore = ts
			c.ArtifactStore = as
		})
	}
	rmdRunCases(t, call(req), []rmdCase{
		{name: "registry_error", want: 500, mutate: func(c *APIController) {
			c.RegistryRepository = &rmdRegRepo{byParentNameE: errors.New("db")}
		}},
		{name: "docker_success", want: 200, mutate: setup(artifact.PackageTypeDOCKER,
			rmdTagStore{artifactsByRepo: arts, countArtifacts: 3, quarantineStatuses: []bool{true, false}},
			rmdArtifactStore{})},
		{name: "docker_store_error", want: 500, mutate: setup(artifact.PackageTypeDOCKER,
			rmdTagStore{artifactsByRepoErr: errors.New("db")}, rmdArtifactStore{})},
		{name: "docker_quarantine_error", want: 500, mutate: setup(artifact.PackageTypeDOCKER,
			rmdTagStore{artifactsByRepo: arts, quarantineErr: errors.New("db")}, rmdArtifactStore{})},
		{name: "generic_success", want: 200, mutate: setup(artifact.PackageTypeGENERIC,
			rmdTagStore{quarantineStatuses: []bool{false}}, rmdArtifactStore{byRepo: arts, countByRepo: 3})},
		{name: "generic_store_error", want: 500, mutate: setup(artifact.PackageTypeGENERIC,
			rmdTagStore{}, rmdArtifactStore{byRepoErr: errors.New("db")})},
		{name: "empty_result_skips_quarantine_lookup", want: 200, mutate: setup(artifact.PackageTypeGENERIC,
			rmdTagStore{quarantineErr: errors.New("must not be called")},
			rmdArtifactStore{byRepo: &[]types.ArtifactMetadata{}})},
	})

	t.Run("quarantine_flags_are_applied", func(t *testing.T) {
		c := rmdNewCtrl(setup(artifact.PackageTypeGENERIC, rmdTagStore{quarantineStatuses: []bool{true, false}},
			rmdArtifactStore{byRepo: arts, countByRepo: 3}))
		resp, err := c.GetAllArtifactsByRegistry(rmdAdminCtx(), req)
		require.NoError(t, err)
		ok, isOK := resp.(artifact.GetAllArtifactsByRegistry200JSONResponse)
		require.True(t, isOK)
		require.Len(t, ok.Data.Artifacts, 3)
		require.True(t, *ok.Data.Artifacts[0].IsQuarantined)
		require.False(t, *ok.Data.Artifacts[1].IsQuarantined)
		require.True(t, *ok.Data.Artifacts[2].IsQuarantined, "status is per image name")
	})

	t.Run("negative_page_size_and_invalid_type", func(t *testing.T) {
		neg := artifact.PageSize(-1)
		r := req
		r.Params.Size = &neg
		resp, err := rmdNewCtrl().GetAllArtifactsByRegistry(rmdAdminCtx(), r)
		require.NoError(t, err)
		require.Equal(t, 400, rmdStatus(t, resp))

		bad := artifact.GetAllArtifactsByRegistryParamsArtifactType("space")
		r2 := req
		r2.Params.ArtifactType = &bad
		resp, err = rmdNewCtrl(setup(artifact.PackageTypeGENERIC, rmdTagStore{}, rmdArtifactStore{})).
			GetAllArtifactsByRegistry(rmdAdminCtx(), r2)
		require.NoError(t, err)
		require.Equal(t, 400, rmdStatus(t, resp))
	})
}

func TestRmdUpdateArtifactLabels(t *testing.T) {
	req := artifact.UpdateArtifactLabelsRequestObject{
		RegistryRef: "root/reg", Artifact: "img",
		Body: &artifact.UpdateArtifactLabelsJSONRequestBody{Labels: []string{"x", "y"}},
	}
	call := func(c *APIController, ctx context.Context) (any, error) { return c.UpdateArtifactLabels(ctx, req) }
	rmdRunPrelude(t, rmdPrelude{name: "updatelabels", call: call, space: 400, anon: 401, forbidden: 403})

	setup := func(is *rmdImageStore, ts rmdTagStore) func(*APIController) {
		return func(c *APIController) {
			c.ImageStore = is
			c.TagStore = ts
		}
	}
	tag := rmdTagStore{latestTagMeta: &types.ArtifactMetadata{Name: "img", PackageType: artifact.PackageTypeDOCKER}}
	rmdRunCases(t, call, []rmdCase{
		{name: "unknown_artifact", want: 404, mutate: setup(&rmdImageStore{image: &types.Image{}}, tag)},
		{name: "update_error", want: 400, mutate: setup(&rmdImageStore{image: rmdImage(), updateErr: errors.New("db")}, tag)},
		{name: "latest_tag_error", want: 500, mutate: setup(&rmdImageStore{image: rmdImage()},
			rmdTagStore{latestTagMetaErr: errors.New("db")})},
		{name: "success", want: 200, mutate: setup(&rmdImageStore{image: rmdImage()}, tag)},
	})

	t.Run("labels_are_replaced", func(t *testing.T) {
		is := &rmdImageStore{image: rmdImage()}
		resp, err := rmdNewCtrl(setup(is, tag)).UpdateArtifactLabels(rmdAdminCtx(), req)
		require.NoError(t, err)
		require.Equal(t, 200, rmdStatus(t, resp))
		require.Equal(t, []string{"x", "y"}, is.updated.Labels)
		require.Equal(t, int64(9), is.updated.ID)
	})
}

func TestRmdDeleteArtifactVersion(t *testing.T) {
	req := artifact.DeleteArtifactVersionRequestObject{RegistryRef: "root/reg", Artifact: "img", Version: "v1"}
	call := func(c *APIController, ctx context.Context) (any, error) {
		resp, err := c.DeleteArtifactVersion(ctx, req)
		// The controller returns the failure both as response and error for pre-auth failures.
		return resp, err
	}

	for name, mutate := range map[string]func(*APIController){
		"base_info_error": func(c *APIController) { c.RegistryMetadataHelper = rmdHelper{infoErr: errRmdNotAvailable} },
		"space_error":     func(c *APIController) { c.SpaceFinder = rmdSpaceFinder{err: errRmdNotAvailable} },
	} {
		t.Run(name, func(t *testing.T) {
			resp, err := call(rmdNewCtrl(mutate), rmdAdminCtx())
			require.Error(t, err)
			require.Equal(t, 400, rmdStatus(t, resp))
		})
	}

	rmdRunCases(t, func(c *APIController, ctx context.Context) (any, error) { return call(c, ctx) }, []rmdCase{
		{name: "anonymous_denied", want: 401, ctx: rmdAnonCtx(),
			mutate: func(c *APIController) { c.Authorizer = rmdAuthorizer{allow: false} }},
		{name: "forbidden", want: 403, mutate: func(c *APIController) { c.Authorizer = rmdAuthorizer{allow: false} }},
		{name: "registry_missing", want: 404, mutate: func(c *APIController) {
			c.RegistryRepository = &rmdRegRepo{byParentNameE: errors.New("db")}
		}},
		{name: "artifact_missing", want: 404, mutate: rmdWithAll(rmdRegistryRepo(artifact.PackageTypeNPM),
			func(c *APIController) { c.ImageStore = &rmdImageStore{imageErr: errRmdNotFound} })},
	})

	t.Run("error_helper", func(t *testing.T) {
		require.Equal(t, "boom", throwDeleteArtifactVersion500Error(errors.New("boom")).Message)
	})
}

func TestRmdListArtifactLabels(t *testing.T) {
	size := artifact.PageSize(10)
	req := artifact.ListArtifactLabelsRequestObject{
		RegistryRef: "root/reg", Params: artifact.ListArtifactLabelsParams{Size: &size},
	}
	call := func(c *APIController, ctx context.Context) (any, error) { return c.ListArtifactLabels(ctx, req) }
	rmdRunPrelude(t, rmdPrelude{name: "listlabels", call: call, base: 400, space: 400, forbidden: 403})
	rmdRunCases(t, call, []rmdCase{
		{name: "store_error", want: 500, mutate: func(c *APIController) {
			c.ImageStore = &rmdImageStore{labelsErr: errors.New("db")}
		}},
		{name: "success", want: 200, mutate: func(c *APIController) {
			c.ImageStore = &rmdImageStore{labels: []string{"a", "b"}, labelCount: 2}
		}},
	})
}

func TestRmdArtifactStats(t *testing.T) {
	space := artifact.GetArtifactStatsForSpaceRequestObject{SpaceRef: "root"}
	reg := artifact.GetArtifactStatsForRegistryRequestObject{RegistryRef: "root/reg"}
	rmdRunPrelude(t, rmdPrelude{
		name: "statsspace", base: 400, space: 400, forbidden: 403,
		call: func(c *APIController, ctx context.Context) (any, error) {
			return c.GetArtifactStatsForSpace(ctx, space)
		},
	})
	rmdRunPrelude(t, rmdPrelude{
		name: "statsregistry", base: 400, space: 400, forbidden: 403,
		call: func(c *APIController, ctx context.Context) (any, error) {
			return c.GetArtifactStatsForRegistry(ctx, reg)
		},
	})

	t.Run("not_implemented_endpoints_do_not_fail", func(t *testing.T) {
		c := rmdNewCtrl()
		_, err := c.GetArtifactStats(rmdAdminCtx(), artifact.GetArtifactStatsRequestObject{})
		require.NoError(t, err)
		_, err = c.GetArtifactStatsForSpace(rmdAdminCtx(), space)
		require.NoError(t, err)
		_, err = c.GetArtifactStatsForRegistry(rmdAdminCtx(), reg)
		require.NoError(t, err)
	})
}

func TestRmdDockerLayerHelpers(t *testing.T) {
	require.Equal(t, "0 B", GetSizeString(0))
	require.Equal(t, "1023 B", GetSizeString(1023))
	require.Equal(t, "1.00 KB", GetSizeString(KB))
	require.Equal(t, "1.50 MB", GetSizeString(MB+MB/2))
	require.Equal(t, "2.00 GB", GetSizeString(2*GB))
}
