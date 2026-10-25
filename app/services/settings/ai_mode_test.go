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

package settings

import (
	"testing"

	"github.com/harness/gitness/types/enum"
)

func TestValidateAIMode(t *testing.T) {
	for _, m := range []enum.AIMode{"", "full", "decisions-only", "off"} {
		if err := ValidateAIMode(&m); err != nil {
			t.Errorf("%q should be valid: %v", m, err)
		}
	}
	if err := ValidateAIMode(nil); err != nil {
		t.Errorf("nil should be valid: %v", err)
	}
	for _, m := range []enum.AIMode{"FULL", "completo", "on", "x"} {
		if err := ValidateAIMode(&m); err == nil {
			t.Errorf("%q should be invalid", m)
		}
	}
}

func TestDefaultAIModeForVisibility(t *testing.T) {
	if DefaultAIModeForVisibility(true) != enum.AIModeFull {
		t.Error("public should default to full")
	}
	if DefaultAIModeForVisibility(false) != enum.AIModeDecisionsOnly {
		t.Error("private should default to decisions-only")
	}
}
