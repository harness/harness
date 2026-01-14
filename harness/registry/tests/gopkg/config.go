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

package gopkgconformance

import (
	"fmt"
	"os"
	"time"
)

// Config holds the test configuration.
type Config struct {
	RootURL       string
	Username      string
	Password      string
	Namespace     string
	RegistryName  string
	Debug         bool
	DisabledTests map[string]bool
	StartTime     int64 // Used to generate unique versions.
}

// TestCategory represents different test categories.
type TestCategory string

const (
	TestDownload         TestCategory = "download"
	TestUpload           TestCategory = "upload"
	TestContentDiscovery TestCategory = "content_discovery"
	TestErrorHandling    TestCategory = "error_handling"
)

var (
	// TestConfig holds the global test configuration.
	TestConfig Config
)

// InitConfig initializes the test configuration.
func InitConfig() {
	// For Gitness integration, we need to ensure values are properly set.
	// to work with the Gitness server structure.
	TestConfig = Config{
		// Base URL for the Gitness server.
		RootURL: getEnv("REGISTRY_ROOT_URL", "http://localhost:3000"),

		// Use admin@gitness.io as per .local.env configuration.
		Username: getEnv("REGISTRY_USERNAME", "admin@gitness.io"),

		// Password will be filled by setup_test.sh via token authentication
		Password: getEnv("REGISTRY_PASSWORD", ""),

		// These values come from setup_test.sh.
		Namespace:    getEnv("REGISTRY_NAMESPACE", ""),
		RegistryName: getEnv("REGISTRY_NAME", ""),

		// Enable debug to see detailed logs.
		Debug: getEnv("DEBUG", "true") == "true",

		// Store start time for unique version generation.
		StartTime: time.Now().Unix(),

		// All tests are now enabled
		DisabledTests: map[string]bool{},
	}
}

// IsTestEnabled checks if a test category is enabled.
func IsTestEnabled(category TestCategory) bool {
	return !TestConfig.DisabledTests[string(category)]
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

// GetUniqueVersion generates a unique version string for tests.
// Each test should call this with a different test ID to ensure uniqueness.
func GetUniqueVersion(testID int) string {
	return fmt.Sprintf("1.0.%d-%d", TestConfig.StartTime, testID)
}

// GetUniqueArtifactName generates a unique artifact name for tests.
// This ensures that different test contexts don't conflict with each other.
func GetUniqueArtifactName(testContext string, testID int) string {
	return fmt.Sprintf("test-artifact-%s-%d-%d", testContext, TestConfig.StartTime, testID)
}
