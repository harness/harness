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

package gopackage_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/harness/gitness/app/api/usererror"
	gopackagectrl "github.com/harness/gitness/registry/app/api/controller/pkg/gopackage"
	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/registry/app/pkg"
	"github.com/harness/gitness/registry/app/pkg/base"
	"github.com/harness/gitness/registry/app/pkg/commons"
	gopkg "github.com/harness/gitness/registry/app/pkg/gopackage"
	"github.com/harness/gitness/registry/app/pkg/quarantine"
	gotype "github.com/harness/gitness/registry/app/pkg/types/gopackage"
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

type rchGoRegistry struct {
	gopkg.Registry
	err error
}

func (r *rchGoRegistry) GetArtifactType() artifact.RegistryType { return artifact.RegistryTypeVIRTUAL }

func (r *rchGoRegistry) GetPackageTypes() []artifact.PackageType {
	return []artifact.PackageType{artifact.PackageTypeGO}
}

func (r *rchGoRegistry) UploadPackage(
	context.Context, gotype.ArtifactInfo, io.ReadCloser, io.ReadCloser,
) (*commons.ResponseHeaders, error) {
	return &commons.ResponseHeaders{Code: http.StatusCreated}, r.err
}

func (r *rchGoRegistry) DownloadPackageFile(
	context.Context, gotype.ArtifactInfo,
) (*commons.ResponseHeaders, *storage.FileReader, io.ReadCloser, string, error) {
	return &commons.ResponseHeaders{Code: http.StatusOK}, nil, io.NopCloser(strings.NewReader("zip")), "", r.err
}

func (r *rchGoRegistry) RegeneratePackageIndex(
	context.Context, gotype.ArtifactInfo,
) (*commons.ResponseHeaders, error) {
	return &commons.ResponseHeaders{Code: http.StatusOK}, r.err
}

func (r *rchGoRegistry) RegeneratePackageMetadata(
	context.Context, gotype.ArtifactInfo,
) (*commons.ResponseHeaders, error) {
	return &commons.ResponseHeaders{Code: http.StatusOK}, r.err
}

func rchSetup(t *testing.T, daoErr, regErr, quarantineErr error) gopackagectrl.Controller {
	t.Helper()
	base.Register(&rchGoRegistry{err: regErr})
	t.Cleanup(func() { delete(base.TypeRegistry, "GO:VIRTUAL") })
	dao := &rchRegistryDao{
		reg: &registrytypes.Registry{
			ID: 9, Name: "reg", ParentID: 5,
			PackageType: artifact.PackageTypeGO, Type: artifact.RegistryTypeVIRTUAL,
		},
		err: daoErr,
	}
	return gopackagectrl.NewController(
		nil, dao, nil, nil, nil, nil, nil, nil, nil, nil, rchQuarantine{err: quarantineErr}, nil,
	)
}

func rchInfo() *gotype.ArtifactInfo {
	return &gotype.ArtifactInfo{
		ArtifactInfo: pkg.ArtifactInfo{
			BaseInfo:      &pkg.BaseInfo{ParentID: 5},
			RegIdentifier: "reg",
			Image:         "example.com/hello",
		},
		Version:  "v1.0.0",
		FileName: "v1.0.0.zip",
	}
}

type rchOp struct {
	name string
	call func(c gopackagectrl.Controller) error
}

func rchOps() []rchOp {
	ctx := context.Background()
	return []rchOp{
		{"upload", func(c gopackagectrl.Controller) error {
			rc := io.NopCloser(strings.NewReader("x"))
			return c.UploadPackage(ctx, rchInfo(), rc, rc).GetError()
		}},
		{"download", func(c gopackagectrl.Controller) error { return c.DownloadPackageFile(ctx, rchInfo()).GetError() }},
		{"regenerate index", func(c gopackagectrl.Controller) error {
			return c.RegeneratePackageIndex(ctx, rchInfo()).GetError()
		}},
		{"regenerate metadata", func(c gopackagectrl.Controller) error {
			return c.RegeneratePackageMetadata(ctx, rchInfo()).GetError()
		}},
	}
}

func TestRchGoControllerSuccess(t *testing.T) {
	for _, op := range rchOps() {
		t.Run(op.name, func(t *testing.T) {
			require.NoError(t, op.call(rchSetup(t, nil, nil, nil)))
		})
	}
}

func TestRchGoControllerRegistryError(t *testing.T) {
	regErr := errors.New("backend failed")
	for _, op := range rchOps() {
		t.Run(op.name, func(t *testing.T) {
			require.ErrorIs(t, op.call(rchSetup(t, nil, regErr, nil)), regErr)
		})
	}
}

func TestRchGoControllerRegistryNotFound(t *testing.T) {
	for _, op := range rchOps() {
		t.Run(op.name, func(t *testing.T) {
			require.Error(t, op.call(rchSetup(t, errors.New("missing"), nil, nil)))
		})
	}
}

func TestRchGoControllerQuarantinedDownload(t *testing.T) {
	c := rchSetup(t, nil, nil, usererror.ErrQuarantinedArtifact)
	err := c.DownloadPackageFile(context.Background(), rchInfo()).GetError()
	require.ErrorIs(t, err, usererror.ErrQuarantinedArtifact)
}

func TestRchGoControllerResponses(t *testing.T) {
	ctx := context.Background()
	c := rchSetup(t, nil, nil, nil)

	rc := io.NopCloser(strings.NewReader("x"))
	upload := c.UploadPackage(ctx, rchInfo(), rc, rc)
	require.Equal(t, "success", upload.Status)
	require.Equal(t, "example.com/hello", upload.Image)
	require.Equal(t, "v1.0.0", upload.Version)
	require.Equal(t, http.StatusOK, upload.ResponseHeaders.Code)

	require.True(t, c.RegeneratePackageIndex(ctx, rchInfo()).Ok)
	require.True(t, c.RegeneratePackageMetadata(ctx, rchInfo()).Ok)

	dl := c.DownloadPackageFile(ctx, rchInfo())
	require.NotNil(t, dl.ReadCloser)
	require.NoError(t, dl.ReadCloser.Close())
}
