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

package repo

import (
	"github.com/harness/gitness/app/api/controller/lfs"
	"github.com/harness/gitness/app/api/controller/limiter"
	"github.com/harness/gitness/app/auth/authz"
	repoevents "github.com/harness/gitness/app/events/repo"
	"github.com/harness/gitness/app/services/autolink"
	"github.com/harness/gitness/app/services/codeowners"
	"github.com/harness/gitness/app/services/dotrange"
	"github.com/harness/gitness/app/services/importer"
	"github.com/harness/gitness/app/services/instrument"
	"github.com/harness/gitness/app/services/keywordsearch"
	"github.com/harness/gitness/app/services/label"
	"github.com/harness/gitness/app/services/locker"
	"github.com/harness/gitness/app/services/protection"
	"github.com/harness/gitness/app/services/publicaccess"
	"github.com/harness/gitness/app/services/publickey"
	"github.com/harness/gitness/app/services/refcache"
	"github.com/harness/gitness/app/services/rules"
	"github.com/harness/gitness/app/services/settings"
	"github.com/harness/gitness/app/services/usergroup"
	"github.com/harness/gitness/app/sse"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/app/url"
	"github.com/harness/gitness/audit"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/lock"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/check"

	"github.com/google/wire"
)

// WireSet provides a wire set for this package.
var WireSet = wire.NewSet(
	ProvideController,
)

func ProvideController(
	config *types.Config,
	tx dbtx.Transactor,
	urlProvider url.Provider,
	authorizer authz.Authorizer,
	repoStore store.RepoStore,
	linkedRepoStore store.LinkedRepoStore,
	spaceStore store.SpaceStore,
	pipelineStore store.PipelineStore,
	principalStore store.PrincipalStore,
	executionStore store.ExecutionStore,
	ruleStore store.RuleStore,
	checkStore store.CheckStore,
	pullReqStore store.PullReqStore,
	settings *settings.Service,
	principalInfoCache store.PrincipalInfoCache,
	protectionManager *protection.Manager,
	rpcClient git.Interface,
	spaceFinder refcache.SpaceFinder,
	repoFinder refcache.RepoFinder,
	importer *importer.JobRepository,
	referenceSync *importer.JobReferenceSync,
	importLinked *importer.JobRepositoryLink,
	codeOwners *codeowners.Service,
	repoReporter *repoevents.Reporter,
	indexer keywordsearch.Indexer,
	limiter limiter.ResourceLimiter,
	locker *locker.Locker,
	auditService audit.Service,
	mtxManager lock.MutexManager,
	identifierCheck check.RepoIdentifier,
	repoChecks Check,
	publicAccess publicaccess.Service,
	labelSvc *label.Service,
	instrumentation instrument.Service,
	userGroupStore store.UserGroupStore,
	userGroupService usergroup.Service,
	rulesSvc *rules.Service,
	sseStreamer sse.Streamer,
	lfsCtrl *lfs.Controller,
	favoriteStore store.FavoriteStore,
	signatureVerifyService publickey.SignatureVerifyService,
	autolinkSvc *autolink.Service,
	dotRangeService *dotrange.Service,
	connectorService importer.ConnectorService,
	repoLangStore store.RepoLangStore,
) *Controller {
	return NewController(config, tx, urlProvider,
		authorizer,
		repoStore, linkedRepoStore, spaceStore, pipelineStore, executionStore,
		principalStore, ruleStore, checkStore, pullReqStore, settings,
		principalInfoCache, protectionManager, rpcClient, spaceFinder, repoFinder,
		importer, referenceSync, importLinked,
		codeOwners, repoReporter, indexer, limiter, locker, auditService, mtxManager, identifierCheck,
		repoChecks, publicAccess, labelSvc, instrumentation, userGroupStore, userGroupService,
		rulesSvc, sseStreamer, lfsCtrl, favoriteStore, signatureVerifyService,
		autolinkSvc, dotRangeService, connectorService,
		repoLangStore,
	)
}

func ProvideRepoCheck() Check {
	return NewNoOpRepoChecks()
}
