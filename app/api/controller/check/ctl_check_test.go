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

package check

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/app/auth"
	checkevents "github.com/harness/gitness/app/events/check"
	"github.com/harness/gitness/app/sse"
	"github.com/harness/gitness/app/store/cache"
	"github.com/harness/gitness/app/store/database"
	"github.com/harness/gitness/events"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/require"
)

const ctlSHA = "0123456789abcdef0123456789abcdef01234567"

var errCtlGit = errors.New("git failure")

type ctlSSE struct{ published int }

func (s *ctlSSE) Publish(context.Context, int64, enum.SSEType, any) { s.published++ }

func (*ctlSSE) Stream(context.Context, int64) (<-chan *sse.Event, <-chan error, func(context.Context) error) {
	return nil, nil, nil
}

// ctlGit only implements GetCommit; every other method would panic on the nil embedded interface.
type ctlGit struct {
	git.Interface
	err error
}

func (g ctlGit) GetCommit(context.Context, *git.GetCommitParams) (*git.GetCommitOutput, error) {
	if g.err != nil {
		return nil, g.err
	}
	return &git.GetCommitOutput{}, nil
}

type ctlResolver struct {
	id  int64
	err error
}

func (r ctlResolver) Resolve(context.Context, string, enum.PrincipalType) (int64, error) {
	return r.id, r.err
}

type ctlEnv struct {
	*ctlBase
	ctrl  *Controller
	sse   *ctlSSE
	repo  *types.Repository
	space *types.Space
}

func ctlNewEnv(t *testing.T) *ctlEnv {
	t.Helper()
	b := ctlNewBase(t)

	system, err := events.ProvideSystem(
		events.Config{Mode: events.ModeInMemory, Namespace: "ctl", MaxStreamLength: 100},
		nil, events.NewNoopCollector())
	require.NoError(t, err)
	reporter, err := checkevents.NewReporter(system)
	require.NoError(t, err)

	pCache := cache.ProvidePrincipalInfoCache(database.NewPrincipalInfoView(b.db))
	checkStore := database.NewCheckStore(b.db, pCache)
	streamer := &ctlSSE{}

	space := b.createSpace(t, "root", 0)
	repo := b.createRepo(t, space, "repo1")

	return &ctlEnv{
		ctlBase: b,
		ctrl: NewController(
			b.tx, ctlAuthorizer{mode: ctlAllow}, b.spaces, checkStore, b.principals, b.spaceFinder, b.repoFinder,
			ctlGit{}, ProvideCheckSanitizers(), streamer, reporter, ctlResolver{},
		),
		sse:   streamer,
		repo:  repo,
		space: space,
	}
}

func ctlReport(identifier string, status enum.CheckStatus) *ReportInput {
	return &ReportInput{
		Identifier: identifier,
		Status:     status,
		Summary:    "summary",
		Link:       "https://example.com/" + identifier,
		Payload:    types.CheckPayload{Kind: enum.CheckPayloadKindEmpty},
	}
}

func TestCtlCheckController_ReportAndList(t *testing.T) {
	ctx := context.Background()
	e := ctlNewEnv(t)
	s := e.session()

	first, err := e.ctrl.Report(
		ctx, s, "root/repo1", ctlSHA, ctlReport("build", enum.CheckStatusRunning), map[string]string{"k": "v"})
	require.NoError(t, err)
	require.Equal(t, "build", first.Identifier)
	require.NotZero(t, first.Started)
	require.Zero(t, first.Ended)
	require.Equal(t, 1, e.sse.published)

	// Updating a check to a terminal state keeps the start time and sets the end time.
	done, err := e.ctrl.Report(ctx, s, "root/repo1", ctlSHA, ctlReport("build", enum.CheckStatusSuccess), nil)
	require.NoError(t, err)
	require.Equal(t, first.Started, done.Started)
	require.NotZero(t, done.Ended)

	_, err = e.ctrl.Report(ctx, s, "root/repo1", ctlSHA, ctlReport("lint", enum.CheckStatusPending), nil)
	require.NoError(t, err)

	checks, count, err := e.ctrl.ListChecks(ctx, s, "root/repo1", ctlSHA, types.CheckListOptions{
		ListQueryFilter: types.ListQueryFilter{Pagination: types.Pagination{Page: 1, Size: 10}},
	})
	require.NoError(t, err)
	require.Equal(t, 2, count)
	require.Len(t, checks, 2)

	// A full page triggers the count query.
	checks, count, err = e.ctrl.ListChecks(ctx, s, "root/repo1", ctlSHA, types.CheckListOptions{
		ListQueryFilter: types.ListQueryFilter{Pagination: types.Pagination{Page: 1, Size: 1}},
	})
	require.NoError(t, err)
	require.Equal(t, 2, count)
	require.Len(t, checks, 1)

	recent, err := e.ctrl.ListRecentChecks(ctx, s, "root/repo1", types.CheckRecentOptions{})
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"build", "lint"}, recent)

	recent, err = e.ctrl.ListRecentChecksSpace(ctx, s, "root", false, types.CheckRecentOptions{})
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"build", "lint"}, recent)

	recent, err = e.ctrl.ListRecentChecksSpace(ctx, s, "root", true, types.CheckRecentOptions{Since: 1})
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"build", "lint"}, recent)

	summaries, err := e.ctrl.ListCheckSummaries(ctx, s, "root/repo1", &SummaryInput{CommitSHAs: []string{ctlSHA}})
	require.NoError(t, err)
	require.Len(t, summaries, 1)
	require.Equal(t, ctlSHA, summaries[0].CommitSHA.String())

	// Unknown commits are simply absent from the summaries.
	summaries, err = e.ctrl.ListCheckSummaries(ctx, s, "root/repo1", &SummaryInput{
		CommitSHAs: []string{strings.Repeat("a", 40)},
	})
	require.NoError(t, err)
	require.Empty(t, summaries)
}

func TestCtlCheckController_ReportErrors(t *testing.T) {
	ctx := context.Background()
	e := ctlNewEnv(t)
	s := e.session()

	_, err := e.ctrl.Report(ctx, s, "", ctlSHA, ctlReport("a", enum.CheckStatusRunning), nil)
	require.Error(t, err)
	_, err = e.ctrl.Report(ctx, s, "root/unknown", ctlSHA, ctlReport("a", enum.CheckStatusRunning), nil)
	require.Error(t, err)

	_, err = e.ctrl.Report(ctx, s, "root/repo1", ctlSHA, ctlReport("", enum.CheckStatusRunning), nil)
	require.Error(t, err)

	_, err = e.ctrl.Report(ctx, s, "root/repo1", "short", ctlReport("a", enum.CheckStatusRunning), nil)
	require.Error(t, err)

	// Git lookup failure.
	c := *e.ctrl
	c.git = ctlGit{err: errCtlGit}
	_, err = c.Report(ctx, s, "root/repo1", ctlSHA, ctlReport("a", enum.CheckStatusRunning), nil)
	require.ErrorIs(t, err, errCtlGit)
}

func TestCtlCheckController_AccessFailures(t *testing.T) {
	ctx := context.Background()
	allow := ctlNewEnv(t)
	s := allow.session()

	for _, mode := range []ctlAuthMode{ctlDeny, ctlError} {
		c := *allow.ctrl
		c.authorizer = ctlAuthorizer{mode: mode}

		errs := map[string]error{}
		_, errs["report"] = c.Report(ctx, s, "root/repo1", ctlSHA, ctlReport("a", enum.CheckStatusRunning), nil)
		_, _, errs["list"] = c.ListChecks(ctx, s, "root/repo1", ctlSHA, types.CheckListOptions{})
		_, errs["recent"] = c.ListRecentChecks(ctx, s, "root/repo1", types.CheckRecentOptions{})
		_, errs["recent space"] = c.ListRecentChecksSpace(ctx, s, "root", false, types.CheckRecentOptions{})
		_, errs["summary"] = c.ListCheckSummaries(ctx, s, "root/repo1", &SummaryInput{})
		for name, err := range errs {
			require.Error(t, err, name)
			if mode == ctlError {
				require.ErrorIs(t, err, errCtlAuthz, name)
			}
		}
	}
}

func TestCtlCheckController_Bypass(t *testing.T) {
	ctx := context.Background()
	e := ctlNewEnv(t)
	service := e.createUser(t, "svc", false)
	target := e.createUser(t, "target", false)
	svcSession := &auth.Session{Principal: *service.ToPrincipal()}
	svcSession.Principal.Type = enum.PrincipalTypeService

	// Existing principal id.
	id := target.ID
	in := ctlReport("bypassed", enum.CheckStatusSuccess)
	in.BypassedBy = &id
	res, err := e.ctrl.Report(ctx, svcSession, "root/repo1", ctlSHA, in, nil)
	require.NoError(t, err)
	require.NotNil(t, res.BypassedBy)
	require.Equal(t, target.UID, res.BypassedBy.UID)

	// Non-service principals cannot bypass.
	in = ctlReport("bypassed2", enum.CheckStatusSuccess)
	in.BypassedBy = &id
	_, err = e.ctrl.Report(ctx, e.session(), "root/repo1", ctlSHA, in, nil)
	var userErr *usererror.Error
	require.ErrorAs(t, err, &userErr)
	require.Equal(t, 403, userErr.Status)

	// Resolved by UID.
	c := *e.ctrl
	c.principalResolver = ctlResolver{id: target.ID}
	in = ctlReport("bypassed3", enum.CheckStatusSuccess)
	in.BypassedByUID, in.BypassedByType = "someone", enum.PrincipalTypeUser
	res, err = c.Report(ctx, svcSession, "root/repo1", ctlSHA, in, nil)
	require.NoError(t, err)
	require.Equal(t, target.UID, res.BypassedBy.UID)

	// Resolver failures / invalid values.
	for name, tc := range map[string]struct {
		resolver ctlResolver
		input    func() *ReportInput
	}{
		"resolver error": {ctlResolver{err: errCtlGit}, func() *ReportInput {
			rep := ctlReport("b4", enum.CheckStatusSuccess)
			rep.BypassedByUID, rep.BypassedByType = "x", enum.PrincipalTypeUser
			return rep
		}},
		"resolver zero": {ctlResolver{id: 0}, func() *ReportInput {
			rep := ctlReport("b5", enum.CheckStatusSuccess)
			rep.BypassedByUID, rep.BypassedByType = "x", enum.PrincipalTypeUser
			return rep
		}},
		"resolved principal missing": {ctlResolver{id: 99999}, func() *ReportInput {
			rep := ctlReport("b6", enum.CheckStatusSuccess)
			rep.BypassedByUID, rep.BypassedByType = "x", enum.PrincipalTypeUser
			return rep
		}},
		"missing type": {ctlResolver{id: target.ID}, func() *ReportInput {
			rep := ctlReport("b7", enum.CheckStatusSuccess)
			rep.BypassedByUID = "x"
			return rep
		}},
		"unknown id without uid": {ctlResolver{}, func() *ReportInput {
			unknown := int64(99999)
			rep := ctlReport("b8", enum.CheckStatusSuccess)
			rep.BypassedBy = &unknown
			return rep
		}},
		"negative id": {ctlResolver{}, func() *ReportInput {
			negative := int64(-1)
			rep := ctlReport("b9", enum.CheckStatusSuccess)
			rep.BypassedBy = &negative
			return rep
		}},
	} {
		cc := *e.ctrl
		cc.principalResolver = tc.resolver
		_, err = cc.Report(ctx, svcSession, "root/repo1", ctlSHA, tc.input(), nil)
		require.Error(t, err, name)
	}
}

func TestCtlReportInput_Sanitize(t *testing.T) {
	sanitizers := ProvideCheckSanitizers()
	s := &auth.Session{Principal: types.Principal{Type: enum.PrincipalTypeUser}}
	svc := &auth.Session{Principal: types.Principal{Type: enum.PrincipalTypeService}}

	valid := func() *ReportInput { return ctlReport("ok", enum.CheckStatusSuccess) }
	negative := int64(-5)

	tests := []struct {
		name    string
		session *auth.Session
		mutate  func(in *ReportInput)
		wantErr bool
	}{
		{"valid", s, func(*ReportInput) {}, false},
		{"legacy uid", s, func(in *ReportInput) { in.Identifier, in.CheckUID = "", "legacy" }, false},
		{"missing identifier", s, func(in *ReportInput) { in.Identifier = "" }, true},
		{"bad identifier", s, func(in *ReportInput) { in.Identifier = "no spaces" }, true},
		{"long summary", s, func(in *ReportInput) { in.Summary = strings.Repeat("x", maxSummaryLength+1) }, true},
		{"long link", s, func(in *ReportInput) { in.Link = strings.Repeat("x", maxLinkLength+1) }, true},
		{"bad status", s, func(in *ReportInput) { in.Status = "nope" }, true},
		{"bad kind", s, func(in *ReportInput) { in.Payload.Kind = "nope" }, true},
		{"empty kind without link", s, func(in *ReportInput) { in.Link = "" }, true},
		{"pipeline kind", s, func(in *ReportInput) { in.Payload.Kind = enum.CheckPayloadKindPipeline }, true},
		{"raw kind", s, func(in *ReportInput) {
			in.Payload = types.CheckPayload{Kind: enum.CheckPayloadKindRaw, Data: json.RawMessage(`{"details":"d"}`)}
		}, false},
		{"markdown kind empty data", s, func(in *ReportInput) {
			in.Payload = types.CheckPayload{Kind: enum.CheckPayloadKindMarkdown}
		}, false},
		{"raw kind with version", s, func(in *ReportInput) {
			in.Payload = types.CheckPayload{Kind: enum.CheckPayloadKindRaw, Version: "v1"}
		}, true},
		{"raw kind unknown field", s, func(in *ReportInput) {
			in.Payload = types.CheckPayload{Kind: enum.CheckPayloadKindRaw, Data: json.RawMessage(`{"x":1}`)}
		}, true},
		{"end before start", s, func(in *ReportInput) { in.Started, in.Ended = 10, 5 }, true},
		{"bypass by user", s, func(in *ReportInput) { in.BypassedByUID = "x" }, true},
		{"bypass by service", svc, func(in *ReportInput) { in.BypassedByUID = "x" }, false},
		{"bypass negative id", svc, func(in *ReportInput) { in.BypassedBy = &negative }, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := valid()
			tt.mutate(in)
			err := in.Sanitize(sanitizers, tt.session)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestCtlGetEndTime(t *testing.T) {
	require.EqualValues(t, 42, getEndTime(&ReportInput{Ended: 42}, 1))
	require.EqualValues(t, 7, getEndTime(&ReportInput{Status: enum.CheckStatusSuccess}, 7))
	require.EqualValues(t, 0, getEndTime(&ReportInput{Status: enum.CheckStatusRunning}, 7))
}

func TestCtlSanitizeJSONPayload(t *testing.T) {
	out, err := SanitizeJSONPayload(nil, &types.CheckPayloadText{})
	require.NoError(t, err)
	require.JSONEq(t, `{"details":""}`, string(out))

	out, err = SanitizeJSONPayload(json.RawMessage(`{"details":"a<b"}`), &types.CheckPayloadText{})
	require.NoError(t, err)
	require.Equal(t, `{"details":"a<b"}`, string(out))

	_, err = SanitizeJSONPayload(json.RawMessage(`{"details":`), &types.CheckPayloadText{})
	require.Error(t, err)
}

func TestCtlSummaryInput_Sanitize(t *testing.T) {
	in := &SummaryInput{CommitSHAs: []string{ctlSHA, ctlSHA, strings.Repeat("a", 40)}}
	require.NoError(t, in.Sanitize())
	require.Len(t, in.CommitSHAs, 2)

	require.Error(t, (&SummaryInput{CommitSHAs: []string{"bad"}}).Sanitize())
	require.Error(t, (&SummaryInput{ExcludePayloadKinds: make([]string, maxExcludePayloadKinds+1)}).Sanitize())
	require.Error(t, (&SummaryInput{CommitSHAs: make([]string, 101)}).Sanitize())
}
