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

package database_test

import (
	"testing"

	"github.com/harness/gitness/app/store/database"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

func TestInsightsEventStore_ListReleases(t *testing.T) {
	ctx, env, teardown := setupInsightsEnv(t)
	defer teardown()

	p := newInsightsProject(1, "api", "key-api")
	other := newInsightsProject(1, "web", "key-web")
	for _, pr := range []*types.InsightsProject{p, other} {
		if err := env.projects.Create(ctx, pr); err != nil {
			t.Fatal(err)
		}
	}
	issue := newInsightsIssue(p, "fp", 1000)
	otherIssue := newInsightsIssue(other, "fp", 1000)
	for _, is := range []*types.InsightsIssue{issue, otherIssue} {
		if err := env.issues.Create(ctx, is); err != nil {
			t.Fatal(err)
		}
	}

	add := func(is *types.InsightsIssue, id, release string, ts int64) {
		t.Helper()
		if err := env.events.Create(ctx, &types.InsightsEvent{
			IssueID: is.ID, ProjectID: is.ProjectID, EventID: id, Timestamp: ts, Release: release,
			Payload: "{}", Created: ts,
		}); err != nil {
			t.Fatal(err)
		}
	}
	add(issue, "a1", "1.0.0", 300)
	add(issue, "a2", "1.0.0", 100)
	add(issue, "a3", "1.1.0", 500)
	add(issue, "a4", "", 50)
	add(otherIssue, "b1", "9.9.9", 10)

	got, err := env.events.ListReleases(ctx, p.ID)
	if err != nil {
		t.Fatalf("ListReleases() error = %v", err)
	}
	if len(got) != 2 || got[0].Release != "1.0.0" || got[0].FirstSeen != 100 ||
		got[1].Release != "1.1.0" || got[1].FirstSeen != 500 {
		t.Errorf("ListReleases() = %+v", got)
	}
}

func TestTaskLinkStore_ListByStateAndLinkKind(t *testing.T) {
	ctx, env, teardown := setupTaskEnv(t)
	defer teardown()

	mk := func(state enum.TaskState, resolved *int64) *types.Task {
		tk := newTask(1, "t")
		tk.State = state
		tk.Resolved = resolved
		if err := env.tasks.Create(ctx, tk); err != nil {
			t.Fatal(err)
		}
		return tk
	}
	res := int64(1234)
	resolvedA := mk(enum.TaskStateResolved, &res)
	resolvedB := mk(enum.TaskStateResolved, nil)
	open := mk(enum.TaskStateOpen, nil)
	pr := mk(enum.TaskStateResolved, &res)

	links := database.NewTaskLinkStore(env.db)
	for _, l := range []*types.TaskLink{
		{TaskID: resolvedA.ID, Kind: enum.TaskLinkKindInsightsIssue, RefID: "7", Created: 1},
		{TaskID: resolvedB.ID, Kind: enum.TaskLinkKindInsightsIssue, RefID: "8", Created: 1},
		{TaskID: open.ID, Kind: enum.TaskLinkKindInsightsIssue, RefID: "9", Created: 1},
		{TaskID: pr.ID, Kind: enum.TaskLinkKindPullReq, RefID: "1", Created: 1},
	} {
		if err := links.Create(ctx, l); err != nil {
			t.Fatal(err)
		}
	}

	got, err := links.ListByStateAndLinkKind(ctx, enum.TaskStateResolved, enum.TaskLinkKindInsightsIssue, 0, 10)
	if err != nil {
		t.Fatalf("ListByStateAndLinkKind() error = %v", err)
	}
	if len(got) != 2 || got[0].TaskID != resolvedA.ID || got[0].RefID != "7" || got[0].Resolved != 1234 ||
		got[1].TaskID != resolvedB.ID || got[1].Resolved != 0 {
		t.Errorf("ListByStateAndLinkKind() = %+v", got)
	}

	got, err = links.ListByStateAndLinkKind(ctx, enum.TaskStateResolved, enum.TaskLinkKindInsightsIssue, resolvedA.ID, 1)
	if err != nil || len(got) != 1 || got[0].TaskID != resolvedB.ID {
		t.Errorf("paged ListByStateAndLinkKind() = %+v, %v", got, err)
	}
}
