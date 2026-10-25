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

package request

import (
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	"github.com/harness/gitness/types/enum"
)

func TestParseTaskFilter(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet,
		"/tasks?state=abierta&state=triage&state=abierta&priority=p0&origin=shield&assignee_id=3&assignee_id=4"+
			"&assignee_group_id=9&is_agent=true&label_id=5&query=npe&page=2&limit=10", nil)

	f, err := ParseTaskFilter(r)
	if err != nil {
		t.Fatalf("ParseTaskFilter() error = %v", err)
	}

	if !slices.Equal(f.States, []enum.TaskState{enum.TaskStateOpen, enum.TaskStateTriage}) {
		t.Errorf("States = %v", f.States)
	}
	if !slices.Equal(f.Priorities, []enum.TaskPriority{enum.TaskPriorityP0}) {
		t.Errorf("Priorities = %v", f.Priorities)
	}
	if !slices.Equal(f.Origins, []enum.TaskOrigin{enum.TaskOriginShield}) {
		t.Errorf("Origins = %v", f.Origins)
	}
	if !slices.Equal(f.AssigneeIDs, []int64{3, 4}) || !slices.Equal(f.LabelIDs, []int64{5}) {
		t.Errorf("AssigneeIDs = %v, LabelIDs = %v", f.AssigneeIDs, f.LabelIDs)
	}
	if f.AssigneeGroupID == nil || *f.AssigneeGroupID != 9 {
		t.Errorf("AssigneeGroupID = %v", f.AssigneeGroupID)
	}
	if f.IsAgent == nil || !*f.IsAgent {
		t.Errorf("IsAgent = %v", f.IsAgent)
	}
	if f.Query != "npe" || f.Page != 2 || f.Size != 10 {
		t.Errorf("Query = %q, Page = %d, Size = %d", f.Query, f.Page, f.Size)
	}
}

func TestParseTaskFilterDefaultsAndErrors(t *testing.T) {
	f, err := ParseTaskFilter(httptest.NewRequest(http.MethodGet, "/tasks", nil))
	if err != nil {
		t.Fatalf("ParseTaskFilter() error = %v", err)
	}
	if f.Page != 1 || f.Size != PerPageDefault || f.IsAgent != nil || f.AssigneeGroupID != nil || len(f.States) != 0 {
		t.Errorf("unexpected defaults: %+v", f)
	}

	for _, q := range []string{
		"state=nope", "priority=p9", "origin=email", "assignee_id=x", "label_id=-1", "is_agent=maybe",
		"assignee_group_id=0",
	} {
		if _, err = ParseTaskFilter(httptest.NewRequest(http.MethodGet, "/tasks?"+q, nil)); err == nil {
			t.Errorf("ParseTaskFilter(%q) error = nil, want bad request", q)
		}
	}
}
