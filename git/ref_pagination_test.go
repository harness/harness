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

package git

import (
	"testing"

	"github.com/harness/gitness/git/api"
)

func TestWrapInstructorPageExtra(t *testing.T) {
	const pageSize = 3

	tests := []struct {
		name  string
		total int
		page  int32
		extra int32
		want  int
	}{
		{"last page exact, no extra", 6, 2, 0, 3},
		{"last page exact, extra sees nothing", 6, 2, 1, 3},
		{"full page, extra sees one more", 6, 1, 1, 4},
		{"partial last page", 5, 2, 1, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inst, _, err := wrapInstructorPage(api.DefaultInstructor, tt.page, pageSize, tt.extra)
			if err != nil {
				t.Fatal(err)
			}
			handled := 0
			for i := 0; i < tt.total; i++ {
				res, err := inst(api.WalkReferencesEntry{})
				if err != nil {
					t.Fatal(err)
				}
				if res == api.WalkInstructionStop {
					break
				}
				if res == api.WalkInstructionHandle {
					handled++
				}
			}
			if handled != tt.want {
				t.Fatalf("handled=%d want=%d", handled, tt.want)
			}
		})
	}
}
