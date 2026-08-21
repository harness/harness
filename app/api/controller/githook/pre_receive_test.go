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

package githook

import (
	"testing"

	"github.com/harness/gitness/types/enum"
)

func TestCheckPushProtection_DoesNotBypassForPlainAPIContent(t *testing.T) {
	t.Parallel()

	if allowPushProtectionBypass(enum.GitOpTypeAPIContent) {
		t.Fatal("plain API content must not bypass push protection")
	}
}

func TestCheckPushProtection_DoesBypassForAPIContentBypassRules(t *testing.T) {
	t.Parallel()

	if !allowPushProtectionBypass(enum.GitOpTypeAPIContentBypassRules) {
		t.Fatal("API content with an explicit bypass request must bypass push protection")
	}
}

func TestAllowPushProtectionBypass(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		opType     enum.GitOpType
		wantBypass bool
	}{
		{name: "git push", opType: enum.GitOpTypeGitPush, wantBypass: true},
		{name: "refs only", opType: enum.GitOpTypeAPIRefsOnly, wantBypass: false},
		{name: "system refs", opType: enum.GitOpTypeAPISystemRefs, wantBypass: false},
		{name: "linked sync", opType: enum.GitOpTypeAPILinkedSync, wantBypass: false},
		{name: "plain API content", opType: enum.GitOpTypeAPIContent, wantBypass: false},
		{name: "explicit API bypass", opType: enum.GitOpTypeAPIContentBypassRules, wantBypass: true},
		{name: "repository management", opType: enum.GitOpTypeManageRepo, wantBypass: false},
		{name: "merge queue", opType: enum.GitOpTypeMergeQueue, wantBypass: false},
		{name: "unknown operation", opType: enum.GitOpType("unknown"), wantBypass: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := allowPushProtectionBypass(tt.opType); got != tt.wantBypass {
				t.Fatalf("allowPushProtectionBypass(%q) = %t, want %t", tt.opType, got, tt.wantBypass)
			}
		})
	}
}
