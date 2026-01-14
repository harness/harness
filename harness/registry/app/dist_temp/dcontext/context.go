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

package dcontext

import (
	"context"
	"sync"

	"github.com/google/uuid"
)

// instanceContext is a context that provides only an instance id. It is
// provided as the main background context.
type instanceContext struct {
	context.Context
	id   string    // id of context, logged as "instance.id"
	once sync.Once // once protect generation of the id
}

func (ic *instanceContext) Value(key interface{}) interface{} {
	if key == "instance.id" {
		ic.once.Do(
			func() {
				// We want to lazy initialize the UUID such that we don't
				// call a random generator from the package initialization
				// code. For various reasons random could not be available
				// https://github.com/distribution/distribution/issues/782
				ic.id = uuid.NewString()
			},
		)
		return ic.id
	}

	return ic.Context.Value(key)
}

var background = &instanceContext{
	Context: context.Background(),
}

// Background returns a non-nil, empty Context. The background context
// provides a single key, "instance.id" that is globally unique to the
// process.
func Background() context.Context {
	return background
}

// stringMapContext is a simple context implementation that checks a map for a
// key, falling back to a parent if not present.
type stringMapContext struct {
	context.Context
	m map[string]interface{}
}

// WithValues returns a context that proxies lookups through a map. Only
// supports string keys.
func WithValues(ctx context.Context, m map[string]interface{}) context.Context {
	mo := make(map[string]interface{}, len(m)) // make our own copy.
	for k, v := range m {
		mo[k] = v
	}

	return stringMapContext{
		Context: ctx,
		m:       mo,
	}
}

func (smc stringMapContext) Value(key interface{}) interface{} {
	if ks, ok := key.(string); ok {
		if v, ok1 := smc.m[ks]; ok1 {
			return v
		}
	}

	return smc.Context.Value(key)
}
