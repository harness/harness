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

package openapi

import (
	"strings"
	"testing"
)

func TestTaskOperationsAreDocumented(t *testing.T) {
	spec := NewOpenAPIService().Generate()

	want := map[string][]string{
		"/repos/{repo_ref}/tasks":                          {"GET", "POST"},
		"/repos/{repo_ref}/tasks/{task_number}":            {"GET", "PATCH"},
		"/repos/{repo_ref}/tasks/{task_number}/state":      {"POST"},
		"/repos/{repo_ref}/tasks/{task_number}/resume":     {"POST"},
		"/repos/{repo_ref}/tasks/{task_number}/activities": {"GET"},
		"/repos/{repo_ref}/tasks/{task_number}/comments":   {"POST"},
		"/repos/{repo_ref}/tasks/{task_number}/links":      {"GET"},
		"/spaces/{space_ref}/tasks":                        {"GET"},
	}

	for path, methods := range want {
		item, ok := spec.Paths.MapOfPathItemValues[path]
		if !ok {
			t.Errorf("path %s is not documented", path)
			continue
		}
		for _, m := range methods {
			_, found := item.MapOfOperationValues[strings.ToLower(m)]
			if !found {
				t.Errorf("%s %s is not documented", m, path)
			}
		}
	}
}
