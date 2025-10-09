// Source: https://github.com/distribution/distribution

// Copyright 2014 https://github.com/distribution/distribution Authors
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

package storage

import (
	"context"

	"github.com/harness/gitness/registry/app/driver"

	"github.com/opencontainers/go-digest"
)

type Service struct {
	deleteEnabled          bool
	resumableDigestEnabled bool
	redirect               bool
	driver                 driver.StorageDriver
}

// Option is the type used for functional options for NewRegistry.
type Option func(*Service) error

// EnableRedirect is a functional option for NewRegistry. It causes the backend
// blob server to attempt using (StorageDriver).RedirectURL to serve all blobs.
func EnableRedirect(registry *Service) error {
	registry.redirect = true
	return nil
}

// EnableDelete is a functional option for NewRegistry. It enables deletion on
// the registry.
func EnableDelete(registry *Service) error {
	registry.deleteEnabled = true
	return nil
}

func NewStorageService(driver driver.StorageDriver, options ...Option) (*Service, error) {
	registry := &Service{
		resumableDigestEnabled: true,
		driver:                 driver,
	}

	for _, option := range options {
		if err := option(registry); err != nil {
			return nil, err
		}
	}

	return registry, nil
}

func (storage *Service) OciBlobsStore(ctx context.Context, repoKey string, rootParentRef string) OciBlobStore {
	return &ociBlobStore{
		repoKey:                repoKey,
		ctx:                    ctx,
		driver:                 storage.driver,
		pathFn:                 PathFn,
		redirect:               storage.redirect,
		deleteEnabled:          storage.deleteEnabled,
		resumableDigestEnabled: storage.resumableDigestEnabled,
		rootParentRef:          rootParentRef,
	}
}

func (storage *Service) GenericBlobsStore(rootParentRef string) GenericBlobStore {
	return &genericBlobStore{
		driver:        storage.driver,
		redirect:      storage.redirect,
		rootParentRef: rootParentRef,
	}
}

// path returns the canonical path for the blob identified by digest. The blob
// may or may not exist.
func PathFn(pathPrefix string, dgst digest.Digest) (string, error) {
	bp, err := pathFor(
		blobDataPathSpec{
			digest: dgst,
			path:   pathPrefix,
		},
	)
	if err != nil {
		return "", err
	}

	return bp, nil
}
