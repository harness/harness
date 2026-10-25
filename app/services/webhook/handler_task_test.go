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

package webhook

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

func TestTaskInfoFrom(t *testing.T) {
	assignee := int64(5)
	task := &types.Task{
		Number:              12,
		Title:               "Fix the leak",
		State:               enum.TaskStateInProgress,
		Priority:            enum.TaskPriorityP1,
		Origin:              enum.TaskOriginShield,
		AssigneePrincipalID: &assignee,
		AssigneeIsAgent:     true,
	}
	repo := &types.Repository{Path: "space/repo"}

	info := taskInfoFrom(context.Background(), task, repo, &mockURLProvider{})

	if info.Number != 12 || info.Title != "Fix the leak" || info.State != enum.TaskStateInProgress ||
		info.Priority != enum.TaskPriorityP1 || info.Origin != enum.TaskOriginShield {
		t.Fatalf("unexpected task info: %+v", info)
	}
	if info.Assignee.PrincipalID == nil || *info.Assignee.PrincipalID != 5 || !info.Assignee.IsAgent {
		t.Fatalf("unexpected assignee: %+v", info.Assignee)
	}
	if want := "http://example.com/repo/space/repo/tasks/12"; info.URL != want {
		t.Fatalf("url = %q, want %q", info.URL, want)
	}
}

func TestTaskUpdatedPayloadJSON(t *testing.T) {
	payload := TaskUpdatedPayload{
		BaseSegment: BaseSegment{Trigger: enum.WebhookTriggerTaskUpdated},
		Task:        TaskInfo{Number: 1},
		Change:      TaskChangeInfo{Kind: "state", From: "abierta", To: "triage"},
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got map[string]any
	if err = json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	for _, key := range []string{"trigger", "repo", "principal", "task", "change"} {
		if _, ok := got[key]; !ok {
			t.Fatalf("payload misses %q: %s", key, raw)
		}
	}
	if got["trigger"] != "task_updated" {
		t.Fatalf("trigger = %v", got["trigger"])
	}
}

func TestTaskTriggersAreValid(t *testing.T) {
	for _, tr := range []enum.WebhookTrigger{
		enum.WebhookTriggerTaskCreated, enum.WebhookTriggerTaskUpdated, enum.WebhookTriggerTaskResolved,
	} {
		if _, ok := tr.Sanitize(); !ok {
			t.Fatalf("trigger %q is not accepted", tr)
		}
	}
}
