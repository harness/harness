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

package runner

import (
	"testing"

	"github.com/harness/gitness/types"
)

func TestDockerOpts(t *testing.T) {
	tests := []struct {
		name           string
		dockerHost     string
		dockerVersion  string
		expectedMinLen int // minimum expected options (WithAPIVersionNegotiation is always included)
	}{
		{
			name:           "empty config returns at least API version negotiation option",
			dockerHost:     "",
			dockerVersion:  "",
			expectedMinLen: 1, // WithAPIVersionNegotiation
		},
		{
			name:           "with docker host adds host option",
			dockerHost:     "unix:///var/run/docker.sock",
			dockerVersion:  "",
			expectedMinLen: 2, // WithAPIVersionNegotiation + WithHost
		},
		{
			name:           "with docker API version adds version option",
			dockerHost:     "",
			dockerVersion:  "1.45",
			expectedMinLen: 2, // WithAPIVersionNegotiation + WithVersion
		},
		{
			name:           "with both host and API version adds both options",
			dockerHost:     "unix:///var/run/docker.sock",
			dockerVersion:  "1.45",
			expectedMinLen: 3, // WithAPIVersionNegotiation + WithHost + WithVersion
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &types.Config{}
			config.Docker.Host = tt.dockerHost
			config.Docker.APIVersion = tt.dockerVersion

			opts := dockerOpts(config)
			if len(opts) < tt.expectedMinLen {
				t.Errorf("dockerOpts() returned %d options, expected at least %d", len(opts), tt.expectedMinLen)
			}
		})
	}
}

func TestDockerOptsAlwaysIncludesAPIVersionNegotiation(t *testing.T) {
	// This test verifies that dockerOpts always returns at least one option
	// (WithAPIVersionNegotiation) even with an empty config.
	// This is critical for compatibility with Docker 29.0+ which requires
	// API version 1.44 or higher.
	config := &types.Config{}

	opts := dockerOpts(config)

	if len(opts) == 0 {
		t.Error("dockerOpts() should always return at least one option (WithAPIVersionNegotiation)")
	}

	// We can't directly test the option type since dockerclient.Opt is a function type,
	// but we verify that with an empty config, we still get the negotiation option.
	if len(opts) != 1 {
		t.Errorf("dockerOpts() with empty config should return exactly 1 option, got %d", len(opts))
	}
}
