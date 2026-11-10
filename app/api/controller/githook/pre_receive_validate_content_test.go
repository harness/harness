// Copyright 2026 Harness, Inc.
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

package githook

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/harness/gitness/app/services/filevalidator"
	gitnesserrors "github.com/harness/gitness/errors"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/git/hook"
	"github.com/harness/gitness/git/sha"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/require"
)

const (
	oldCommit = "1111111111111111111111111111111111111111"
	newCommit = "2222222222222222222222222222222222222222"
	oldBlob   = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	newBlob   = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	otherBlob = "cccccccccccccccccccccccccccccccccccccccc"

	otherCommit = "3333333333333333333333333333333333333333"
)

// ghostValidator flags any content containing "ghost" as invalid on line 1.
type ghostValidator struct{}

func (ghostValidator) Paths() []string { return []string{"CODEOWNERS"} }

func (ghostValidator) Validate(
	_ context.Context, _ *types.RepositoryCore, content []byte,
) ([]types.ContentViolation, error) {
	if strings.Contains(string(content), "ghost") {
		return []types.ContentViolation{{Code: "user_not_found", Message: "unknown owner", Line: 1}}, nil
	}
	return nil, nil
}

// stubRestrictedGIT answers tree-node and blob reads from canned data and records what it was asked.
type stubRestrictedGIT struct {
	RestrictedGIT

	// nodes maps a commit to the blob SHA of CODEOWNERS there. A missing commit means the file is absent.
	nodes map[string]string
	// blobs maps a blob SHA to its content. When nil, every blob reads back as content.
	blobs   map[string]string
	content string
	blobErr error

	blobCalls  int
	treeParams []*git.GetTreeNodeParams
	blobParams []*git.GetBlobParams
}

func (s *stubRestrictedGIT) GetTreeNode(
	_ context.Context, params *git.GetTreeNodeParams,
) (*git.GetTreeNodeOutput, error) {
	s.treeParams = append(s.treeParams, params)
	blobSHA, ok := s.nodes[params.GitREF]
	if !ok {
		return nil, gitnesserrors.NotFound("path not found")
	}
	return &git.GetTreeNodeOutput{Node: git.TreeNode{SHA: blobSHA}}, nil
}

func (s *stubRestrictedGIT) GetBlob(_ context.Context, params *git.GetBlobParams) (*git.GetBlobOutput, error) {
	s.blobCalls++
	s.blobParams = append(s.blobParams, params)
	if s.blobErr != nil {
		return nil, s.blobErr
	}
	content := s.content
	if s.blobs != nil {
		content = s.blobs[params.SHA]
	}
	return &git.GetBlobOutput{Content: io.NopCloser(strings.NewReader(content))}, nil
}

func newContentController(t *testing.T) *Controller {
	t.Helper()
	registry, err := filevalidator.NewRegistry(ghostValidator{})
	require.NoError(t, err)
	return &Controller{contentValidators: registry}
}

func pushInput(ref, oldSHA, newSHA string, altDirs ...string) types.GithookPreReceiveInput {
	in := types.GithookPreReceiveInput{
		PreReceiveInput: hook.PreReceiveInput{
			Environment: hook.Environment{AlternateObjectDirs: altDirs},
			RefUpdates: []hook.ReferenceUpdate{{
				Ref: ref, Old: sha.Must(oldSHA), New: sha.Must(newSHA),
			}},
		},
	}
	in.OperationType = enum.GitOpTypeGitPush
	return in
}

func TestValidateChangedContent_SkipsWhenBlobUnchanged(t *testing.T) {
	rgit := &stubRestrictedGIT{
		nodes:   map[string]string{oldCommit: newBlob, newCommit: newBlob},
		content: "* ghost@harness.io\n",
	}
	got := newContentController(t).validateChangedContent(
		context.Background(), rgit, newTestRepo(), pushInput("refs/heads/main", oldCommit, newCommit))

	require.Empty(t, got)
	require.Zero(t, rgit.blobCalls)
}

func TestValidateChangedContent_ValidatesWhenBlobChanged(t *testing.T) {
	rgit := &stubRestrictedGIT{
		nodes:   map[string]string{oldCommit: oldBlob, newCommit: newBlob},
		content: "* ghost@harness.io\n",
	}
	got := newContentController(t).validateChangedContent(
		context.Background(), rgit, newTestRepo(), pushInput("refs/heads/main", oldCommit, newCommit))

	require.Len(t, got, 1)
	require.Equal(t, "CODEOWNERS", got[0].Path)
	require.Equal(t, "user_not_found", got[0].Violations[0].Code)
}

func TestValidateChangedContent_ValidatesOnNewBranch(t *testing.T) {
	rgit := &stubRestrictedGIT{
		nodes:   map[string]string{newCommit: newBlob},
		content: "* ghost@harness.io\n",
	}
	got := newContentController(t).validateChangedContent(
		context.Background(), rgit, newTestRepo(), pushInput("refs/heads/feature", sha.Nil.String(), newCommit))

	require.Len(t, got, 1)
}

func TestValidateChangedContent_SkipsDeletedRefsAndTags(t *testing.T) {
	rgit := &stubRestrictedGIT{
		nodes:   map[string]string{oldCommit: oldBlob, newCommit: newBlob},
		content: "* ghost@harness.io\n",
	}
	c := newContentController(t)

	deleted := c.validateChangedContent(
		context.Background(), rgit, newTestRepo(), pushInput("refs/heads/main", oldCommit, sha.Nil.String()))
	tag := c.validateChangedContent(
		context.Background(), rgit, newTestRepo(), pushInput("refs/tags/v1", sha.Nil.String(), newCommit))

	require.Empty(t, deleted)
	require.Empty(t, tag)
	require.Zero(t, rgit.blobCalls)
}

func TestValidateChangedContent_SkipsDeletedFile(t *testing.T) {
	rgit := &stubRestrictedGIT{
		nodes:   map[string]string{oldCommit: oldBlob}, // absent at the new commit
		content: "* ghost@harness.io\n",
	}
	got := newContentController(t).validateChangedContent(
		context.Background(), rgit, newTestRepo(), pushInput("refs/heads/main", oldCommit, newCommit))

	require.Empty(t, got)
	require.Zero(t, rgit.blobCalls)
}

func TestValidateChangedContent_PassesAlternateObjectDirs(t *testing.T) {
	rgit := &stubRestrictedGIT{
		nodes:   map[string]string{oldCommit: oldBlob, newCommit: newBlob},
		content: "* ghost@harness.io\n",
	}
	dirs := []string{"/quarantine/objects"}
	newContentController(t).validateChangedContent(
		context.Background(), rgit, newTestRepo(), pushInput("refs/heads/main", oldCommit, newCommit, dirs...))

	require.NotEmpty(t, rgit.treeParams)
	for _, p := range rgit.treeParams {
		require.Equal(t, dirs, p.AlternateObjectDirs)
	}
	require.Len(t, rgit.blobParams, 1)
	require.Equal(t, dirs, rgit.blobParams[0].AlternateObjectDirs)
}

func TestValidateChangedContent_GitErrorYieldsNoViolations(t *testing.T) {
	rgit := &stubRestrictedGIT{
		nodes:   map[string]string{oldCommit: oldBlob, newCommit: newBlob},
		blobErr: errors.New("object store down"),
	}
	got := newContentController(t).validateChangedContent(
		context.Background(), rgit, newTestRepo(), pushInput("refs/heads/main", oldCommit, newCommit))

	require.Empty(t, got)
}

func TestContentValidationMessages(t *testing.T) {
	got := contentValidationMessages([]types.ContentValidation{{
		Path: ".harness/CODEOWNERS",
		Violations: []types.ContentViolation{
			{Message: "unknown owner", Line: 3},
			{Message: "file too large"},
		},
	}})

	require.Len(t, got, 2)
	require.Contains(t, got[0], ".harness/CODEOWNERS")
	require.Contains(t, got[0], "line 3")
	require.Contains(t, got[0], "unknown owner")
	require.Contains(t, got[1], ".harness/CODEOWNERS")
	require.Contains(t, got[1], "file too large")
	require.NotContains(t, got[1], "line")
	require.Empty(t, contentValidationMessages(nil))
}

func TestValidateChangedContent_NewBranchSkipsFileIdenticalToDefaultBranch(t *testing.T) {
	// "main" is the repo's default branch; the new branch carries the very same (already invalid) CODEOWNERS.
	rgit := &stubRestrictedGIT{
		nodes:   map[string]string{"main": newBlob, newCommit: newBlob},
		content: "* ghost@harness.io\n",
	}
	got := newContentController(t).validateChangedContent(
		context.Background(), rgit, newTestRepo(), pushInput("refs/heads/feature", sha.Nil.String(), newCommit))

	require.Empty(t, got)
	require.Zero(t, rgit.blobCalls)
}

func TestValidateChangedContent_NewBranchValidatesFileDifferingFromDefaultBranch(t *testing.T) {
	rgit := &stubRestrictedGIT{
		nodes:   map[string]string{"main": oldBlob, newCommit: newBlob},
		content: "* ghost@harness.io\n",
	}
	got := newContentController(t).validateChangedContent(
		context.Background(), rgit, newTestRepo(), pushInput("refs/heads/feature", sha.Nil.String(), newCommit))

	require.Len(t, got, 1)
}

func TestValidateChangedContent_SkipsCommitAPIOperations(t *testing.T) {
	// The commit API already reported on this content in its response; validating again would double the work.
	rgit := &stubRestrictedGIT{
		nodes:   map[string]string{oldCommit: oldBlob, newCommit: newBlob},
		content: "* ghost@harness.io\n",
	}
	in := pushInput("refs/heads/main", oldCommit, newCommit)
	in.OperationType = enum.GitOpTypeAPIContent

	got := newContentController(t).validateChangedContent(context.Background(), rgit, newTestRepo(), in)

	require.Empty(t, got)
	require.Empty(t, rgit.treeParams)
}

// Only a direct git push warrants content warnings: the commit API reports violations in its own response
// (with or without bypass_rules), and system-initiated pushes have no human pusher to read a warning.
func TestValidateChangedContent_OnlyValidatesDirectGitPushes(t *testing.T) {
	skipped := []enum.GitOpType{
		enum.GitOpTypeAPIContent,
		enum.GitOpTypeAPIContentBypassRules,
		enum.GitOpTypeAPILinkedSync,
		enum.GitOpTypeAPISystemRefs,
		enum.GitOpTypeMergeQueue,
		enum.GitOpTypeManageRepo,
		enum.GitOpTypeAPIRefsOnly,
	}

	for _, opType := range skipped {
		t.Run(string(opType), func(t *testing.T) {
			rgit := &stubRestrictedGIT{
				nodes:   map[string]string{oldCommit: oldBlob, newCommit: newBlob},
				content: "* ghost@harness.io\n",
			}
			in := pushInput("refs/heads/main", oldCommit, newCommit)
			in.OperationType = opType

			got := newContentController(t).validateChangedContent(context.Background(), rgit, newTestRepo(), in)

			require.Empty(t, got)
			require.Empty(t, rgit.treeParams, "no git reads should be spent on %s", opType)
		})
	}

	t.Run(string(enum.GitOpTypeGitPush), func(t *testing.T) {
		rgit := &stubRestrictedGIT{
			nodes:   map[string]string{oldCommit: oldBlob, newCommit: newBlob},
			content: "* ghost@harness.io\n",
		}
		in := pushInput("refs/heads/main", oldCommit, newCommit)
		in.OperationType = enum.GitOpTypeGitPush

		got := newContentController(t).validateChangedContent(context.Background(), rgit, newTestRepo(), in)

		require.Len(t, got, 1)
	})
}

// multiRefPushInput builds a push that updates several branches at once, each ref given as
// "ref:oldSHA:newSHA".
func multiRefPushInput(refs ...[3]string) types.GithookPreReceiveInput {
	updates := make([]hook.ReferenceUpdate, 0, len(refs))
	for _, ref := range refs {
		updates = append(updates, hook.ReferenceUpdate{
			Ref: ref[0], Old: sha.Must(ref[1]), New: sha.Must(ref[2]),
		})
	}
	in := types.GithookPreReceiveInput{
		PreReceiveInput: hook.PreReceiveInput{RefUpdates: updates},
	}
	in.OperationType = enum.GitOpTypeGitPush
	return in
}

// A single push can update CODEOWNERS on several branches. Each branch's version is its own file,
// so each must be validated - warning only about the first branch's content hides the rest.
func TestValidateChangedContent_ValidatesEachBranchsVersionOfTheFile(t *testing.T) {
	rgit := &stubRestrictedGIT{
		nodes: map[string]string{
			oldCommit:   oldBlob,
			newCommit:   newBlob,
			otherCommit: otherBlob,
		},
		blobs: map[string]string{
			oldBlob:   "* admin@harness.io\n",
			newBlob:   "* ghost@harness.io\n",
			otherBlob: "* ghost@harness.io\n* ghost2@harness.io\n",
		},
	}

	got := newContentController(t).validateChangedContent(context.Background(), rgit, newTestRepo(),
		multiRefPushInput(
			[3]string{"refs/heads/one", oldCommit, newCommit},
			[3]string{"refs/heads/two", oldCommit, otherCommit},
		))

	require.Len(t, got, 2, "both branches carry a different invalid CODEOWNERS")
}

// Pushing a stack of branches that all carry the same new CODEOWNERS must warn once, not once
// per branch: the content is identical, so the warning would be a verbatim repeat.
func TestValidateChangedContent_WarnsOnceWhenBranchesShareContent(t *testing.T) {
	rgit := &stubRestrictedGIT{
		nodes: map[string]string{
			oldCommit:   oldBlob,
			newCommit:   newBlob,
			otherCommit: newBlob, // same blob as newCommit
		},
		blobs: map[string]string{
			oldBlob: "* admin@harness.io\n",
			newBlob: "* ghost@harness.io\n",
		},
	}

	got := newContentController(t).validateChangedContent(context.Background(), rgit, newTestRepo(),
		multiRefPushInput(
			[3]string{"refs/heads/one", oldCommit, newCommit},
			[3]string{"refs/heads/two", oldCommit, otherCommit},
		))

	require.Len(t, got, 1)
	require.Equal(t, 1, rgit.blobCalls, "identical content must not be read or validated twice")
}

func TestContentValidationMessages_CapsLongLists(t *testing.T) {
	// A CODEOWNERS whose owners were all deprovisioned must not echo thousands of lines over the sideband
	// on every push.
	violations := make([]types.ContentViolation, 0, 500)
	for i := 1; i <= 500; i++ {
		violations = append(violations, types.ContentViolation{Message: "unknown owner", Line: int64(i)})
	}

	got := contentValidationMessages([]types.ContentValidation{{Path: "CODEOWNERS", Violations: violations}})

	require.Len(t, got, maxContentValidationMessages+1)
	require.Contains(t, got[len(got)-1], "470 more")
}
