// Source: https://gitlab.com/gitlab-org/container-registry

// Copyright 2019 Gitlab Inc.
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
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/harness/gitness/app/auth/authz"
	corestore "github.com/harness/gitness/app/store"
	"github.com/harness/gitness/registry/app/dist_temp/errcode"
	"github.com/harness/gitness/registry/app/pkg"
	"github.com/harness/gitness/registry/app/pkg/commons"
	"github.com/harness/gitness/registry/app/storage"
	"github.com/harness/gitness/registry/app/store"
	registrytypes "github.com/harness/gitness/registry/types"
	"github.com/harness/gitness/types/enum"

	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/rs/zerolog/log"
)

type Controller struct {
	*pkg.CoreController
	local      *LocalRegistry
	remote     *RemoteRegistry
	spaceStore corestore.SpaceStore
	authorizer authz.Authorizer
	DBStore    *DBStore
}

type DBStore struct {
	BlobRepo         store.BlobRepository
	ImageDao         store.ImageRepository
	ArtifactDao      store.ArtifactRepository
	BandwidthStatDao store.BandwidthStatRepository
	DownloadStatDao  store.DownloadStatRepository
}

type TagsAPIResponse struct {
	Name string   `json:"name"`
	Tags []string `json:"tags"`
}

var _ pkg.Artifact = (*LocalRegistry)(nil)
var _ pkg.Artifact = (*RemoteRegistry)(nil)

func NewController(
	local *LocalRegistry,
	remote *RemoteRegistry,
	coreController *pkg.CoreController,
	spaceStore corestore.SpaceStore,
	authorizer authz.Authorizer,
	dBStore *DBStore,
) *Controller {
	c := &Controller{
		CoreController: coreController,
		local:          local,
		remote:         remote,
		spaceStore:     spaceStore,
		authorizer:     authorizer,
		DBStore:        dBStore,
	}

	pkg.TypeRegistry[pkg.LocalRegistry] = local
	pkg.TypeRegistry[pkg.RemoteRegistry] = remote
	return c
}

func NewDBStore(
	blobRepo store.BlobRepository,
	imageDao store.ImageRepository,
	artifactDao store.ArtifactRepository,
	bandwidthStatDao store.BandwidthStatRepository,
	downloadStatDao store.DownloadStatRepository,
) *DBStore {
	return &DBStore{
		BlobRepo:         blobRepo,
		ImageDao:         imageDao,
		ArtifactDao:      artifactDao,
		BandwidthStatDao: bandwidthStatDao,
		DownloadStatDao:  downloadStatDao,
	}
}

const (
	ResourceTypeBlob     = "blob"
	ResourceTypeManifest = "manifest"
)

func (c *Controller) ProxyWrapper(
	ctx context.Context,
	f func(registry registrytypes.Registry, imageName string, artInfo pkg.Artifact) Response,
	info pkg.RegistryInfo,
	resourceType string,
) Response {
	none := pkg.RegistryInfo{}
	if info == none {
		log.Ctx(ctx).Error().Stack().Msg("artifactinfo is not found")
		return nil
	}

	var response Response
	requestRepoKey := info.RegIdentifier
	imageName := info.Image
	if repos, err := c.GetOrderedRepos(ctx, requestRepoKey, *info.BaseInfo); err == nil {
		for _, registry := range repos {
			log.Ctx(ctx).Info().Msgf("Using Repository: %s, Type: %s", registry.Name, registry.Type)
			artifact, ok := c.GetArtifact(registry).(Registry)
			if !ok {
				log.Ctx(ctx).Warn().Msgf("artifact %s is not a registry", registry.Name)
				continue
			}
			if artifact != nil {
				response = f(registry, imageName, artifact)
				if pkg.IsEmpty(response.GetErrors()) {
					return response
				}
				log.Ctx(ctx).Warn().Msgf("Repository: %s, Type: %s, errors: %v", registry.Name, registry.Type,
					response.GetErrors())
			}
		}
	}
	if response != nil && !pkg.IsEmpty(response.GetErrors()) {
		switch resourceType {
		case ResourceTypeManifest:
			response.SetError(errcode.ErrCodeManifestUnknown)
		case ResourceTypeBlob:
			response.SetError(errcode.ErrCodeBlobUnknown)
		default:
			// do nothing
		}
	}
	return response
}

func (c *Controller) HeadManifest(
	ctx context.Context,
	art pkg.RegistryInfo,
	acceptHeaders []string,
	ifNoneMatchHeader []string,
) Response {
	err := pkg.GetRegistryCheckAccess(
		ctx, c.RegistryDao, c.authorizer, c.spaceStore, art.RegIdentifier, art.ParentID,
		enum.PermissionArtifactsDownload,
	)
	if err != nil {
		return &GetManifestResponse{
			Errors: []error{errcode.ErrCodeDenied},
		}
	}

	f := func(registry registrytypes.Registry, imageName string, a pkg.Artifact) Response {
		art.SetRepoKey(registry.Name)
		art.ParentID = registry.ParentID
		// Need to reassign original imageName to art because we are updating image name based on upstream proxy source inside
		art.Image = imageName
		headers, desc, man, e := a.(Registry).ManifestExist(ctx, art, acceptHeaders, ifNoneMatchHeader)
		response := &GetManifestResponse{e, headers, desc, man}
		return response
	}

	result := c.ProxyWrapper(ctx, f, art, ResourceTypeManifest)
	return result
}

func (c *Controller) PullManifest(
	ctx context.Context,
	art pkg.RegistryInfo,
	acceptHeaders []string,
	ifNoneMatchHeader []string,
) Response {
	err := pkg.GetRegistryCheckAccess(
		ctx, c.RegistryDao, c.authorizer, c.spaceStore, art.RegIdentifier, art.ParentID,
		enum.PermissionArtifactsDownload,
	)
	if err != nil {
		return &GetManifestResponse{
			Errors: []error{errcode.ErrCodeDenied},
		}
	}
	f := func(registry registrytypes.Registry, imageName string, a pkg.Artifact) Response {
		art.SetRepoKey(registry.Name)
		art.ParentID = registry.ParentID
		// Need to reassign original imageName to art because we are updating image name based on upstream proxy source inside
		art.Image = imageName
		headers, desc, man, e := a.(Registry).PullManifest(ctx, art, acceptHeaders, ifNoneMatchHeader)
		response := &GetManifestResponse{e, headers, desc, man}
		return response
	}

	result := c.ProxyWrapper(ctx, f, art, ResourceTypeManifest)
	return result
}

func (c *Controller) PutManifest(
	ctx context.Context,
	artInfo pkg.RegistryInfo,
	mediaType string,
	body io.ReadCloser,
	length int64,
) (responseHeaders *commons.ResponseHeaders, errs []error) {
	err := pkg.GetRegistryCheckAccess(
		ctx, c.RegistryDao, c.authorizer, c.spaceStore, artInfo.RegIdentifier,
		artInfo.ParentID, enum.PermissionArtifactsUpload, enum.PermissionArtifactsDownload,
	)
	if err != nil {
		return nil, []error{errcode.ErrCodeDenied}
	}
	return c.local.PutManifest(ctx, artInfo, mediaType, body, length)
}

func (c *Controller) DeleteManifest(
	ctx context.Context,
	artInfo pkg.RegistryInfo,
) (errs []error, responseHeaders *commons.ResponseHeaders) {
	err := pkg.GetRegistryCheckAccess(
		ctx, c.RegistryDao, c.authorizer, c.spaceStore, artInfo.RegIdentifier, artInfo.ParentID,
		enum.PermissionArtifactsDelete,
	)
	if err != nil {
		return []error{errcode.ErrCodeDenied}, nil
	}
	return c.local.DeleteManifest(ctx, artInfo)
}

func (c *Controller) HeadBlob(
	ctx context.Context,
	info pkg.RegistryInfo,
) (
	responseHeaders *commons.ResponseHeaders, fr *storage.FileReader, size int64, readCloser io.ReadCloser,
	redirectURL string, errs []error,
) {
	err := pkg.GetRegistryCheckAccess(
		ctx, c.RegistryDao, c.authorizer, c.spaceStore, info.RegIdentifier, info.ParentID,
		enum.PermissionArtifactsDownload,
	)
	if err != nil {
		return nil, nil, 0, nil, "", []error{errcode.ErrCodeDenied}
	}
	return c.local.HeadBlob(ctx, info)
}

func (c *Controller) GetBlob(ctx context.Context, info pkg.RegistryInfo) Response {
	err := pkg.GetRegistryCheckAccess(
		ctx, c.RegistryDao, c.authorizer, c.spaceStore, info.RegIdentifier, info.ParentID,
		enum.PermissionArtifactsDownload,
	)
	if err != nil {
		return &GetBlobResponse{
			Errors: []error{errcode.ErrCodeDenied},
		}
	}
	f := func(registry registrytypes.Registry, imageName string, a pkg.Artifact) Response {
		info.SetRepoKey(registry.Name)
		info.ParentID = registry.ParentID
		info.Image = imageName
		headers, body, size, readCloser, redirectURL, errs := a.(Registry).GetBlob(ctx, info)
		return &GetBlobResponse{errs, headers, body, size, readCloser, redirectURL}
	}

	return c.ProxyWrapper(ctx, f, info, ResourceTypeBlob)
}

func (c *Controller) InitiateUploadBlob(
	ctx context.Context,
	info pkg.RegistryInfo,
	fromImageRef string,
	mountDigest string,
) (*commons.ResponseHeaders, []error) {
	err := pkg.GetRegistryCheckAccess(
		ctx, c.RegistryDao, c.authorizer, c.spaceStore, info.RegIdentifier, info.ParentID,
		enum.PermissionArtifactsUpload,
		enum.PermissionArtifactsDownload,
	)
	if err != nil {
		return nil, []error{errcode.ErrCodeDenied}
	}
	return c.local.InitBlobUpload(ctx, info, fromImageRef, mountDigest)
}

func (c *Controller) GetUploadBlobStatus(
	ctx context.Context,
	info pkg.RegistryInfo,
	token string,
) (responseHeaders *commons.ResponseHeaders, errs []error) {
	err := pkg.GetRegistryCheckAccess(
		ctx, c.RegistryDao, c.authorizer, c.spaceStore, info.RegIdentifier, info.ParentID,
		enum.PermissionArtifactsDownload,
	)
	if err != nil {
		return nil, []error{errcode.ErrCodeDenied}
	}
	blobCtx := c.local.App.GetBlobsContext(ctx, info)
	return c.local.GetBlobUploadStatus(blobCtx, info, token)
}

func (c *Controller) PatchBlobUpload(
	ctx context.Context,
	info pkg.RegistryInfo,
	ct string,
	cr string,
	cl string,
	length int64,
	token string,
	body io.ReadCloser,
) (responseHeaders *commons.ResponseHeaders, errors []error) {
	blobCtx := c.local.App.GetBlobsContext(ctx, info)
	err := pkg.GetRegistryCheckAccess(
		ctx, c.RegistryDao, c.authorizer, c.spaceStore, info.RegIdentifier, info.ParentID,
		enum.PermissionArtifactsDownload,
		enum.PermissionArtifactsUpload,
	)
	if err != nil {
		return nil, []error{errcode.ErrCodeDenied}
	}
	errors = make([]error, 0)
	if blobCtx.UUID != "" {
		errs := ResumeBlobUpload(blobCtx, token)
		errors = append(errors, errs...)
	}
	if blobCtx.Upload != nil {
		defer blobCtx.Upload.Close()
	}

	rs, errs := c.local.PushBlobChunk(blobCtx, info, ct, cr, cl, body, length)
	if !commons.IsEmpty(errs) {
		errors = append(errors, errs...)
	}
	return rs, errors
}

func (c *Controller) CompleteBlobUpload(
	ctx context.Context,
	info pkg.RegistryInfo,
	body io.ReadCloser,
	length int64,
	stateToken string,
) (responseHeaders *commons.ResponseHeaders, errs []error) {
	err := pkg.GetRegistryCheckAccess(
		ctx, c.RegistryDao, c.authorizer, c.spaceStore, info.RegIdentifier, info.ParentID,
		enum.PermissionArtifactsUpload,
		enum.PermissionArtifactsDownload,
	)
	if err != nil {
		return nil, []error{errcode.ErrCodeDenied}
	}
	return c.local.PushBlob(ctx, info, body, length, stateToken)
}

func (c *Controller) CancelBlobUpload(
	ctx context.Context,
	info pkg.RegistryInfo,
	stateToken string,
) (responseHeaders *commons.ResponseHeaders, errors []error) {
	err := pkg.GetRegistryCheckAccess(
		ctx, c.RegistryDao, c.authorizer, c.spaceStore, info.RegIdentifier, info.ParentID,
		enum.PermissionArtifactsDelete,
	)
	if err != nil {
		return nil, []error{errcode.ErrCodeDenied}
	}

	blobCtx := c.local.App.GetBlobsContext(ctx, info)

	errors = make([]error, 0)

	if blobCtx.UUID != "" {
		errs := ResumeBlobUpload(blobCtx, stateToken)
		errors = append(errors, errs...)
	}

	if blobCtx.Upload == nil {
		e := errcode.ErrCodeBlobUploadUnknown
		errors = append(errors, e)
		return responseHeaders, errors
	}
	defer blobCtx.Upload.Close()

	responseHeaders = &commons.ResponseHeaders{
		Headers: map[string]string{"Docker-Upload-UUID": blobCtx.UUID},
	}

	if err := blobCtx.Upload.Cancel(blobCtx); err != nil {
		log.Ctx(ctx).Error().Stack().Err(err).Msgf("error encountered canceling upload: %v", err)
		errors = append(errors, errcode.ErrCodeUnknown.WithDetail(err))
	}

	responseHeaders.Code = http.StatusNoContent
	return responseHeaders, errors
}

func (c *Controller) DeleteBlob(
	ctx context.Context,
	info pkg.RegistryInfo,
) (responseHeaders *commons.ResponseHeaders, errs []error) {
	err := pkg.GetRegistryCheckAccess(
		ctx, c.RegistryDao, c.authorizer, c.spaceStore, info.RegIdentifier, info.ParentID,
		enum.PermissionArtifactsDelete,
	)
	if err != nil {
		return nil, []error{errcode.ErrCodeDenied}
	}
	blobCtx := c.local.App.GetBlobsContext(ctx, info)
	return c.local.DeleteBlob(blobCtx, info)
}

func (c *Controller) GetTags(
	ctx context.Context,
	lastEntry string,
	maxEntries int,
	origURL string,
	artInfo pkg.RegistryInfo,
) (*commons.ResponseHeaders, []string, error) {
	err := pkg.GetRegistryCheckAccess(
		ctx, c.RegistryDao, c.authorizer, c.spaceStore, artInfo.RegIdentifier,
		artInfo.ParentID, enum.PermissionArtifactsDownload,
	)
	if err != nil {
		return nil, nil, errcode.ErrCodeDenied
	}
	return c.local.ListTags(ctx, lastEntry, maxEntries, origURL, artInfo)
}

func (c *Controller) GetCatalog(_ http.ResponseWriter, _ *http.Request) {
	log.Info().Msgf("Not implemented yet!")
}

func (c *Controller) GetReferrers(
	ctx context.Context,
	artInfo pkg.RegistryInfo,
	artifactType string,
) (index *v1.Index, responseHeaders *commons.ResponseHeaders, err error) {
	accessErr := pkg.GetRegistryCheckAccess(
		ctx, c.RegistryDao, c.authorizer, c.spaceStore, artInfo.RegIdentifier,
		artInfo.ParentID, enum.PermissionArtifactsDownload,
	)
	if accessErr != nil {
		return nil, nil, errcode.ErrCodeDenied
	}
	return c.local.ListReferrers(ctx, artInfo, artifactType)
}

func ResumeBlobUpload(ctx *Context, stateToken string) []error {
	var errs []error
	state, err := hmacKey(ctx.App.Config.Registry.HTTP.Secret).unpackUploadState(stateToken)
	if err != nil {
		log.Ctx(ctx).Info().Msgf("error resolving upload: %v", err)
		errs = append(errs, errcode.ErrCodeBlobUploadInvalid.WithDetail(err))
		return errs
	}
	ctx.State = state

	if state.Path != ctx.OciBlobStore.Path() {
		log.Ctx(ctx).Info().Msgf("mismatched path in upload state: %q != %q", state.Path, ctx.OciBlobStore.Path())
		errs = append(errs, errcode.ErrCodeBlobUploadInvalid.WithDetail(err))
		return errs
	}

	if state.UUID != ctx.UUID {
		log.Ctx(ctx).Info().Msgf("mismatched uuid in upload state: %q != %q", state.UUID, ctx.UUID)
		errs = append(errs, errcode.ErrCodeBlobUploadInvalid.WithDetail(err))
		return errs
	}

	blobs := ctx.OciBlobStore
	upload, err := blobs.Resume(ctx.Context, ctx.UUID)
	if err != nil {
		log.Ctx(ctx).Error().Stack().Err(err).Msgf("error resolving upload: %v", err)
		if errors.Is(err, storage.ErrBlobUploadUnknown) {
			errs = append(errs, errcode.ErrCodeBlobUploadUnknown.WithDetail(err))
			return errs
		}

		errs = append(errs, errcode.ErrCodeUnknown.WithDetail(err))
		return errs
	}
	ctx.Upload = upload

	if size := upload.Size(); size != ctx.State.Offset {
		log.Ctx(ctx).Error().Stack().Err(err).Msgf("upload resumed at wrong offset: %d != %d", size, ctx.State.Offset)
		errs = append(errs, errcode.ErrCodeRangeInvalid.WithDetail(err))
		return errs
	}
	return errs
}

// unpackUploadState unpacks and validates the blob upload state from the
// token, using the hmacKey secret.
func (secret hmacKey) unpackUploadState(token string) (BlobUploadState, error) {
	var state BlobUploadState

	tokenBytes, err := base64.URLEncoding.DecodeString(token)
	if err != nil {
		return state, fmt.Errorf("failed to decode token: %w", err)
	}
	mac := hmac.New(sha256.New, []byte(secret))

	if len(tokenBytes) < mac.Size() {
		return state, errInvalidSecret
	}

	macBytes := tokenBytes[:mac.Size()]
	messageBytes := tokenBytes[mac.Size():]

	mac.Write(messageBytes)
	if !hmac.Equal(mac.Sum(nil), macBytes) {
		return state, errInvalidSecret
	}

	if err := json.Unmarshal(messageBytes, &state); err != nil {
		return state, err
	}

	return state, nil
}
