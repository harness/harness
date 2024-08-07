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

package migrate

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

func TestGenerateThreads(t *testing.T) {
	// comments with treelike structure
	t0 := time.Now()
	comments := []ExternalComment{
		/* 0 */ {ID: 1, Body: "A", ParentID: 0},
		/* 1 */ {ID: 2, Body: "B", ParentID: 0},
		/* 2 */ {ID: 3, Body: "A1", ParentID: 1},
		/* 3 */ {ID: 4, Body: "B1", ParentID: 2},
		/* 4 */ {ID: 5, Body: "A2", ParentID: 1},
		/* 5 */ {ID: 6, Body: "A2X", ParentID: 5},
		/* 6 */ {ID: 7, Body: "A1X", ParentID: 3},
		/* 7 */ {ID: 8, Body: "B1X", ParentID: 4},
		/* 8 */ {ID: 9, Body: "C", ParentID: 0},
		/* 9 */ {ID: 10, Body: "D1", ParentID: 11}, // Wrong order - a reply before its parent
		/* 10 */ {ID: 11, Body: "D", ParentID: 0},
		{ID: 20, Body: "Self-parent", ParentID: 20},   // Invalid
		{ID: 30, Body: "Crosslinked-X", ParentID: 31}, // Invalid
		{ID: 31, Body: "Crosslinked-Y", ParentID: 30}, // Invalid
	}

	for i := range comments {
		comments[i].Created = t0.Add(time.Duration(i) * time.Minute)
	}

	// flattened threads with top level comments and a list of replies to each of them
	wantThreads := []*externalCommentThread{
		{
			TopLevel: comments[0],                                                           // A
			Replies:  []ExternalComment{comments[2], comments[4], comments[5], comments[6]}, // A1, A2, A2X, A1X
		},
		{
			TopLevel: comments[1],                                 // B
			Replies:  []ExternalComment{comments[3], comments[7]}, // B1, B1X
		},
		{
			TopLevel: comments[8], // C
			Replies:  []ExternalComment{},
		},
		{
			TopLevel: comments[10],                   // D
			Replies:  []ExternalComment{comments[9]}, // D1
		},
	}

	gotThreads := generateThreads(comments)
	if diff := cmp.Diff(gotThreads, wantThreads); diff != "" {
		t.Errorf(diff)
	}
}
