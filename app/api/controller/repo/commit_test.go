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

package repo

import (
	"strings"
	"testing"
)

func TestCommitFilesOptions_Sanitize(t *testing.T) {
	tests := []struct {
		name      string
		opts      CommitFilesOptions
		wantErr   bool
		errSubstr string
	}{
		{
			name: "valid with no new branch",
			opts: CommitFilesOptions{
				Title:   "my commit",
				Message: "description",
				Branch:  "main",
			},
			wantErr: false,
		},
		{
			name: "valid with new branch",
			opts: CommitFilesOptions{
				Title:     "my commit",
				Branch:    "main",
				NewBranch: "feat/new-feature",
			},
			wantErr: false,
		},
		{
			name: "invalid new branch with spaces",
			opts: CommitFilesOptions{
				Title:     "my commit",
				Branch:    "main",
				NewBranch: "feat: QA override",
			},
			wantErr:   true,
			errSubstr: "Invalid branch name",
		},
		{
			name: "invalid new branch with colon",
			opts: CommitFilesOptions{
				Title:     "my commit",
				Branch:    "main",
				NewBranch: "feat:new-branch",
			},
			wantErr:   true,
			errSubstr: "Invalid branch name",
		},
		{
			name: "invalid new branch with brackets",
			opts: CommitFilesOptions{
				Title:     "my commit",
				Branch:    "main",
				NewBranch: "feat/[CCM-123]-fix",
			},
			wantErr:   true,
			errSubstr: "Invalid branch name",
		},
		{
			name: "invalid new branch with tilde",
			opts: CommitFilesOptions{
				Title:     "my commit",
				Branch:    "main",
				NewBranch: "my~branch",
			},
			wantErr:   true,
			errSubstr: "Invalid branch name",
		},
		{
			name: "empty new branch is allowed (defaults to source branch)",
			opts: CommitFilesOptions{
				Title:  "my commit",
				Branch: "main",
			},
			wantErr: false,
		},
		{
			name:    "title too long",
			opts:    CommitFilesOptions{Title: strings.Repeat("a", 1025)},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.opts.Sanitize()
			if (err != nil) != tt.wantErr {
				t.Errorf("Sanitize() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.errSubstr != "" && err != nil {
				if !strings.Contains(err.Error(), tt.errSubstr) {
					t.Errorf("Sanitize() error = %q, want substring %q", err.Error(), tt.errSubstr)
				}
			}
		})
	}
}
