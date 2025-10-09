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

package gitspace

import (
	"github.com/harness/gitness/app/api/controller/limiter"
	"github.com/harness/gitness/app/auth/authz"
	"github.com/harness/gitness/app/gitspace/logutil"
	"github.com/harness/gitness/app/gitspace/scm"
	"github.com/harness/gitness/app/services/gitspace"
	"github.com/harness/gitness/app/services/gitspacesettings"
	"github.com/harness/gitness/app/services/infraprovider"
	"github.com/harness/gitness/app/services/refcache"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/store/database/dbtx"

	"github.com/google/wire"
)

// WireSet provides a wire set for this package.
var WireSet = wire.NewSet(
	ProvideController,
)

func ProvideController(
	tx dbtx.Transactor,
	authorizer authz.Authorizer,
	infraProviderSvc *infraprovider.Service,
	spaceStore store.SpaceStore,
	spaceFinder refcache.SpaceFinder,
	eventStore store.GitspaceEventStore,
	statefulLogger *logutil.StatefulLogger,
	scm *scm.SCM,
	gitspaceSvc *gitspace.Service,
	gitspaceLimiter limiter.Gitspace,
	repoFinder refcache.RepoFinder,
	settingsService gitspacesettings.Service,
) *Controller {
	return NewController(
		tx,
		authorizer,
		infraProviderSvc,
		spaceStore,
		spaceFinder,
		eventStore,
		statefulLogger,
		scm,
		gitspaceSvc,
		gitspaceLimiter,
		repoFinder,
		settingsService,
	)
}
