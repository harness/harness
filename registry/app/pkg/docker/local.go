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
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/harness/gitness/app/paths"
	"github.com/harness/gitness/registry/app/api/openapi/contracts/artifact"
	"github.com/harness/gitness/registry/app/dist_temp/dcontext"
	"github.com/harness/gitness/registry/app/dist_temp/errcode"
	"github.com/harness/gitness/registry/app/event"
	"github.com/harness/gitness/registry/app/manifest"
	"github.com/harness/gitness/registry/app/manifest/manifestlist"
	"github.com/harness/gitness/registry/app/manifest/ocischema"
	"github.com/harness/gitness/registry/app/manifest/schema2"
	"github.com/harness/gitness/registry/app/pkg"
	"github.com/harness/gitness/registry/app/pkg/commons"
	"github.com/harness/gitness/registry/app/storage"
	"github.com/harness/gitness/registry/app/store"
	"github.com/harness/gitness/registry/app/store/database/util"
	"github.com/harness/gitness/registry/gc"
	"github.com/harness/gitness/registry/types"
	store2 "github.com/harness/gitness/store"
	"github.com/harness/gitness/store/database/dbtx"
	gitnesstypes "github.com/harness/gitness/types"

	"github.com/distribution/distribution/v3"
	"github.com/distribution/reference"
	"github.com/opencontainers/go-digest"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/rs/zerolog/log"
)

const (
	defaultArch = "amd64"
	defaultOS   = "linux"
	imageClass  = "image"
)

type storageType int

const (
	manifestSchema2     storageType = iota // 0
	manifestlistSchema                     // 1
	ociSchema                              // 2
	ociImageIndexSchema                    // 3
	numStorageTypes                        // 4
)

const (
	manifestListCreateGCReviewWindow = 1 * time.Hour
	manifestListCreateGCLockTimeout  = 10 * time.Second
	manifestTagGCLockTimeout         = 30 * time.Second
	tagDeleteGCLockTimeout           = 10 * time.Second
	manifestTagGCReviewWindow        = 1 * time.Hour
	manifestDeleteGCReviewWindow     = 1 * time.Hour
	manifestDeleteGCLockTimeout      = 10 * time.Second
	blobExistsGCLockTimeout          = 10 * time.Second
	blobExistsGCReviewWindow         = 1 * time.Hour
	DefaultMaximumReturnedEntries    = 100
)

const (
	ReferrersSchemaVersion    = 2
	ReferrersMediaType        = "application/vnd.oci.image.index.v1+json"
	ArtifactTypeLocalRegistry = "Local Registry"
)

type CatalogAPIResponse struct {
	Repositories []string `json:"repositories"`
}

type S3Store interface {
}

var errInvalidSecret = fmt.Errorf("invalid secret")

type hmacKey string

func NewLocalRegistry(
	app *App, ms ManifestService, manifestDao store.ManifestRepository,
	registryDao store.RegistryRepository, registryBlobDao store.RegistryBlobRepository,
	blobRepo store.BlobRepository, mtRepository store.MediaTypesRepository,
	tagDao store.TagRepository, imageDao store.ImageRepository, artifactDao store.ArtifactRepository,
	bandwidthStatDao store.BandwidthStatRepository, downloadStatDao store.DownloadStatRepository,
	gcService gc.Service, tx dbtx.Transactor, reporter event.Reporter,
) Registry {
	return &LocalRegistry{
		App:              app,
		ms:               ms,
		registryDao:      registryDao,
		manifestDao:      manifestDao,
		registryBlobDao:  registryBlobDao,
		blobRepo:         blobRepo,
		mtRepository:     mtRepository,
		tagDao:           tagDao,
		imageDao:         imageDao,
		artifactDao:      artifactDao,
		bandwidthStatDao: bandwidthStatDao,
		downloadStatDao:  downloadStatDao,
		gcService:        gcService,
		tx:               tx,
		reporter:         reporter,
	}
}

type LocalRegistry struct {
	App              *App
	ms               ManifestService
	registryDao      store.RegistryRepository
	manifestDao      store.ManifestRepository
	registryBlobDao  store.RegistryBlobRepository
	blobRepo         store.BlobRepository
	mtRepository     store.MediaTypesRepository
	tagDao           store.TagRepository
	imageDao         store.ImageRepository
	artifactDao      store.ArtifactRepository
	bandwidthStatDao store.BandwidthStatRepository
	downloadStatDao  store.DownloadStatRepository
	gcService        gc.Service
	tx               dbtx.Transactor
	reporter         event.Reporter
}

func (r *LocalRegistry) Base() error {
	return nil
}

func (r *LocalRegistry) CanBeMount() (mount bool, repository string, err error) {
	// TODO implement me
	panic("implement me")
}

func (r *LocalRegistry) GetArtifactType() artifact.RegistryType {
	return artifact.RegistryTypeVIRTUAL
}

func (r *LocalRegistry) GetPackageTypes() []artifact.PackageType {
	return []artifact.PackageType{artifact.PackageTypeDOCKER, artifact.PackageTypeHELM}
}

func (r *LocalRegistry) getManifest(
	ctx context.Context,
	manifestDigest digest.Digest,
	repoKey string,
	imageName string,
	info pkg.RegistryInfo,
) (manifest.Manifest, error) {
	dbRepo, err := r.registryDao.GetByParentIDAndName(ctx, info.ParentID, repoKey)
	if err != nil {
		return nil, err
	}

	if dbRepo == nil {
		return nil, manifest.RegistryUnknownError{Name: repoKey}
	}

	log.Ctx(ctx).Debug().Msgf("getting manifest by digest from database")

	dig, _ := types.NewDigest(manifestDigest)
	// Find manifest by its digest
	dbManifest, err := r.manifestDao.FindManifestByDigest(ctx, dbRepo.ID, imageName, dig)
	if err != nil {
		if errors.Is(err, store2.ErrResourceNotFound) {
			return nil, manifest.UnknownRevisionError{
				Name:     repoKey,
				Revision: manifestDigest,
			}
		}
		return nil, err
	}

	return DBManifestToManifest(dbManifest)
}

func DBManifestToManifest(dbm *types.Manifest) (manifest.Manifest, error) {
	if dbm.SchemaVersion == 1 {
		return nil, manifest.ErrSchemaV1Unsupported
	}

	if dbm.SchemaVersion != 2 {
		return nil, fmt.Errorf("unrecognized manifest schema version %d", dbm.SchemaVersion)
	}

	mediaType := dbm.MediaType
	if dbm.NonConformant {
		// parse payload and get real media type
		var versioned manifest.Versioned
		if err := json.Unmarshal(dbm.Payload, &versioned); err != nil {
			return nil, fmt.Errorf("failed to unmarshal manifest payload: %w", err)
		}
		mediaType = versioned.MediaType
	}

	// This can be an image manifest or a manifest list
	switch mediaType {
	case schema2.MediaTypeManifest:
		m := &schema2.DeserializedManifest{}
		if err := m.UnmarshalJSON(dbm.Payload); err != nil {
			return nil, err
		}

		return m, nil
	case v1.MediaTypeImageManifest:
		m := &ocischema.DeserializedManifest{}
		if err := m.UnmarshalJSON(dbm.Payload); err != nil {
			return nil, err
		}

		return m, nil
	case manifestlist.MediaTypeManifestList, v1.MediaTypeImageIndex:
		m := &manifestlist.DeserializedManifestList{}
		if err := m.UnmarshalJSON(dbm.Payload); err != nil {
			return nil, err
		}

		return m, nil
	case "":
		// OCI image or image index - no media type in the content

		// First see if it looks like an image index
		resIndex := &manifestlist.DeserializedManifestList{}
		if err := resIndex.UnmarshalJSON(dbm.Payload); err != nil {
			return nil, err
		}
		if resIndex.Manifests != nil {
			return resIndex, nil
		}

		// Otherwise, assume it must be an image manifest
		m := &ocischema.DeserializedManifest{}
		if err := m.UnmarshalJSON(dbm.Payload); err != nil {
			return nil, err
		}

		return m, nil
	default:
		return nil,
			manifest.VerificationErrors{
				fmt.Errorf("unrecognized manifest content type %s", dbm.MediaType),
			}
	}
}

func (r *LocalRegistry) getTag(ctx context.Context, info pkg.RegistryInfo) (*manifest.Descriptor, error) {
	dbRepo, err := r.registryDao.GetByParentIDAndName(ctx, info.ParentID, info.RegIdentifier)

	if err != nil {
		return nil, err
	}

	if dbRepo == nil {
		return nil, manifest.RegistryUnknownError{Name: info.RegIdentifier}
	}

	log.Ctx(ctx).Info().Msgf("getting manifest by tag from database")
	dbManifest, err := r.manifestDao.FindManifestByTagName(ctx, dbRepo.ID, info.Image, info.Tag)
	if err != nil {
		// at the DB level a tag has a FK to manifests, so a tag cannot exist
		// unless it points to an existing manifest
		if errors.Is(err, store2.ErrResourceNotFound) {
			return nil, manifest.TagUnknownError{Tag: info.Tag}
		}
		return nil, err
	}

	return &manifest.Descriptor{Digest: dbManifest.Digest}, nil
}

func etagMatch(headers []string, etag string) bool {
	for _, headerVal := range headers {
		if headerVal == etag || headerVal == fmt.Sprintf(`"%s"`, etag) {
			// allow quoted or unquoted
			return true
		}
	}
	return false
}

// copyFullPayload copies the payload of an HTTP request to destWriter. If it
// receives less content than expected, and the client disconnected during the
// upload, it avoids sending a 400 error to keep the logs cleaner.
//
// The copy will be limited to `limit` bytes, if limit is greater than zero.
func copyFullPayload(
	ctx context.Context, length int64, body io.ReadCloser,
	destWriter io.Writer, action string,
) error {
	// Get a channel that tells us if the client disconnects
	clientClosed := ctx.Done()

	// Read in the data, if any.
	copied, err := io.Copy(destWriter, body)
	if clientClosed != nil && (err != nil || (length > 0 && copied < length)) {
		// Didn't receive as much content as expected. Did the client
		// disconnect during the request? If so, avoid returning a 400
		// error to keep the logs cleaner.
		select {
		case <-clientClosed:
			// Set the response Code to "499 Client Closed Request"
			// Even though the connection has already been closed,
			// this causes the logger to pick up a 499 error
			// instead of showing 0 for the HTTP status.
			// responseWriter.WriteHeader(499)

			dcontext.GetLoggerWithFields(
				ctx, log.Error(), map[interface{}]interface{}{
					"error":         err,
					"copied":        copied,
					"contentLength": length,
				}, "error", "copied", "contentLength",
			).Msg("client disconnected during " + action)
			return errors.New("client disconnected")
		default:
		}
	}

	if err != nil {
		dcontext.GetLogger(ctx, log.Error()).Msgf("unknown error reading request payload: %v", err)
		return err
	}

	return nil
}

func (r *LocalRegistry) HeadBlob(
	ctx2 context.Context,
	artInfo pkg.RegistryInfo,
) (
	responseHeaders *commons.ResponseHeaders, fr *storage.FileReader, size int64, readCloser io.ReadCloser,
	redirectURL string, errs []error,
) {
	return r.fetchBlobInternal(ctx2, http.MethodHead, artInfo)
}

func (r *LocalRegistry) GetBlob(
	ctx2 context.Context,
	artInfo pkg.RegistryInfo,
) (
	responseHeaders *commons.ResponseHeaders, fr *storage.FileReader, size int64,
	readCloser io.ReadCloser, redirectURL string, errs []error,
) {
	return r.fetchBlobInternal(ctx2, http.MethodGet, artInfo)
}

func (r *LocalRegistry) fetchBlobInternal(
	ctx2 context.Context, method string, info pkg.RegistryInfo,
) (*commons.ResponseHeaders, *storage.FileReader, int64, io.ReadCloser, string, []error) {
	ctx := r.App.GetBlobsContext(ctx2, info)

	responseHeaders := &commons.ResponseHeaders{
		Code:    0,
		Headers: make(map[string]string),
	}
	errs := make([]error, 0)
	var dgst digest.Digest
	blobs := ctx.OciBlobStore

	if err := r.dbBlobLinkExists(ctx, ctx.Digest, info.RegIdentifier, info); err != nil { //nolint:contextcheck
		errs = append(errs, errcode.FromUnknownError(err))
		return responseHeaders, nil, -1, nil, "", errs
	}
	dgst = ctx.Digest
	headers := make(map[string]string)
	//nolint:contextcheck
	fileReader, redirectURL, size, err := blobs.ServeBlobInternal(
		ctx.Context,
		info.RootIdentifier,
		dgst,
		headers,
		method,
	)
	if err != nil {
		if fileReader != nil {
			fileReader.Close()
		}
		if errors.Is(err, storage.ErrBlobUnknown) {
			errs = append(errs, errcode.ErrCodeBlobUnknown.WithDetail(ctx.Digest))
		} else {
			errs = append(errs, errcode.FromUnknownError(err))
		}
		return responseHeaders, nil, -1, nil, "", errs
	}
	if redirectURL != "" {
		return responseHeaders, nil, -1, nil, redirectURL, errs
	}

	for key, value := range headers {
		responseHeaders.Headers[key] = value
	}

	return responseHeaders, fileReader, size, nil, "", errs
}

func (r *LocalRegistry) PullManifest(
	ctx context.Context,
	artInfo pkg.RegistryInfo,
	acceptHeaders []string,
	ifNoneMatchHeader []string,
) (responseHeaders *commons.ResponseHeaders, descriptor manifest.Descriptor, manifest manifest.Manifest, errs []error) {
	responseHeaders, descriptor, manifest, errs = r.ManifestExist(ctx, artInfo, acceptHeaders, ifNoneMatchHeader)
	return responseHeaders, descriptor, manifest, errs
}

func (r *LocalRegistry) getDigestByTag(ctx context.Context, artInfo pkg.RegistryInfo) (digest.Digest, error) {
	desc, err := r.getTag(ctx, artInfo)
	if err != nil {
		var tagUnknownError manifest.TagUnknownError
		if errors.As(err, &tagUnknownError) {
			return "", errcode.ErrCodeManifestUnknown.WithDetail(err)
		}
		return "", err
	}
	return desc.Digest, nil
}

func getDigestFromInfo(artInfo pkg.RegistryInfo) digest.Digest {
	if artInfo.Digest != "" {
		return digest.Digest(artInfo.Digest)
	}
	return digest.Digest(artInfo.Reference)
}

func (r *LocalRegistry) getDigest(ctx context.Context, artInfo pkg.RegistryInfo) (digest.Digest, error) {
	if artInfo.Tag != "" {
		return r.getDigestByTag(ctx, artInfo)
	}
	return getDigestFromInfo(artInfo), nil
}

func (r *LocalRegistry) ManifestExist(
	ctx context.Context,
	artInfo pkg.RegistryInfo,
	acceptHeaders []string,
	ifNoneMatchHeader []string,
) (
	responseHeaders *commons.ResponseHeaders, descriptor manifest.Descriptor, manifestResult manifest.Manifest,
	errs []error,
) {
	tag := artInfo.Tag
	supports := r.getSupportsList(acceptHeaders)

	d, err := r.getDigest(ctx, artInfo)
	if err != nil {
		return responseHeaders, descriptor, manifestResult, []error{err}
	}

	if etagMatch(ifNoneMatchHeader, d.String()) {
		r2 := &commons.ResponseHeaders{
			Code: http.StatusNotModified,
		}
		return r2, manifest.Descriptor{Digest: d}, nil, nil
	}

	manifestResult, err = r.getManifest(ctx, d, artInfo.RegIdentifier, artInfo.Image, artInfo)
	if err != nil {
		var manifestUnknownRevisionError manifest.UnknownRevisionError
		if errors.As(err, &manifestUnknownRevisionError) {
			errs = append(errs, errcode.ErrCodeManifestUnknown.WithDetail(err))
		}
		return responseHeaders, descriptor, manifestResult, errs
	}
	// determine the type of the returned manifest
	manifestType := manifestSchema2
	manifestList, isManifestList := manifestResult.(*manifestlist.DeserializedManifestList)
	if _, isOCImanifest := manifestResult.(*ocischema.DeserializedManifest); isOCImanifest {
		manifestType = ociSchema
	} else if isManifestList {
		if manifestList.MediaType == manifestlist.MediaTypeManifestList {
			manifestType = manifestlistSchema
		} else if manifestList.MediaType == v1.MediaTypeImageIndex {
			manifestType = ociImageIndexSchema
		}
	}

	if manifestType == ociSchema && !supports[ociSchema] {
		errs = append(
			errs,
			errcode.ErrCodeManifestUnknown.WithMessage(
				"OCI manifest found, but accept header does not support OCI manifests",
			),
		)
		return responseHeaders, descriptor, manifestResult, errs
	}
	if manifestType == ociImageIndexSchema && !supports[ociImageIndexSchema] {
		errs = append(
			errs,
			errcode.ErrCodeManifestUnknown.WithMessage(
				"OCI index found, but accept header does not support OCI indexes",
			),
		)
		return responseHeaders, descriptor, manifestResult, errs
	}

	if tag != "" && manifestType == manifestlistSchema && !supports[manifestlistSchema] {
		d, manifestResult, err = r.rewriteManifest(ctx, artInfo, d, manifestList, supports)
		if err != nil {
			errs = append(errs, err)
			return responseHeaders, descriptor, manifestResult, errs
		}
	}

	ct, p, err := manifestResult.Payload()
	if err != nil {
		return responseHeaders, descriptor, manifestResult, errs
	}

	r2 := &commons.ResponseHeaders{
		Headers: map[string]string{
			"Content-Type":          ct,
			"Content-Length":        fmt.Sprint(len(p)),
			"Docker-Content-Digest": d.String(),
			"Etag":                  fmt.Sprintf(`"%s"`, d),
		},
	}

	return r2, manifest.Descriptor{Digest: d}, manifestResult, nil
}

func (r *LocalRegistry) rewriteManifest(
	ctx context.Context, artInfo pkg.RegistryInfo, d digest.Digest, manifestList *manifestlist.DeserializedManifestList,
	supports [4]bool,
) (digest.Digest, manifest.Manifest, error) {
	// Rewrite manifest in schema1 format
	log.Ctx(ctx).Info().Msgf(
		"rewriting manifest list %s in schema1 format to support old client", d.String(),
	)

	// Find the image manifest corresponding to the default
	// platform
	var manifestDigest digest.Digest
	for _, manifestDescriptor := range manifestList.Manifests {
		if manifestDescriptor.Platform.Architecture == defaultArch &&
			manifestDescriptor.Platform.OS == defaultOS {
			manifestDigest = manifestDescriptor.Digest
			break
		}
	}

	if manifestDigest == "" {
		return "", nil, errcode.ErrCodeManifestUnknown
	}

	manifestResult, err := r.getManifest(
		ctx, manifestDigest,
		artInfo.RegIdentifier, artInfo.Image, artInfo,
	)
	if err != nil {
		var manifestUnknownRevisionError manifest.UnknownRevisionError
		if errors.As(err, &manifestUnknownRevisionError) {
			return "", nil, errcode.ErrCodeManifestUnknown.WithDetail(err)
		}
		return "", nil, err
	}

	if _, isSchema2 := manifestResult.(*schema2.DeserializedManifest); isSchema2 && !supports[manifestSchema2] {
		return "", manifestResult, errcode.ErrCodeManifestInvalid.WithMessage("Schema 2 manifest not supported by client")
	}
	d = manifestDigest
	return d, manifestResult, nil
}

func (r *LocalRegistry) getSupportsList(acceptHeaders []string) [4]bool {
	var supports [numStorageTypes]bool
	// this parsing of Accept Headers is not quite as full-featured as
	// godoc.org's parser, but we don't care about "q=" values
	// https://github.com/golang/gddo/blob/
	// e91d4165076d7474d20abda83f92d15c7ebc3e81/httputil/header/header.go#L165-L202
	for _, acceptHeader := range acceptHeaders {
		// r.Header[...] is a slice in case the request contains
		//  the same header more than once
		// if the header isn't set, we'll get the zero value,
		// which "range" will handle gracefully

		// we need to split each header value on "," to get the full
		// list of "Accept" values (per RFC 2616)
		// https://www.w3.org/Protocols/rfc2616/rfc2616-sec14.html#sec14.1
		for _, mediaType := range strings.Split(acceptHeader, ",") {
			mediaType = strings.TrimSpace(mediaType)
			if _, _, err := mime.ParseMediaType(mediaType); err != nil {
				continue
			}

			switch mediaType {
			case schema2.MediaTypeManifest:
				supports[manifestSchema2] = true
			case manifestlist.MediaTypeManifestList:
				supports[manifestlistSchema] = true
			case v1.MediaTypeImageManifest:
				supports[ociSchema] = true
			case v1.MediaTypeImageIndex:
				supports[ociImageIndexSchema] = true
			}
		}
	}
	return supports
}

func (r *LocalRegistry) appendPutError(err error, errList []error) []error {
	// TODO: Move this error list inside the context

	if errors.Is(err, manifest.ErrUnsupported) {
		errList = append(errList, errcode.ErrCodeUnsupported)
		return errList
	}
	if errors.Is(err, manifest.ErrAccessDenied) {
		errList = append(errList, errcode.ErrCodeDenied)
		return errList
	}
	if errors.Is(err, manifest.ErrSchemaV1Unsupported) {
		errList = append(
			errList,
			errcode.ErrCodeManifestInvalid.WithDetail(
				"manifest type unsupported",
			),
		)
		return errList
	}
	if errors.Is(err, digest.ErrDigestInvalidFormat) {
		errList = append(errList, errcode.ErrCodeDigestInvalid.WithDetail(err))
		return errList
	}

	switch {
	case errors.As(err, &manifest.VerificationErrors{}):
		var verificationError manifest.VerificationErrors
		errors.As(err, &verificationError)
		for _, verificationError := range verificationError {
			switch {
			case errors.As(verificationError, &manifest.BlobUnknownError{}):
				var manifestBlobUnknownError manifest.BlobUnknownError
				errors.As(verificationError, &manifestBlobUnknownError)
				errList = append(
					errList, errcode.ErrCodeManifestBlobUnknown.WithDetail(
						manifestBlobUnknownError.Digest,
					),
				)
			case errors.As(verificationError, &manifest.NameInvalidError{}):
				errList = append(
					errList, errcode.ErrCodeNameInvalid.WithDetail(err),
				)
			case errors.As(verificationError, &manifest.UnverifiedError{}):
				errList = append(errList, errcode.ErrCodeManifestUnverified)
			case errors.As(verificationError, &manifest.ReferencesExceedLimitError{}):
				errList = append(
					errList, errcode.ErrCodeManifestReferenceLimit.WithDetail(err),
				)
			case errors.As(verificationError, &manifest.PayloadSizeExceedsLimitError{}):
				errList = append(
					errList, errcode.ErrCodeManifestPayloadSizeLimit.WithDetail(err.Error()),
				)
			default:
				if errors.Is(verificationError, digest.ErrDigestInvalidFormat) {
					errList = append(errList, errcode.ErrCodeDigestInvalid)
				} else {
					errList = append(errList, errcode.FromUnknownError(verificationError))
				}
			}
		}
	case errors.As(err, &errcode.Error{}):
		errList = append(errList, err)
	default:
		errList = append(errList, errcode.FromUnknownError(err))
	}
	return errList
}

func (r *LocalRegistry) PutManifest(
	ctx context.Context,
	artInfo pkg.RegistryInfo,
	mediaType string,
	body io.ReadCloser,
	length int64,
) (responseHeaders *commons.ResponseHeaders, errs []error) {
	var jsonBuf bytes.Buffer
	d, _ := digest.Parse(artInfo.Digest)
	tag := artInfo.Tag
	log.Ctx(ctx).Info().Msgf("Pushing manifest %s, digest: %q, tag: %s", artInfo.RegIdentifier, d, tag)

	responseHeaders = &commons.ResponseHeaders{
		Headers: map[string]string{},
		Code:    http.StatusCreated,
	}

	if err := copyFullPayload(ctx, length, body, &jsonBuf, "image manifest PUT"); err != nil {
		// copyFullPayload reports the error if necessary
		errs = append(errs, errcode.ErrCodeManifestInvalid.WithDetail(err.Error()))
		return responseHeaders, errs
	}

	unmarshalManifest, desc, err := manifest.UnmarshalManifest(mediaType, jsonBuf.Bytes())
	if err != nil {
		errs = append(errs, errcode.ErrCodeManifestInvalid.WithDetail(err))
		return responseHeaders, errs
	}

	if d != "" {
		if desc.Digest != d {
			log.Ctx(ctx).Error().Stack().Err(err).Msgf("payload digest does not match: %q != %q", desc.Digest, d)
			errs = append(errs, errcode.ErrCodeDigestInvalid)
			return responseHeaders, errs
		}
	} else {
		if tag != "" {
			d = desc.Digest
			log.Ctx(ctx).Debug().Msgf("payload digest: %q", d)
		} else {
			errs = append(errs, errcode.ErrCodeTagInvalid.WithDetail("no tag or digest specified"))
			return responseHeaders, errs
		}
	}

	isAnOCIManifest := mediaType == v1.MediaTypeImageManifest ||
		mediaType == v1.MediaTypeImageIndex

	if isAnOCIManifest {
		log.Ctx(ctx).Debug().Msg("Putting an OCI Manifest!")
	} else {
		log.Ctx(ctx).Debug().Msg("Putting a Docker Manifest!")
	}

	// We don't need to store manifest file in S3 storage
	// manifestServicePut(ctx, _manifest, options...)

	if err = r.ms.DBPut(
		ctx, unmarshalManifest, d, artInfo.RegIdentifier,
		responseHeaders, artInfo,
	); err != nil {
		errs = r.appendPutError(err, errs)
		return responseHeaders, errs
	}

	// Tag this manifest
	if tag != "" {
		if err = r.ms.DBTag(
			ctx, unmarshalManifest, d, tag, artInfo.RegIdentifier,
			responseHeaders, artInfo,
		); err != nil {
			errs = r.appendPutError(err, errs)
			return responseHeaders, errs
		}
	}

	if err != nil {
		return r.handlePutManifestErrors(err, errs, responseHeaders)
	}

	// Tag this manifest
	if tag != "" {
		err = tagserviceTag()
		// err = tags.Tag(imh, Tag, desc)
		if err != nil {
			errs = append(errs, errcode.ErrCodeUnknown.WithDetail(err))
			return responseHeaders, errs
		}
	}

	// Construct a canonical url for the uploaded manifest.
	name, _ := reference.WithName(fmt.Sprintf("%s/%s/%s", artInfo.PathRoot, artInfo.RegIdentifier, artInfo.Image))
	canonicalRef, err := reference.WithDigest(name, d)
	if err != nil {
		errs = append(errs, errcode.ErrCodeUnknown.WithDetail(err))
		return responseHeaders, errs
	}

	builder := artInfo.URLBuilder
	location, err := builder.BuildManifestURL(canonicalRef)
	if err != nil {
		log.Ctx(ctx).Error().Stack().Err(
			err,
		).Msgf("error building manifest url from digest: %v", err)
	}

	responseHeaders.Headers["Location"] = location
	responseHeaders.Headers["Docker-Content-Digest"] = d.String()
	responseHeaders.Code = http.StatusCreated

	log.Ctx(ctx).Debug().Msgf("Succeeded in putting manifest: %s", d.String())
	return responseHeaders, errs
}

func (r *LocalRegistry) handlePutManifestErrors(
	err error, errs []error, responseHeaders *commons.ResponseHeaders,
) (*commons.ResponseHeaders, []error) {
	if errors.Is(err, manifest.ErrUnsupported) {
		errs = append(errs, errcode.ErrCodeUnsupported)
		return responseHeaders, errs
	}
	if errors.Is(err, manifest.ErrAccessDenied) {
		errs = append(errs, errcode.ErrCodeDenied)
		return responseHeaders, errs
	}
	switch {
	case errors.As(err, &manifest.VerificationErrors{}):
		var verificationError manifest.VerificationErrors
		errors.As(err, &verificationError)
		for _, verificationError := range verificationError {
			switch {
			case errors.As(verificationError, &manifest.BlobUnknownError{}):
				var manifestBlobUnknownError manifest.BlobUnknownError
				errors.As(verificationError, &manifestBlobUnknownError)
				dgst := manifestBlobUnknownError.Digest
				errs = append(
					errs,
					errcode.ErrCodeManifestBlobUnknown.WithDetail(
						dgst,
					),
				)
			case errors.As(verificationError, &manifest.NameInvalidError{}):
				errs = append(
					errs,
					errcode.ErrCodeNameInvalid.WithDetail(err),
				)
			case errors.As(verificationError, &manifest.UnverifiedError{}):
				errs = append(errs, errcode.ErrCodeManifestUnverified)
			default:
				if errors.Is(verificationError, digest.ErrDigestInvalidFormat) {
					errs = append(errs, errcode.ErrCodeDigestInvalid)
				} else {
					errs = append(errs, errcode.ErrCodeUnknown, verificationError)
				}
			}
		}
	case errors.As(err, &errcode.Error{}):
		errs = append(errs, err)
	default:
		errs = append(errs, errcode.ErrCodeUnknown.WithDetail(err))
	}
	return responseHeaders, errs
}

func tagserviceTag() error {
	// TODO: implement this
	return nil
}

func (r *LocalRegistry) PushBlobMonolith(
	_ context.Context,
	_ pkg.RegistryInfo,
	_ int64,
	_ io.Reader,
) error {
	return nil
}

func (r *LocalRegistry) InitBlobUpload(
	ctx2 context.Context,
	artInfo pkg.RegistryInfo,
	fromRepo, mountDigest string,
) (*commons.ResponseHeaders, []error) {
	blobCtx := r.App.GetBlobsContext(ctx2, artInfo)
	var errList []error
	responseHeaders := &commons.ResponseHeaders{
		Headers: make(map[string]string),
		Code:    0,
	}
	digest := digest.Digest(mountDigest)
	if mountDigest != "" && fromRepo != "" {
		err := r.dbMountBlob(blobCtx, fromRepo, artInfo.RegIdentifier, digest, artInfo) //nolint:contextcheck
		if err != nil {
			e := fmt.Errorf("failed to mount blob in database: %w", err)
			errList = append(errList, errcode.FromUnknownError(e))
		}
		if err = writeBlobCreatedHeaders(
			blobCtx, digest,
			responseHeaders, artInfo,
		); err != nil {
			errList = append(errList, errcode.ErrCodeUnknown.WithDetail(err))
		}
		return responseHeaders, errList
	}

	blobs := blobCtx.OciBlobStore
	upload, err := blobs.Create(blobCtx.Context) //nolint:contextcheck
	if err != nil {
		if errors.Is(err, storage.ErrUnsupported) {
			errList = append(errList, errcode.ErrCodeUnsupported)
		} else {
			errList = append(errList, errcode.ErrCodeUnknown.WithDetail(err))
		}
		return responseHeaders, errList
	}

	blobCtx.Upload = upload

	if err = blobUploadResponse(
		blobCtx, responseHeaders,
		artInfo.RegIdentifier, artInfo,
	); err != nil {
		errList = append(errList, errcode.ErrCodeUnknown.WithDetail(err))
		return responseHeaders, errList
	}
	responseHeaders.Headers[storage.HeaderDockerUploadUUID] = blobCtx.Upload.ID()
	responseHeaders.Code = http.StatusAccepted
	return responseHeaders, nil
}

func (r *LocalRegistry) PushBlobMonolithWithDigest(
	_ context.Context,
	_ pkg.RegistryInfo,
	_ int64,
	_ io.Reader,
) error {
	return nil
}

func (r *LocalRegistry) PushBlobChunk(
	ctx *Context,
	artInfo pkg.RegistryInfo,
	contentType string,
	contentRange string,
	contentLength string,
	body io.ReadCloser,
	contentLengthFromRequest int64,
) (responseHeaders *commons.ResponseHeaders, errs []error) {
	responseHeaders = &commons.ResponseHeaders{
		Code:    0,
		Headers: make(map[string]string),
	}

	errs = make([]error, 0)

	if ctx.Upload == nil {
		e := errcode.ErrCodeBlobUploadUnknown
		errs = append(errs, e)
		return responseHeaders, errs
	}

	if contentType != "" && contentType != "application/octet-stream" {
		e := errcode.ErrCodeUnknown.WithDetail(fmt.Errorf("bad Content-Type"))
		errs = append(errs, e)
		return responseHeaders, errs
	}

	if contentRange != "" && contentLength != "" {
		start, end, err := parseContentRange(contentRange)
		if err != nil {
			errs = append(errs, errcode.ErrCodeUnknown.WithDetail(err.Error()))
			return responseHeaders, errs
		}
		if start > end || start != ctx.Upload.Size() {
			errs = append(errs, errcode.ErrCodeRangeInvalid)
			return responseHeaders, errs
		}

		clInt, err := strconv.ParseInt(contentLength, 10, 64)
		if err != nil {
			errs = append(errs, errcode.ErrCodeUnknown.WithDetail(err.Error()))
			return responseHeaders, errs
		}
		if clInt != (end-start)+1 {
			errs = append(errs, errcode.ErrCodeSizeInvalid)
			return responseHeaders, errs
		}
	}

	if err := copyFullPayload(
		ctx, contentLengthFromRequest, body, ctx.Upload,
		"blob PATCH",
	); err != nil {
		errs = append(
			errs,
			errcode.ErrCodeUnknown.WithDetail(err.Error()),
		)
		return responseHeaders, errs
	}

	if err := blobUploadResponse(
		ctx, responseHeaders, artInfo.RegIdentifier,
		artInfo,
	); err != nil {
		errs = append(errs, errcode.ErrCodeUnknown.WithDetail(err))
		return responseHeaders, errs
	}

	responseHeaders.Code = http.StatusAccepted
	return responseHeaders, errs
}

func (r *LocalRegistry) PushBlob(
	ctx2 context.Context,
	artInfo pkg.RegistryInfo,
	body io.ReadCloser,
	contentLength int64,
	stateToken string,
) (responseHeaders *commons.ResponseHeaders, errs []error) {
	errs = make([]error, 0)
	responseHeaders = &commons.ResponseHeaders{
		Code:    0,
		Headers: make(map[string]string),
	}
	ctx := r.App.GetBlobsContext(ctx2, artInfo)
	if ctx.UUID != "" {
		resumeErrs := ResumeBlobUpload(ctx, stateToken) //nolint:contextcheck
		errs = append(errs, resumeErrs...)
	}

	if ctx.Upload == nil {
		err := errcode.ErrCodeBlobUploadUnknown
		errs = append(errs, err)
		return responseHeaders, errs
	}

	defer ctx.Upload.Close()

	if artInfo.Digest == "" {
		// no digest? return error, but allow retry.
		err := errcode.ErrCodeDigestInvalid.WithDetail("digest missing")
		errs = append(errs, err)
		return responseHeaders, errs
	}

	dgst, err := digest.Parse(artInfo.Digest)
	if err != nil {
		// no digest? return error, but allow retry.
		errs = append(
			errs,
			errcode.ErrCodeDigestInvalid.WithDetail(
				"digest parsing failed",
			),
		)
		return responseHeaders, errs
	}

	//nolint:contextcheck
	if err := copyFullPayload(
		ctx, contentLength, body, ctx.Upload,
		"blob PUT",
	); err != nil {
		errs = append(
			errs,
			errcode.ErrCodeUnknown.WithDetail(err.Error()),
		)
		return responseHeaders, errs
	}

	//nolint:contextcheck
	desc, err := ctx.Upload.Commit(
		ctx, artInfo.RootIdentifier, manifest.Descriptor{
			Digest: dgst,
		},
	)

	if err != nil {
		switch {
		case errors.As(err, &storage.BlobInvalidDigestError{}):
			errs = append(
				errs,
				errcode.ErrCodeDigestInvalid.WithDetail(err),
			)
		case errors.As(err, &errcode.Error{}):
			errs = append(errs, err)
		default:
			switch {
			case errors.Is(err, storage.ErrAccessDenied):
				errs = append(errs, errcode.ErrCodeDenied)
			case errors.Is(err, storage.ErrUnsupported):
				errs = append(
					errs,
					errcode.ErrCodeUnsupported,
				)
			case errors.Is(err, storage.ErrBlobInvalidLength), errors.Is(err, storage.ErrBlobDigestUnsupported):
				errs = append(
					errs,
					errcode.ErrCodeBlobUploadInvalid.WithDetail(err),
				)
			default:
				//nolint:contextcheck
				log.Ctx(ctx).Error().Msgf("unknown error completing upload: %v", err)
				errs = append(errs, errcode.ErrCodeUnknown.WithDetail(err))
			}
		}
		//nolint:contextcheck
		// Clean up the backend blob data if there was an error.
		if err := ctx.Upload.Cancel(ctx); err != nil {
			// If the cleanup fails, all we can do is observe and report.
			log.Error().Stack().Err(
				err,
			).Msgf("error canceling upload after error: %v", err)
		}
		return responseHeaders, errs
	}

	//nolint:contextcheck
	err = r.dbPutBlobUploadComplete(
		ctx,
		artInfo.RegIdentifier,
		"application/octet-stream",
		artInfo.Digest,
		int(desc.Size),
		artInfo,
	)
	if err != nil {
		errs = append(errs, err)
		log.Error().Stack().Err(err).Msgf(
			"ensure blob %s failed, error: %v",
			artInfo.Digest, err,
		)
		return responseHeaders, errs
	}

	if err := writeBlobCreatedHeaders(
		ctx, desc.Digest,
		responseHeaders, artInfo,
	); err != nil {
		errs = append(errs, errcode.ErrCodeUnknown.WithDetail(err))
		return responseHeaders, errs
	}

	return responseHeaders, errs
}

func (r *LocalRegistry) ListTags(
	c context.Context,
	lastEntry string,
	maxEntries int,
	origURL string,
	artInfo pkg.RegistryInfo,
) (*commons.ResponseHeaders, []string, error) {
	filters := types.FilterParams{
		LastEntry:  lastEntry,
		MaxEntries: maxEntries,
	}

	tags, moreEntries, err := r.dbGetTags(c, filters, artInfo)
	if err != nil {
		return nil, nil, err
	}
	if len(tags) == 0 {
		// If no tags are found, the current implementation (`else`)
		// returns a nil slice instead of an empty one,
		// so we have to enforce the same behavior here, for consistency.
		tags = nil
	}

	responseHeaders := &commons.ResponseHeaders{
		Headers: map[string]string{"Content-Type": "application/json"},
		Code:    0,
	}

	// Add a link header if there are more entries to retrieve
	// (only supported by the metadata database backend)
	if moreEntries {
		filters.LastEntry = tags[len(tags)-1]
		urlStr, err := CreateLinkEntry(origURL, filters, "", "")
		if err != nil {
			return responseHeaders, nil, errcode.ErrCodeUnknown.WithDetail(err)
		}
		if urlStr != "" {
			responseHeaders.Headers["Link"] = urlStr
		}
	}

	return responseHeaders, tags, nil
}

func (r *LocalRegistry) ListFilteredTags(
	_ context.Context,
	_ int,
	_, _ string,
	_ pkg.RegistryInfo,
) (tags []string, err error) {
	return nil, nil
}

func (r *LocalRegistry) DeleteManifest(
	ctx context.Context,
	artInfo pkg.RegistryInfo,
) (errs []error, responseHeaders *commons.ResponseHeaders) {
	log.Debug().Msg("DeleteImageManifest")
	var tag = artInfo.Tag
	var d = artInfo.Digest

	responseHeaders = &commons.ResponseHeaders{}

	// TODO: If Tag is not empty, we just untag the tag, nothing more!
	if tag != "" {
		log.Debug().Msg("DeleteImageTag")
		_, err := r.ms.DeleteTag(ctx, artInfo.RegIdentifier, tag, artInfo)
		if err != nil {
			errs = append(errs, err)
			return errs, responseHeaders
		}
		responseHeaders.Code = http.StatusAccepted
		return errs, responseHeaders
	}

	err := r.ms.DeleteManifest(
		ctx, artInfo.RegIdentifier,
		digest.Digest(d), artInfo,
	)
	if err != nil {
		switch {
		case errors.Is(err, digest.ErrDigestUnsupported):
		case errors.Is(err, digest.ErrDigestInvalidFormat):
			errs = append(errs, errcode.ErrCodeDigestInvalid)
		case errors.Is(err, storage.ErrBlobUnknown):
			errs = append(errs, errcode.ErrCodeManifestUnknown)
		case errors.Is(err, manifest.ErrUnsupported):
			errs = append(errs, errcode.ErrCodeUnsupported)
		case errors.Is(err, util.ErrManifestNotFound):
			errs = append(errs, errcode.ErrCodeManifestUnknown)
		case errors.Is(err, util.ErrManifestReferencedInList):
			errs = append(errs, errcode.ErrCodeManifestReferencedInList)
		default:
			errs = append(errs, errcode.ErrCodeUnknown)
		}
		return errs, responseHeaders
	}
	responseHeaders.Code = http.StatusAccepted
	return errs, responseHeaders
}

func (r *LocalRegistry) DeleteBlob(
	ctx *Context,
	artInfo pkg.RegistryInfo,
) (responseHeaders *commons.ResponseHeaders, errs []error) {
	responseHeaders = &commons.ResponseHeaders{
		Code:    0,
		Headers: make(map[string]string),
	}

	errs = make([]error, 0)

	err := r.dbDeleteBlob(ctx, r.App.Config, artInfo.RegIdentifier, digest.Digest(artInfo.Digest), artInfo)
	if err != nil {
		switch {
		case errors.Is(err, storage.ErrUnsupported):
			errs = append(errs, errcode.ErrCodeUnsupported)
		case errors.Is(err, storage.ErrBlobUnknown):
			errs = append(errs, errcode.ErrCodeBlobUnknown)
		case errors.Is(err, storage.RegistryUnknownError{Name: artInfo.RegIdentifier}):
			errs = append(errs, errcode.ErrCodeNameUnknown)
		default:
			errs = append(errs, errcode.FromUnknownError(err))
			log.Error().Stack().Msg("failed to delete blob")
		}
		return
	}

	responseHeaders.Headers["Content-Length"] = "0"
	responseHeaders.Code = http.StatusAccepted
	return
}

func (r *LocalRegistry) MountBlob(
	_ context.Context,
	_ pkg.RegistryInfo,
	_, _ string,
) (err error) {
	return nil
}

func (r *LocalRegistry) ListReferrers(
	ctx context.Context,
	artInfo pkg.RegistryInfo,
	artifactType string,
) (index *v1.Index, responseHeaders *commons.ResponseHeaders, err error) {
	mfs := make([]v1.Descriptor, 0)
	rsHeaders := &commons.ResponseHeaders{
		Headers: map[string]string{"Content-Type": ReferrersMediaType},
		Code:    0,
	}
	if artifactType != "" {
		rsHeaders.Headers["OCI-Filters-Applied"] = "artifactType"
	}
	registry, err := r.registryDao.GetByParentIDAndName(ctx, artInfo.ParentID, artInfo.RegIdentifier)
	if err != nil {
		return nil, rsHeaders, err
	}
	if registry == nil {
		err := errcode.ErrCodeNameUnknown.WithDetail(artInfo.RegIdentifier)
		return nil, rsHeaders, err
	}
	subjectDigest, err := types.NewDigest(digest.Digest(artInfo.Digest))
	if err != nil {
		return nil, rsHeaders, err
	}
	manifests, err := r.manifestDao.ListManifestsBySubjectDigest(
		ctx, registry.ID, subjectDigest,
	)
	if err != nil && !errors.Is(err, store2.ErrResourceNotFound) {
		return nil, rsHeaders, err
	}

	for _, m := range manifests {
		mf := v1.Descriptor{
			MediaType:   m.MediaType,
			Size:        m.TotalSize,
			Digest:      m.Digest,
			Annotations: m.Annotations,
		}

		if m.ArtifactType.Valid {
			mf.ArtifactType = m.ArtifactType.String
		} else {
			mf.ArtifactType = m.Configuration.MediaType
		}

		// filter by the artifactType since the artifactType is
		// actually the config media type of the artifact.
		if artifactType != "" {
			if mf.ArtifactType == artifactType {
				mfs = append(mfs, mf)
			}
		} else {
			mfs = append(mfs, mf)
		}
	}

	// Populate index manifest
	result := &v1.Index{}
	result.SchemaVersion = ReferrersSchemaVersion
	result.MediaType = ReferrersMediaType
	result.Manifests = mfs

	return result, rsHeaders, nil
}

func (r *LocalRegistry) GetBlobUploadStatus(
	ctx *Context,
	artInfo pkg.RegistryInfo,
	_ string,
) (*commons.ResponseHeaders, []error) {
	responseHeaders := &commons.ResponseHeaders{
		Code:    0,
		Headers: make(map[string]string),
	}

	errList := make([]error, 0)
	log.Debug().Msgf("GetBlobUploadStatus")

	if ctx.Upload == nil {
		blobs := ctx.OciBlobStore
		upload, err := blobs.Resume(ctx, ctx.UUID)
		if err != nil {
			if errors.Is(err, distribution.ErrBlobUploadUnknown) {
				errList = append(
					errList,
					errcode.ErrCodeBlobUploadUnknown.WithDetail(err),
				)
			} else {
				errList = append(
					errList,
					errcode.ErrCodeUnknown.WithDetail(err),
				)
			}
			return responseHeaders, errList
		}

		ctx.Upload = upload
	}

	if err := blobUploadResponse(
		ctx, responseHeaders,
		artInfo.RegIdentifier, artInfo,
	); err != nil {
		errList = append(
			errList,
			errcode.ErrCodeUnknown.WithDetail(err),
		)
		return responseHeaders, errList
	}

	responseHeaders.Code = http.StatusNoContent
	return responseHeaders, nil
}
func (r *LocalRegistry) GetCatalog() (repositories []string, err error) {
	return nil, nil
}
func (r *LocalRegistry) DeleteTag(
	_, _ string,
	_ pkg.RegistryInfo,
) error {
	return nil
}

func (r *LocalRegistry) PullBlobChunk(
	_, _ string,
	_, _, _ int64,
	_ pkg.RegistryInfo,
) (size int64, blob io.ReadCloser, err error) {
	return 0, nil, nil
}

// WriteBlobCreatedHeaders writes the standard Headers
//
//	describing a newly
//
// created blob. A 201 Created is written as well as the
//
//	canonical URL and
//
// blob digest.
func writeBlobCreatedHeaders(
	context *Context,
	digest digest.Digest,
	headers *commons.ResponseHeaders,
	info pkg.RegistryInfo,
) error {
	path, err := reference.WithName(fmt.Sprintf("%s/%s/%s", info.PathRoot, info.RegIdentifier, info.Image))
	if err != nil {
		return err
	}
	ref, err := reference.WithDigest(path, digest)
	if err != nil {
		return err
	}
	blobURL, err := context.URLBuilder.BuildBlobURL(ref)
	if err != nil {
		return err
	}

	headers.Headers = map[string]string{
		"Location":              blobURL,
		"Docker-Content-Digest": digest.String(),
		"Content-Length":        "0",
	}
	headers.Code = http.StatusCreated

	return nil
}

func blobUploadResponse(
	context *Context,
	headers *commons.ResponseHeaders,
	repoKey string,
	info pkg.RegistryInfo,
) error {
	context.State.Path = context.OciBlobStore.Path()
	context.State.UUID = context.Upload.ID()
	context.Upload.Close()
	context.State.Offset = context.Upload.Size()

	token, err := hmacKey(
		context.Config.Registry.HTTP.Secret,
	).packUploadState(
		context.State,
	)
	if err != nil {
		log.Info().Msgf("error building upload state token: %s", err)
		return err
	}
	image := info.Image
	path, err := reference.WithName(fmt.Sprintf("%s/%s/%s", info.PathRoot, repoKey, image))
	if err != nil {
		return err
	}
	uploadURL, err := context.URLBuilder.BuildBlobUploadChunkURL(
		path, context.Upload.ID(),
		url.Values{
			"_state": []string{token},
		},
	)
	if err != nil {
		log.Info().Msgf("error building upload url: %s", err)
		return err
	}

	endRange := context.Upload.Size()
	if endRange > 0 {
		endRange--
	}
	headers.Headers["Docker-Upload-UUID"] = context.UUID
	headers.Headers["Location"] = uploadURL

	headers.Headers["Content-Length"] = "0"
	headers.Headers["Range"] = fmt.Sprintf("0-%d", endRange)

	return nil
}

// packUploadState packs the upload state signed with and hmac digest using
// the hmacKey secret, encoding to url safe base64. The resulting token can be
// used to share data with minimized risk of external tampering.
func (secret hmacKey) packUploadState(lus BlobUploadState) (string, error) {
	mac := hmac.New(sha256.New, []byte(secret))
	p, err := json.Marshal(lus)
	if err != nil {
		return "", err
	}

	mac.Write(p)

	return base64.URLEncoding.EncodeToString(append(mac.Sum(nil), p...)), nil
}

func parseContentRange(cr string) (start int64, end int64, err error) {
	rStart, rEnd, ok := strings.Cut(cr, "-")
	if !ok {
		return -1, -1, fmt.Errorf("invalid content range format, %s", cr)
	}
	start, err = strconv.ParseInt(rStart, 10, 64)
	if err != nil {
		return -1, -1, err
	}
	end, err = strconv.ParseInt(rEnd, 10, 64)
	if err != nil {
		return -1, -1, err
	}
	return start, end, nil
}

func (r *LocalRegistry) dbBlobLinkExists(
	ctx context.Context, dgst digest.Digest, repoKey string,
	info pkg.RegistryInfo,
) error {
	reg, err := r.registryDao.GetByParentIDAndName(ctx, info.ParentID, repoKey)
	if err != nil {
		return err
	}
	if r == nil {
		err := errcode.ErrCodeNameUnknown.WithDetail(repoKey)
		return err
	}
	blob, err := r.blobRepo.FindByDigestAndRootParentID(ctx, dgst, info.RootParentID)
	if err != nil {
		if errors.Is(err, store2.ErrResourceNotFound) {
			err = errcode.ErrCodeBlobUnknown.WithDetail(dgst)
		}
		return err
	}

	err = r.tx.WithTx(
		ctx, func(ctx context.Context) error {
			// Prevent long running transactions by setting an upper limit of blobExistsGCLockTimeout. If the GC is holding
			// the lock of a related review record, the processing there should be fast enough to avoid this. Regardless, we
			// should not let transactions open (and clients waiting) for too long. If this sensible timeout is exceeded, abort
			// the operation and let the client retry. This will bubble up and lead to a 503 Service Unavailable response.
			ctx, cancel := context.WithTimeout(ctx, blobExistsGCLockTimeout)
			defer cancel()

			bt, err := r.gcService.BlobFindAndLockBefore(ctx, blob.ID, time.Now().Add(blobExistsGCReviewWindow))
			if err != nil && !errors.Is(err, sql.ErrNoRows) {
				return err
			}

			if bt != nil {
				err = r.gcService.BlobReschedule(ctx, bt, 24*time.Hour)
				if err != nil {
					return err
				}
			}

			found, err := r.blobRepo.ExistsBlob(ctx, reg.ID, dgst, info.Image)
			if err != nil {
				return err
			}

			if !found {
				err := errcode.ErrCodeBlobUnknown.WithDetail(dgst)
				return err
			}
			return nil
		},
	)

	if err != nil {
		return fmt.Errorf("committing database transaction: %w", err)
	}

	return nil
}

func (r *LocalRegistry) dbPutBlobUploadComplete(
	ctx *Context,
	repoName string,
	mediaType string,
	digestVal string,
	size int,
	info pkg.RegistryInfo,
) error {
	blob := &types.Blob{
		RootParentID: info.RootParentID,
		Digest:       digest.Digest(digestVal),
		MediaType:    mediaType,
		Size:         int64(size),
	}

	var storedBlob *types.Blob
	created := false
	err := r.tx.WithTx(
		ctx.Context, func(ctx context.Context) error {
			registry, err := r.registryDao.GetByParentIDAndName(ctx, info.ParentID, repoName)
			if err != nil {
				return err
			}

			storedBlob, created, err = r.blobRepo.CreateOrFind(ctx, blob)
			if err != nil && !errors.Is(err, store2.ErrResourceNotFound) {
				return err
			}

			// link blob to repository
			if err := r.registryBlobDao.LinkBlob(ctx, info.Image, registry, storedBlob.ID); err != nil {
				return err
			}

			err = r.ms.UpsertImage(ctx, repoName, info)
			if err != nil {
				return err
			}

			return nil
		}, dbtx.TxDefault,
	)
	if err != nil {
		log.Error().Msgf("failed to put blob in database: %v", err)
		return fmt.Errorf("committing database transaction: %w", err)
	}

	// Emit blob create event
	if created {
		event.ReportEventAsync(ctx.Context, ctx.OciBlobStore.Path(),
			r.reporter, event.BlobCreate, storedBlob.ID,
			"", digestVal, r.App.Config)
	}
	return nil
}

// dbDeleteBlob does not actually delete a blob from the database
// (that's GC's responsibility), it only unlinks it from
// a repository.
func (r *LocalRegistry) dbDeleteBlob(
	ctx *Context, config *gitnesstypes.Config,
	repoName string, d digest.Digest, info pkg.RegistryInfo,
) error {
	log.Debug().Msgf("deleting blob from repository in database")

	if !config.Registry.Storage.S3Storage.Delete {
		return storage.ErrUnsupported
	}

	reg, err := r.registryDao.GetByParentIDAndName(ctx, info.ParentID, repoName)

	if err != nil {
		return err
	}
	if r == nil {
		return storage.RegistryUnknownError{Name: repoName}
	}

	blob, err := r.blobRepo.FindByDigestAndRepoID(ctx, d, reg.ID, info.Image)
	if err != nil {
		if errors.Is(err, store2.ErrResourceNotFound) {
			return storage.ErrBlobUnknown
		}
		return err
	}
	found, err := r.registryBlobDao.UnlinkBlob(ctx, info.Image, reg, blob.ID)
	if err != nil {
		return err
	}
	if !found {
		return storage.ErrBlobUnknown
	}

	// No need to emit blob delete event here. The GC will take care of it even in the replicated regions.
	return nil
}

func (r *LocalRegistry) dbGetTags(
	ctx context.Context, filters types.FilterParams,
	info pkg.RegistryInfo,
) ([]string, bool, error) {
	log.Debug().Msgf("finding tags in database")

	reg, err := r.registryDao.GetByParentIDAndName(ctx, info.ParentID, info.RegIdentifier)
	if err != nil {
		return nil, false, err
	}
	if r == nil {
		return nil, false,
			errcode.ErrCodeNameUnknown.WithDetail(map[string]string{"name": info.RegIdentifier})
	}

	tt, err := r.tagDao.TagsPaginated(ctx, reg.ID, info.Image, filters)
	if err != nil {
		return nil, false, err
	}

	tags := make([]string, 0, len(tt))
	for _, t := range tt {
		tags = append(tags, t.Name)
	}

	var moreEntries bool
	if len(tt) > 0 {
		filters.LastEntry = tt[len(tt)-1].Name
		moreEntries, err = r.tagDao.HasTagsAfterName(ctx, reg.ID, filters)
		if err != nil {
			return nil, false, err
		}
	}

	return tags, moreEntries, nil
}

func (r *LocalRegistry) dbMountBlob(
	ctx context.Context, fromImageRef, toRepo string,
	d digest.Digest, info pkg.RegistryInfo,
) error {
	log.Debug().Msgf("cross repository blob mounting")

	destRepo, err := r.registryDao.GetByParentIDAndName(ctx, info.ParentID, toRepo)
	if err != nil {
		return err
	}
	if destRepo == nil {
		return fmt.Errorf(
			"destination repository: [%s] not found in database",
			toRepo,
		)
	}

	_, imageRef, err := paths.DisectRoot(fromImageRef)
	if err != nil {
		return fmt.Errorf(
			"failed to parse image reference from: [%s]",
			fromImageRef,
		)
	}
	fromRepo, fromImageName, err := paths.DisectRoot(imageRef)
	if err != nil {
		return fmt.Errorf(
			"failed to parse image name from: [%s]",
			imageRef,
		)
	}

	sourceRepo, err := r.registryDao.GetByParentIDAndName(ctx, info.ParentID, fromRepo)
	if err != nil {
		return err
	}
	if sourceRepo == nil {
		return fmt.Errorf(
			"source repository: [%s] not found in database",
			fromRepo,
		)
	}

	b, err := r.ms.DBFindRepositoryBlob(
		ctx, manifest.Descriptor{Digest: d},
		sourceRepo.ID, fromImageName,
	)
	if err != nil {
		return err
	}

	return r.registryBlobDao.LinkBlob(ctx, info.Image, destRepo, b.ID)
}
