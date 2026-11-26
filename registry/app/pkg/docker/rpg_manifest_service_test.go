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
	"database/sql"
	"encoding/json"
	"testing"

	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	registryevents "github.com/harness/gitness/registry/app/events/artifact"
	"github.com/harness/gitness/registry/app/manifest"
	"github.com/harness/gitness/registry/app/manifest/manifestlist"
	"github.com/harness/gitness/registry/app/manifest/ocischema"
	"github.com/harness/gitness/registry/app/pkg"
	"github.com/harness/gitness/registry/app/store/database/util"
	registrytypes "github.com/harness/gitness/registry/types"
	gitnessstore "github.com/harness/gitness/store"

	"github.com/distribution/distribution/v3"
	"github.com/opencontainers/go-digest"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/stretchr/testify/require"
)

// rpgOtherManifest is a manifest implementation the service does not know how to store.
type rpgOtherManifest struct {
	manifest.Manifest
}

func (rpgOtherManifest) Payload() (string, []byte, error) { return "x/y", []byte("{}"), nil }

func rpgUpstreamInfo(t *testing.T) pkg.RegistryInfo {
	t.Helper()
	info := rpgInfo(t)
	info.Registry.Type = artifact.RegistryTypeUPSTREAM
	return info
}

func rpgBuildkitList(t *testing.T) (*manifestlist.DeserializedManifestList, []byte) {
	t.Helper()
	body := map[string]any{
		"schemaVersion": 2,
		"mediaType":     v1.MediaTypeImageIndex,
		"manifests": []any{
			map[string]any{"mediaType": MediaTypeBuildxCacheConfig, "size": 3, "digest": rpgDigest("cfg").String()},
			map[string]any{
				"mediaType": "application/vnd.oci.image.layer.v1.tar+gzip", "size": 4,
				"digest": rpgDigest("layer").String(),
			},
		},
	}
	payload, err := json.Marshal(body)
	require.NoError(t, err)
	return rpgManifestList(t, payload), payload
}

func TestRpgNewManifestServiceAndRegistries(t *testing.T) {
	svc := NewManifestService(
		nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, rpgSpaceFinder(nil), nil,
		registryevents.Reporter{}, nil, nil, nil, nil,
	)
	require.NotNil(t, svc)

	local := NewLocalRegistry(
		&App{}, svc, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
	)
	require.NotNil(t, local)

	store := NewDBStore(nil, nil, nil, nil, nil, nil, nil)
	require.NotNil(t, store)
}

func TestRpgDBPutManifestList(t *testing.T) {
	ctx := rpgAuthCtx(context.Background())
	_, childPayload := rpgDockerManifest(t, "child")
	childDigest := rpgPayloadDigest(childPayload)
	mt, payload := rpgIndex(t, rpgListMT, childDigest)
	listDigest := rpgPayloadDigest(payload)

	put := func(env *rpgEnv, info pkg.RegistryInfo) error {
		m, _, err := manifest.UnmarshalManifest(mt, payload)
		require.NoError(t, err)
		return env.ms.DBPut(ctx, m, listDigest, rpgHeaders(), info)
	}

	t.Run("stores the list and associates children", func(t *testing.T) {
		env := rpgNewEnv(t)
		env.addManifest(t, 1, rpgDockerMT, childPayload)
		require.NoError(t, put(env, rpgInfo(t)))
		require.Equal(t, 1, env.refs.calls)
		require.Zero(t, env.mappings.created)
	})
	t.Run("upstream registries skip missing children and map them", func(t *testing.T) {
		env := rpgNewEnv(t)
		require.NoError(t, put(env, rpgUpstreamInfo(t)))
		require.Zero(t, env.refs.calls)
		require.Equal(t, 1, env.mappings.created)

		env.mappings.createErr = errRpgBoom
		require.ErrorContains(t, put(env, rpgUpstreamInfo(t)), "failed to map manifest list")
	})
	t.Run("missing child on a local registry", func(t *testing.T) {
		env := rpgNewEnv(t)
		require.ErrorIs(t, put(env, rpgInfo(t)), gitnessstore.ErrResourceNotFound)
	})
	t.Run("child lookup failure", func(t *testing.T) {
		env := rpgNewEnv(t)
		env.manifests.byDigest = func(int64, string, registrytypes.Digest) (*registrytypes.Manifest, error) {
			return nil, errRpgBoom
		}
		require.ErrorIs(t, put(env, rpgInfo(t)), errRpgBoom)
	})
	t.Run("transaction failures", func(t *testing.T) {
		tests := []struct {
			name  string
			setup func(env *rpgEnv)
		}{
			{"gc lock", func(env *rpgEnv) { env.gc.mfNErr = errRpgBoom }},
			{"create", func(env *rpgEnv) {
				env.manifests.createOrFind = func(*registrytypes.Manifest) error { return errRpgBoom }
			}},
			{"image", func(env *rpgEnv) { env.images.upsertErr = errRpgBoom }},
			{"associate", func(env *rpgEnv) { env.refs.err = errRpgBoom }},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				env := rpgNewEnv(t)
				env.addManifest(t, 1, rpgDockerMT, childPayload)
				tt.setup(env)
				require.ErrorContains(t, put(env, rpgInfo(t)), "failed to create manifest list in database")
			})
		}
	})
	t.Run("child removed by garbage collection", func(t *testing.T) {
		env := rpgNewEnv(t)
		env.addManifest(t, 1, rpgDockerMT, childPayload)
		env.refs.err = util.ErrRefManifestNotFound
		err := put(env, rpgInfo(t))
		var verification distribution.ErrManifestVerification
		require.ErrorAs(t, err, &verification)
	})
	t.Run("buildkit cache index", func(t *testing.T) {
		env := rpgNewEnv(t)
		list, listPayload := rpgBuildkitList(t)
		require.NoError(t, env.ms.DBPut(ctx, list, rpgPayloadDigest(listPayload), rpgHeaders(), rpgInfo(t)))
		stored := env.rows[rpgStoredDigest(t, rpgPayloadDigest(listPayload))]
		require.NotNil(t, stored)
		require.True(t, stored.NonConformant)
	})
	t.Run("invalid buildkit cache index", func(t *testing.T) {
		env := rpgNewEnv(t)
		list, listPayload := rpgBuildkitList(t)
		list.Manifests = append(list.Manifests, manifestlist.ManifestDescriptor{
			Descriptor: manifest.Descriptor{MediaType: rpgDockerMT, Digest: rpgDigest("extra")},
		})
		err := env.ms.DBPut(ctx, list, rpgPayloadDigest(listPayload), rpgHeaders(), rpgInfo(t))
		require.ErrorContains(t, err, "converting buildkit index to manifest")
	})
}

func TestRpgDBPutImageIndex(t *testing.T) {
	ctx := rpgAuthCtx(context.Background())
	_, childPayload := rpgDockerManifest(t, "child")
	childDigest := rpgPayloadDigest(childPayload)
	mt, payload := rpgIndex(t, v1.MediaTypeImageIndex, childDigest)
	indexDigest := rpgPayloadDigest(payload)

	put := func(env *rpgEnv, info pkg.RegistryInfo) error {
		m, _, err := manifest.UnmarshalManifest(mt, payload)
		require.NoError(t, err)
		_, isIndex := m.(*ocischema.DeserializedImageIndex)
		require.True(t, isIndex)
		return env.ms.DBPut(ctx, m, indexDigest, rpgHeaders(), info)
	}

	t.Run("stores the index", func(t *testing.T) {
		env := rpgNewEnv(t)
		env.addManifest(t, 1, rpgDockerMT, childPayload)
		require.NoError(t, put(env, rpgInfo(t)))
		require.Equal(t, 1, env.refs.calls)
	})
	t.Run("upstream index maps children", func(t *testing.T) {
		env := rpgNewEnv(t)
		upstream := rpgRegistry()
		upstream.Type = artifact.RegistryTypeUPSTREAM
		env.regDao.reg = &upstream
		require.NoError(t, put(env, rpgUpstreamInfo(t)))
		require.Equal(t, 1, env.mappings.created)
		env.mappings.createErr = errRpgBoom
		require.ErrorContains(t, put(env, rpgUpstreamInfo(t)), "failed to map manifest index")
	})
	t.Run("registry lookup", func(t *testing.T) {
		env := rpgNewEnv(t)
		env.regDao.err = errRpgBoom
		require.ErrorIs(t, put(env, rpgInfo(t)), errRpgBoom)
		env.regDao.err = nil
		env.regDao.reg = nil
		require.ErrorContains(t, put(env, rpgInfo(t)), "repository not found")
	})
	t.Run("existing lookup failure", func(t *testing.T) {
		env := rpgNewEnv(t)
		env.manifests.byDigest = func(int64, string, registrytypes.Digest) (*registrytypes.Manifest, error) {
			return nil, errRpgBoom
		}
		require.ErrorIs(t, put(env, rpgInfo(t)), errRpgBoom)
	})
	t.Run("missing child", func(t *testing.T) {
		env := rpgNewEnv(t)
		require.ErrorContains(t, put(env, rpgInfo(t)), "failed to map manifest index")
	})
	t.Run("transaction failures", func(t *testing.T) {
		tests := []struct {
			name  string
			setup func(env *rpgEnv)
		}{
			{"gc lock", func(env *rpgEnv) { env.gc.mfNErr = errRpgBoom }},
			{"create", func(env *rpgEnv) {
				env.manifests.createOrFind = func(*registrytypes.Manifest) error { return errRpgBoom }
			}},
			{"image", func(env *rpgEnv) { env.images.upsertErr = errRpgBoom }},
			{"associate", func(env *rpgEnv) { env.refs.err = errRpgBoom }},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				env := rpgNewEnv(t)
				env.addManifest(t, 1, rpgDockerMT, childPayload)
				tt.setup(env)
				require.Error(t, put(env, rpgInfo(t)))
			})
		}
	})
	t.Run("child removed by garbage collection", func(t *testing.T) {
		env := rpgNewEnv(t)
		env.addManifest(t, 1, rpgDockerMT, childPayload)
		env.refs.err = util.ErrRefManifestNotFound
		var verification distribution.ErrManifestVerification
		require.ErrorAs(t, put(env, rpgInfo(t)), &verification)
	})
	t.Run("index with a subject", func(t *testing.T) {
		env := rpgNewEnv(t)
		env.addManifest(t, 1, rpgDockerMT, childPayload)
		body := map[string]any{
			"schemaVersion": 2, "mediaType": v1.MediaTypeImageIndex, "artifactType": "type/index",
			"manifests": []any{map[string]any{
				"mediaType": rpgDockerMT, "size": 5, "digest": childDigest.String(),
				"platform": map[string]any{"architecture": "amd64", "os": "linux"},
			}},
			"subject": map[string]any{"mediaType": rpgDockerMT, "size": 5, "digest": childDigest.String()},
		}
		subjectPayload, err := json.Marshal(body)
		require.NoError(t, err)
		m, _, err := manifest.UnmarshalManifest(v1.MediaTypeImageIndex, subjectPayload)
		require.NoError(t, err)
		headers := rpgHeaders()
		require.NoError(t, env.ms.DBPut(ctx, m, rpgPayloadDigest(subjectPayload), headers, rpgInfo(t)))
		require.Equal(t, childDigest.String(), headers.Headers["OCI-Subject"])
	})
}

func TestRpgDBPutManifestFailures(t *testing.T) {
	ctx := rpgAuthCtx(context.Background())
	mt, payload := rpgDockerManifest(t, "fail")
	dgst := rpgPayloadDigest(payload)

	put := func(env *rpgEnv) error {
		m, _, err := manifest.UnmarshalManifest(mt, payload)
		require.NoError(t, err)
		return env.ms.DBPut(ctx, m, dgst, rpgHeaders(), rpgInfo(t))
	}

	tests := []struct {
		name  string
		setup func(env *rpgEnv)
		want  string
	}{
		{"existing lookup", func(env *rpgEnv) {
			env.manifests.byDigest = func(int64, string, registrytypes.Digest) (*registrytypes.Manifest, error) {
				return nil, errRpgBoom
			}
		}, "boom"},
		{"create", func(env *rpgEnv) {
			env.manifests.createOrFind = func(*registrytypes.Manifest) error { return errRpgBoom }
		}, "failed to create manifest and artifact"},
		{"image", func(env *rpgEnv) { env.images.upsertErr = errRpgBoom }, "failed to create manifest and artifact"},
		{"artifact", func(env *rpgEnv) { env.artifacts.upsertErr = errRpgBoom }, "failed to create manifest and artifact"},
		{"layer blob", func(env *rpgEnv) {
			calls := 0
			env.blobs.byRepo = func(d digest.Digest) (*registrytypes.Blob, error) {
				calls++
				if calls > 1 {
					return nil, errRpgBoom
				}
				return &registrytypes.Blob{ID: 1, Digest: d}, nil
			}
		}, "failed to create manifest and artifact"},
		{"layer association", func(env *rpgEnv) { env.layers.err = errRpgBoom }, "failed to create manifest and artifact"},
		{"unknown media type", func(env *rpgEnv) {
			env.layers.err = util.UnknownMediaTypeError{MediaType: "x/y"}
		}, "manifest invalid"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := rpgNewEnv(t)
			tt.setup(env)
			require.ErrorContains(t, put(env), tt.want)
		})
	}

	t.Run("unknown layer media types are tolerated", func(t *testing.T) {
		env := rpgNewEnv(t)
		env.ms.mtRepository = rpgMediaTypes{exists: false}
		require.NoError(t, put(env))
		env = rpgNewEnv(t)
		env.ms.mtRepository = rpgMediaTypes{err: errRpgBoom}
		require.NoError(t, put(env))
	})
	t.Run("unsupported manifest type", func(t *testing.T) {
		env := rpgNewEnv(t)
		err := env.ms.DBPut(ctx, rpgOtherManifest{}, dgst, rpgHeaders(), rpgInfo(t))
		require.ErrorContains(t, err, "manifest invalid")
	})
	t.Run("subject lookup failure", func(t *testing.T) {
		env := rpgNewEnv(t)
		subject := rpgDigest("subject")
		env.manifests.byDigest = func(int64, string, registrytypes.Digest) (*registrytypes.Manifest, error) {
			return nil, errRpgBoom
		}
		_, ociPayload := rpgOCIManifest(t, "oci", &subject)
		m, _, err := manifest.UnmarshalManifest(v1.MediaTypeImageManifest, ociPayload)
		require.NoError(t, err)
		require.ErrorIs(t, env.ms.DBPut(ctx, m, rpgPayloadDigest(ociPayload), rpgHeaders(), rpgInfo(t)), errRpgBoom)
	})
	t.Run("subject not yet stored", func(t *testing.T) {
		env := rpgNewEnv(t)
		subject := rpgDigest("subject")
		_, ociPayload := rpgOCIManifest(t, "oci", &subject)
		m, _, err := manifest.UnmarshalManifest(v1.MediaTypeImageManifest, ociPayload)
		require.NoError(t, err)
		require.NoError(t, env.ms.DBPut(ctx, m, rpgPayloadDigest(ociPayload), rpgHeaders(), rpgInfo(t)))
		stored := env.rows[rpgStoredDigest(t, rpgPayloadDigest(ociPayload))]
		require.False(t, stored.SubjectID.Valid)
		require.Equal(t, subject, stored.SubjectDigest)
	})
}

func TestRpgDBPutUntaggedMode(t *testing.T) {
	ctx := rpgAuthCtx(context.Background())
	mt, payload := rpgDockerManifest(t, "untagged")
	dgst := rpgPayloadDigest(payload)
	m, _, err := manifest.UnmarshalManifest(mt, payload)
	require.NoError(t, err)

	t.Run("space lookup failure", func(t *testing.T) {
		env := rpgNewEnv(t)
		env.untagged = true
		env.ms.spaceFinder = rpgSpaceFinder(errRpgBoom)
		require.ErrorIs(t, env.ms.DBPut(ctx, m, dgst, rpgHeaders(), rpgInfo(t)), errRpgBoom)
	})
	t.Run("unknown package type", func(t *testing.T) {
		env := rpgNewEnv(t)
		env.untagged = true
		info := rpgInfo(t)
		info.Registry.PackageType = "NOPE"
		require.ErrorContains(t, env.ms.DBPut(ctx, m, dgst, rpgHeaders(), info), "invalid PackageType")
	})
	t.Run("digest lookup failure after the put", func(t *testing.T) {
		env := rpgNewEnv(t)
		env.untagged = true
		env.manifests.byDigest = func(int64, string, registrytypes.Digest) (*registrytypes.Manifest, error) {
			if len(env.rows) > 0 {
				return nil, errRpgBoom
			}
			return nil, gitnessstore.ErrResourceNotFound
		}
		require.ErrorIs(t, env.ms.DBPut(ctx, m, dgst, rpgHeaders(), rpgInfo(t)), errRpgBoom)
	})
	t.Run("payload error", func(t *testing.T) {
		env := rpgNewEnv(t)
		require.Error(t, env.ms.DBPut(ctx, &rpgBrokenManifest{}, dgst, rpgHeaders(), rpgInfo(t)))
	})
}

type rpgBrokenManifest struct {
	manifest.Manifest
}

func (rpgBrokenManifest) Payload() (string, []byte, error) { return "", nil, errRpgBoom }

func TestRpgDBTagAndHandleTagError(t *testing.T) {
	ctx := rpgAuthCtx(context.Background())
	mt, payload := rpgDockerManifest(t, "tag")
	dgst := rpgPayloadDigest(payload)
	m, _, err := manifest.UnmarshalManifest(mt, payload)
	require.NoError(t, err)

	t.Run("tags an existing manifest", func(t *testing.T) {
		env := rpgNewEnv(t)
		env.addManifest(t, 1, mt, payload)
		require.NoError(t, env.ms.DBTag(ctx, m, dgst, "v1", rpgHeaders(), rpgInfo(t)))
		require.Len(t, env.tags.created, 1)
	})
	t.Run("manifest removed by gc cannot be recreated", func(t *testing.T) {
		env := rpgNewEnv(t)
		env.addManifest(t, 1, mt, payload)
		env.tags.createErrs = []error{util.ErrManifestNotFound}
		env.blobs.byRepo = func(digest.Digest) (*registrytypes.Blob, error) { return nil, errRpgBoom }
		err := env.ms.DBTag(ctx, m, dgst, "v1", rpgHeaders(), rpgInfo(t))
		require.ErrorContains(t, err, "failed to recreate manifest in database")
	})
	t.Run("other tag error", func(t *testing.T) {
		env := rpgNewEnv(t)
		env.addManifest(t, 1, mt, payload)
		env.tags.createErr = errRpgBoom
		require.ErrorContains(t, env.ms.DBTag(ctx, m, dgst, "v1", rpgHeaders(), rpgInfo(t)), "failed to create tag")
	})
	t.Run("lookup problems", func(t *testing.T) {
		env := rpgNewEnv(t)
		err := env.ms.dbTagManifest(ctx, dgst, "v1", "img", rpgInfo(t))
		require.ErrorContains(t, err, "not found in database")

		err = env.ms.dbTagManifest(ctx, "bad", "v1", "img", rpgInfo(t))
		require.ErrorContains(t, err, "failed to tag manifest")

		env.manifests.byDigest = func(int64, string, registrytypes.Digest) (*registrytypes.Manifest, error) {
			return nil, errRpgBoom
		}
		err = env.ms.dbTagManifest(ctx, dgst, "v1", "img", rpgInfo(t))
		require.ErrorIs(t, err, errRpgBoom)
	})
	t.Run("event reporting is skipped without a space", func(t *testing.T) {
		env := rpgNewEnv(t)
		env.addManifest(t, 1, mt, payload)
		env.ms.spaceFinder = rpgSpaceFinder(errRpgBoom)
		require.NoError(t, env.ms.dbTagManifest(ctx, dgst, "v1", "img", rpgInfo(t)))
		require.Zero(t, env.producer.count())
	})
}

func TestRpgAddTagsToManifest(t *testing.T) {
	ctx := rpgAuthCtx(context.Background())
	mt, payload := rpgDockerManifest(t, "addtags")
	dgst := rpgPayloadDigest(payload)

	t.Run("only new tags are added", func(t *testing.T) {
		env := rpgNewEnv(t)
		env.addManifest(t, 1, mt, payload)
		env.tags.manifestTg = []string{"old"}
		require.NoError(t, env.ms.AddTagsToManifest(ctx, dgst, []string{"old", "new"}, "img", rpgInfo(t)))
		require.Len(t, env.tags.created, 1)
		require.Equal(t, "new", env.tags.created[0].Name)
	})
	t.Run("failures", func(t *testing.T) {
		env := rpgNewEnv(t)
		require.ErrorContains(t, env.ms.AddTagsToManifest(ctx, dgst, nil, "img", rpgInfo(t)), "not found in database")
		require.ErrorContains(t, env.ms.AddTagsToManifest(ctx, "bad", nil, "img", rpgInfo(t)), "failed to tag manifest")

		env.addManifest(t, 1, mt, payload)
		env.tags.mfTagsErr = errRpgBoom
		require.ErrorIs(t, env.ms.AddTagsToManifest(ctx, dgst, []string{"a"}, "img", rpgInfo(t)), errRpgBoom)

		env.tags.mfTagsErr = nil
		env.tags.createErr = errRpgBoom
		require.ErrorIs(t, env.ms.AddTagsToManifest(ctx, dgst, []string{"a"}, "img", rpgInfo(t)), errRpgBoom)

		env.manifests.byDigest = func(int64, string, registrytypes.Digest) (*registrytypes.Manifest, error) {
			return nil, errRpgBoom
		}
		require.ErrorIs(t, env.ms.AddTagsToManifest(ctx, dgst, []string{"a"}, "img", rpgInfo(t)), errRpgBoom)
	})
	t.Run("reports even if the space is unknown", func(t *testing.T) {
		env := rpgNewEnv(t)
		env.addManifest(t, 1, mt, payload)
		env.ms.spaceFinder = rpgSpaceFinder(errRpgBoom)
		require.NoError(t, env.ms.AddTagsToManifest(ctx, dgst, []string{"a"}, "img", rpgInfo(t)))
	})
}

func TestRpgLockManifestForGC(t *testing.T) {
	env := rpgNewEnv(t)
	ctx := context.Background()
	require.NoError(t, env.ms.lockManifestForGC(ctx, 1, 2))
	env.gc.mfErr = sql.ErrNoRows
	require.NoError(t, env.ms.lockManifestForGC(ctx, 1, 2))
	env.gc.mfErr = errRpgBoom
	require.ErrorContains(t, env.ms.lockManifestForGC(ctx, 1, 2), "failed to lock manifest for GC review")
}

func TestRpgUpsertImage(t *testing.T) {
	ctx := context.Background()
	env := rpgNewEnv(t)
	info := rpgInfo(t)

	env.images.getErr = gitnessstore.ErrResourceNotFound
	require.NoError(t, env.ms.UpsertImage(ctx, info))
	require.Len(t, env.images.upserted, 1)
	require.False(t, env.images.upserted[0].Enabled)

	env.images.getErr = nil
	env.images.getImage = &registrytypes.Image{ID: 1}
	require.NoError(t, env.ms.UpsertImage(ctx, info))
	require.Len(t, env.images.upserted, 1)

	env.images.getImage = nil
	env.images.getErr = errRpgBoom
	require.ErrorIs(t, env.ms.UpsertImage(ctx, info), errRpgBoom)

	env.images.getErr = gitnessstore.ErrResourceNotFound
	env.images.upsertErr = errRpgBoom
	require.ErrorIs(t, env.ms.UpsertImage(ctx, info), errRpgBoom)
}

func TestRpgAddManifestAssociation(t *testing.T) {
	ctx := context.Background()
	mt, payload := rpgDockerManifest(t, "child")
	dgst := rpgPayloadDigest(payload)

	setup := func(t *testing.T) *rpgEnv {
		env := rpgNewEnv(t)
		env.addManifest(t, 1, mt, payload)
		env.addManifest(t, 2, rpgListMT, []byte(`{"p":1}`))
		env.mappings.mappings = []*registrytypes.OCIImageIndexMapping{{ParentManifestID: 2}}
		return env
	}

	env := setup(t)
	require.NoError(t, env.ms.AddManifestAssociation(ctx, "reg", dgst, rpgInfo(t)))
	require.Equal(t, 1, env.refs.calls)

	env = setup(t)
	require.ErrorContains(t, env.ms.AddManifestAssociation(ctx, "reg", "bad", rpgInfo(t)), "failed to create digest")

	env = setup(t)
	env.regDao.err = errRpgBoom
	require.ErrorContains(t, env.ms.AddManifestAssociation(ctx, "reg", dgst, rpgInfo(t)), "failed to get registry")

	env = setup(t)
	require.ErrorContains(
		t, env.ms.AddManifestAssociation(ctx, "reg", rpgDigest("other"), rpgInfo(t)), "failed to find manifest",
	)

	env = setup(t)
	env.mappings.getErr = errRpgBoom
	require.ErrorContains(t, env.ms.AddManifestAssociation(ctx, "reg", dgst, rpgInfo(t)), "oci image index mappings")

	env = setup(t)
	env.mappings.mappings = []*registrytypes.OCIImageIndexMapping{{ParentManifestID: 404}}
	require.ErrorContains(t, env.ms.AddManifestAssociation(ctx, "reg", dgst, rpgInfo(t)), "failed to get manifest")

	env = setup(t)
	env.refs.err = util.ErrRefManifestNotFound
	require.ErrorIs(t, env.ms.AddManifestAssociation(ctx, "reg", dgst, rpgInfo(t)), util.ErrRefManifestNotFound)

	// Other association failures are only skipped over.
	env = setup(t)
	env.refs.err = errRpgBoom
	require.NoError(t, env.ms.AddManifestAssociation(ctx, "reg", dgst, rpgInfo(t)))
}

func TestRpgManifestServiceDeleteTag(t *testing.T) {
	ctx := rpgAuthCtx(context.Background())
	env := rpgNewEnv(t)
	info := rpgInfo(t)

	env.regDao.err = errRpgBoom
	_, err := env.ms.DeleteTag(ctx, "reg", "v1", info)
	require.ErrorIs(t, err, errRpgBoom)
	_, err = env.ms.DeleteTagsByManifestID(ctx, "reg", 1, info)
	require.ErrorIs(t, err, errRpgBoom)

	env.regDao.err = nil
	env.tags.deleteErr = errRpgBoom
	_, err = env.ms.DeleteTag(ctx, "reg", "v1", info)
	require.ErrorContains(t, err, "failed to delete tag in database")

	env.tags.deleteErr = nil
	env.untagged = true
	_, payload := rpgDockerManifest(t, "deltag")
	row := env.addManifest(t, 1, rpgDockerMT, payload)
	env.tags.findTag = &registrytypes.Tag{ManifestID: row.ID}
	ok, err := env.ms.DeleteTag(ctx, "reg", "v1", info)
	require.NoError(t, err)
	require.True(t, ok)

	deleted, err := env.ms.DeleteTagsByManifestID(ctx, "reg", 1, info)
	require.NoError(t, err)
	require.True(t, deleted)
}

func TestRpgManifestServiceDeleteManifest(t *testing.T) {
	ctx := context.Background()
	mt, payload := rpgDockerManifest(t, "delete")
	dgst := rpgPayloadDigest(payload)
	_, listPayload := rpgIndex(t, rpgListMT, rpgDigest("child"))

	setup := func(t *testing.T) *rpgEnv {
		env := rpgNewEnv(t)
		env.addManifest(t, 1, mt, payload)
		env.manifests.referencedBy = func(*registrytypes.Manifest) (registrytypes.Manifests, error) { return nil, nil }
		env.manifests.deleteMf = func(digest.Digest) (bool, error) { return true, nil }
		env.manifests.count = func() (int64, error) { return 1, nil }
		return env
	}

	t.Run("removes the image when it was the last manifest", func(t *testing.T) {
		env := setup(t)
		env.manifests.count = func() (int64, error) { return 0, nil }
		require.NoError(t, env.ms.DeleteManifest(ctx, "reg", dgst, rpgInfo(t)))
	})
	t.Run("keeps the image when manifests remain", func(t *testing.T) {
		env := setup(t)
		env.images.deleteErr = errRpgBoom
		require.NoError(t, env.ms.DeleteManifest(ctx, "reg", dgst, rpgInfo(t)))
	})
	t.Run("lists lock their children", func(t *testing.T) {
		env := setup(t)
		list := env.addManifest(t, 2, rpgListMT, listPayload)
		env.manifests.references = func(*registrytypes.Manifest) (registrytypes.Manifests, error) {
			return registrytypes.Manifests{{ID: 1}}, nil
		}
		require.NoError(t, env.ms.DeleteManifest(ctx, "reg", list.Digest, rpgInfo(t)))

		env.manifests.references = func(*registrytypes.Manifest) (registrytypes.Manifests, error) { return nil, nil }
		require.NoError(t, env.ms.DeleteManifest(ctx, "reg", list.Digest, rpgInfo(t)))

		env.manifests.references = func(*registrytypes.Manifest) (registrytypes.Manifests, error) {
			return nil, errRpgBoom
		}
		require.ErrorIs(t, env.ms.DeleteManifest(ctx, "reg", list.Digest, rpgInfo(t)), errRpgBoom)

		env.manifests.references = func(*registrytypes.Manifest) (registrytypes.Manifests, error) {
			return registrytypes.Manifests{{ID: 1}}, nil
		}
		env.gc.mfNErr = errRpgBoom
		require.ErrorIs(t, env.ms.DeleteManifest(ctx, "reg", list.Digest, rpgInfo(t)), errRpgBoom)
	})
	t.Run("failures", func(t *testing.T) {
		tests := []struct {
			name  string
			setup func(env *rpgEnv)
			want  string
		}{
			{"registry", func(env *rpgEnv) { env.regDao.err = errRpgBoom }, "repository not found"},
			{"registry missing", func(env *rpgEnv) { env.regDao.reg = nil }, "repository not found"},
			{"not found", func(env *rpgEnv) { env.rows = map[registrytypes.Digest]*registrytypes.Manifest{} }, "manifest"},
			{"referenced by a list", func(env *rpgEnv) {
				env.manifests.referencedBy = func(*registrytypes.Manifest) (registrytypes.Manifests, error) {
					return registrytypes.Manifests{{Digest: rpgDigest("parent")}}, nil
				}
			}, "cannot delete manifest"},
			{"referenced lookup", func(env *rpgEnv) {
				env.manifests.referencedBy = func(*registrytypes.Manifest) (registrytypes.Manifests, error) {
					return nil, errRpgBoom
				}
			}, "failed to find existing manifests"},
			{"delete", func(env *rpgEnv) {
				env.manifests.deleteMf = func(digest.Digest) (bool, error) { return false, errRpgBoom }
			}, "boom"},
			{"delete not found", func(env *rpgEnv) {
				env.manifests.deleteMf = func(digest.Digest) (bool, error) { return false, nil }
			}, "manifest"},
			{"tags", func(env *rpgEnv) { env.tags.deleteErr = errRpgBoom }, "failed to delete tags"},
			{"artifact", func(env *rpgEnv) { env.artifacts.deleteErr = errRpgBoom }, "failed to delete artifact"},
			{"count", func(env *rpgEnv) {
				env.manifests.count = func() (int64, error) { return 0, errRpgBoom }
			}, "failed to count manifests"},
			{"image", func(env *rpgEnv) {
				env.manifests.count = func() (int64, error) { return 0, nil }
				env.images.deleteErr = errRpgBoom
			}, "failed to delete image"},
			{"unlink", func(env *rpgEnv) {
				env.manifests.count = func() (int64, error) { return 0, nil }
				env.regBlobs.unlinkErr = errRpgBoom
			}, "failed to unlink registry blobs"},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				env := setup(t)
				tt.setup(env)
				require.ErrorContains(t, env.ms.DeleteManifest(ctx, "reg", dgst, rpgInfo(t)), tt.want)
			})
		}
	})
	t.Run("invalid digest", func(t *testing.T) {
		env := setup(t)
		require.Error(t, env.ms.DeleteManifest(ctx, "reg", "bad", rpgInfo(t)))
	})
	t.Run("lookup failure", func(t *testing.T) {
		env := setup(t)
		env.manifests.byDigest = func(int64, string, registrytypes.Digest) (*registrytypes.Manifest, error) {
			return nil, errRpgBoom
		}
		require.ErrorIs(t, env.ms.DeleteManifest(ctx, "reg", dgst, rpgInfo(t)), errRpgBoom)
	})
}
