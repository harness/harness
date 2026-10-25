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

package verify

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/harness/gitness/app/services/insights/insightstest"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

const hour = int64(time.Hour / time.Millisecond)

var base = time.Unix(1_800_000_000, 0)

type call struct {
	taskID int64
	to     enum.TaskState
	reason string
}

type fakeTransitions struct{ calls []call }

func (f *fakeTransitions) Transition(
	_ context.Context, _, taskID int64, to enum.TaskState, reason string,
) (*types.Task, error) {
	f.calls = append(f.calls, call{taskID, to, reason})
	return &types.Task{ID: taskID, State: to}, nil
}

type fakeLinked struct{ rows []*types.TaskLinkedRef }

func (f *fakeLinked) ListByStateAndLinkKind(
	_ context.Context, _ enum.TaskState, _ enum.TaskLinkKind, after int64, limit int,
) ([]*types.TaskLinkedRef, error) {
	var out []*types.TaskLinkedRef
	for _, r := range f.rows {
		if r.TaskID > after && len(out) < limit {
			out = append(out, r)
		}
	}
	return out, nil
}

type fakeReleases struct{ rels []*types.InsightsReleaseInfo }

func (f *fakeReleases) ListReleases(context.Context, int64) ([]*types.InsightsReleaseInfo, error) {
	return f.rels, nil
}

type fixture struct {
	st    *insightstest.Store
	tr    *fakeTransitions
	link  *fakeLinked
	rels  *fakeReleases
	v     *Verifier
	issue *types.InsightsIssue
}

// newFixture has a resolved issue (fixed in 1.1.0, last seen 100h ago) with a task resolved 80h ago.
func newFixture(t *testing.T) *fixture {
	t.Helper()
	f := &fixture{st: insightstest.New(), tr: &fakeTransitions{}, link: &fakeLinked{}, rels: &fakeReleases{}}
	now := base.UnixMilli()
	f.issue = &types.InsightsIssue{
		ProjectID: 1, RepoID: 1, Fingerprint: "fp", FirstSeen: now - 200*hour, LastSeen: now - 100*hour,
		FirstRelease: "1.0.0", LastRelease: "1.0.0", Status: enum.InsightsIssueStatusResolved,
		ResolvedInRelease: "1.1.0",
	}
	if err := f.st.IssueStore().Create(context.Background(), f.issue); err != nil {
		t.Fatal(err)
	}
	f.link.rows = []*types.TaskLinkedRef{{TaskID: 10, RefID: IssueRef(f.issue.ID), Resolved: now - 80*hour}}
	f.v = NewVerifier(Config{}, f.link, f.st.IssueStore(), f.rels, f.tr, func() int64 { return 7 })
	f.v.now = func() time.Time { return base }
	return f
}

func TestVerifier_VerifiedByNewRelease(t *testing.T) {
	f := newFixture(t)
	// fix release live for 30h > 24h window; the issue was last seen before it.
	f.rels.rels = []*types.InsightsReleaseInfo{
		{Release: "1.0.0", FirstSeen: base.UnixMilli() - 300*hour},
		{Release: "1.2.0", FirstSeen: base.UnixMilli() - 30*hour}, // newer than the fix counts too
	}
	// make the quiet rule not apply so only the release rule can fire.
	f.link.rows[0].Resolved = base.UnixMilli() - 10*hour
	f.issue.LastSeen = base.UnixMilli() - 40*hour
	updateIssue(t, f)

	res, err := f.v.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if res.Verified != 1 || len(f.tr.calls) != 1 {
		t.Fatalf("result = %+v, calls = %+v", res, f.tr.calls)
	}
	c := f.tr.calls[0]
	if c.taskID != 10 || c.to != enum.TaskStateVerified || !strings.Contains(c.reason, "1.2.0") {
		t.Errorf("call = %+v", c)
	}
}

func TestVerifier_NewReleaseTooRecent(t *testing.T) {
	f := newFixture(t)
	f.rels.rels = []*types.InsightsReleaseInfo{{Release: "1.1.0", FirstSeen: base.UnixMilli() - 2*hour}}
	f.link.rows[0].Resolved = base.UnixMilli() - 10*hour
	f.issue.LastSeen = base.UnixMilli() - 40*hour
	updateIssue(t, f)

	if res, _ := f.v.Run(context.Background()); res.Verified != 0 || len(f.tr.calls) != 0 {
		t.Errorf("verified too early: %+v", f.tr.calls)
	}
}

func TestVerifier_NotVerifiedWhenEventsContinue(t *testing.T) {
	f := newFixture(t)
	// release 1.1.0 live for 30h but the issue was seen 5h ago (still happening in it).
	f.rels.rels = []*types.InsightsReleaseInfo{{Release: "1.1.0", FirstSeen: base.UnixMilli() - 30*hour}}
	f.link.rows[0].Resolved = base.UnixMilli() - 60*hour
	f.issue.LastSeen = base.UnixMilli() - 5*hour
	updateIssue(t, f)

	if res, _ := f.v.Run(context.Background()); res.Checked != 1 || res.Verified != 0 {
		t.Errorf("result = %+v, calls = %+v", res, f.tr.calls)
	}
}

func TestVerifier_VerifiedByQuietPeriod(t *testing.T) {
	f := newFixture(t) // last seen 100h ago, resolved 80h ago, no release data at all
	f.issue.ResolvedInRelease = ""
	updateIssue(t, f)

	res, err := f.v.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if res.Verified != 1 || f.tr.calls[0].to != enum.TaskStateVerified ||
		!strings.Contains(f.tr.calls[0].reason, "not seen") {
		t.Errorf("result = %+v, calls = %+v", res, f.tr.calls)
	}
}

func TestVerifier_QuietPeriodCountsFromResolution(t *testing.T) {
	f := newFixture(t)
	f.issue.ResolvedInRelease = ""
	updateIssue(t, f)
	f.link.rows[0].Resolved = base.UnixMilli() - 10*hour // resolved recently: quiet only 10h

	if res, _ := f.v.Run(context.Background()); res.Verified != 0 {
		t.Errorf("verified before quiet period: %+v", f.tr.calls)
	}
}

func TestVerifier_SkipsNonResolvedIssue(t *testing.T) {
	f := newFixture(t)
	f.issue.Status = enum.InsightsIssueStatusRegressed
	updateIssue(t, f)

	if res, _ := f.v.Run(context.Background()); res.Verified != 0 {
		t.Errorf("verified a regressed issue: %+v", f.tr.calls)
	}
}

func updateIssue(t *testing.T, f *fixture) {
	t.Helper()
	cur, err := f.st.IssueStore().Find(context.Background(), f.issue.ID)
	if err != nil {
		t.Fatal(err)
	}
	f.issue.Version = cur.Version
	if err = f.st.IssueStore().Update(context.Background(), f.issue); err != nil {
		t.Fatal(err)
	}
}

type fakeFinder struct{ tasks []*types.Task }

func (f *fakeFinder) FindTasksByRef(context.Context, enum.TaskLinkKind, string) ([]*types.Task, error) {
	return f.tasks, nil
}

func TestReopenObserver_ReopensResolvedAndVerified(t *testing.T) {
	tr := &fakeTransitions{}
	finder := &fakeFinder{tasks: []*types.Task{
		{ID: 1, State: enum.TaskStateResolved},
		{ID: 2, State: enum.TaskStateVerified},
		{ID: 3, State: enum.TaskStateInProgress},
		{ID: 4, State: enum.TaskStateDismissed},
	}}
	o := NewReopenObserver(finder, nil, tr, func() int64 { return 7 })

	issue := &types.InsightsIssue{ID: 5, LastRelease: "1.0.0"}
	err := o.OnIssueRegressed(context.Background(), nil, issue, &types.InsightsEvent{Release: "2.0.0"})
	if err != nil {
		t.Fatal(err)
	}
	if len(tr.calls) != 2 || tr.calls[0].taskID != 1 || tr.calls[1].taskID != 2 {
		t.Fatalf("calls = %+v", tr.calls)
	}
	for _, c := range tr.calls {
		if c.to != enum.TaskStateReopened || c.reason != "error reappeared in release 2.0.0" {
			t.Errorf("call = %+v", c)
		}
	}
}

func TestReopenObserver_UsesIssueTaskID(t *testing.T) {
	tr := &fakeTransitions{}
	id := int64(9)
	o := NewReopenObserver(&fakeFinder{}, func(_ context.Context, id int64) (*types.Task, error) {
		return &types.Task{ID: id, State: enum.TaskStateResolved}, nil
	}, tr, func() int64 { return 7 })

	err := o.OnIssueRegressed(context.Background(), nil, &types.InsightsIssue{ID: 5, TaskID: &id, LastRelease: "2.0.0"}, nil)
	if err != nil || len(tr.calls) != 1 || tr.calls[0].taskID != 9 {
		t.Errorf("err = %v, calls = %+v", err, tr.calls)
	}
}
