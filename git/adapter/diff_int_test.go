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

package adapter_test

import (
	"bytes"
	"context"
	"testing"
)

func TestAdapter_RawDiff(t *testing.T) {
	git := setupGit(t)
	repo, teardown := setupRepo(t, git, "testrawdiff")
	defer teardown()

	testFileName := "file.txt"

	baseBranch := "main"
	// write file to main branch
	oid1, parentSHA := writeFile(t, repo, testFileName, "some content", nil)

	err := repo.SetReference("refs/heads/"+baseBranch, parentSHA.String())
	if err != nil {
		t.Fatalf("failed updating reference '%s': %v", baseBranch, err)
	}

	baseTag := "0.0.1"
	err = repo.CreateAnnotatedTag(baseTag, "test tag 1", parentSHA.String())
	if err != nil {
		t.Fatalf("error creating annotated tag '%s': %v", baseTag, err)
	}

	headBranch := "dev"

	// create branch
	err = repo.CreateBranch(headBranch, baseBranch)
	if err != nil {
		t.Fatalf("failed creating a branch '%s': %v", headBranch, err)
	}

	// write file to main branch
	oid2, sha := writeFile(t, repo, testFileName, "new content", []string{parentSHA.String()})

	err = repo.SetReference("refs/heads/"+headBranch, sha.String())
	if err != nil {
		t.Fatalf("failed updating reference '%s': %v", headBranch, err)
	}

	headTag := "0.0.2"
	err = repo.CreateAnnotatedTag(headTag, "test tag 2", sha.String())
	if err != nil {
		t.Fatalf("error creating annotated tag '%s': %v", headTag, err)
	}

	want := `diff --git a/` + testFileName + ` b/` + testFileName + `
index ` + oid1.String() + `..` + oid2.String() + ` 100644
--- a/` + testFileName + `
+++ b/` + testFileName + `
@@ -1 +1 @@
-some content
\ No newline at end of file
+new content
\ No newline at end of file
`

	type args struct {
		ctx       context.Context
		repoPath  string
		baseRef   string
		headRef   string
		mergeBase bool
	}
	tests := []struct {
		name    string
		args    args
		wantW   string
		wantErr bool
	}{
		{
			name: "test branches",
			args: args{
				ctx:       context.Background(),
				repoPath:  repo.Path,
				baseRef:   baseBranch,
				headRef:   headBranch,
				mergeBase: false,
			},
			wantW:   want,
			wantErr: false,
		},
		{
			name: "test annotated tag",
			args: args{
				ctx:       context.Background(),
				repoPath:  repo.Path,
				baseRef:   baseTag,
				headRef:   headTag,
				mergeBase: false,
			},
			wantW:   want,
			wantErr: false,
		},
		{
			name: "test branches using merge-base",
			args: args{
				ctx:       context.Background(),
				repoPath:  repo.Path,
				baseRef:   baseBranch,
				headRef:   headBranch,
				mergeBase: true,
			},
			wantW:   want,
			wantErr: false,
		},
		{
			name: "test annotated tag using merge-base",
			args: args{
				ctx:       context.Background(),
				repoPath:  repo.Path,
				baseRef:   baseTag,
				headRef:   headTag,
				mergeBase: true,
			},
			wantW:   want,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &bytes.Buffer{}
			err := git.RawDiff(tt.args.ctx, w, tt.args.repoPath, tt.args.baseRef, tt.args.headRef, tt.args.mergeBase)
			if (err != nil) != tt.wantErr {
				t.Errorf("RawDiff() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotW := w.String(); gotW != tt.wantW {
				t.Errorf("RawDiff() gotW = %v, want %v", gotW, tt.wantW)
			}
		})
	}
}
