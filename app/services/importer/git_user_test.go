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

func TestGitCloneUser(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		provider Provider
		want     string
	}{
		{
			name: "bitbucket email uses the static git username",
			provider: Provider{
				Type:     ProviderTypeBitbucket,
				Username: "dev@example.com",
				Password: "secret",
			},
			want: bitbucketCloudGitUser,
		},
		{
			name: "bitbucket username also uses the static git username",
			provider: Provider{
				Type:     ProviderTypeBitbucket,
				Username: "devuser",
				Password: "secret",
			},
			want: bitbucketCloudGitUser,
		},
		{
			name: "bitbucket with no password still uses the static git username",
			provider: Provider{
				Type:     ProviderTypeBitbucket,
				Username: "",
			},
			want: bitbucketCloudGitUser,
		},
		{
			name: "github token username is unchanged",
			provider: Provider{
				Type:     ProviderTypeGitHub,
				Username: "dev@example.com",
				Password: "token",
			},
			want: "dev@example.com",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := gitCloneUser(tc.provider); got != tc.want {
				t.Fatalf("gitCloneUser() = %q, want %q", got, tc.want)
			}
		})
	}
}
