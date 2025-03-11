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

package lfs

import (
	"github.com/harness/gitness/app/auth/authz"
	"github.com/harness/gitness/app/services/refcache"
	"github.com/harness/gitness/app/services/remoteauth"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/app/url"
	"github.com/harness/gitness/blob"

	"github.com/google/wire"
)

var WireSet = wire.NewSet(
	ProvideController,
)

func ProvideController(
	authorizer authz.Authorizer,
	repoFinder refcache.RepoFinder,
	principalStore store.PrincipalStore,
	lfsStore store.LFSObjectStore,
	blobStore blob.Store,
	remoteAuth remoteauth.Service,
	urlProvider url.Provider,
) *Controller {
	return NewController(authorizer, repoFinder, principalStore, lfsStore, blobStore, remoteAuth, urlProvider)
}
