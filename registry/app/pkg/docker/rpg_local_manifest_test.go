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
	"bytes"
	"context"
	"database/sql"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/harness/gitness/registry/app/dist_temp/errcode"
	"github.com/harness/gitness/registry/app/manifest"
	"github.com/harness/gitness/registry/app/pkg"
	"github.com/harness/gitness/registry/app/store/database/util"
	registrytypes "github.com/harness/gitness/registry/types"
	gitnessstore "github.com/harness/gitness/store"

	"github.com/distribution/distribution/v3"
	"github.com/opencontainers/go-digest"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/stretchr/testify/require"
)

const (
	rpgLatestTag = "latest"
	rpgDockerMT  = "application/vnd.docker.distribution.manifest.v2+json"
	rpgListMT    = "application/vnd.docker.distribution.manifest.list.v2+json"
)

func rpgCodes(errs []error) []errcode.CodeError {
	out := make([]errcode.CodeError, 0, len(errs))
	for _, e := range errs {
		var coder errcode.ErrorCoder
		if errors.As(e, &coder) {
			out = append(out, coder.ErrorCode())
		} else {
			out = append(out, errcode.CodeError(-1))
		}
	}
	return out
}

func TestRpgEtagMatch(t *testing.T) {
	require.True(t, etagMatch([]string{"x", "abc"}, "abc"))
	require.True(t, etagMatch([]string{`"abc"`}, "abc"))
	require.False(t, etagMatch([]string{"nope"}, "abc"))
	require.False(t, etagMatch(nil, "abc"))
}

func TestRpgGetSupportsList(t *testing.T) {
	r := &LocalRegistry{}
	got := r.getSupportsList([]string{
		rpgDockerMT + ", " + rpgListMT,
		v1.MediaTypeImageManifest,
		"bad media type;;,",
		v1.MediaTypeImageIndex,
	})
	require.Equal(t, [4]bool{true, true, true, true}, got)
	require.Equal(t, [4]bool{}, r.getSupportsList(nil))
	require.Equal(t, [4]bool{false, false, true, false}, r.getSupportsList([]string{v1.MediaTypeImageManifest}))
}

func TestRpgGetDigestFromInfo(t *testing.T) {
	require.Equal(t, digest.Digest("a"), getDigestFromInfo(pkg.RegistryInfo{Digest: "a", Reference: "b"}))
	require.Equal(t, digest.Digest("b"), getDigestFromInfo(pkg.RegistryInfo{Reference: "b"}))
}

func TestRpgSimpleLocalRegistryAccessors(t *testing.T) {
	r := &LocalRegistry{}
	require.NoError(t, r.Base())
	require.Len(t, r.GetPackageTypes(), 2)
	require.NotEmpty(t, r.GetArtifactType())
	require.Panics(t, func() { _, _, _ = r.CanBeMount() })

	ctx := context.Background()
	info := pkg.RegistryInfo{}
	require.NoError(t, r.PushBlobMonolith(ctx, info, 0, nil))
	require.NoError(t, r.PushBlobMonolithWithDigest(ctx, info, 0, nil))
	require.NoError(t, r.MountBlob(ctx, info, "", ""))
	require.NoError(t, r.DeleteTag("", "", info))
	tags, err := r.ListFilteredTags(ctx, 0, "", "", info)
	require.NoError(t, err)
	require.Nil(t, tags)
	repos, err := r.GetCatalog()
	require.NoError(t, err)
	require.Nil(t, repos)
	size, blob, err := r.PullBlobChunk("", "", 0, 0, 0, info)
	require.NoError(t, err)
	require.Nil(t, blob)
	require.Zero(t, size)
	require.NoError(t, tagserviceTag())
}

func TestRpgDBManifestToManifest(t *testing.T) {
	_, dockerPayload := rpgDockerManifest(t, "m")
	_, ociPayload := rpgOCIManifest(t, "m", nil)
	_, listPayload := rpgIndex(t, rpgListMT, rpgDigest("child"))
	_, ociIndexPayload := rpgIndex(t, v1.MediaTypeImageIndex, rpgDigest("child"))

	tests := []struct {
		name    string
		m       registrytypes.Manifest
		wantErr string
		wantNil bool
	}{
		{name: "schema v1", m: registrytypes.Manifest{SchemaVersion: 1}, wantErr: "schema"},
		{name: "unknown schema", m: registrytypes.Manifest{SchemaVersion: 3}, wantErr: "unrecognized manifest schema"},
		{name: "docker", m: registrytypes.Manifest{SchemaVersion: 2, MediaType: rpgDockerMT, Payload: dockerPayload}},
		{
			name: "oci",
			m:    registrytypes.Manifest{SchemaVersion: 2, MediaType: v1.MediaTypeImageManifest, Payload: ociPayload},
		},
		{name: "list", m: registrytypes.Manifest{SchemaVersion: 2, MediaType: rpgListMT, Payload: listPayload}},
		{
			name: "oci index",
			m:    registrytypes.Manifest{SchemaVersion: 2, MediaType: v1.MediaTypeImageIndex, Payload: ociIndexPayload},
		},
		{name: "empty media type index", m: registrytypes.Manifest{SchemaVersion: 2, Payload: ociIndexPayload}},
		{name: "empty media type manifest", m: registrytypes.Manifest{SchemaVersion: 2, Payload: ociPayload}},
		{
			name: "non conformant",
			m:    registrytypes.Manifest{SchemaVersion: 2, NonConformant: true, Payload: ociPayload, MediaType: "x"},
		},
		{
			name:    "non conformant bad payload",
			m:       registrytypes.Manifest{SchemaVersion: 2, NonConformant: true, Payload: []byte("{")},
			wantErr: "unmarshal",
		},
		{name: "unknown type", m: registrytypes.Manifest{SchemaVersion: 2, MediaType: "text/plain"}, wantErr: "unrecognized"},
		{
			name:    "bad docker payload",
			m:       registrytypes.Manifest{SchemaVersion: 2, MediaType: rpgDockerMT, Payload: []byte("{")},
			wantErr: "unexpected end",
		},
		{
			name:    "bad oci payload",
			m:       registrytypes.Manifest{SchemaVersion: 2, MediaType: v1.MediaTypeImageManifest, Payload: []byte("{")},
			wantErr: "unexpected end",
		},
		{
			name:    "bad list payload",
			m:       registrytypes.Manifest{SchemaVersion: 2, MediaType: rpgListMT, Payload: []byte("{")},
			wantErr: "unexpected end",
		},
		{
			name:    "bad empty-type payload",
			m:       registrytypes.Manifest{SchemaVersion: 2, Payload: []byte("{")},
			wantErr: "unexpected end",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := DBManifestToManifest(&tt.m)
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, got)
		})
	}
}

func TestRpgCopyFullPayload(t *testing.T) {
	var buf bytes.Buffer
	require.NoError(t, copyFullPayload(context.Background(), 3, io.NopCloser(strings.NewReader("abc")), &buf, "test"))
	require.Equal(t, "abc", buf.String())

	// A short body with a cancelled context looks like a client disconnect.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := copyFullPayload(ctx, 10, io.NopCloser(strings.NewReader("abc")), io.Discard, "test")
	require.ErrorContains(t, err, "client disconnected")

	// A read error with a live context is reported as is.
	err = copyFullPayload(
		context.Background(), 3, io.NopCloser(&rpgFailingReader{}), io.Discard, "test",
	)
	require.ErrorIs(t, err, errRpgBoom)
}

type rpgFailingReader struct{}

func (*rpgFailingReader) Read([]byte) (int, error) { return 0, errRpgBoom }

func TestRpgParseContentRange(t *testing.T) {
	start, end, err := parseContentRange("3-9")
	require.NoError(t, err)
	require.Equal(t, int64(3), start)
	require.Equal(t, int64(9), end)

	for _, bad := range []string{"39", "a-9", "3-b"} {
		_, _, err = parseContentRange(bad)
		require.Error(t, err, bad)
	}
}

func TestRpgUploadStateRoundTrip(t *testing.T) {
	key := hmacKey("secret")
	state := BlobUploadState{Name: "n", Path: "p", UUID: "u", Offset: 12}
	token, err := key.packUploadState(state)
	require.NoError(t, err)

	got, err := key.unpackUploadState(token)
	require.NoError(t, err)
	require.Equal(t, state, got)

	_, err = hmacKey("other").unpackUploadState(token)
	require.ErrorIs(t, err, errInvalidSecret)
	_, err = key.unpackUploadState("short")
	require.Error(t, err)
	_, err = key.unpackUploadState("!!!")
	require.ErrorContains(t, err, "failed to decode token")
}

func TestRpgAppendPutError(t *testing.T) {
	r := &LocalRegistry{}
	tests := []struct {
		name string
		err  error
		want []errcode.CodeError
	}{
		{"unsupported", manifest.ErrUnsupported, []errcode.CodeError{errcode.ErrCodeUnsupported}},
		{"denied", manifest.ErrAccessDenied, []errcode.CodeError{errcode.ErrCodeDenied}},
		{"schema v1", manifest.ErrSchemaV1Unsupported, []errcode.CodeError{errcode.ErrCodeManifestInvalid}},
		{"digest format", digest.ErrDigestInvalidFormat, []errcode.CodeError{errcode.ErrCodeDigestInvalid}},
		{
			"verification",
			manifest.VerificationErrors{
				manifest.BlobUnknownError{Digest: rpgDigest("a")},
				manifest.NameInvalidError{},
				manifest.UnverifiedError{},
				manifest.ReferencesExceedLimitError{},
				manifest.PayloadSizeExceedsLimitError{},
				digest.ErrDigestInvalidFormat,
				errRpgBoom,
			},
			[]errcode.CodeError{
				errcode.ErrCodeManifestBlobUnknown, errcode.ErrCodeNameInvalid, errcode.ErrCodeManifestUnverified,
				errcode.ErrCodeManifestReferenceLimit, errcode.ErrCodeManifestPayloadSizeLimit,
				errcode.ErrCodeDigestInvalid, errcode.ErrCodeUnknown,
			},
		},
		{"errcode", errcode.ErrCodeTagInvalid.WithDetail("x"), []errcode.CodeError{errcode.ErrCodeTagInvalid}},
		{"unknown", errRpgBoom, []errcode.CodeError{errcode.ErrCodeUnknown}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, rpgCodes(r.appendPutError(tt.err, nil)))
		})
	}
}

func TestRpgHandlePutManifestErrors(t *testing.T) {
	r := &LocalRegistry{}
	headers := rpgHeaders()
	tests := []struct {
		name string
		err  error
		want []errcode.CodeError
	}{
		{"unsupported", manifest.ErrUnsupported, []errcode.CodeError{errcode.ErrCodeUnsupported}},
		{"denied", manifest.ErrAccessDenied, []errcode.CodeError{errcode.ErrCodeDenied}},
		{
			"verification",
			manifest.VerificationErrors{
				manifest.BlobUnknownError{Digest: rpgDigest("a")},
				manifest.NameInvalidError{},
				manifest.UnverifiedError{},
				digest.ErrDigestInvalidFormat,
				errRpgBoom,
			},
			[]errcode.CodeError{
				errcode.ErrCodeManifestBlobUnknown, errcode.ErrCodeNameInvalid, errcode.ErrCodeManifestUnverified,
				errcode.ErrCodeDigestInvalid, errcode.ErrCodeUnknown, -1,
			},
		},
		{"errcode", errcode.ErrCodeTagInvalid.WithDetail("x"), []errcode.CodeError{errcode.ErrCodeTagInvalid}},
		{"unknown", errRpgBoom, []errcode.CodeError{errcode.ErrCodeUnknown}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, errs := r.handlePutManifestErrors(tt.err, nil, headers)
			require.Same(t, headers, got)
			require.Equal(t, tt.want, rpgCodes(errs))
		})
	}
}

func TestRpgGetManifestAndTag(t *testing.T) {
	ctx := context.Background()
	env := rpgNewEnv(t)
	info := rpgInfo(t)
	_, payload := rpgDockerManifest(t, "m")
	row := env.addManifest(t, 1, rpgDockerMT, payload)

	got, err := env.local.getManifest(ctx, row.Digest, "img", info)
	require.NoError(t, err)
	require.NotNil(t, got)

	_, err = env.local.getManifest(ctx, rpgDigest("missing"), "img", info)
	var unknown manifest.UnknownRevisionError
	require.ErrorAs(t, err, &unknown)

	env.manifests.byDigest = func(int64, string, registrytypes.Digest) (*registrytypes.Manifest, error) {
		return nil, errRpgBoom
	}
	_, err = env.local.getManifest(ctx, row.Digest, "img", info)
	require.ErrorIs(t, err, errRpgBoom)

	env.manifests.byTag = func(_ int64, _ string, tag string) (*registrytypes.Manifest, error) {
		switch tag {
		case rpgLatestTag:
			return row, nil
		case "gone":
			return nil, gitnessstore.ErrResourceNotFound
		}
		return nil, errRpgBoom
	}
	info.Tag = rpgLatestTag
	desc, err := env.local.getTag(ctx, info)
	require.NoError(t, err)
	require.Equal(t, row.Digest, desc.Digest)
	dgst, err := env.local.getDigest(ctx, info)
	require.NoError(t, err)
	require.Equal(t, row.Digest, dgst)

	info.Tag = "gone"
	_, err = env.local.getTag(ctx, info)
	var tagUnknown manifest.TagUnknownError
	require.ErrorAs(t, err, &tagUnknown)
	_, err = env.local.getDigestByTag(ctx, info)
	require.Equal(t, errcode.ErrCodeManifestUnknown, rpgCodes([]error{err})[0])

	info.Tag = "other"
	_, err = env.local.getDigestByTag(ctx, info)
	require.ErrorIs(t, err, errRpgBoom)

	info.Tag = ""
	info.Digest = "sha256:abc"
	dgst, err = env.local.getDigest(ctx, info)
	require.NoError(t, err)
	require.Equal(t, digest.Digest("sha256:abc"), dgst)
}

func TestRpgManifestExist(t *testing.T) {
	ctx := context.Background()
	_, dockerPayload := rpgDockerManifest(t, "docker")
	_, ociPayload := rpgOCIManifest(t, "oci", nil)
	_, childPayload := rpgDockerManifest(t, "child")
	childDigest := rpgPayloadDigest(childPayload)
	_, listPayload := rpgIndex(t, rpgListMT, childDigest)
	_, ociIndexPayload := rpgIndex(t, v1.MediaTypeImageIndex, childDigest)

	setup := func(t *testing.T) (*rpgEnv, pkg.RegistryInfo) {
		env := rpgNewEnv(t)
		env.addManifest(t, 1, rpgDockerMT, dockerPayload)
		env.addManifest(t, 2, v1.MediaTypeImageManifest, ociPayload)
		env.addManifest(t, 3, rpgListMT, listPayload)
		env.addManifest(t, 4, v1.MediaTypeImageIndex, ociIndexPayload)
		env.addManifest(t, 5, rpgDockerMT, childPayload)
		return env, rpgInfo(t)
	}

	t.Run("docker by digest", func(t *testing.T) {
		env, info := setup(t)
		info.Digest = rpgPayloadDigest(dockerPayload).String()
		headers, desc, m, errs := env.local.ManifestExist(ctx, info, []string{rpgDockerMT}, nil)
		require.Empty(t, errs)
		require.NotNil(t, m)
		require.Equal(t, rpgPayloadDigest(dockerPayload), desc.Digest)
		require.Equal(t, rpgDockerMT, headers.Headers["Content-Type"])
		require.Equal(t, `"`+desc.Digest.String()+`"`, headers.Headers["Etag"])
	})
	t.Run("not modified", func(t *testing.T) {
		env, info := setup(t)
		info.Digest = rpgPayloadDigest(dockerPayload).String()
		headers, _, m, errs := env.local.ManifestExist(ctx, info, nil, []string{info.Digest})
		require.Empty(t, errs)
		require.Nil(t, m)
		require.Equal(t, http.StatusNotModified, headers.Code)
	})
	t.Run("unknown tag", func(t *testing.T) {
		env, info := setup(t)
		env.manifests.byTag = func(int64, string, string) (*registrytypes.Manifest, error) {
			return nil, gitnessstore.ErrResourceNotFound
		}
		info.Tag = "nope"
		_, _, _, errs := env.local.ManifestExist(ctx, info, nil, nil)
		require.Equal(t, []errcode.CodeError{errcode.ErrCodeManifestUnknown}, rpgCodes(errs))
	})
	t.Run("unknown digest", func(t *testing.T) {
		env, info := setup(t)
		info.Digest = rpgDigest("missing").String()
		_, _, _, errs := env.local.ManifestExist(ctx, info, nil, nil)
		require.Equal(t, []errcode.CodeError{errcode.ErrCodeManifestUnknown}, rpgCodes(errs))
	})
	t.Run("soft deleted artifact", func(t *testing.T) {
		env, info := setup(t)
		env.artifacts.getErr = gitnessstore.ErrResourceNotFound
		info.Digest = rpgPayloadDigest(dockerPayload).String()
		_, _, m, errs := env.local.ManifestExist(ctx, info, []string{rpgDockerMT}, nil)
		require.Nil(t, m)
		require.Equal(t, []errcode.CodeError{errcode.ErrCodeManifestUnknown}, rpgCodes(errs))
	})
	t.Run("bad digest cannot be converted", func(t *testing.T) {
		env, info := setup(t)
		env.manifests.byDigest = func(int64, string, registrytypes.Digest) (*registrytypes.Manifest, error) {
			return &registrytypes.Manifest{SchemaVersion: 2, MediaType: rpgDockerMT, Payload: dockerPayload}, nil
		}
		info.Digest = "weird:abc"
		_, _, _, errs := env.local.ManifestExist(ctx, info, nil, nil)
		require.NotEmpty(t, errs)
	})
	t.Run("oci manifest needs accept", func(t *testing.T) {
		env, info := setup(t)
		info.Digest = rpgPayloadDigest(ociPayload).String()
		_, _, _, errs := env.local.ManifestExist(ctx, info, []string{rpgDockerMT}, nil)
		require.Equal(t, []errcode.CodeError{errcode.ErrCodeManifestUnknown}, rpgCodes(errs))
		_, _, m, errs := env.local.ManifestExist(ctx, info, []string{v1.MediaTypeImageManifest}, nil)
		require.Empty(t, errs)
		require.NotNil(t, m)
	})
	t.Run("oci index needs accept", func(t *testing.T) {
		env, info := setup(t)
		info.Digest = rpgPayloadDigest(ociIndexPayload).String()
		_, _, _, errs := env.local.ManifestExist(ctx, info, []string{rpgDockerMT}, nil)
		require.Equal(t, []errcode.CodeError{errcode.ErrCodeManifestUnknown}, rpgCodes(errs))
		_, _, m, errs := env.local.ManifestExist(ctx, info, []string{v1.MediaTypeImageIndex}, nil)
		require.Empty(t, errs)
		require.NotNil(t, m)
	})
	t.Run("manifest list rewritten for old clients", func(t *testing.T) {
		env, info := setup(t)
		env.manifests.byTag = func(int64, string, string) (*registrytypes.Manifest, error) {
			return env.rows[rpgStoredDigest(t, rpgPayloadDigest(listPayload))], nil
		}
		info.Tag = rpgLatestTag
		_, desc, m, errs := env.local.ManifestExist(ctx, info, []string{rpgDockerMT}, nil)
		require.Empty(t, errs)
		require.NotNil(t, m)
		require.Equal(t, childDigest, desc.Digest)

		// Clients accepting the list media type get the list itself.
		_, desc, _, errs = env.local.ManifestExist(ctx, info, []string{rpgListMT}, nil)
		require.Empty(t, errs)
		require.Equal(t, rpgPayloadDigest(listPayload), desc.Digest)

		// Clients that accept neither cannot be served the child.
		_, _, _, errs = env.local.ManifestExist(ctx, info, []string{v1.MediaTypeImageManifest}, nil)
		require.Equal(t, []errcode.CodeError{errcode.ErrCodeManifestInvalid}, rpgCodes(errs))
	})
}

func TestRpgRewriteManifest(t *testing.T) {
	ctx := context.Background()
	env := rpgNewEnv(t)
	info := rpgInfo(t)
	_, childPayload := rpgDockerManifest(t, "child")
	_, listPayload := rpgIndex(t, rpgListMT, rpgPayloadDigest(childPayload))

	list, err := DBManifestToManifest(
		&registrytypes.Manifest{SchemaVersion: 2, MediaType: rpgListMT, Payload: listPayload},
	)
	require.NoError(t, err)
	require.NotNil(t, list)

	// The child is not stored, so the rewrite fails with an unknown manifest.
	typed := rpgManifestList(t, listPayload)
	_, _, err = env.local.rewriteManifest(ctx, info, rpgDigest("list"), typed, [4]bool{true})
	require.Equal(t, errcode.ErrCodeManifestUnknown, rpgCodes([]error{err})[0])

	env.manifests.byDigest = func(int64, string, registrytypes.Digest) (*registrytypes.Manifest, error) {
		return nil, errRpgBoom
	}
	_, _, err = env.local.rewriteManifest(ctx, info, rpgDigest("list"), typed, [4]bool{true})
	require.ErrorIs(t, err, errRpgBoom)

	// A list without an amd64/linux entry cannot be rewritten.
	noDefault := rpgManifestList(t, []byte(`{"schemaVersion":2,"mediaType":"`+rpgListMT+`","manifests":[{`+
		`"mediaType":"`+rpgDockerMT+`","size":1,"digest":"`+rpgDigest("x").String()+`",`+
		`"platform":{"architecture":"arm64","os":"linux"}}]}`))
	_, _, err = env.local.rewriteManifest(ctx, info, rpgDigest("list"), noDefault, [4]bool{true})
	require.Equal(t, errcode.ErrCodeManifestUnknown, rpgCodes([]error{err})[0])
}

func TestRpgPutManifest(t *testing.T) {
	ctx := rpgAuthCtx(context.Background())

	t.Run("docker manifest with tag", func(t *testing.T) {
		env := rpgNewEnv(t)
		env.events.done = make(chan struct{})
		mt, payload := rpgDockerManifest(t, "put")
		info := rpgInfo(t)
		info.Tag = "v1"
		headers, errs := env.local.PutManifest(ctx, info, mt, io.NopCloser(bytes.NewReader(payload)), int64(len(payload)))
		require.Empty(t, errs)
		require.Equal(t, http.StatusCreated, headers.Code)
		require.Equal(t, rpgPayloadDigest(payload).String(), headers.Headers["Docker-Content-Digest"])
		require.Contains(t, headers.Headers["Location"], "/manifests/")
		require.Len(t, env.tags.created, 1)
		require.Equal(t, "v1", env.tags.created[0].Name)
		require.Equal(t, 1, env.layers.calls)
		<-env.events.done
		require.Equal(t, 1, env.producer.count())
	})
	t.Run("oci manifest with digest and subject", func(t *testing.T) {
		env := rpgNewEnv(t)
		_, subjectPayload := rpgDockerManifest(t, "subject")
		subject := rpgPayloadDigest(subjectPayload)
		env.addManifest(t, 8, rpgDockerMT, subjectPayload)
		mt, payload := rpgOCIManifest(t, "oci", &subject)
		info := rpgInfo(t)
		info.Digest = rpgPayloadDigest(payload).String()
		headers, errs := env.local.PutManifest(ctx, info, mt, io.NopCloser(bytes.NewReader(payload)), int64(len(payload)))
		require.Empty(t, errs)
		require.Equal(t, subject.String(), headers.Headers["OCI-Subject"])
		stored := env.rows[rpgStoredDigest(t, rpgPayloadDigest(payload))]
		require.NotNil(t, stored)
		require.True(t, stored.SubjectID.Valid)
		require.Equal(t, "application/vnd.example.sbom", stored.ArtifactType.String)
	})
	t.Run("existing manifest is not stored again", func(t *testing.T) {
		env := rpgNewEnv(t)
		mt, payload := rpgDockerManifest(t, "again")
		env.addManifest(t, 3, mt, payload)
		info := rpgInfo(t)
		info.Digest = rpgPayloadDigest(payload).String()
		_, errs := env.local.PutManifest(ctx, info, mt, io.NopCloser(bytes.NewReader(payload)), int64(len(payload)))
		require.Empty(t, errs)
		require.Zero(t, env.layers.calls)
	})
	t.Run("untagged images mode", func(t *testing.T) {
		env := rpgNewEnv(t)
		env.untagged = true
		env.events.done = make(chan struct{})
		mt, payload := rpgDockerManifest(t, "untagged")
		info := rpgInfo(t)
		info.Digest = rpgPayloadDigest(payload).String()
		_, errs := env.local.PutManifest(ctx, info, mt, io.NopCloser(bytes.NewReader(payload)), int64(len(payload)))
		require.Empty(t, errs)
		<-env.events.done
	})
	t.Run("invalid payload", func(t *testing.T) {
		env := rpgNewEnv(t)
		info := rpgInfo(t)
		info.Tag = "v1"
		_, errs := env.local.PutManifest(ctx, info, rpgDockerMT, io.NopCloser(strings.NewReader("{")), 1)
		require.Equal(t, []errcode.CodeError{errcode.ErrCodeManifestInvalid}, rpgCodes(errs))
	})
	t.Run("body read failure", func(t *testing.T) {
		env := rpgNewEnv(t)
		_, errs := env.local.PutManifest(ctx, rpgInfo(t), rpgDockerMT, io.NopCloser(&rpgFailingReader{}), 1)
		require.Equal(t, []errcode.CodeError{errcode.ErrCodeManifestInvalid}, rpgCodes(errs))
	})
	t.Run("digest mismatch", func(t *testing.T) {
		env := rpgNewEnv(t)
		mt, payload := rpgDockerManifest(t, "mismatch")
		info := rpgInfo(t)
		info.Digest = rpgDigest("other").String()
		_, errs := env.local.PutManifest(ctx, info, mt, io.NopCloser(bytes.NewReader(payload)), int64(len(payload)))
		require.Equal(t, []errcode.CodeError{errcode.ErrCodeDigestInvalid}, rpgCodes(errs))
	})
	t.Run("neither tag nor digest", func(t *testing.T) {
		env := rpgNewEnv(t)
		mt, payload := rpgDockerManifest(t, "none")
		_, errs := env.local.PutManifest(ctx, rpgInfo(t), mt, io.NopCloser(bytes.NewReader(payload)), int64(len(payload)))
		require.Equal(t, []errcode.CodeError{errcode.ErrCodeTagInvalid}, rpgCodes(errs))
	})
	t.Run("config blob missing", func(t *testing.T) {
		env := rpgNewEnv(t)
		env.blobs.byRepo = func(digest.Digest) (*registrytypes.Blob, error) { return nil, gitnessstore.ErrResourceNotFound }
		mt, payload := rpgDockerManifest(t, "noblob")
		info := rpgInfo(t)
		info.Tag = "v1"
		_, errs := env.local.PutManifest(ctx, info, mt, io.NopCloser(bytes.NewReader(payload)), int64(len(payload)))
		require.Equal(t, []errcode.CodeError{errcode.ErrCodeUnknown}, rpgCodes(errs))
	})
	t.Run("tagging fails", func(t *testing.T) {
		env := rpgNewEnv(t)
		env.tags.createErr = errRpgBoom
		mt, payload := rpgDockerManifest(t, "tagfail")
		info := rpgInfo(t)
		info.Tag = "v1"
		_, errs := env.local.PutManifest(ctx, info, mt, io.NopCloser(bytes.NewReader(payload)), int64(len(payload)))
		require.Equal(t, []errcode.CodeError{errcode.ErrCodeUnknown}, rpgCodes(errs))
	})
}

func TestRpgDeleteManifest(t *testing.T) {
	ctx := rpgAuthCtx(context.Background())
	_, payload := rpgDockerManifest(t, "del")
	dgst := rpgPayloadDigest(payload)

	t.Run("by tag", func(t *testing.T) {
		env := rpgNewEnv(t)
		env.tags.findTag = nil
		env.tags.findErr = gitnessstore.ErrResourceNotFound
		info := rpgInfo(t)
		info.Tag = "v1"
		errs, headers := env.local.DeleteManifest(ctx, info)
		require.Empty(t, errs)
		require.Equal(t, http.StatusAccepted, headers.Code)
	})
	t.Run("by tag with event", func(t *testing.T) {
		env := rpgNewEnv(t)
		row := env.addManifest(t, 1, rpgDockerMT, payload)
		env.tags.findTag = &registrytypes.Tag{ManifestID: row.ID}
		info := rpgInfo(t)
		info.Tag = "v1"
		errs, _ := env.local.DeleteManifest(ctx, info)
		require.Empty(t, errs)
		require.Equal(t, 1, env.producer.count())
	})
	t.Run("unknown tag", func(t *testing.T) {
		env := rpgNewEnv(t)
		env.tags.deleted = false
		info := rpgInfo(t)
		info.Tag = "v1"
		errs, _ := env.local.DeleteManifest(ctx, info)
		var unknown distribution.ErrTagUnknown
		require.ErrorAs(t, errs[0], &unknown)
	})
	t.Run("by digest", func(t *testing.T) {
		env := rpgNewEnv(t)
		env.addManifest(t, 1, rpgDockerMT, payload)
		env.manifests.referencedBy = func(*registrytypes.Manifest) (registrytypes.Manifests, error) { return nil, nil }
		env.manifests.deleteMf = func(digest.Digest) (bool, error) { return true, nil }
		env.manifests.count = func() (int64, error) { return 0, nil }
		info := rpgInfo(t)
		info.Digest = dgst.String()
		errs, headers := env.local.DeleteManifest(ctx, info)
		require.Empty(t, errs)
		require.Equal(t, http.StatusAccepted, headers.Code)
	})
	t.Run("error mapping", func(t *testing.T) {
		tests := []struct {
			name  string
			setup func(env *rpgEnv)
			want  []errcode.CodeError
		}{
			{"not found", func(env *rpgEnv) { env.rows = map[registrytypes.Digest]*registrytypes.Manifest{} },
				[]errcode.CodeError{errcode.ErrCodeManifestUnknown}},
			{"referenced in list", func(env *rpgEnv) {
				env.addManifest(t, 1, rpgDockerMT, payload)
				env.manifests.referencedBy = func(*registrytypes.Manifest) (registrytypes.Manifests, error) {
					return registrytypes.Manifests{{Digest: rpgDigest("parent")}}, nil
				}
			}, []errcode.CodeError{errcode.ErrCodeUnknown}},
			{"unknown", func(env *rpgEnv) {
				env.manifests.byDigest = func(int64, string, registrytypes.Digest) (*registrytypes.Manifest, error) {
					return nil, errRpgBoom
				}
			}, []errcode.CodeError{errcode.ErrCodeUnknown}},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				env := rpgNewEnv(t)
				tt.setup(env)
				info := rpgInfo(t)
				info.Digest = dgst.String()
				errs, headers := env.local.DeleteManifest(ctx, info)
				require.Equal(t, tt.want, rpgCodes(errs))
				require.Zero(t, headers.Code)
			})
		}
	})
}

func TestRpgDeleteManifestErrorCodes(t *testing.T) {
	ctx := context.Background()
	info := rpgInfo(t)
	info.Digest = rpgDigest("x").String()
	tests := []struct {
		err  error
		want []errcode.CodeError
	}{
		{digest.ErrDigestUnsupported, nil},
		{digest.ErrDigestInvalidFormat, []errcode.CodeError{errcode.ErrCodeDigestInvalid}},
		{manifest.ErrUnsupported, []errcode.CodeError{errcode.ErrCodeUnsupported}},
		{util.ErrManifestNotFound, []errcode.CodeError{errcode.ErrCodeManifestUnknown}},
		{util.ErrManifestReferencedInList, []errcode.CodeError{errcode.ErrCodeManifestReferencedInList}},
		{errRpgBoom, []errcode.CodeError{errcode.ErrCodeUnknown}},
	}
	for _, tt := range tests {
		t.Run(tt.err.Error(), func(t *testing.T) {
			env := rpgNewEnv(t)
			env.local.ms = &rpgFakeMS{deleteErr: tt.err}
			errs, _ := env.local.DeleteManifest(ctx, info)
			require.Equal(t, append([]errcode.CodeError{}, tt.want...), append([]errcode.CodeError{}, rpgCodes(errs)...))
		})
	}
}

func TestRpgListTags(t *testing.T) {
	ctx := context.Background()
	env := rpgNewEnv(t)
	info := rpgInfo(t)

	env.tags.tags = []*registrytypes.Tag{{Name: "a"}, {Name: "b"}}
	env.tags.hasMore = true
	headers, tags, err := env.local.ListTags(ctx, "", 2, "http://localhost/v2/root/reg/img/tags/list?n=2", info)
	require.NoError(t, err)
	require.Equal(t, []string{"a", "b"}, tags)
	require.Contains(t, headers.Headers["Link"], "last=b")

	env.tags.hasMore = false
	headers, tags, err = env.local.ListTags(ctx, "", 2, "http://localhost/x", info)
	require.NoError(t, err)
	require.Equal(t, []string{"a", "b"}, tags)
	require.Empty(t, headers.Headers["Link"])

	env.tags.tags = nil
	_, tags, err = env.local.ListTags(ctx, "", 2, "http://localhost/x", info)
	require.NoError(t, err)
	require.Nil(t, tags)

	env.tags.tagsErr = errRpgBoom
	_, _, err = env.local.ListTags(ctx, "", 2, "http://localhost/x", info)
	require.ErrorIs(t, err, errRpgBoom)

	env.tags.tagsErr = nil
	env.tags.tags = []*registrytypes.Tag{{Name: "a"}}
	env.tags.hasMoreErr = errRpgBoom
	_, _, err = env.local.ListTags(ctx, "", 2, "http://localhost/x", info)
	require.ErrorIs(t, err, errRpgBoom)

	env.tags.hasMoreErr = nil
	env.tags.hasMore = true
	_, _, err = env.local.ListTags(ctx, "", 2, "://bad", info)
	require.Error(t, err)
}

func TestRpgListReferrers(t *testing.T) {
	ctx := context.Background()
	env := rpgNewEnv(t)
	info := rpgInfo(t)
	info.Digest = rpgDigest("subject").String()

	withType := &registrytypes.Manifest{
		MediaType: v1.MediaTypeImageManifest, Digest: rpgDigest("a"), TotalSize: 4,
		ArtifactType: sql.NullString{String: "type/a", Valid: true},
	}
	withConfig := &registrytypes.Manifest{
		MediaType: v1.MediaTypeImageManifest, Digest: rpgDigest("b"),
		Configuration: &registrytypes.Configuration{MediaType: "type/b"},
	}
	env.manifests.bySubject = func(registrytypes.Digest) (registrytypes.Manifests, error) {
		return registrytypes.Manifests{withType, withConfig}, nil
	}

	index, headers, err := env.local.ListReferrers(ctx, info, "")
	require.NoError(t, err)
	require.Len(t, index.Manifests, 2)
	require.Equal(t, ReferrersMediaType, headers.Headers["Content-Type"])
	require.Empty(t, headers.Headers["OCI-Filters-Applied"])

	index, headers, err = env.local.ListReferrers(ctx, info, "type/b")
	require.NoError(t, err)
	require.Len(t, index.Manifests, 1)
	require.Equal(t, "type/b", index.Manifests[0].ArtifactType)
	require.Equal(t, "artifactType", headers.Headers["OCI-Filters-Applied"])

	env.manifests.bySubject = func(registrytypes.Digest) (registrytypes.Manifests, error) {
		return nil, gitnessstore.ErrResourceNotFound
	}
	index, _, err = env.local.ListReferrers(ctx, info, "")
	require.NoError(t, err)
	require.Empty(t, index.Manifests)

	env.manifests.bySubject = func(registrytypes.Digest) (registrytypes.Manifests, error) { return nil, errRpgBoom }
	_, _, err = env.local.ListReferrers(ctx, info, "")
	require.ErrorIs(t, err, errRpgBoom)

	info.Digest = "weird:abc"
	_, _, err = env.local.ListReferrers(ctx, info, "")
	require.Error(t, err)
}
