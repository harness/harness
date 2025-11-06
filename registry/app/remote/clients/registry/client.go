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

package registry

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	commonhttp "github.com/harness/gitness/registry/app/common/http"
	"github.com/harness/gitness/registry/app/common/lib"
	"github.com/harness/gitness/registry/app/common/lib/errors"
	"github.com/harness/gitness/registry/app/manifest"
	"github.com/harness/gitness/registry/app/manifest/manifestlist"
	"github.com/harness/gitness/registry/app/manifest/schema2"
	"github.com/harness/gitness/registry/app/pkg/commons"
	"github.com/harness/gitness/registry/app/pkg/maven/utils"
	"github.com/harness/gitness/registry/app/remote/clients/registry/auth"
	"github.com/harness/gitness/registry/app/remote/clients/registry/interceptor"

	"github.com/opencontainers/go-digest"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/rs/zerolog/log"

	_ "github.com/harness/gitness/registry/app/manifest/ocischema" // register oci manifest unmarshal function
)

var (
	accepts = []string{
		v1.MediaTypeImageIndex,
		manifestlist.MediaTypeManifestList,
		v1.MediaTypeImageManifest,
		schema2.MediaTypeManifest,
		MediaTypeSignedManifest,
		MediaTypeManifest,
	}
)

// const definition.
const (
	UserAgent = "gitness-registry-client"
	// DefaultHTTPClientTimeout is the default timeout for registry http client.
	DefaultHTTPClientTimeout = 30 * time.Minute
	// MediaTypeManifest specifies the mediaType for the current version. Note
	// that for schema version 1, the media is optionally "application/json".
	MediaTypeManifest = "application/vnd.docker.distribution.manifest.v1+json"
	// MediaTypeSignedManifest specifies the mediatype for current SignedManifest version.
	MediaTypeSignedManifest = "application/vnd.docker.distribution.manifest.v1+prettyjws"
	// MediaTypeManifestLayer specifies the media type for manifest layers.
	MediaTypeManifestLayer = "application/vnd.docker.container.image.rootfs.diff+x-gtar"
)

var (
	// registryHTTPClientTimeout is the timeout for registry http client.
	registryHTTPClientTimeout time.Duration
)

func init() {
	registryHTTPClientTimeout = DefaultHTTPClientTimeout
	// override it if read from environment variable, in minutes
	if env := os.Getenv("GITNESS_REGISTRY_HTTP_CLIENT_TIMEOUT"); len(env) > 0 {
		timeout, err := strconv.ParseInt(env, 10, 64)
		if err != nil {
			log.Error().
				Stack().
				Err(err).
				Msg(
					fmt.Sprintf(
						"Failed to parse GITNESS_REGISTRY_HTTP_CLIENT_TIMEOUT: %v, use default value: %v",
						err, DefaultHTTPClientTimeout,
					),
				)
		} else if timeout > 0 {
			registryHTTPClientTimeout = time.Duration(timeout) * time.Minute
		}
	}
}

// Client defines the methods that a registry client should implements.
type Client interface {
	// Ping the base API endpoint "/v2/"
	Ping(ctx context.Context) (err error)
	// Catalog the repositories
	Catalog(ctx context.Context) (repositories []string, err error)
	// ListTags lists the tags under the specified repository
	ListTags(ctx context.Context, repository string) (tags []string, err error)
	// ManifestExist checks the existence of the manifest
	ManifestExist(ctx context.Context, repository, reference string) (exist bool, desc *manifest.Descriptor, err error)
	// PullManifest pulls the specified manifest
	PullManifest(
		ctx context.Context,
		repository, reference string,
		acceptedMediaTypes ...string,
	) (manifest manifest.Manifest, digest string, err error)
	// PushManifest pushes the specified manifest
	PushManifest(ctx context.Context, repository, reference, mediaType string, payload []byte) (
		digest string,
		err error,
	)
	// DeleteManifest deletes the specified manifest. The "reference" can be "tag" or "digest"
	DeleteManifest(ctx context.Context, repository, reference string) (err error)
	// BlobExist checks the existence of the specified blob
	BlobExist(ctx context.Context, repository, digest string) (exist bool, err error)
	// PullBlob pulls the specified blob. The caller must close the returned "blob"
	PullBlob(ctx context.Context, repository, digest string) (size int64, blob io.ReadCloser, err error)
	// PullBlobChunk pulls the specified blob, but by chunked
	PullBlobChunk(ctx context.Context, repository, digest string, blobSize, start, end int64) (
		size int64,
		blob io.ReadCloser,
		err error,
	)
	// PushBlob pushes the specified blob
	PushBlob(ctx context.Context, repository, digest string, size int64, blob io.Reader) error
	// PushBlobChunk pushes the specified blob, but by chunked
	PushBlobChunk(
		ctx context.Context,
		repository, digest string,
		blobSize int64,
		chunk io.Reader,
		start, end int64,
		location string,
	) (nextUploadLocation string, endRange int64, err error)
	// MountBlob mounts the blob from the source repository
	MountBlob(ctx context.Context, srcRepository, digest, dstRepository string) (err error)
	// DeleteBlob deletes the specified blob
	DeleteBlob(ctx context.Context, repository, digest string) (err error)
	// Copy the artifact from source repository to the destination. The "override"
	// is used to specify whether the destination artifact will be overridden if
	// its name is same with source but digest isn't
	Copy(
		ctx context.Context,
		srcRepository, srcReference, dstRepository, dstReference string,
		override bool,
	) (err error)
	// Do send generic HTTP requests to the target registry service
	Do(req *http.Request) (*http.Response, error)

	// GetFile Download the file
	GetFile(ctx context.Context, filePath string) (*commons.ResponseHeaders, io.ReadCloser, error)

	// HeadFile Check existence of file
	HeadFile(ctx context.Context, filePath string) (*commons.ResponseHeaders, error)

	// GetFileFromURL Download the file from URL instead of provided endpoint. Authorizer still remains the same.
	GetFileFromURL(ctx context.Context, url string) (*commons.ResponseHeaders, io.ReadCloser, error)
	GetURL(ctx context.Context, filePath string) string
}

// NewClient creates a registry client with the default authorizer which determines the auth scheme
// of the registry automatically and calls the corresponding underlying authorizers(basic/bearer) to
// do the auth work. If a customized authorizer is needed, use "NewClientWithAuthorizer" instead.
func NewClient(url, username, password string, insecure, isOCI bool, interceptors ...interceptor.Interceptor) Client {
	authorizer := auth.NewAuthorizer(username, password, insecure, isOCI)
	return NewClientWithAuthorizer(url, authorizer, insecure, interceptors...)
}

// NewClientWithAuthorizer creates a registry client with the provided authorizer.
func NewClientWithAuthorizer(
	url string,
	authorizer lib.Authorizer,
	insecure bool,
	interceptors ...interceptor.Interceptor,
) Client {
	return &client{
		url:          url,
		authorizer:   authorizer,
		interceptors: interceptors,
		client: &http.Client{
			Transport: commonhttp.GetHTTPTransport(commonhttp.WithInsecure(insecure)),
			Timeout:   registryHTTPClientTimeout,
		},
	}
}

type client struct {
	url          string
	authorizer   lib.Authorizer
	interceptors []interceptor.Interceptor
	client       *http.Client
}

func (c *client) Ping(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, buildPingURL(c.url), nil)
	if err != nil {
		return err
	}
	resp, err := c.do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

func (c *client) Catalog(ctx context.Context) ([]string, error) {
	var repositories []string
	url := buildCatalogURL(c.url)
	for {
		repos, next, err := c.catalog(ctx, url)
		if err != nil {
			return nil, err
		}
		repositories = append(repositories, repos...)

		url = next
		// no next page, end the loop
		if len(url) == 0 {
			break
		}
		// relative URL
		if !strings.Contains(url, "://") {
			url = c.url + url
		}
	}
	return repositories, nil
}

func (c *client) catalog(ctx context.Context, url string) ([]string, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, "", err
	}
	resp, err := c.do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", err
	}
	repositories := struct {
		Repositories []string `json:"repositories"`
	}{}
	if err := json.Unmarshal(body, &repositories); err != nil {
		return nil, "", err
	}
	return repositories.Repositories, next(resp.Header.Get("Link")), nil
}

func (c *client) ListTags(ctx context.Context, repository string) ([]string, error) {
	var tags []string
	url := buildTagListURL(c.url, repository)
	for {
		tgs, next, err := c.listTags(ctx, url)
		if err != nil {
			return nil, err
		}
		tags = append(tags, tgs...)

		url = next
		// no next page, end the loop
		if len(url) == 0 {
			break
		}
		// relative URL
		if !strings.Contains(url, "://") {
			url = c.url + url
		}
	}
	return tags, nil
}

func (c *client) listTags(ctx context.Context, url string) ([]string, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, "", err
	}
	resp, err := c.do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", err
	}
	tgs := struct {
		Tags []string `json:"tags"`
	}{}
	if err := json.Unmarshal(body, &tgs); err != nil {
		return nil, "", err
	}
	return tgs.Tags, next(resp.Header.Get("Link")), nil
}

func (c *client) ManifestExist(ctx context.Context, repository, reference string) (bool, *manifest.Descriptor, error) {
	req, err := http.NewRequestWithContext(
		ctx, http.MethodHead, buildManifestURL(c.url, repository, reference), nil,
	)
	if err != nil {
		return false, nil, err
	}
	for _, mediaType := range accepts {
		req.Header.Add("Accept", mediaType)
	}
	resp, err := c.do(req)
	if err != nil {
		if errors.IsErr(err, errors.NotFoundCode) {
			return false, nil, nil
		}
		return false, nil, err
	}
	defer resp.Body.Close()
	dig := resp.Header.Get("Docker-Content-Digest")
	contentType := resp.Header.Get("Content-Type")
	contentLen := resp.Header.Get("Content-Length")
	length, _ := strconv.Atoi(contentLen)
	return true, &manifest.Descriptor{Digest: digest.Digest(dig), MediaType: contentType, Size: int64(length)}, nil
}

func (c *client) PullManifest(
	ctx context.Context,
	repository, reference string,
	acceptedMediaTypes ...string,
) (manifest.Manifest, string, error) {
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet, buildManifestURL(
			c.url, repository,
			reference,
		), nil,
	)
	if err != nil {
		return nil, "", err
	}
	if len(acceptedMediaTypes) == 0 {
		acceptedMediaTypes = accepts
	}
	for _, mediaType := range acceptedMediaTypes {
		req.Header.Add("Accept", mediaType)
	}
	resp, err := c.do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	payload, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", err
	}
	mediaType := resp.Header.Get("Content-Type")
	manifest, _, err := manifest.UnmarshalManifest(mediaType, payload)
	if err != nil {
		return nil, "", err
	}
	digest := resp.Header.Get("Docker-Content-Digest")
	return manifest, digest, nil
}

func (c *client) PushManifest(ctx context.Context, repository, reference, mediaType string, payload []byte) (
	string,
	error,
) {
	req, err := http.NewRequestWithContext(
		ctx, http.MethodPut, buildManifestURL(c.url, repository, reference),
		bytes.NewReader(payload),
	)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", mediaType)
	resp, err := c.do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	return resp.Header.Get("Docker-Content-Digest"), nil
}

func (c *client) DeleteManifest(ctx context.Context, repository, reference string) error {
	_, err := digest.Parse(reference)
	if err != nil {
		// the reference is tag, get the digest first
		exist, desc, err := c.ManifestExist(ctx, repository, reference)
		if err != nil {
			return err
		}
		if !exist {
			return errors.New(nil).WithCode(errors.NotFoundCode).
				WithMessage("%s:%s not found", repository, reference)
		}
		reference = string(desc.Digest)
	}
	req, err := http.NewRequestWithContext(
		ctx, http.MethodDelete,
		buildManifestURL(c.url, repository, reference), nil,
	)
	if err != nil {
		return err
	}
	resp, err := c.do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

func (c *client) BlobExist(ctx context.Context, repository, digest string) (bool, error) {
	req, err := http.NewRequestWithContext(
		ctx, http.MethodHead, buildBlobURL(c.url, repository, digest), nil,
	)
	if err != nil {
		return false, err
	}
	resp, err := c.do(req)
	if err != nil {
		if errors.IsErr(err, errors.NotFoundCode) {
			return false, nil
		}
		return false, err
	}
	defer resp.Body.Close()
	return true, nil
}

func (c *client) PullBlob(ctx context.Context, repository, digest string) (int64, io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, buildBlobURL(c.url, repository, digest), nil)
	if err != nil {
		return 0, nil, err
	}

	req.Header.Add("Accept-Encoding", "identity")
	resp, err := c.do(req)
	if err != nil {
		return 0, nil, err
	}

	var size int64
	n := resp.Header.Get("Content-Length")
	// no content-length is acceptable, which can taken from manifests
	if len(n) > 0 {
		size, err = strconv.ParseInt(n, 10, 64)
		if err != nil {
			defer resp.Body.Close()
			return 0, nil, err
		}
	}

	return size, resp.Body, nil
}

// PullBlobChunk pulls the specified blob, but by chunked, refer to
// https://github.com/opencontainers/distribution-spec/blob/main/spec.md#pull
// for more details.
func (c *client) PullBlobChunk(ctx context.Context, repository, digest string, _ int64, start, end int64) (
	int64,
	io.ReadCloser,
	error,
) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, buildBlobURL(c.url, repository, digest), nil)
	if err != nil {
		return 0, nil, err
	}

	req.Header.Add("Accept-Encoding", "identity")
	req.Header.Add("Range", fmt.Sprintf("bytes=%d-%d", start, end))
	resp, err := c.do(req)
	if err != nil {
		return 0, nil, err
	}

	var size int64
	n := resp.Header.Get("Content-Length")
	// no content-length is acceptable, which can taken from manifests
	if len(n) > 0 {
		size, err = strconv.ParseInt(n, 10, 64)
		if err != nil {
			defer resp.Body.Close()
			return 0, nil, err
		}
	}

	return size, resp.Body, nil
}

func (c *client) PushBlob(ctx context.Context, repository, digest string, size int64, blob io.Reader) error {
	location, err := c.initiateBlobUpload(ctx, repository)
	if err != nil {
		return err
	}
	return c.monolithicBlobUpload(ctx, location, digest, size, blob)
}

// PushBlobChunk pushes the specified blob, but by chunked,
// refer to https://github.com/opencontainers/distribution-spec/blob/main/spec.md#push
// for more details.
func (c *client) PushBlobChunk(
	ctx context.Context,
	repository, digest string,
	blobSize int64,
	chunk io.Reader,
	start, end int64,
	location string,
) (string, int64, error) {
	var err error
	// first chunk need to initialize blob upload location
	if start == 0 {
		location, err = c.initiateBlobUpload(ctx, repository)
		if err != nil {
			return location, end, err
		}
	}

	// the range is from 0 to (blobSize-1), so (end == blobSize-1) means it is last chunk
	lastChunk := end == blobSize-1
	url, err := buildChunkBlobUploadURL(c.url, location, digest, lastChunk)
	if err != nil {
		return location, end, err
	}

	// use PUT instead of PATCH for last chunk which can reduce a final request
	method := http.MethodPatch
	if lastChunk {
		method = http.MethodPut
	}
	req, err := http.NewRequestWithContext(ctx, method, url, chunk)
	if err != nil {
		return location, end, err
	}

	req.Header.Set("Content-Length", fmt.Sprintf("%d", end-start+1))
	req.Header.Set("Content-Range", fmt.Sprintf("%d-%d", start, end))
	resp, err := c.do(req)
	if err != nil {
		// if push chunk error, we should query the upload progress for new location and end range.
		newLocation, newEnd, err1 := c.getUploadStatus(ctx, location)
		if err1 == nil {
			return newLocation, newEnd, err
		}
		// end should return start-1 to re-push this chunk
		return location, start - 1, fmt.Errorf("failed to get upload status: %w", err1)
	}

	defer resp.Body.Close()
	// return the location for next chunk upload
	return resp.Header.Get("Location"), end, nil
}

func (c *client) getUploadStatus(ctx context.Context, location string) (string, int64, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, location, nil)
	if err != nil {
		return location, -1, err
	}

	resp, err := c.do(req)
	if err != nil {
		return location, -1, err
	}

	defer resp.Body.Close()

	_, end, err := parseContentRange(resp.Header.Get("Range"))
	if err != nil {
		return location, -1, err
	}

	return resp.Header.Get("Location"), end, nil
}

func (c *client) initiateBlobUpload(ctx context.Context, repository string) (string, error) {
	req, err := http.NewRequestWithContext(
		ctx, http.MethodPost,
		buildInitiateBlobUploadURL(c.url, repository), nil,
	)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Length", "0")
	resp, err := c.do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	return resp.Header.Get("Location"), nil
}

func (c *client) monolithicBlobUpload(ctx context.Context, location, digest string, size int64, data io.Reader) error {
	url, err := buildMonolithicBlobUploadURL(c.url, location, digest)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, data)
	if err != nil {
		return err
	}
	req.ContentLength = size
	resp, err := c.do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

func (c *client) MountBlob(ctx context.Context, srcRepository, digest, dstRepository string) (err error) {
	req, err := http.NewRequestWithContext(
		ctx, http.MethodPost,
		buildMountBlobURL(c.url, dstRepository, digest, srcRepository), nil,
	)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Length", "0")
	resp, err := c.do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

func (c *client) DeleteBlob(ctx context.Context, repository, digest string) (err error) {
	req, err := http.NewRequestWithContext(
		ctx, http.MethodDelete, buildBlobURL(c.url, repository, digest), nil,
	)
	if err != nil {
		return err
	}
	resp, err := c.do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

func (c *client) Copy(
	ctx context.Context,
	srcRepository, srcReference, dstRepository, dstReference string,
	override bool,
) (err error) {
	// pull the manifest from the source repository
	manifest, srcDgt, err := c.PullManifest(ctx, srcRepository, srcReference)
	if err != nil {
		return err
	}

	// check the existence of the artifact on the destination repository
	blobExist, desc, err := c.ManifestExist(ctx, dstRepository, dstReference)
	if err != nil {
		return err
	}
	if blobExist {
		// the same artifact already exists
		if desc != nil && srcDgt == string(desc.Digest) {
			return nil
		}
		// the same name artifact exists, but not allowed to override
		if !override {
			return errors.New(nil).WithCode(errors.PreconditionCode).
				WithMessage("the same name but different digest artifact exists, but the override is set to false")
		}
	}

	for _, descriptor := range manifest.References() {
		digest := descriptor.Digest.String()
		switch descriptor.MediaType {
		// skip foreign layer
		case schema2.MediaTypeForeignLayer:
			continue
		// manifest or index
		case v1.MediaTypeImageIndex, manifestlist.MediaTypeManifestList,
			v1.MediaTypeImageManifest, schema2.MediaTypeManifest,
			MediaTypeSignedManifest, MediaTypeManifest:
			if err = c.Copy(ctx, srcRepository, digest, dstRepository, digest, false); err != nil {
				return err
			}
		// common layer
		default:
			blobExist, err = c.BlobExist(ctx, dstRepository, digest)
			if err != nil {
				return err
			}
			// the layer already exist, skip
			if blobExist {
				continue
			}
			// when the copy happens inside the same registry, use mount
			if err = c.MountBlob(ctx, srcRepository, digest, dstRepository); err != nil {
				return err
			}
		}
	}

	mediaType, payload, err := manifest.Payload()
	if err != nil {
		return err
	}
	// push manifest to the destination repository
	if _, err = c.PushManifest(ctx, dstRepository, dstReference, mediaType, payload); err != nil {
		return err
	}

	return nil
}

func (c *client) Do(req *http.Request) (*http.Response, error) {
	return c.do(req)
}

func (c *client) do(req *http.Request) (*http.Response, error) {
	for _, interceptor := range c.interceptors {
		if err := interceptor.Intercept(req); err != nil {
			return nil, err
		}
	}
	if c.authorizer != nil {
		if err := c.authorizer.Modify(req); err != nil {
			return nil, err
		}
	}
	req.Header.Set("User-Agent", UserAgent)
	log.Info().Msgf("[Remote Call]: Request: %s %s", req.Method, req.URL.String())
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		code := errors.GeneralCode
		switch resp.StatusCode {
		case http.StatusUnauthorized:
			code = errors.UnAuthorizedCode
		case http.StatusForbidden:
			code = errors.ForbiddenCode
		case http.StatusNotFound:
			code = errors.NotFoundCode
		case http.StatusTooManyRequests:
			code = errors.RateLimitCode
		}
		return nil, errors.New(nil).WithCode(code).
			WithMessage("http status code: %d, body: %s", resp.StatusCode, string(body))
	}
	return resp, nil
}

// parse the next page link from the link header.
func next(link string) string {
	links := lib.ParseLinks(link)
	for _, lk := range links {
		if lk.Rel == "next" {
			return lk.URL
		}
	}
	return ""
}

func buildPingURL(endpoint string) string {
	return fmt.Sprintf("%s/v2/", endpoint)
}

func buildCatalogURL(endpoint string) string {
	return fmt.Sprintf("%s/v2/_catalog?n=1000", endpoint)
}

func buildTagListURL(endpoint, repository string) string {
	return fmt.Sprintf("%s/v2/%s/tags/list", endpoint, repository)
}

func buildManifestURL(endpoint, repository, reference string) string {
	return fmt.Sprintf("%s/v2/%s/manifests/%s", endpoint, repository, reference)
}

func buildBlobURL(endpoint, repository, reference string) string {
	return fmt.Sprintf("%s/v2/%s/blobs/%s", endpoint, repository, reference)
}

func buildMountBlobURL(endpoint, repository, digest, from string) string {
	return fmt.Sprintf("%s/v2/%s/blobs/uploads/?mount=%s&from=%s", endpoint, repository, digest, from)
}

func buildInitiateBlobUploadURL(endpoint, repository string) string {
	return fmt.Sprintf("%s/v2/%s/blobs/uploads/", endpoint, repository)
}

func buildChunkBlobUploadURL(endpoint, location, digest string, lastChunk bool) (string, error) {
	url, err := url.Parse(location)
	if err != nil {
		return "", err
	}
	q := url.Query()
	if lastChunk {
		q.Set("digest", digest)
	}
	url.RawQuery = q.Encode()
	if url.IsAbs() {
		return url.String(), nil
	}
	// the "relativeurls" is enabled in registry
	return endpoint + url.String(), nil
}

func buildMonolithicBlobUploadURL(endpoint, location, digest string) (string, error) {
	url, err := url.Parse(location)
	if err != nil {
		return "", err
	}
	q := url.Query()
	q.Set("digest", digest)
	url.RawQuery = q.Encode()
	if url.IsAbs() {
		return url.String(), nil
	}
	// the "relativeurls" is enabled in registry
	return endpoint + url.String(), nil
}

func (c *client) GetFile(ctx context.Context, filePath string) (*commons.ResponseHeaders, io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		buildFileURL(c.url, filePath), nil)
	if err != nil {
		return nil, nil, err
	}

	resp, err := c.Do(req)
	if err != nil {
		return nil, nil, err
	}
	responseHeaders := utils.ParseResponseHeaders(resp)
	return responseHeaders, resp.Body, nil
}

func (c *client) GetFileFromURL(ctx context.Context, url string) (*commons.ResponseHeaders, io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		url, nil)
	if err != nil {
		return nil, nil, err
	}

	resp, err := c.Do(req)
	if err != nil {
		return nil, nil, err
	}
	responseHeaders := utils.ParseResponseHeaders(resp)
	return responseHeaders, resp.Body, nil
}

func (c *client) HeadFile(ctx context.Context, filePath string) (*commons.ResponseHeaders, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodHead,
		buildFileURL(c.url, filePath), nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	responseHeaders := utils.ParseResponseHeaders(resp)
	return responseHeaders, nil
}

func buildFileURL(endpoint, filePath string) string {
	return fmt.Sprintf("%s/%s", strings.TrimSuffix(endpoint, "/"), strings.TrimPrefix(filePath, "/"))
}

func (c *client) GetURL(_ context.Context, filePath string) string {
	return buildFileURL(c.url, filePath)
}

func parseContentRange(cr string) (int64, int64, error) {
	ranges := strings.Split(cr, "-")
	if len(ranges) != 2 {
		return -1, -1, fmt.Errorf("invalid content range format, %s", cr)
	}
	start, err := strconv.ParseInt(ranges[0], 10, 64)
	if err != nil {
		return -1, -1, err
	}
	end, err := strconv.ParseInt(ranges[1], 10, 64)
	if err != nil {
		return -1, -1, err
	}

	return start, end, nil
}
