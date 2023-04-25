// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package parser

import (
	"strings"
	"testing"

	"github.com/harness/gitness/gitrpc/enum"
	"github.com/harness/gitness/gitrpc/internal/types"

	"github.com/google/go-cmp/cmp"
)

func TestGetHunkHeaders(t *testing.T) {
	input := `diff --git a/new_file.txt b/new_file.txt
new file mode 100644
index 0000000..fb0c863
--- /dev/null
+++ b/new_file.txt
@@ -0,0 +1,3 @@
+This is a new file
+created for this
+unit test.
diff --git a/old_file_name.txt b/changed_file.txt
index f043b93..e9449b5 100644
--- a/changed_file.txt
+++ b/changed_file.txt
@@ -7,3 +7,4 @@
 Unchanged line 
-Removed line 1
+Added line 1
+Added line 2
 Unchanged line 
@@ -27,2 +28,3 @@
 Unchanged line 
+Added line
 Unchanged line 
diff --git a/deleted_file.txt b/deleted_file.txt
deleted file mode 100644
index f043b93..0000000
--- a/deleted_file.txt
+++ /dev/null
@@ -1,3 +0,0 @@
-This is content of
-a deleted file
-in git diff output.
`

	got, err := GetHunkHeaders(strings.NewReader(input))
	if err != nil {
		t.Errorf("got error: %v", err)
		return
	}

	want := []*types.DiffFileHunkHeaders{
		{
			FileHeader: types.DiffFileHeader{
				OldFileName: "new_file.txt",
				NewFileName: "new_file.txt",
				Extensions: map[string]string{
					enum.DiffExtHeaderNewFileMode: "100644",
					enum.DiffExtHeaderIndex:       "0000000..fb0c863",
				},
			},
			HunksHeaders: []types.HunkHeader{{OldLine: 0, OldSpan: 0, NewLine: 1, NewSpan: 3}},
		},
		{
			FileHeader: types.DiffFileHeader{
				OldFileName: "old_file_name.txt",
				NewFileName: "changed_file.txt",
				Extensions: map[string]string{
					enum.DiffExtHeaderIndex: "f043b93..e9449b5 100644",
				},
			},
			HunksHeaders: []types.HunkHeader{
				{OldLine: 7, OldSpan: 3, NewLine: 7, NewSpan: 4},
				{OldLine: 27, OldSpan: 2, NewLine: 28, NewSpan: 3},
			},
		},
		{
			FileHeader: types.DiffFileHeader{
				OldFileName: "deleted_file.txt",
				NewFileName: "deleted_file.txt",
				Extensions: map[string]string{
					enum.DiffExtHeaderDeletedFileMode: "100644",
					enum.DiffExtHeaderIndex:           "f043b93..0000000",
				},
			},
			HunksHeaders: []types.HunkHeader{{OldLine: 1, OldSpan: 3, NewLine: 0, NewSpan: 0}},
		},
	}

	if diff := cmp.Diff(got, want); diff != "" {
		t.Errorf(diff)
	}
}
