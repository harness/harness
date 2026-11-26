//  Copyright 2023 Harness, Inc.
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

package maven

import (
	"context"
	"database/sql"
	"encoding/gob"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"testing"

	"github.com/harness/gitness/app/api/request"
	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/auth/authz"
	"github.com/harness/gitness/app/services/refcache"
	"github.com/harness/gitness/app/store/cache"
	"github.com/harness/gitness/events"
	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	registryevents "github.com/harness/gitness/registry/app/events/artifact"
	"github.com/harness/gitness/registry/app/metadata"
	"github.com/harness/gitness/registry/app/pkg"
	"github.com/harness/gitness/registry/app/pkg/base"
	"github.com/harness/gitness/registry/app/pkg/commons"
	"github.com/harness/gitness/registry/app/pkg/filemanager"
	"github.com/harness/gitness/registry/app/pkg/quarantine"
	"github.com/harness/gitness/registry/app/storage"
	"github.com/harness/gitness/registry/app/store"
	"github.com/harness/gitness/registry/types"
	gitnessstore "github.com/harness/gitness/store"
	gtypes "github.com/harness/gitness/types"

	"github.com/stretchr/testify/require"
)

type rpfProducer struct {
	streams []string
}

func (p *rpfProducer) Send(_ context.Context, streamID string, _ map[string]any) (string, error) {
	p.streams = append(p.streams, streamID)
	return "1", nil
}

type rpfCollector struct{}

func (rpfCollector) RecordError(string, string)   {}
func (rpfCollector) RecordDiscard(string, string) {}

func rpfEventSystem(t *testing.T) (*events.System, *rpfProducer) {
	t.Helper()
	gob.Register(&registryevents.CommonArtifact{})
	producer := &rpfProducer{}
	system, err := events.NewSystem(
		func(string, string) (events.StreamConsumer, error) { return nil, errors.New("unused") },
		producer, rpfCollector{})
	require.NoError(t, err)
	return system, producer
}

func rpfArtifactReporter(t *testing.T) (*registryevents.Reporter, *rpfProducer) {
	t.Helper()
	system, producer := rpfEventSystem(t)
	reporter, err := registryevents.NewReporter(system)
	require.NoError(t, err)
	return reporter, producer
}

func rpfAuthedContext() context.Context {
	return request.WithAuthSession(context.Background(), &auth.Session{Principal: gtypes.Principal{ID: 9}})
}

type rpfImageRepo struct {
	store.ImageRepository
	image     *types.Image
	getErr    error
	upsertErr error
}

func (r *rpfImageRepo) GetByName(context.Context, int64, string, ...types.QueryOption) (*types.Image, error) {
	return r.image, r.getErr
}

func (r *rpfImageRepo) CreateOrUpdate(_ context.Context, image *types.Image) error {
	image.UUID = "img-uuid"
	return r.upsertErr
}

type rpfArtifactRepo struct {
	store.ArtifactRepository
	existing  *types.Artifact
	getErr    error
	upsertErr error
	saved     *types.Artifact
}

func (r *rpfArtifactRepo) GetByName(context.Context, int64, string, ...types.QueryOption) (*types.Artifact, error) {
	return r.existing, r.getErr
}

func (r *rpfArtifactRepo) CreateOrUpdate(_ context.Context, a *types.Artifact) (int64, error) {
	a.UUID = "art-uuid"
	r.saved = a
	return 1, r.upsertErr
}

type rpfRegistryRepo struct {
	store.RegistryRepository
	registries []types.Registry
	err        error
}

func (r *rpfRegistryRepo) GetByIDIn(context.Context, []int64, ...types.QueryOption) (*[]types.Registry, error) {
	if r.err != nil {
		return nil, r.err
	}
	return &r.registries, nil
}

type rpfProxyStore struct {
	store.UpstreamProxyConfigRepository
	err error
}

func (s *rpfProxyStore) GetByRegistryIdentifier(
	context.Context, int64, string, ...types.QueryOption,
) (*types.UpstreamProxy, error) {
	if s.err != nil {
		return nil, s.err
	}
	return &types.UpstreamProxy{RepoKey: "up"}, nil
}

type rpfFileManager struct {
	filemanager.FileManager
	metaErr     error
	downloadErr error
	uploadErr   error
	uploaded    string
	t           *testing.T
}

func (f *rpfFileManager) GetFileMetadata(context.Context, int64, string) (types.FileInfo, error) {
	return types.FileInfo{Size: 10, Filename: "demo.jar"}, f.metaErr
}

func (f *rpfFileManager) DownloadFileByPath(
	context.Context, string, int64, string, string, bool,
) (*storage.FileReader, int64, string, error) {
	if f.downloadErr != nil {
		return nil, 0, "", f.downloadErr
	}
	return nil, 10, "https://redirect.test/jar", nil
}

func (f *rpfFileManager) UploadFile(
	_ context.Context, path string, _ int64, _ int64, _ string, _ multipart.File, _ io.Reader, _ int64,
) (types.FileInfo, error) {
	f.uploaded = path
	return types.FileInfo{Size: 10}, f.uploadErr
}

type rpfLocalBase struct {
	base.LocalBase
	versionExists bool
	existsErr     error
	fileExists    bool
	fileErr       error
	audits        int
}

func (b *rpfLocalBase) CheckIfVersionExists(context.Context, pkg.PackageArtifactInfo) (bool, error) {
	return b.versionExists, b.existsErr
}

func (b *rpfLocalBase) ExistsByFilePath(context.Context, int64, string) (bool, error) {
	return b.fileExists, b.fileErr
}

func (b *rpfLocalBase) AuditPush(context.Context, pkg.ArtifactInfo, string, string, string) {
	b.audits++
}

type rpfTx struct{ err error }

func (t rpfTx) WithTx(ctx context.Context, fn func(ctx context.Context) error, _ ...any) error {
	if t.err != nil {
		return t.err
	}
	return fn(ctx)
}

func rpfMavenInfo(file string) pkg.MavenArtifactInfo {
	return pkg.MavenArtifactInfo{
		ArtifactInfo: &pkg.ArtifactInfo{
			BaseInfo:      &pkg.BaseInfo{ParentID: 7, RootIdentifier: "root", RootParentID: 2},
			RegIdentifier: "reg",
			RegistryID:    3,
			Image:         "org.acme:demo",
			Registry:      types.Registry{ID: 3, Name: "reg", Type: artifact.RegistryTypeVIRTUAL, ParentID: 7},
		},
		GroupID:    "org.acme",
		ArtifactID: "demo",
		Version:    "1.0",
		FileName:   file,
	}
}

type rpfLocalDeps struct {
	lb        *rpfLocalBase
	fm        *rpfFileManager
	images    *rpfImageRepo
	artifacts *rpfArtifactRepo
	tx        rpfTx
}

func rpfNewLocal(t *testing.T, d rpfLocalDeps) (*LocalRegistry, *rpfProducer) {
	t.Helper()
	reporter, producer := rpfArtifactReporter(t)
	if d.lb == nil {
		d.lb = &rpfLocalBase{}
	}
	if d.fm == nil {
		d.fm = &rpfFileManager{}
	}
	d.fm.t = t
	if d.images == nil {
		d.images = &rpfImageRepo{image: &types.Image{ID: 5}}
	}
	if d.artifacts == nil {
		d.artifacts = &rpfArtifactRepo{}
	}
	db := NewDBStore(nil, d.images, d.artifacts, nil, nil, nil, nil, nil)
	reg := NewLocalRegistry(d.lb, db, d.tx, d.fm, reporter)
	local, ok := reg.(*LocalRegistry)
	require.True(t, ok)
	return local, producer
}

func TestRpfLocalFetchArtifact(t *testing.T) {
	ctx := context.Background()

	t.Run("get returns headers and a redirect", func(t *testing.T) {
		l, _ := rpfNewLocal(t, rpfLocalDeps{})
		require.Equal(t, ArtifactTypeLocalRegistry, l.GetMavenArtifactType())
		headers, _, _, redirect, errs := l.GetArtifact(ctx, rpfMavenInfo("demo-1.0.jar"))
		require.Empty(t, errs)
		require.Equal(t, "application/java-archive", headers.Headers["Content-Type"])
		require.Equal(t, "https://redirect.test/jar", redirect)
	})

	t.Run("head does not download", func(t *testing.T) {
		l, _ := rpfNewLocal(t, rpfLocalDeps{fm: &rpfFileManager{downloadErr: errors.New("must not be called")}})
		headers, errs := l.HeadArtifact(ctx, rpfMavenInfo("demo-1.0.pom"))
		require.Empty(t, errs)
		require.Equal(t, "text/xml", headers.Headers["Content-Type"])
	})

	t.Run("version-less metadata file skips the artifact lookup", func(t *testing.T) {
		info := rpfMavenInfo("maven-metadata.xml")
		info.Version = ""
		l, _ := rpfNewLocal(t, rpfLocalDeps{artifacts: &rpfArtifactRepo{getErr: errors.New("must not be called")}})
		_, errs := l.HeadArtifact(ctx, info)
		require.Empty(t, errs)
	})

	t.Run("missing records map to not found", func(t *testing.T) {
		missingErrs := []error{sql.ErrNoRows, errors.New("resource not found"), errors.New("http status code: 404")}
		for _, missing := range missingErrs {
			l, _ := rpfNewLocal(t, rpfLocalDeps{images: &rpfImageRepo{getErr: missing}})
			_, errs := l.HeadArtifact(ctx, rpfMavenInfo("demo-1.0.jar"))
			require.Len(t, errs, 1)
			var userErr *commons.Error
			require.ErrorAs(t, errs[0], &userErr)
			require.Equal(t, http.StatusNotFound, userErr.Status)
		}

		l, _ := rpfNewLocal(t, rpfLocalDeps{artifacts: &rpfArtifactRepo{getErr: sql.ErrNoRows}})
		_, errs := l.HeadArtifact(ctx, rpfMavenInfo("demo-1.0.jar"))
		require.Len(t, errs, 1)

		l, _ = rpfNewLocal(t, rpfLocalDeps{fm: &rpfFileManager{metaErr: sql.ErrNoRows}})
		_, errs = l.HeadArtifact(ctx, rpfMavenInfo("demo-1.0.jar"))
		require.Len(t, errs, 1)
	})

	t.Run("other failures are passed through", func(t *testing.T) {
		boom := errors.New("boom")
		l, _ := rpfNewLocal(t, rpfLocalDeps{images: &rpfImageRepo{getErr: boom}})
		_, errs := l.HeadArtifact(ctx, rpfMavenInfo("demo-1.0.jar"))
		require.Equal(t, []error{boom}, errs)

		l, _ = rpfNewLocal(t, rpfLocalDeps{fm: &rpfFileManager{downloadErr: boom}})
		_, _, _, _, errs = l.GetArtifact(ctx, rpfMavenInfo("demo-1.0.jar"))
		require.Equal(t, []error{boom}, errs)
	})
}

func TestRpfLocalPutArtifact(t *testing.T) {
	ctx := rpfAuthedContext()
	body := func() io.Reader { return strings.NewReader("jar") }

	t.Run("new main artifact creates records, audits and publishes", func(t *testing.T) {
		lb := &rpfLocalBase{}
		fm := &rpfFileManager{}
		artifacts := &rpfArtifactRepo{getErr: errors.New("resource not found")}
		l, producer := rpfNewLocal(t, rpfLocalDeps{lb: lb, fm: fm, artifacts: artifacts})
		headers, errs := l.PutArtifact(ctx, rpfMavenInfo("demo-1.0.jar"), body())
		require.Empty(t, errs)
		require.Equal(t, http.StatusCreated, headers.Code)
		require.Equal(t, "/org/acme/demo/1.0/demo-1.0.jar", fm.uploaded)
		require.Equal(t, 1, lb.audits)
		require.Len(t, producer.streams, 1)

		var md metadata.MavenMetadata
		require.NoError(t, json.Unmarshal(artifacts.saved.Metadata, &md))
		require.Equal(t, int64(1), md.FileCount)
		require.Equal(t, int64(10), md.Size)
	})

	t.Run("additional files are added to existing metadata", func(t *testing.T) {
		existing, err := json.Marshal(metadata.MavenMetadata{
			Files: []metadata.File{{Filename: "demo-1.0.jar", Size: 10}}, FileCount: 1, Size: 10,
		})
		require.NoError(t, err)
		artifacts := &rpfArtifactRepo{existing: &types.Artifact{Metadata: existing}}
		l, producer := rpfNewLocal(t, rpfLocalDeps{artifacts: artifacts})
		_, errs := l.PutArtifact(ctx, rpfMavenInfo("demo-1.0.pom"), body())
		require.Empty(t, errs)
		require.Empty(t, producer.streams, "non-main files do not publish events")

		var md metadata.MavenMetadata
		require.NoError(t, json.Unmarshal(artifacts.saved.Metadata, &md))
		require.Equal(t, int64(2), md.FileCount)
		require.Equal(t, int64(20), md.Size)

		// Re-uploading a known file leaves the metadata untouched.
		_, errs = l.PutArtifact(ctx, rpfMavenInfo("demo-1.0.jar"), body())
		require.Empty(t, errs)
	})

	t.Run("metadata files are always overwritten", func(t *testing.T) {
		info := rpfMavenInfo("maven-metadata.xml")
		info.Version = ""
		lb := &rpfLocalBase{versionExists: true, fileExists: true}
		fm := &rpfFileManager{}
		l, _ := rpfNewLocal(t, rpfLocalDeps{lb: lb, fm: fm})
		headers, errs := l.PutArtifact(ctx, info, body())
		require.Empty(t, errs)
		require.Equal(t, http.StatusCreated, headers.Code)
		require.Equal(t, "/org/acme/demo/maven-metadata.xml", fm.uploaded)
	})

	t.Run("existing file is not uploaded again", func(t *testing.T) {
		fm := &rpfFileManager{}
		l, _ := rpfNewLocal(t, rpfLocalDeps{lb: &rpfLocalBase{versionExists: true, fileExists: true}, fm: fm})
		headers, errs := l.PutArtifact(ctx, rpfMavenInfo("demo-1.0.jar"), body())
		require.Empty(t, errs)
		require.Equal(t, http.StatusOK, headers.Code)
		require.Empty(t, fm.uploaded)
	})

	t.Run("failures", func(t *testing.T) {
		boom := errors.New("boom")
		put := func(d rpfLocalDeps) []error {
			l, _ := rpfNewLocal(t, d)
			_, errs := l.PutArtifact(ctx, rpfMavenInfo("demo-1.0.jar"), body())
			return errs
		}
		require.Len(t, put(rpfLocalDeps{lb: &rpfLocalBase{existsErr: boom}}), 1)
		require.Empty(t, put(rpfLocalDeps{lb: &rpfLocalBase{existsErr: gitnessstore.ErrResourceNotFound}}))
		require.Len(t, put(rpfLocalDeps{lb: &rpfLocalBase{fileErr: boom}}), 1)
		require.Len(t, put(rpfLocalDeps{fm: &rpfFileManager{uploadErr: boom}}), 1)
		require.Len(t, put(rpfLocalDeps{tx: rpfTx{err: boom}}), 1)
		require.Len(t, put(rpfLocalDeps{images: &rpfImageRepo{upsertErr: boom}}), 1)
		require.Len(t, put(rpfLocalDeps{artifacts: &rpfArtifactRepo{getErr: boom}}), 1)
		require.Len(t, put(rpfLocalDeps{artifacts: &rpfArtifactRepo{upsertErr: boom}}), 1)
		require.Len(t, put(rpfLocalDeps{artifacts: &rpfArtifactRepo{existing: &types.Artifact{Metadata: []byte("{")}}}), 1)
	})
}

type rpfFakeProxyController struct {
	useLocal  bool
	proxyErr  error
	proxyBody string
}

func (c *rpfFakeProxyController) UseLocalFile(
	context.Context, pkg.MavenArtifactInfo,
) (*commons.ResponseHeaders, *storage.FileReader, string, bool) {
	return &commons.ResponseHeaders{Code: http.StatusOK}, nil, "local-redirect", c.useLocal
}

func (c *rpfFakeProxyController) ProxyFile(
	context.Context, pkg.MavenArtifactInfo, types.UpstreamProxy, bool,
) (*commons.ResponseHeaders, io.ReadCloser, error) {
	if c.proxyErr != nil {
		return nil, nil, c.proxyErr
	}
	return &commons.ResponseHeaders{Code: http.StatusOK}, io.NopCloser(strings.NewReader(c.proxyBody)), nil
}

func TestRpfRemoteRegistry(t *testing.T) {
	ctx := context.Background()
	info := rpfMavenInfo("demo-1.0.jar")
	local, _ := rpfNewLocal(t, rpfLocalDeps{})
	newRemote := func(ctrl *rpfFakeProxyController, proxyErr error) *RemoteRegistry {
		db := NewDBStore(nil, nil, nil, nil, nil, nil, nil, &rpfProxyStore{err: proxyErr})
		remote, ok := NewRemoteRegistry(db, nil, local, ctrl).(*RemoteRegistry)
		require.True(t, ok)
		return remote
	}

	r := newRemote(&rpfFakeProxyController{useLocal: true}, nil)
	require.Equal(t, ArtifactTypeRemoteRegistry, r.GetMavenArtifactType())
	_, _, _, redirect, errs := r.GetArtifact(ctx, info)
	require.Empty(t, errs)
	require.Equal(t, "local-redirect", redirect)
	headers, errs := r.HeadArtifact(ctx, info)
	require.Empty(t, errs)
	require.Equal(t, http.StatusOK, headers.Code)

	r = newRemote(&rpfFakeProxyController{proxyBody: "from-upstream"}, nil)
	_, _, rc, _, errs := r.GetArtifact(ctx, info)
	require.Empty(t, errs)
	data, err := io.ReadAll(rc)
	require.NoError(t, err)
	require.Equal(t, "from-upstream", string(data))

	_, _, _, _, errs = newRemote(&rpfFakeProxyController{}, sql.ErrNoRows).GetArtifact(ctx, info)
	require.Len(t, errs, 1)

	boom := errors.New("upstream down")
	_, _, _, _, errs = newRemote(&rpfFakeProxyController{proxyErr: boom}, nil).GetArtifact(ctx, info)
	require.Equal(t, []error{boom}, errs)

	headers, errs = r.PutArtifact(ctx, info, strings.NewReader("x"))
	require.Nil(t, headers)
	require.Nil(t, errs)
}

type rpfSpaceCache struct{}

func (rpfSpaceCache) Stats() (int64, int64) { return 0, 0 }

func (rpfSpaceCache) Get(_ context.Context, key int64) (*gtypes.SpaceCore, error) {
	if key != 7 {
		return nil, errors.New("space not found")
	}
	return &gtypes.SpaceCore{ID: 7, Path: "acme/team"}, nil
}

func (rpfSpaceCache) Evict(context.Context, int64) {}

type rpfAuthorizer struct {
	authz.Authorizer
	allowed bool
}

func (a rpfAuthorizer) CheckAll(context.Context, *auth.Session, ...gtypes.PermissionCheck) (bool, error) {
	return a.allowed, nil
}

type rpfQuarantineFinder struct {
	quarantine.Finder
	err error
}

func (f rpfQuarantineFinder) CheckArtifactQuarantineStatus(
	context.Context, int64, string, string, *artifact.ArtifactType,
) error {
	return f.err
}

type rpfFirewall struct {
	err error
}

func (f rpfFirewall) CheckPolicyViolation(
	context.Context, int64, string, string, *artifact.ArtifactType,
) error {
	return f.err
}

func rpfNewController(
	t *testing.T, allowed bool, qErr error, fwErr error, upstreams []types.Registry, lookupErr error,
	local *LocalRegistry, remote *RemoteRegistry,
) *Controller {
	t.Helper()
	finder := refcache.NewSpaceFinder(rpfSpaceCache{}, nil, nil, cache.Evictor[*gtypes.SpaceCore]{})
	db := NewDBStore(&rpfRegistryRepo{registries: upstreams, err: lookupErr}, nil, nil, nil, nil, nil, nil, nil)
	t.Cleanup(func() { delete(TypeRegistry, LocalRegistryType); delete(TypeRegistry, RemoteRegistryType) })
	return NewController(local, remote, rpfAuthorizer{allowed: allowed}, db, finder,
		rpfQuarantineFinder{err: qErr}, rpfFirewall{err: fwErr})
}

func TestRpfControllerGetAndHead(t *testing.T) {
	ctx := rpfAuthedContext()
	info := rpfMavenInfo("demo-1.0.jar")

	local, _ := rpfNewLocal(t, rpfLocalDeps{})
	failingLocal, _ := rpfNewLocal(t, rpfLocalDeps{images: &rpfImageRepo{getErr: sql.ErrNoRows}})
	remote := &RemoteRegistry{
		proxyController: &rpfFakeProxyController{useLocal: true},
		DBStore:         NewDBStore(nil, nil, nil, nil, nil, nil, nil, &rpfProxyStore{}),
	}

	t.Run("access denied", func(t *testing.T) {
		c := rpfNewController(t, false, nil, nil, nil, nil, local, remote)
		resp := c.GetArtifact(ctx, info)
		require.NotEmpty(t, resp.Errors)
		require.NotEmpty(t, c.HeadArtifact(ctx, info).Errors)
	})

	t.Run("served from the local registry", func(t *testing.T) {
		c := rpfNewController(t, true, nil, nil, nil, nil, local, remote)
		resp := c.GetArtifact(ctx, info)
		require.Empty(t, resp.Errors)
		require.Equal(t, "https://redirect.test/jar", resp.RedirectURL)
		head := c.HeadArtifact(ctx, info)
		require.Empty(t, head.Errors)
		require.NotNil(t, head.ResponseHeaders)
	})

	t.Run("quarantined artifacts are rejected", func(t *testing.T) {
		c := rpfNewController(t, true, usererror.ErrQuarantinedArtifact, nil, nil, nil, local, remote)
		resp := c.GetArtifact(ctx, info)
		require.ErrorIs(t, resp.Errors[0], usererror.ErrQuarantinedArtifact)

		c = rpfNewController(t, true, errors.New("cache down"), nil, nil, nil, local, remote)
		resp = c.GetArtifact(ctx, info)
		require.ErrorContains(t, resp.Errors[0], "cache down")
	})

	t.Run("upstream proxies are checked against the dependency firewall", func(t *testing.T) {
		upstream := []types.Registry{{ID: 10, Name: "up", Type: artifact.RegistryTypeUPSTREAM, ParentID: 7}}
		withProxy := info
		withProxy.ArtifactInfo = &pkg.ArtifactInfo{
			BaseInfo: info.BaseInfo, RegIdentifier: "reg", RegistryID: 3, Image: "org.acme:demo",
			Registry: types.Registry{
				ID: 3, Name: "reg", Type: artifact.RegistryTypeVIRTUAL, ParentID: 7, UpstreamProxies: []int64{10},
			},
		}

		// The local registry has no record, so the request falls through to the upstream.
		c := rpfNewController(t, true, nil, usererror.ErrArtifactBlocked, upstream, nil, failingLocal, remote)
		resp := c.GetArtifact(ctx, withProxy)
		require.NotEmpty(t, resp.Errors)

		c = rpfNewController(t, true, nil, errors.New("firewall down"), upstream, nil, failingLocal, remote)
		resp = c.GetArtifact(ctx, withProxy)
		require.NotEmpty(t, resp.Errors)

		c = rpfNewController(t, true, nil, nil, upstream, nil, failingLocal, remote)
		resp = c.GetArtifact(ctx, withProxy)
		require.Empty(t, resp.Errors)
		require.Equal(t, "local-redirect", resp.RedirectURL)

		head := c.HeadArtifact(ctx, withProxy)
		require.Empty(t, head.Errors)
	})

	t.Run("nothing found anywhere", func(t *testing.T) {
		c := rpfNewController(t, true, nil, nil, nil, nil, failingLocal, remote)
		require.NotEmpty(t, c.GetArtifact(ctx, info).Errors)
		require.NotEmpty(t, c.HeadArtifact(ctx, info).Errors)
	})
}

func TestRpfControllerPutAndRouting(t *testing.T) {
	ctx := rpfAuthedContext()
	info := rpfMavenInfo("demo-1.0.jar")
	local, _ := rpfNewLocal(t, rpfLocalDeps{})
	remote := &RemoteRegistry{}

	c := rpfNewController(t, false, nil, nil, nil, nil, local, remote)
	resp := c.PutArtifact(ctx, info, strings.NewReader("x"))
	require.Equal(t, http.StatusForbidden, resp.ResponseHeaders.Code)
	require.NotEmpty(t, resp.Errors)

	c = rpfNewController(t, true, nil, nil, nil, nil, local, remote)
	resp = c.PutArtifact(ctx, info, strings.NewReader("x"))
	require.Empty(t, resp.Errors)
	require.Equal(t, http.StatusCreated, resp.ResponseHeaders.Code)

	require.Same(t, local, c.GetArtifactRegistry(ctx, types.Registry{Type: artifact.RegistryTypeVIRTUAL}))
	require.Same(t, remote, c.GetArtifactRegistry(ctx, types.Registry{Type: artifact.RegistryTypeUPSTREAM}))
	require.Nil(t, c.factory(ctx, ArtifactType(1<<9)))
}

func TestRpfControllerProxyWrapperAndOrdering(t *testing.T) {
	ctx := context.Background()
	local, _ := rpfNewLocal(t, rpfLocalDeps{})
	call := func(registry types.Registry, a Artifact) Response { return &HeadArtifactResponse{} }

	c := rpfNewController(t, true, nil, nil, nil, nil, local, &RemoteRegistry{})
	_, err := c.ProxyWrapper(ctx, call, pkg.MavenArtifactInfo{})
	require.ErrorContains(t, err, "artifactinfo is not found")

	t.Run("lookup failure", func(t *testing.T) {
		ctrl := rpfNewController(t, true, nil, nil, nil, errors.New("db down"), local, &RemoteRegistry{})
		info := rpfMavenInfo("demo-1.0.jar")
		info.Registry.UpstreamProxies = []int64{10}
		_, err := ctrl.ProxyWrapper(ctx, call, info)
		require.ErrorContains(t, err, "db down")
	})

	t.Run("upstream repositories follow the virtual one", func(t *testing.T) {
		ctrl := rpfNewController(t, true, nil, nil,
			[]types.Registry{{ID: 10, Name: "up"}}, nil, local, &RemoteRegistry{})
		repos, err := ctrl.GetOrderedRepos(ctx, "reg", types.Registry{ID: 3, UpstreamProxies: []int64{10}})
		require.NoError(t, err)
		require.Len(t, repos, 2)
		require.Equal(t, "up", repos[1].Name)

		repos, err = ctrl.GetOrderedRepos(ctx, "reg", types.Registry{ID: 3})
		require.NoError(t, err)
		require.Len(t, repos, 1)
	})

	t.Run("the first successful registry wins", func(t *testing.T) {
		ctrl := rpfNewController(t, true, nil, nil, nil, nil, local, &RemoteRegistry{})
		got, err := ctrl.ProxyWrapper(ctx, call, rpfMavenInfo("demo-1.0.jar"))
		require.NoError(t, err)
		require.Empty(t, got.GetErrors())
	})

	t.Run("the last error is reported", func(t *testing.T) {
		boom := errors.New("boom")
		failing := func(types.Registry, Artifact) Response { return &HeadArtifactResponse{Errors: []error{boom}} }
		ctrl := rpfNewController(t, true, nil, nil, nil, nil, local, &RemoteRegistry{})
		_, err := ctrl.ProxyWrapper(ctx, failing, rpfMavenInfo("demo-1.0.jar"))
		require.ErrorIs(t, err, boom)
	})
}
