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
	"context"
	"encoding/base64"
	"errors"
	"strings"
	"testing"

	"github.com/harness/gitness/app/services/filevalidator"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/git/api"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/require"
)

func TestCommitFilesOptions_Sanitize(t *testing.T) {
	tests := []struct {
		name          string
		opts          CommitFilesOptions
		wantErr       bool
		errSubstr     string
		wantBranch    string
		wantNewBranch string
	}{
		{
			name: "valid with no new branch",
			opts: CommitFilesOptions{
				Title:   "my commit",
				Message: "description",
				Branch:  "main",
			},
			wantErr:    false,
			wantBranch: "main",
		},
		{
			name: "valid with new branch",
			opts: CommitFilesOptions{
				Title:     "my commit",
				Branch:    "main",
				NewBranch: "feat/new-feature",
			},
			wantErr:       false,
			wantBranch:    "main",
			wantNewBranch: "feat/new-feature",
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
			wantErr:    false,
			wantBranch: "main",
		},
		{
			name:    "title too long",
			opts:    CommitFilesOptions{Title: strings.Repeat("a", 1025)},
			wantErr: true,
		},
		{
			name: "branch with refs/heads/ prefix is stripped",
			opts: CommitFilesOptions{
				Title:  "my commit",
				Branch: "refs/heads/test-abhishek",
			},
			wantErr:    false,
			wantBranch: "test-abhishek",
		},
		{
			name: "new branch with refs/heads/ prefix is stripped",
			opts: CommitFilesOptions{
				Title:     "my commit",
				Branch:    "main",
				NewBranch: "refs/heads/feat/new-feature",
			},
			wantErr:       false,
			wantBranch:    "main",
			wantNewBranch: "feat/new-feature",
		},
		{
			name: "both branch and new branch with refs/heads/ prefix are stripped",
			opts: CommitFilesOptions{
				Title:     "my commit",
				Branch:    "refs/heads/main",
				NewBranch: "refs/heads/feat/new-feature",
			},
			wantErr:       false,
			wantBranch:    "main",
			wantNewBranch: "feat/new-feature",
		},
		{
			name: "branch with refs/heads/ prefix and whitespace",
			opts: CommitFilesOptions{
				Title:  "my commit",
				Branch: "  refs/heads/test-branch  ",
			},
			wantErr:    false,
			wantBranch: "test-branch",
		},
		{
			name: "branch without refs/heads/ prefix is unchanged",
			opts: CommitFilesOptions{
				Title:  "my commit",
				Branch: "feature/my-branch",
			},
			wantErr:    false,
			wantBranch: "feature/my-branch",
		},
		{
			name: "new branch with refs/heads/ prefix and invalid chars still fails validation",
			opts: CommitFilesOptions{
				Title:     "my commit",
				Branch:    "main",
				NewBranch: "refs/heads/my~branch",
			},
			wantErr:   true,
			errSubstr: "Invalid branch name",
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
			if err == nil && tt.wantBranch != "" {
				if tt.opts.Branch != tt.wantBranch {
					t.Errorf("Sanitize() Branch = %q, want %q", tt.opts.Branch, tt.wantBranch)
				}
			}
			if err == nil && tt.wantNewBranch != "" {
				if tt.opts.NewBranch != tt.wantNewBranch {
					t.Errorf("Sanitize() NewBranch = %q, want %q", tt.opts.NewBranch, tt.wantNewBranch)
				}
			}
		})
	}
}

type recordingValidator struct {
	paths []string
	err   error
	seen  map[string][]byte
}

func (v *recordingValidator) Paths() []string { return v.paths }

func (v *recordingValidator) Validate(
	_ context.Context,
	_ *types.RepositoryCore,
	content []byte,
) ([]types.ContentViolation, error) {
	if v.err != nil {
		return nil, v.err
	}
	// The registry does not pass the path, so key the stub by call order via content.
	v.seen[string(content)] = content
	return []types.ContentViolation{{Code: "bad", Message: "bad content"}}, nil
}

func newValidationController(t *testing.T, v *recordingValidator) *Controller {
	t.Helper()
	v.seen = map[string][]byte{}
	registry, err := filevalidator.NewRegistry(v)
	require.NoError(t, err)
	return &Controller{contentValidators: registry}
}

func TestValidateActionContent_SkipsUnclaimedAndDeletes(t *testing.T) {
	v := &recordingValidator{paths: []string{"CODEOWNERS"}}
	c := newValidationController(t, v)

	got := c.validateActionContent(context.Background(), &types.RepositoryCore{}, []CommitFileAction{
		{Action: git.CreateAction, Path: "src/main.go", Payload: "package main"},
		{Action: git.DeleteAction, Path: "CODEOWNERS"},
	})

	require.Empty(t, got)
	require.Empty(t, v.seen)
}

func TestValidateActionContent_DecodesBase64Payload(t *testing.T) {
	v := &recordingValidator{paths: []string{"CODEOWNERS"}}
	c := newValidationController(t, v)

	got := c.validateActionContent(context.Background(), &types.RepositoryCore{}, []CommitFileAction{{
		Action:   git.UpdateAction,
		Path:     "CODEOWNERS",
		Payload:  base64.StdEncoding.EncodeToString([]byte("* ghost@harness.io\n")),
		Encoding: enum.ContentEncodingTypeBase64,
	}})

	require.Len(t, got, 1)
	require.Equal(t, "CODEOWNERS", got[0].Path)
	require.Contains(t, v.seen, "* ghost@harness.io\n")
}

func TestValidateActionContent_MoveUsesDestinationPathAndContent(t *testing.T) {
	t.Run("destination claimed", func(t *testing.T) {
		v := &recordingValidator{paths: []string{"CODEOWNERS"}}
		c := newValidationController(t, v)
		got := c.validateActionContent(context.Background(), &types.RepositoryCore{}, []CommitFileAction{
			{Action: git.MoveAction, Path: "OWNERS", Payload: "CODEOWNERS\x00* ghost@harness.io\n"},
		})
		require.Len(t, got, 1)
		require.Equal(t, "CODEOWNERS", got[0].Path)
		require.Contains(t, v.seen, "* ghost@harness.io\n")
	})
	t.Run("destination unclaimed", func(t *testing.T) {
		v := &recordingValidator{paths: []string{"CODEOWNERS"}}
		c := newValidationController(t, v)
		got := c.validateActionContent(context.Background(), &types.RepositoryCore{}, []CommitFileAction{
			{Action: git.MoveAction, Path: "CODEOWNERS", Payload: "OWNERS\x00* ghost@harness.io\n"},
		})
		require.Empty(t, got)
		require.Empty(t, v.seen)
	})
	t.Run("pure rename is not content", func(t *testing.T) {
		v := &recordingValidator{paths: []string{"CODEOWNERS"}}
		c := newValidationController(t, v)
		got := c.validateActionContent(context.Background(), &types.RepositoryCore{}, []CommitFileAction{
			{Action: git.MoveAction, Path: "OWNERS", Payload: "CODEOWNERS"},
		})
		require.Empty(t, got)
		require.Empty(t, v.seen)
	})
}

// Git writes the file at the cleaned path, so validation must claim it by that same path: an action path that
// only resolves to CODEOWNERS after cleaning still writes CODEOWNERS.
func TestValidateActionContent_NormalizesPathTheWayGitWritesIt(t *testing.T) {
	for _, path := range []string{"docs/../CODEOWNERS", "./CODEOWNERS", "/CODEOWNERS", "a//CODEOWNERS/"} {
		t.Run(path, func(t *testing.T) {
			v := &recordingValidator{paths: []string{"CODEOWNERS", "a/CODEOWNERS"}}
			c := newValidationController(t, v)

			got := c.validateActionContent(context.Background(), &types.RepositoryCore{}, []CommitFileAction{
				{Action: git.UpdateAction, Path: path, Payload: "* ghost@harness.io\n"},
			})

			require.Len(t, got, 1)
			require.Equal(t, api.CleanUploadFileName(path), got[0].Path)
		})
	}
}

func TestValidateActionContent_ReportsPerPath(t *testing.T) {
	v := &recordingValidator{paths: []string{"CODEOWNERS", ".harness/CODEOWNERS"}}
	c := newValidationController(t, v)

	got := c.validateActionContent(context.Background(), &types.RepositoryCore{}, []CommitFileAction{
		{Action: git.CreateAction, Path: "CODEOWNERS", Payload: "a"},
		{Action: git.UpdateAction, Path: ".harness/CODEOWNERS", Payload: "b"},
	})

	require.Len(t, got, 2)
	require.Equal(t, "CODEOWNERS", got[0].Path)
	require.Equal(t, ".harness/CODEOWNERS", got[1].Path)
	require.Len(t, got[0].Violations, 1)
	require.Len(t, got[1].Violations, 1)
}

func TestValidateActionContent_ValidatorErrorYieldsNoViolations(t *testing.T) {
	v := &recordingValidator{paths: []string{"CODEOWNERS"}, err: errors.New("lookup outage")}
	c := newValidationController(t, v)

	got := c.validateActionContent(context.Background(), &types.RepositoryCore{}, []CommitFileAction{
		{Action: git.CreateAction, Path: "CODEOWNERS", Payload: "* a@b.c\n"},
	})

	require.Empty(t, got)
}
