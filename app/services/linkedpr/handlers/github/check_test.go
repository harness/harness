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

package github

import (
	"context"
	"errors"
	"testing"
	"time"

	checkevents "github.com/harness/gitness/app/events/check"
	"github.com/harness/gitness/app/services/linkedpr"
	"github.com/harness/gitness/git/sha"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

func TestMapCheckStatus(t *testing.T) {
	tests := []struct {
		status     string
		conclusion string
		want       enum.CheckStatus
	}{
		// --- Non-terminal states (no conclusion) ---
		// Raw GitHub Checks API status values:
		{"queued", "", enum.CheckStatusPending},
		{"in_progress", "", enum.CheckStatusRunning},
		// go-scm normalized status values:
		{"pending", "", enum.CheckStatusPending},
		{"running", "", enum.CheckStatusRunning},

		// --- Terminal states via conclusion (GitHub Checks API) ---
		// go-scm passes the raw GitHub conclusion string as proto.conclusion.
		// The proto.status may be "Unknown" for some conclusions, so
		// conclusion must be checked first.
		{"success", "success", enum.CheckStatusSuccess},
		{"Unknown", "neutral", enum.CheckStatusSuccess},
		{"Unknown", "skipped", enum.CheckStatusSuccess},
		{"failed", "failure", enum.CheckStatusFailure}, // primary bug case
		{"Unknown", "timed_out", enum.CheckStatusFailure},
		{"Unknown", "action_required", enum.CheckStatusFailure},
		{"canceled", "cancelled", enum.CheckStatusFailure},
		{"Unknown", "startup_failure", enum.CheckStatusError},
		// Older test vectors that used raw GitHub status="completed" still
		// pass because conclusion takes precedence:
		{"completed", "success", enum.CheckStatusSuccess},
		{"completed", "neutral", enum.CheckStatusSuccess},
		{"completed", "skipped", enum.CheckStatusSuccess},
		{"completed", "failure", enum.CheckStatusFailure},
		{"completed", "timed_out", enum.CheckStatusFailure},
		{"completed", "action_required", enum.CheckStatusFailure},
		{"completed", "cancelled", enum.CheckStatusFailure},
		{"completed", "startup_failure", enum.CheckStatusError},

		// --- GitHub Statuses API (legacy) ---
		// convertStatusHook sets both Status and Conclusion to the raw state,
		// so conclusion is always non-empty and handled by the conclusion branch.
		{"success", "success", enum.CheckStatusSuccess},
		{"failed", "failure", enum.CheckStatusFailure},
		{"Unknown", "error", enum.CheckStatusError},
		{"pending", "pending", enum.CheckStatusPending},

		// --- Unknown / safe default (no conclusion, non-terminal status) ---
		{"unknown_state", "", enum.CheckStatusPending},
		{"Unknown", "", enum.CheckStatusPending},

		// --- stale: required check cleared by maintainer → pending ---
		{"Unknown", "stale", enum.CheckStatusPending},
		{"completed", "stale", enum.CheckStatusPending},
	}

	for _, tt := range tests {
		got := mapCheckStatus(tt.status, tt.conclusion)
		if got != tt.want {
			t.Errorf("mapCheckStatus(%q, %q) = %v; want %v",
				tt.status, tt.conclusion, got, tt.want)
		}
	}
}

func TestParseTimeMillis(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  int64
	}{
		{"empty string", "", 0},
		{"garbage", "not-a-time", 0},
		{"valid RFC3339", "2024-01-15T10:30:00Z", time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC).UnixMilli()},
		{"valid RFC3339 with offset", "2024-01-15T15:30:00+05:30",
			time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC).UnixMilli()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseTimeMillis(tt.input)
			if got != tt.want {
				t.Errorf("parseTimeMillis(%q) = %d; want %d", tt.input, got, tt.want)
			}
		})
	}
}

// ─── stubs ────────────────────────────────────────────────────────────────────

// stubAuthorResolver returns a fixed principal ID or error.
type stubAuthorResolver struct {
	principalID int64
	err         error
}

func (s *stubAuthorResolver) Resolve(_ context.Context, _ *linkedpr.Event) (int64, error) {
	return s.principalID, s.err
}

// stubCheckStore implements store.CheckStore; only Upsert is wired to a
// configurable function — all other methods return zero values.
type stubCheckStore struct {
	upsertFn func(*types.Check) error
	upserted []*types.Check
}

func (s *stubCheckStore) Upsert(_ context.Context, check *types.Check) error {
	s.upserted = append(s.upserted, check)
	if s.upsertFn != nil {
		return s.upsertFn(check)
	}
	return nil
}

func (s *stubCheckStore) FindByIdentifier(
	_ context.Context, _ int64, _ string, _ string,
) (types.Check, error) {
	return types.Check{}, nil
}

func (s *stubCheckStore) Count(
	_ context.Context, _ int64, _ string, _ types.CheckListOptions,
) (int, error) {
	return 0, nil
}

func (s *stubCheckStore) List(
	_ context.Context, _ int64, _ string, _ types.CheckListOptions,
) ([]types.Check, error) {
	return []types.Check{}, nil
}

func (s *stubCheckStore) ListRecent(
	_ context.Context, _ int64, _ types.CheckRecentOptions,
) ([]string, error) {
	return []string{}, nil
}

func (s *stubCheckStore) ListRecentSpace(
	_ context.Context, _ []int64, _ types.CheckRecentOptions,
) ([]string, error) {
	return []string{}, nil
}

func (s *stubCheckStore) ListResults(
	_ context.Context, _ int64, _ string,
) ([]types.CheckResult, error) {
	return []types.CheckResult{}, nil
}

func (s *stubCheckStore) ResultSummary(
	_ context.Context, _ int64, _ []string,
) (map[sha.SHA]types.CheckCountSummary, error) {
	return map[sha.SHA]types.CheckCountSummary{}, nil
}

// ─── helpers ──────────────────────────────────────────────────────────────────

// errStopAfterUpsert is a sentinel returned by upsertFn in happy-path tests
// to stop Handle before it reaches eventReporter.Reported (no live event bus
// in unit tests). Tests verify the captured *types.Check before checking for
// this specific error.
var errStopAfterUpsert = errors.New("stop-after-upsert sentinel")

func newTestHandler(cs *stubCheckStore, ar *stubAuthorResolver) *CheckHandler {
	// eventReporter is always non-nil in production (wired by Wire). Tests
	// that exercise paths reaching Reported must arrange to stop beforehand
	// (see errStopAfterUpsert pattern in TestCheckHandler_Handle_HappyPath).
	return NewCheckHandler(cs, &checkevents.Reporter{}, ar)
}

func testLinkedRepo(repoID int64) *types.LinkedRepo {
	return &types.LinkedRepo{RepoID: repoID}
}

func testEvent() *linkedpr.Event {
	return &linkedpr.Event{
		Provider:  linkedpr.ProviderGitHub,
		AccountID: "acc-test",
	}
}

func validEntry() linkedpr.CheckEntry {
	return linkedpr.CheckEntry{
		Identifier: "ci/build",
		Status:     "completed",
		Conclusion: "success",
		Link:       "https://github.com/acme/repo/runs/42",
		SHA:        "abc1234abc1234abc1234abc1234abc1234abc12",
		Started:    "2024-01-15T10:00:00Z",
		Completed:  "2024-01-15T10:05:00Z",
	}
}

// ─── tests ────────────────────────────────────────────────────────────────────

func TestCheckHandler_Handle_AuthorResolverError(t *testing.T) {
	resolveErr := errors.New("resolver boom")
	cs := &stubCheckStore{}
	h := newTestHandler(cs, &stubAuthorResolver{err: resolveErr})

	payload := linkedpr.CheckPayload{
		Checks: []linkedpr.CheckEntry{validEntry()},
	}
	err := h.Handle(context.Background(), testEvent(), payload, testLinkedRepo(1))

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, resolveErr) {
		t.Errorf("expected wrapped resolveErr, got: %v", err)
	}
	if len(cs.upserted) != 0 {
		t.Errorf("expected no upserts on resolver error, got %d", len(cs.upserted))
	}
}

func TestCheckHandler_Handle_SkipEmptySHA(t *testing.T) {
	cs := &stubCheckStore{}
	h := newTestHandler(cs, &stubAuthorResolver{principalID: 1})

	entry := validEntry()
	entry.SHA = ""
	payload := linkedpr.CheckPayload{Checks: []linkedpr.CheckEntry{entry}}

	if err := h.Handle(context.Background(), testEvent(), payload, testLinkedRepo(1)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cs.upserted) != 0 {
		t.Errorf("expected skip on empty SHA, got %d upserts", len(cs.upserted))
	}
}

func TestCheckHandler_Handle_SkipEmptyIdentifier(t *testing.T) {
	cs := &stubCheckStore{}
	h := newTestHandler(cs, &stubAuthorResolver{principalID: 1})

	entry := validEntry()
	entry.Identifier = ""
	payload := linkedpr.CheckPayload{Checks: []linkedpr.CheckEntry{entry}}

	if err := h.Handle(context.Background(), testEvent(), payload, testLinkedRepo(1)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cs.upserted) != 0 {
		t.Errorf("expected skip on empty Identifier, got %d upserts", len(cs.upserted))
	}
}

func TestCheckHandler_Handle_SkipEmptyLink(t *testing.T) {
	cs := &stubCheckStore{}
	h := newTestHandler(cs, &stubAuthorResolver{principalID: 1})

	entry := validEntry()
	entry.Link = ""
	payload := linkedpr.CheckPayload{Checks: []linkedpr.CheckEntry{entry}}

	if err := h.Handle(context.Background(), testEvent(), payload, testLinkedRepo(1)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cs.upserted) != 0 {
		t.Errorf("expected skip on empty Link, got %d upserts", len(cs.upserted))
	}
}

func TestCheckHandler_Handle_UpsertError(t *testing.T) {
	upsertErr := errors.New("db boom")
	cs := &stubCheckStore{upsertFn: func(_ *types.Check) error { return upsertErr }}
	h := newTestHandler(cs, &stubAuthorResolver{principalID: 1})

	payload := linkedpr.CheckPayload{Checks: []linkedpr.CheckEntry{validEntry()}}
	err := h.Handle(context.Background(), testEvent(), payload, testLinkedRepo(1))

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, upsertErr) {
		t.Errorf("expected wrapped upsertErr, got: %v", err)
	}
}

func TestCheckHandler_Handle_HappyPath(t *testing.T) {
	const repoID = int64(99)
	const principalID = int64(7)

	var captured *types.Check
	cs := &stubCheckStore{
		// Return the sentinel after capturing so Handle stops before reaching
		// eventReporter.Reported (no live event bus in unit tests).
		upsertFn: func(c *types.Check) error {
			captured = c
			return errStopAfterUpsert
		},
	}
	h := newTestHandler(cs, &stubAuthorResolver{principalID: principalID})

	entry := validEntry()
	payload := linkedpr.CheckPayload{Checks: []linkedpr.CheckEntry{entry}}

	err := h.Handle(context.Background(), testEvent(), payload, testLinkedRepo(repoID))
	if !errors.Is(err, errStopAfterUpsert) {
		t.Fatalf("expected errStopAfterUpsert, got: %v", err)
	}
	if captured == nil {
		t.Fatal("upsert was never called")
	}

	checkField := func(name string, got, want interface{}) {
		t.Helper()
		if got != want {
			t.Errorf("Check.%s = %v; want %v", name, got, want)
		}
	}
	checkField("RepoID", captured.RepoID, repoID)
	checkField("CommitSHA", captured.CommitSHA, entry.SHA)
	checkField("Identifier", captured.Identifier, entry.Identifier)
	checkField("Status", captured.Status, enum.CheckStatusSuccess)
	checkField("Link", captured.Link, entry.Link)
	checkField("CreatedBy", captured.CreatedBy, principalID)
	checkField("Payload.Kind", captured.Payload.Kind, enum.CheckPayloadKindEmpty)
	checkField("Started", captured.Started, parseTimeMillis(entry.Started))
	checkField("Ended", captured.Ended, parseTimeMillis(entry.Completed))
}

func TestCheckHandler_Handle_PartialSkip(t *testing.T) {
	cs := &stubCheckStore{upsertFn: func(_ *types.Check) error { return errStopAfterUpsert }}
	h := newTestHandler(cs, &stubAuthorResolver{principalID: 1})

	entries := []linkedpr.CheckEntry{
		{Identifier: "skip-no-sha", Link: "https://x", SHA: ""},
		validEntry(), // only this one should be upserted
		{Identifier: "", SHA: "abc", Link: "https://x"},
		{Identifier: "skip-no-link", SHA: "abc", Link: ""},
	}
	payload := linkedpr.CheckPayload{Checks: entries}

	err := h.Handle(context.Background(), testEvent(), payload, testLinkedRepo(1))
	if !errors.Is(err, errStopAfterUpsert) {
		t.Fatalf("expected errStopAfterUpsert after the one valid upsert, got: %v", err)
	}
	if len(cs.upserted) != 1 {
		t.Errorf("expected 1 upsert (only valid entry), got %d", len(cs.upserted))
	}
}
