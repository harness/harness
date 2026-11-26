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
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/auth/authz"
	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/registry/app/dist_temp/errcode"
	"github.com/harness/gitness/registry/app/manifest"
	"github.com/harness/gitness/registry/app/pkg"
	"github.com/harness/gitness/registry/app/pkg/commons"
	"github.com/harness/gitness/registry/app/pkg/quarantine"
	registrystorage "github.com/harness/gitness/registry/app/storage"
	registrytypes "github.com/harness/gitness/registry/types"
	"github.com/harness/gitness/types"

	"github.com/opencontainers/go-digest"
	"github.com/stretchr/testify/require"
)

type rpgAuthorizer struct {
	authz.Authorizer
	allow bool
	err   error
}

func (a *rpgAuthorizer) CheckAll(context.Context, *auth.Session, ...types.PermissionCheck) (bool, error) {
	return a.allow, a.err
}

type rpgQuarantine struct {
	quarantine.Finder
	err error
}

func (q *rpgQuarantine) CheckOCIManifestQuarantineStatus(context.Context, int64, string, string, string) error {
	return q.err
}

// rpgRegistryFake serves manifests and blobs for the proxy wrapper tests.
type rpgRegistryFake struct {
	Registry
	manifest  func() (*commons.ResponseHeaders, manifest.Descriptor, manifest.Manifest, []error)
	blobErrs  []error
	blobCalls int
}

func (f *rpgRegistryFake) ManifestExist(
	context.Context, pkg.RegistryInfo, []string, []string,
) (*commons.ResponseHeaders, manifest.Descriptor, manifest.Manifest, []error) {
	return f.manifest()
}

func (f *rpgRegistryFake) PullManifest(
	context.Context, pkg.RegistryInfo, []string, []string,
) (*commons.ResponseHeaders, manifest.Descriptor, manifest.Manifest, []error) {
	return f.manifest()
}

func (f *rpgRegistryFake) GetBlob(
	context.Context, pkg.RegistryInfo,
) (*commons.ResponseHeaders, *registrystorage.FileReader, int64, io.ReadCloser, string, []error) {
	f.blobCalls++
	return rpgHeaders(), nil, 5, nil, "", f.blobErrs
}

func rpgServedManifest() (*commons.ResponseHeaders, manifest.Descriptor, manifest.Manifest, []error) {
	return rpgHeaders(), manifest.Descriptor{Digest: rpgDigest("m")}, nil, nil
}

func rpgController(t *testing.T, env *rpgEnv, allow bool) *Controller {
	t.Helper()
	return &Controller{
		CoreController: pkg.NewCoreController(env.regDao, &rpgQuarantine{}),
		local:          env.local,
		authorizer:     &rpgAuthorizer{allow: allow},
		SpaceFinder:    rpgSpaceFinder(nil),
	}
}

func rpgSetRegistries(t *testing.T, local, remote pkg.Artifact) {
	t.Helper()
	prevLocal := pkg.TypeRegistry[pkg.LocalRegistry]
	prevRemote := pkg.TypeRegistry[pkg.RemoteRegistry]
	pkg.TypeRegistry[pkg.LocalRegistry] = local
	pkg.TypeRegistry[pkg.RemoteRegistry] = remote
	t.Cleanup(func() {
		pkg.TypeRegistry[pkg.LocalRegistry] = prevLocal
		pkg.TypeRegistry[pkg.RemoteRegistry] = prevRemote
	})
}

func TestRpgNewController(t *testing.T) {
	env := rpgNewEnv(t)
	remote := &RemoteRegistry{}
	rpgSetRegistries(t, nil, nil)

	c := NewController(env.local, remote, pkg.NewCoreController(nil, nil), nil, nil, &DBStore{}, rpgSpaceFinder(nil))
	require.NotNil(t, c)
	require.Same(t, env.local, pkg.TypeRegistry[pkg.LocalRegistry])
	require.Same(t, remote, pkg.TypeRegistry[pkg.RemoteRegistry])

	c.GetCatalog(nil, nil)
}

func TestRpgProxyWrapper(t *testing.T) {
	ctx := context.Background()
	env := rpgNewEnv(t)
	upstream := registrytypes.Registry{ID: 10, Name: "upstream", Type: artifact.RegistryTypeUPSTREAM}
	env.regDao.ids = &[]registrytypes.Registry{upstream}
	c := rpgController(t, env, true)

	info := rpgInfo(t)
	info.Registry.UpstreamProxies = []int64{10}

	var seen []string
	call := func(registry registrytypes.Registry, _ string, _ pkg.Artifact) Response {
		seen = append(seen, registry.Name)
		return &GetManifestResponse{Errors: []error{errRpgBoom}}
	}

	t.Run("empty info", func(t *testing.T) {
		_, err := c.ProxyWrapper(ctx, call, pkg.RegistryInfo{}, ResourceTypeBlob)
		require.ErrorContains(t, err, "artifactinfo is not found")
	})
	t.Run("upstream lookup failure", func(t *testing.T) {
		env.regDao.err = errRpgBoom
		_, err := c.ProxyWrapper(ctx, call, info, ResourceTypeBlob)
		require.ErrorIs(t, err, errRpgBoom)
		env.regDao.err = nil
	})
	t.Run("registries are not usable", func(t *testing.T) {
		rpgSetRegistries(t, nil, nil)
		resp, err := c.ProxyWrapper(ctx, call, info, ResourceTypeBlob)
		require.NoError(t, err)
		require.Nil(t, resp)
		require.Empty(t, seen)
	})
	t.Run("falls back to the upstream", func(t *testing.T) {
		rpgSetRegistries(t, &rpgRegistryFake{}, &rpgRegistryFake{})
		seen = nil
		resp, err := c.ProxyWrapper(ctx, func(registry registrytypes.Registry, _ string, _ pkg.Artifact) Response {
			seen = append(seen, registry.Name)
			if registry.Name == "upstream" {
				return &GetManifestResponse{}
			}
			return &GetManifestResponse{Errors: []error{errRpgBoom}}
		}, info, ResourceTypeManifest)
		require.NoError(t, err)
		require.Empty(t, resp.GetErrors())
		require.Equal(t, []string{"reg", "upstream"}, seen)
	})
	t.Run("error mapping when nothing serves the request", func(t *testing.T) {
		rpgSetRegistries(t, &rpgRegistryFake{}, &rpgRegistryFake{})
		tests := []struct {
			resource string
			err      error
			want     errcode.CodeError
		}{
			{ResourceTypeManifest, errRpgBoom, errcode.ErrCodeManifestUnknown},
			{ResourceTypeManifest, usererror.ErrQuarantinedArtifact, errcode.ErrCodeManifestQuarantined},
			{ResourceTypeBlob, errRpgBoom, errcode.ErrCodeBlobUnknown},
			{"other", errRpgBoom, -1},
		}
		for _, tt := range tests {
			resp, err := c.ProxyWrapper(ctx, func(registrytypes.Registry, string, pkg.Artifact) Response {
				return &GetManifestResponse{Errors: []error{tt.err}}
			}, info, tt.resource)
			require.NoError(t, err)
			got := rpgCodes(resp.GetErrors())
			require.Len(t, got, 1)
			if tt.want != -1 {
				require.Equal(t, tt.want, got[0], tt.resource)
			}
		}
	})
}

func TestRpgControllerManifestAndBlobReads(t *testing.T) {
	ctx := rpgAuthCtx(context.Background())
	env := rpgNewEnv(t)
	c := rpgController(t, env, true)
	info := rpgInfo(t)
	info.Digest = rpgDigest("m").String()

	served := &rpgRegistryFake{manifest: rpgServedManifest}
	rpgSetRegistries(t, served, served)

	for name, call := range map[string]func(*Controller, pkg.RegistryInfo) Response{
		"head": func(c *Controller, i pkg.RegistryInfo) Response { return c.HeadManifest(ctx, i, nil, nil) },
		"pull": func(c *Controller, i pkg.RegistryInfo) Response { return c.PullManifest(ctx, i, nil, nil) },
	} {
		t.Run(name, func(t *testing.T) {
			resp := call(c, info)
			require.Empty(t, resp.GetErrors())
			manifestResp, ok := resp.(*GetManifestResponse)
			require.True(t, ok)
			require.Equal(t, rpgDigest("m"), manifestResp.descriptor.Digest)

			denied := rpgController(t, env, false)
			require.Equal(t, []errcode.CodeError{errcode.ErrCodeDenied}, rpgCodes(call(denied, info).GetErrors()))
		})
	}

	t.Run("pull is blocked by quarantine", func(t *testing.T) {
		quarantined := rpgController(t, env, true)
		quarantined.QuarantineFinder = &rpgQuarantine{err: usererror.ErrQuarantinedArtifact}
		resp := quarantined.PullManifest(ctx, info, nil, nil)
		require.Equal(t, []errcode.CodeError{errcode.ErrCodeManifestQuarantined}, rpgCodes(resp.GetErrors()))

		quarantined.QuarantineFinder = &rpgQuarantine{err: errRpgBoom}
		resp = quarantined.PullManifest(ctx, info, nil, nil)
		require.Equal(t, []errcode.CodeError{errcode.ErrCodeManifestUnknown}, rpgCodes(resp.GetErrors()))
	})
	t.Run("proxy failure becomes an error response", func(t *testing.T) {
		env.regDao.err = errRpgBoom
		defer func() { env.regDao.err = nil }()
		proxied := rpgInfo(t)
		proxied.Registry.UpstreamProxies = []int64{10}
		for _, resp := range []Response{
			c.HeadManifest(ctx, proxied, nil, nil), c.PullManifest(ctx, proxied, nil, nil), c.GetBlob(ctx, proxied),
		} {
			require.Len(t, resp.GetErrors(), 1)
			require.ErrorIs(t, resp.GetErrors()[0], errRpgBoom)
		}
	})
	t.Run("blob", func(t *testing.T) {
		resp := c.GetBlob(ctx, info)
		require.Empty(t, resp.GetErrors())
		blobResp, ok := resp.(*GetBlobResponse)
		require.True(t, ok)
		require.Equal(t, int64(5), blobResp.Size)
		require.Equal(t, 1, served.blobCalls)

		denied := rpgController(t, env, false)
		require.Equal(t, []errcode.CodeError{errcode.ErrCodeDenied}, rpgCodes(denied.GetBlob(ctx, info).GetErrors()))
	})
}

func TestRpgControllerDeniedWrites(t *testing.T) {
	ctx := rpgAuthCtx(context.Background())
	env := rpgNewEnv(t)
	c := rpgController(t, env, false)
	info := rpgInfo(t)
	denied := []errcode.CodeError{errcode.ErrCodeDenied}
	body := io.NopCloser(strings.NewReader("x"))

	headers, errs := c.PutManifest(ctx, info, rpgDockerMT, body, 1)
	require.Nil(t, headers)
	require.Equal(t, denied, rpgCodes(errs))

	errs, headers = c.DeleteManifest(ctx, info)
	require.Nil(t, headers)
	require.Equal(t, denied, rpgCodes(errs))

	headers, errs = c.HeadBlob(ctx, info)
	require.Nil(t, headers)
	require.Equal(t, denied, rpgCodes(errs))

	_, errs = c.InitiateUploadBlob(ctx, info, "", "")
	require.Equal(t, denied, rpgCodes(errs))
	_, errs = c.GetUploadBlobStatus(ctx, info, "")
	require.Equal(t, denied, rpgCodes(errs))
	_, errs = c.PatchBlobUpload(ctx, info, "", "", "", 1, "", body)
	require.Equal(t, denied, rpgCodes(errs))
	_, errs = c.CompleteBlobUpload(ctx, info, body, 1, "")
	require.Equal(t, denied, rpgCodes(errs))
	_, errs = c.CancelBlobUpload(ctx, info, "")
	require.Equal(t, denied, rpgCodes(errs))
	_, errs = c.DeleteBlob(ctx, info)
	require.Equal(t, denied, rpgCodes(errs))

	_, _, err := c.GetTags(ctx, "", 1, "", info)
	require.Equal(t, errcode.ErrCodeDenied, rpgCodes([]error{err})[0])
	_, _, err = c.GetReferrers(ctx, info, "")
	require.Equal(t, errcode.ErrCodeDenied, rpgCodes([]error{err})[0])
}

func TestRpgControllerUploadFlow(t *testing.T) {
	ctx := rpgAuthCtx(context.Background())
	env := rpgNewEnv(t)
	env.app.bucketService = &rpgBucket{}
	c := rpgController(t, env, true)
	info := rpgInfo(t)
	content := []byte("controller blob")
	dgst := digest.FromBytes(content)

	headers, errs := c.InitiateUploadBlob(ctx, info, "", "")
	require.Empty(t, errs)
	uploadID := headers.Headers["Docker-Upload-UUID"]
	token := rpgStateFromLocation(t, headers.Headers["Location"])
	info.Reference = uploadID

	headers, errs = c.GetUploadBlobStatus(ctx, info, token)
	require.Empty(t, errs)
	require.Equal(t, http.StatusNoContent, headers.Code)

	headers, errs = c.PatchBlobUpload(
		ctx, info, "application/octet-stream", "0-5", "6", 6, token, io.NopCloser(bytes.NewReader(content[:6])),
	)
	require.Empty(t, errs)
	require.Equal(t, http.StatusAccepted, headers.Code)
	token = rpgStateFromLocation(t, headers.Headers["Location"])

	info.Digest = dgst.String()
	headers, errs = c.CompleteBlobUpload(
		ctx, info, io.NopCloser(bytes.NewReader(content[6:])), int64(len(content)-6), token,
	)
	require.Empty(t, errs)
	require.Equal(t, http.StatusCreated, headers.Code)

	headBlob, errs := c.HeadBlob(ctx, info)
	require.Empty(t, errs)
	require.NotEmpty(t, headBlob.Headers["Content-Length"])

	headers, errs = c.DeleteBlob(ctx, info)
	require.Empty(t, errs)
	require.Equal(t, http.StatusAccepted, headers.Code)

	// A second upload is cancelled.
	info.Digest = ""
	info.Reference = ""
	headers, errs = c.InitiateUploadBlob(ctx, info, "", "")
	require.Empty(t, errs)
	info.Reference = headers.Headers["Docker-Upload-UUID"]
	token = rpgStateFromLocation(t, headers.Headers["Location"])
	headers, errs = c.CancelBlobUpload(ctx, info, token)
	require.Empty(t, errs)
	require.Equal(t, http.StatusNoContent, headers.Code)
	require.Equal(t, info.Reference, headers.Headers["Docker-Upload-UUID"])

	// Cancelling without an upload id reports an unknown upload.
	info.Reference = ""
	_, errs = c.CancelBlobUpload(ctx, info, token)
	require.Equal(t, []errcode.CodeError{errcode.ErrCodeBlobUploadUnknown}, rpgCodes(errs))
}

func TestRpgControllerDelegatesToLocalRegistry(t *testing.T) {
	ctx := rpgAuthCtx(context.Background())
	env := rpgNewEnv(t)
	c := rpgController(t, env, true)
	info := rpgInfo(t)

	_, errs := c.PutManifest(ctx, info, rpgDockerMT, io.NopCloser(strings.NewReader("{")), 1)
	require.Equal(t, []errcode.CodeError{errcode.ErrCodeManifestInvalid}, rpgCodes(errs))

	info.Tag = "v1"
	errs, headers := c.DeleteManifest(ctx, info)
	require.Empty(t, errs)
	require.Equal(t, http.StatusAccepted, headers.Code)

	env.tags.tags = []*registrytypes.Tag{{Name: "v1"}}
	_, tags, err := c.GetTags(ctx, "", 10, "http://localhost/x", info)
	require.NoError(t, err)
	require.Equal(t, []string{"v1"}, tags)

	env.manifests.bySubject = func(registrytypes.Digest) (registrytypes.Manifests, error) { return nil, nil }
	info.Digest = rpgDigest("subject").String()
	index, _, err := c.GetReferrers(ctx, info, "")
	require.NoError(t, err)
	require.Empty(t, index.Manifests)

	// Failing blob stores are reported by the upload endpoints.
	env.app.storageService, _ = registrystorage.NewStorageService(rpgFailingResolver{})
	_, errs = c.GetUploadBlobStatus(ctx, info, "")
	require.Equal(t, []errcode.CodeError{errcode.ErrCodeUnknown}, rpgCodes(errs))
	_, errs = c.PatchBlobUpload(ctx, info, "", "", "", 1, "", io.NopCloser(strings.NewReader("x")))
	require.Equal(t, []errcode.CodeError{errcode.ErrCodeUnknown}, rpgCodes(errs))
	_, errs = c.CancelBlobUpload(ctx, info, "")
	require.Equal(t, []errcode.CodeError{errcode.ErrCodeUnknown}, rpgCodes(errs))
	_, errs = c.DeleteBlob(ctx, info)
	require.Equal(t, []errcode.CodeError{errcode.ErrCodeUnknown}, rpgCodes(errs))
}

func TestRpgResumeBlobUpload(t *testing.T) {
	ctx := context.Background()
	env := rpgNewEnv(t)
	info := rpgInfo(t)
	uploadID, token := rpgUpload(ctx, t, env, info)
	info.Reference = uploadID

	newCtx := func() *Context {
		blobCtx, err := env.app.GetBlobsContext(ctx, info, registrytypes.BlobLocator{})
		require.NoError(t, err)
		return blobCtx
	}
	secret := hmacKey(rpgSecret)

	blobCtx := newCtx()
	require.Empty(t, ResumeBlobUpload(blobCtx, token))
	require.NotNil(t, blobCtx.Upload)

	require.Equal(
		t, []errcode.CodeError{errcode.ErrCodeBlobUploadInvalid}, rpgCodes(ResumeBlobUpload(newCtx(), "garbage")),
	)

	wrongPath, err := secret.packUploadState(BlobUploadState{Path: "elsewhere", UUID: uploadID})
	require.NoError(t, err)
	require.Equal(
		t, []errcode.CodeError{errcode.ErrCodeBlobUploadInvalid}, rpgCodes(ResumeBlobUpload(newCtx(), wrongPath)),
	)

	storePath := newCtx().OciBlobStore.Path()
	wrongUUID, err := secret.packUploadState(BlobUploadState{Path: storePath, UUID: "other"})
	require.NoError(t, err)
	require.Equal(
		t, []errcode.CodeError{errcode.ErrCodeBlobUploadInvalid}, rpgCodes(ResumeBlobUpload(newCtx(), wrongUUID)),
	)

	wrongOffset, err := secret.packUploadState(BlobUploadState{Path: storePath, UUID: uploadID, Offset: 99})
	require.NoError(t, err)
	require.Equal(t, []errcode.CodeError{errcode.ErrCodeRangeInvalid}, rpgCodes(ResumeBlobUpload(newCtx(), wrongOffset)))

	for resumeErr, want := range map[error]errcode.CodeError{
		registrystorage.ErrBlobUploadUnknown: errcode.ErrCodeBlobUploadUnknown,
		errRpgBoom:                           errcode.ErrCodeUnknown,
	} {
		fake := &rpgResumeStore{err: resumeErr}
		failing := &Context{App: env.app, Context: ctx, OciBlobStore: fake, UUID: uploadID}
		failing.OciBlobStore = &rpgPathStore{rpgResumeStore: fake, path: "p"}
		state, packErr := secret.packUploadState(BlobUploadState{Path: "p", UUID: uploadID})
		require.NoError(t, packErr)
		require.Equal(t, []errcode.CodeError{want}, rpgCodes(ResumeBlobUpload(failing, state)))
	}
}

type rpgPathStore struct {
	*rpgResumeStore
	path string
}

func (s *rpgPathStore) Path() string { return s.path }

func TestRpgHandleQuarantineError(t *testing.T) {
	ctx := context.Background()
	resp := handleQuarantineError(ctx, usererror.ErrQuarantinedArtifact, "img", "v1", "sha")
	require.ErrorIs(t, resp.Errors[0], usererror.ErrQuarantinedArtifact)

	other := errors.New("lookup failed")
	resp = handleQuarantineError(ctx, other, "img", "v1", "sha")
	require.ErrorIs(t, resp.Errors[0], other)
}
