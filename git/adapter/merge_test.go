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
	"context"
	"testing"

	"github.com/harness/gitness/git/adapter"
)

func TestAdapter_GetMergeBase(t *testing.T) {
	git := setupGit(t)
	repo, teardown := setupRepo(t, git, "testmergebase")
	defer teardown()

	baseBranch := "main"
	// write file to main branch
	parentSHA := writeFile(t, repo, "file1.txt", "some content", nil)

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
	sha := writeFile(t, repo, "file1.txt", "new content", []string{parentSHA.String()})

	err = repo.SetReference("refs/heads/"+headBranch, sha.String())
	if err != nil {
		t.Fatalf("failed updating reference '%s': %v", headBranch, err)
	}

	headTag := "0.0.2"
	err = repo.CreateAnnotatedTag(headTag, "test tag 2", sha.String())
	if err != nil {
		t.Fatalf("error creating annotated tag '%s': %v", headTag, err)
	}

	type args struct {
		ctx      context.Context
		repoPath string
		remote   string
		base     string
		head     string
	}
	tests := []struct {
		name    string
		git     adapter.Adapter
		args    args
		want    string
		want1   string
		wantErr bool
	}{
		{
			name: "git merge base using branch names",
			git:  git,
			args: args{
				ctx:      context.Background(),
				repoPath: repo.Path,
				remote:   "",
				base:     baseBranch,
				head:     headBranch,
			},
			want:  parentSHA.String(),
			want1: baseBranch,
		},
		{
			name: "git merge base using annotated tags",
			git:  git,
			args: args{
				ctx:      context.Background(),
				repoPath: repo.Path,
				remote:   "",
				base:     baseTag,
				head:     headTag,
			},
			want:  parentSHA.String(),
			want1: baseTag,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, err := tt.git.GetMergeBase(tt.args.ctx, tt.args.repoPath, tt.args.remote, tt.args.base, tt.args.head)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetMergeBase() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GetMergeBase() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("GetMergeBase() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}
