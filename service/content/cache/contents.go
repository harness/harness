// Copyright 2019 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by the Drone Non-Commercial License
// that can be found in the LICENSE file.

// +build !oss

package cache

import (
	"context"
	"fmt"

	"github.com/drone/drone/core"

	"github.com/hashicorp/golang-lru"
)

// content key pattern used in the cache, comprised of the
// repository slug, commit and path.
const contentKey = "%s/%s/%s"

// Contents returns a new FileService that is wrapped
// with an in-memory cache.
func Contents(base core.FileService) core.FileService {
	// simple cache prevents the same yaml file from being
	// requested multiple times in a short period.
	cache, _ := lru.New(25)
	return &service{
		service: base,
		cache:   cache,
	}
}

type service struct {
	cache   *lru.Cache
	service core.FileService
	user    *core.User
}

func (s *service) Find(ctx context.Context, user *core.User, repo, commit, ref, path string) (*core.File, error) {
	key := fmt.Sprintf(contentKey, repo, commit, path)
	cached, ok := s.cache.Get(key)
	if ok {
		return cached.(*core.File), nil
	}
	file, err := s.service.Find(ctx, user, repo, commit, ref, path)
	if err != nil {
		return nil, err
	}
	s.cache.Add(key, file)
	return file, nil
}
