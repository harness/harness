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

package npm_test

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/harness/gitness/app/api/usererror"
	npmctrl "github.com/harness/gitness/registry/app/api/controller/pkg/npm"
	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	npmmeta "github.com/harness/gitness/registry/app/metadata/npm"
	"github.com/harness/gitness/registry/app/pkg"
	"github.com/harness/gitness/registry/app/pkg/base"
	"github.com/harness/gitness/registry/app/pkg/commons"
	npmpkg "github.com/harness/gitness/registry/app/pkg/npm"
	"github.com/harness/gitness/registry/app/pkg/quarantine"
	npmtype "github.com/harness/gitness/registry/app/pkg/types/npm"
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
	_ context.Context, _ int64, _ string, _ string, _ *artifact.ArtifactType,
) error {
	return q.err
}

type rchNpmRegistry struct {
	npmpkg.Registry
	err    error
	tags   map[string]string
	exists bool
}

func (r *rchNpmRegistry) GetArtifactType() artifact.RegistryType { return artifact.RegistryTypeVIRTUAL }

func (r *rchNpmRegistry) GetPackageTypes() []artifact.PackageType {
	return []artifact.PackageType{artifact.PackageTypeNPM}
}

func (r *rchNpmRegistry) UploadPackageFile(
	_ context.Context, _ npmtype.ArtifactInfo, _ io.ReadCloser,
) (*commons.ResponseHeaders, string, error) {
	return &commons.ResponseHeaders{Code: 201}, "sha", r.err
}

func (r *rchNpmRegistry) DownloadPackageFile(
	_ context.Context, _ npmtype.ArtifactInfo,
) (*commons.ResponseHeaders, *storage.FileReader, io.ReadCloser, string, error) {
	return &commons.ResponseHeaders{Code: 200}, nil, io.NopCloser(strings.NewReader("x")), "http://r", r.err
}

func (r *rchNpmRegistry) GetPackageMetadata(
	_ context.Context, _ npmtype.ArtifactInfo,
) (npmmeta.PackageMetadata, error) {
	return npmmeta.PackageMetadata{Name: "pkg"}, r.err
}

func (r *rchNpmRegistry) HeadPackageMetadata(_ context.Context, _ npmtype.ArtifactInfo) (bool, error) {
	return r.exists, r.err
}

func (r *rchNpmRegistry) ListTags(_ context.Context, _ npmtype.ArtifactInfo) (map[string]string, error) {
	return r.tags, r.err
}

func (r *rchNpmRegistry) AddTag(_ context.Context, _ npmtype.ArtifactInfo) (map[string]string, error) {
	return r.tags, r.err
}

func (r *rchNpmRegistry) DeleteTag(_ context.Context, _ npmtype.ArtifactInfo) (map[string]string, error) {
	return r.tags, r.err
}

func (r *rchNpmRegistry) DeletePackage(_ context.Context, _ npmtype.ArtifactInfo) error { return r.err }
func (r *rchNpmRegistry) DeleteVersion(_ context.Context, _ npmtype.ArtifactInfo) error { return r.err }

func (r *rchNpmRegistry) SearchPackage(
	_ context.Context, _ npmtype.ArtifactInfo, _ int, _ int,
) (*npmmeta.PackageSearch, error) {
	return &npmmeta.PackageSearch{Total: 3}, r.err
}

func rchNewInfo() *npmtype.ArtifactInfo {
	return &npmtype.ArtifactInfo{
		ArtifactInfo: pkg.ArtifactInfo{
			BaseInfo:      &pkg.BaseInfo{ParentID: 5},
			RegIdentifier: "reg",
			Image:         "left-pad",
		},
		Version:  "1.0.0",
		Filename: "left-pad-1.0.0.tgz",
	}
}

type rchEnv struct {
	ctrl npmctrl.Controller
	reg  *rchNpmRegistry
}

func rchSetup(t *testing.T, daoErr error, regErr error, quarantineErr error) rchEnv {
	t.Helper()
	fake := &rchNpmRegistry{
		err:    regErr,
		tags:   map[string]string{"latest": "1.0.0"},
		exists: true,
	}
	base.Register(fake)
	t.Cleanup(func() { delete(base.TypeRegistry, "NPM:VIRTUAL") })

	dao := &rchRegistryDao{
		reg: &registrytypes.Registry{
			ID: 9, Name: "reg", ParentID: 5,
			PackageType: artifact.PackageTypeNPM, Type: artifact.RegistryTypeVIRTUAL,
		},
		err: daoErr,
	}
	var qf rchQuarantine
	qf.err = quarantineErr
	ctrl := npmctrl.NewController(nil, dao, nil, nil, nil, nil, nil, nil, nil, nil, qf, nil)
	return rchEnv{ctrl: ctrl, reg: fake}
}

type rchOp struct {
	name string
	call func(c npmctrl.Controller, info *npmtype.ArtifactInfo) error
}

func rchOps() []rchOp {
	ctx := context.Background()
	return []rchOp{
		{"upload", func(c npmctrl.Controller, i *npmtype.ArtifactInfo) error {
			return c.UploadPackageFile(ctx, i, io.NopCloser(strings.NewReader("data"))).GetError()
		}},
		{"download", func(c npmctrl.Controller, i *npmtype.ArtifactInfo) error {
			return c.DownloadPackageFile(ctx, i).GetError()
		}},
		{"metadata", func(c npmctrl.Controller, i *npmtype.ArtifactInfo) error {
			return c.GetPackageMetadata(ctx, i).GetError()
		}},
		{"head", func(c npmctrl.Controller, i *npmtype.ArtifactInfo) error {
			return c.HeadPackageFileByName(ctx, i).GetError()
		}},
		{"list tags", func(c npmctrl.Controller, i *npmtype.ArtifactInfo) error {
			return c.ListTags(ctx, i).GetError()
		}},
		{"add tag", func(c npmctrl.Controller, i *npmtype.ArtifactInfo) error {
			return c.AddTag(ctx, i).GetError()
		}},
		{"delete tag", func(c npmctrl.Controller, i *npmtype.ArtifactInfo) error {
			return c.DeleteTag(ctx, i).GetError()
		}},
		{"delete version", func(c npmctrl.Controller, i *npmtype.ArtifactInfo) error {
			return c.DeleteVersion(ctx, i).GetError()
		}},
		{"delete package", func(c npmctrl.Controller, i *npmtype.ArtifactInfo) error {
			return c.DeletePackage(ctx, i).GetError()
		}},
		{"search", func(c npmctrl.Controller, i *npmtype.ArtifactInfo) error {
			return c.SearchPackage(ctx, i, 10, 0).GetError()
		}},
	}
}

func TestRchNpmControllerSuccess(t *testing.T) {
	for _, op := range rchOps() {
		t.Run(op.name, func(t *testing.T) {
			env := rchSetup(t, nil, nil, nil)
			info := rchNewInfo()
			require.NoError(t, op.call(env.ctrl, info))
			require.Equal(t, int64(9), info.RegistryID)
		})
	}
}

func TestRchNpmControllerRegistryError(t *testing.T) {
	regErr := errors.New("backend failed")
	for _, op := range rchOps() {
		t.Run(op.name, func(t *testing.T) {
			env := rchSetup(t, nil, regErr, nil)
			require.ErrorIs(t, op.call(env.ctrl, rchNewInfo()), regErr)
		})
	}
}

func TestRchNpmControllerRegistryNotFound(t *testing.T) {
	for _, op := range rchOps() {
		t.Run(op.name, func(t *testing.T) {
			env := rchSetup(t, errors.New("missing"), nil, nil)
			require.Error(t, op.call(env.ctrl, rchNewInfo()))
		})
	}
}

func TestRchNpmControllerQuarantined(t *testing.T) {
	env := rchSetup(t, nil, nil, usererror.ErrQuarantinedArtifact)
	err := env.ctrl.DownloadPackageFile(context.Background(), rchNewInfo()).GetError()
	require.ErrorIs(t, err, usererror.ErrQuarantinedArtifact)
}

func TestRchNpmControllerResponsePayloads(t *testing.T) {
	ctx := context.Background()
	env := rchSetup(t, nil, nil, nil)

	tags := env.ctrl.ListTags(ctx, rchNewInfo())
	require.Equal(t, map[string]string{"latest": "1.0.0"}, tags.Tags)

	search := env.ctrl.SearchPackage(ctx, rchNewInfo(), 10, 0)
	require.NotNil(t, search.Artifacts)
	require.Equal(t, int64(3), search.Artifacts.Total)

	head := env.ctrl.HeadPackageFileByName(ctx, rchNewInfo())
	require.True(t, head.Exists)

	meta := env.ctrl.GetPackageMetadata(ctx, rchNewInfo())
	require.Equal(t, "pkg", meta.PackageMetadata.Name)

	up := env.ctrl.UploadPackageFile(ctx, rchNewInfo(), io.NopCloser(strings.NewReader("d")))
	require.Equal(t, "sha", up.Sha256)

	dl := env.ctrl.DownloadPackageFile(ctx, rchNewInfo())
	require.Equal(t, "http://r", dl.RedirectURL)
	require.NotNil(t, dl.ReadCloser)
	require.NoError(t, dl.ReadCloser.Close())
}
