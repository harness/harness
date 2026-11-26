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
	"sync"
	"testing"

	"github.com/harness/gitness/registry/app/driver/filesystem"
	"github.com/harness/gitness/registry/app/events/replication"
	"github.com/harness/gitness/registry/app/services/hook"
	registrystorage "github.com/harness/gitness/registry/app/storage"
	registrytypes "github.com/harness/gitness/registry/types"
	gitnessstore "github.com/harness/gitness/store"
	"github.com/harness/gitness/types"

	"github.com/opencontainers/go-digest"
	"github.com/stretchr/testify/require"
)

const rpgSecret = "rpg-shared-secret" //nolint:gosec // Test fixture, not a credential.

type rpgHook struct {
	mu      sync.Mutex
	commits int
	reads   int
	err     error
}

func (h *rpgHook) OnCommit(context.Context, hook.BlobCommitEvent) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.commits++
	return h.err
}

func (h *rpgHook) OnRead(context.Context, hook.BlobReadEvent) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.reads++
	return h.err
}

type rpgReplication struct {
	mu    sync.Mutex
	calls int
}

func (r *rpgReplication) ReportEventAsync(
	context.Context, string, replication.BlobAction, int64, string, string, *types.Config,
	[]replication.CloudLocation,
) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls++
}

// rpgEnv bundles a local registry wired to fake stores and a real filesystem blob store.
type rpgEnv struct {
	local     *LocalRegistry
	app       *App
	ms        *manifestService
	manifests *rpgManifestDao
	blobs     *rpgBlobDao
	regBlobs  *rpgRegBlobDao
	tags      *rpgTagDao
	artifacts *rpgArtifactDao
	images    *rpgImageDao
	regDao    *rpgRegistryDao
	layers    *rpgLayerDao
	refs      *rpgManifestRefDao
	mappings  *rpgMappingDao
	gc        *rpgGC
	hook      *rpgHook
	repl      *rpgReplication
	events    *rpgEventReporter
	producer  *rpgProducer
	rows      map[registrytypes.Digest]*registrytypes.Manifest
	untagged  bool
}

func rpgNewEnv(t *testing.T) *rpgEnv {
	t.Helper()
	env := &rpgEnv{
		manifests: &rpgManifestDao{},
		blobs:     &rpgBlobDao{},
		regBlobs:  &rpgRegBlobDao{unlink: true},
		tags:      &rpgTagDao{deleted: true},
		artifacts: &rpgArtifactDao{},
		images:    &rpgImageDao{},
		regDao:    &rpgRegistryDao{},
		layers:    &rpgLayerDao{},
		refs:      &rpgManifestRefDao{},
		mappings:  &rpgMappingDao{},
		gc:        &rpgGC{},
		hook:      &rpgHook{},
		repl:      &rpgReplication{},
		events:    &rpgEventReporter{},
		rows:      map[registrytypes.Digest]*registrytypes.Manifest{},
	}
	reg := rpgRegistry()
	env.regDao.reg = &reg

	env.manifests.byDigest = func(_ int64, _ string, d registrytypes.Digest) (*registrytypes.Manifest, error) {
		if m, ok := env.rows[d]; ok {
			return m, nil
		}
		return nil, gitnessstore.ErrResourceNotFound
	}
	env.manifests.createOrFind = func(m *registrytypes.Manifest) error {
		m.ID = int64(len(env.rows) + 100)
		key, err := registrytypes.NewDigest(m.Digest)
		if err != nil {
			return err
		}
		env.rows[key] = m
		return nil
	}
	env.manifests.get = func(id int64) (*registrytypes.Manifest, error) {
		for _, m := range env.rows {
			if m.ID == id {
				return m, nil
			}
		}
		return nil, gitnessstore.ErrResourceNotFound
	}
	env.blobs.byRepo = func(d digest.Digest) (*registrytypes.Blob, error) {
		return &registrytypes.Blob{ID: 9, Digest: d, Size: 3}, nil
	}
	env.blobs.byRoot = env.blobs.byRepo
	env.blobs.exists = func() (bool, error) { return true, nil }
	env.blobs.byID = func(id int64) (*registrytypes.Blob, error) {
		return &registrytypes.Blob{ID: id, Size: 123}, nil
	}
	env.blobs.createOrFind = func(b *registrytypes.Blob) (*registrytypes.Blob, bool, error) {
		b.ID = 77
		return b, true, nil
	}

	cfg := &types.Config{}
	cfg.Registry.HTTP.Secret = rpgSecret
	cfg.Registry.Storage.S3Storage.Delete = true
	fsDriver := filesystem.New(filesystem.DriverParameters{RootDirectory: t.TempDir(), MaxThreads: 10})
	storageSvc, err := registrystorage.NewStorageService(
		registrystorage.NewStaticStorageResolver(fsDriver), registrystorage.EnableDelete,
	)
	require.NoError(t, err)
	env.app = &App{
		Context: context.Background(), Config: cfg, storageService: storageSvc, spaceFinder: rpgSpaceFinder(nil),
	}

	artifactReporter, producer := rpgArtifactReporter(t)
	env.producer = producer
	env.ms = &manifestService{
		registryDao:             env.regDao,
		manifestDao:             env.manifests,
		layerDao:                env.layers,
		blobRepo:                env.blobs,
		mtRepository:            rpgMediaTypes{exists: true},
		tagDao:                  env.tags,
		imageDao:                env.images,
		artifactDao:             env.artifacts,
		manifestRefDao:          env.refs,
		ociImageIndexMappingDao: env.mappings,
		spaceFinder:             rpgSpaceFinder(nil),
		gcService:               env.gc,
		tx:                      rpgTx{},
		reporter:                env.events,
		artifactEventReporter:   artifactReporter,
		urlProvider:             rpgURLProvider{},
		untaggedImagesEnabled:   func(context.Context) bool { return env.untagged },
		auditService:            &rpgAudit{},
		registryBlobStore:       env.regBlobs,
	}
	env.local = &LocalRegistry{
		App:                 env.app,
		ms:                  env.ms,
		registryDao:         env.regDao,
		manifestDao:         env.manifests,
		registryBlobDao:     env.regBlobs,
		blobRepo:            env.blobs,
		tagDao:              env.tags,
		artifactDao:         env.artifacts,
		gcService:           env.gc,
		tx:                  rpgTx{},
		replicationReporter: env.repl,
		blobActionHook:      env.hook,
	}
	return env
}

// rpgAddManifest registers a manifest payload as stored in the fake manifest table.
func (e *rpgEnv) addManifest(t *testing.T, id int64, mediaType string, payload []byte) *registrytypes.Manifest {
	t.Helper()
	dgst := digest.FromBytes(payload)
	m := &registrytypes.Manifest{
		ID: id, RegistryID: 5, SchemaVersion: 2, MediaType: mediaType, Digest: dgst,
		Payload: payload, ImageName: "img",
	}
	e.rows[rpgStoredDigest(t, dgst)] = m
	return m
}

func rpgPayloadDigest(payload []byte) digest.Digest {
	return digest.FromBytes(payload)
}
