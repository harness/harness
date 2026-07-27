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

package types

import "testing"

// TestParseOCIReference locks in the accept/reject behaviour of the local
// parseOCIReference implementation, which is a dependency-light stand-in for
// oras.land/oras-go/v2/registry.ParseReference. The cases below cover the
// subtle parsing rules (split-on-first-"/", "@"-before-":" precedence, digest
// vs. tag disambiguation, and the anchored repository/tag regexes) so that any
// drift from oras semantics is caught here rather than silently changing which
// devcontainer feature sources are accepted.
func TestParseOCIReference(t *testing.T) {
	const validDigest = "sha256:b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9"

	tests := []struct {
		name     string
		artifact string
		wantErr  bool
	}{
		{
			name:     "valid tag reference",
			artifact: "docker.io/foo/bar:latest",
			wantErr:  false,
		},
		{
			name:     "valid digest reference",
			artifact: "docker.io/foo/bar@" + validDigest,
			wantErr:  false,
		},
		{
			name:     "valid reference without tag or digest",
			artifact: "docker.io/foo/bar",
			wantErr:  false,
		},
		{
			name:     "valid single-segment repository",
			artifact: "docker.io/foo:latest",
			wantErr:  false,
		},
		{
			name:     "registry with port",
			artifact: "localhost:5000/foo/bar:latest",
			wantErr:  false,
		},
		{
			// "@" takes precedence over ":", so the ":1.0" is stripped from the
			// repository and the sha256 digest is validated. Accepted by oras.
			name:     "tag and digest together resolve to digest",
			artifact: "docker.io/foo/bar:1.0@" + validDigest,
			wantErr:  false,
		},
		{
			// Registry validation only checks the host, which is case-insensitive;
			// this was accepted by registry.ParseReference and must stay accepted
			// to preserve equivalence.
			name:     "mixed-case registry",
			artifact: "Docker.IO/foo/bar:latest",
			wantErr:  false,
		},
		{
			name:     "uppercase registry",
			artifact: "DOCKER.IO/foo/bar:latest",
			wantErr:  false,
		},
		{
			name:     "missing registry (no slash)",
			artifact: "foobar",
			wantErr:  true,
		},
		{
			name:     "empty string",
			artifact: "",
			wantErr:  true,
		},
		{
			name:     "empty repository with tag",
			artifact: "docker.io/:tag",
			wantErr:  true,
		},
		{
			name:     "empty repository, trailing slash",
			artifact: "docker.io/",
			wantErr:  true,
		},
		{
			name:     "uppercase repository",
			artifact: "docker.io/Foo/Bar:latest",
			wantErr:  true,
		},
		{
			name:     "invalid tag leading dot",
			artifact: "docker.io/foo/bar:.bad",
			wantErr:  true,
		},
		{
			name:     "invalid tag with space",
			artifact: "docker.io/foo/bar:in valid",
			wantErr:  true,
		},
		{
			name:     "malformed digest",
			artifact: "docker.io/foo/bar@sha256:xyz",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := parseOCIReference(tt.artifact)
			if tt.wantErr && err == nil {
				t.Errorf("parseOCIReference(%q) = nil, want error", tt.artifact)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("parseOCIReference(%q) = %v, want nil", tt.artifact, err)
			}
		})
	}
}
