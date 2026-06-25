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

package github

import (
	checkevents "github.com/harness/gitness/app/events/check"
	pullreqevents "github.com/harness/gitness/app/events/pullreq"
	"github.com/harness/gitness/app/services/importer"
	"github.com/harness/gitness/app/services/linkedpr"
	"github.com/harness/gitness/app/services/refcache"
	"github.com/harness/gitness/app/store"
	gitnessurl "github.com/harness/gitness/app/url"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/store/database/dbtx"

	"github.com/google/wire"
)

// WireSet exposes the GitHub-specific providers.
var WireSet = wire.NewSet(
	ProvidePullRequestHandler,
	ProvideCheckHandler,
)

func ProvideCheckHandler(
	checkStore store.CheckStore,
	eventReporter *checkevents.Reporter,
	authorResolver linkedpr.AuthorResolver,
) *CheckHandler {
	return NewCheckHandler(checkStore, eventReporter, authorResolver)
}

func ProvidePullRequestHandler(
	pullReqStore store.PullReqStore,
	linkedPullReqStore store.LinkedPullReqStore,
	activityStore store.PullReqActivityStore,
	authorResolver linkedpr.AuthorResolver,
	reporter *pullreqevents.Reporter,
	gitClient git.Interface,
	repoFinder refcache.RepoFinder,
	urlProvider gitnessurl.Provider,
	connectorService importer.ConnectorService,
	tx dbtx.Transactor,
) *PullRequestHandler {
	return NewPullRequestHandler(
		pullReqStore, linkedPullReqStore, activityStore, authorResolver,
		reporter, gitClient, repoFinder, urlProvider, connectorService, tx,
	)
}
