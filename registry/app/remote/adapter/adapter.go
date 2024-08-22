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

	store2 "github.com/harness/gitness/app/store"
	"github.com/harness/gitness/encrypt"
	"github.com/harness/gitness/registry/app/manifest"
	"github.com/harness/gitness/registry/types"
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
		ctx context.Context, secretStore store2.SecretStore, encrypter encrypt.Encrypter,
		record types.UpstreamProxy,
	) (Adapter, error)
}

// Adapter interface defines the capabilities of registry.
type Adapter interface {
	// HealthCheck checks health status of registry.
	HealthCheck() (string, error)
}

// ArtifactRegistry defines the capabilities that an artifact registry should have.
type ArtifactRegistry interface {
	ManifestExist(repository, reference string) (exist bool, desc *manifest.Descriptor, err error)
	PullManifest(
		repository, reference string,
		accepttedMediaTypes ...string,
	) (manifest manifest.Manifest, digest string, err error)
	PushManifest(repository, reference, mediaType string, payload []byte) (string, error)
	DeleteManifest(
		repository, reference string,
	) error // the "reference" can be "tag" or "digest", the function needs to handle both
	BlobExist(repository, digest string) (exist bool, err error)
	PullBlob(repository, digest string) (size int64, blob io.ReadCloser, err error)
	PullBlobChunk(repository, digest string, blobSize, start, end int64) (size int64, blob io.ReadCloser, err error)
	PushBlobChunk(
		repository, digest string,
		size int64,
		chunk io.Reader,
		start, end int64,
		location string,
	) (nextUploadLocation string, endRange int64, err error)
	PushBlob(repository, digest string, size int64, blob io.Reader) error
	MountBlob(srcRepository, digest, dstRepository string) (err error)
	CanBeMount(
		digest string,
	) (mount bool, repository string, err error) // check whether the blob can be mounted from the remote registry
	DeleteTag(repository, tag string) error
	ListTags(repository string) (tags []string, err error)
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
