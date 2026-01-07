//  Copyright 2023 Harness, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package publicaccess

import (
	"context"
	"encoding/gob"
	"fmt"
	"time"

	"github.com/harness/gitness/app/services/publicaccess"
	cache2 "github.com/harness/gitness/app/store/cache"
	"github.com/harness/gitness/cache"
	"github.com/harness/gitness/types/enum"
)

type CacheService interface {
	Get(
		ctx context.Context,
		resourceType enum.PublicResourceType,
		resourcePath string,
	) (bool, error)

	Set(
		ctx context.Context,
		resourceType enum.PublicResourceType,
		resourcePath string,
		enable bool,
	) error

	Delete(
		ctx context.Context,
		resourceType enum.PublicResourceType,
		resourcePath string,
	) error

	IsPublicAccessSupported(
		ctx context.Context,
		resourceType enum.PublicResourceType,
		parentSpacePath string,
	) (bool, error)

	MarkChanged(ctx context.Context, publicAccessCacheKey *CacheKey)
}

type CacheKey struct {
	ResourceType enum.PublicResourceType
	ResourcePath string
}

type Cache cache.Cache[CacheKey, bool]

func NewPublicAccessService(
	publicAccessService publicaccess.Service,
	publicAccessCache Cache,
	evictor cache2.Evictor[*CacheKey],
) CacheService {
	gob.Register(&CacheKey{})
	return &service{
		publicAccessService: publicAccessService,
		publicAccessCache:   publicAccessCache,
		evictor:             evictor,
	}
}

type service struct {
	publicAccessService publicaccess.Service
	publicAccessCache   Cache
	evictor             cache2.Evictor[*CacheKey]
}

func NewPublicAccessCacheCache(
	appCtx context.Context,
	publicAccessService publicaccess.Service,
	evictor cache2.Evictor[*CacheKey],
	dur time.Duration,
) Cache {
	c := cache.New[CacheKey, bool](publicAccessCacheGetter{publicAccessService: publicAccessService}, dur)

	evictor.Subscribe(appCtx, func(key *CacheKey) error {
		c.Evict(appCtx, *key)
		return nil
	})

	return c
}

func (s *service) Get(
	ctx context.Context,
	resourceType enum.PublicResourceType,
	resourcePath string,
) (bool, error) {
	isPublic, err := s.publicAccessCache.Get(ctx,
		CacheKey{ResourceType: resourceType, ResourcePath: resourcePath})
	if err != nil {
		return false, fmt.Errorf("failed to get public access: %w", err)
	}

	return isPublic, nil
}

func (s *service) Set(
	ctx context.Context,
	resourceType enum.PublicResourceType,
	resourcePath string,
	enable bool,
) error {
	err := s.publicAccessService.Set(ctx, resourceType, resourcePath, enable)
	if err == nil {
		s.MarkChanged(ctx, &CacheKey{ResourceType: resourceType, ResourcePath: resourcePath})
	}
	return err
}

func (s *service) Delete(
	ctx context.Context,
	resourceType enum.PublicResourceType,
	resourcePath string,
) error {
	err := s.publicAccessService.Delete(ctx, resourceType, resourcePath)
	if err == nil {
		s.MarkChanged(ctx, &CacheKey{ResourceType: resourceType, ResourcePath: resourcePath})
	}
	return err
}

func (s *service) IsPublicAccessSupported(
	ctx context.Context,
	resourceType enum.PublicResourceType,
	resourcePath string,
) (bool, error) {
	return s.publicAccessService.IsPublicAccessSupported(ctx, resourceType, resourcePath)
}

func (s *service) MarkChanged(ctx context.Context, publicAccessCacheKey *CacheKey) {
	s.evictor.Evict(ctx, publicAccessCacheKey)
}

type publicAccessCacheGetter struct {
	publicAccessService publicaccess.Service
}

func (c publicAccessCacheGetter) Find(
	ctx context.Context,
	publicAccessCacheKey CacheKey,
) (bool, error) {
	isPublic, err := c.publicAccessService.Get(ctx, publicAccessCacheKey.ResourceType, publicAccessCacheKey.ResourcePath)
	if err != nil {
		return false, fmt.Errorf("failed to find repo by ID: %w", err)
	}

	return isPublic, nil
}
