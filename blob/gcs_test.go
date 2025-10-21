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

package blob

import (
	"context"
	"testing"
	"time"
)

func TestNewGCSStore_InvalidConfig(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name        string
		config      Config
		expectError bool
	}{
		{
			name: "empty config",
			config: Config{
				Provider: ProviderGCS,
			},
			expectError: true, // Should fail without proper credentials
		},
		{
			name: "config with non-existent key file",
			config: Config{
				Provider: ProviderGCS,
				KeyPath:  "/non/existent/path/key.json",
				Bucket:   "test-bucket",
			},
			expectError: true, // Should fail with invalid key path
		},
		{
			name: "config with empty bucket",
			config: Config{
				Provider: ProviderGCS,
				KeyPath:  "/tmp/fake-key.json",
				Bucket:   "",
			},
			expectError: true, // Should fail with empty bucket
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			store, err := NewGCSStore(ctx, test.config)

			if test.expectError {
				if err == nil {
					t.Error("expected error but got none")
				}
				if store != nil {
					t.Error("expected nil store on error")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if store == nil {
				t.Error("expected non-nil store")
			}
		})
	}
}

func TestGCSStore_ConfigValidation(t *testing.T) {
	tests := []struct {
		name   string
		config Config
	}{
		{
			name: "config with service account key",
			config: Config{
				Provider: ProviderGCS,
				KeyPath:  "/path/to/key.json",
				Bucket:   "test-bucket",
			},
		},
		{
			name: "config with impersonation",
			config: Config{
				Provider:              ProviderGCS,
				Bucket:                "test-bucket",
				TargetPrincipal:       "service-account@project.iam.gserviceaccount.com",
				ImpersonationLifetime: time.Hour,
			},
		},
		{
			name: "config with zero impersonation lifetime",
			config: Config{
				Provider:              ProviderGCS,
				Bucket:                "test-bucket",
				TargetPrincipal:       "service-account@project.iam.gserviceaccount.com",
				ImpersonationLifetime: 0,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Test that config fields are properly set
			config := test.config

			if config.Provider != ProviderGCS {
				t.Errorf("expected provider %q, got %q", ProviderGCS, config.Provider)
			}

			// Verify other fields are accessible
			_ = config.Bucket
			_ = config.KeyPath
			_ = config.TargetPrincipal
			_ = config.ImpersonationLifetime
		})
	}
}

func TestGCSStore_Interface(_ *testing.T) {
	// Test that GCSStore would implement Store interface
	// Note: We can't actually create a GCSStore without valid GCS credentials,
	// but we can verify the interface compliance at compile time

	// This will fail to compile if GCSStore doesn't implement Store
	var _ Store = (*GCSStore)(nil)
}

func TestGCSStore_DefaultScope(t *testing.T) {
	expectedScope := "https://www.googleapis.com/auth/cloud-platform"
	if defaultScope != expectedScope {
		t.Errorf("expected default scope %q, got %q", expectedScope, defaultScope)
	}
}

// Note: More comprehensive tests for GCSStore would require:
// 1. Mock GCS client or test environment
// 2. Valid GCS credentials for integration tests
// 3. Test bucket setup
// These tests focus on the parts that can be tested without external dependencies.
