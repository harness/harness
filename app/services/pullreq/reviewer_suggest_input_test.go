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
	"math"
	"strings"
	"testing"

	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/require"
)

func TestReviewerSuggestBatchInput_Sanitize_Legacy(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		err := (&ReviewerSuggestBatchInput{}).Sanitize()
		require.ErrorContains(t, err, "must not be empty")
	})

	t.Run("too many", func(t *testing.T) {
		ids := make([]int64, maxReviewerSuggestions+1)
		for i := range ids {
			ids[i] = int64(i + 1)
		}
		err := (&ReviewerSuggestBatchInput{ReviewerIDs: ids}).Sanitize()
		require.ErrorContains(t, err, "must not exceed 100")
	})

	t.Run("non positive", func(t *testing.T) {
		for _, id := range []int64{0, -1} {
			err := (&ReviewerSuggestBatchInput{ReviewerIDs: []int64{3, id}}).Sanitize()
			require.ErrorContains(t, err, "greater than 0")
		}
	})

	t.Run("sorted and compacted", func(t *testing.T) {
		in := &ReviewerSuggestBatchInput{ReviewerIDs: []int64{5, 2, 5, 3, 2}}
		require.NoError(t, in.Sanitize())
		require.Equal(t, []int64{2, 3, 5}, in.ReviewerIDs)
	})
}

func TestReviewerSuggestBatchInput_Sanitize_BothFormsRejected(t *testing.T) {
	in := &ReviewerSuggestBatchInput{
		ReviewerIDs: []int64{1},
		Suggestions: []ReviewerSuggestionInput{{PrincipalID: 2}},
	}
	require.ErrorContains(t, in.Sanitize(), "reviewer_ids and suggestions")
}

func TestReviewerSuggestBatchInput_Sanitize_SuggestionsEmptyRejected(t *testing.T) {
	in := &ReviewerSuggestBatchInput{Suggestions: []ReviewerSuggestionInput{}}
	require.ErrorContains(t, in.Sanitize(), "must not be empty")
}

func TestReviewerSuggestBatchInput_Sanitize_SuggestionsTooMany(t *testing.T) {
	build := func(n int) *ReviewerSuggestBatchInput {
		rows := make([]ReviewerSuggestionInput, n)
		for i := range rows {
			rows[i] = ReviewerSuggestionInput{PrincipalID: int64(i + 1)}
		}
		return &ReviewerSuggestBatchInput{Suggestions: rows}
	}

	require.NoError(t, build(MaxSuggestionsPerPullReq).Sanitize())
	require.ErrorContains(t, build(MaxSuggestionsPerPullReq+1).Sanitize(), "50")
}

func TestReviewerSuggestBatchInput_Sanitize_ReasonBoundary(t *testing.T) {
	one := func(reason string) *ReviewerSuggestBatchInput {
		return &ReviewerSuggestBatchInput{
			Suggestions: []ReviewerSuggestionInput{{PrincipalID: 1, Reason: reason}},
		}
	}

	require.NoError(t, one(strings.Repeat("a", 500)).Sanitize())
	require.ErrorContains(t, one(strings.Repeat("a", 501)).Sanitize(), "500")

	// Limit counts runes, not bytes: 500 three-byte runes are accepted, 501 are rejected.
	require.NoError(t, one(strings.Repeat("é", 500)).Sanitize())
	require.NoError(t, one(strings.Repeat("世", 500)).Sanitize())
	require.ErrorContains(t, one(strings.Repeat("世", 501)).Sanitize(), "500")
}

func TestReviewerSuggestBatchInput_Sanitize_UnknownKindRejected(t *testing.T) {
	in := &ReviewerSuggestBatchInput{
		Suggestions: []ReviewerSuggestionInput{{PrincipalID: 1, Kind: "REQUIRED"}},
	}
	require.ErrorContains(t, in.Sanitize(), "kind")

	in = &ReviewerSuggestBatchInput{
		Suggestions: []ReviewerSuggestionInput{{PrincipalID: 1, Kind: ""}},
	}
	require.NoError(t, in.Sanitize())
	require.Equal(t, enum.ReviewerSuggestionKindOptional, in.Suggestions[0].Kind)

	in = &ReviewerSuggestBatchInput{
		Suggestions: []ReviewerSuggestionInput{{PrincipalID: 1, Kind: enum.ReviewerSuggestionKindNotNeeded}},
	}
	require.NoError(t, in.Sanitize())
	require.Equal(t, enum.ReviewerSuggestionKindNotNeeded, in.Suggestions[0].Kind)
}

func TestReviewerSuggestBatchInput_Sanitize_DuplicatePrincipalLastWins(t *testing.T) {
	in := &ReviewerSuggestBatchInput{
		Suggestions: []ReviewerSuggestionInput{
			{PrincipalID: 9, Reason: "other"},
			{PrincipalID: 7, Reason: "first"},
			{PrincipalID: 7, Reason: "second"},
		},
	}
	require.NoError(t, in.Sanitize())
	require.Len(t, in.Suggestions, 2)
	// Sorted by principal_id.
	require.Equal(t, int64(7), in.Suggestions[0].PrincipalID)
	require.Equal(t, "second", in.Suggestions[0].Reason)
	require.Equal(t, int64(9), in.Suggestions[1].PrincipalID)
}

func TestReviewerSuggestBatchInput_Sanitize_RankBoundary(t *testing.T) {
	one := func(rank *int64) *ReviewerSuggestBatchInput {
		return &ReviewerSuggestBatchInput{
			Suggestions: []ReviewerSuggestionInput{{PrincipalID: 1, Rank: rank}},
		}
	}

	rank := func(v int64) *int64 { return &v }

	require.NoError(t, one(nil).Sanitize())
	require.NoError(t, one(rank(0)).Sanitize())
	require.NoError(t, one(rank(math.MaxInt32)).Sanitize())

	require.ErrorContains(t, one(rank(-1)).Sanitize(), "rank")
	require.ErrorContains(t, one(rank(math.MaxInt32+1)).Sanitize(), "rank")
}

func TestReviewerSuggestBatchInput_Sanitize_CommitSHABoundary(t *testing.T) {
	one := func(sha string) *ReviewerSuggestBatchInput {
		return &ReviewerSuggestBatchInput{
			Suggestions: []ReviewerSuggestionInput{{PrincipalID: 1, CommitSHA: sha}},
		}
	}

	require.NoError(t, one("").Sanitize())
	require.NoError(t, one(strings.Repeat("a", maxSuggestionCommitSHALen)).Sanitize())
	require.ErrorContains(t, one(strings.Repeat("a", maxSuggestionCommitSHALen+1)).Sanitize(), "commit_sha")
}

func TestReviewerSuggestBatchInput_Sanitize_NonPositivePrincipalRejected(t *testing.T) {
	for _, id := range []int64{0, -1} {
		in := &ReviewerSuggestBatchInput{Suggestions: []ReviewerSuggestionInput{{PrincipalID: id}}}
		require.ErrorContains(t, in.Sanitize(), "greater than 0")
	}
}
