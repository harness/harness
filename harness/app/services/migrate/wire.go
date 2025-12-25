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

package migrate

import (
	"github.com/harness/gitness/app/services/refcache"
	"github.com/harness/gitness/app/services/webhook"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/app/url"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/lock"
	"github.com/harness/gitness/store/database/dbtx"

	"github.com/google/wire"
)

var WireSet = wire.NewSet(
	ProvidePullReqImporter,
	ProvideRuleImporter,
	ProvideWebhookImporter,
	ProvideLabelImporter,
)

func ProvidePullReqImporter(
	urlProvider url.Provider,
	git git.Interface,
	principalStore store.PrincipalStore,
	spaceStore store.SpaceStore,
	repoStore store.RepoStore,
	pullReqStore store.PullReqStore,
	pullReqActStore store.PullReqActivityStore,
	labelStore store.LabelStore,
	labelValueStore store.LabelValueStore,
	pullReqLabelAssignmentStore store.PullReqLabelAssignmentStore,
	pullReqReviewerStore store.PullReqReviewerStore,
	pullReqReviewStore store.PullReqReviewStore,
	repoFinder refcache.RepoFinder,
	tx dbtx.Transactor,
	mtxManager lock.MutexManager,
) *PullReq {
	return NewPullReq(
		urlProvider, git, principalStore, spaceStore, repoStore, pullReqStore, pullReqActStore,
		labelStore, labelValueStore, pullReqLabelAssignmentStore, pullReqReviewerStore, pullReqReviewStore,
		repoFinder, tx, mtxManager)
}

func ProvideRuleImporter(
	ruleStore store.RuleStore,
	tx dbtx.Transactor,
	principalStore store.PrincipalStore,
) *Rule {
	return NewRule(ruleStore, tx, principalStore)
}

func ProvideWebhookImporter(
	config webhook.Config,
	tx dbtx.Transactor,
	webhookStore store.WebhookStore,
) *Webhook {
	return NewWebhook(config, tx, webhookStore)
}

func ProvideLabelImporter(
	tx dbtx.Transactor,
	labelStore store.LabelStore,
	labelValueStore store.LabelValueStore,
	spaceStore store.SpaceStore,
) *Label {
	return NewLabel(labelStore, labelValueStore, spaceStore, tx)
}
