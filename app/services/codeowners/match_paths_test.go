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

package codeowners

import (
	"context"
	"io"
	"reflect"
	"strings"
	"testing"

	"github.com/harness/gitness/git"
	"github.com/harness/gitness/types"
)

func TestService_MatchPaths(t *testing.T) {
	const content = "* @all\n/app/ a@harness.io\n/app/gen/ \n*.md b@harness.io c@harness.io\n"
	s := &Service{
		git: &stubGit{
			treeNode: &git.GetTreeNodeOutput{
				Node: git.TreeNode{Mode: git.TreeNodeModeFile, SHA: "sha", Path: "CODEOWNERS"},
			},
			blob: &git.GetBlobOutput{
				Size:    int64(len(content)),
				Content: io.NopCloser(strings.NewReader(content)),
			},
		},
		config: Config{FilePaths: []string{"CODEOWNERS"}},
	}

	repo := &types.RepositoryCore{GitUID: "uid", DefaultBranch: "main"}
	got, err := s.MatchPaths(
		context.Background(), repo, "", []string{"app/x.go", "app/gen/y.go", "README.md", "z.txt"},
	)
	if err != nil {
		t.Fatalf("MatchPaths() error = %v", err)
	}

	gotPatterns := map[string]string{}
	for _, m := range got.Matches {
		gotPatterns[m.Path] = m.Entry.Pattern
	}
	want := map[string]string{"app/x.go": "/app/", "README.md": "*.md", "z.txt": "*"}
	if !reflect.DeepEqual(gotPatterns, want) {
		t.Errorf("MatchPaths() patterns = %v, want %v (app/gen is reset and must be omitted)", gotPatterns, want)
	}
}
