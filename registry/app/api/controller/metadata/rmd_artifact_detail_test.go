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
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/registry/types"

	"github.com/stretchr/testify/require"
)

func TestRmdGetArtifactDetails(t *testing.T) {
	req := artifact.GetArtifactDetailsRequestObject{RegistryRef: "root/reg", Artifact: "img", Version: "1.0"}
	call := func(r artifact.GetArtifactDetailsRequestObject) rmdCallFn {
		return func(c *APIController, ctx context.Context) (any, error) { return c.GetArtifactDetails(ctx, r) }
	}
	rmdRunPrelude(t, rmdPrelude{name: "details", call: call(req), base: 400, space: 400, anon: 401, forbidden: 403})

	newArt := func(meta string) *types.Artifact {
		return &types.Artifact{
			ID: 3, ImageID: 9, Version: "1.0", Metadata: json.RawMessage(meta),
			CreatedAt: time.UnixMilli(1), UpdatedAt: time.UnixMilli(2),
		}
	}
	setup := func(pkg artifact.PackageType, as rmdArtifactStore, qr rmdQuarantineRepo,
		pw rmdPkgWrapper) func(*APIController) {
		return rmdWithAll(rmdRegistryRepo(pkg), func(c *APIController) {
			c.ImageStore = &rmdImageStore{image: rmdImage()}
			c.ArtifactStore = as
			c.QuarantineArtifactRepository = qr
			c.PackageWrapper = pw
		})
	}

	metadataByPackage := map[artifact.PackageType]string{
		artifact.PackageTypeMAVEN:       `{"size":2048,"files":[{"size":1}]}`,
		artifact.PackageTypeGENERIC:     `{"description":"generic"}`,
		artifact.PackageTypePYTHON:      `{"summary":"py"}`,
		artifact.PackageTypeNPM:         `{"size":10,"name":"n"}`,
		artifact.PackageTypeRPM:         `{"size":10,"file_metadata":{"files":["a"]}}`,
		artifact.PackageTypeNUGET:       `{"size":10}`,
		artifact.PackageTypeHUGGINGFACE: `{"size":10}`,
		artifact.PackageTypeGO:          `{"size":10}`,
	}
	for pkg, meta := range metadataByPackage {
		t.Run("success_"+string(pkg), func(t *testing.T) {
			as := rmdArtifactStore{byName: newArt(meta), downloadCount: 5}
			resp, err := rmdNewCtrl(setup(pkg, as, rmdQuarantineRepo{}, rmdPkgWrapper{})).
				GetArtifactDetails(rmdAdminCtx(), req)
			require.NoError(t, err)
			ok, isOK := resp.(artifact.GetArtifactDetails200JSONResponse)
			require.True(t, isOK)
			require.Equal(t, artifact.StatusSUCCESS, ok.Status)
			require.False(t, *ok.Data.IsQuarantined)

			bad := rmdArtifactStore{byName: newArt("not-json")}
			resp, err = rmdNewCtrl(setup(pkg, bad, rmdQuarantineRepo{}, rmdPkgWrapper{})).
				GetArtifactDetails(rmdAdminCtx(), req)
			require.NoError(t, err)
			require.Equal(t, 500, rmdStatus(t, resp), "unreadable metadata")
		})
	}

	good := rmdArtifactStore{byName: newArt(`{}`)}
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
		{name: "version_not_found", want: 404, mutate: setup(artifact.PackageTypeNPM,
			rmdArtifactStore{byNameErr: errRmdNotFound}, rmdQuarantineRepo{}, rmdPkgWrapper{})},
		{name: "version_error", want: 500, mutate: setup(artifact.PackageTypeNPM,
			rmdArtifactStore{byNameErr: errors.New("db")}, rmdQuarantineRepo{}, rmdPkgWrapper{})},
		{name: "download_count_error", want: 500, mutate: setup(artifact.PackageTypeNPM,
			rmdArtifactStore{byName: newArt(`{}`), downloadCountErr: errors.New("db")}, rmdQuarantineRepo{},
			rmdPkgWrapper{})},
		{name: "quarantine_lookup_error", want: 500, mutate: setup(artifact.PackageTypeNPM, good,
			rmdQuarantineRepo{byPathErr: errors.New("db")}, rmdPkgWrapper{})},
		{name: "docker_has_no_extra_details", want: 200, mutate: setup(artifact.PackageTypeDOCKER, good,
			rmdQuarantineRepo{}, rmdPkgWrapper{})},
		{name: "other_package_uses_wrapper", want: 200, mutate: setup(artifact.PackageTypeCARGO, good,
			rmdQuarantineRepo{}, rmdPkgWrapper{detail: &artifact.ArtifactDetail{Version: "1.0"}})},
		{name: "wrapper_error", want: 400, mutate: setup(artifact.PackageTypeCARGO, good, rmdQuarantineRepo{},
			rmdPkgWrapper{detailErr: errors.New("unsupported")})},
	})

	t.Run("quarantine_status_is_reported", func(t *testing.T) {
		qr := rmdQuarantineRepo{byPath: []*types.QuarantineArtifact{{Reason: "bad"}}}
		resp, err := rmdNewCtrl(setup(artifact.PackageTypeNPM, good, qr, rmdPkgWrapper{})).
			GetArtifactDetails(rmdAdminCtx(), req)
		require.NoError(t, err)
		ok, isOK := resp.(artifact.GetArtifactDetails200JSONResponse)
		require.True(t, isOK)
		require.True(t, *ok.Data.IsQuarantined)
		require.Equal(t, "bad", *ok.Data.QuarantineReason)
	})

	t.Run("invalid_artifact_type", func(t *testing.T) {
		bad := artifact.GetArtifactDetailsParamsArtifactType("space")
		r := req
		r.Params.ArtifactType = &bad
		resp, err := rmdNewCtrl(rmdRegistryRepo(artifact.PackageTypeNPM)).GetArtifactDetails(rmdAdminCtx(), r)
		require.NoError(t, err)
		require.Equal(t, 400, rmdStatus(t, resp))
	})
}
