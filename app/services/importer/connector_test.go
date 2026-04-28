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

package importer

import "testing"

func TestDecodeConnectorRef(t *testing.T) {
	cases := []struct {
		name           string
		parentPath     string
		ref            string
		wantPath       string
		wantIdentifier string
	}{
		{
			name:           "account ref against full project path",
			parentPath:     "acme/platform/code",
			ref:            "account.githubConn",
			wantPath:       "acme",
			wantIdentifier: "githubConn",
		},
		{
			name:           "org ref against full project path",
			parentPath:     "acme/platform/code",
			ref:            "org.githubConn",
			wantPath:       "acme/platform",
			wantIdentifier: "githubConn",
		},
		{
			name:           "bare ref keeps full project path",
			parentPath:     "acme/platform/code",
			ref:            "githubConn",
			wantPath:       "acme/platform/code",
			wantIdentifier: "githubConn",
		},
		{
			name:           "account ref against shorter parent (degrades gracefully)",
			parentPath:     "acme",
			ref:            "account.githubConn",
			wantPath:       "acme",
			wantIdentifier: "githubConn",
		},
		{
			name:           "org ref against account-only parent (degrades gracefully)",
			parentPath:     "acme",
			ref:            "org.githubConn",
			wantPath:       "acme",
			wantIdentifier: "githubConn",
		},
		{
			name:           "identifier whose body starts with 'account' but has no dot is project-scope",
			parentPath:     "acme/platform/code",
			ref:            "accountish",
			wantPath:       "acme/platform/code",
			wantIdentifier: "accountish",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotPath, gotIdentifier := DecodeConnectorRef(tc.parentPath, tc.ref)
			if gotPath != tc.wantPath {
				t.Errorf("DecodeConnectorRef(%q, %q) path = %q; want %q",
					tc.parentPath, tc.ref, gotPath, tc.wantPath)
			}
			if gotIdentifier != tc.wantIdentifier {
				t.Errorf("DecodeConnectorRef(%q, %q) identifier = %q; want %q",
					tc.parentPath, tc.ref, gotIdentifier, tc.wantIdentifier)
			}
		})
	}
}
