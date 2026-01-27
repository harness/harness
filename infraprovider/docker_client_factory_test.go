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

package infraprovider

import (
	"testing"
)

func TestDockerClientFactoryDockerOpts(t *testing.T) {
	tests := []struct {
		name        string
		config      *DockerConfig
		expectedLen int
		expectError bool
	}{
		{
			name:        "empty config returns no additional overrides",
			config:      &DockerConfig{},
			expectedLen: 0,
			expectError: false,
		},
		{
			name: "with docker host adds host option",
			config: &DockerConfig{
				DockerHost: "unix:///var/run/docker.sock",
			},
			expectedLen: 1,
			expectError: false,
		},
		{
			name: "with docker API version adds version option",
			config: &DockerConfig{
				DockerAPIVersion: "1.45",
			},
			expectedLen: 1,
			expectError: false,
		},
		{
			name: "with both host and API version adds both options",
			config: &DockerConfig{
				DockerHost:       "unix:///var/run/docker.sock",
				DockerAPIVersion: "1.45",
			},
			expectedLen: 2,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			factory := NewDockerClientFactory(tt.config)
			opts, err := factory.dockerOpts(tt.config)

			if tt.expectError && err == nil {
				t.Error("dockerOpts() expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("dockerOpts() unexpected error: %v", err)
			}
			if len(opts) != tt.expectedLen {
				t.Errorf("dockerOpts() returned %d options, expected %d", len(opts), tt.expectedLen)
			}
		})
	}
}

func TestNewDockerClientFactory(t *testing.T) {
	config := &DockerConfig{
		DockerHost:       "unix:///var/run/docker.sock",
		DockerAPIVersion: "1.45",
	}

	factory := NewDockerClientFactory(config)

	if factory == nil {
		t.Fatal("NewDockerClientFactory() returned nil")
	}

	if factory.config != config {
		t.Error("NewDockerClientFactory() did not set config correctly")
	}
}
