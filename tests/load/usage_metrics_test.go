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

package load

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	repoctl "github.com/harness/gitness/app/api/controller/repo"
	"github.com/harness/gitness/app/services/importer"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	vegeta "github.com/tsenart/vegeta/v12/lib"
)

func TestUsageMetricsLoad(t *testing.T) {
	if !isE2ETestsEnabled() {
		t.Skip("E2E testing is disabled")
	}
	baseURL := getEnv("GITNESS_BASE_URL", "http://localhost:3000")
	authToken := os.Getenv("GITNESS_AUTH_TOKEN")

	// Authenticate if no token provided
	if authToken == "" {
		var err error
		authToken, err = authenticate(t.Context(), baseURL)
		if err != nil {
			t.Fatalf("Failed to authenticate: %v", err)
		}
	}

	spaceRef := getEnv("SPACE_REF", "default")
	now := time.Now().UnixMilli()
	startTime := now - (30 * 24 * 60 * 60 * 1000) // 30 days ago
	metricsURL := fmt.Sprintf("%s/api/v1/spaces/%s/metric?start_time=%d&end_time=%d",
		baseURL, spaceRef, startTime, now)
	targeter := vegeta.NewStaticTargeter(vegeta.Target{
		Method: http.MethodGet,
		URL:    metricsURL,
		Header: http.Header{
			"Authorization": []string{"Bearer " + authToken},
			"Content-Type":  []string{"application/json"},
		},
	})
	rate := vegeta.Rate{Freq: 50, Per: time.Second} // 50 RPS
	duration := 30 * time.Second
	attacker := vegeta.NewAttacker()
	var metrics vegeta.Metrics
	for res := range attacker.Attack(targeter, rate, duration, "Usage Metrics Load Test") {
		metrics.Add(res)
	}
	metrics.Close()
	// Print results
	t.Logf("Requests:      %d", metrics.Requests)
	t.Logf("Success Rate:  %.2f%%", metrics.Success*100)
	t.Logf("Mean Latency:  %s", metrics.Latencies.Mean)
	t.Logf("P95 Latency:   %s", metrics.Latencies.P95)
	t.Logf("P99 Latency:   %s", metrics.Latencies.P99)
	t.Logf("Max Latency:   %s", metrics.Latencies.Max)
	t.Logf("Throughput:    %.2f req/s", metrics.Throughput)
	// Assertions
	if metrics.Success < 0.95 {
		t.Errorf("Success rate %.2f%% is below 95%%", metrics.Success*100)
	}
	if metrics.Latencies.P95 > 500*time.Millisecond {
		t.Errorf("P95 latency %s exceeds 500ms", metrics.Latencies.P95)
	}
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func authenticate(
	ctx context.Context,
	baseURL string,
) (string, error) {
	username := os.Getenv("GITNESS_PRINCIPAL_ADMIN_EMAIL")
	password := os.Getenv("GITNESS_PRINCIPAL_ADMIN_PASSWORD")

	loginURL, err := url.JoinPath(baseURL, "/api/v1/login")
	if err != nil {
		return "", fmt.Errorf("invalid base URL: %w", err)
	}

	parsedURL, err := url.Parse(loginURL)
	if err != nil {
		return "", fmt.Errorf("invalid login URL: %w", err)
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return "", fmt.Errorf("invalid URL scheme: %s", parsedURL.Scheme)
	}

	payload := fmt.Sprintf(`{"login_identifier":%q,"password":%q}`, username, password)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, parsedURL.String(), strings.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("failed to create login request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("login request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("login failed with status %d: %s", resp.StatusCode, body)
	}

	var result struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode login response: %w", err)
	}

	return result.AccessToken, nil
}

// TestRepositoryImportAndFileAccess tests the full workflow:
// 1. Login to Gitness
// 2. Import a repository from GitHub (google/uuid)
// 3. Wait for import to complete
// 4. Fetch a file from the imported repository.
func TestRepositoryImportAndFileAccess(t *testing.T) {
	if !isE2ETestsEnabled() {
		t.Skip("E2E testing is disabled")
	}

	ctx := context.Background()
	baseURL := getEnv("GITNESS_BASE_URL", "http://localhost:3000")
	spaceRef := getEnv("SPACE_REF", "default")

	// Step 1: Authenticate
	authToken := os.Getenv("GITNESS_AUTH_TOKEN")
	if authToken == "" {
		var err error
		authToken, err = authenticate(ctx, baseURL)
		require.NoError(t, err, "Failed to authenticate")
	}

	t.Logf("Successfully authenticated")

	// Step 2: Import repository from github.com/google/uuid
	repoIdentifier := fmt.Sprintf("uuid-test-%d", time.Now().Unix())
	repo, err := importRepository(ctx, baseURL, authToken, spaceRef, repoIdentifier)
	require.NoError(t, err, "Failed to import repository")
	assert.NotEmpty(t, repo.ID, "Repository ID should not be empty")
	assert.Equal(t, repoIdentifier, repo.Identifier, "Repository identifier mismatch")

	t.Logf("Repository imported: %s (ID: %d)", repo.Identifier, repo.ID)

	// Step 3: Wait for import to complete
	t.Logf("Waiting for repository import to complete...")
	err = waitForRepositoryReady(ctx, baseURL, authToken, spaceRef, repoIdentifier, 5*time.Minute)
	require.NoError(t, err, "Repository import timed out or failed")

	t.Logf("Repository import completed successfully")

	// Step 4: Fetch a file from the repository (README.md)
	fileContent, err := getFile(ctx, baseURL, authToken, spaceRef, repoIdentifier, "README.md", "main")
	require.NoError(t, err, "Failed to fetch file")
	assert.NotEmpty(t, fileContent, "File content should not be empty")
	assert.Contains(t, string(fileContent), "uuid", "README should mention uuid")

	t.Logf("Successfully fetched README.md (%d bytes)", len(fileContent))

	// Clean up: Delete the repository
	err = deleteRepository(ctx, baseURL, authToken, spaceRef, repoIdentifier)
	if err != nil {
		t.Logf("Warning: Failed to clean up repository: %v", err)
	} else {
		t.Logf("Repository cleaned up successfully")
	}
}

// TestFileAccessLoad performs load testing on file access endpoint.
func TestFileAccessLoad(t *testing.T) {
	if !isE2ETestsEnabled() {
		t.Skip("E2E testing is disabled")
	}

	ctx := context.Background()
	baseURL := getEnv("GITNESS_BASE_URL", "http://localhost:3000")
	spaceRef := getEnv("SPACE_REF", "default")

	// Authenticate
	authToken := os.Getenv("GITNESS_AUTH_TOKEN")
	if authToken == "" {
		var err error
		authToken, err = authenticate(ctx, baseURL)
		require.NoError(t, err, "Failed to authenticate")
	}

	// Import repository
	repoIdentifier := fmt.Sprintf("uuid-load-test-%d", time.Now().Unix())
	repo, err := importRepository(ctx, baseURL, authToken, spaceRef, repoIdentifier)
	require.NoError(t, err, "Failed to import repository")

	t.Logf("Repository imported: %s", repo.Identifier)

	// Wait for import to complete
	err = waitForRepositoryReady(ctx, baseURL, authToken, spaceRef, repoIdentifier, 5*time.Minute)
	require.NoError(t, err, "Repository import timed out")

	// Perform load test on file access
	fileURL := fmt.Sprintf("%s/api/v1/repos/%s/%s/+/content/README.md?git_ref=refs/heads/master",
		baseURL, spaceRef, repoIdentifier)

	targeter := vegeta.NewStaticTargeter(vegeta.Target{
		Method: http.MethodGet,
		URL:    fileURL,
		Header: http.Header{
			"Authorization": []string{"Bearer " + authToken},
		},
	})

	rate := vegeta.Rate{Freq: 100, Per: time.Second} // 100 RPS
	duration := 30 * time.Second
	attacker := vegeta.NewAttacker()

	var metrics vegeta.Metrics
	for res := range attacker.Attack(targeter, rate, duration, "File Access Load Test") {
		metrics.Add(res)
	}
	metrics.Close()

	// Print results
	t.Logf("=== File Access Load Test Results ===")
	t.Logf("Requests:      %d", metrics.Requests)
	t.Logf("Success Rate:  %.2f%%", metrics.Success*100)
	t.Logf("Mean Latency:  %s", metrics.Latencies.Mean)
	t.Logf("P50 Latency:   %s", metrics.Latencies.P50)
	t.Logf("P95 Latency:   %s", metrics.Latencies.P95)
	t.Logf("P99 Latency:   %s", metrics.Latencies.P99)
	t.Logf("Max Latency:   %s", metrics.Latencies.Max)
	t.Logf("Throughput:    %.2f req/s", metrics.Throughput)

	// Assertions
	assert.GreaterOrEqual(t, metrics.Success, 0.99, "Success rate should be at least 99%%")
	assert.LessOrEqual(t, metrics.Latencies.P95, 200*time.Millisecond, "P95 latency should be under 200ms")
	assert.LessOrEqual(t, metrics.Latencies.P99, 500*time.Millisecond, "P99 latency should be under 500ms")

	// Clean up
	err = deleteRepository(ctx, baseURL, authToken, spaceRef, repoIdentifier)
	if err != nil {
		t.Logf("Warning: Failed to clean up repository: %v", err)
	}
}

// importRepository imports a repository from GitHub using the actual project types.
func importRepository(
	ctx context.Context,
	baseURL string,
	authToken string,
	spaceRef string,
	identifier string,
) (*repoctl.RepositoryOutput, error) {
	importURL := fmt.Sprintf("%s/api/v1/repos/import", baseURL)

	importReq := repoctl.ImportInput{
		ParentRef:   spaceRef,
		Identifier:  identifier,
		Description: "Test repository imported from google/uuid",
		Provider: importer.Provider{
			Type: importer.ProviderTypeGitHub,
		},
		ProviderRepo: "google/uuid",
	}

	payload, err := json.Marshal(importReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal import request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, importURL, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("failed to create import request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+authToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("import request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("import failed with status %d: %s", resp.StatusCode, body)
	}

	var repo repoctl.RepositoryOutput
	if err := json.NewDecoder(resp.Body).Decode(&repo); err != nil {
		return nil, fmt.Errorf("failed to decode repository response: %w", err)
	}

	return &repo, nil
}

// waitForRepositoryReady polls the repository until it's ready or timeout.
func waitForRepositoryReady(
	ctx context.Context,
	baseURL string,
	authToken string,
	spaceRef string,
	repoIdentifier string,
	timeout time.Duration,
) error {
	repoURL := fmt.Sprintf("%s/api/v1/repos/%s/%s/+/content/%s?git_ref=%s",
		baseURL, spaceRef, repoIdentifier, "README.md", "refs/heads/master")
	deadline := time.Now().Add(timeout)
	pollInterval := 5 * time.Second

	for time.Now().Before(deadline) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, repoURL, nil)
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("Authorization", "Bearer "+authToken)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return fmt.Errorf("request failed: %w", err)
		}

		if resp.StatusCode == http.StatusOK {
			var repo repoctl.RepositoryOutput
			if err := json.NewDecoder(resp.Body).Decode(&repo); err != nil {
				resp.Body.Close()
				return fmt.Errorf("failed to decode response: %w", err)
			}
			resp.Body.Close()

			// Check if import is complete (importing should be false)
			if !repo.Importing {
				return nil
			}
		} else {
			resp.Body.Close()
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(pollInterval):
			// Continue polling
		}
	}

	return fmt.Errorf("repository import timed out after %v", timeout)
}

// getFile fetches a file from a repository.
func getFile(
	ctx context.Context,
	baseURL string,
	authToken string,
	spaceRef string,
	repoIdentifier string,
	filePath string,
	ref string,
) ([]byte, error) {
	fileURL := fmt.Sprintf("%s/api/v1/repos/%s/%s/content/%s?ref=%s",
		baseURL, spaceRef, repoIdentifier, filePath, ref)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fileURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+authToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to fetch file with status %d: %s", resp.StatusCode, body)
	}

	// The response might be JSON with content field, or raw content
	// Try to decode as JSON first
	var fileResponse struct {
		Content struct {
			Data string `json:"data"`
		} `json:"content"`
		Type string `json:"type"`
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Try JSON decode
	if err := json.Unmarshal(bodyBytes, &fileResponse); err == nil && fileResponse.Content.Data != "" {
		// Response is JSON with base64 content
		return []byte(fileResponse.Content.Data), nil
	}

	// Otherwise, return raw content
	return bodyBytes, nil
}

// deleteRepository deletes a repository.
func deleteRepository(
	ctx context.Context,
	baseURL string,
	authToken string,
	spaceRef string,
	repoIdentifier string,
) error {
	deleteURL := fmt.Sprintf("%s/api/v1/repos/%s/%s", baseURL, spaceRef, repoIdentifier)

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, deleteURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create delete request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+authToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("delete request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("delete failed with status %d: %s", resp.StatusCode, body)
	}

	return nil
}
