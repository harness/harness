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

package deletion

import (
	"context"
	"encoding/gob"
	"errors"
	"sync"
	"testing"

	"github.com/harness/gitness/app/url"
	"github.com/harness/gitness/events"
	"github.com/harness/gitness/registry/app/api/interfaces"
	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	registryevents "github.com/harness/gitness/registry/app/events/artifact"
	"github.com/harness/gitness/registry/app/manifest/manifestlist"
	"github.com/harness/gitness/registry/app/pkg/filemanager"
	"github.com/harness/gitness/registry/app/services/reindexing"
	"github.com/harness/gitness/registry/app/store"
	registrytypes "github.com/harness/gitness/registry/types"

	"github.com/opencontainers/go-digest"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/stretchr/testify/require"
)

var errRb5Boom = errors.New("boom")

const (
	rb5Digest = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	rb5Image  = "img"
)

// rb5Rec records calls and injects errors by method name.
type rb5Rec struct {
	mu    sync.Mutex
	calls []string
	errs  map[string]error
}

func (r *rb5Rec) hit(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls = append(r.calls, name)
	return r.errs[name]
}

func (r *rb5Rec) called(name string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, c := range r.calls {
		if c == name {
			return true
		}
	}
	return false
}

type rb5Manifests struct {
	store.ManifestRepository
	rec        *rb5Rec
	manifest   *registrytypes.Manifest
	referenced registrytypes.Manifests
	count      int64
}

func (m *rb5Manifests) FindManifestByDigest(
	context.Context, int64, string, registrytypes.Digest,
) (*registrytypes.Manifest, error) {
	if err := m.rec.hit("FindManifestByDigest"); err != nil {
		return nil, err
	}
	return m.manifest, nil
}

func (m *rb5Manifests) ReferencedBy(context.Context, *registrytypes.Manifest) (registrytypes.Manifests, error) {
	return m.referenced, m.rec.hit("ReferencedBy")
}

func (m *rb5Manifests) Delete(context.Context, int64, int64) error {
	return m.rec.hit("ManifestDelete")
}

func (m *rb5Manifests) CountByImageName(context.Context, int64, string) (int64, error) {
	return m.count, m.rec.hit("CountByImageName")
}

func (m *rb5Manifests) Get(context.Context, int64) (*registrytypes.Manifest, error) {
	if err := m.rec.hit("ManifestGet"); err != nil {
		return nil, err
	}
	return m.manifest, nil
}

func (m *rb5Manifests) DeleteManifestByImageName(context.Context, int64, string) (bool, error) {
	return true, m.rec.hit("DeleteManifestByImageName")
}

type rb5Tags struct {
	store.TagRepository
	rec *rb5Rec
	tag *registrytypes.Tag
}

func (t *rb5Tags) DeleteTagByManifestID(context.Context, int64, int64) (bool, error) {
	return true, t.rec.hit("DeleteTagByManifestID")
}

func (t *rb5Tags) DeleteTag(context.Context, int64, string, string) error {
	return t.rec.hit("DeleteTag")
}

func (t *rb5Tags) FindTag(context.Context, int64, string, string) (*registrytypes.Tag, error) {
	if err := t.rec.hit("FindTag"); err != nil {
		return nil, err
	}
	return t.tag, nil
}

type rb5Images struct {
	store.ImageRepository
	rec *rb5Rec
}

func (i *rb5Images) GetByName(
	context.Context, int64, string, ...registrytypes.QueryOption,
) (*registrytypes.Image, error) {
	if err := i.rec.hit("ImageGetByName"); err != nil {
		return nil, err
	}
	return &registrytypes.Image{Name: rb5Image}, nil
}

func (i *rb5Images) DeleteByImageNameAndRegID(context.Context, int64, string) error {
	return i.rec.hit("DeleteByImageNameAndRegID")
}

func (i *rb5Images) DeleteByImageNameIfNoLinkedArtifacts(context.Context, int64, string) error {
	return i.rec.hit("DeleteByImageNameIfNoLinkedArtifacts")
}

type rb5Artifacts struct {
	store.ArtifactRepository
	rec *rb5Rec
}

func (a *rb5Artifacts) DeleteByVersionAndImageName(context.Context, string, string, int64) error {
	return a.rec.hit("DeleteByVersionAndImageName")
}

func (a *rb5Artifacts) DeleteByImageNameAndRegistryID(context.Context, int64, string) error {
	return a.rec.hit("DeleteByImageNameAndRegistryID")
}

type rb5Blobs struct {
	store.RegistryBlobRepository
	rec *rb5Rec
}

func (b *rb5Blobs) UnlinkBlobByImageName(context.Context, int64, string) (bool, error) {
	return true, b.rec.hit("UnlinkBlobByImageName")
}

type rb5Files struct {
	filemanager.FileManager
	rec *rb5Rec
}

func (f *rb5Files) DeleteFile(context.Context, int64, string) error { return f.rec.hit("DeleteFile") }

type rb5Tx struct{}

func (rb5Tx) WithTx(ctx context.Context, fn func(context.Context) error, _ ...any) error {
	return fn(ctx)
}

type rb5Wrapper struct {
	interfaces.PackageWrapper
	rec *rb5Rec
}

func (w *rb5Wrapper) DeleteArtifactVersion(
	context.Context, *registrytypes.RegistryRequestBaseInfo, *registrytypes.Image, string, string,
) error {
	return w.rec.hit("WrapperDeleteVersion")
}

func (w *rb5Wrapper) DeleteArtifact(context.Context, *registrytypes.RegistryRequestBaseInfo, string) error {
	return w.rec.hit("WrapperDeleteArtifact")
}

func (w *rb5Wrapper) TriggerIndexEvents(context.Context, int64, string, string) error {
	return w.rec.hit("WrapperTriggerIndex")
}

type rb5Producer struct {
	mu   sync.Mutex
	sent []string
}

func (p *rb5Producer) Send(_ context.Context, streamID string, _ map[string]any) (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.sent = append(p.sent, streamID)
	return "1", nil
}

type rb5Collector struct{}

func (rb5Collector) RecordError(string, string)   {}
func (rb5Collector) RecordDiscard(string, string) {}

var rb5GobOnce sync.Once

func rb5Reporter(t *testing.T) (*registryevents.Reporter, *rb5Producer) {
	t.Helper()
	rb5GobOnce.Do(func() {
		gob.Register(&registryevents.DockerArtifact{})
		gob.Register(&registryevents.HelmArtifact{})
		gob.Register(&registryevents.CommonArtifact{})
	})
	producer := &rb5Producer{}
	system, err := events.NewSystem(
		func(string, string) (events.StreamConsumer, error) { return nil, errRb5Boom },
		producer, rb5Collector{},
		false,
	)
	require.NoError(t, err)
	reporter, err := registryevents.NewReporter(system)
	require.NoError(t, err)
	return reporter, producer
}

type rb5Env struct {
	svc       *Service
	rec       *rb5Rec
	manifests *rb5Manifests
	tags      *rb5Tags
	producer  *rb5Producer
	untagged  bool
}

func rb5NewEnv(t *testing.T, untagged bool, errs map[string]error) *rb5Env {
	t.Helper()
	rec := &rb5Rec{errs: errs}
	reporter, producer := rb5Reporter(t)
	wrapper := &rb5Wrapper{rec: rec}
	manifests := &rb5Manifests{rec: rec, manifest: &registrytypes.Manifest{
		ID: 7, MediaType: v1.MediaTypeImageManifest, Digest: rb5Digest,
	}}
	tags := &rb5Tags{rec: rec, tag: &registrytypes.Tag{ManifestID: 7}}
	env := &rb5Env{rec: rec, manifests: manifests, tags: tags, producer: producer, untagged: untagged}
	env.svc = NewService(
		&rb5Artifacts{rec: rec}, &rb5Images{rec: rec}, manifests, tags, &rb5Blobs{rec: rec},
		&rb5Files{rec: rec}, rb5Tx{}, func(context.Context) bool { return untagged }, wrapper,
		reindexing.NewService(nil, *reporter, wrapper), reporter, nil,
	)
	return env
}

func TestRb5DeleteImageByPackageType(t *testing.T) {
	tests := []struct {
		name    string
		pt      artifact.PackageType
		errs    map[string]error
		wantErr string
		want    string
	}{
		{name: "docker", pt: artifact.PackageTypeDOCKER, want: "DeleteManifestByImageName"},
		{name: "helm", pt: artifact.PackageTypeHELM, want: "DeleteByImageNameAndRegID"},
		{name: "npm", pt: artifact.PackageTypeNPM, want: "DeleteFile"},
		{name: "generic", pt: artifact.PackageTypeGENERIC, want: "DeleteByImageNameAndRegistryID"},
		{name: "rpm", pt: artifact.PackageTypeRPM, wantErr: "not supported for rpm"},
		{name: "huggingface", pt: artifact.PackageTypeHUGGINGFACE, wantErr: "unsupported package type"},
		{name: "cargo wrapper", pt: artifact.PackageTypeCARGO, want: "WrapperDeleteArtifact"},
		{
			name: "cargo wrapper error", pt: artifact.PackageTypeCARGO,
			errs: map[string]error{"WrapperDeleteArtifact": errRb5Boom}, wantErr: "boom",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := rb5NewEnv(t, false, tt.errs)
			err := env.svc.DeleteImageByPackageType(
				context.Background(), &registrytypes.RegistryRequestBaseInfo{RegistryID: 1}, tt.pt, rb5Image)
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			require.True(t, env.rec.called(tt.want))
		})
	}
}

func TestRb5DeleteOCIImageErrors(t *testing.T) {
	for _, step := range []string{
		"DeleteManifestByImageName", "UnlinkBlobByImageName", "DeleteByImageNameAndRegistryID",
		"DeleteByImageNameAndRegID",
	} {
		t.Run(step, func(t *testing.T) {
			env := rb5NewEnv(t, false, map[string]error{step: errRb5Boom})
			err := env.svc.DeleteOCIImage(context.Background(), 1, rb5Image)
			require.ErrorIs(t, err, errRb5Boom)
		})
	}
	env := rb5NewEnv(t, false, nil)
	require.NoError(t, env.svc.DeleteOCIImage(context.Background(), 1, rb5Image))
}

func TestRb5DeleteGenericImage(t *testing.T) {
	ctx := context.Background()
	for _, step := range []string{"DeleteFile", "DeleteByImageNameAndRegistryID", "DeleteByImageNameAndRegID"} {
		t.Run(step, func(t *testing.T) {
			env := rb5NewEnv(t, false, map[string]error{step: errRb5Boom})
			err := env.svc.DeleteGenericImage(ctx, 1, artifact.PackageTypeNPM, rb5Image)
			require.ErrorIs(t, err, errRb5Boom)
		})
	}
	env := rb5NewEnv(t, false, nil)
	require.NoError(t, env.svc.DeleteGenericImage(ctx, 1, artifact.PackageTypeNPM, rb5Image))
	// docker has no file path, so the call fails before touching storage.
	require.ErrorContains(t, env.svc.DeleteGenericImage(ctx, 1, artifact.PackageTypeDOCKER, rb5Image),
		"failed to get file path")
}

func TestRb5DeleteGenericArtifact(t *testing.T) {
	ctx := context.Background()
	for _, step := range []string{"DeleteFile", "DeleteByVersionAndImageName", "DeleteByImageNameIfNoLinkedArtifacts"} {
		t.Run(step, func(t *testing.T) {
			env := rb5NewEnv(t, false, map[string]error{step: errRb5Boom})
			err := env.svc.DeleteGenericArtifact(ctx, 1, artifact.PackageTypeMAVEN, "g:a", "1.0")
			require.ErrorIs(t, err, errRb5Boom)
		})
	}
	env := rb5NewEnv(t, false, nil)
	require.NoError(t, env.svc.DeleteGenericArtifact(ctx, 1, artifact.PackageTypeMAVEN, "g:a", "1.0"))
	require.ErrorContains(t, env.svc.DeleteGenericArtifact(ctx, 1, artifact.PackageTypeHELM, "a", "1"),
		"failed to get file path")
}

func TestRb5DeleteOCIArtifactVersionTagged(t *testing.T) {
	ctx := context.Background()
	regInfo := &registrytypes.RegistryRequestBaseInfo{RegistryID: 1, PackageType: artifact.PackageTypeDOCKER}
	pid := int64(5)

	t.Run("deletes tag and fires webhook", func(t *testing.T) {
		env := rb5NewEnv(t, false, nil)
		env.svc.urlProvider = rb5URL{}
		require.NoError(t, env.svc.DeleteOCIArtifactVersion(ctx, regInfo, rb5Image, "v1", &pid, "reg"))
		require.True(t, env.rec.called("DeleteTag"))
		require.Len(t, env.producer.sent, 1)
	})
	t.Run("no principal means no webhook", func(t *testing.T) {
		env := rb5NewEnv(t, false, nil)
		require.NoError(t, env.svc.DeleteOCIArtifactVersion(ctx, regInfo, rb5Image, "v1", nil, "reg"))
		require.Empty(t, env.producer.sent)
	})
	t.Run("tag not found still deletes without webhook", func(t *testing.T) {
		env := rb5NewEnv(t, false, map[string]error{"FindTag": errRb5Boom})
		env.svc.urlProvider = rb5URL{}
		require.NoError(t, env.svc.DeleteOCIArtifactVersion(ctx, regInfo, rb5Image, "v1", &pid, "reg"))
		require.Empty(t, env.producer.sent)
	})
	t.Run("manifest not found still deletes without webhook", func(t *testing.T) {
		env := rb5NewEnv(t, false, map[string]error{"ManifestGet": errRb5Boom})
		require.NoError(t, env.svc.DeleteOCIArtifactVersion(ctx, regInfo, rb5Image, "v1", &pid, "reg"))
		require.Empty(t, env.producer.sent)
	})
	t.Run("delete tag error", func(t *testing.T) {
		env := rb5NewEnv(t, false, map[string]error{"DeleteTag": errRb5Boom})
		require.ErrorIs(t, env.svc.DeleteOCIArtifactVersion(ctx, regInfo, rb5Image, "v1", &pid, "reg"), errRb5Boom)
	})
}

func TestRb5DeleteOCIArtifactVersionUntagged(t *testing.T) {
	ctx := context.Background()
	regInfo := &registrytypes.RegistryRequestBaseInfo{RegistryID: 1, PackageType: artifact.PackageTypeDOCKER}
	pid := int64(5)

	t.Run("success removes image when last manifest", func(t *testing.T) {
		env := rb5NewEnv(t, true, nil)
		env.svc.urlProvider = rb5URL{}
		require.NoError(t, env.svc.DeleteOCIArtifactVersion(ctx, regInfo, rb5Image, rb5Digest, &pid, "reg"))
		require.True(t, env.rec.called("DeleteByImageNameAndRegID"))
		require.True(t, env.rec.called("UnlinkBlobByImageName"))
		require.Len(t, env.producer.sent, 1)
	})
	t.Run("keeps image when manifests remain", func(t *testing.T) {
		env := rb5NewEnv(t, true, nil)
		env.manifests.count = 2
		require.NoError(t, env.svc.DeleteOCIArtifactVersion(ctx, regInfo, rb5Image, rb5Digest, nil, "reg"))
		require.False(t, env.rec.called("DeleteByImageNameAndRegID"))
	})
	t.Run("manifest list can be deleted without parent check", func(t *testing.T) {
		env := rb5NewEnv(t, true, nil)
		env.manifests.manifest.MediaType = manifestlist.MediaTypeManifestList
		require.NoError(t, env.svc.DeleteOCIArtifactVersion(ctx, regInfo, rb5Image, rb5Digest, nil, "reg"))
		require.False(t, env.rec.called("ReferencedBy"))
	})
	t.Run("referenced manifest is rejected", func(t *testing.T) {
		env := rb5NewEnv(t, true, nil)
		env.manifests.referenced = registrytypes.Manifests{{Digest: digest.Digest(rb5Digest)}}
		err := env.svc.DeleteOCIArtifactVersion(ctx, regInfo, rb5Image, rb5Digest, nil, "reg")
		require.ErrorContains(t, err, "referenced by")
	})
	for _, step := range []string{
		"FindManifestByDigest", "ReferencedBy", "ManifestDelete", "DeleteTagByManifestID",
		"DeleteByVersionAndImageName", "CountByImageName", "DeleteByImageNameAndRegID", "UnlinkBlobByImageName",
	} {
		t.Run("error at "+step, func(t *testing.T) {
			env := rb5NewEnv(t, true, map[string]error{step: errRb5Boom})
			err := env.svc.DeleteOCIArtifactVersion(ctx, regInfo, rb5Image, rb5Digest, nil, "reg")
			require.ErrorIs(t, err, errRb5Boom)
		})
	}
}

func TestRb5DeleteArtifactVersionByPackageType(t *testing.T) {
	ctx := context.Background()
	pid := int64(9)
	tests := []struct {
		name    string
		pt      artifact.PackageType
		errs    map[string]error
		wantErr bool
		want    string
	}{
		{name: "docker", pt: artifact.PackageTypeDOCKER, want: "DeleteTag"},
		{
			name: "docker error", pt: artifact.PackageTypeDOCKER,
			errs: map[string]error{"DeleteTag": errRb5Boom}, wantErr: true,
		},
		{name: "npm", pt: artifact.PackageTypeNPM, want: "DeleteFile"},
		{name: "npm error", pt: artifact.PackageTypeNPM, errs: map[string]error{"DeleteFile": errRb5Boom}, wantErr: true},
		{name: "cargo wrapper", pt: artifact.PackageTypeCARGO, want: "WrapperTriggerIndex"},
		{
			name: "cargo image lookup error", pt: artifact.PackageTypeCARGO,
			errs: map[string]error{"ImageGetByName": errRb5Boom}, wantErr: true,
		},
		{
			name: "cargo wrapper delete error", pt: artifact.PackageTypeCARGO,
			errs: map[string]error{"WrapperDeleteVersion": errRb5Boom}, wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := rb5NewEnv(t, false, tt.errs)
			env.svc.urlProvider = rb5URL{}
			regInfo := &registrytypes.RegistryRequestBaseInfo{RegistryID: 1, PackageType: tt.pt}
			err := env.svc.DeleteArtifactVersionByPackageType(ctx, regInfo, rb5Image, "v1", &pid, "reg")
			if tt.wantErr {
				require.ErrorIs(t, err, errRb5Boom)
				return
			}
			require.NoError(t, err)
			require.True(t, env.rec.called(tt.want))
		})
	}
	t.Run("nil principal", func(t *testing.T) {
		env := rb5NewEnv(t, false, nil)
		regInfo := &registrytypes.RegistryRequestBaseInfo{RegistryID: 1, PackageType: artifact.PackageTypeNPM}
		require.NoError(t, env.svc.DeleteArtifactVersionByPackageType(ctx, regInfo, rb5Image, "1", nil, ""))
		require.Empty(t, env.producer.sent)
	})
	t.Run("npm with principal reports event", func(t *testing.T) {
		env := rb5NewEnv(t, false, nil)
		regInfo := &registrytypes.RegistryRequestBaseInfo{RegistryID: 1, PackageType: artifact.PackageTypeNPM}
		require.NoError(t, env.svc.DeleteArtifactVersionByPackageType(ctx, regInfo, rb5Image, "1", &pid, ""))
		require.Len(t, env.producer.sent, 1)
	})
}

type rb5URL struct {
	url.Provider
}

func (rb5URL) RegistryURL(context.Context, ...string) string { return "https://host/v2/root/reg" }
