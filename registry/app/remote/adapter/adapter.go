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

package adapter

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/harness/gitness/app/services/refcache"
	"github.com/harness/gitness/registry/app/manifest"
	"github.com/harness/gitness/registry/app/pkg/commons"
	"github.com/harness/gitness/registry/types"
	"github.com/harness/gitness/secret"
)

// const definition.
const (
	MaxConcurrency = 100
)

var registry = map[string]Factory{}
var registryKeys = []string{}

// Factory creates a specific Adapter according to the params.
type Factory interface {
	Create(
		ctx context.Context, spaceFinder refcache.SpaceFinder, record types.UpstreamProxy, service secret.Service,
	) (Adapter, error)
}

// Adapter interface defines the capabilities of registry.
type Adapter interface {
	// HealthCheck checks health status of registry.
	HealthCheck() (string, error)
	GetImageName(imageName string) (string, error)
}

// ArtifactRegistry defines the capabilities that an artifact registry should have.
type ArtifactRegistry interface {
	ManifestExist(ctx context.Context, repository, reference string) (exist bool, desc *manifest.Descriptor, err error)
	PullManifest(
		ctx context.Context,
		repository, reference string,
		accepttedMediaTypes ...string,
	) (manifest manifest.Manifest, digest string, err error)
	PushManifest(ctx context.Context, repository, reference, mediaType string, payload []byte) (string, error)
	DeleteManifest(ctx context.Context, repository, reference string) error
	// the "reference" can be "tag" or "digest", the function needs to handle both
	BlobExist(ctx context.Context, repository, digest string) (exist bool, err error)
	PullBlob(ctx context.Context, repository, digest string) (size int64, blob io.ReadCloser, err error)
	PullBlobChunk(ctx context.Context, repository, digest string, blobSize, start, end int64) (
		size int64,
		blob io.ReadCloser,
		err error,
	)
	PushBlobChunk(
		ctx context.Context,
		repository, digest string,
		size int64,
		chunk io.Reader,
		start, end int64,
		location string,
	) (nextUploadLocation string, endRange int64, err error)
	PushBlob(ctx context.Context, repository, digest string, size int64, blob io.Reader) error
	MountBlob(ctx context.Context, srcRepository, digest, dstRepository string) (err error)
	CanBeMount(ctx context.Context, digest string) (mount bool, repository string, err error)
	// check whether the blob can be mounted from the remote registry
	DeleteTag(ctx context.Context, repository, tag string) error
	ListTags(ctx context.Context, repository string) (tags []string, err error)
	// Download the file
	GetFile(ctx context.Context, filePath string) (*commons.ResponseHeaders, io.ReadCloser, error)
	// Check existence of file
	HeadFile(ctx context.Context, filePath string) (*commons.ResponseHeaders, bool, error)
}

// RegisterFactory registers one adapter factory to the registry.
func RegisterFactory(t string, factory Factory) error {
	if len(t) == 0 {
		return errors.New("invalid type")
	}
	if factory == nil {
		return errors.New("empty adapter factory")
	}

	if _, exist := registry[t]; exist {
		return fmt.Errorf("adapter factory for %s already exists", t)
	}
	registry[t] = factory
	registryKeys = append(registryKeys, t)
	return nil
}

// GetFactory gets the adapter factory by the specified name.
func GetFactory(t string) (Factory, error) {
	factory, exist := registry[t]
	if !exist {
		return nil, fmt.Errorf("adapter factory for %s not found", t)
	}
	return factory, nil
}

// ListRegisteredAdapterTypes lists the registered Adapter type.
func ListRegisteredAdapterTypes() []string {
	return registryKeys
}
