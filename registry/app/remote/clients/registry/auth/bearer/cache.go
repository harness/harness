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

package bearer

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

// newCache initializes a new cache with the specified capacity and latency.
// The latency is the delay applied to cache operations (if applicable).
func newCache(capacity int, latency int) *cache {
	return &cache{
		latency:  latency,
		capacity: capacity,
		cache:    map[string]*token{},
	}
}

type cache struct {
	sync.RWMutex
	latency int // second, the network latency in case that when the
	// token is checked it doesn't expire but it does when used.
	capacity int // the capacity of the cache map.
	cache    map[string]*token
}

func (c *cache) get(scopes []*scope) *token {
	c.RLock()
	defer c.RUnlock()
	token := c.cache[c.key(scopes)]
	if token == nil {
		return nil
	}
	expired, _ := c.expired(token)
	if expired {
		token = nil
	}
	return token
}

func (c *cache) set(scopes []*scope, token *token) {
	c.Lock()
	defer c.Unlock()
	// exceed the capacity, empty some elements: all expired token will be removed,
	// if no expired token, move the earliest one.
	if len(c.cache) >= c.capacity {
		var candidates []string
		var earliestKey string
		var earliestExpireTime time.Time
		for key, value := range c.cache {
			expired, expireAt := c.expired(value)
			// expired.
			if expired {
				candidates = append(candidates, key)
				continue
			}
			// doesn't expired.
			if len(earliestKey) == 0 || expireAt.Before(earliestExpireTime) {
				earliestKey = key
				earliestExpireTime = expireAt
				continue
			}
		}
		if len(candidates) == 0 {
			candidates = append(candidates, earliestKey)
		}
		for _, candidate := range candidates {
			delete(c.cache, candidate)
		}
	}
	c.cache[c.key(scopes)] = token
}

func (c *cache) key(scopes []*scope) string {
	var strs []string
	for _, scope := range scopes {
		strs = append(strs, scope.String())
	}
	return strings.Join(strs, "#")
}

// return whether the token is expired or not and the expired time.
func (c *cache) expired(token *token) (bool, time.Time) {
	// check time whether empty.
	if len(token.IssuedAt) == 0 {
		log.Warn().Msg("token issued time is empty, return expired to refresh token")
		return true, time.Time{}
	}

	issueAt, err := time.Parse(time.RFC3339, token.IssuedAt)
	if err != nil {
		log.Error().
			Stack().
			Err(err).
			Msg(fmt.Sprintf("failed to parse the issued at time of token %s: %v", token.IssuedAt, err))
		return true, time.Time{}
	}
	expireAt := issueAt.Add(time.Duration(token.ExpiresIn-c.latency) * time.Second)
	return expireAt.Before(time.Now()), expireAt
}
