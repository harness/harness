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
	"context"
	"io"

	"github.com/harness/gitness/registry/app/manifest"
	"github.com/harness/gitness/registry/app/pkg"
	"github.com/harness/gitness/registry/app/pkg/commons"
	"github.com/harness/gitness/registry/app/storage"

	"github.com/opencontainers/go-digest"
)

// registryInterface defines operations related to localRegistry registry under proxy mode.
type registryInterface interface {
	Base() error
	PullManifest(
		ctx context.Context,
		artInfo pkg.RegistryInfo,
		acceptHeaders []string,
		ifNoneMatchHeader []string,
	) (
		responseHeaders *commons.ResponseHeaders,
		descriptor manifest.Descriptor,
		manifest manifest.Manifest,
		Errors []error,
	)
	PutManifest(
		ctx context.Context,
		artInfo pkg.RegistryInfo,
		mediaType string,
		body io.ReadCloser,
		length int64,
	) (*commons.ResponseHeaders, []error)
	DeleteManifest(
		ctx context.Context,
		artInfo pkg.RegistryInfo,
	) (errs []error, responseHeaders *commons.ResponseHeaders)
	GetBlob(
		ctx2 context.Context,
		artInfo pkg.RegistryInfo,
	) (
		responseHeaders *commons.ResponseHeaders, fr *storage.FileReader,
		size int64, readCloser io.ReadCloser, redirectURL string,
		Errors []error,
	)
	InitBlobUpload(
		ctx context.Context,
		artInfo pkg.RegistryInfo,
		fromRepo, mountDigest string,
	) (*commons.ResponseHeaders, []error)
	PushBlob(
		ctx2 context.Context,
		artInfo pkg.RegistryInfo,
		body io.ReadCloser,
		contentLength int64,
		stateToken string,
	) (*commons.ResponseHeaders, []error)
}

// registryInterface defines operations related to localRegistry registry under proxy mode.
type registryManifestInterface interface {
	DBTag(
		ctx context.Context,
		mfst manifest.Manifest,
		d digest.Digest,
		tag string,
		repoKey string,
		headers *commons.ResponseHeaders,
		info pkg.RegistryInfo,
	) error
}
