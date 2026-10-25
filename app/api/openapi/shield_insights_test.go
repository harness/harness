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

package openapi

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestShieldInsightsSystemAIOperationsAreDocumented(t *testing.T) {
	spec := NewOpenAPIService().Generate()

	want := map[string][]string{
		"/repos/{repo_ref}/shield/scans":                              {"GET", "POST"},
		"/repos/{repo_ref}/shield/findings":                           {"GET"},
		"/repos/{repo_ref}/shield/findings/{finding_id}":              {"PATCH"},
		"/repos/{repo_ref}/insights/projects":                         {"GET", "POST"},
		"/repos/{repo_ref}/insights/projects/{project_id}":            {"PATCH", "DELETE"},
		"/repos/{repo_ref}/insights/projects/{project_id}/rotate-key": {"POST"},
		"/repos/{repo_ref}/insights/issues":                           {"GET"},
		"/repos/{repo_ref}/insights/issues/{issue_id}":                {"GET", "PATCH"},
		"/repos/{repo_ref}/insights/issues/{issue_id}/events":         {"GET"},
		"/system/ai/models":                                           {"GET"},
		"/system/ai/quota":                                            {"GET"},
	}

	for path, methods := range want {
		item, ok := spec.Paths.MapOfPathItemValues[path]
		if !ok {
			t.Errorf("path %s is not documented", path)
			continue
		}
		for _, m := range methods {
			if _, found := item.MapOfOperationValues[strings.ToLower(m)]; !found {
				t.Errorf("%s %s is not documented", m, path)
			}
		}
	}
}

func TestWebhookTaskTriggersAreExposed(t *testing.T) {
	spec := NewOpenAPIService().Generate()

	raw, err := json.Marshal(spec)
	if err != nil {
		t.Fatal(err)
	}
	for _, trigger := range []string{"task_created", "task_updated", "task_resolved"} {
		if !strings.Contains(string(raw), `"`+trigger+`"`) {
			t.Errorf("webhook trigger %s is not part of the spec", trigger)
		}
	}
}
