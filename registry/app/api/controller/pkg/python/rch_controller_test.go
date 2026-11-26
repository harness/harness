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

package python_test

import (
	"context"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"testing"

	"github.com/harness/gitness/app/api/usererror"
	pythonctrl "github.com/harness/gitness/registry/app/api/controller/pkg/python"
	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/registry/app/pkg"
	"github.com/harness/gitness/registry/app/pkg/base"
	"github.com/harness/gitness/registry/app/pkg/commons"
	pythonpkg "github.com/harness/gitness/registry/app/pkg/python"
	"github.com/harness/gitness/registry/app/pkg/quarantine"
	pythontype "github.com/harness/gitness/registry/app/pkg/types/python"
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

type rchPythonRegistry struct {
	pythonpkg.Registry
	err error
}

func (r *rchPythonRegistry) GetArtifactType() artifact.RegistryType {
	return artifact.RegistryTypeVIRTUAL
}

func (r *rchPythonRegistry) GetPackageTypes() []artifact.PackageType {
	return []artifact.PackageType{artifact.PackageTypePYTHON}
}

func (r *rchPythonRegistry) GetPackageMetadata(
	context.Context, pythontype.ArtifactInfo,
) (pythontype.PackageMetadata, error) {
	return pythontype.PackageMetadata{Name: "requests"}, r.err
}

func (r *rchPythonRegistry) UploadPackageFile(
	_ context.Context, _ pythontype.ArtifactInfo, _ multipart.File, filename string,
) (*commons.ResponseHeaders, string, error) {
	return &commons.ResponseHeaders{Code: http.StatusCreated}, "sha-" + filename, r.err
}

func (r *rchPythonRegistry) DownloadPackageFile(
	context.Context, pythontype.ArtifactInfo,
) (*commons.ResponseHeaders, *storage.FileReader, io.ReadCloser, string, error) {
	return &commons.ResponseHeaders{Code: http.StatusOK}, nil, io.NopCloser(strings.NewReader("whl")), "", r.err
}

func rchSetup(t *testing.T, daoErr, regErr, quarantineErr error) pythonctrl.Controller {
	t.Helper()
	base.Register(&rchPythonRegistry{err: regErr})
	t.Cleanup(func() { delete(base.TypeRegistry, "PYTHON:VIRTUAL") })
	dao := &rchRegistryDao{
		reg: &registrytypes.Registry{
			ID: 9, Name: "reg", ParentID: 5,
			PackageType: artifact.PackageTypePYTHON, Type: artifact.RegistryTypeVIRTUAL,
		},
		err: daoErr,
	}
	return pythonctrl.NewController(
		nil, dao, nil, nil, nil, nil, nil, nil, nil, rchQuarantine{err: quarantineErr}, nil,
	)
}

func rchInfo() pythontype.ArtifactInfo {
	return pythontype.ArtifactInfo{
		ArtifactInfo: pkg.ArtifactInfo{
			BaseInfo:      &pkg.BaseInfo{ParentID: 5},
			RegIdentifier: "reg",
			Image:         "requests",
		},
		Version:  "2.0.0",
		Filename: "requests-2.0.0.whl",
	}
}

type rchOp struct {
	name string
	call func(c pythonctrl.Controller) error
}

func rchOps() []rchOp {
	ctx := context.Background()
	return []rchOp{
		{"metadata", func(c pythonctrl.Controller) error { return c.GetPackageMetadata(ctx, rchInfo()).GetError() }},
		{"download", func(c pythonctrl.Controller) error { return c.DownloadPackageFile(ctx, rchInfo()).GetError() }},
		{"upload", func(c pythonctrl.Controller) error {
			header := &multipart.FileHeader{Filename: "requests-2.0.0.whl"}
			return c.UploadPackageFile(ctx, rchInfo(), nil, header).GetError()
		}},
	}
}

func TestRchPythonControllerSuccess(t *testing.T) {
	for _, op := range rchOps() {
		t.Run(op.name, func(t *testing.T) {
			require.NoError(t, op.call(rchSetup(t, nil, nil, nil)))
		})
	}
}

func TestRchPythonControllerRegistryError(t *testing.T) {
	regErr := errors.New("backend failed")
	for _, op := range rchOps() {
		t.Run(op.name, func(t *testing.T) {
			require.ErrorIs(t, op.call(rchSetup(t, nil, regErr, nil)), regErr)
		})
	}
}

func TestRchPythonControllerRegistryNotFound(t *testing.T) {
	for _, op := range rchOps() {
		t.Run(op.name, func(t *testing.T) {
			require.Error(t, op.call(rchSetup(t, errors.New("missing"), nil, nil)))
		})
	}
}

func TestRchPythonControllerQuarantinedDownload(t *testing.T) {
	c := rchSetup(t, nil, nil, usererror.ErrQuarantinedArtifact)
	err := c.DownloadPackageFile(context.Background(), rchInfo()).GetError()
	require.ErrorIs(t, err, usererror.ErrQuarantinedArtifact)
}

func TestRchPythonControllerPayloads(t *testing.T) {
	ctx := context.Background()
	c := rchSetup(t, nil, nil, nil)

	require.Equal(t, "requests", c.GetPackageMetadata(ctx, rchInfo()).PackageMetadata.Name)

	up := c.UploadPackageFile(ctx, rchInfo(), nil, &multipart.FileHeader{Filename: "pkg.whl"})
	require.Equal(t, "sha-pkg.whl", up.Sha256)

	dl := c.DownloadPackageFile(ctx, rchInfo())
	require.NotNil(t, dl.ReadCloser)
	require.NoError(t, dl.ReadCloser.Close())
}
