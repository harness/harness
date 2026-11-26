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

package huggingface_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/harness/gitness/app/api/usererror"
	hfctrl "github.com/harness/gitness/registry/app/api/controller/pkg/huggingface"
	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/registry/app/pkg"
	"github.com/harness/gitness/registry/app/pkg/base"
	"github.com/harness/gitness/registry/app/pkg/commons"
	hfpkg "github.com/harness/gitness/registry/app/pkg/huggingface"
	"github.com/harness/gitness/registry/app/pkg/quarantine"
	hftype "github.com/harness/gitness/registry/app/pkg/types/huggingface"
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

type rchQuarantine struct {
	quarantine.Finder
	err error
}

func (q rchQuarantine) CheckArtifactQuarantineStatus(
	context.Context, int64, string, string, *artifact.ArtifactType,
) error {
	return q.err
}

type rchHFRegistry struct {
	hfpkg.Registry
	err error
}

func (r *rchHFRegistry) GetArtifactType() artifact.RegistryType { return artifact.RegistryTypeVIRTUAL }

func (r *rchHFRegistry) GetPackageTypes() []artifact.PackageType {
	return []artifact.PackageType{artifact.PackageTypeHUGGINGFACE}
}

func (r *rchHFRegistry) headers() *commons.ResponseHeaders {
	return &commons.ResponseHeaders{Code: http.StatusOK}
}

func (r *rchHFRegistry) ValidateYaml(
	context.Context, hftype.ArtifactInfo, io.ReadCloser,
) (*commons.ResponseHeaders, *hftype.ValidateYamlResponse, error) {
	return r.headers(), &hftype.ValidateYamlResponse{}, r.err
}

func (r *rchHFRegistry) PreUpload(
	context.Context, hftype.ArtifactInfo, io.ReadCloser,
) (*commons.ResponseHeaders, *hftype.PreUploadResponse, error) {
	return r.headers(), &hftype.PreUploadResponse{}, r.err
}

func (r *rchHFRegistry) RevisionInfo(
	context.Context, hftype.ArtifactInfo, map[string][]string,
) (*commons.ResponseHeaders, *hftype.RevisionInfoResponse, error) {
	return r.headers(), &hftype.RevisionInfoResponse{XetEnabled: true}, r.err
}

func (r *rchHFRegistry) LfsInfo(
	context.Context, hftype.ArtifactInfo, io.ReadCloser, string,
) (*commons.ResponseHeaders, *hftype.LfsInfoResponse, error) {
	return r.headers(), &hftype.LfsInfoResponse{}, r.err
}

func (r *rchHFRegistry) LfsUpload(
	context.Context, hftype.ArtifactInfo, io.ReadCloser,
) (*commons.ResponseHeaders, *hftype.LfsUploadResponse, error) {
	return r.headers(), &hftype.LfsUploadResponse{Success: true}, r.err
}

func (r *rchHFRegistry) LfsVerify(
	context.Context, hftype.ArtifactInfo, io.ReadCloser,
) (*commons.ResponseHeaders, *hftype.LfsVerifyResponse, error) {
	return r.headers(), &hftype.LfsVerifyResponse{Success: true}, r.err
}

func (r *rchHFRegistry) CommitRevision(
	context.Context, hftype.ArtifactInfo, io.ReadCloser,
) (*commons.ResponseHeaders, *hftype.CommitRevisionResponse, error) {
	return r.headers(), &hftype.CommitRevisionResponse{OID: "abc", Success: true}, r.err
}

func (r *rchHFRegistry) HeadFile(context.Context, hftype.ArtifactInfo, string) (*commons.ResponseHeaders, error) {
	return r.headers(), r.err
}

func (r *rchHFRegistry) DownloadFile(
	context.Context, hftype.ArtifactInfo, string,
) (*commons.ResponseHeaders, *storage.FileReader, string, error) {
	return r.headers(), nil, "http://redirect", r.err
}

func rchSetup(t *testing.T, daoErr, regErr, quarantineErr error) hfctrl.Controller {
	t.Helper()
	base.Register(&rchHFRegistry{err: regErr})
	t.Cleanup(func() { delete(base.TypeRegistry, "HUGGINGFACE:VIRTUAL") })
	dao := &rchRegistryDao{
		reg: &registrytypes.Registry{
			ID: 9, Name: "reg", ParentID: 5,
			PackageType: artifact.PackageTypeHUGGINGFACE, Type: artifact.RegistryTypeVIRTUAL,
		},
		err: daoErr,
	}
	return hfctrl.NewController(nil, dao, nil, nil, nil, nil, nil, nil, rchQuarantine{err: quarantineErr})
}

func rchInfo() hftype.ArtifactInfo {
	return hftype.ArtifactInfo{
		ArtifactInfo: pkg.ArtifactInfo{
			BaseInfo:      &pkg.BaseInfo{ParentID: 5},
			RegIdentifier: "reg",
			Image:         "org/model",
		},
		Repo:     "org/model",
		Revision: "main",
	}
}

type rchOp struct {
	name string
	call func(c hfctrl.Controller) error
}

func rchBody() io.ReadCloser { return io.NopCloser(strings.NewReader("{}")) }

func rchOps() []rchOp {
	ctx := context.Background()
	return []rchOp{
		{"validate yaml", func(c hfctrl.Controller) error { return c.ValidateYaml(ctx, rchInfo(), rchBody()).GetError() }},
		{"pre upload", func(c hfctrl.Controller) error { return c.PreUpload(ctx, rchInfo(), rchBody()).GetError() }},
		{"revision info", func(c hfctrl.Controller) error {
			return c.RevisionInfo(ctx, rchInfo(), map[string][]string{"blobs": {"true"}}).GetError()
		}},
		{"lfs info", func(c hfctrl.Controller) error {
			return c.LfsInfo(ctx, rchInfo(), rchBody(), "token").GetError()
		}},
		{"lfs upload", func(c hfctrl.Controller) error { return c.LfsUpload(ctx, rchInfo(), rchBody()).GetError() }},
		{"lfs verify", func(c hfctrl.Controller) error { return c.LfsVerify(ctx, rchInfo(), rchBody()).GetError() }},
		{"commit revision", func(c hfctrl.Controller) error {
			return c.CommitRevision(ctx, rchInfo(), rchBody()).GetError()
		}},
		{"head file", func(c hfctrl.Controller) error { return c.HeadFile(ctx, rchInfo(), "config.json").GetError() }},
		{"download file", func(c hfctrl.Controller) error {
			return c.DownloadFile(ctx, rchInfo(), "config.json").GetError()
		}},
	}
}

func TestRchHFControllerSuccess(t *testing.T) {
	for _, op := range rchOps() {
		t.Run(op.name, func(t *testing.T) {
			require.NoError(t, op.call(rchSetup(t, nil, nil, nil)))
		})
	}
}

func TestRchHFControllerRegistryError(t *testing.T) {
	regErr := errors.New("backend failed")
	for _, op := range rchOps() {
		t.Run(op.name, func(t *testing.T) {
			require.ErrorIs(t, op.call(rchSetup(t, nil, regErr, nil)), regErr)
		})
	}
}

func TestRchHFControllerRegistryNotFound(t *testing.T) {
	for _, op := range rchOps() {
		t.Run(op.name, func(t *testing.T) {
			require.Error(t, op.call(rchSetup(t, errors.New("missing"), nil, nil)))
		})
	}
}

func TestRchHFControllerQuarantinedDownload(t *testing.T) {
	c := rchSetup(t, nil, nil, usererror.ErrQuarantinedArtifact)
	err := c.DownloadFile(context.Background(), rchInfo(), "config.json").GetError()
	require.ErrorIs(t, err, usererror.ErrQuarantinedArtifact)
}

func TestRchHFControllerPayloads(t *testing.T) {
	ctx := context.Background()
	c := rchSetup(t, nil, nil, nil)

	require.True(t, c.RevisionInfo(ctx, rchInfo(), nil).Response.XetEnabled)
	require.True(t, c.LfsUpload(ctx, rchInfo(), rchBody()).Response.Success)
	require.True(t, c.LfsVerify(ctx, rchInfo(), rchBody()).Response.Success)
	require.Equal(t, "abc", c.CommitRevision(ctx, rchInfo(), rchBody()).Response.OID)
	require.Equal(t, "http://redirect", c.DownloadFile(ctx, rchInfo(), "config.json").RedirectURL)
	require.Equal(t, http.StatusOK, c.HeadFile(ctx, rchInfo(), "config.json").ResponseHeaders.Code)
}
