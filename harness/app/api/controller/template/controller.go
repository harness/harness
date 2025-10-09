// Copyright 2023 Harness, Inc.
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

package template

import (
	"github.com/harness/gitness/app/auth/authz"
	"github.com/harness/gitness/app/services/refcache"
	"github.com/harness/gitness/app/store"
)

type Controller struct {
	templateStore store.TemplateStore
	authorizer    authz.Authorizer
	spaceFinder   refcache.SpaceFinder
}

func NewController(
	authorizer authz.Authorizer,
	templateStore store.TemplateStore,
	spaceFinder refcache.SpaceFinder,
) *Controller {
	return &Controller{
		templateStore: templateStore,
		authorizer:    authorizer,
		spaceFinder:   spaceFinder,
	}
}
