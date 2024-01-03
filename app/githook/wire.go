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

package githook

import (
	"github.com/harness/gitness/app/api/controller/githook"
	"github.com/harness/gitness/app/api/controller/limiter"
	"github.com/harness/gitness/app/auth/authz"
	eventsgit "github.com/harness/gitness/app/events/git"
	"github.com/harness/gitness/app/services/protection"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/app/url"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/git/hook"

	"github.com/google/wire"
)

// WireSet provides a wire set for this package.
var WireSet = wire.NewSet(
	ProvideController,
	ProvideFactory,
)

func ProvideFactory() hook.ClientFactory {
	return &ControllerClientFactory{
		// will be set in ProvideController (to break cyclic dependency during wiring)
		githookCtrl: nil,
	}
}

func ProvideController(
	authorizer authz.Authorizer,
	principalStore store.PrincipalStore,
	repoStore store.RepoStore,
	gitReporter *eventsgit.Reporter,
	git git.Interface,
	pullreqStore store.PullReqStore,
	urlProvider url.Provider,
	protectionManager *protection.Manager,
	githookFactory hook.ClientFactory,
	limiter limiter.ResourceLimiter,
) *githook.Controller {
	ctrl := githook.NewController(
		authorizer,
		principalStore,
		repoStore,
		gitReporter,
		git,
		pullreqStore,
		urlProvider,
		protectionManager,
		limiter)

	// TODO: improve wiring if possible
	if fct, ok := githookFactory.(*ControllerClientFactory); ok {
		fct.githookCtrl = ctrl
	}

	return ctrl
}
