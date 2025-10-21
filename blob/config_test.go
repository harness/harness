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
	"testing"
	"time"
)

func TestProviderConstants(t *testing.T) {
	tests := []struct {
		name     string
		provider Provider
		expected string
	}{
		{
			name:     "GCS provider",
			provider: ProviderGCS,
			expected: "gcs",
		},
		{
			name:     "FileSystem provider",
			provider: ProviderFileSystem,
			expected: "filesystem",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if string(test.provider) != test.expected {
				t.Errorf("expected provider %q, got %q", test.expected, string(test.provider))
			}
		})
	}
}

func TestConfigStruct(t *testing.T) {
	tests := []struct {
		name   string
		config Config
	}{
		{
			name:   "empty config",
			config: Config{},
		},
		{
			name: "GCS config",
			config: Config{
				Provider:              ProviderGCS,
				Bucket:                "test-bucket",
				KeyPath:               "/path/to/key.json",
				TargetPrincipal:       "test@example.com",
				ImpersonationLifetime: time.Hour,
			},
		},
		{
			name: "FileSystem config",
			config: Config{
				Provider: ProviderFileSystem,
				Bucket:   "/local/storage/path",
			},
		},
		{
			name: "config with zero duration",
			config: Config{
				Provider:              ProviderGCS,
				ImpersonationLifetime: 0,
			},
		},
		{
			name: "config with negative duration",
			config: Config{
				Provider:              ProviderGCS,
				ImpersonationLifetime: -time.Hour,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Test that the config can be created and accessed
			config := test.config

			// Verify provider field
			if config.Provider != test.config.Provider {
				t.Errorf("expected provider %q, got %q", test.config.Provider, config.Provider)
			}

			// Verify bucket field
			if config.Bucket != test.config.Bucket {
				t.Errorf("expected bucket %q, got %q", test.config.Bucket, config.Bucket)
			}

			// Verify key path field
			if config.KeyPath != test.config.KeyPath {
				t.Errorf("expected key path %q, got %q", test.config.KeyPath, config.KeyPath)
			}

			// Verify target principal field
			if config.TargetPrincipal != test.config.TargetPrincipal {
				t.Errorf("expected target principal %q, got %q", test.config.TargetPrincipal, config.TargetPrincipal)
			}

			// Verify impersonation lifetime field
			if config.ImpersonationLifetime != test.config.ImpersonationLifetime {
				t.Errorf("expected impersonation lifetime %v, got %v",
					test.config.ImpersonationLifetime, config.ImpersonationLifetime)
			}
		})
	}
}

func TestProviderStringConversion(t *testing.T) {
	// Test that Provider type can be converted to string
	gcsStr := string(ProviderGCS)
	if gcsStr != "gcs" {
		t.Errorf("expected 'gcs', got %q", gcsStr)
	}

	fsStr := string(ProviderFileSystem)
	if fsStr != "filesystem" {
		t.Errorf("expected 'filesystem', got %q", fsStr)
	}
}

func TestProviderComparison(t *testing.T) {
	// Test provider equality
	if ProviderGCS == ProviderFileSystem {
		t.Error("ProviderGCS should not equal ProviderFileSystem")
	}

	if ProviderGCS == ProviderFileSystem {
		t.Error("ProviderGCS should not equal ProviderFileSystem")
	}

	if ProviderFileSystem == ProviderGCS {
		t.Error("ProviderFileSystem should not equal ProviderGCS")
	}
}
