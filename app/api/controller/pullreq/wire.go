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

package pullreq

import (
	"github.com/harness/gitness/app/auth/authz"
	pullreqevents "github.com/harness/gitness/app/events/pullreq"
	"github.com/harness/gitness/app/services/codecomments"
	"github.com/harness/gitness/app/services/codeowners"
	"github.com/harness/gitness/app/services/label"
	"github.com/harness/gitness/app/services/locker"
	"github.com/harness/gitness/app/services/migrate"
	"github.com/harness/gitness/app/services/protection"
	"github.com/harness/gitness/app/services/pullreq"
	"github.com/harness/gitness/app/sse"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/app/url"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/store/database/dbtx"

	"github.com/google/wire"
)

// WireSet provides a wire set for this package.
var WireSet = wire.NewSet(
	ProvideController,
)

func ProvideController(tx dbtx.Transactor, urlProvider url.Provider, authorizer authz.Authorizer,
	pullReqStore store.PullReqStore, pullReqActivityStore store.PullReqActivityStore,
	codeCommentsView store.CodeCommentView,
	pullReqReviewStore store.PullReqReviewStore, pullReqReviewerStore store.PullReqReviewerStore,
	repoStore store.RepoStore,
	principalStore store.PrincipalStore, principalInfoCache store.PrincipalInfoCache,
	fileViewStore store.PullReqFileViewStore, membershipStore store.MembershipStore,
	checkStore store.CheckStore,
	rpcClient git.Interface, eventReporter *pullreqevents.Reporter, codeCommentMigrator *codecomments.Migrator,
	pullreqService *pullreq.Service, ruleManager *protection.Manager, sseStreamer sse.Streamer,
	codeOwners *codeowners.Service, locker *locker.Locker, importer *migrate.PullReq,
	labelSvc *label.Service,
) *Controller {
	return NewController(tx, urlProvider, authorizer,
		pullReqStore, pullReqActivityStore,
		codeCommentsView,
		pullReqReviewStore, pullReqReviewerStore,
		repoStore,
		principalStore, principalInfoCache,
		fileViewStore, membershipStore,
		checkStore,
		rpcClient, eventReporter,
		codeCommentMigrator,
		pullreqService, ruleManager, sseStreamer, codeOwners, locker, importer, labelSvc)
}
