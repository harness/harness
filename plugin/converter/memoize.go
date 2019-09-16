// Copyright 2019 Drone IO, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// +build !oss

package converter

import (
	"context"
	"fmt"

	"github.com/drone/drone/core"

	lru "github.com/hashicorp/golang-lru"
)

// cache key pattern used in the cache, comprised of the
// repository slug and commit sha.
const keyf = "%d/%s"

// Memoize caches the conversion results for subsequent calls.
// This micro-optimization is intended for multi-pipeline
// projects that would otherwise covert the file for each
// pipeline execution.
func Memoize(base core.ConvertService) core.ConvertService {
	// simple cache prevents the same yaml file from being
	// requested multiple times in a short period.
	cache, _ := lru.New(25)
	return &memoize{base: base, cache: cache}
}

type memoize struct {
	base  core.ConvertService
	cache *lru.Cache
}

func (c *memoize) Convert(ctx context.Context, req *core.ConvertArgs) (*core.Config, error) {
	// this is a minor optimization that prevents caching if the
	// base converter is a remote converter and is disabled.
	if remote, ok := c.base.(*remote); ok == true && remote.endpoint == "" {
		return nil, nil
	}

	// generate the key used to cache the converted file.
	key := fmt.Sprintf(keyf, req.Repo.ID, req.Build.After)

	// some source control management systems (gogs) do not provide
	// the commit sha for tag webhooks. If no commit sha is available
	// the ref is used.
	if req.Build.After == "" {
		key = fmt.Sprintf(keyf, req.Repo.ID, req.Build.Ref)
	}

	// check the cache for the file and return if exists.
	cached, ok := c.cache.Get(key)
	if ok {
		return cached.(*core.Config), nil
	}

	// else convert the configuration file.
	config, err := c.base.Convert(ctx, req)
	if err != nil {
		return nil, err
	}

	if config == nil {
		return nil, nil
	}
	if config.Data == "" {
		return nil, nil
	}

	// if the configuration file was converted
	// it is temporarily cached.
	c.cache.Add(key, config)
	return config, nil
}
