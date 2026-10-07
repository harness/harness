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

// bitbucketCloudGitUser is the static username Bitbucket Cloud expects for Git.
// The REST lookup still uses the username from the import form.
const bitbucketCloudGitUser = "x-bitbucket-api-token-auth"

// gitCloneUser returns the username embedded in the import clone URL.
// Bitbucket Cloud always uses the static Git username. Other providers are unchanged.
func gitCloneUser(provider Provider) string {
	if provider.Type == ProviderTypeBitbucket {
		return bitbucketCloudGitUser
	}
	return provider.Username
}
