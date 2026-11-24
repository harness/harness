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

// Package e2e holds end to end tests that run against a live gitness server.
// They are skipped unless GITNESS_E2E_TEST_ENABLED is set to true.
package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

func init() {
	if err := godotenv.Load("../../.local.env"); err != nil {
		log.Println("Error loading .local.env file")
	}
	if err := godotenv.Load("../../.test.env"); err != nil {
		log.Println("Error loading .test.env file")
	}
}

func isE2ETestsEnabled() bool {
	env := strings.ToLower(strings.TrimSpace(os.Getenv("GITNESS_E2E_TEST_ENABLED")))
	return env == "1" || env == "true"
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

// authenticate logs in as the admin principal and returns an access token.
func authenticate(ctx context.Context, baseURL string) (string, error) {
	payload := fmt.Sprintf(`{"login_identifier":%q,"password":%q}`,
		os.Getenv("GITNESS_PRINCIPAL_ADMIN_EMAIL"), os.Getenv("GITNESS_PRINCIPAL_ADMIN_PASSWORD"))

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/api/v1/login", strings.NewReader(payload))
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
