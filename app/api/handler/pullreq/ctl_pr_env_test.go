// Copyright 2026 Harness, Inc.
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
	"context"
	"testing"

	prctrl "github.com/harness/gitness/app/api/controller/pullreq"
	pullreqevents "github.com/harness/gitness/app/events/pullreq"
	"github.com/harness/gitness/app/services/instrument"
	"github.com/harness/gitness/app/services/label"
	"github.com/harness/gitness/app/services/protection"
	"github.com/harness/gitness/app/services/publickey"
	"github.com/harness/gitness/app/services/settings"
	"github.com/harness/gitness/app/sse"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/app/store/cache"
	"github.com/harness/gitness/app/store/database"
	"github.com/harness/gitness/app/url"
	"github.com/harness/gitness/audit"
	"github.com/harness/gitness/events"
	gitmock "github.com/harness/gitness/mocks/git"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/require"
)

type ctlSSE struct{}

func (ctlSSE) Publish(context.Context, int64, enum.SSEType, any) {}

func (ctlSSE) Stream(context.Context, int64) (<-chan *sse.Event, <-chan error, func(context.Context) error) {
	return nil, nil, nil
}

type ctlURLProvider struct {
	url.Provider
}

func (ctlURLProvider) GetInternalAPIURL(context.Context) string { return "http://localhost:3000" }

func (ctlURLProvider) GenerateUIPRURL(_ context.Context, repoPath string, prID int64) string {
	return "https://ui.example.com/" + repoPath + "/pulls/" + string(rune('0'+prID))
}

type ctlEnv struct {
	*ctlBase
	ctrl  *prctrl.Controller
	git   *gitmock.Interface
	space *types.Space
	repo  *types.Repository
	prs   *database.PullReqStore
}

func ctlNewEnv(t *testing.T, mode ctlAuthMode) *ctlEnv {
	t.Helper()
	b := ctlNewBase(t)

	pCache := cache.ProvidePrincipalInfoCache(database.NewPrincipalInfoView(b.db))
	eventsSystem, err := events.ProvideSystem(
		events.Config{Mode: events.ModeInMemory, Namespace: "ctl", MaxStreamLength: 100},
		nil, events.NewNoopCollector())
	require.NoError(t, err)
	reporter, err := pullreqevents.NewReporter(eventsSystem)
	require.NoError(t, err)

	ruleStore := database.NewRuleStore(b.db, pCache)
	protectionManager, err := protection.ProvideManager(ruleStore)
	require.NoError(t, err)

	userGroupStore := database.NewUserGroupStore(b.db)
	prStore := database.NewPullReqStore(b.db, pCache)
	gitMock := &gitmock.Interface{}

	var userGroupReviewerStore store.UserGroupReviewerStore = database.NewUsergroupReviewerStore(
		b.db, pCache, userGroupStore)

	settingsSvc := settings.NewService(database.NewSettingsStore(b.db), b.spaceFinder)
	labelSvc := label.New(
		b.tx, b.spaces, database.NewLabelStore(b.db), database.NewLabelValueStore(b.db),
		database.NewPullReqLabelStore(b.db), database.NewPullReqLabelSuggestionStore(b.db), pCache, b.spaceFinder,
	)

	ctrl := prctrl.NewController(
		b.tx, ctlURLProvider{}, ctlAuthorizer{mode: mode}, audit.New(), prStore,
		database.NewPullReqActivityStore(b.db, pCache), database.NewCodeCommentView(b.db),
		database.NewPullReqReviewStore(b.db), database.NewPullReqReviewerStore(b.db, pCache),
		database.NewPullReqReviewerSuggestionStore(b.db), database.NewPullReqWatcherStore(b.db, pCache),
		b.repos, nil, b.principals, userGroupStore, userGroupReviewerStore, pCache,
		database.NewPullReqFileViewStore(b.db), database.NewPullReqFileGroupStore(b.db), b.memberships,
		database.NewCheckStore(b.db, pCache), database.NewAutoMergeStore(b.db),
		database.NewMergeQueueStore(b.db), database.NewMergeQueueEntryStore(b.db), nil, gitMock, b.repoFinder,
		reporter, nil, nil, nil, nil, nil, protectionManager, ctlSSE{}, nil, nil, nil, settingsSvc, nil, labelSvc,
		database.NewLabelStore(b.db), database.NewLabelValueStore(b.db),
		database.NewPullReqLabelSuggestionStore(b.db), instrument.Noop{}, nil, database.NewBranchStore(b.db), nil,
		publickey.SignatureVerifyService{},
	)

	e := &ctlEnv{ctlBase: b, ctrl: ctrl, git: gitMock, prs: prStore}
	e.space = b.createSpace(t, "root", 0)
	e.repo = b.createRepo(t, e.space, "repo1")
	return e
}
