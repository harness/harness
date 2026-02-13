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

	"github.com/harness/gitness/registry/types"

	"github.com/opencontainers/go-digest"
	"github.com/rs/zerolog/log"
)

type Service struct {
	deleteEnabled          bool
	resumableDigestEnabled bool
	redirect               bool
	storageResolver        StorageResolver
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

func NewStorageService(resolver StorageResolver, options ...Option) (*Service, error) {
	svc := &Service{
		resumableDigestEnabled: true,
		storageResolver:        resolver,
	}

	for _, option := range options {
		if err := option(svc); err != nil {
			return nil, err
		}
	}

	return svc, nil
}

func (s *Service) OciBlobsStore(
	ctx context.Context,
	repoKey string,
	rootParentRef string,
	locator types.BlobLocator,
) OciBlobStore {
	target, err := s.storageResolver.Resolve(ctx, types.StorageLookup{
		BlobLocator: locator,
	})
	if err != nil {
		// TODO(Arvind): Return this error
		log.Ctx(ctx).Fatal().Err(err).Msgf("Failed to resolve storage target for %s", locator.String())
		return nil
	}

	if !target.IsDefault() {
		return s.GlobalBlobsStore(ctx, target, true)
	}

	return &ociBlobStore{
		repoKey:                repoKey,
		ctx:                    ctx,
		driver:                 target.Driver,
		pathFn:                 PathFn,
		redirect:               s.redirect,
		deleteEnabled:          s.deleteEnabled,
		resumableDigestEnabled: s.resumableDigestEnabled,
		rootParentRef:          rootParentRef,
	}
}

func (s *Service) GenericBlobsStore(
	ctx context.Context,
	rootParentRef string,
	locator types.BlobLocator,
) GenericBlobStore {
	target, err := s.storageResolver.Resolve(ctx, types.StorageLookup{
		BlobLocator: locator,
	})

	if err != nil {
		// TODO(Arvind): Return this error
		log.Ctx(ctx).Fatal().Err(err).Msgf("Failed to resolve storage target for %s", locator.String())
		return nil
	}

	if !target.IsDefault() {
		return s.GlobalBlobsStore(ctx, target, false)
	}

	return &genericBlobStore{
		driver:        target.Driver,
		redirect:      s.redirect,
		rootParentRef: rootParentRef,
	}
}

func (s *Service) GlobalBlobsStore(ctx context.Context, target StorageTarget, oci bool) GlobalBlobStore {
	return &globalBlobStore{
		bucketKey:              target.BucketKey,
		driver:                 target.Driver,
		ctx:                    ctx,
		resumableDigestEnabled: s.resumableDigestEnabled,
		redirect:               s.redirect,
		deleteEnabled:          s.deleteEnabled,
		multipartEnabled:       oci,
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
