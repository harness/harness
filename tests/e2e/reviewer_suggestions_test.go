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

package e2e

// E2E coverage for CODE-6883 (reviewer suggestion schema: reason, kind, rank, user group,
// commit sha) against a running server.
//
// Enable with GITNESS_E2E_TEST_ENABLED=true. Uses GITNESS_BASE_URL (default http://localhost:3000),
// and GITNESS_AUTH_TOKEN or GITNESS_PRINCIPAL_ADMIN_EMAIL/PASSWORD. Each test creates and removes its own space.
//
// The caller in the main tests is an admin *user*, so kind/reason can't be set. The service-account
// path (kind/reason accepted, user can't overwrite) needs a service principal that has repo_review
// on the test repo; the open source membership model can't grant that, so it runs only when
// GITNESS_E2E_SERVICE_TOKEN is provided.

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type e2eClient struct {
	t       *testing.T
	baseURL string
	token   string
}

// do sends a JSON request and returns status and raw body.
func (c *e2eClient) do(method, path string, body any) (int, []byte) {
	c.t.Helper()
	return c.doWithToken(c.token, method, path, body)
}

func (c *e2eClient) doWithToken(token, method, path string, body any) (int, []byte) {
	c.t.Helper()

	var reader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		require.NoError(c.t, err)
		reader = bytes.NewReader(payload)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+"/api/v1"+path, reader)
	require.NoError(c.t, err)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(c.t, err)
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	require.NoError(c.t, err)

	return resp.StatusCode, data
}

// mustDo asserts the expected status and decodes the response into out (if not nil).
func (c *e2eClient) mustDo(method, path string, body any, wantStatus int, out any) {
	c.t.Helper()
	status, data := c.do(method, path, body)
	require.Equal(c.t, wantStatus, status, "%s %s: %s", method, path, data)
	if out != nil {
		require.NoError(c.t, json.Unmarshal(data, out), string(data))
	}
}

type e2eSuggestion struct {
	PrincipalID int64  `json:"principal_id"`
	Kind        string `json:"kind,omitempty"`
	Reason      string `json:"reason,omitempty"`
	Rank        *int64 `json:"rank,omitempty"`
	UserGroupID *int64 `json:"user_group_id,omitempty"`
	CommitSHA   string `json:"commit_sha,omitempty"`
}

type e2eBatchResponse struct {
	Results []struct {
		PrincipalID int64  `json:"principal_id"`
		Status      string `json:"status"`
		Error       string `json:"error"`
	} `json:"results"`
}

type e2eListResponse struct {
	Suggestions []struct {
		Reviewer struct {
			ID int64 `json:"id"`
		} `json:"reviewer"`
		SuggestedBy struct {
			ID int64 `json:"id"`
		} `json:"suggested_by"`
		Kind      string `json:"kind"`
		Reason    string `json:"reason"`
		Rank      *int64 `json:"rank"`
		CommitSHA string `json:"commit_sha"`
		UserGroup *struct {
			ID int64 `json:"id"`
		} `json:"user_group"`
	} `json:"suggestions"`
}

type e2eFixture struct {
	client    *e2eClient
	prPath    string // /repos/{space}/{repo}/pullreq/{n}
	headSHA   string
	adminID   int64
	reviewers []int64 // contributors, can review
	outsider  int64   // user without access to the space
}

func newE2EClient(t *testing.T) *e2eClient {
	t.Helper()
	if !isE2ETestsEnabled() {
		t.Skip("E2E testing is disabled")
	}

	baseURL := getEnv("GITNESS_BASE_URL", "http://localhost:3000")
	token := os.Getenv("GITNESS_AUTH_TOKEN")
	if token == "" {
		var err error
		token, err = authenticate(t.Context(), baseURL)
		require.NoError(t, err, "authenticate")
	}

	return &e2eClient{t: t, baseURL: baseURL, token: token}
}

// setupSuggestionFixture creates a repo, a feature branch with a commit, a pull request authored
// by the admin, three space contributors and one user without any access. Everything is removed
// on cleanup.
func setupSuggestionFixture(t *testing.T) *e2eFixture {
	t.Helper()
	c := newE2EClient(t)

	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	spaceRef := "e2e-suggest-space-" + suffix
	c.mustDo(http.MethodPost, "/spaces", map[string]any{"identifier": spaceRef}, http.StatusCreated, nil)
	t.Cleanup(func() { c.do(http.MethodDelete, "/spaces/"+spaceRef, nil) })
	repoID := "e2e-suggest-" + suffix
	repoRef := spaceRef + "/" + repoID

	var self struct {
		ID int64 `json:"id"`
	}
	c.mustDo(http.MethodGet, "/user", nil, http.StatusOK, &self)

	c.mustDo(http.MethodPost, "/repos", map[string]any{
		"parent_ref": spaceRef, "identifier": repoID, "default_branch": "main", "readme": true,
	}, http.StatusCreated, nil)
	t.Cleanup(func() { c.do(http.MethodDelete, "/repos/"+repoRef, nil) })

	c.mustDo(http.MethodPost, "/repos/"+repoRef+"/+/commits", map[string]any{
		"title": "feature", "branch": "main", "new_branch": "feature",
		"actions": []map[string]any{{
			"action": "CREATE", "path": "feature.txt", "payload": "hello", "encoding": "utf8",
		}},
	}, http.StatusOK, nil)

	var pr struct {
		Number    int64  `json:"number"`
		SourceSHA string `json:"source_sha"`
	}
	c.mustDo(http.MethodPost, "/repos/"+repoRef+"/+/pullreq", map[string]any{
		"title": "e2e reviewer suggestions", "source_branch": "feature", "target_branch": "main",
	}, http.StatusCreated, &pr)

	newUser := func(role string) int64 {
		uid := fmt.Sprintf("e2e-%s-%s", role, suffix)
		var u struct {
			ID int64 `json:"id"`
		}
		c.mustDo(http.MethodPost, "/admin/users", map[string]any{
			"uid": uid, "email": uid + "@example.com", "display_name": uid, "password": "Passw0rd!e2e-" + suffix,
		}, http.StatusCreated, &u)
		t.Cleanup(func() { c.do(http.MethodDelete, "/admin/users/"+uid, nil) })

		if role != "outsider" {
			c.mustDo(http.MethodPost, "/spaces/"+spaceRef+"/members", map[string]any{
				"user_uid": uid, "role": "contributor",
			}, http.StatusCreated, nil)
			t.Cleanup(func() { c.do(http.MethodDelete, "/spaces/"+spaceRef+"/members/"+uid, nil) })
		}
		return u.ID
	}

	return &e2eFixture{
		client:    c,
		prPath:    fmt.Sprintf("/repos/%s/+/pullreq/%d", repoRef, pr.Number),
		headSHA:   pr.SourceSHA,
		adminID:   self.ID,
		reviewers: []int64{newUser("a"), newUser("b"), newUser("c")},
		outsider:  newUser("outsider"),
	}
}

func (f *e2eFixture) batch(body any, wantStatus int) (*e2eBatchResponse, []byte) {
	f.client.t.Helper()
	status, data := f.client.do(http.MethodPost, f.prPath+"/suggestions/reviewers/batch", body)
	require.Equal(f.client.t, wantStatus, status, string(data))
	if wantStatus != http.StatusOK || len(data) == 0 {
		return nil, data
	}
	var out e2eBatchResponse
	require.NoError(f.client.t, json.Unmarshal(data, &out), string(data))
	return &out, data
}

func (f *e2eFixture) list() *e2eListResponse {
	f.client.t.Helper()
	var out e2eListResponse
	f.client.mustDo(http.MethodGet, f.prPath+"/suggestions/reviewers", nil, http.StatusOK, &out)
	return &out
}

// Legacy reviewer_ids form must stay byte-compatible: empty 200, and rows come back as optional.
func TestE2EReviewerSuggest_LegacyReviewerIDs(t *testing.T) {
	f := setupSuggestionFixture(t)

	out, raw := f.batch(map[string]any{"reviewer_ids": []int64{f.reviewers[0], f.reviewers[1]}}, http.StatusOK)
	assert.Nil(t, out)
	assert.Empty(t, strings.TrimSpace(string(raw)), "legacy form returns an empty body")

	list := f.list()
	require.Len(t, list.Suggestions, 2)
	for _, s := range list.Suggestions {
		assert.Equal(t, "optional", s.Kind)
		assert.Empty(t, s.Reason)
		assert.Nil(t, s.Rank)
		assert.Nil(t, s.UserGroup)
		assert.Equal(t, f.adminID, s.SuggestedBy.ID)
	}

	// legacy: author can't be suggested, whole request fails
	_, _ = f.batch(map[string]any{"reviewer_ids": []int64{f.reviewers[2], f.adminID}}, http.StatusBadRequest)
	// legacy: unknown principal is a 404 for the whole request
	_, _ = f.batch(map[string]any{"reviewer_ids": []int64{999999999}}, http.StatusNotFound)
}

// One bad row must not discard the good ones; every row gets its own result.
func TestE2EReviewerSuggest_PerRowResultsAndList(t *testing.T) {
	f := setupSuggestionFixture(t)
	a, b, c := f.reviewers[0], f.reviewers[1], f.reviewers[2]

	out, _ := f.batch(map[string]any{"suggestions": []e2eSuggestion{
		{PrincipalID: a, Rank: new(int64(2)), CommitSHA: f.headSHA},
		{PrincipalID: b, Rank: new(int64(1))},
		{PrincipalID: c}, // no rank: must sort last
		{PrincipalID: f.adminID},
		{PrincipalID: f.outsider},
		{PrincipalID: 999999999},
		{PrincipalID: b, Rank: new(int64(1))}, // duplicate collapses (last wins)
	}}, http.StatusOK)

	status := map[int64]string{}
	errs := map[int64]string{}
	for _, r := range out.Results {
		status[r.PrincipalID] = r.Status
		errs[r.PrincipalID] = r.Error
	}
	assert.Equal(t, "created", status[a])
	assert.Equal(t, "created", status[b])
	assert.Equal(t, "created", status[c])
	assert.Equal(t, "rejected", status[f.adminID])
	assert.Contains(t, errs[f.adminID], "author")
	assert.Equal(t, "rejected", status[f.outsider])
	assert.Contains(t, errs[f.outsider], "permission")
	assert.Equal(t, "rejected", status[999999999])
	assert.Contains(t, errs[999999999], "could not be found")
	assert.Len(t, out.Results, 6, "duplicate principal collapses into one row")

	// order: rank asc (b=1, a=2), nulls last (c)
	list := f.list()
	require.Len(t, list.Suggestions, 3)
	assert.Equal(t, []int64{b, a, c}, []int64{
		list.Suggestions[0].Reviewer.ID, list.Suggestions[1].Reviewer.ID, list.Suggestions[2].Reviewer.ID,
	})
	assert.Equal(t, f.headSHA, list.Suggestions[1].CommitSHA)
	assert.Nil(t, list.Suggestions[2].Rank)
	for _, s := range list.Suggestions {
		assert.Equal(t, "optional", s.Kind)
	}

	// re-suggesting an existing principal updates the row in place
	out, _ = f.batch(map[string]any{"suggestions": []e2eSuggestion{
		{PrincipalID: a, Rank: new(int64(9))},
	}}, http.StatusOK)
	require.Len(t, out.Results, 1)
	assert.Equal(t, "updated", out.Results[0].Status)
	assert.Len(t, f.list().Suggestions, 3)
}

// A user (non-service) caller can't set kind or reason: they are ignored, so a user can't
// self-declare a reviewer as required.
func TestE2EReviewerSuggest_UserCannotSetKindOrReason(t *testing.T) {
	f := setupSuggestionFixture(t)

	out, _ := f.batch(map[string]any{"suggestions": []e2eSuggestion{
		{PrincipalID: f.reviewers[0], Kind: "required", Reason: "owns the module"},
	}}, http.StatusOK)
	require.Len(t, out.Results, 1)
	assert.Equal(t, "created", out.Results[0].Status)

	list := f.list()
	require.Len(t, list.Suggestions, 1)
	assert.Equal(t, "optional", list.Suggestions[0].Kind)
	assert.Empty(t, list.Suggestions[0].Reason)
}

func TestE2EReviewerSuggest_UserGroupValidation(t *testing.T) {
	f := setupSuggestionFixture(t)
	a, b := f.reviewers[0], f.reviewers[1]

	out, _ := f.batch(map[string]any{"suggestions": []e2eSuggestion{
		{PrincipalID: a, UserGroupID: new(int64(987654321))}, // unknown group
		{PrincipalID: b},
	}}, http.StatusOK)

	got := map[int64]string{}
	for _, r := range out.Results {
		got[r.PrincipalID] = r.Status
		if r.PrincipalID == a {
			assert.Contains(t, r.Error, "user group")
		}
	}
	assert.Equal(t, "rejected", got[a])
	assert.Equal(t, "created", got[b], "valid row is accepted next to the rejected one")

	list := f.list()
	require.Len(t, list.Suggestions, 1)
	assert.Equal(t, b, list.Suggestions[0].Reviewer.ID)
	assert.Nil(t, list.Suggestions[0].UserGroup)
}

func TestE2EReviewerSuggest_InputValidation(t *testing.T) {
	f := setupSuggestionFixture(t)
	a := f.reviewers[0]

	cases := map[string]any{
		"both forms":      map[string]any{"reviewer_ids": []int64{a}, "suggestions": []e2eSuggestion{{PrincipalID: a}}},
		"neither form":    map[string]any{},
		"invalid kind":    map[string]any{"suggestions": []e2eSuggestion{{PrincipalID: a, Kind: "mandatory"}}},
		"wrong-case kind": map[string]any{"suggestions": []e2eSuggestion{{PrincipalID: a, Kind: "REQUIRED"}}},
		"reason too long": map[string]any{"suggestions": []e2eSuggestion{{PrincipalID: a, Reason: strings.Repeat("x", 501)}}},
		"negative rank":   map[string]any{"suggestions": []e2eSuggestion{{PrincipalID: a, Rank: new(int64(-1))}}},
		"zero principal":  map[string]any{"suggestions": []e2eSuggestion{{PrincipalID: 0}}},
		"commit sha long": map[string]any{
			"suggestions": []e2eSuggestion{{PrincipalID: a, CommitSHA: strings.Repeat("a", 65)}},
		},
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			f.batch(body, http.StatusBadRequest)
		})
	}

	// 51 rows exceeds the per-request/PR cap
	tooMany := make([]e2eSuggestion, 51)
	for i := range tooMany {
		tooMany[i] = e2eSuggestion{PrincipalID: int64(i + 1)}
	}
	f.batch(map[string]any{"suggestions": tooMany}, http.StatusBadRequest)

	// boundary: a 500 character reason (multi-byte counts as characters) passes validation
	f.batch(map[string]any{
		"suggestions": []e2eSuggestion{{PrincipalID: a, Reason: strings.Repeat("é", 500)}},
	}, http.StatusOK)

	assert.Len(t, f.list().Suggestions, 1, "rejected requests wrote nothing")
}

func TestE2EReviewerSuggest_DeleteAndRBAC(t *testing.T) {
	f := setupSuggestionFixture(t)
	a, b := f.reviewers[0], f.reviewers[1]

	f.batch(map[string]any{"suggestions": []e2eSuggestion{{PrincipalID: a}, {PrincipalID: b}}}, http.StatusOK)
	require.Len(t, f.list().Suggestions, 2)

	f.client.mustDo(http.MethodDelete, fmt.Sprintf("%s/suggestions/reviewers/%d", f.prPath, a), nil,
		http.StatusNoContent, nil)
	list := f.list()
	require.Len(t, list.Suggestions, 1)
	assert.Equal(t, b, list.Suggestions[0].Reviewer.ID)

	// unauthenticated token: no access to list or batch
	status, _ := f.client.doWithToken("invalid", http.MethodGet, f.prPath+"/suggestions/reviewers", nil)
	assert.Equal(t, http.StatusUnauthorized, status)
	status, _ = f.client.doWithToken("invalid", http.MethodPost, f.prPath+"/suggestions/reviewers/batch",
		map[string]any{"suggestions": []e2eSuggestion{{PrincipalID: a}}})
	assert.Equal(t, http.StatusUnauthorized, status)
}

// Service path: kind and reason are stored and returned, and a user can't overwrite the row.
// Requires a service principal token with repo_review on the test space.
func TestE2EReviewerSuggest_ServicePrincipal(t *testing.T) {
	serviceToken := os.Getenv("GITNESS_E2E_SERVICE_TOKEN")
	if serviceToken == "" {
		t.Skip("GITNESS_E2E_SERVICE_TOKEN not set")
	}
	f := setupSuggestionFixture(t)
	a, b := f.reviewers[0], f.reviewers[1]

	status, data := f.client.doWithToken(serviceToken, http.MethodPost, f.prPath+"/suggestions/reviewers/batch",
		map[string]any{"suggestions": []e2eSuggestion{
			{PrincipalID: a, Kind: "required", Reason: "owns payments/", Rank: new(int64(1)), CommitSHA: f.headSHA},
			{PrincipalID: b, Kind: "not_needed", Reason: "just a reformat"},
		}})
	require.Equal(t, http.StatusOK, status, string(data))

	list := f.list()
	require.Len(t, list.Suggestions, 2)
	assert.Equal(t, "required", list.Suggestions[0].Kind)
	assert.Equal(t, "owns payments/", list.Suggestions[0].Reason)
	assert.Equal(t, f.headSHA, list.Suggestions[0].CommitSHA)
	assert.Equal(t, "not_needed", list.Suggestions[1].Kind)

	// a user cannot overwrite a service-created row
	out, _ := f.batch(map[string]any{"suggestions": []e2eSuggestion{{PrincipalID: a, Kind: "optional"}}}, http.StatusOK)
	require.Len(t, out.Results, 1)
	assert.Equal(t, "rejected", out.Results[0].Status)
	assert.Contains(t, out.Results[0].Error, "service account")
	assert.Equal(t, "required", f.list().Suggestions[0].Kind)
}

// Corner cases around existing reviewers and numeric/length boundaries.
func TestE2EReviewerSuggest_CornerCases(t *testing.T) {
	f := setupSuggestionFixture(t)
	a, b, c := f.reviewers[0], f.reviewers[1], f.reviewers[2]

	t.Run("already a reviewer is skipped", func(t *testing.T) {
		f.client.mustDo(http.MethodPut, f.prPath+"/reviewers", map[string]any{"reviewer_id": a},
			http.StatusOK, nil)

		out, _ := f.batch(map[string]any{"suggestions": []e2eSuggestion{{PrincipalID: a}}}, http.StatusOK)
		require.Len(t, out.Results, 1)
		assert.Equal(t, "skipped", out.Results[0].Status)
		assert.Empty(t, f.list().Suggestions)
	})

	t.Run("rank and commit_sha boundaries are accepted", func(t *testing.T) {
		out, _ := f.batch(map[string]any{"suggestions": []e2eSuggestion{
			{PrincipalID: b, Rank: new(int64(0)), CommitSHA: strings.Repeat("a", 64)},
			{PrincipalID: c, Rank: new(int64(2147483647))},
		}}, http.StatusOK)
		for _, r := range out.Results {
			assert.Equal(t, "created", r.Status, r.Error)
		}

		list := f.list()
		require.Len(t, list.Suggestions, 2)
		assert.Equal(t, b, list.Suggestions[0].Reviewer.ID, "rank 0 sorts first")
		require.NotNil(t, list.Suggestions[0].Rank, "rank 0 is a value, not a missing rank")
		assert.Equal(t, int64(0), *list.Suggestions[0].Rank)
		assert.Len(t, list.Suggestions[0].CommitSHA, 64)
		assert.Equal(t, c, list.Suggestions[1].Reviewer.ID)
	})

	t.Run("rank above int32 is rejected", func(t *testing.T) {
		f.batch(map[string]any{"suggestions": []e2eSuggestion{
			{PrincipalID: b, Rank: new(int64(2147483648))},
		}}, http.StatusBadRequest)
	})

	t.Run("refresh can clear rank and commit sha", func(t *testing.T) {
		f.batch(map[string]any{"suggestions": []e2eSuggestion{{PrincipalID: b}}}, http.StatusOK)
		for _, s := range f.list().Suggestions {
			if s.Reviewer.ID == b {
				assert.Nil(t, s.Rank)
				assert.Empty(t, s.CommitSHA)
			}
		}
	})
}

// Concurrent identical batches must not 500 or duplicate rows.
func TestE2EReviewerSuggest_ConcurrentBatches(t *testing.T) {
	f := setupSuggestionFixture(t)

	body := map[string]any{"suggestions": []e2eSuggestion{
		{PrincipalID: f.reviewers[0]}, {PrincipalID: f.reviewers[1]}, {PrincipalID: f.reviewers[2]},
	}}

	const workers = 8
	statuses := make(chan int, workers)
	for range workers {
		go func() {
			status, _ := f.client.do(http.MethodPost, f.prPath+"/suggestions/reviewers/batch", body)
			statuses <- status
		}()
	}
	for range workers {
		assert.Equal(t, http.StatusOK, <-statuses)
	}

	assert.Len(t, f.list().Suggestions, 3, "no duplicate rows after concurrent upserts")
}

// Applying a suggestion adds the reviewer; the suggestion list stays consistent afterwards.
func TestE2EReviewerSuggest_Apply(t *testing.T) {
	f := setupSuggestionFixture(t)
	a, b := f.reviewers[0], f.reviewers[1]

	f.batch(map[string]any{"suggestions": []e2eSuggestion{
		{PrincipalID: a, Rank: new(int64(1)), Reason: "ignored for users"}, {PrincipalID: b},
	}}, http.StatusOK)

	status, data := f.client.do(http.MethodPost, fmt.Sprintf("%s/suggestions/reviewers/%d/apply", f.prPath, a), nil)
	require.Contains(t, []int{http.StatusOK, http.StatusCreated}, status, string(data))

	var reviewers []struct {
		Reviewer struct {
			ID int64 `json:"id"`
		} `json:"reviewer"`
	}
	f.client.mustDo(http.MethodGet, f.prPath+"/reviewers", nil, http.StatusOK, &reviewers)
	var ids []int64
	for _, r := range reviewers {
		ids = append(ids, r.Reviewer.ID)
	}
	assert.Contains(t, ids, a)

	// re-suggesting someone who is now a reviewer is skipped
	out, _ := f.batch(map[string]any{"suggestions": []e2eSuggestion{{PrincipalID: a}}}, http.StatusOK)
	assert.Equal(t, "skipped", out.Results[0].Status)
}
