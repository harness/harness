// Copyright 2026 Harness, Inc.
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

package rpm_test

import (
	"context"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"testing"

	rpmctrl "github.com/harness/gitness/registry/app/api/controller/pkg/rpm"
	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/registry/app/pkg"
	"github.com/harness/gitness/registry/app/pkg/base"
	"github.com/harness/gitness/registry/app/pkg/commons"
	rpmpkg "github.com/harness/gitness/registry/app/pkg/rpm"
	rpmtype "github.com/harness/gitness/registry/app/pkg/types/rpm"
	"github.com/harness/gitness/registry/app/storage"
	"github.com/harness/gitness/registry/app/store"
	registrytypes "github.com/harness/gitness/registry/types"

	"github.com/stretchr/testify/require"
)

type rchRegistryDao struct {
	store.RegistryRepository
	reg *registrytypes.Registry
	err error
}

func (d *rchRegistryDao) GetByParentIDAndName(
	_ context.Context, _ int64, _ string, _ ...registrytypes.QueryOption,
) (*registrytypes.Registry, error) {
	return d.reg, d.err
}

type rchRPMRegistry struct {
	rpmpkg.Registry
	err error
}

func (r *rchRPMRegistry) GetArtifactType() artifact.RegistryType { return artifact.RegistryTypeVIRTUAL }

func (r *rchRPMRegistry) GetPackageTypes() []artifact.PackageType {
	return []artifact.PackageType{artifact.PackageTypeRPM}
}

func (r *rchRPMRegistry) UploadPackageFile(
	context.Context, rpmtype.ArtifactInfo, io.Reader, string,
) (*commons.ResponseHeaders, string, error) {
	return &commons.ResponseHeaders{Code: http.StatusCreated}, "sha", r.err
}

func (r *rchRPMRegistry) DownloadPackageFile(
	context.Context, rpmtype.ArtifactInfo,
) (*commons.ResponseHeaders, *storage.FileReader, io.ReadCloser, string, error) {
	return &commons.ResponseHeaders{Code: http.StatusOK}, nil, io.NopCloser(strings.NewReader("rpm")), "", r.err
}

func (r *rchRPMRegistry) GetRepoData(
	context.Context, rpmtype.ArtifactInfo, string,
) (*commons.ResponseHeaders, *storage.FileReader, io.ReadCloser, string, error) {
	return &commons.ResponseHeaders{Code: http.StatusOK}, nil, nil, "http://redirect", r.err
}

func rchReg() *registrytypes.Registry {
	return &registrytypes.Registry{
		ID: 9, Name: "reg", ParentID: 5,
		PackageType: artifact.PackageTypeRPM, Type: artifact.RegistryTypeVIRTUAL,
	}
}

func rchSetup(t *testing.T, daoErr, regErr error) rpmctrl.Controller {
	t.Helper()
	base.Register(&rchRPMRegistry{err: regErr})
	t.Cleanup(func() { delete(base.TypeRegistry, "RPM:VIRTUAL") })
	dao := &rchRegistryDao{reg: rchReg(), err: daoErr}
	return rpmctrl.NewController(nil, dao, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
}

func rchInfo() rpmtype.ArtifactInfo {
	return rpmtype.ArtifactInfo{
		ArtifactInfo: pkg.ArtifactInfo{
			BaseInfo:      &pkg.BaseInfo{ParentID: 5},
			RegIdentifier: "reg",
			Image:         "bash",
			Registry:      *rchReg(),
		},
		Version:  "5.2.x86_64",
		FileName: "bash.rpm",
	}
}

func TestRchRPMControllerUpload(t *testing.T) {
	ctx := context.Background()

	resp := rchSetup(t, nil, nil).UploadPackageFile(ctx, rchInfo(), multipart.Part{}, "bash.rpm")
	require.NoError(t, resp.GetError())
	require.Equal(t, "sha", resp.Sha256)
	require.Equal(t, http.StatusCreated, resp.ResponseHeaders.Code)

	regErr := errors.New("backend failed")
	resp = rchSetup(t, nil, regErr).UploadPackageFile(ctx, rchInfo(), multipart.Part{}, "bash.rpm")
	require.ErrorIs(t, resp.GetError(), regErr)

	resp = rchSetup(t, errors.New("missing"), nil).UploadPackageFile(ctx, rchInfo(), multipart.Part{}, "bash.rpm")
	require.Error(t, resp.GetError())
}

func TestRchRPMControllerDownload(t *testing.T) {
	ctx := context.Background()

	resp := rchSetup(t, nil, nil).DownloadPackageFile(ctx, rchInfo())
	require.NoError(t, resp.GetError())
	require.NotNil(t, resp.ReadCloser)
	require.NoError(t, resp.ReadCloser.Close())

	regErr := errors.New("backend failed")
	require.ErrorIs(t, rchSetup(t, nil, regErr).DownloadPackageFile(ctx, rchInfo()).GetError(), regErr)
	require.Error(t, rchSetup(t, errors.New("missing"), nil).DownloadPackageFile(ctx, rchInfo()).GetError())
}

func TestRchRPMControllerGetRepoData(t *testing.T) {
	ctx := context.Background()

	resp := rchSetup(t, nil, nil).GetRepoData(ctx, rchInfo(), "repomd.xml")
	require.NoError(t, resp.GetError())
	require.Equal(t, "http://redirect", resp.RedirectURL)

	regErr := errors.New("backend failed")
	require.ErrorIs(t, rchSetup(t, nil, regErr).GetRepoData(ctx, rchInfo(), "repomd.xml").GetError(), regErr)

	// Without a registered RPM registry for the type the call is rejected.
	delete(base.TypeRegistry, "RPM:VIRTUAL")
	resp = rchSetup(t, nil, nil).GetRepoData(ctx, rpmtype.ArtifactInfo{}, "repomd.xml")
	require.Error(t, resp.GetError())
}
