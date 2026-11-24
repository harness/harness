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

package enum_test

import (
	"testing"

	"github.com/harness/gitness/types/enum"
)

func TestReviewerSuggestionKind_Sanitize(t *testing.T) {
	// Empty defaults to optional.
	got, ok := enum.ReviewerSuggestionKind("").Sanitize()
	if !ok || got != enum.ReviewerSuggestionKindOptional {
		t.Errorf("empty: got (%q, %v), want (%q, true)", got, ok, enum.ReviewerSuggestionKindOptional)
	}

	// Known values round-trip.
	for _, v := range []enum.ReviewerSuggestionKind{
		enum.ReviewerSuggestionKindRequired,
		enum.ReviewerSuggestionKindOptional,
		enum.ReviewerSuggestionKindNotNeeded,
	} {
		got, ok = v.Sanitize()
		if !ok || got != v {
			t.Errorf("%q: got (%q, %v), want (%q, true)", v, got, ok, v)
		}
	}

	// Unknown and wrongly-cased values are rejected.
	for _, v := range []enum.ReviewerSuggestionKind{"REQUIRED", "blocker", "Optional"} {
		if _, ok := v.Sanitize(); ok {
			t.Errorf("%q: expected not ok", v)
		}
	}
}
