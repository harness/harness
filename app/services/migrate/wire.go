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
	"github.com/harness/gitness/app/services/webhook"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/app/url"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/store/database/dbtx"

	"github.com/google/wire"
)

var WireSet = wire.NewSet(
	ProvidePullReqImporter,
	ProvideRuleImporter,
	ProvideWebhookImporter,
)

func ProvidePullReqImporter(
	urlProvider url.Provider,
	git git.Interface,
	principalStore store.PrincipalStore,
	repoStore store.RepoStore,
	pullReqStore store.PullReqStore,
	pullReqActStore store.PullReqActivityStore,
	tx dbtx.Transactor,
) *PullReq {
	return NewPullReq(urlProvider, git, principalStore, repoStore, pullReqStore, pullReqActStore, tx)
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
