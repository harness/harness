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

package triage

import (
	"strings"

	"github.com/harness/gitness/types"
)

// InAppPaths returns the file paths of the in-app stack frames of the event, innermost first and without
// duplicates. Only relative file names are returned: absolute paths of the reporting machine are of no use
// in the repository.
func InAppPaths(ev *types.InsightsEvent) []string {
	if ev == nil {
		return nil
	}
	var out []string
	seen := map[string]bool{}
	for _, f := range parseEvent(ev).Frames {
		p := strings.TrimPrefix(strings.TrimSpace(f.Filename), "./")
		if p == "" || strings.HasPrefix(p, "/") || strings.Contains(p, "://") || seen[p] {
			continue
		}
		seen[p] = true
		out = append(out, p)
	}
	return out
}
