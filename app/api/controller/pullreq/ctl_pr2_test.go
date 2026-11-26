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

package pullreq

import (
	"context"
	"strings"
	"testing"

	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/require"
)

func TestCtlCommentCreateInput(t *testing.T) {
	require.False(t, (&CommentCreateInput{}).IsReply())
	require.True(t, (&CommentCreateInput{ParentID: 3}).IsReply())
	require.False(t, (&CommentCreateInput{}).IsCodeComment())
	require.True(t, (&CommentCreateInput{SourceCommitSHA: "abc"}).IsCodeComment())

	tests := []struct {
		name    string
		in      CommentCreateInput
		wantErr bool
	}{
		{"plain comment", CommentCreateInput{Text: "  hello  "}, false},
		{"only source sha", CommentCreateInput{Text: "x", SourceCommitSHA: "a"}, true},
		{"only target sha", CommentCreateInput{Text: "x", TargetCommitSHA: "a"}, true},
		{"reply as code comment", CommentCreateInput{
			Text: "x", SourceCommitSHA: "a", TargetCommitSHA: "b", ParentID: 1,
		}, true},
		{"no path", CommentCreateInput{Text: "x", SourceCommitSHA: "a", TargetCommitSHA: "b"}, true},
		{"no lines", CommentCreateInput{
			Text: "x", SourceCommitSHA: "a", TargetCommitSHA: "b", Path: "f.go",
		}, true},
		{"mixed sides", CommentCreateInput{
			Text: "x", SourceCommitSHA: "a", TargetCommitSHA: "b", Path: "f.go",
			LineStart: 1, LineEnd: 2, LineStartNew: true,
		}, true},
		{"mixed sides reversed", CommentCreateInput{
			Text: "x", SourceCommitSHA: "a", TargetCommitSHA: "b", Path: "f.go",
			LineStart: 1, LineEnd: 2, LineEndNew: true,
		}, true},
		{"valid code comment", CommentCreateInput{
			Text: "x", SourceCommitSHA: "a", TargetCommitSHA: "b", Path: "f.go",
			LineStart: 1, LineEnd: 2, LineStartNew: true, LineEndNew: true,
		}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.in.Sanitize()
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}

	trimmed := CommentCreateInput{Text: "  hello  "}
	require.NoError(t, trimmed.Sanitize())
	require.Equal(t, "hello", trimmed.Text)
}

func TestCtlCommentUpdateAndStatusInput(t *testing.T) {
	in := &CommentUpdateInput{Text: "  new  "}
	require.NoError(t, in.Sanitize())
	require.Equal(t, "new", in.Text)
	require.Error(t, (&CommentUpdateInput{Text: strings.Repeat("x", 100000)}).Sanitize())

	require.True(t, in.hasChanges(&types.PullReqActivity{Text: "old"}))
	require.False(t, in.hasChanges(&types.PullReqActivity{Text: "new"}))

	require.NoError(t, (&CommentStatusInput{Status: enum.PullReqCommentStatusResolved}).Validate())
	require.NoError(t, (&CommentStatusInput{Status: enum.PullReqCommentStatusActive}).Validate())
	require.Error(t, (&CommentStatusInput{Status: "nope"}).Validate())

	resolvedBy := int64(7)
	resolved := int64(100)
	active := &types.PullReqActivity{}
	done := &types.PullReqActivity{Resolved: &resolved, ResolvedBy: &resolvedBy}

	setResolved := &CommentStatusInput{Status: enum.PullReqCommentStatusResolved}
	require.True(t, setResolved.hasChanges(active, 7))
	require.True(t, setResolved.hasChanges(done, 8), "resolved by someone else")
	require.False(t, setResolved.hasChanges(done, 7))
	require.True(t, setResolved.hasChanges(&types.PullReqActivity{Resolved: &resolved}, 7))

	setActive := &CommentStatusInput{Status: enum.PullReqCommentStatusActive}
	require.True(t, setActive.hasChanges(done, 7))
	require.False(t, setActive.hasChanges(active, 7))
}

func TestCtlApplySuggestionsInput(t *testing.T) {
	require.Error(t, (&CommentApplySuggestionsInput{}).sanitize())
	require.Error(t, (&CommentApplySuggestionsInput{
		Suggestions: []SuggestionReference{{CommentID: 0, CheckSum: "x"}},
	}).sanitize())
	require.Error(t, (&CommentApplySuggestionsInput{
		Suggestions: []SuggestionReference{{CommentID: 1, CheckSum: "  "}},
	}).sanitize())

	in := &CommentApplySuggestionsInput{
		Suggestions: []SuggestionReference{{CommentID: 1, CheckSum: " sum "}},
		Title:       "  title ",
		Message:     " msg  ",
	}
	require.NoError(t, in.sanitize())
	require.Equal(t, "title", in.Title)
	require.Equal(t, "msg", in.Message)
}

func TestCtlFileViewAddInput(t *testing.T) {
	sha := strings.Repeat("a", 40)
	require.NoError(t, (&FileViewAddInput{Path: "a.go", CommitSHA: sha}).Validate())
	require.Error(t, (&FileViewAddInput{CommitSHA: sha}).Validate())
	require.Error(t, (&FileViewAddInput{Path: "a.go", CommitSHA: "short"}).Validate())
}

func TestCtlAutoMergeAndQueueEnableInput(t *testing.T) {
	require.Error(t, (&AutoMergeEnableInput{}).sanitize())
	require.Error(t, (&AutoMergeEnableInput{Method: "nope"}).sanitize())
	require.Error(t, (&AutoMergeEnableInput{Method: enum.MergeMethodRebase, Title: "t"}).sanitize())
	require.Error(t, (&AutoMergeEnableInput{Method: enum.MergeMethodFastForward, Message: "m"}).sanitize())

	in := &AutoMergeEnableInput{Method: enum.MergeMethodSquash, Title: " t ", Message: " m "}
	require.NoError(t, in.sanitize())
	require.Equal(t, "t", in.Title)
	require.Equal(t, "m", in.Message)

	require.Error(t, (&MergeQueueEnableInput{}).sanitize())
	require.Error(t, (&MergeQueueEnableInput{Method: "nope"}).sanitize())
	require.Error(t, (&MergeQueueEnableInput{Method: enum.MergeMethodFastForward}).sanitize())
	require.NoError(t, (&MergeQueueEnableInput{Method: enum.MergeMethodMerge}).sanitize())
}

func TestCtlParseMentions(t *testing.T) {
	ctx := context.Background()

	require.Empty(t, parseMentions(ctx, "no mentions"))
	require.Equal(t, []int64{1, 22}, parseMentions(ctx, "hi @[1] and @[22], not @[x] or @[]"))
	require.Equal(t, []int64{5, 5}, parseMentions(ctx, "@[5]@[5]"))

	// A huge number overflows int64 and is skipped.
	require.Empty(t, parseMentions(ctx, "@[99999999999999999999999]"))

	e := ctlNewEnv(t)
	infos, err := e.ctrl.processMentions(ctx, "nothing")
	require.NoError(t, err)
	require.Empty(t, infos)

	infos, err = e.ctrl.processMentions(ctx, "cc @[1]")
	require.NoError(t, err)
	require.Len(t, infos, 1)
	require.Equal(t, e.admin.UID, infos[e.admin.ID].UID)
}
