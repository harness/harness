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
	"github.com/harness/gitness/app/api/controller/limiter"
	"github.com/harness/gitness/app/auth/authz"
	eventsgit "github.com/harness/gitness/app/events/git"
	eventsrepo "github.com/harness/gitness/app/events/repo"
	"github.com/harness/gitness/app/services/protection"
	"github.com/harness/gitness/app/services/refcache"
	"github.com/harness/gitness/app/services/settings"
	"github.com/harness/gitness/app/services/usergroup"
	"github.com/harness/gitness/app/sse"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/app/url"
	"github.com/harness/gitness/audit"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/git/hook"

	"github.com/google/wire"
)

var WireSet = wire.NewSet(
	ProvideController,
	ProvideFactory,
)

func ProvideFactory() hook.ClientFactory {
	return &ControllerClientFactory{
		// fields are set in ProvideController to avoid import
		githookCtrl: nil,
		git:         nil,
	}
}

func ProvideController(
	authorizer authz.Authorizer,
	principalStore store.PrincipalStore,
	repoStore store.RepoStore,
	repoFinder refcache.RepoFinder,
	gitReporter *eventsgit.Reporter,
	repoReporter *eventsrepo.Reporter,
	git git.Interface,
	pullreqStore store.PullReqStore,
	urlProvider url.Provider,
	protectionManager *protection.Manager,
	githookFactory hook.ClientFactory,
	limiter limiter.ResourceLimiter,
	settings *settings.Service,
	preReceiveExtender PreReceiveExtender,
	updateExtender UpdateExtender,
	postReceiveExtender PostReceiveExtender,
	sseStreamer sse.Streamer,
	lfsStore store.LFSObjectStore,
	auditService audit.Service,
	userGroupService usergroup.Service,
) *Controller {
	ctrl := NewController(
		authorizer,
		principalStore,
		repoStore,
		repoFinder,
		gitReporter,
		repoReporter,
		pullreqStore,
		urlProvider,
		protectionManager,
		limiter,
		settings,
		preReceiveExtender,
		updateExtender,
		postReceiveExtender,
		sseStreamer,
		lfsStore,
		auditService,
		userGroupService,
	)

	// TODO: improve wiring if possible
	if fct, ok := githookFactory.(*ControllerClientFactory); ok {
		fct.githookCtrl = ctrl
		fct.git = git
	}

	return ctrl
}

var ExtenderWireSet = wire.NewSet(
	ProvidePreReceiveExtender,
	ProvideUpdateExtender,
	ProvidePostReceiveExtender,
)

func ProvidePreReceiveExtender() (PreReceiveExtender, error) {
	return NewPreReceiveExtender(), nil
}

func ProvideUpdateExtender() (UpdateExtender, error) {
	return NewUpdateExtender(), nil
}

func ProvidePostReceiveExtender() (PostReceiveExtender, error) {
	return NewPostReceiveExtender(), nil
}
