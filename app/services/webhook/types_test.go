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

package webhook

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/harness/gitness/app/url"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

// mockURLProvider is a simple mock implementation of url.Provider for testing.
type mockURLProvider struct{}

func (m *mockURLProvider) GenerateUIRepoURL(ctx context.Context, repoPath string) string {
	return "http://example.com/repo/" + repoPath
}

func (m *mockURLProvider) GenerateGITCloneURL(ctx context.Context, repoPath string) string {
	return "http://example.com/git/" + repoPath
}

func (m *mockURLProvider) GenerateGITCloneSSHURL(ctx context.Context, repoPath string) string {
	return "git@example.com:" + repoPath
}

func (m *mockURLProvider) GenerateContainerGITCloneURL(_ context.Context, repoPath string) string {
	return "http://container.example.com/git/" + repoPath
}

func (m *mockURLProvider) GetInternalAPIURL(_ context.Context) string { return "" }
func (m *mockURLProvider) GenerateUIPRURL(_ context.Context, _ string, _ int64) string {
	return ""
}
func (m *mockURLProvider) GenerateUICompareURL(_ context.Context, _, _, _ string) string { return "" }
func (m *mockURLProvider) GenerateUIRefURL(_ context.Context, _, _ string) string        { return "" }
func (m *mockURLProvider) GetAPIHostname(_ context.Context) string                       { return "" }
func (m *mockURLProvider) GenerateUIBuildURL(_ context.Context, _, _ string, _ int64) string {
	return ""
}
func (m *mockURLProvider) GetGITHostname(_ context.Context) string                       { return "" }
func (m *mockURLProvider) GetAPIProto(_ context.Context) string                          { return "" }
func (m *mockURLProvider) RegistryURL(_ context.Context, _ ...string) string             { return "" }
func (m *mockURLProvider) PackageURL(_ context.Context, _, _ string, _ ...string) string { return "" }
func (m *mockURLProvider) GetUIBaseURL(_ context.Context, _ ...string) string            { return "" }
func (m *mockURLProvider) GenerateUIRegistryURL(_ context.Context, _, _ string) string   { return "" }
func (m *mockURLProvider) PackagePathFor(_ context.Context, _ url.PackagePathSpec) (string, error) {
	return "", nil
}

func TestRepositoryInfoFrom_WithTags(t *testing.T) {
	ctx := context.Background()
	urlProvider := &mockURLProvider{}

	tags := map[string]string{
		"env":  "production",
		"team": "backend",
		"tier": "critical",
	}

	tagsJSON, err := json.Marshal(tags)
	if err != nil {
		t.Fatalf("failed to marshal tags: %v", err)
	}

	repo := &types.Repository{
		ID:            123,
		Path:          "org/repo",
		Identifier:    "repo",
		Description:   "Test repository",
		DefaultBranch: "main",
		State:         enum.RepoStateActive,
		Tags:          tagsJSON,
	}

	repoInfo := repositoryInfoFrom(ctx, repo, urlProvider)

	// Verify all standard fields
	if repoInfo.ID != repo.ID {
		t.Errorf("Expected ID %d, got %d", repo.ID, repoInfo.ID)
	}
	if repoInfo.Path != repo.Path {
		t.Errorf("Expected Path %s, got %s", repo.Path, repoInfo.Path)
	}
	if repoInfo.Identifier != repo.Identifier {
		t.Errorf("Expected Identifier %s, got %s", repo.Identifier, repoInfo.Identifier)
	}
	if repoInfo.Description != repo.Description {
		t.Errorf("Expected Description %s, got %s", repo.Description, repoInfo.Description)
	}
	if repoInfo.DefaultBranch != repo.DefaultBranch {
		t.Errorf("Expected DefaultBranch %s, got %s", repo.DefaultBranch, repoInfo.DefaultBranch)
	}

	// Verify tags field: Tags is json.RawMessage — unmarshal to compare.
	if repoInfo.Tags == nil {
		t.Fatal("Expected Tags to be populated, got nil")
	}

	var actualTags map[string]string
	if err := json.Unmarshal(repoInfo.Tags, &actualTags); err != nil {
		t.Fatalf("failed to unmarshal repoInfo.Tags: %v", err)
	}

	if len(actualTags) != len(tags) {
		t.Errorf("Expected %d tags, got %d", len(tags), len(actualTags))
	}

	for key, expectedValue := range tags {
		actualValue, exists := actualTags[key]
		if !exists {
			t.Errorf("Expected tag key %s to exist", key)
		}
		if actualValue != expectedValue {
			t.Errorf("Expected tag %s to have value %s, got %s", key, expectedValue, actualValue)
		}
	}
}

func TestRepositoryInfoFrom_WithoutTags(t *testing.T) {
	ctx := context.Background()
	urlProvider := &mockURLProvider{}

	repo := &types.Repository{
		ID:            456,
		Path:          "org/another-repo",
		Identifier:    "another-repo",
		Description:   "Repository without tags",
		DefaultBranch: "master",
		State:         enum.RepoStateActive,
		Tags:          nil, // No tags
	}

	repoInfo := repositoryInfoFrom(ctx, repo, urlProvider)

	// Verify tags field is nil when no tags are present
	if repoInfo.Tags != nil {
		t.Errorf("Expected Tags to be nil for repository without tags, got %v", repoInfo.Tags)
	}
}

func TestRepositoryInfoFrom_WithEmptyTags(t *testing.T) {
	ctx := context.Background()
	urlProvider := &mockURLProvider{}

	emptyTags := map[string]string{}
	tagsJSON, err := json.Marshal(emptyTags)
	if err != nil {
		t.Fatalf("failed to marshal empty tags: %v", err)
	}

	repo := &types.Repository{
		ID:            789,
		Path:          "org/empty-tags-repo",
		Identifier:    "empty-tags-repo",
		Description:   "Repository with empty tags",
		DefaultBranch: "main",
		State:         enum.RepoStateActive,
		Tags:          tagsJSON,
	}

	repoInfo := repositoryInfoFrom(ctx, repo, urlProvider)

	// Verify tags field is an empty JSON object (json.RawMessage `{}`).
	if repoInfo.Tags == nil {
		t.Error("Expected Tags to be populated, got nil")
	}
	var actualTags map[string]string
	if err := json.Unmarshal(repoInfo.Tags, &actualTags); err != nil {
		t.Fatalf("failed to unmarshal repoInfo.Tags: %v", err)
	}
	if len(actualTags) != 0 {
		t.Errorf("Expected Tags to be empty, got %d tags", len(actualTags))
	}
}

func TestRepositoryInfoFrom_NilRepository(t *testing.T) {
	ctx := context.Background()
	urlProvider := &mockURLProvider{}

	repoInfo := repositoryInfoFrom(ctx, nil, urlProvider)

	// Verify empty RepositoryInfo is returned for nil input
	if repoInfo.ID != 0 {
		t.Errorf("Expected empty RepositoryInfo for nil input, got ID %d", repoInfo.ID)
	}
	if repoInfo.Tags != nil {
		t.Errorf("Expected Tags to be nil for nil repository, got %v", repoInfo.Tags)
	}
}

// pullReqInfoJSON marshals the webhook PullReqInfo of the given pull request into a generic map,
// so the tests assert on the wire format that webhook consumers see.
func pullReqInfoJSON(t *testing.T, pr *types.PullReq) map[string]any {
	t.Helper()

	repo := &types.Repository{Path: "space/repo"}
	info := pullReqInfoFrom(context.Background(), pr, repo, &mockURLProvider{})

	raw, err := json.Marshal(info)
	if err != nil {
		t.Fatalf("failed to marshal PullReqInfo: %v", err)
	}

	var decoded map[string]any
	if err = json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("failed to unmarshal PullReqInfo: %v", err)
	}

	return decoded
}

func TestPullReqInfoFrom_MergeSHA(t *testing.T) {
	const mergeSHA = "a27342b1c0ffee00000000000000000000000000"
	mergeSHAValue := mergeSHA
	squash := enum.MergeMethodSquash

	pr := &types.PullReq{
		Number:       1,
		State:        enum.PullReqStateMerged,
		MergeBaseSHA: "856e21b1c0ffee00000000000000000000000000",
		MergeSHA:     &mergeSHAValue,
		MergeMethod:  &squash,
	}

	got, ok := pullReqInfoJSON(t, pr)["merge_sha"]
	if !ok {
		t.Fatal("expected merge_sha to be present in the payload of a merged pull request")
	}
	if got != mergeSHA {
		t.Errorf("expected merge_sha %q, got %v", mergeSHA, got)
	}
}

func TestPullReqInfoFrom_NoMergeSHA(t *testing.T) {
	// Open pull requests never have a merge commit; the key must be omitted, not sent as an empty string.
	pr := &types.PullReq{
		Number:       2,
		State:        enum.PullReqStateOpen,
		MergeBaseSHA: "856e21b1c0ffee00000000000000000000000000",
		MergeSHA:     nil,
	}

	if got, ok := pullReqInfoJSON(t, pr)["merge_sha"]; ok {
		t.Errorf("expected merge_sha to be omitted for a pull request without merge commit, got %v", got)
	}
}

func TestPullReqInfoForMerged_MergeSHA(t *testing.T) {
	const (
		fromPR    = "a27342b1c0ffee00000000000000000000000000"
		fromEvent = "b38453c2d1ffee00000000000000000000000000"
	)
	prSHA, emptySHA := fromPR, ""

	tests := []struct {
		name    string
		prSHA   *string
		eventID string
		want    string
	}{
		{name: "merge sha of the pull request wins over the event", prSHA: &prSHA, eventID: fromEvent, want: fromPR},
		{name: "event merge sha fills in when the pull request has none", prSHA: nil, eventID: fromEvent, want: fromEvent},
		{name: "empty pull request merge sha falls back to the event", prSHA: &emptySHA, eventID: fromEvent, want: fromEvent},
		{name: "stays empty when neither side has one", prSHA: nil, eventID: "", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pr := &types.PullReq{Number: 1, State: enum.PullReqStateMerged, MergeSHA: tt.prSHA}
			repo := &types.Repository{Path: "space/repo"}

			got := pullReqInfoForMerged(context.Background(), pr, repo, &mockURLProvider{}, tt.eventID)

			if got.MergeSHA != tt.want {
				t.Errorf("expected MergeSHA %q, got %q", tt.want, got.MergeSHA)
			}
			if got.Number != pr.Number || got.State != enum.PullReqStateMerged {
				t.Errorf("expected the rest of the pull request info to be preserved, got %+v", got)
			}
		})
	}
}
