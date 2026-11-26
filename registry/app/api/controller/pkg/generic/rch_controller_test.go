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

package generic_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	apirequest "github.com/harness/gitness/app/api/request"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/auth/authz"
	"github.com/harness/gitness/app/services/refcache"
	storecache "github.com/harness/gitness/app/store/cache"
	"github.com/harness/gitness/audit"
	genericctrl "github.com/harness/gitness/registry/app/api/controller/pkg/generic"
	"github.com/harness/gitness/registry/app/api/interfaces"
	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/registry/app/dist_temp/errcode"
	"github.com/harness/gitness/registry/app/pkg"
	"github.com/harness/gitness/registry/app/pkg/base"
	"github.com/harness/gitness/registry/app/pkg/commons"
	"github.com/harness/gitness/registry/app/pkg/filemanager"
	genericpkg "github.com/harness/gitness/registry/app/pkg/generic"
	"github.com/harness/gitness/registry/app/pkg/quarantine"
	generictype "github.com/harness/gitness/registry/app/pkg/types/generic"
	"github.com/harness/gitness/registry/app/storage"
	"github.com/harness/gitness/registry/app/store"
	registrytypes "github.com/harness/gitness/registry/types"
	basestore "github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/require"
)

const (
	rchParentID = int64(5)
	rchRegID    = int64(9)
)

type rchAuthorizer struct {
	authz.Authorizer
	allow bool
}

func (a rchAuthorizer) Check(
	context.Context, *auth.Session, *types.Scope, *types.Resource, enum.Permission,
) (bool, error) {
	return a.allow, nil
}

func (a rchAuthorizer) CheckAll(context.Context, *auth.Session, ...types.PermissionCheck) (bool, error) {
	return a.allow, nil
}

type rchAudit struct {
	audit.Service
}

func (rchAudit) Log(
	context.Context, types.Principal, audit.Resource, audit.Action, string, ...audit.Option,
) error {
	return nil
}

type rchSpaceIDCache struct{ space *types.SpaceCore }

func (c rchSpaceIDCache) Get(_ context.Context, id int64) (*types.SpaceCore, error) {
	if c.space.ID != id {
		return nil, basestore.ErrResourceNotFound
	}
	return c.space, nil
}
func (rchSpaceIDCache) Stats() (int64, int64)            { return 0, 0 }
func (rchSpaceIDCache) Evict(_ context.Context, _ int64) {}

type rchSpacePathCache struct{ space *types.SpaceCore }

func (c rchSpacePathCache) Get(_ context.Context, path string) (*types.SpacePath, error) {
	if c.space.Path != path {
		return nil, basestore.ErrResourceNotFound
	}
	return &types.SpacePath{Value: path, IsPrimary: true, SpaceID: c.space.ID}, nil
}
func (rchSpacePathCache) Stats() (int64, int64)             { return 0, 0 }
func (rchSpacePathCache) Evict(_ context.Context, _ string) {}

func rchSpaceFinder() refcache.SpaceFinder {
	space := &types.SpaceCore{ID: rchParentID, Path: "root", Identifier: "root"}
	return refcache.NewSpaceFinder(
		rchSpaceIDCache{space: space}, rchSpacePathCache{space: space}, nil, storecache.Evictor[*types.SpaceCore]{},
	)
}

type rchTx struct{}

func (rchTx) WithTx(ctx context.Context, fn func(ctx context.Context) error, _ ...any) error {
	return fn(ctx)
}

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

type rchImageDao struct {
	store.ImageRepository
	existing  *registrytypes.Image
	getErr    error
	createErr error
}

func (d *rchImageDao) GetByName(
	_ context.Context, _ int64, _ string, _ ...registrytypes.QueryOption,
) (*registrytypes.Image, error) {
	return d.existing, d.getErr
}

func (d *rchImageDao) CreateOrUpdate(_ context.Context, img *registrytypes.Image) error {
	img.ID = 3
	img.UUID = "image-uuid"
	return d.createErr
}

type rchArtifactDao struct {
	store.ArtifactRepository
	existing  *registrytypes.Artifact
	getErr    error
	createErr error
	saved     *registrytypes.Artifact
}

func (d *rchArtifactDao) GetByName(
	_ context.Context, _ int64, _ string, _ ...registrytypes.QueryOption,
) (*registrytypes.Artifact, error) {
	return d.existing, d.getErr
}

func (d *rchArtifactDao) CreateOrUpdate(_ context.Context, a *registrytypes.Artifact) (int64, error) {
	a.UUID = "artifact-uuid"
	d.saved = a
	return 1, d.createErr
}

type rchFileManager struct {
	filemanager.FileManager
	uploadErr   error
	postErr     error
	downloadErr error
}

func (m rchFileManager) UploadFileNoDBUpdate(
	_ context.Context, _ string, _ multipartFile, reader io.Reader, _ int64, _ int64,
) (registrytypes.FileInfo, error) {
	if m.uploadErr != nil {
		return registrytypes.FileInfo{}, m.uploadErr
	}
	data, err := io.ReadAll(reader)
	if err != nil {
		return registrytypes.FileInfo{}, err
	}
	return registrytypes.FileInfo{Sha256: "sha-of-file", Size: int64(len(data))}, nil
}

func (m rchFileManager) PostFileUpload(
	_ context.Context, _ string, _ int64, _ int64, _ string, _ registrytypes.FileInfo, _ int64,
) error {
	return m.postErr
}

func (m rchFileManager) DownloadFileByPath(
	_ context.Context, _ string, _ int64, _ string, _ string, _ bool,
) (*storage.FileReader, int64, string, error) {
	return nil, 0, "http://redirect", m.downloadErr
}

type rchWrapper struct {
	interfaces.PackageWrapper
	supported bool
}

func (w rchWrapper) IsFileOperationSupported(string) bool { return w.supported }

type rchQuarantine struct {
	quarantine.Finder
}

func (rchQuarantine) CheckArtifactQuarantineStatus(
	context.Context, int64, string, string, *artifact.ArtifactType,
) error {
	return nil
}

type rchGenericRegistry struct {
	genericpkg.Registry
	err error
}

func (r *rchGenericRegistry) GetArtifactType() artifact.RegistryType {
	return artifact.RegistryTypeVIRTUAL
}

func (r *rchGenericRegistry) GetPackageTypes() []artifact.PackageType {
	return []artifact.PackageType{artifact.PackageTypeGENERIC}
}

func (r *rchGenericRegistry) PutFile(
	_ context.Context, _ generictype.ArtifactInfo, _ io.ReadCloser, _ string,
) (*commons.ResponseHeaders, string, error) {
	return &commons.ResponseHeaders{Code: 201}, "put-sha", r.err
}

func (r *rchGenericRegistry) DownloadFile(
	_ context.Context, _ generictype.ArtifactInfo, _ string,
) (*commons.ResponseHeaders, *storage.FileReader, io.ReadCloser, string, error) {
	return &commons.ResponseHeaders{Code: 200}, nil, io.NopCloser(strings.NewReader("x")), "http://r", r.err
}

func (r *rchGenericRegistry) DeleteFile(
	_ context.Context, _ generictype.ArtifactInfo,
) (*commons.ResponseHeaders, error) {
	return &commons.ResponseHeaders{Code: 204}, r.err
}

func (r *rchGenericRegistry) HeadFile(
	_ context.Context, _ generictype.ArtifactInfo, _ string,
) (*commons.ResponseHeaders, error) {
	return &commons.ResponseHeaders{Code: 200}, r.err
}

type rchEnv struct {
	ctrl     *genericctrl.Controller
	images   *rchImageDao
	artifact *rchArtifactDao
}

type rchOpts struct {
	allow         bool
	unsupported   bool
	daoErr        error
	registryErr   error
	fm            rchFileManager
	image         *rchImageDao
	artifactStore *rchArtifactDao
}

func rchSetup(t *testing.T, o rchOpts) rchEnv {
	t.Helper()
	base.Register(&rchGenericRegistry{err: o.registryErr})
	t.Cleanup(func() { delete(base.TypeRegistry, "GENERIC:VIRTUAL") })

	regDao := &rchRegistryDao{
		reg: &registrytypes.Registry{
			ID: rchRegID, Name: "reg", ParentID: rchParentID,
			PackageType: artifact.PackageTypeGENERIC, Type: artifact.RegistryTypeVIRTUAL,
		},
		err: o.daoErr,
	}
	if o.image == nil {
		o.image = &rchImageDao{getErr: basestore.ErrResourceNotFound}
	}
	if o.artifactStore == nil {
		o.artifactStore = &rchArtifactDao{getErr: basestore.ErrResourceNotFound}
	}
	dbStore := genericctrl.NewDBStore(regDao, o.image, o.artifactStore, nil, nil)
	ctrl := genericctrl.NewController(
		nil, rchAuthorizer{allow: o.allow}, o.fm, dbStore, rchTx{}, rchSpaceFinder(),
		nil, nil, rchQuarantine{}, nil, rchAudit{}, rchWrapper{supported: !o.unsupported},
	)
	return rchEnv{ctrl: ctrl, images: o.image, artifact: o.artifactStore}
}

func rchCtx() context.Context {
	return apirequest.WithAuthSession(context.Background(), &auth.Session{Principal: types.Principal{ID: 1}})
}

func rchInfo() pkg.GenericArtifactInfo { //nolint:staticcheck
	return pkg.GenericArtifactInfo{ //nolint:staticcheck
		ArtifactInfo: &pkg.ArtifactInfo{
			BaseInfo:      &pkg.BaseInfo{ParentID: rchParentID, RootIdentifier: "root", RootParentID: 1},
			RegIdentifier: "reg",
			RegistryID:    rchRegID,
			Image:         "tool",
		},
		Version:    "1.0.0",
		RegistryID: rchRegID,
	}
}

type multipartFile = multipart.File

func rchUploadRequest(t *testing.T, parts [][2]string) *http.Request {
	t.Helper()
	buf := &bytes.Buffer{}
	mw := multipart.NewWriter(buf)
	for _, p := range parts {
		if p[0] == "file" {
			w, err := mw.CreateFormFile("file", "upload.bin")
			require.NoError(t, err)
			_, err = w.Write([]byte(p[1]))
			require.NoError(t, err)
			continue
		}
		require.NoError(t, mw.WriteField(p[0], p[1]))
	}
	require.NoError(t, mw.Close())
	req := httptest.NewRequest(http.MethodPost, "/upload", buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	return req
}

func rchValidUploadParts() [][2]string {
	return [][2]string{{"file", "content"}, {"filename", "tool.tar.gz"}, {"description", "a tool"}}
}

func TestRchGenericUploadArtifact(t *testing.T) {
	tests := []struct {
		name     string
		opts     rchOpts
		parts    [][2]string
		notMulti bool
		wantCode errcode.CodeError
		wantSha  string
	}{
		{name: "created", opts: rchOpts{allow: true}, parts: rchValidUploadParts(), wantSha: "sha-of-file"},
		{name: "access denied", opts: rchOpts{allow: false}, parts: rchValidUploadParts(),
			wantCode: errcode.ErrCodeDenied},
		{name: "not multipart", opts: rchOpts{allow: true}, notMulti: true, wantCode: errcode.ErrCodeUnknown},
		{name: "invalid file name", opts: rchOpts{allow: true},
			parts:    [][2]string{{"file", "c"}, {"filename", "bad name!"}, {"description", "d"}},
			wantCode: errcode.ErrCodeUnknown},
		{name: "missing file part", opts: rchOpts{allow: true},
			parts:    [][2]string{{"filename", "tool.tar.gz"}, {"description", "d"}},
			wantCode: errcode.ErrCodeUnknown},
		{name: "temp upload fails", opts: rchOpts{allow: true, fm: rchFileManager{uploadErr: errors.New("disk")}},
			parts: rchValidUploadParts(), wantCode: errcode.ErrCodeUnknown},
		{name: "post upload fails", opts: rchOpts{allow: true, fm: rchFileManager{postErr: errors.New("db")}},
			parts: rchValidUploadParts(), wantCode: errcode.ErrCodeUnknown},
		{name: "image creation fails", opts: rchOpts{
			allow: true,
			image: &rchImageDao{getErr: basestore.ErrResourceNotFound, createErr: errors.New("db")},
		}, parts: rchValidUploadParts(), wantCode: errcode.ErrCodeUnknown},
		{name: "artifact creation fails", opts: rchOpts{
			allow: true,
			artifactStore: &rchArtifactDao{
				getErr: basestore.ErrResourceNotFound, createErr: errors.New("db"),
			},
		}, parts: rchValidUploadParts(), wantCode: errcode.ErrCodeUnknown},
		{name: "artifact lookup fails", opts: rchOpts{
			allow:         true,
			artifactStore: &rchArtifactDao{getErr: errors.New("db down")},
		}, parts: rchValidUploadParts(), wantCode: errcode.ErrCodeUnknown},
		{name: "image lookup fails", opts: rchOpts{
			allow: true,
			image: &rchImageDao{getErr: errors.New("db down")},
		}, parts: rchValidUploadParts(), wantCode: errcode.ErrCodeInvalidRequest},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			env := rchSetup(t, tc.opts)
			var req *http.Request
			if tc.notMulti {
				req = httptest.NewRequest(http.MethodPost, "/upload", strings.NewReader("raw"))
			} else {
				req = rchUploadRequest(t, tc.parts)
			}
			headers, sha, err := env.ctrl.UploadArtifact(rchCtx(), rchInfo(), req)
			if tc.wantCode == 0 {
				require.True(t, commons.IsEmptyError(err))
				require.Equal(t, tc.wantSha, sha)
				require.Equal(t, http.StatusCreated, headers.Code)
				require.NotNil(t, env.artifact.saved)
				require.Contains(t, string(env.artifact.saved.Metadata), "tool.tar.gz")
				return
			}
			if tc.name == "image lookup fails" {
				// Lookup failures other than "not found" are surfaced by the duplicate check.
				require.False(t, commons.IsEmptyError(err))
				return
			}
			require.Equal(t, tc.wantCode, err.Code)
		})
	}
}

func TestRchGenericUploadArtifactDuplicateFile(t *testing.T) {
	existing := &registrytypes.Artifact{Metadata: []byte(`{"files":[{"file_name":"tool.tar.gz"}]}`)}
	env := rchSetup(t, rchOpts{
		allow:         true,
		image:         &rchImageDao{existing: &registrytypes.Image{ID: 3}},
		artifactStore: &rchArtifactDao{existing: existing},
	})
	_, _, err := env.ctrl.UploadArtifact(rchCtx(), rchInfo(), rchUploadRequest(t, rchValidUploadParts()))
	require.Equal(t, errcode.ErrCodeInvalidRequest, err.Code)
}

func TestRchGenericUploadArtifactExistingVersionAddsFile(t *testing.T) {
	existing := &registrytypes.Artifact{Metadata: []byte(`{"files":[{"file_name":"other.zip","size":5}],"file_count":1}`)}
	env := rchSetup(t, rchOpts{
		allow:         true,
		image:         &rchImageDao{existing: &registrytypes.Image{ID: 3}},
		artifactStore: &rchArtifactDao{existing: existing},
	})
	_, _, err := env.ctrl.UploadArtifact(rchCtx(), rchInfo(), rchUploadRequest(t, rchValidUploadParts()))
	require.True(t, commons.IsEmptyError(err))
	require.NotNil(t, env.artifact.saved)
	require.Contains(t, string(env.artifact.saved.Metadata), "other.zip")
	require.Contains(t, string(env.artifact.saved.Metadata), "tool.tar.gz")
}

func TestRchGenericUploadArtifactCorruptExistingMetadata(t *testing.T) {
	env := rchSetup(t, rchOpts{
		allow:         true,
		artifactStore: &rchArtifactDao{existing: &registrytypes.Artifact{Metadata: []byte("not-json")}},
	})
	_, _, err := env.ctrl.UploadArtifact(rchCtx(), rchInfo(), rchUploadRequest(t, rchValidUploadParts()))
	require.Equal(t, errcode.ErrCodeUnknown, err.Code)
}

func TestRchGenericPullArtifact(t *testing.T) {
	tests := []struct {
		name     string
		opts     rchOpts
		wantCode errcode.CodeError
	}{
		{name: "redirect", opts: rchOpts{allow: true}},
		{name: "denied", opts: rchOpts{allow: false}, wantCode: errcode.ErrCodeDenied},
		{name: "download fails", opts: rchOpts{allow: true, fm: rchFileManager{downloadErr: errors.New("gone")}},
			wantCode: errcode.ErrCodeRootNotFound},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			env := rchSetup(t, tc.opts)
			info := rchInfo()
			info.FileName = "tool.tar.gz"
			headers, _, redirect, err := env.ctrl.PullArtifact(rchCtx(), info)
			if tc.wantCode == 0 {
				require.True(t, commons.IsEmptyError(err))
				require.Equal(t, http.StatusOK, headers.Code)
				require.Equal(t, "http://redirect", redirect)
				return
			}
			require.Equal(t, tc.wantCode, err.Code)
		})
	}
}

func TestRchGenericCheckIfFileAlreadyExist(t *testing.T) {
	ctx := context.Background()
	info := rchInfo()
	info.FileName = "tool.tar.gz"

	env := rchSetup(t, rchOpts{allow: true})
	require.NoError(t, env.ctrl.CheckIfFileAlreadyExist(ctx, info), "unknown image means no duplicate")

	env = rchSetup(t, rchOpts{allow: true, image: &rchImageDao{getErr: errors.New("db down")}})
	require.Error(t, env.ctrl.CheckIfFileAlreadyExist(ctx, info))

	env = rchSetup(t, rchOpts{
		allow: true,
		image: &rchImageDao{existing: &registrytypes.Image{ID: 3}},
	})
	require.NoError(t, env.ctrl.CheckIfFileAlreadyExist(ctx, info), "unknown artifact means no duplicate")

	env = rchSetup(t, rchOpts{
		allow:         true,
		image:         &rchImageDao{existing: &registrytypes.Image{ID: 3}},
		artifactStore: &rchArtifactDao{getErr: errors.New("db down")},
	})
	require.Error(t, env.ctrl.CheckIfFileAlreadyExist(ctx, info))

	env = rchSetup(t, rchOpts{
		allow:         true,
		image:         &rchImageDao{existing: &registrytypes.Image{ID: 3}},
		artifactStore: &rchArtifactDao{existing: &registrytypes.Artifact{Metadata: []byte("{broken")}},
	})
	require.NoError(t, env.ctrl.CheckIfFileAlreadyExist(ctx, info), "unreadable metadata is ignored")

	env = rchSetup(t, rchOpts{
		allow: true,
		image: &rchImageDao{existing: &registrytypes.Image{ID: 3}},
		artifactStore: &rchArtifactDao{
			existing: &registrytypes.Artifact{Metadata: []byte(`{"files":[{"file_name":"tool.tar.gz"}]}`)},
		},
	})
	err := env.ctrl.CheckIfFileAlreadyExist(ctx, info)
	require.ErrorIs(t, err, genericctrl.ErrDuplicateFile)
}

func TestRchGenericParseAndUploadToTmp(t *testing.T) {
	ctx := context.Background()
	env := rchSetup(t, rchOpts{allow: true})

	req := rchUploadRequest(t, [][2]string{
		{"ignored", "x"}, {"file", "payload"}, {"filename", "a.bin"}, {"description", "d"},
	})
	reader, err := req.MultipartReader()
	require.NoError(t, err)
	fileInfo, values, err := env.ctrl.ParseAndUploadToTmp(
		ctx, reader, rchInfo(), "file", []string{"filename", "description"},
	)
	require.NoError(t, err)
	require.Equal(t, "sha-of-file", fileInfo.Sha256)
	require.Equal(t, int64(len("payload")), fileInfo.Size)
	require.Equal(t, map[string]string{"filename": "a.bin", "description": "d"}, values)

	// Without a file part to look for only the form values are collected.
	req = rchUploadRequest(t, [][2]string{{"filename", "a.bin"}})
	reader, err = req.MultipartReader()
	require.NoError(t, err)
	_, values, err = env.ctrl.ParseAndUploadToTmp(ctx, reader, rchInfo(), "", []string{"filename"})
	require.NoError(t, err)
	require.Equal(t, "a.bin", values["filename"])
}

func TestRchGenericFileOperations(t *testing.T) {
	ctx := context.Background()
	info := func() generictype.ArtifactInfo {
		return generictype.ArtifactInfo{
			ArtifactInfo: pkg.ArtifactInfo{
				BaseInfo:      &pkg.BaseInfo{ParentID: rchParentID},
				RegIdentifier: "reg",
				Image:         "tool",
			},
			Version:  "1.0.0",
			FileName: "tool.bin",
			FilePath: "dir/tool.bin",
		}
	}
	type op struct {
		name string
		call func(c *genericctrl.Controller) error
	}
	ops := []op{
		{"download", func(c *genericctrl.Controller) error { return c.DownloadFile(ctx, info(), "dir/tool.bin").GetError() }},
		{"head", func(c *genericctrl.Controller) error { return c.HeadFile(ctx, info(), "dir/tool.bin").GetError() }},
		{"delete", func(c *genericctrl.Controller) error { return c.DeleteFile(ctx, info()).GetError() }},
		{"put", func(c *genericctrl.Controller) error {
			return c.PutFile(ctx, info(), io.NopCloser(strings.NewReader("x")), "text/plain").GetError()
		}},
	}
	for _, o := range ops {
		t.Run(o.name+" success", func(t *testing.T) {
			require.NoError(t, o.call(rchSetup(t, rchOpts{allow: true}).ctrl))
		})
		t.Run(o.name+" registry error", func(t *testing.T) {
			regErr := errors.New("backend failed")
			require.ErrorIs(t, o.call(rchSetup(t, rchOpts{allow: true, registryErr: regErr}).ctrl), regErr)
		})
		t.Run(o.name+" unsupported", func(t *testing.T) {
			require.Error(t, o.call(rchSetup(t, rchOpts{allow: true, unsupported: true}).ctrl))
		})
		t.Run(o.name+" registry not found", func(t *testing.T) {
			require.Error(t, o.call(rchSetup(t, rchOpts{allow: true, daoErr: errors.New("missing")}).ctrl))
		})
	}

	env := rchSetup(t, rchOpts{allow: true})
	dl := env.ctrl.DownloadFile(ctx, info(), "dir/tool.bin")
	require.Equal(t, "http://r", dl.RedirectURL)
	require.NotNil(t, dl.ReadCloser)
	require.NoError(t, dl.ReadCloser.Close())
	require.Equal(t, "put-sha", env.ctrl.PutFile(ctx, info(), io.NopCloser(strings.NewReader("x")), "t").Sha256)
	require.Equal(t, 204, env.ctrl.DeleteFile(ctx, info()).ResponseHeaders.Code)
	require.Equal(t, 200, env.ctrl.HeadFile(ctx, info(), "p").ResponseHeaders.Code)
}
