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

package docker

import (
	"context"
	"encoding/gob"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/harness/gitness/app/api/request"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/services/refcache"
	corestore "github.com/harness/gitness/app/store"
	"github.com/harness/gitness/app/store/cache"
	urlprovider "github.com/harness/gitness/app/url"
	"github.com/harness/gitness/audit"
	"github.com/harness/gitness/events"
	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	registryevents "github.com/harness/gitness/registry/app/events/artifact"
	"github.com/harness/gitness/registry/app/manifest/manifestlist"
	"github.com/harness/gitness/registry/app/pkg"
	"github.com/harness/gitness/registry/app/pkg/commons"
	registrystorage "github.com/harness/gitness/registry/app/storage"
	"github.com/harness/gitness/registry/app/store"
	registrytypes "github.com/harness/gitness/registry/types"
	"github.com/harness/gitness/types"

	v2 "github.com/distribution/distribution/v3/registry/api/v2"
	"github.com/opencontainers/go-digest"
	"github.com/stretchr/testify/require"
)

var errRpgBoom = errors.New("boom")

// rpgTx runs the transaction function inline.
type rpgTx struct {
	err error
}

func (t rpgTx) WithTx(ctx context.Context, fn func(context.Context) error, _ ...any) error {
	if t.err != nil {
		return t.err
	}
	return fn(ctx)
}

// rpgManifestDao overrides the manifest repository methods used by the docker package.
type rpgManifestDao struct {
	store.ManifestRepository
	byDigest     func(repoID int64, image string, d registrytypes.Digest) (*registrytypes.Manifest, error)
	byTag        func(repoID int64, image, tag string) (*registrytypes.Manifest, error)
	bySubject    func(d registrytypes.Digest) (registrytypes.Manifests, error)
	get          func(id int64) (*registrytypes.Manifest, error)
	createOrFind func(m *registrytypes.Manifest) error
	references   func(m *registrytypes.Manifest) (registrytypes.Manifests, error)
	referencedBy func(m *registrytypes.Manifest) (registrytypes.Manifests, error)
	deleteMf     func(d digest.Digest) (bool, error)
	count        func() (int64, error)
}

func (m *rpgManifestDao) FindManifestByDigest(
	_ context.Context, repoID int64, image string, d registrytypes.Digest,
) (*registrytypes.Manifest, error) {
	return m.byDigest(repoID, image, d)
}

func (m *rpgManifestDao) FindManifestByTagName(
	_ context.Context, repoID int64, image, tag string,
) (*registrytypes.Manifest, error) {
	return m.byTag(repoID, image, tag)
}

func (m *rpgManifestDao) ListManifestsBySubjectDigest(
	_ context.Context, _ int64, d registrytypes.Digest,
) (registrytypes.Manifests, error) {
	return m.bySubject(d)
}

func (m *rpgManifestDao) Get(_ context.Context, id int64) (*registrytypes.Manifest, error) {
	return m.get(id)
}

func (m *rpgManifestDao) CreateOrFind(_ context.Context, mf *registrytypes.Manifest) error {
	return m.createOrFind(mf)
}

func (m *rpgManifestDao) References(_ context.Context, mf *registrytypes.Manifest) (registrytypes.Manifests, error) {
	return m.references(mf)
}

func (m *rpgManifestDao) ReferencedBy(_ context.Context, mf *registrytypes.Manifest) (registrytypes.Manifests, error) {
	return m.referencedBy(mf)
}

func (m *rpgManifestDao) DeleteManifest(_ context.Context, _ int64, _ string, d digest.Digest) (bool, error) {
	return m.deleteMf(d)
}

func (m *rpgManifestDao) CountByImageName(_ context.Context, _ int64, _ string) (int64, error) {
	return m.count()
}

type rpgBlobDao struct {
	store.BlobRepository
	byID         func(id int64) (*registrytypes.Blob, error)
	byRoot       func(d digest.Digest) (*registrytypes.Blob, error)
	byRepo       func(d digest.Digest) (*registrytypes.Blob, error)
	createOrFind func(b *registrytypes.Blob) (*registrytypes.Blob, bool, error)
	exists       func() (bool, error)
}

func (b *rpgBlobDao) FindByID(_ context.Context, id int64) (*registrytypes.Blob, error) {
	return b.byID(id)
}

func (b *rpgBlobDao) FindByDigestAndRootParentID(
	_ context.Context, d digest.Digest, _ int64,
) (*registrytypes.Blob, error) {
	return b.byRoot(d)
}

func (b *rpgBlobDao) FindByDigestAndRepoID(
	_ context.Context, d digest.Digest, _ int64, _ string,
) (*registrytypes.Blob, error) {
	return b.byRepo(d)
}

func (b *rpgBlobDao) CreateOrFind(_ context.Context, blob *registrytypes.Blob) (*registrytypes.Blob, bool, error) {
	return b.createOrFind(blob)
}

func (b *rpgBlobDao) ExistsBlob(_ context.Context, _ int64, _ digest.Digest, _ string) (bool, error) {
	return b.exists()
}

type rpgRegBlobDao struct {
	store.RegistryBlobRepository
	linkErr   error
	unlink    bool
	unlinkErr error
	linked    []int64
}

func (r *rpgRegBlobDao) LinkBlob(_ context.Context, _ string, _ *registrytypes.Registry, id int64) error {
	r.linked = append(r.linked, id)
	return r.linkErr
}

func (r *rpgRegBlobDao) UnlinkBlob(_ context.Context, _ string, _ *registrytypes.Registry, _ int64) (bool, error) {
	return r.unlink, r.unlinkErr
}

func (r *rpgRegBlobDao) UnlinkBlobByImageName(_ context.Context, _ int64, _ string) (bool, error) {
	return true, r.unlinkErr
}

type rpgTagDao struct {
	store.TagRepository
	tags       []*registrytypes.Tag
	tagsErr    error
	hasMore    bool
	hasMoreErr error
	created    []*registrytypes.Tag
	createErr  error
	createErrs []error
	deleted    bool
	deleteErr  error
	findTag    *registrytypes.Tag
	findErr    error
	manifestTg []string
	mfTagsErr  error
}

func (t *rpgTagDao) TagsPaginated(
	_ context.Context, _ int64, _ string, _ registrytypes.FilterParams,
) ([]*registrytypes.Tag, error) {
	return t.tags, t.tagsErr
}

func (t *rpgTagDao) HasTagsAfterName(_ context.Context, _ int64, _ registrytypes.FilterParams) (bool, error) {
	return t.hasMore, t.hasMoreErr
}

func (t *rpgTagDao) CreateOrUpdate(_ context.Context, tag *registrytypes.Tag) error {
	t.created = append(t.created, tag)
	if len(t.createErrs) > 0 {
		next := t.createErrs[0]
		t.createErrs = t.createErrs[1:]
		return next
	}
	return t.createErr
}

func (t *rpgTagDao) DeleteTagByName(_ context.Context, _ int64, _ string) (bool, error) {
	return t.deleted, t.deleteErr
}

func (t *rpgTagDao) DeleteTagByManifestID(_ context.Context, _ int64, _ int64) (bool, error) {
	return t.deleted, t.deleteErr
}

func (t *rpgTagDao) FindTag(_ context.Context, _ int64, _ string, _ string) (*registrytypes.Tag, error) {
	return t.findTag, t.findErr
}

func (t *rpgTagDao) GetTagsByManifestID(_ context.Context, _ int64) (*[]string, error) {
	return &t.manifestTg, t.mfTagsErr
}

type rpgArtifactDao struct {
	store.ArtifactRepository
	getErr    error
	upsertErr error
	deleteErr error
}

func (a *rpgArtifactDao) GetByRegistryImageAndVersion(
	_ context.Context, _ int64, _ string, _ string,
) (*registrytypes.Artifact, error) {
	return &registrytypes.Artifact{ID: 1}, a.getErr
}

func (a *rpgArtifactDao) CreateOrUpdate(_ context.Context, art *registrytypes.Artifact) (int64, error) {
	art.UUID = "artifact-uuid"
	return 1, a.upsertErr
}

func (a *rpgArtifactDao) DeleteByVersionAndImageName(_ context.Context, _ string, _ string, _ int64) error {
	return a.deleteErr
}

type rpgImageDao struct {
	store.ImageRepository
	getImage  *registrytypes.Image
	getErr    error
	upsertErr error
	deleteErr error
	upserted  []*registrytypes.Image
}

func (i *rpgImageDao) GetByName(_ context.Context, _ int64, _ string, _ ...registrytypes.QueryOption) (
	*registrytypes.Image, error,
) {
	return i.getImage, i.getErr
}

func (i *rpgImageDao) CreateOrUpdate(_ context.Context, img *registrytypes.Image) error {
	img.UUID = "image-uuid"
	i.upserted = append(i.upserted, img)
	return i.upsertErr
}

func (i *rpgImageDao) DeleteByImageNameAndRegID(_ context.Context, _ int64, _ string) error {
	return i.deleteErr
}

type rpgRegistryDao struct {
	store.RegistryRepository
	reg *registrytypes.Registry
	err error
	ids *[]registrytypes.Registry
}

func (r *rpgRegistryDao) GetByParentIDAndName(
	_ context.Context, _ int64, _ string, _ ...registrytypes.QueryOption,
) (*registrytypes.Registry, error) {
	return r.reg, r.err
}

func (r *rpgRegistryDao) GetByIDIn(
	_ context.Context, _ []int64, _ ...registrytypes.QueryOption,
) (*[]registrytypes.Registry, error) {
	return r.ids, r.err
}

type rpgLayerDao struct {
	store.LayerRepository
	err   error
	calls int
}

func (l *rpgLayerDao) AssociateLayerBlob(_ context.Context, _ *registrytypes.Manifest, _ *registrytypes.Blob) error {
	l.calls++
	return l.err
}

type rpgManifestRefDao struct {
	store.ManifestReferenceRepository
	err   error
	calls int
}

func (m *rpgManifestRefDao) AssociateManifest(_ context.Context, _, _ *registrytypes.Manifest) error {
	m.calls++
	return m.err
}

type rpgMappingDao struct {
	store.OCIImageIndexMappingRepository
	mappings  []*registrytypes.OCIImageIndexMapping
	getErr    error
	createErr error
	created   int
}

func (m *rpgMappingDao) GetAllByChildDigest(
	_ context.Context, _ int64, _ string, _ registrytypes.Digest,
) ([]*registrytypes.OCIImageIndexMapping, error) {
	return m.mappings, m.getErr
}

func (m *rpgMappingDao) Create(_ context.Context, _ *registrytypes.OCIImageIndexMapping) error {
	m.created++
	return m.createErr
}

type rpgMediaTypes struct {
	store.MediaTypesRepository
	exists bool
	err    error
}

func (m rpgMediaTypes) MediaTypeExists(_ context.Context, _ string) (bool, error) {
	return m.exists, m.err
}

// rpgGC fakes the online garbage collection service.
type rpgGC struct {
	blobTask    *registrytypes.GCBlobTask
	blobErr     error
	rescheduled int
	rescErr     error
	mfErr       error
	mfNErr      error
}

func (g *rpgGC) Start(
	context.Context, corestore.SpaceStore, store.BlobRepository, registrystorage.StorageResolver, *types.Config,
) {
}

func (g *rpgGC) BlobFindAndLockBefore(_ context.Context, _ int64, _ time.Time) (*registrytypes.GCBlobTask, error) {
	return g.blobTask, g.blobErr
}

func (g *rpgGC) BlobReschedule(_ context.Context, _ *registrytypes.GCBlobTask, _ time.Duration) error {
	g.rescheduled++
	return g.rescErr
}

func (g *rpgGC) ManifestFindAndLockBefore(
	_ context.Context, _, _ int64, _ time.Time,
) (*registrytypes.GCManifestTask, error) {
	return nil, g.mfErr
}

func (g *rpgGC) ManifestFindAndLockNBefore(
	_ context.Context, _ int64, _ []int64, _ time.Time,
) ([]*registrytypes.GCManifestTask, error) {
	return nil, g.mfNErr
}

type rpgSpaceCache struct {
	err error
}

func (rpgSpaceCache) Stats() (int64, int64) { return 0, 0 }

func (c rpgSpaceCache) Get(_ context.Context, key int64) (*types.SpaceCore, error) {
	if c.err != nil {
		return nil, c.err
	}
	return &types.SpaceCore{ID: key, Identifier: "acme", Path: "acme"}, nil
}

func (rpgSpaceCache) Evict(context.Context, int64) {}

func rpgSpaceFinder(err error) refcache.SpaceFinder {
	return refcache.NewSpaceFinder(rpgSpaceCache{err: err}, nil, nil, cache.Evictor[*types.SpaceCore]{})
}

type rpgAudit struct {
	audit.Service
	logs int
}

func (a *rpgAudit) Log(
	context.Context, types.Principal, audit.Resource, audit.Action, string, ...audit.Option,
) error {
	a.logs++
	return nil
}

type rpgCollector struct{}

func (rpgCollector) RecordError(string, string)   {}
func (rpgCollector) RecordDiscard(string, string) {}

type rpgProducer struct {
	mu   sync.Mutex
	sent []map[string]any
}

func (p *rpgProducer) Send(_ context.Context, _ string, payload map[string]any) (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.sent = append(p.sent, payload)
	return "1", nil
}

func (p *rpgProducer) count() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.sent)
}

var rpgGobOnce sync.Once

func rpgArtifactReporter(t *testing.T) (registryevents.Reporter, *rpgProducer) {
	t.Helper()
	rpgGobOnce.Do(func() {
		gob.Register(&registryevents.DockerArtifact{})
		gob.Register(&registryevents.HelmArtifact{})
	})
	producer := &rpgProducer{}
	system, err := events.NewSystem(
		func(string, string) (events.StreamConsumer, error) { return nil, errRpgBoom },
		producer, rpgCollector{},
		false,
	)
	require.NoError(t, err)
	reporter, err := registryevents.NewReporter(system)
	require.NoError(t, err)
	return *reporter, producer
}

type rpgEventReporter struct {
	mu    sync.Mutex
	calls int
	done  chan struct{}
}

func (r *rpgEventReporter) ReportEvent(_ context.Context, _ any, _ string) {
	r.mu.Lock()
	r.calls++
	r.mu.Unlock()
	if r.done != nil {
		close(r.done)
	}
}

type rpgURLProvider struct {
	urlprovider.Provider
}

func (rpgURLProvider) RegistryURL(_ context.Context, params ...string) string {
	out := "https://registry.example.com"
	for _, p := range params {
		out += "/" + p
	}
	return out
}

func rpgAuthCtx(ctx context.Context) context.Context {
	return request.WithAuthSession(ctx, &auth.Session{Principal: types.Principal{ID: 42}})
}

func rpgRegistry() registrytypes.Registry {
	return registrytypes.Registry{
		ID: 5, Name: "reg", ParentID: 1, RootParentID: 1,
		Type: artifact.RegistryTypeVIRTUAL, PackageType: artifact.PackageTypeDOCKER,
	}
}

func rpgInfo(t *testing.T) pkg.RegistryInfo {
	t.Helper()
	builder, err := v2.NewURLBuilderFromString("http://localhost:3000/", false)
	require.NoError(t, err)
	reg := rpgRegistry()
	return pkg.RegistryInfo{
		ArtifactInfo: &pkg.ArtifactInfo{
			BaseInfo: &pkg.BaseInfo{
				PathRoot: "root", RootIdentifier: "root", ParentID: 1, RootParentID: 1,
				PathPackageType: artifact.PackageTypeDOCKER,
			},
			RegIdentifier: "reg", RegistryID: reg.ID, Registry: reg, Image: "img",
		},
		URLBuilder:  builder,
		PackageType: artifact.PackageTypeDOCKER,
	}
}

func rpgDigest(seed string) digest.Digest {
	return digest.FromString(seed)
}

func rpgStoredDigest(t *testing.T, d digest.Digest) registrytypes.Digest {
	t.Helper()
	out, err := registrytypes.NewDigest(d)
	require.NoError(t, err)
	return out
}

func rpgDockerManifest(t *testing.T, seed string) (string, []byte) {
	t.Helper()
	body := map[string]any{
		"schemaVersion": 2,
		"mediaType":     "application/vnd.docker.distribution.manifest.v2+json",
		"config": map[string]any{
			"mediaType": "application/vnd.docker.container.image.v1+json",
			"size":      10, "digest": rpgDigest(seed + "-config").String(),
		},
		"layers": []any{map[string]any{
			"mediaType": "application/vnd.docker.image.rootfs.diff.tar.gzip",
			"size":      20, "digest": rpgDigest(seed + "-layer").String(),
		}},
	}
	payload, err := json.Marshal(body)
	require.NoError(t, err)
	return "application/vnd.docker.distribution.manifest.v2+json", payload
}

func rpgOCIManifest(t *testing.T, seed string, subject *digest.Digest) (string, []byte) {
	t.Helper()
	body := map[string]any{
		"schemaVersion": 2,
		"mediaType":     "application/vnd.oci.image.manifest.v1+json",
		"artifactType":  "application/vnd.example.sbom",
		"config": map[string]any{
			"mediaType": "application/vnd.oci.image.config.v1+json",
			"size":      10, "digest": rpgDigest(seed + "-config").String(),
		},
		"layers": []any{map[string]any{
			"mediaType": "application/vnd.oci.image.layer.v1.tar+gzip",
			"size":      20, "digest": rpgDigest(seed + "-layer").String(),
		}},
	}
	if subject != nil {
		body["subject"] = map[string]any{
			"mediaType": "application/vnd.oci.image.manifest.v1+json",
			"size":      5, "digest": subject.String(),
		}
	}
	payload, err := json.Marshal(body)
	require.NoError(t, err)
	return "application/vnd.oci.image.manifest.v1+json", payload
}

func rpgIndex(t *testing.T, mediaType string, children ...digest.Digest) (string, []byte) {
	t.Helper()
	descs := make([]any, 0, len(children))
	for _, c := range children {
		descs = append(descs, map[string]any{
			"mediaType": "application/vnd.docker.distribution.manifest.v2+json",
			"size":      5, "digest": c.String(),
			"platform": map[string]any{"architecture": "amd64", "os": "linux"},
		})
	}
	body := map[string]any{"schemaVersion": 2, "mediaType": mediaType, "manifests": descs}
	payload, err := json.Marshal(body)
	require.NoError(t, err)
	return mediaType, payload
}

func rpgHeaders() *commons.ResponseHeaders {
	return &commons.ResponseHeaders{Headers: map[string]string{}}
}

func rpgManifestList(t *testing.T, payload []byte) *manifestlist.DeserializedManifestList {
	t.Helper()
	list := &manifestlist.DeserializedManifestList{}
	require.NoError(t, list.UnmarshalJSON(payload))
	return list
}

// rpgFakeMS overrides only the manifest service methods a test needs.
type rpgFakeMS struct {
	ManifestService
	deleteErr error
}

func (m *rpgFakeMS) DeleteManifest(context.Context, string, digest.Digest, pkg.RegistryInfo) error {
	return m.deleteErr
}
