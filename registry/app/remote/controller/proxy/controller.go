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

package proxy

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/url"
	"time"

	"github.com/harness/gitness/app/api/request"
	"github.com/harness/gitness/app/services/refcache"
	"github.com/harness/gitness/registry/app/common/lib/errors"
	"github.com/harness/gitness/registry/app/manifest"
	"github.com/harness/gitness/registry/app/pkg"
	"github.com/harness/gitness/registry/app/pkg/commons"
	cfg "github.com/harness/gitness/registry/config"
	"github.com/harness/gitness/registry/types"
	"github.com/harness/gitness/secret"

	"github.com/distribution/distribution/v3/registry/api/errcode"
	"github.com/opencontainers/go-digest"
	"github.com/rs/zerolog/log"
)

const (
	// wait more time than manifest (maxManifestWait) because manifest list depends on manifest ready.
	maxManifestWait               = 10
	maxManifestMappingWait        = 10
	maxManifestMappingIntervalSec = 10
	sleepIntervalSec              = 20
	// keep manifest list in cache for one week.
)

// Controller defines the operations related with pull through proxy.
type Controller interface {
	// UseLocalBlob check if the blob should use localRegistry copy.
	UseLocalBlob(ctx context.Context, art pkg.RegistryInfo) bool
	// UseLocalManifest check manifest should use localRegistry copy
	UseLocalManifest(
		ctx context.Context,
		art pkg.RegistryInfo,
		remote RemoteInterface,
		acceptHeader []string,
		ifNoneMatchHeader []string,
	) (bool, *ManifestList, error)
	// ProxyBlob proxy the blob request to the remote server, p is the proxy project
	// art is the RegistryInfo which includes the digest of the blob
	ProxyBlob(
		ctx context.Context, art pkg.RegistryInfo, repoKey string, proxy types.UpstreamProxy,
	) (int64, io.ReadCloser, error)
	// ProxyManifest proxy the manifest request to the remote server, p is the proxy project,
	// art is the RegistryInfo which includes the tag or digest of the manifest
	ProxyManifest(
		ctx context.Context,
		art pkg.RegistryInfo,
		remote RemoteInterface,
		repoKey string,
		imageName string,
		acceptHeader []string,
		ifNoneMatchHeader []string,
	) (manifest.Manifest, error)
	// HeadManifest send manifest head request to the remote server
	HeadManifest(ctx context.Context, art pkg.RegistryInfo, remote RemoteInterface) (bool, *manifest.Descriptor, error)
	// EnsureTag ensure tag for digest
	EnsureTag(
		ctx context.Context,
		rsHeaders *commons.ResponseHeaders,
		info pkg.RegistryInfo,
		acceptHeader []string,
		ifNoneMatchHeader []string,
	) error
}

type controller struct {
	localRegistry           registryInterface
	localManifestRegistry   registryManifestInterface
	secretService           secret.Service
	spaceFinder             refcache.SpaceFinder
	manifestCacheHandlerMap map[string]ManifestCacheHandler
}

// NewProxyController -- get the proxy controller instance.
func NewProxyController(
	l registryInterface, lm registryManifestInterface, secretService secret.Service,
	spaceFinder refcache.SpaceFinder, manifestCacheHandlerMap map[string]ManifestCacheHandler,
) Controller {
	return &controller{
		localRegistry:           l,
		localManifestRegistry:   lm,
		secretService:           secretService,
		spaceFinder:             spaceFinder,
		manifestCacheHandlerMap: manifestCacheHandlerMap,
	}
}

func (c *controller) EnsureTag(
	ctx context.Context,
	rsHeaders *commons.ResponseHeaders,
	info pkg.RegistryInfo,
	acceptHeader []string,
	ifNoneMatchHeader []string,
) error {
	// search the digest in cache and query with trimmed digest

	_, desc, mfst, err := c.localRegistry.PullManifest(ctx, info, acceptHeader, ifNoneMatchHeader)
	if len(err) > 0 {
		return err[0]
	}

	// Fixme: Need to properly pick tag.
	e := c.localManifestRegistry.DBTag(ctx, mfst, desc.Digest, info.Reference, rsHeaders, info)
	if e != nil {
		log.Error().Err(e).Msgf("Error in ensuring tag: %s", e)
	}
	return e
}

func (c *controller) UseLocalBlob(ctx context.Context, art pkg.RegistryInfo) bool {
	if len(art.Digest) == 0 {
		return false
	}
	// TODO: Get from Local storage.
	_, _, _, _, _, e := c.localRegistry.GetBlob(ctx, art)
	return len(e) == 0
}

// ManifestList ...
type ManifestList struct {
	Content     []byte
	Digest      string
	ContentType string
}

// UseLocalManifest check if these manifest could be found in localRegistry registry,
// the return error should be nil when it is not found in localRegistry and
// need to delegate to remote registry
// the return error should be NotFoundError when it is not found in remote registry
// the error will be captured by framework and return 404 to client.
func (c *controller) UseLocalManifest(
	ctx context.Context,
	art pkg.RegistryInfo,
	remote RemoteInterface,
	acceptHeaders []string,
	ifNoneMatchHeader []string,
) (bool, *ManifestList, error) {
	// TODO: get from DB
	_, d, man, e := c.localRegistry.PullManifest(ctx, art, acceptHeaders, ifNoneMatchHeader)
	if len(e) > 0 {
		return false, nil, nil
	}

	remoteRepo := getRemoteRepo(art)
	exist, desc, err := remote.ManifestExist(remoteRepo, getReference(art)) // HEAD.
	// TODO: Check for rate limit error.
	if err != nil {
		if errors.IsRateLimitError(err) { // if rate limit, use localRegistry if it exists, otherwise return error.
			log.Ctx(ctx).Warn().Msgf("Rate limit error: %v", err)
			return true, nil, nil
		}
		log.Ctx(ctx).Warn().Msgf("Error in checking remote manifest exist: %v", err)
		return false, nil, err
	}
	log.Info().Msgf("Manifest exist: %t %s %d %s", exist, desc.Digest.String(), desc.Size, desc.MediaType)

	// TODO: Delete if does not exist on remote. Validate this
	if !exist || desc == nil {
		go func() {
			c.localRegistry.DeleteManifest(ctx, art)
		}()
		return false, nil, errors.NotFoundError(fmt.Errorf("registry %v, tag %v not found", art.RegIdentifier, art.Tag))
	}

	log.Info().Msgf("Manifest: %s", getReference(art))
	mediaType, payload, _ := man.Payload()

	return true, &ManifestList{payload, d.Digest.String(), mediaType}, nil
}

func ByteToReadCloser(b []byte) io.ReadCloser {
	reader := bytes.NewReader(b)
	readCloser := io.NopCloser(reader)
	return readCloser
}

func (c *controller) ProxyManifest(
	ctx context.Context,
	art pkg.RegistryInfo,
	remote RemoteInterface,
	repoKey string,
	imageName string,
	acceptHeader []string,
	ifNoneMatchHeader []string,
) (manifest.Manifest, error) {
	var man manifest.Manifest
	remoteRepo := getRemoteRepo(art)
	ref := getReference(art)
	man, dig, err := remote.Manifest(remoteRepo, ref)
	if err != nil {
		if errors.IsNotFoundErr(err) {
			log.Info().Msgf("TODO: Delete manifest %s from localRegistry registry", dig)
			// go func() {
			//	c.localRegistry.DeleteManifest(remoteRepo, art.Tag)
			// }()
		}
		return man, err
	}
	ct, _, err := man.Payload()
	log.Info().Msgf("Content type: %s", ct)
	if err != nil {
		return man, err
	}

	// This GoRoutine is to push the manifest from Remote to Local registry.
	go func(_, ct string) {
		session, _ := request.AuthSessionFrom(ctx)
		ctx2 := request.WithAuthSession(ctx, session)
		ctx2 = context.WithoutCancel(ctx2)
		ctx2 = context.WithValue(ctx2, cfg.GoRoutineKey, "UpdateManifest")
		var count = 0
		for n := 0; n < maxManifestWait; n++ {
			time.Sleep(sleepIntervalSec * time.Second)
			count++
			log.Ctx(ctx2).Info().Msgf("Current retry=%v artifact: %v:%v, digest: %s",
				count, repoKey, imageName,
				art.Digest)
			_, des, _, e := c.localRegistry.PullManifest(ctx2, art, acceptHeader, ifNoneMatchHeader)
			if len(e) > 0 {
				log.Ctx(ctx2).Info().Stack().Err(err).Msgf("Local manifest doesn't exist, error %v", e[0])
			}
			// Push manifest to localRegistry when pull with digest, or artifact not found, or digest mismatch.
			errs := []error{}
			if len(art.Tag) == 0 || e != nil || des.Digest.String() != dig {
				artInfo := art
				if len(artInfo.Digest) == 0 {
					artInfo.Digest = dig
				}
				err = c.waitAndPushManifest(ctx2, art, ct, man)
				if err != nil {
					continue
				}
			}

			// Query artifact after push.
			if e == nil || commons.IsEmpty(errs) {
				_, _, _, err := c.localRegistry.PullManifest(ctx2, art, acceptHeader, ifNoneMatchHeader)
				if err != nil {
					log.Ctx(ctx2).Error().Stack().Msgf("failed to get manifest, error %v", err)
				} else {
					log.Ctx(ctx2).Info().Msgf(
						"Completed manifest push to localRegistry registry. Image: %s, Tag: %s, Digest: %s",
						art.Image, art.Tag, art.Digest,
					)
					return
				}
			}
			// if e != nil {
			// TODO: Place to send events
			// SendPullEvent(bCtx, a, art.Tag, operator)
			// }
		}
	}("System", ct)

	return man, nil
}

func (c *controller) HeadManifest(
	_ context.Context,
	art pkg.RegistryInfo,
	remote RemoteInterface,
) (bool, *manifest.Descriptor, error) {
	remoteRepo := getRemoteRepo(art)
	ref := getReference(art)
	return remote.ManifestExist(remoteRepo, ref)
}

func (c *controller) ProxyBlob(
	ctx context.Context, art pkg.RegistryInfo, repoKey string, proxy types.UpstreamProxy,
) (int64, io.ReadCloser, error) {
	rHelper, err := NewRemoteHelper(ctx, c.spaceFinder, c.secretService, repoKey, proxy)
	if err != nil {
		return 0, nil, err
	}

	art.Image, err = rHelper.GetImageName(ctx, c.spaceFinder, art.Image)
	if err != nil {
		return 0, nil, err
	}

	remoteImage := getRemoteRepo(art)
	log.Debug().Msgf("The blob doesn't exist, proxy the request to the target server, url:%v", remoteImage)

	size, bReader, err := rHelper.BlobReader(remoteImage, art.Digest)
	if err != nil {
		log.Error().Stack().Err(err).Msgf("failed to pull blob, error %v", err)
		return 0, nil, errcode.ErrorCodeBlobUnknown.WithDetail(art.Digest)
	}
	desc := manifest.Descriptor{Size: size, Digest: digest.Digest(art.Digest)}

	// This GoRoutine is to push the blob from Remote to Local registry. No retry logic is defined here.
	go func(art pkg.RegistryInfo) {
		// Cloning Context.
		session, ok := request.AuthSessionFrom(ctx)
		if !ok {
			log.Error().Stack().Err(err).Msg("failed to get auth session from context")
			return
		}
		ctx2 := request.WithAuthSession(ctx, session)
		ctx2 = context.WithoutCancel(ctx2)
		ctx2 = context.WithValue(ctx2, cfg.GoRoutineKey, "AddBlob")
		ctx2 = log.Ctx(ctx2).With().Logger().WithContext(ctx2)
		err := c.putBlobToLocal(ctx2, art, remoteImage, repoKey, desc, rHelper)
		if err != nil {
			log.Ctx(ctx2).Error().Stack().Err(err).
				Msgf("error while putting blob to localRegistry registry, %v", err)
			return
		}
		log.Ctx(ctx2).Info().Msgf("Successfully updated the cache for digest %s", art.Digest)
	}(art)
	return size, bReader, nil
}

func (c *controller) putBlobToLocal(
	ctx context.Context,
	art pkg.RegistryInfo,
	image string,
	localRepo string,
	desc manifest.Descriptor,
	r RemoteInterface,
) error {
	log.Debug().
		Msgf(
			"Put blob to localRegistry registry!, sourceRepo:%v, localRepo:%v, digest: %v", image, localRepo,
			desc.Digest,
		)
	cl, bReader, err := r.BlobReader(image, string(desc.Digest))
	if err != nil {
		log.Error().Stack().Err(err).Msgf("failed to create blob reader, error %v", err)
		return err
	}
	defer bReader.Close()
	headers, errs := c.localRegistry.InitBlobUpload(ctx, art, "", "")
	if len(errs) > 0 {
		log.Error().Stack().Err(err).Msgf("failed to init blob upload, error %v", errs)
		return errs[0]
	}

	location, uuid := headers.Headers["Location"], headers.Headers["Docker-Upload-UUID"]
	parsedURL, err := url.Parse(location)
	if err != nil {
		log.Error().Err(err).Msgf("Error parsing URL: %s", err)
		return err
	}
	stateToken := parsedURL.Query().Get("_state")
	art.SetReference(uuid)
	c.localRegistry.PushBlob(ctx, art, bReader, cl, stateToken)
	return err
}

func (c *controller) waitAndPushManifest(
	ctx context.Context, art pkg.RegistryInfo, contentType string, man manifest.Manifest,
) error {
	h, ok := c.manifestCacheHandlerMap[contentType]
	if !ok {
		h, ok = c.manifestCacheHandlerMap[DefaultHandler]
		if !ok {
			return fmt.Errorf("failed to get default manifest cache handler")
		}
	}
	err := h.CacheContent(ctx, art, contentType, man)
	if err != nil {
		log.Error().Stack().Err(err).Msgf("Error in caching manifest: %s", err)
		return err
	}
	return nil
}

func getRemoteRepo(art pkg.RegistryInfo) string {
	return art.Image
}

func getReference(art pkg.RegistryInfo) string {
	if len(art.Digest) > 0 {
		return art.Digest
	}
	return art.Tag
}

const DefaultHandler = "default"

// ManifestCache default Manifest handler.
type ManifestCache struct {
	localRegistry         registryInterface
	localManifestRegistry registryManifestInterface
}

func GetManifestCache(localRegistry registryInterface, localManifestRegistry registryManifestInterface) *ManifestCache {
	return &ManifestCache{
		localRegistry:         localRegistry,
		localManifestRegistry: localManifestRegistry,
	}
}

// ManifestListCache handle Manifest list type and index type.
type ManifestListCache struct {
	localRegistry registryInterface
}

func GetManifestListCache(localRegistry registryInterface) *ManifestListCache {
	return &ManifestListCache{localRegistry: localRegistry}
}

// ManifestCacheHandler define how to cache manifest content.
type ManifestCacheHandler interface {
	// CacheContent - cache the content of the manifest
	CacheContent(ctx context.Context, art pkg.RegistryInfo, contentType string, m manifest.Manifest) error
}

func (m *ManifestCache) CacheContent(
	ctx context.Context, art pkg.RegistryInfo, contentType string, man manifest.Manifest,
) error {
	_, payload, err := man.Payload()
	if err != nil {
		return err
	}
	// Push manifest to localRegistry.
	_, errs := m.localRegistry.PutManifest(ctx, art, contentType, ByteToReadCloser(payload), int64(len(payload)))
	if len(errs) > 0 {
		return errs[0]
	}

	for n := 0; n < maxManifestMappingWait; n++ {
		time.Sleep(maxManifestMappingIntervalSec * time.Second)
		err = m.localManifestRegistry.AddManifestAssociation(ctx, art.RegIdentifier, digest.Digest(art.Digest), art)
		if err != nil {
			log.Error().Stack().Err(err).Msgf("failed to add manifest association, error %v", err)
			continue
		}
		return nil
	}
	log.Ctx(ctx).Info().Msgf("Successfully cached manifest for image: %s, tag: %s, digest: %s",
		art.Image, art.Tag, art.Digest)
	return err
}

func (m *ManifestListCache) CacheContent(
	ctx context.Context, art pkg.RegistryInfo, contentType string, man manifest.Manifest,
) error {
	_, payload, err := man.Payload()
	if err != nil {
		log.Error().Msg("failed to get payload")
		return err
	}
	if len(getReference(art)) == 0 {
		log.Error().Msg("failed to get reference, reference is empty, skip to cache manifest list")
		return fmt.Errorf("failed to get reference, reference is empty, skip to cache manifest list")
	}
	// cache key should contain digest if digest exist
	if len(art.Digest) == 0 {
		art.Digest = string(digest.FromBytes(payload))
	}

	if err = m.push(ctx, art, man, contentType); err != nil {
		log.Error().Msgf("error when push manifest list to local :%v", err)
		return err
	}
	log.Ctx(ctx).Info().Msgf("Successfully cached manifest list for image: %s, tag: %s, digest: %s",
		art.Image, art.Tag, art.Digest)
	return nil
}

func (m *ManifestListCache) push(
	ctx context.Context, art pkg.RegistryInfo, man manifest.Manifest, contentType string,
) error {
	if len(man.References()) == 0 {
		return errors.New("manifest list doesn't contain any pushed manifest")
	}
	_, pl, err := man.Payload()
	if err != nil {
		log.Error().Msgf("failed to get payload, error %v", err)
		return err
	}
	log.Debug().Msgf("The manifest list payload: %v", string(pl))
	newDig := digest.FromBytes(pl)
	// Because the manifest list maybe updated, need to recheck if it is exist in local
	_, descriptor, manifest2, _ := m.localRegistry.PullManifest(ctx, art, nil, nil)
	if manifest2 != nil && descriptor.Digest == newDig {
		return nil
	}

	_, errs := m.localRegistry.PutManifest(ctx, art, contentType, ByteToReadCloser(pl), int64(len(pl)))
	if len(errs) > 0 {
		return errs[0]
	}
	return nil
}
