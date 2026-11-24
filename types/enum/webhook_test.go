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

package enum

import (
	"slices"
	"testing"
)

func TestWebhookTriggerPullReqDraftChanged(t *testing.T) {
	if WebhookTriggerPullReqDraftChanged != "pullreq_draft_changed" {
		t.Fatalf("unexpected trigger value %q", WebhookTriggerPullReqDraftChanged)
	}

	got, ok := WebhookTriggerPullReqDraftChanged.Sanitize()
	if !ok {
		t.Fatal("expected pullreq_draft_changed to be accepted by Sanitize")
	}
	if got != WebhookTriggerPullReqDraftChanged {
		t.Fatalf("Sanitize returned %q, want %q", got, WebhookTriggerPullReqDraftChanged)
	}

	if !slices.Contains(WebhookTrigger("").Enum(), any(WebhookTriggerPullReqDraftChanged)) {
		t.Fatal("expected pullreq_draft_changed to be listed in WebhookTrigger.Enum()")
	}
}

// TestWebhookTriggers_SortedAndUnique guards the binary-search based Sanitize:
// every registered trigger must be found, which requires a sorted, duplicate-free list.
func TestWebhookTriggers_SortedAndUnique(t *testing.T) {
	all, _ := GetAllWebhookTriggers()

	if !slices.IsSorted(all) {
		t.Fatal("webhook triggers must be sorted")
	}

	seen := make(map[WebhookTrigger]struct{}, len(all))
	for _, trigger := range all {
		if _, dup := seen[trigger]; dup {
			t.Fatalf("duplicate webhook trigger %q", trigger)
		}
		seen[trigger] = struct{}{}

		if _, ok := trigger.Sanitize(); !ok {
			t.Errorf("registered trigger %q rejected by Sanitize", trigger)
		}
	}
}
