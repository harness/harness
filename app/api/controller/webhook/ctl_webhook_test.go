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

package webhook

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	checkevents "github.com/harness/gitness/app/events/check"
	gitevents "github.com/harness/gitness/app/events/git"
	mergequeueevents "github.com/harness/gitness/app/events/mergequeue"
	pullreqevents "github.com/harness/gitness/app/events/pullreq"
	"github.com/harness/gitness/app/services/webhook"
	"github.com/harness/gitness/app/sse"
	"github.com/harness/gitness/app/store/database"
	"github.com/harness/gitness/audit"
	"github.com/harness/gitness/encrypt"
	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/events"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/require"
)

type ctlSSE struct{}

func (ctlSSE) Publish(context.Context, int64, enum.SSEType, any) {}

func (ctlSSE) Stream(context.Context, int64) (<-chan *sse.Event, <-chan error, func(context.Context) error) {
	return nil, nil, nil
}

type ctlURLProvider struct{}

func (ctlURLProvider) GetWebhookURL(context.Context, *types.WebhookCore) (string, error) {
	return "http://localhost/internal", nil
}

type ctlEnv struct {
	*ctlBase
	ctrl       *Controller
	executions *database.WebhookExecutionStore
	space      *types.Space
	repo       *types.Repository
}

func ctlNewEnv(t *testing.T) *ctlEnv {
	t.Helper()
	b := ctlNewBase(t)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	system, err := events.ProvideSystem(
		events.Config{Mode: events.ModeInMemory, Namespace: "ctl", MaxStreamLength: 100},
		nil, events.NewNoopCollector())
	require.NoError(t, err)
	gitFactory, err := gitevents.ProvideReaderFactory(system)
	require.NoError(t, err)
	prFactory, err := pullreqevents.ProvideReaderFactory(system)
	require.NoError(t, err)
	mqFactory, err := mergequeueevents.ProvideReaderFactory(system)
	require.NoError(t, err)
	checkFactory, err := checkevents.ProvideReaderFactory(system)
	require.NoError(t, err)

	encrypter, err := encrypt.New(strings.Repeat("k", 32), false)
	require.NoError(t, err)

	webhookStore := database.NewWebhookStore(b.db)
	executionStore := database.NewWebhookExecutionStore(b.db)

	svc, err := webhook.NewService(
		ctx,
		webhook.Config{
			UserAgentIdentity: "ctl-test", EventReaderName: "ctl-reader", Concurrency: 1, AllowLoopback: true,
		},
		b.tx, gitFactory, prFactory, mqFactory, checkFactory,
		webhookStore, executionStore, b.spaces, b.repos,
		nil, nil, nil, b.principals, nil, encrypter, nil, ctlURLProvider{}, nil,
		audit.New(), ctlSSE{}, nil, b.spacePaths, nil,
	)
	require.NoError(t, err)

	space := b.createSpace(t, "root", 0)
	repo := b.createRepo(t, space, "repo1")

	return &ctlEnv{
		ctlBase: b,
		ctrl: NewController(
			ctlAuthorizer{mode: ctlAllow}, b.spaceFinder, b.repoFinder, svc, encrypter, NoopPreprocessor{}),
		executions: executionStore,
		space:      space,
		repo:       repo,
	}
}

func ctlHookInput(identifier, url string) *types.WebhookCreateInput {
	return &types.WebhookCreateInput{
		Identifier: identifier,
		URL:        url,
		Enabled:    true,
		Triggers:   []enum.WebhookTrigger{enum.WebhookTriggerBranchCreated},
	}
}

func TestCtlWebhookController_RepoLifecycle(t *testing.T) {
	ctx := context.Background()
	e := ctlNewEnv(t)
	s := e.session()
	repoRef := "root/repo1"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	hook, err := e.ctrl.CreateRepo(ctx, s, repoRef, ctlHookInput("hook1", srv.URL), nil)
	require.NoError(t, err)
	require.Equal(t, "hook1", hook.Identifier)
	require.Equal(t, enum.WebhookParentRepo, hook.ParentType)

	// Duplicate identifiers are rejected.
	_, err = e.ctrl.CreateRepo(ctx, s, repoRef, ctlHookInput("hook1", srv.URL), nil)
	require.Error(t, err)

	// Invalid input.
	_, err = e.ctrl.CreateRepo(ctx, s, repoRef, ctlHookInput("hook2", "not a url"), nil)
	require.Error(t, err)

	found, err := e.ctrl.FindRepo(ctx, s, repoRef, "hook1")
	require.NoError(t, err)
	require.Equal(t, hook.ID, found.ID)

	_, err = e.ctrl.FindRepo(ctx, s, repoRef, "missing")
	require.Error(t, err)

	list, count, err := e.ctrl.ListRepo(ctx, s, repoRef, false, &types.WebhookFilter{Page: 1, Size: 10})
	require.NoError(t, err)
	require.EqualValues(t, 1, count)
	require.Len(t, list, 1)

	newName := "Renamed"
	updated, err := e.ctrl.UpdateRepo(ctx, s, repoRef, "hook1", &types.WebhookUpdateInput{DisplayName: &newName}, nil)
	require.NoError(t, err)
	require.Equal(t, newName, updated.DisplayName)

	// Executions: none yet, then add one and retrigger it.
	execs, total, err := e.ctrl.ListExecutionsRepo(
		ctx, s, repoRef, "hook1", &types.WebhookExecutionFilter{Page: 1, Size: 10})
	require.NoError(t, err)
	require.EqualValues(t, 0, total)
	require.Empty(t, execs)

	exec := &types.WebhookExecution{
		WebhookID:     hook.ID,
		TriggerType:   enum.WebhookTriggerBranchCreated,
		TriggerID:     "trigger-1",
		Result:        enum.WebhookExecutionResultSuccess,
		Retriggerable: true,
		Request:       types.WebhookExecutionRequest{URL: srv.URL, Body: `{"a":1}`},
	}
	require.NoError(t, e.executions.Create(ctx, exec))

	gotExec, err := e.ctrl.FindExecutionRepo(ctx, s, repoRef, "hook1", exec.ID)
	require.NoError(t, err)
	require.Equal(t, exec.ID, gotExec.ID)

	_, err = e.ctrl.FindExecutionRepo(ctx, s, repoRef, "hook1", exec.ID+100)
	require.Error(t, err)

	retriggered, err := e.ctrl.RetriggerExecutionRepo(ctx, s, repoRef, "hook1", exec.ID)
	require.NoError(t, err)
	require.NotNil(t, retriggered.RetriggerOf)
	require.Equal(t, exec.ID, *retriggered.RetriggerOf)

	_, err = e.ctrl.RetriggerExecutionRepo(ctx, s, repoRef, "hook1", exec.ID+100)
	require.Error(t, err)

	require.NoError(t, e.ctrl.DeleteRepo(ctx, s, repoRef, "hook1"))
	require.Error(t, e.ctrl.DeleteRepo(ctx, s, repoRef, "hook1"))
}

func TestCtlWebhookController_SpaceLifecycle(t *testing.T) {
	ctx := context.Background()
	e := ctlNewEnv(t)
	s := e.session()
	spaceRef := "root"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	hook, err := e.ctrl.CreateSpace(ctx, s, spaceRef, ctlHookInput("shook", srv.URL), nil)
	require.NoError(t, err)
	require.Equal(t, enum.WebhookParentSpace, hook.ParentType)

	_, err = e.ctrl.CreateSpace(ctx, s, spaceRef, ctlHookInput("bad", "not a url"), nil)
	require.Error(t, err)

	found, err := e.ctrl.FindSpace(ctx, s, spaceRef, "shook")
	require.NoError(t, err)
	require.Equal(t, hook.ID, found.ID)

	list, count, err := e.ctrl.ListSpace(ctx, s, spaceRef, false, &types.WebhookFilter{Page: 1, Size: 10})
	require.NoError(t, err)
	require.EqualValues(t, 1, count)
	require.Len(t, list, 1)

	desc := "updated"
	updated, err := e.ctrl.UpdateSpace(ctx, s, spaceRef, "shook", &types.WebhookUpdateInput{Description: &desc}, nil)
	require.NoError(t, err)
	require.Equal(t, desc, updated.Description)

	exec := &types.WebhookExecution{
		WebhookID:     hook.ID,
		TriggerType:   enum.WebhookTriggerBranchCreated,
		TriggerID:     "trigger-2",
		Result:        enum.WebhookExecutionResultSuccess,
		Retriggerable: true,
		Request:       types.WebhookExecutionRequest{URL: srv.URL, Body: `{}`},
	}
	require.NoError(t, e.executions.Create(ctx, exec))

	execs, total, err := e.ctrl.ListExecutionsSpace(
		ctx, s, spaceRef, "shook", &types.WebhookExecutionFilter{Page: 1, Size: 10})
	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	require.Len(t, execs, 1)

	gotExec, err := e.ctrl.FindExecutionSpace(ctx, s, spaceRef, "shook", exec.ID)
	require.NoError(t, err)
	require.Equal(t, exec.ID, gotExec.ID)

	retriggered, err := e.ctrl.RetriggerExecutionSpace(ctx, s, spaceRef, "shook", exec.ID)
	require.NoError(t, err)
	require.NotNil(t, retriggered.RetriggerOf)

	require.NoError(t, e.ctrl.DeleteSpace(ctx, s, spaceRef, "shook"))
	require.Error(t, e.ctrl.DeleteSpace(ctx, s, spaceRef, "shook"))
}

func TestCtlWebhookController_AccessFailures(t *testing.T) {
	ctx := context.Background()
	allow := ctlNewEnv(t)
	s := allow.session()
	in := ctlHookInput("hook", "http://localhost:1")

	for _, mode := range []ctlAuthMode{ctlDeny, ctlError} {
		c := *allow.ctrl
		c.authorizer = ctlAuthorizer{mode: mode}

		errs := map[string]error{}
		_, errs["create repo"] = c.CreateRepo(ctx, s, "root/repo1", in, nil)
		_, errs["update repo"] = c.UpdateRepo(ctx, s, "root/repo1", "h", &types.WebhookUpdateInput{}, nil)
		errs["delete repo"] = c.DeleteRepo(ctx, s, "root/repo1", "h")
		_, errs["find repo"] = c.FindRepo(ctx, s, "root/repo1", "h")
		_, _, errs["list repo"] = c.ListRepo(ctx, s, "root/repo1", false, &types.WebhookFilter{})
		_, errs["find exec repo"] = c.FindExecutionRepo(ctx, s, "root/repo1", "h", 1)
		_, _, errs["list exec repo"] = c.ListExecutionsRepo(ctx, s, "root/repo1", "h", &types.WebhookExecutionFilter{})
		_, errs["retrigger repo"] = c.RetriggerExecutionRepo(ctx, s, "root/repo1", "h", 1)
		_, errs["create space"] = c.CreateSpace(ctx, s, "root", in, nil)
		_, errs["update space"] = c.UpdateSpace(ctx, s, "root", "h", &types.WebhookUpdateInput{}, nil)
		errs["delete space"] = c.DeleteSpace(ctx, s, "root", "h")
		_, errs["find space"] = c.FindSpace(ctx, s, "root", "h")
		_, _, errs["list space"] = c.ListSpace(ctx, s, "root", false, &types.WebhookFilter{})
		_, errs["find exec space"] = c.FindExecutionSpace(ctx, s, "root", "h", 1)
		_, _, errs["list exec space"] = c.ListExecutionsSpace(ctx, s, "root", "h", &types.WebhookExecutionFilter{})
		_, errs["retrigger space"] = c.RetriggerExecutionSpace(ctx, s, "root", "h", 1)

		for name, err := range errs {
			require.Error(t, err, name)
			if mode == ctlError {
				require.ErrorIs(t, err, errCtlAuthz, name)
			}
		}
	}
}

func TestCtlWebhookController_ReferenceErrors(t *testing.T) {
	ctx := context.Background()
	e := ctlNewEnv(t)
	s := e.session()
	in := ctlHookInput("hook", "http://localhost:1")

	// Empty repo reference is an invalid argument.
	_, err := e.ctrl.CreateRepo(ctx, s, "", in, nil)
	require.True(t, errors.IsInvalidArgument(err))

	// Unknown repo / space.
	_, err = e.ctrl.FindRepo(ctx, s, "root/unknown", "h")
	require.Error(t, err)
	_, err = e.ctrl.FindSpace(ctx, s, "unknown", "h")
	require.Error(t, err)

	// Repo in a state that does not allow edits.
	other := e.createRepo(t, e.space, "migrating")
	other.State = enum.RepoStateGitImport
	require.NoError(t, e.repos.Update(ctx, other))
	_, err = e.ctrl.CreateRepo(ctx, s, "root/migrating", in, nil)
	require.Error(t, err)
}

func TestCtlNoopPreprocessor(t *testing.T) {
	p := NoopPreprocessor{}

	typ, err := p.PreprocessCreateInput(enum.PrincipalTypeUser, &types.WebhookCreateInput{}, nil)
	require.NoError(t, err)
	require.Equal(t, enum.WebhookTypeExternal, typ)

	typ, err = p.PreprocessUpdateInput(enum.PrincipalTypeUser, &types.WebhookUpdateInput{}, nil)
	require.NoError(t, err)
	require.Equal(t, enum.WebhookTypeExternal, typ)

	filter := &types.WebhookFilter{}
	p.PreprocessFilter(enum.PrincipalTypeUser, filter)
	require.Equal(t, enum.OrderAsc, filter.Order)
	require.True(t, filter.SkipInternal)

	filter = &types.WebhookFilter{Order: enum.OrderDesc}
	p.PreprocessFilter(enum.PrincipalTypeUser, filter)
	require.Equal(t, enum.OrderDesc, filter.Order)

	require.False(t, p.IsInternalCall(enum.PrincipalTypeService))
}
