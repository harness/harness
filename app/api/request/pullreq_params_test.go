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

package request

import (
	"testing"

	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/require"
)

func TestPullReqPathParams(t *testing.T) {
	req := newTestRequest(t, "/",
		PathParamPullReqNumber, "12",
		PathParamReviewerID, "34",
		PathParamUserGroupID, "56",
		PathParamPullReqCommentID, "78",
		PathParamSourceBranch, "feature%2Fx",
		PathParamTargetBranch, "main",
	)

	number, err := GetPullReqNumberFromPath(req)
	require.NoError(t, err)
	require.Equal(t, int64(12), number)

	reviewer, err := GetReviewerIDFromPath(req)
	require.NoError(t, err)
	require.Equal(t, int64(34), reviewer)

	group, err := GetUserGroupIDFromPath(req)
	require.NoError(t, err)
	require.Equal(t, int64(56), group)

	comment, err := GetPullReqCommentIDPath(req)
	require.NoError(t, err)
	require.Equal(t, int64(78), comment)

	source, err := GetPullReqSourceBranchFromPath(req)
	require.NoError(t, err)
	require.Equal(t, "feature/x", source)

	target, err := GetPullReqTargetBranchFromPath(req)
	require.NoError(t, err)
	require.Equal(t, "main", target)

	empty := newTestRequest(t, "/")
	for _, get := range []func() error{
		func() error { _, err := GetPullReqNumberFromPath(empty); return err },
		func() error { _, err := GetReviewerIDFromPath(empty); return err },
		func() error { _, err := GetUserGroupIDFromPath(empty); return err },
		func() error { _, err := GetPullReqCommentIDPath(empty); return err },
		func() error { _, err := GetPullReqSourceBranchFromPath(empty); return err },
		func() error { _, err := GetPullReqTargetBranchFromPath(empty); return err },
	} {
		requireBadRequest(t, get(), "not found")
	}
}

func TestGetPullReqCommentReactionEmojiFromPath(t *testing.T) {
	emojis, _ := enum.GetAllPullReqCommentReactionEmojis()
	require.NotEmpty(t, emojis)

	req := newTestRequest(t, "/", PathParamPullReqCommentReactionEmoji, string(emojis[0]))
	got, err := GetPullReqCommentReactionEmojiFromPath(req)
	require.NoError(t, err)
	require.Equal(t, emojis[0], got)

	_, err = GetPullReqCommentReactionEmojiFromPath(
		newTestRequest(t, "/", PathParamPullReqCommentReactionEmoji, "not-an-emoji"))
	require.True(t, errors.IsInvalidArgument(err))

	_, err = GetPullReqCommentReactionEmojiFromPath(newTestRequest(t, "/"))
	requireBadRequest(t, err, "not found")
}

func TestParseSortPullReq(t *testing.T) {
	require.Equal(t, enum.PullReqSort(enum.PullReqSortUpdated), ParseSortPullReq(newTestRequest(t, "/?sort=updated")))
	require.Equal(t, enum.PullReqSort(enum.PullReqSortCreated), ParseSortPullReq(newTestRequest(t, "/?sort=created")))
	// unknown values fall back to the default sort
	require.Equal(t, enum.PullReqSort(enum.PullReqSortNumber), ParseSortPullReq(newTestRequest(t, "/?sort=bogus")))
	require.Equal(t, enum.PullReqSort(enum.PullReqSortNumber), ParseSortPullReq(newTestRequest(t, "/")))
}

func TestParsePullReqStatesAndDecisions(t *testing.T) {
	states := parsePullReqStates(newTestRequest(t, "/?state=open&state=merged&state=open&state=bogus"))
	require.ElementsMatch(t, []enum.PullReqState{enum.PullReqStateOpen, enum.PullReqStateMerged}, states)
	require.Empty(t, parsePullReqStates(newTestRequest(t, "/")))

	decisions, _ := enum.GetAllPullReqReviewDecisions()
	require.NotEmpty(t, decisions)
	got := parseReviewDecisions(newTestRequest(t, "/?review_decision="+string(decisions[0])+
		"&review_decision="+string(decisions[0])+"&review_decision=bogus"))
	require.Equal(t, []enum.PullReqReviewDecision{decisions[0]}, got)
	require.Empty(t, parseReviewDecisions(newTestRequest(t, "/")))
}

func TestParsePullReqMetadataOptions(t *testing.T) {
	got, err := ParsePullReqMetadataOptions(newTestRequest(t, "/"))
	require.NoError(t, err)
	require.Equal(t, types.PullReqMetadataOptions{IncludeGitStats: true}, got)

	got, err = ParsePullReqMetadataOptions(newTestRequest(t, "/?include_checks=true&include_rules=true"))
	require.NoError(t, err)
	require.True(t, got.IncludeChecks)
	require.True(t, got.IncludeRules)
	require.True(t, got.IncludeGitStats)

	_, err = ParsePullReqMetadataOptions(newTestRequest(t, "/?include_checks=maybe"))
	requireBadRequest(t, err, "must be a boolean")
	_, err = ParsePullReqMetadataOptions(newTestRequest(t, "/?include_rules=maybe"))
	requireBadRequest(t, err, "must be a boolean")
}

func TestParsePullReqFilter(t *testing.T) {
	t.Run("defaults", func(t *testing.T) {
		got, err := ParsePullReqFilter(newTestRequest(t, "/"))
		require.NoError(t, err)
		require.Equal(t, 1, got.Page)
		require.Equal(t, PerPageDefault, got.Size)
		require.Empty(t, got.CreatedBy)
		require.Empty(t, got.States)
		require.False(t, got.ExcludeDescription)
	})

	t.Run("all options", func(t *testing.T) {
		decisions, _ := enum.GetAllPullReqReviewDecisions()
		query := "/?page=2&limit=5&query=fix&created_by=1&created_by=2&author_id=3&label_id=4&value_id=5" +
			"&commenter_id=6&reviewer_id=7&mentioned_id=8&exclude_description=true" +
			"&source_repo_ref=space%2Frepo&source_branch=feature&target_branch=main" +
			"&state=open&sort=created&order=desc&review_decision=" + string(decisions[0]) +
			"&created_lt=100&updated_gt=50&edited_lt=70"

		got, err := ParsePullReqFilter(newTestRequest(t, query))
		require.NoError(t, err)

		require.Equal(t, 2, got.Page)
		require.Equal(t, 5, got.Size)
		require.Equal(t, "fix", got.Query)
		require.Equal(t, []int64{1, 2, 3}, got.CreatedBy, "the author is merged into the creators")
		require.Equal(t, []int64{4}, got.LabelID)
		require.Equal(t, []int64{5}, got.ValueID)
		require.Equal(t, int64(6), got.CommenterID)
		require.Equal(t, int64(7), got.ReviewerID)
		require.Equal(t, int64(8), got.MentionedID)
		require.True(t, got.ExcludeDescription)
		require.Equal(t, "space/repo", got.SourceRepoRef)
		require.Equal(t, "feature", got.SourceBranch)
		require.Equal(t, "main", got.TargetBranch)
		require.Equal(t, []enum.PullReqState{enum.PullReqStateOpen}, got.States)
		require.Equal(t, enum.PullReqSort(enum.PullReqSortCreated), got.Sort)
		require.Equal(t, enum.OrderDesc, got.Order)
		require.Equal(t, []enum.PullReqReviewDecision{decisions[0]}, got.ReviewDecisions)
		require.Equal(t, int64(100), got.CreatedLt)
		require.Equal(t, int64(50), got.UpdatedGt)
		require.Equal(t, int64(70), got.EditedLt)
		require.True(t, got.IncludeGitStats)
	})

	t.Run("review decisions need a reviewer", func(t *testing.T) {
		decisions, _ := enum.GetAllPullReqReviewDecisions()
		_, err := ParsePullReqFilter(newTestRequest(t, "/?review_decision="+string(decisions[0])))
		require.True(t, errors.IsInvalidArgument(err))
	})

	t.Run("invalid values", func(t *testing.T) {
		for _, query := range []string{
			"created_by=x", "label_id=0", "value_id=-1", "created_lt=x", "updated_lt=x", "edited_lt=x",
			"exclude_description=maybe", "author_id=x", "commenter_id=x", "reviewer_id=x", "mentioned_id=x",
			"include_checks=maybe", "include_rules=maybe",
		} {
			_, err := ParsePullReqFilter(newTestRequest(t, "/?"+query))
			require.Error(t, err, query)
		}
	})
}

func TestParsePullReqActivityFilter(t *testing.T) {
	types1, _ := enum.GetAllPullReqActivityTypes()
	kinds, _ := enum.GetAllPullReqActivityKinds()
	require.NotEmpty(t, types1)
	require.NotEmpty(t, kinds)

	got, err := ParsePullReqActivityFilter(newTestRequest(t, "/"))
	require.NoError(t, err)
	require.Equal(t, &types.PullReqActivityFilter{}, got)

	got, err = ParsePullReqActivityFilter(newTestRequest(t,
		"/?after=1&before=2&limit=3&type="+string(types1[0])+"&type="+string(types1[0])+"&type=bogus"+
			"&kind="+string(kinds[0])+"&kind=bogus"))
	require.NoError(t, err)
	require.Equal(t, int64(1), got.After)
	require.Equal(t, int64(2), got.Before)
	require.Equal(t, 3, got.Limit)
	require.Equal(t, []enum.PullReqActivityType{types1[0]}, got.Types)
	require.Equal(t, []enum.PullReqActivityKind{kinds[0]}, got.Kinds)

	for _, query := range []string{"after=x", "before=0", "limit=-1"} {
		_, err := ParsePullReqActivityFilter(newTestRequest(t, "/?"+query))
		requireBadRequest(t, err, "must be a positive integer")
	}
}
