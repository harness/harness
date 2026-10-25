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

package tasklink

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseReferences(t *testing.T) {
	tests := []struct {
		name  string
		texts []string
		want  []Reference
	}{
		{name: "empty", texts: []string{""}, want: nil},
		{name: "no references", texts: []string{"Add feature", "something else"}, want: nil},
		{name: "fixes", texts: []string{"Fixes TASK-12"}, want: []Reference{{12, true}}},
		{name: "fix", texts: []string{"Fix TASK-1"}, want: []Reference{{1, true}}},
		{name: "closes", texts: []string{"Closes TASK-2"}, want: []Reference{{2, true}}},
		{name: "close", texts: []string{"close TASK-2"}, want: []Reference{{2, true}}},
		{name: "resolves", texts: []string{"RESOLVES TASK-3"}, want: []Reference{{3, true}}},
		{name: "resolve", texts: []string{"resolve TASK-3"}, want: []Reference{{3, true}}},
		{name: "hash prefix", texts: []string{"Fixes #TASK-7"}, want: []Reference{{7, true}}},
		{name: "colon", texts: []string{"Fixes: TASK-7"}, want: []Reference{{7, true}}},
		{
			name:  "list with commas and and",
			texts: []string{"Fixes TASK-1, TASK-2 and #TASK-3"},
			want:  []Reference{{1, true}, {2, true}, {3, true}},
		},
		{
			name:  "multiple keywords across lines",
			texts: []string{"Title", "Fixes TASK-1\nCloses TASK-2"},
			want:  []Reference{{1, true}, {2, true}},
		},
		{name: "plain mention", texts: []string{"Related to TASK-5"}, want: []Reference{{5, false}}},
		{name: "hash mention", texts: []string{"see #TASK-5."}, want: []Reference{{5, false}}},
		{
			name:  "mix of closing and mention",
			texts: []string{"Fixes TASK-1, see also TASK-2"},
			want:  []Reference{{1, true}, {2, false}},
		},
		{
			name:  "mention then closing is closing",
			texts: []string{"TASK-4 is part of this", "Fixes TASK-4"},
			want:  []Reference{{4, true}},
		},
		{name: "duplicates", texts: []string{"Fixes TASK-1 and TASK-1"}, want: []Reference{{1, true}}},
		{name: "keyword only in other text", texts: []string{"Fixes bug", "TASK-9"}, want: []Reference{{9, false}}},
		{name: "title mention in parens", texts: []string{"(TASK-8) cleanup"}, want: []Reference{{8, false}}},
		{name: "glued to word", texts: []string{"SUBTASK-1 and sub-TASK-2 and xTASK-3"}, want: nil},
		{name: "lowercase task is ignored", texts: []string{"Fixes task-3"}, want: nil},
		{name: "keyword inside word", texts: []string{"Prefixes TASK-3"}, want: []Reference{{3, false}}},
		{name: "not a number", texts: []string{"Fixes TASK-abc"}, want: nil},
		{name: "zero", texts: []string{"Fixes TASK-0"}, want: nil},
		{name: "overflow", texts: []string{"Fixes TASK-99999999999999999999"}, want: nil},
		{name: "keyword on other line", texts: []string{"Fixes\nTASK-3"}, want: []Reference{{3, false}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, ParseReferences(tt.texts...))
		})
	}
}
