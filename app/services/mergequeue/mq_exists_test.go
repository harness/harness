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

package mergequeue

import (
	"testing"

	"github.com/harness/gitness/types/enum"
)

func TestViolation(t *testing.T) {
	v := Violation("main")

	if v.Rule.Identifier != RuleIdentifier {
		t.Errorf("want rule identifier %q, got %q", RuleIdentifier, v.Rule.Identifier)
	}

	if v.Rule.State != enum.RuleStateActive {
		t.Errorf("want rule state %q, got %q", enum.RuleStateActive, v.Rule.State)
	}

	if len(v.Violations) != 1 {
		t.Fatalf("want exactly 1 violation entry, got %d", len(v.Violations))
	}

	if v.Violations[0].Code != RuleIdentifier {
		t.Errorf("want violation code %q, got %q", RuleIdentifier, v.Violations[0].Code)
	}

	if v.Violations[0].Message == "" {
		t.Error("violation message must not be empty")
	}
}
