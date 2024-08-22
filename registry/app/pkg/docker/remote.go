// Source: https://github.com/goharbor/harbor

// Copyright 2016 Project Harbor Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package docker

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/harness/gitness/app/api/request"
	store2 "github.com/harness/gitness/app/store"
	"github.com/harness/gitness/encrypt"
	"github.com/harness/gitness/registry/app/common/lib/errors"
	"github.com/harness/gitness/registry/app/manifest"
	"github.com/harness/gitness/registry/app/pkg"
	"github.com/harness/gitness/registry/app/pkg/commons"
	proxy2 "github.com/harness/gitness/registry/app/remote/controller/proxy"
	"github.com/harness/gitness/registry/app/storage"
	"github.com/harness/gitness/registry/app/store"

	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/rs/zerolog/log"
)

const (
	contentLength       = "Content-Length"
	contentType         = "Content-Type"
	dockerContentDigest = "Docker-Content-Digest"
	etag                = "Etag"
	ensureTagInterval   = 10 * time.Second
	ensureTagMaxRetry   = 60
)

func NewRemoteRegistry(
	local *LocalRegistry,
	app *App,
	upstreamProxyConfigRepo store.UpstreamProxyConfigRepository,
	secretStore store2.SecretStore,
	encrypter encrypt.Encrypter,
) Registry {
	return &RemoteRegistry{
		local:                   local,
		App:                     app,
		upstreamProxyConfigRepo: upstreamProxyConfigRepo,
		secretStore:             secretStore,
		encrypter:               encrypter,
	}
}

func (r *RemoteRegistry) GetArtifactType() string {
	return "Remote Registry"
}

type RemoteRegistry struct {
	local                   *LocalRegistry
	App                     *App
	upstreamProxyConfigRepo store.UpstreamProxyConfigRepository
	secretStore             store2.SecretStore
	encrypter               encrypt.Encrypter
}

func (r *RemoteRegistry) Base() error {
	panic("Not implemented yet, will be done during Replication flows")
}

func defaultLibrary() (bool, string, error) {
	// get upstream Repository and check if the path contains library prefix. If yes, redirect to the correct path without
	// library prefix.
	return false, "", nil
}

// defaultManifestURL return the real url for request with default project.
func defaultManifestURL(regIdentifier string, name string, a pkg.RegistryInfo) string {
	return fmt.Sprintf("/v2/%s/library/%s/manifests/%s", regIdentifier, name, a.Reference)
}

func proxyManifestHead(
	ctx context.Context,
	responseHeaders *commons.ResponseHeaders,
	ctl proxy2.Controller,
	art pkg.RegistryInfo,
	remote proxy2.RemoteInterface,
	info pkg.RegistryInfo,
	acceptHeaders []string,
	ifNoneMatchHeader []string,
) error {
	// remote call
	exist, desc, err := ctl.HeadManifest(ctx, art, remote)
	if err != nil {
		return err
	}
	if !exist || desc == nil {
		return errors.NotFoundError(fmt.Errorf("the tag %v:%v is not found", art.Image, art.Tag))
	}

	if len(art.Tag) > 0 {
		go func(art pkg.RegistryInfo) {
			// Write function to update local storage.
			session, _ := request.AuthSessionFrom(ctx)
			ctx2 := request.WithAuthSession(context.Background(), session)
			tag := art.Tag
			art.Tag = ""
			art.Digest = desc.Digest.String()

			var count = 0
			for i := 0; i < ensureTagMaxRetry; i++ {
				time.Sleep(ensureTagInterval)
				count++
				log.Ctx(ctx).Info().Msgf("Ensure tag: %s for image: %s, retry: %d", tag, info.Image, count)
				e := ctl.EnsureTag(ctx2, responseHeaders, art, acceptHeaders, ifNoneMatchHeader)
				if e != nil {
					log.Ctx(ctx).Warn().Err(e).Msgf("Failed to update tag: ")
				} else {
					log.Ctx(ctx).Info().Msgf("Tag updated: %s for image: %s", tag, info.Image)
					return
				}
			}
		}(art)
	}

	responseHeaders.Headers[contentLength] = fmt.Sprintf("%v", desc.Size)
	responseHeaders.Headers[contentType] = desc.MediaType
	responseHeaders.Headers[dockerContentDigest] = string(desc.Digest)
	responseHeaders.Headers[etag] = string(desc.Digest)
	return nil
}

func (r *RemoteRegistry) ManifestExist(
	ctx context.Context,
	artInfo pkg.RegistryInfo,
	acceptHeaders []string,
	ifNoneMatchHeader []string,
) (
	responseHeaders *commons.ResponseHeaders, descriptor manifest.Descriptor, manifestResult manifest.Manifest,
	errs []error,
) {
	proxyCtl := proxy2.ControllerInstance(r.local, r.local.ms)
	responseHeaders = &commons.ResponseHeaders{
		Headers: make(map[string]string),
	}
	defaultProj, name, err := defaultLibrary()
	if err != nil {
		errs = append(errs, err)
		return responseHeaders, descriptor, manifestResult, errs
	}
	registryInfo := artInfo
	if defaultProj {
		responseHeaders.Code = http.StatusMovedPermanently
		responseHeaders.Headers = map[string]string{
			"Location": defaultManifestURL(artInfo.RegIdentifier, name, registryInfo),
		}
		return responseHeaders, descriptor, manifestResult, errs
	}

	if !canProxy() {
		errs = append(errs, errors.New("Proxy is down"))
		return responseHeaders, descriptor, manifestResult, errs
	}

	upstreamProxy, err := r.upstreamProxyConfigRepo.GetByRegistryIdentifier(
		ctx, artInfo.ParentID, artInfo.RegIdentifier,
	)
	if err != nil {
		errs = append(errs, err)
		return responseHeaders, descriptor, manifestResult, errs
	}
	remoteHelper, err := proxy2.NewRemoteHelper(ctx, r.secretStore, r.encrypter, artInfo.RegIdentifier, *upstreamProxy)
	if err != nil {
		errs = append(errs, errors.New("Proxy is down"))
		return responseHeaders, descriptor, manifestResult, errs
	}
	useLocal, man, err := proxyCtl.UseLocalManifest(ctx, registryInfo, remoteHelper, acceptHeaders, ifNoneMatchHeader)

	if err != nil {
		errs = append(errs, err)
		return responseHeaders, descriptor, manifestResult, errs
	}

	if useLocal {
		if man != nil {
			responseHeaders.Headers[contentLength] = fmt.Sprintf("%v", len(man.Content))
			responseHeaders.Headers[contentType] = man.ContentType
			responseHeaders.Headers[dockerContentDigest] = man.Digest
			responseHeaders.Headers[etag] = man.Digest
			manifestResult, descriptor, err = manifest.UnmarshalManifest(man.ContentType, man.Content)
			if err != nil {
				errs = append(errs, err)
				return responseHeaders, descriptor, manifestResult, errs
			}
			return responseHeaders, descriptor, manifestResult, errs
		}
		errs = append(errs, errors.New("Not found"))
	}

	log.Ctx(ctx).Debug().Msgf("the tag is %s, digest is %s", registryInfo.Tag, registryInfo.Digest)
	err = proxyManifestHead(
		ctx,
		responseHeaders,
		proxyCtl,
		registryInfo,
		remoteHelper,
		artInfo,
		acceptHeaders,
		ifNoneMatchHeader,
	)

	if err != nil {
		errs = append(errs, err)
		log.Ctx(ctx).Warn().Msgf(
			"Proxy to remote failed, fallback to local registry: %s",
			err.Error(),
		)
	}
	return responseHeaders, descriptor, manifestResult, errs
}

func (r *RemoteRegistry) PullManifest(
	ctx context.Context,
	artInfo pkg.RegistryInfo,
	acceptHeaders []string,
	ifNoneMatchHeader []string,
) (
	responseHeaders *commons.ResponseHeaders, descriptor manifest.Descriptor, manifestResult manifest.Manifest,
	errs []error,
) {
	proxyCtl := proxy2.ControllerInstance(r.local, r.local.ms)
	responseHeaders = &commons.ResponseHeaders{
		Headers: make(map[string]string),
	}
	defaultProj, name, err := defaultLibrary()
	if err != nil {
		errs = append(errs, err)
		return responseHeaders, descriptor, manifestResult, errs
	}
	registryInfo := artInfo
	if defaultProj {
		responseHeaders.Code = http.StatusMovedPermanently
		responseHeaders.Headers = map[string]string{
			"Location": defaultManifestURL(artInfo.RegIdentifier, name, registryInfo),
		}
		return responseHeaders, descriptor, manifestResult, errs
	}

	if !canProxy() {
		errs = append(errs, errors.New("Proxy is down"))
		return responseHeaders, descriptor, manifestResult, errs
	}
	upstreamProxy, err := r.upstreamProxyConfigRepo.GetByRegistryIdentifier(
		ctx, artInfo.ParentID, artInfo.RegIdentifier,
	)
	if err != nil {
		errs = append(errs, err)
		return responseHeaders, descriptor, manifestResult, errs
	}
	remoteHelper, err := proxy2.NewRemoteHelper(ctx, r.secretStore, r.encrypter, artInfo.RegIdentifier, *upstreamProxy)
	if err != nil {
		errs = append(errs, errors.New("Proxy is down"))
		return responseHeaders, descriptor, manifestResult, errs
	}
	useLocal, man, err := proxyCtl.UseLocalManifest(ctx, registryInfo, remoteHelper, acceptHeaders, ifNoneMatchHeader)

	if err != nil {
		errs = append(errs, err)
		return responseHeaders, descriptor, manifestResult, errs
	}

	if useLocal {
		if man != nil {
			responseHeaders.Headers[contentLength] = fmt.Sprintf("%v", len(man.Content))
			responseHeaders.Headers[contentType] = man.ContentType
			responseHeaders.Headers[dockerContentDigest] = man.Digest
			responseHeaders.Headers[etag] = man.Digest
			manifestResult, descriptor, err = manifest.UnmarshalManifest(man.ContentType, man.Content)
			if err != nil {
				errs = append(errs, err)
				return responseHeaders, descriptor, manifestResult, errs
			}
			return responseHeaders, descriptor, manifestResult, errs
		}
		errs = append(errs, errors.New("Not found"))
	}

	log.Ctx(ctx).Debug().Msgf("the tag is %s, digest is %s", registryInfo.Tag, registryInfo.Digest)
	log.Ctx(ctx).Warn().
		Msgf(
			"Artifact: %s:%v, digest:%v is not found in proxy cache, fetch it from remote registry",
			artInfo.RegIdentifier, registryInfo.Tag, registryInfo.Digest,
		)
	manifestResult, err = proxyManifestGet(
		ctx,
		responseHeaders,
		proxyCtl,
		registryInfo,
		remoteHelper,
		artInfo.RegIdentifier,
		artInfo.Image,
		acceptHeaders,
		ifNoneMatchHeader,
	)
	if err != nil {
		errs = append(errs, err)
		log.Ctx(ctx).Warn().Msgf("Proxy to remote failed, fallback to local registry: %s", err.Error())
	}
	return responseHeaders, descriptor, manifestResult, errs
}

func (r *RemoteRegistry) HeadBlob(
	ctx2 context.Context,
	artInfo pkg.RegistryInfo,
) (
	responseHeaders *commons.ResponseHeaders, fr *storage.FileReader, size int64,
	readCloser io.ReadCloser, redirectURL string, errs []error,
) {
	return r.fetchBlobInternal(ctx2, artInfo.RegIdentifier, http.MethodHead, artInfo)
}

// TODO (Arvind): There is a known issue where if the remote itself
// is a proxy, then the first pull will fail with error: `error pulling image configuration:
// image config verification failed for digest` and the second pull will succeed. This is a
// known issue and is being worked on. The workaround is to pull the image twice.
func (r *RemoteRegistry) GetBlob(
	ctx2 context.Context,
	artInfo pkg.RegistryInfo,
) (
	responseHeaders *commons.ResponseHeaders, fr *storage.FileReader, size int64, readCloser io.ReadCloser,
	redirectURL string, errs []error,
) {
	return r.fetchBlobInternal(ctx2, artInfo.RegIdentifier, http.MethodGet, artInfo)
}

func (r *RemoteRegistry) fetchBlobInternal(
	ctx context.Context,
	repoKey string,
	method string,
	info pkg.RegistryInfo,
) (
	responseHeaders *commons.ResponseHeaders, fr *storage.FileReader, size int64, readCloser io.ReadCloser,
	redirectURL string, errs []error,
) {
	proxyCtl := proxy2.ControllerInstance(r.local, r.local.ms)
	responseHeaders = &commons.ResponseHeaders{
		Headers: make(map[string]string),
	}

	log.Ctx(ctx).Info().Msgf("Proxy: %s", repoKey)

	// Handle dockerhub request without library prefix.
	isDefault, name, err := defaultLibrary()
	if err != nil {
		errs = append(errs, err)
		return responseHeaders, fr, size, readCloser, redirectURL, errs
	}
	registryInfo := info
	if isDefault {
		responseHeaders.Code = http.StatusMovedPermanently
		responseHeaders.Headers = map[string]string{
			"Location": defaultManifestURL(repoKey, name, registryInfo),
		}
		return responseHeaders, fr, size, readCloser, redirectURL, errs
	}

	if !canProxy() {
		errs = append(errs, errors.New("Blob not found"))
	}

	if proxyCtl.UseLocalBlob(ctx, registryInfo) {
		switch method {
		case http.MethodGet:
			headers, reader, s, closer, url, e := r.local.GetBlob(ctx, info)
			return headers, reader, s, closer, url, e
		case http.MethodHead:
			headers, reader, s, closer, url, e := r.local.HeadBlob(ctx, info)
			return headers, reader, s, closer, url, e
		default:
			errs = append(errs, errors.New("Method not supported"))
			return responseHeaders, fr, size, readCloser, redirectURL, errs
		}
	}

	upstreamProxy, err := r.upstreamProxyConfigRepo.GetByRegistryIdentifier(ctx, info.ParentID, repoKey)
	if err != nil {
		errs = append(errs, err)
	}

	// This is start of proxy Code.
	size, readCloser, err = proxyCtl.ProxyBlob(ctx, r.secretStore, r.encrypter, registryInfo, repoKey, *upstreamProxy)
	if err != nil {
		errs = append(errs, err)
		return responseHeaders, fr, size, readCloser, redirectURL, errs
	}
	setHeaders(responseHeaders, size, "", registryInfo.Digest)
	return responseHeaders, fr, size, readCloser, redirectURL, errs
}

func proxyManifestGet(
	ctx context.Context,
	responseHeaders *commons.ResponseHeaders,
	ctl proxy2.Controller,
	registryInfo pkg.RegistryInfo,
	remote proxy2.RemoteInterface,
	repoKey string,
	imageName string,
	acceptHeader []string,
	ifNoneMatchHeader []string,
) (man manifest.Manifest, err error) {
	man, err = ctl.ProxyManifest(ctx, registryInfo, remote, repoKey, imageName, acceptHeader, ifNoneMatchHeader)
	if err != nil {
		return
	}
	ct, payload, err := man.Payload()
	if err != nil {
		return
	}
	setHeaders(responseHeaders, int64(len(payload)), ct, registryInfo.Digest)
	return
}

func setHeaders(
	responseHeaders *commons.ResponseHeaders, size int64,
	mediaType string, dig string,
) {
	responseHeaders.Headers[contentLength] = fmt.Sprintf("%v", size)
	if len(mediaType) > 0 {
		responseHeaders.Headers[contentType] = mediaType
	}
	responseHeaders.Headers[dockerContentDigest] = dig
	responseHeaders.Headers[etag] = dig
}

func canProxy() bool {
	// TODO Health check.
	return true
}

func (r *RemoteRegistry) PushBlobMonolith(
	_ context.Context,
	_ pkg.RegistryInfo,
	_ int64,
	_ io.Reader,
) error {
	panic("Not implemented yet, will be done during Replication flows")
}

func (r *RemoteRegistry) InitBlobUpload(
	_ context.Context,
	_ pkg.RegistryInfo,
	_, _ string,
) (*commons.ResponseHeaders, []error) {
	panic("Not implemented yet, will be done during Replication flows")
}

func (r *RemoteRegistry) PushBlobMonolithWithDigest(
	_ context.Context,
	_ pkg.RegistryInfo,
	_ int64,
	_ io.Reader,
) error {
	panic("Not implemented yet, will be done during Replication flows")
}

func (r *RemoteRegistry) PushBlobChunk(
	_ *Context,
	_ pkg.RegistryInfo,
	_ string,
	_ string,
	_ string,
	_ io.ReadCloser,
	_ int64,
) (*commons.ResponseHeaders, []error) {
	panic("Not implemented yet, will be done during Replication flows")
}

func (r *RemoteRegistry) PushBlob(
	_ context.Context,
	_ pkg.RegistryInfo,
	_ io.ReadCloser,
	_ int64,
	_ string,
) (*commons.ResponseHeaders, []error) {
	panic("Not implemented yet, will be done during Replication flows")
}

func (r *RemoteRegistry) PutManifest(
	_ context.Context,
	_ pkg.RegistryInfo,
	_ string,
	_ io.ReadCloser,
	_ int64,
) (*commons.ResponseHeaders, []error) {
	panic("Not implemented yet, will be done during Replication flows")
}

func (r *RemoteRegistry) ListTags(
	_ context.Context,
	_ string,
	_ int,
	_ string,
	_ pkg.RegistryInfo,
) (*commons.ResponseHeaders, []string, error) {
	panic("Not implemented yet, will be done during Replication flows")
}

func (r *RemoteRegistry) ListFilteredTags(
	_ context.Context,
	_ int,
	_, _ string,
	_ pkg.RegistryInfo,
) (tags []string, err error) {
	panic("Not implemented yet, will be done during Replication flows")
}

func (r *RemoteRegistry) DeleteManifest(
	_ context.Context,
	_ pkg.RegistryInfo,
) (errs []error, responseHeaders *commons.ResponseHeaders) {
	panic("Not implemented yet, will be done during Replication flows")
}

func (r *RemoteRegistry) DeleteBlob(
	_ *Context,
	_ pkg.RegistryInfo,
) (responseHeaders *commons.ResponseHeaders, errs []error) {
	panic("Not implemented yet, will be done during Replication flows")
}

func (r *RemoteRegistry) MountBlob(
	_ context.Context,
	_ pkg.RegistryInfo,
	_, _ string,
) (err error) {
	panic("Not implemented yet, will be done during Replication flows")
}

func (r *RemoteRegistry) ListReferrers(
	_ context.Context,
	_ pkg.RegistryInfo,
	_ string,
) (index *v1.Index, responseHeaders *commons.ResponseHeaders, err error) {
	panic("Not implemented yet, will be done during Replication flows")
}

func (r *RemoteRegistry) GetBlobUploadStatus(
	_ *Context,
	_ pkg.RegistryInfo,
	_ string,
) (*commons.ResponseHeaders, []error) {
	panic("Not implemented yet, will be done during Replication flows")
}

func (r *RemoteRegistry) GetCatalog() (repositories []string, err error) {
	panic("Not implemented yet, will be done during Replication flows")
}

func (r *RemoteRegistry) DeleteTag(
	_, _ string,
	_ pkg.RegistryInfo,
) error {
	panic("Not implemented yet, will be done during Replication flows")
}

func (r *RemoteRegistry) PullBlobChunk(
	_, _ string,
	_, _, _ int64,
	_ pkg.RegistryInfo,
) (size int64, blob io.ReadCloser, err error) {
	panic(
		"Not implemented yet, will be done during Replication flows",
	)
}

func (r *RemoteRegistry) CanBeMount() (mount bool, repository string, err error) {
	panic("Not implemented yet, will be done during Replication flows")
}
