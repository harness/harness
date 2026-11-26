// Copyright 2023 Harness, Inc.
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

package proxy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/harness/gitness/app/api/request"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/services/refcache"
	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	liberrors "github.com/harness/gitness/registry/app/common/lib/errors"
	"github.com/harness/gitness/registry/app/manifest"
	"github.com/harness/gitness/registry/app/pkg"
	"github.com/harness/gitness/registry/app/pkg/commons"
	"github.com/harness/gitness/registry/app/storage"
	"github.com/harness/gitness/registry/types"

	"github.com/opencontainers/go-digest"
	"github.com/stretchr/testify/require"
)

const (
	ra5Image     = "library/app"
	ra5Repo      = "proxyrepo"
	ra5MediaType = "application/vnd.docker.distribution.manifest.v2+json"
)

var errRa5Boom = errors.New("boom")

type ra5Manifest struct {
	mediaType string
	payload   []byte
	err       error
	refs      []manifest.Descriptor
}

func (m ra5Manifest) References() []manifest.Descriptor { return m.refs }

func (m ra5Manifest) Payload() (string, []byte, error) { return m.mediaType, m.payload, m.err }

type ra5Local struct {
	pullHeaders *commons.ResponseHeaders
	pullDesc    manifest.Descriptor
	pullMan     manifest.Manifest
	pullErrs    []error
	putErrs     []error
	blobErrs    []error
	initHeaders *commons.ResponseHeaders
	initErrs    []error
	deleted     chan struct{}
	pushed      chan string
	puts        int
}

func (l *ra5Local) Base() error { return nil }

func (l *ra5Local) PullManifest(
	_ context.Context, _ pkg.RegistryInfo, _ []string, _ []string,
) (*commons.ResponseHeaders, manifest.Descriptor, manifest.Manifest, []error) {
	return l.pullHeaders, l.pullDesc, l.pullMan, l.pullErrs
}

func (l *ra5Local) PutManifest(
	_ context.Context, _ pkg.RegistryInfo, _ string, _ io.ReadCloser, _ int64,
) (*commons.ResponseHeaders, []error) {
	l.puts++
	return nil, l.putErrs
}

func (l *ra5Local) DeleteManifest(
	_ context.Context, _ pkg.RegistryInfo,
) ([]error, *commons.ResponseHeaders) {
	if l.deleted != nil {
		l.deleted <- struct{}{}
	}
	return nil, nil
}

func (l *ra5Local) GetBlob(
	_ context.Context, _ pkg.RegistryInfo,
) (*commons.ResponseHeaders, *storage.FileReader, int64, io.ReadCloser, string, []error) {
	return nil, nil, 0, nil, "", l.blobErrs
}

func (l *ra5Local) InitBlobUpload(
	_ context.Context, _ pkg.RegistryInfo, _, _ string,
) (*commons.ResponseHeaders, []error) {
	return l.initHeaders, l.initErrs
}

func (l *ra5Local) PushBlob(
	_ context.Context, art pkg.RegistryInfo, body io.ReadCloser, _ int64, state string,
) (*commons.ResponseHeaders, []error) {
	data, _ := io.ReadAll(body)
	if l.pushed != nil {
		l.pushed <- art.Reference + "|" + state + "|" + string(data)
	}
	return nil, nil
}

type ra5LocalManifests struct {
	tagErr   error
	assocErr error
}

func (l ra5LocalManifests) DBTag(
	_ context.Context, _ manifest.Manifest, _ digest.Digest, _ string, _ *commons.ResponseHeaders, _ pkg.RegistryInfo,
) error {
	return l.tagErr
}

func (l ra5LocalManifests) AddManifestAssociation(
	_ context.Context, _ string, _ digest.Digest, _ pkg.RegistryInfo,
) error {
	return l.assocErr
}

type ra5Remote struct {
	man       manifest.Manifest
	manDigest string
	manErr    error
	exist     bool
	desc      *manifest.Descriptor
	existErr  error
	blob      string
	blobErr   error
}

func (r ra5Remote) BlobReader(_ context.Context, _, _ string) (int64, io.ReadCloser, error) {
	return int64(len(r.blob)), io.NopCloser(strings.NewReader(r.blob)), r.blobErr
}

func (r ra5Remote) Manifest(_ context.Context, _, _ string) (manifest.Manifest, string, error) {
	return r.man, r.manDigest, r.manErr
}

func (r ra5Remote) ManifestExist(_ context.Context, _, _ string) (bool, *manifest.Descriptor, error) {
	return r.exist, r.desc, r.existErr
}

func (r ra5Remote) ListTags(_ context.Context, _ string) ([]string, error) { return nil, nil }

func (r ra5Remote) GetImageName(_ context.Context, _ refcache.SpaceFinder, name string) (string, error) {
	return name, nil
}

func ra5Info() pkg.RegistryInfo {
	return pkg.RegistryInfo{
		ArtifactInfo: &pkg.ArtifactInfo{RegIdentifier: ra5Repo, Image: ra5Image},
		Tag:          "latest",
	}
}

func ra5NewController(l *ra5Local, lm ra5LocalManifests, handlers map[string]ManifestCacheHandler) *controller {
	return NewProxyController(l, lm, nil, refcache.SpaceFinder{}, handlers).(*controller) //nolint:errcheck,forcetypeassert
}

func TestRa5Reference(t *testing.T) {
	info := ra5Info()
	require.Equal(t, ra5Image, getRemoteRepo(info))
	require.Equal(t, "latest", getReference(info))
	info.Digest = "sha256:abc"
	require.Equal(t, "sha256:abc", getReference(info))

	rc := ByteToReadCloser([]byte("data"))
	out, err := io.ReadAll(rc)
	require.NoError(t, err)
	require.Equal(t, "data", string(out))
	require.NotNil(t, GetManifestCache(&ra5Local{}, ra5LocalManifests{}))
	require.NotNil(t, GetManifestListCache(&ra5Local{}))
}

func TestRa5EnsureTag(t *testing.T) {
	ctx := context.Background()
	man := ra5Manifest{mediaType: ra5MediaType}

	c := ra5NewController(&ra5Local{pullErrs: []error{errRa5Boom}}, ra5LocalManifests{}, nil)
	require.ErrorIs(t, c.EnsureTag(ctx, nil, ra5Info(), nil, nil), errRa5Boom)

	c = ra5NewController(&ra5Local{pullMan: man}, ra5LocalManifests{tagErr: errRa5Boom}, nil)
	require.ErrorIs(t, c.EnsureTag(ctx, nil, ra5Info(), nil, nil), errRa5Boom)

	c = ra5NewController(&ra5Local{pullMan: man}, ra5LocalManifests{}, nil)
	require.NoError(t, c.EnsureTag(ctx, nil, ra5Info(), nil, nil))
}

func TestRa5UseLocalBlob(t *testing.T) {
	ctx := context.Background()
	info := ra5Info()
	c := ra5NewController(&ra5Local{}, ra5LocalManifests{}, nil)
	require.False(t, c.UseLocalBlob(ctx, info))

	info.Digest = "sha256:abc"
	require.True(t, c.UseLocalBlob(ctx, info))

	c = ra5NewController(&ra5Local{blobErrs: []error{errRa5Boom}}, ra5LocalManifests{}, nil)
	require.False(t, c.UseLocalBlob(ctx, info))
}

func TestRa5UseLocalManifest(t *testing.T) {
	ctx := context.Background()
	const localDigest = "sha256:local"
	localMan := ra5Manifest{mediaType: ra5MediaType, payload: []byte("payload")}
	rateLimited := liberrors.New("slow down").WithCode(liberrors.RateLimitCode)
	desc := func(d string) *manifest.Descriptor { return &manifest.Descriptor{Digest: digest.Digest(d), Size: 7} }
	withLocal := func() *ra5Local {
		return &ra5Local{pullDesc: manifest.Descriptor{Digest: localDigest}, pullMan: localMan}
	}

	t.Run("not found locally", func(t *testing.T) {
		c := ra5NewController(&ra5Local{pullErrs: []error{errRa5Boom}}, ra5LocalManifests{}, nil)
		use, list, err := c.UseLocalManifest(ctx, ra5Info(), ra5Remote{}, nil, nil)
		require.NoError(t, err)
		require.False(t, use)
		require.Nil(t, list)
	})

	t.Run("rate limited with local copy", func(t *testing.T) {
		c := ra5NewController(withLocal(), ra5LocalManifests{}, nil)
		use, list, err := c.UseLocalManifest(ctx, ra5Info(), ra5Remote{existErr: rateLimited}, nil, nil)
		require.NoError(t, err)
		require.True(t, use)
		require.Equal(t, localDigest, list.Digest)
		require.Equal(t, ra5MediaType, list.ContentType)
		require.Equal(t, "payload", string(list.Content))
	})

	t.Run("rate limited without local digest", func(t *testing.T) {
		c := ra5NewController(&ra5Local{pullMan: localMan}, ra5LocalManifests{}, nil)
		use, list, err := c.UseLocalManifest(ctx, ra5Info(), ra5Remote{existErr: rateLimited}, nil, nil)
		require.NoError(t, err)
		require.True(t, use)
		require.Nil(t, list)
	})

	t.Run("remote error", func(t *testing.T) {
		c := ra5NewController(withLocal(), ra5LocalManifests{}, nil)
		use, _, err := c.UseLocalManifest(ctx, ra5Info(), ra5Remote{existErr: errRa5Boom}, nil, nil)
		require.ErrorIs(t, err, errRa5Boom)
		require.False(t, use)
	})

	t.Run("gone upstream", func(t *testing.T) {
		for name, remote := range map[string]ra5Remote{
			"not exist": {exist: false, desc: desc("sha256:x")},
			"no desc":   {exist: true},
		} {
			t.Run(name, func(t *testing.T) {
				local := withLocal()
				local.deleted = make(chan struct{}, 1)
				c := ra5NewController(local, ra5LocalManifests{}, nil)
				use, _, err := c.UseLocalManifest(ctx, ra5Info(), remote, nil, nil)
				require.True(t, liberrors.IsNotFoundErr(err))
				require.False(t, use)
				select {
				case <-local.deleted:
				case <-time.After(5 * time.Second):
					t.Fatal("local manifest was not deleted")
				}
			})
		}
	})

	t.Run("digest changed", func(t *testing.T) {
		c := ra5NewController(withLocal(), ra5LocalManifests{}, nil)
		use, list, err := c.UseLocalManifest(ctx, ra5Info(), ra5Remote{exist: true, desc: desc("sha256:new")}, nil, nil)
		require.NoError(t, err)
		require.False(t, use)
		require.Nil(t, list)
	})

	t.Run("up to date", func(t *testing.T) {
		c := ra5NewController(withLocal(), ra5LocalManifests{}, nil)
		use, list, err := c.UseLocalManifest(ctx, ra5Info(), ra5Remote{exist: true, desc: desc(localDigest)}, nil, nil)
		require.NoError(t, err)
		require.True(t, use)
		require.Equal(t, localDigest, list.Digest)
	})
}

func TestRa5HeadManifest(t *testing.T) {
	c := ra5NewController(&ra5Local{}, ra5LocalManifests{}, nil)
	d := &manifest.Descriptor{Digest: "sha256:abc"}
	exist, got, err := c.HeadManifest(context.Background(), ra5Info(), ra5Remote{exist: true, desc: d})
	require.NoError(t, err)
	require.True(t, exist)
	require.Equal(t, d, got)
}

func TestRa5ProxyManifest(t *testing.T) {
	ctx := context.Background()
	c := ra5NewController(&ra5Local{pullErrs: []error{errRa5Boom}}, ra5LocalManifests{}, nil)

	_, err := c.ProxyManifest(ctx, ra5Info(), ra5Remote{manErr: liberrors.NotFoundError(errRa5Boom)},
		ra5Repo, ra5Image, nil, nil)
	require.True(t, liberrors.IsNotFoundErr(err))

	_, err = c.ProxyManifest(ctx, ra5Info(), ra5Remote{manErr: errRa5Boom}, ra5Repo, ra5Image, nil, nil)
	require.ErrorIs(t, err, errRa5Boom)

	_, err = c.ProxyManifest(ctx, ra5Info(), ra5Remote{man: ra5Manifest{err: errRa5Boom}},
		ra5Repo, ra5Image, nil, nil)
	require.ErrorIs(t, err, errRa5Boom)

	// The background cache goroutine only starts acting after a delay; the call itself returns immediately.
	man := ra5Manifest{mediaType: ra5MediaType, payload: []byte("{}")}
	got, err := c.ProxyManifest(ctx, ra5Info(), ra5Remote{man: man, manDigest: "sha256:abc"},
		ra5Repo, ra5Image, nil, nil)
	require.NoError(t, err)
	require.Equal(t, man, got)
}

func TestRa5PutBlobToLocal(t *testing.T) {
	ctx := context.Background()
	desc := manifest.Descriptor{Digest: "sha256:abc"}
	location := "/v2/upload?_state=tok"
	headers := &commons.ResponseHeaders{Headers: map[string]string{
		"Location": location, "Docker-Upload-UUID": "uuid-1",
	}}

	local := &ra5Local{initHeaders: headers, pushed: make(chan string, 1)}
	c := ra5NewController(local, ra5LocalManifests{}, nil)
	require.NoError(t, c.putBlobToLocal(ctx, ra5Info(), ra5Image, ra5Repo, desc, ra5Remote{blob: "blobdata"}))
	require.Equal(t, "uuid-1|tok|blobdata", <-local.pushed)

	require.ErrorIs(t, c.putBlobToLocal(ctx, ra5Info(), ra5Image, ra5Repo, desc, ra5Remote{blobErr: errRa5Boom}),
		errRa5Boom)

	c = ra5NewController(&ra5Local{initErrs: []error{errRa5Boom}}, ra5LocalManifests{}, nil)
	require.ErrorIs(t, c.putBlobToLocal(ctx, ra5Info(), ra5Image, ra5Repo, desc, ra5Remote{}), errRa5Boom)

	bad := &commons.ResponseHeaders{Headers: map[string]string{"Location": "%zz", "Docker-Upload-UUID": "u"}}
	c = ra5NewController(&ra5Local{initHeaders: bad}, ra5LocalManifests{}, nil)
	require.Error(t, c.putBlobToLocal(ctx, ra5Info(), ra5Image, ra5Repo, desc, ra5Remote{}))
}

type ra5CacheHandler struct {
	err    error
	called int
}

func (h *ra5CacheHandler) CacheContent(_ context.Context, _ pkg.RegistryInfo, _ string, _ manifest.Manifest) error {
	h.called++
	return h.err
}

func TestRa5WaitAndPushManifest(t *testing.T) {
	ctx := context.Background()
	man := ra5Manifest{}

	c := ra5NewController(&ra5Local{}, ra5LocalManifests{}, nil)
	require.ErrorContains(t, c.waitAndPushManifest(ctx, ra5Info(), ra5MediaType, man), "default manifest cache handler")

	specific, fallback := &ra5CacheHandler{}, &ra5CacheHandler{}
	c = ra5NewController(&ra5Local{}, ra5LocalManifests{}, map[string]ManifestCacheHandler{
		ra5MediaType: specific, DefaultHandler: fallback,
	})
	require.NoError(t, c.waitAndPushManifest(ctx, ra5Info(), ra5MediaType, man))
	require.NoError(t, c.waitAndPushManifest(ctx, ra5Info(), "other/type", man))
	require.Equal(t, 1, specific.called)
	require.Equal(t, 1, fallback.called)

	fallback.err = errRa5Boom
	require.ErrorIs(t, c.waitAndPushManifest(ctx, ra5Info(), "other/type", man), errRa5Boom)
}

func TestRa5ManifestCache(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("payload error", func(t *testing.T) {
		t.Parallel()
		mc := GetManifestCache(&ra5Local{}, ra5LocalManifests{})
		require.ErrorIs(t, mc.CacheContent(ctx, ra5Info(), ra5MediaType, ra5Manifest{err: errRa5Boom}), errRa5Boom)
	})

	t.Run("put error", func(t *testing.T) {
		t.Parallel()
		mc := GetManifestCache(&ra5Local{putErrs: []error{errRa5Boom}}, ra5LocalManifests{})
		require.ErrorIs(t, mc.CacheContent(ctx, ra5Info(), ra5MediaType, ra5Manifest{payload: []byte("x")}), errRa5Boom)
	})

	t.Run("success", func(t *testing.T) {
		// The handler waits for the manifest mapping to settle, so this takes a fixed delay.
		t.Parallel()
		local := &ra5Local{}
		mc := GetManifestCache(local, ra5LocalManifests{})
		require.NoError(t, mc.CacheContent(ctx, ra5Info(), ra5MediaType, ra5Manifest{payload: []byte("x")}))
		require.Equal(t, 1, local.puts)
	})
}

func TestRa5ManifestListCache(t *testing.T) {
	ctx := context.Background()
	refs := []manifest.Descriptor{{Digest: "sha256:abc"}}
	payload := []byte(`{"manifests":[]}`)
	man := ra5Manifest{mediaType: ra5MediaType, payload: payload, refs: refs}

	t.Run("payload error", func(t *testing.T) {
		lc := GetManifestListCache(&ra5Local{})
		require.ErrorIs(t, lc.CacheContent(ctx, ra5Info(), ra5MediaType, ra5Manifest{err: errRa5Boom}), errRa5Boom)
	})

	t.Run("empty reference", func(t *testing.T) {
		lc := GetManifestListCache(&ra5Local{})
		info := ra5Info()
		info.Tag = ""
		require.ErrorContains(t, lc.CacheContent(ctx, info, ra5MediaType, man), "reference is empty")
	})

	t.Run("no references", func(t *testing.T) {
		lc := GetManifestListCache(&ra5Local{})
		err := lc.CacheContent(ctx, ra5Info(), ra5MediaType, ra5Manifest{payload: payload})
		require.ErrorContains(t, err, "doesn't contain any pushed manifest")
	})

	t.Run("already cached", func(t *testing.T) {
		local := &ra5Local{pullMan: man, pullDesc: manifest.Descriptor{Digest: digest.FromBytes(payload)}}
		require.NoError(t, GetManifestListCache(local).CacheContent(ctx, ra5Info(), ra5MediaType, man))
		require.Zero(t, local.puts)
	})

	t.Run("pushed", func(t *testing.T) {
		local := &ra5Local{}
		require.NoError(t, GetManifestListCache(local).CacheContent(ctx, ra5Info(), ra5MediaType, man))
		require.Equal(t, 1, local.puts)
	})

	t.Run("push error", func(t *testing.T) {
		local := &ra5Local{putErrs: []error{errRa5Boom}}
		require.ErrorIs(t, GetManifestListCache(local).CacheContent(ctx, ra5Info(), ra5MediaType, man), errRa5Boom)
	})
}

// ra5Upstream is a minimal docker registry (v2 API) used as an anonymous upstream.
func ra5Upstream(t *testing.T, blob []byte) *httptest.Server {
	t.Helper()
	cfg := []byte("{}")
	man, err := json.Marshal(map[string]any{
		"schemaVersion": 2,
		"mediaType":     ra5MediaType,
		"config": map[string]any{
			"mediaType": "application/vnd.docker.container.image.v1+json",
			"size":      len(cfg),
			"digest":    digest.FromBytes(cfg).String(),
		},
		"layers": []any{},
	})
	require.NoError(t, err)
	manDigest := digest.FromBytes(man).String()

	mux := http.NewServeMux()
	mux.HandleFunc("/v2/", func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/v2/":
			w.WriteHeader(http.StatusOK)
		case strings.HasSuffix(r.URL.Path, "/tags/list"):
			_, _ = w.Write([]byte(`{"name":"library/app","tags":["v1","v2"]}`))
		case strings.Contains(r.URL.Path, "/manifests/missing"):
			http.NotFound(w, r)
		case strings.Contains(r.URL.Path, "/manifests/"):
			w.Header().Set("Content-Type", ra5MediaType)
			w.Header().Set("Docker-Content-Digest", manDigest)
			w.Header().Set("Content-Length", fmt.Sprint(len(man)))
			if r.Method != http.MethodHead {
				_, _ = w.Write(man)
			}
		case strings.Contains(r.URL.Path, "/blobs/"+digest.FromBytes(blob).String()):
			w.Header().Set("Content-Length", fmt.Sprint(len(blob)))
			_, _ = w.Write(blob)
		default:
			http.NotFound(w, r)
		}
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func ra5Proxy(url, source string) types.UpstreamProxy {
	return types.UpstreamProxy{
		RepoKey: ra5Repo, RepoURL: url, Source: source, PackageType: artifact.PackageTypeDOCKER,
		RepoAuthType: string(artifact.AuthTypeAnonymous),
	}
}

func TestRa5RemoteHelper(t *testing.T) {
	ctx := context.Background()
	blob := []byte("layer-bytes")
	srv := ra5Upstream(t, blob)
	sf := refcache.SpaceFinder{}

	r, err := NewRemoteHelper(ctx, sf, nil, ra5Repo, ra5Proxy(srv.URL, string(artifact.UpstreamConfigSourceCustom)))
	require.NoError(t, err)

	tags, err := r.ListTags(ctx, ra5Image)
	require.NoError(t, err)
	require.Equal(t, []string{"v1", "v2"}, tags)

	exist, desc, err := r.ManifestExist(ctx, ra5Image, "latest")
	require.NoError(t, err)
	require.True(t, exist)
	require.Equal(t, ra5MediaType, desc.MediaType)
	require.NotEmpty(t, desc.Digest)

	exist, _, err = r.ManifestExist(ctx, ra5Image, "missing")
	require.NoError(t, err)
	require.False(t, exist)

	man, dig, err := r.Manifest(ctx, ra5Image, "latest")
	require.NoError(t, err)
	require.NotEmpty(t, dig)
	mt, _, err := man.Payload()
	require.NoError(t, err)
	require.Equal(t, ra5MediaType, mt)

	_, _, err = r.Manifest(ctx, ra5Image, "missing")
	require.Error(t, err)

	size, rc, err := r.BlobReader(ctx, ra5Image, digest.FromBytes(blob).String())
	require.NoError(t, err)
	got, err := io.ReadAll(rc)
	require.NoError(t, err)
	require.NoError(t, rc.Close())
	require.Equal(t, int64(len(blob)), size)
	require.Equal(t, blob, got)

	name, err := r.GetImageName(ctx, sf, "nginx")
	require.NoError(t, err)
	require.Equal(t, "nginx", name)
}

func TestRa5RemoteHelperDockerhubImageName(t *testing.T) {
	ctx := context.Background()
	r, err := NewRemoteHelper(ctx, refcache.SpaceFinder{}, nil, ra5Repo,
		ra5Proxy("", string(artifact.UpstreamConfigSourceDockerhub)))
	require.NoError(t, err)

	name, err := r.GetImageName(ctx, refcache.SpaceFinder{}, "nginx")
	require.NoError(t, err)
	require.Equal(t, "library/nginx", name)
	name, err = r.GetImageName(ctx, refcache.SpaceFinder{}, "org/tool")
	require.NoError(t, err)
	require.Equal(t, "org/tool", name)

	_, err = NewRemoteHelper(ctx, refcache.SpaceFinder{}, nil, ra5Repo, ra5Proxy("", "NoSuchAdapter"))
	require.Error(t, err)

	bad := &remoteHelper{upstreamProxy: types.UpstreamProxy{Source: "NoSuchAdapter"}}
	_, err = bad.GetImageName(ctx, refcache.SpaceFinder{}, "x")
	require.Error(t, err)
}

func TestRa5ProxyBlob(t *testing.T) {
	blob := []byte("layer-bytes")
	srv := ra5Upstream(t, blob)
	proxy := ra5Proxy(srv.URL, string(artifact.UpstreamConfigSourceCustom))
	info := ra5Info()
	info.Digest = digest.FromBytes(blob).String()

	t.Run("without session", func(t *testing.T) {
		c := ra5NewController(&ra5Local{}, ra5LocalManifests{}, nil)
		size, rc, err := c.ProxyBlob(context.Background(), info, ra5Repo, proxy)
		require.NoError(t, err)
		got, err := io.ReadAll(rc)
		require.NoError(t, err)
		require.NoError(t, rc.Close())
		require.Equal(t, int64(len(blob)), size)
		require.Equal(t, blob, got)
	})

	t.Run("with session caches locally", func(t *testing.T) {
		local := &ra5Local{
			initHeaders: &commons.ResponseHeaders{Headers: map[string]string{
				"Location": "/upload?_state=tok", "Docker-Upload-UUID": "uuid-2",
			}},
			pushed: make(chan string, 1),
		}
		c := ra5NewController(local, ra5LocalManifests{}, nil)
		ctx := request.WithAuthSession(context.Background(), &auth.Session{})
		_, rc, err := c.ProxyBlob(ctx, info, ra5Repo, proxy)
		require.NoError(t, err)
		_, err = io.ReadAll(rc)
		require.NoError(t, err)
		require.NoError(t, rc.Close())
		select {
		case msg := <-local.pushed:
			require.Equal(t, "uuid-2|tok|"+string(blob), msg)
		case <-time.After(10 * time.Second):
			t.Fatal("blob was not pushed to the local registry")
		}
	})

	t.Run("unknown blob", func(t *testing.T) {
		c := ra5NewController(&ra5Local{}, ra5LocalManifests{}, nil)
		missing := info
		missing.Digest = digest.FromString("other").String()
		_, _, err := c.ProxyBlob(context.Background(), missing, ra5Repo, proxy)
		require.Error(t, err)
	})

	t.Run("bad adapter", func(t *testing.T) {
		c := ra5NewController(&ra5Local{}, ra5LocalManifests{}, nil)
		_, _, err := c.ProxyBlob(context.Background(), info, ra5Repo, ra5Proxy(srv.URL, "NoSuchAdapter"))
		require.Error(t, err)
	})
}
