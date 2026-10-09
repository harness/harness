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

package check

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"testing"

	"github.com/harness/gitness/app/api/request"
	"github.com/harness/gitness/app/api/usererror"
	appstore "github.com/harness/gitness/app/store"
	"github.com/harness/gitness/git/sha"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

const (
	summarySHA1 = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	summarySHA2 = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
)

type summaryCheckStore struct {
	appstore.CheckStore
	summary             map[sha.SHA]types.CheckCountSummary
	err                 error
	calls               int
	repoID              int64
	commitSHAs          []string
	excludePayloadKinds []enum.CheckPayloadKind
}

func (s *summaryCheckStore) ResultSummary(
	_ context.Context,
	repoID int64,
	commitSHAs []string,
	excludePayloadKinds []enum.CheckPayloadKind,
) (map[sha.SHA]types.CheckCountSummary, error) {
	s.calls++
	s.repoID = repoID
	s.commitSHAs = commitSHAs
	s.excludePayloadKinds = excludePayloadKinds
	return s.summary, s.err
}

func TestSummaryInput_Sanitize(t *testing.T) {
	tooMany := make([]string, request.PerPageMax+1)
	for i := range tooMany {
		tooMany[i] = fmt.Sprintf("%040x", i+1)
	}

	tests := []struct {
		name    string
		input   SummaryInput
		status  int
		message string
	}{
		{
			name:    "too many SHAs",
			input:   SummaryInput{CommitSHAs: tooMany},
			status:  http.StatusRequestEntityTooLarge,
			message: usererror.ErrRequestTooLarge.Message,
		},
		{
			name:    "invalid SHA",
			input:   SummaryInput{CommitSHAs: []string{summarySHA1, "not-a-sha"}},
			status:  http.StatusBadRequest,
			message: "Invalid commit SHA at index 1.",
		},
		{
			name:    "abbreviated SHA",
			input:   SummaryInput{CommitSHAs: []string{"aaaaaaa"}},
			status:  http.StatusBadRequest,
			message: "Invalid commit SHA at index 0.",
		},
		{
			name:    "uppercase SHA",
			input:   SummaryInput{CommitSHAs: []string{"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"}},
			status:  http.StatusBadRequest,
			message: "Invalid commit SHA at index 0.",
		},
		{
			name: "too many payload kinds",
			input: SummaryInput{
				CommitSHAs:          []string{summarySHA1},
				ExcludePayloadKinds: []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "10", "11"},
			},
			status:  http.StatusBadRequest,
			message: "At most 10 payload kinds can be excluded.",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.input.Sanitize()
			var userErr *usererror.Error
			if !errors.As(err, &userErr) || userErr.Status != tt.status || userErr.Message != tt.message {
				t.Fatalf("Sanitize() error = %v, want %d %q", err, tt.status, tt.message)
			}
		})
	}
}

func TestSummaryInput_Sanitize_sortsAndDeduplicates(t *testing.T) {
	in := SummaryInput{CommitSHAs: []string{summarySHA2, summarySHA1, summarySHA2}}
	if err := in.Sanitize(); err != nil {
		t.Fatalf("Sanitize() error = %v", err)
	}
	if !slices.Equal(in.CommitSHAs, []string{summarySHA1, summarySHA2}) {
		t.Fatalf("CommitSHAs = %v", in.CommitSHAs)
	}
}

func Test_listCheckSummaries_empty(t *testing.T) {
	store := &summaryCheckStore{}
	controller := &Controller{checkStore: store}

	for _, in := range []*SummaryInput{nil, {}, {CommitSHAs: []string{}}} {
		summaries, err := controller.listCheckSummaries(context.Background(), 1, in)
		if err != nil {
			t.Fatalf("listCheckSummaries() error = %v", err)
		}
		if len(summaries) != 0 {
			t.Fatalf("listCheckSummaries() = %v, want empty list", summaries)
		}
	}
	if store.calls != 0 {
		t.Fatalf("ResultSummary calls = %d, want 0", store.calls)
	}
}

func Test_listCheckSummaries(t *testing.T) {
	summary := types.CheckCountSummary{Success: 2, Running: 1}
	store := &summaryCheckStore{
		summary: map[sha.SHA]types.CheckCountSummary{
			sha.Must(summarySHA1): summary,
		},
	}
	controller := &Controller{checkStore: store}

	summaries, err := controller.listCheckSummaries(context.Background(), 7, &SummaryInput{
		CommitSHAs:          []string{summarySHA2, summarySHA1},
		ExcludePayloadKinds: []string{"agent_report"},
	})
	if err != nil {
		t.Fatalf("listCheckSummaries() error = %v", err)
	}
	if store.repoID != 7 || !slices.Equal(store.commitSHAs, []string{summarySHA1, summarySHA2}) {
		t.Fatalf("ResultSummary repoID = %d, SHAs = %v", store.repoID, store.commitSHAs)
	}
	if !slices.Equal(store.excludePayloadKinds, []enum.CheckPayloadKind{"agent_report"}) {
		t.Fatalf("ResultSummary excludePayloadKinds = %v", store.excludePayloadKinds)
	}
	if len(summaries) != 1 || summaries[0].CommitSHA.String() != summarySHA1 || summaries[0].CheckSummary != summary {
		t.Fatalf("listCheckSummaries() = %+v", summaries)
	}
}

func Test_listCheckSummaries_storeError(t *testing.T) {
	storeErr := errors.New("db down")
	controller := &Controller{checkStore: &summaryCheckStore{err: storeErr}}

	_, err := controller.listCheckSummaries(context.Background(), 1, &SummaryInput{
		CommitSHAs: []string{summarySHA1},
	})
	if !errors.Is(err, storeErr) {
		t.Fatalf("listCheckSummaries() error = %v, want %v", err, storeErr)
	}
}
