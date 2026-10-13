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

package linkedsync

// PushPayload carries one push-webhook ref update: a branch or tag tip that
// moved on the upstream SCM, with or without an intervening pull request.
type PushPayload struct {
	Ref   string
	After string

	Repository Repository
}

func (PushPayload) Kind() Kind               { return KindPush }
func (p PushPayload) RepoProviderID() string { return p.Repository.ProviderID }
