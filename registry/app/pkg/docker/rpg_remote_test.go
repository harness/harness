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
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/registry/app/manifest"
	"github.com/harness/gitness/registry/app/pkg"
	"github.com/harness/gitness/registry/app/pkg/commons"
	proxy2 "github.com/harness/gitness/registry/app/remote/controller/proxy"
	"github.com/harness/gitness/registry/app/store"
	registrytypes "github.com/harness/gitness/registry/types"

	"github.com/opencontainers/go-digest"
	"github.com/stretchr/testify/require"

	_ "github.com/harness/gitness/registry/app/remote/adapter/dockerhub" // Registers the adapter used by the proxy.
)

// rpgProxyCtl scripts the proxy controller used by the remote registry.
type rpgProxyCtl struct {
	proxy2.Controller
	useLocalBlob bool
	useLocal     bool
	local        *proxy2.ManifestList
	localErr     error
	headExists   bool
	headDesc     *manifest.Descriptor
	headErr      error
	proxied      manifest.Manifest
	proxyErr     error
	blobSize     int64
	blobErr      error
}

func (p *rpgProxyCtl) UseLocalBlob(context.Context, pkg.RegistryInfo) bool { return p.useLocalBlob }

func (p *rpgProxyCtl) UseLocalManifest(
	context.Context, pkg.RegistryInfo, proxy2.RemoteInterface, []string, []string,
) (bool, *proxy2.ManifestList, error) {
	return p.useLocal, p.local, p.localErr
}

func (p *rpgProxyCtl) ProxyBlob(
	context.Context, pkg.RegistryInfo, string, registrytypes.UpstreamProxy,
) (int64, io.ReadCloser, error) {
	return p.blobSize, io.NopCloser(strings.NewReader("blob")), p.blobErr
}

func (p *rpgProxyCtl) ProxyManifest(
	context.Context, pkg.RegistryInfo, proxy2.RemoteInterface, string, string, []string, []string,
) (manifest.Manifest, error) {
	return p.proxied, p.proxyErr
}

func (p *rpgProxyCtl) HeadManifest(
	context.Context, pkg.RegistryInfo, proxy2.RemoteInterface,
) (bool, *manifest.Descriptor, error) {
	return p.headExists, p.headDesc, p.headErr
}

type rpgUpstreamRepo struct {
	store.UpstreamProxyConfigRepository
	proxy *registrytypes.UpstreamProxy
	err   error
}

func (r *rpgUpstreamRepo) GetByRegistryIdentifier(
	context.Context, int64, string, ...registrytypes.QueryOption,
) (*registrytypes.UpstreamProxy, error) {
	return r.proxy, r.err
}

func rpgRemote(t *testing.T, env *rpgEnv, ctl *rpgProxyCtl, source string) *RemoteRegistry {
	t.Helper()
	repo := &rpgUpstreamRepo{proxy: &registrytypes.UpstreamProxy{
		Source: source, RepoURL: "http://127.0.0.1:1", PackageType: artifact.PackageTypeDOCKER,
		RepoAuthType: string(artifact.AuthTypeAnonymous),
	}}
	remote, ok := NewRemoteRegistry(env.local, env.app, repo, rpgSpaceFinder(nil), nil, ctl).(*RemoteRegistry)
	require.True(t, ok)
	return remote
}

func TestRpgRemoteRegistryAccessors(t *testing.T) {
	env := rpgNewEnv(t)
	remote := rpgRemote(t, env, &rpgProxyCtl{}, string(artifact.UpstreamConfigSourceCustom))
	require.Equal(t, artifact.RegistryTypeUPSTREAM, remote.GetArtifactType())
	require.Len(t, remote.GetPackageTypes(), 2)
	require.Len(t, remote.manifestCacheHandlerMap, 4)
	require.True(t, canProxy())

	ctx := context.Background()
	info := rpgInfo(t)
	body := io.NopCloser(strings.NewReader(""))
	for name, call := range map[string]func(){
		"base":          func() { _ = remote.Base() },
		"monolith":      func() { _ = remote.PushBlobMonolith(ctx, info, 0, nil) },
		"init upload":   func() { _, _ = remote.InitBlobUpload(ctx, info, "", "") },
		"with digest":   func() { _ = remote.PushBlobMonolithWithDigest(ctx, info, 0, nil) },
		"chunk":         func() { _, _ = remote.PushBlobChunk(nil, info, "", "", "", body, 0) },
		"push blob":     func() { _, _ = remote.PushBlob(ctx, info, body, 0, "") },
		"put manifest":  func() { _, _ = remote.PutManifest(ctx, info, "", body, 0) },
		"list tags":     func() { _, _, _ = remote.ListTags(ctx, "", 0, "", info) },
		"filtered tags": func() { _, _ = remote.ListFilteredTags(ctx, 0, "", "", info) },
		"del manifest":  func() { _, _ = remote.DeleteManifest(ctx, info) },
		"del blob":      func() { _, _ = remote.DeleteBlob(nil, info) },
		"mount":         func() { _ = remote.MountBlob(ctx, info, "", "") },
		"referrers":     func() { _, _, _ = remote.ListReferrers(ctx, info, "") },
		"upload status": func() { _, _ = remote.GetBlobUploadStatus(nil, info, "") },
		"catalog":       func() { _, _ = remote.GetCatalog() },
		"delete tag":    func() { _ = remote.DeleteTag("", "", info) },
		"pull chunk":    func() { _, _, _ = remote.PullBlobChunk("", "", 0, 0, 0, info) },
		"can mount":     func() { _, _, _ = remote.CanBeMount() },
	} {
		require.Panics(t, call, name)
	}

	headers, errs := remote.HeadBlob(ctx, info)
	require.Nil(t, headers)
	require.Len(t, errs, 1)
}

func TestRpgRemoteManifestExistAndPull(t *testing.T) {
	ctx := context.Background()
	custom := string(artifact.UpstreamConfigSourceCustom)
	mt, payload := rpgDockerManifest(t, "remote")
	parsed, _, err := manifest.UnmarshalManifest(mt, payload)
	require.NoError(t, err)
	dgst := rpgPayloadDigest(payload)

	type call func(r *RemoteRegistry, info pkg.RegistryInfo) (*commons.ResponseHeaders, manifest.Manifest, []error)
	calls := map[string]call{
		"exist": func(r *RemoteRegistry, info pkg.RegistryInfo) (*commons.ResponseHeaders, manifest.Manifest, []error) {
			h, _, m, errs := r.ManifestExist(ctx, info, nil, nil)
			return h, m, errs
		},
		"pull": func(r *RemoteRegistry, info pkg.RegistryInfo) (*commons.ResponseHeaders, manifest.Manifest, []error) {
			h, _, m, errs := r.PullManifest(ctx, info, nil, nil)
			return h, m, errs
		},
	}

	for name, run := range calls {
		t.Run(name, func(t *testing.T) {
			info := rpgInfo(t)
			info.Digest = dgst.String()

			t.Run("upstream config lookup fails", func(t *testing.T) {
				env := rpgNewEnv(t)
				remote := rpgRemote(t, env, &rpgProxyCtl{}, custom)
				remote.upstreamProxyConfigRepo = &rpgUpstreamRepo{err: errRpgBoom}
				_, _, errs := run(remote, info)
				require.Len(t, errs, 1)
				require.ErrorIs(t, errs[0], errRpgBoom)
			})
			t.Run("unknown adapter", func(t *testing.T) {
				env := rpgNewEnv(t)
				remote := rpgRemote(t, env, &rpgProxyCtl{}, "not-an-adapter")
				_, _, errs := run(remote, info)
				require.Len(t, errs, 1)
				require.ErrorContains(t, errs[0], "Proxy is down")
			})
			t.Run("local lookup fails", func(t *testing.T) {
				env := rpgNewEnv(t)
				remote := rpgRemote(t, env, &rpgProxyCtl{localErr: errRpgBoom}, custom)
				_, _, errs := run(remote, info)
				require.ErrorIs(t, errs[0], errRpgBoom)
			})
			t.Run("served from the local copy", func(t *testing.T) {
				env := rpgNewEnv(t)
				ctl := &rpgProxyCtl{useLocal: true, local: &proxy2.ManifestList{
					Content: payload, ContentType: mt, Digest: dgst.String(),
				}}
				headers, m, errs := run(rpgRemote(t, env, ctl, custom), info)
				require.Empty(t, errs)
				require.NotNil(t, m)
				require.Equal(t, dgst.String(), headers.Headers["Docker-Content-Digest"])
				require.Equal(t, mt, headers.Headers["Content-Type"])
			})
			t.Run("local copy cannot be parsed", func(t *testing.T) {
				env := rpgNewEnv(t)
				ctl := &rpgProxyCtl{useLocal: true, local: &proxy2.ManifestList{Content: []byte("{"), ContentType: mt}}
				_, _, errs := run(rpgRemote(t, env, ctl, custom), info)
				require.NotEmpty(t, errs)
			})
		})
	}

	t.Run("exist falls back to the remote head", func(t *testing.T) {
		env := rpgNewEnv(t)
		info := rpgInfo(t)
		ctl := &rpgProxyCtl{useLocal: true, headExists: true, headDesc: &manifest.Descriptor{
			Digest: dgst, Size: 7, MediaType: mt,
		}}
		headers, _, errs := calls["exist"](rpgRemote(t, env, ctl, custom), info)
		require.Len(t, errs, 1, "the missing local copy is reported")
		require.Equal(t, "7", headers.Headers["Content-Length"])
		require.Equal(t, dgst.String(), headers.Headers["Etag"])
	})
	t.Run("remote head failures", func(t *testing.T) {
		env := rpgNewEnv(t)
		info := rpgInfo(t)
		_, _, errs := calls["exist"](rpgRemote(t, env, &rpgProxyCtl{headErr: errRpgBoom}, custom), info)
		require.ErrorIs(t, errs[0], errRpgBoom)
		_, _, errs = calls["exist"](rpgRemote(t, env, &rpgProxyCtl{}, custom), info)
		require.Len(t, errs, 1)
		_, _, errs = calls["exist"](rpgRemote(t, env, &rpgProxyCtl{headExists: true}, custom), info)
		require.Len(t, errs, 1)
	})
	t.Run("head of a tag refreshes the local tag in the background", func(t *testing.T) {
		env := rpgNewEnv(t)
		info := rpgInfo(t)
		info.Tag = "latest"
		ctl := &rpgProxyCtl{headExists: true, headDesc: &manifest.Descriptor{Digest: dgst, Size: 1, MediaType: mt}}
		headers, _, errs := calls["exist"](rpgRemote(t, env, ctl, custom), info)
		require.Empty(t, errs)
		require.Equal(t, "1", headers.Headers["Content-Length"])
	})
	t.Run("pull proxies the manifest", func(t *testing.T) {
		env := rpgNewEnv(t)
		info := rpgInfo(t)
		info.Digest = dgst.String()
		headers, m, errs := calls["pull"](rpgRemote(t, env, &rpgProxyCtl{proxied: parsed}, custom), info)
		require.Empty(t, errs)
		require.NotNil(t, m)
		require.Equal(t, mt, headers.Headers["Content-Type"])
		require.Equal(t, dgst.String(), headers.Headers["Docker-Content-Digest"])

		_, _, errs = calls["pull"](rpgRemote(t, env, &rpgProxyCtl{proxyErr: errRpgBoom}, custom), info)
		require.ErrorIs(t, errs[0], errRpgBoom)
	})
	t.Run("pull reports a missing local copy and still proxies", func(t *testing.T) {
		env := rpgNewEnv(t)
		info := rpgInfo(t)
		_, m, errs := calls["pull"](rpgRemote(t, env, &rpgProxyCtl{useLocal: true, proxied: parsed}, custom), info)
		require.Len(t, errs, 1)
		require.NotNil(t, m)
	})
}

func TestRpgRemoteGetBlob(t *testing.T) {
	ctx := context.Background()
	custom := string(artifact.UpstreamConfigSourceCustom)
	info := rpgInfo(t)
	info.Digest = rpgDigest("blob").String()

	t.Run("proxied", func(t *testing.T) {
		env := rpgNewEnv(t)
		remote := rpgRemote(t, env, &rpgProxyCtl{blobSize: 4}, custom)
		headers, _, size, reader, _, errs := remote.GetBlob(ctx, info)
		require.Empty(t, errs)
		require.Equal(t, int64(4), size)
		require.NotNil(t, reader)
		require.Equal(t, "4", headers.Headers["Content-Length"])
		require.Equal(t, info.Digest, headers.Headers["Etag"])
		require.NotContains(t, headers.Headers, "Content-Type")
	})
	t.Run("proxy failure", func(t *testing.T) {
		env := rpgNewEnv(t)
		remote := rpgRemote(t, env, &rpgProxyCtl{blobErr: errRpgBoom}, custom)
		_, _, _, _, _, errs := remote.GetBlob(ctx, info)
		require.Len(t, errs, 1)
		require.ErrorIs(t, errs[0], errRpgBoom)
	})
	t.Run("local copy", func(t *testing.T) {
		env := rpgNewEnv(t)
		env.blobs.byRoot = func(_ digest.Digest) (*registrytypes.Blob, error) { return nil, errRpgBoom }
		remote := rpgRemote(t, env, &rpgProxyCtl{useLocalBlob: true}, custom)
		_, _, _, _, _, errs := remote.GetBlob(ctx, info)
		require.Len(t, errs, 1)

		_, _, _, _, _, errs = remote.fetchBlobInternal(ctx, "reg", http.MethodHead, info)
		require.Len(t, errs, 1)
		require.ErrorContains(t, errs[0], "Method not supported")
	})
}
